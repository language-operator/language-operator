package controllers

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"testing"
	"time"

	"github.com/go-logr/logr"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/tools/record"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	langopv1alpha1 "github.com/language-operator/language-operator/api/v1alpha1"
	"github.com/language-operator/language-operator/controllers/testutil"
	"github.com/language-operator/language-operator/internal/testutil/gen"
	langoplabels "github.com/language-operator/language-operator/pkg/labels"
)

// expectedAgentKey recomputes an agent key independently of gatewayAgentKey, the
// way components/model-gateway/custom_auth.py does.
func expectedAgentKey(secret, agentID string) string {
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write([]byte(agentID))
	return "sk-langop-" + agentID + "." + hex.EncodeToString(mac.Sum(nil))[:32]
}

func authSecret(ns, value string) *corev1.Secret {
	return &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{Name: GatewayAuthSecretName, Namespace: ns},
		Data:       map[string][]byte{GatewayAuthSecretKey: []byte(value)},
	}
}

func TestGatewayAgentKeyMatchesCustomAuth(t *testing.T) {
	// Fixed vector, also checked against custom_auth.agent_key in the PR.
	assert.Equal(t, expectedAgentKey("s3cret", "9f8221e3-a8ce-47c0-9e31-4afeb62657df"),
		gatewayAgentKey("s3cret", "9f8221e3-a8ce-47c0-9e31-4afeb62657df"))
}

// --- cluster side: the HMAC secret and the gateway Deployment ---

func reconcileAuthCluster(t *testing.T, objs ...client.Object) (client.Client, *LanguageClusterReconciler, string) {
	t.Helper()
	scheme := testutil.SetupTestScheme(t)
	cluster := gen.LanguageCluster("auth-cluster")
	fc := fake.NewClientBuilder().WithScheme(scheme).WithObjects(append([]client.Object{cluster}, objs...)...).
		WithStatusSubresource(cluster).Build()
	r := &LanguageClusterReconciler{Client: fc, Scheme: scheme, Log: logr.Discard()}
	for i := 0; i < 2; i++ {
		_, err := r.Reconcile(context.Background(), clusterRequest(cluster.Name))
		require.NoError(t, err)
	}
	return fc, r, cluster.Name
}

func gatewayDeployment(t *testing.T, c client.Client, ns string) *appsv1.Deployment {
	t.Helper()
	dep := &appsv1.Deployment{}
	require.NoError(t, c.Get(context.Background(), types.NamespacedName{Name: GatewayResourceName, Namespace: ns}, dep))
	return dep
}

func TestGatewayAuthSecretIsCreatedOnceAndWiredIntoTheGateway(t *testing.T) {
	ctx := context.Background()
	fc, r, ns := reconcileAuthCluster(t)

	secret := &corev1.Secret{}
	require.NoError(t, fc.Get(ctx, types.NamespacedName{Name: GatewayAuthSecretName, Namespace: ns}, secret))
	first := string(secret.Data[GatewayAuthSecretKey])
	assert.Len(t, first, 64, "32 random bytes, hex encoded")

	_, err := r.Reconcile(ctx, clusterRequest(ns))
	require.NoError(t, err)
	require.NoError(t, fc.Get(ctx, types.NamespacedName{Name: GatewayAuthSecretName, Namespace: ns}, secret))
	assert.Equal(t, first, string(secret.Data[GatewayAuthSecretKey]), "the secret must not be regenerated")

	env := gatewayDeployment(t, fc, ns).Spec.Template.Spec.Containers[0].Env
	require.NotEmpty(t, env)
	assert.Equal(t, gatewayHMACSecretEnv, env[0].Name)
	require.NotNil(t, env[0].ValueFrom)
	assert.Equal(t, GatewayAuthSecretName, env[0].ValueFrom.SecretKeyRef.Name)
	assert.Equal(t, GatewayAuthSecretKey, env[0].ValueFrom.SecretKeyRef.Key)
}

func TestGatewayAuthSecretKeepsAUserSuppliedValueAndRotationRestartsTheGateway(t *testing.T) {
	ctx := context.Background()
	fc, r, ns := reconcileAuthCluster(t, authSecret("auth-cluster", "user-chosen"))

	secret := &corev1.Secret{}
	require.NoError(t, fc.Get(ctx, types.NamespacedName{Name: GatewayAuthSecretName, Namespace: ns}, secret))
	assert.Equal(t, "user-chosen", string(secret.Data[GatewayAuthSecretKey]))

	before := gatewayDeployment(t, fc, ns).Spec.Template.Annotations[langoplabels.LabelKeyLangopConfigHash]
	secret.Data[GatewayAuthSecretKey] = []byte("rotated")
	require.NoError(t, fc.Update(ctx, secret))
	_, err := r.Reconcile(ctx, clusterRequest(ns))
	require.NoError(t, err)
	after := gatewayDeployment(t, fc, ns).Spec.Template.Annotations[langoplabels.LabelKeyLangopConfigHash]
	assert.NotEqual(t, before, after, "rotating the HMAC secret must roll the gateway pods")
}

// --- agent side: the per-agent key ---

const testAgentUID = "4a7d6c2e-1b3f-4e5d-9a8b-0c1d2e3f4a5b"

func newKeyedAgentReconciler(t *testing.T, objs ...client.Object) (*LanguageAgentReconciler, client.Client, *langopv1alpha1.LanguageAgent) {
	t.Helper()
	scheme := testutil.SetupTestScheme(t)
	agent := gen.LanguageAgent("keyed-agent", "default")
	agent.UID = testAgentUID
	agent.Spec.Deployment.InitContainers = []corev1.Container{{Name: "setup", Image: "busybox"}}
	fc := fake.NewClientBuilder().WithScheme(scheme).
		WithObjects(append([]client.Object{agent, gen.ReadyCluster("default")}, objs...)...).
		WithStatusSubresource(agent).Build()
	return &LanguageAgentReconciler{
		Client: fc, Scheme: scheme, Log: logr.Discard(),
		Recorder: &record.FakeRecorder{}, RegistryManager: &mockRegistryManager{},
	}, fc, agent
}

func reconcileAgent(t *testing.T, r *LanguageAgentReconciler, agent *langopv1alpha1.LanguageAgent) ctrl.Result {
	t.Helper()
	var res ctrl.Result
	for i := 0; i < 2; i++ {
		var err error
		res, err = r.Reconcile(context.Background(), ctrl.Request{NamespacedName: types.NamespacedName{Name: agent.Name, Namespace: agent.Namespace}})
		require.NoError(t, err)
	}
	return res
}

func TestAgentGetsItsGatewayKeyFromASecret(t *testing.T) {
	ctx := context.Background()
	r, fc, agent := newKeyedAgentReconciler(t, authSecret("default", "s3cret"))
	res := reconcileAgent(t, r, agent)
	assert.Zero(t, res.RequeueAfter, "nothing pending once the key is issued")

	keySecret := &corev1.Secret{}
	require.NoError(t, fc.Get(ctx, types.NamespacedName{Name: "keyed-agent-gateway-key", Namespace: "default"}, keySecret))
	assert.Equal(t, expectedAgentKey("s3cret", testAgentUID), string(keySecret.Data[ModelAPIKeyEnv]),
		"the key is signed over the agent UUID, which the gateway reports as user_id")

	// The key reaches the agent and every init container by reference only: it is
	// never written into the WorkflowTemplate.
	podSpec, _ := agentPodView(t, fc, agent.Name, agent.Namespace)
	for _, c := range []corev1.Container{podSpec.Containers[0], podSpec.InitContainers[len(podSpec.InitContainers)-1]} {
		var keyEnv *corev1.EnvVar
		for i := range c.Env {
			if c.Env[i].Name == ModelAPIKeyEnv {
				keyEnv = &c.Env[i]
			}
		}
		require.NotNil(t, keyEnv, "container %s has no MODEL_API_KEY", c.Name)
		assert.Empty(t, keyEnv.Value)
		require.NotNil(t, keyEnv.ValueFrom.SecretKeyRef)
		assert.Equal(t, "keyed-agent-gateway-key", keyEnv.ValueFrom.SecretKeyRef.Name)
	}
}

func TestRotatingTheGatewaySecretReplacesTheAgentRevision(t *testing.T) {
	ctx := context.Background()
	r, fc, agent := newKeyedAgentReconciler(t, authSecret("default", "s3cret"))
	reconcileAgent(t, r, agent)
	before := agentWorkflowTemplate(t, fc, agent.Name, agent.Namespace).Annotations[langoplabels.LabelKeyLangopConfigHash]

	secret := &corev1.Secret{}
	require.NoError(t, fc.Get(ctx, types.NamespacedName{Name: GatewayAuthSecretName, Namespace: "default"}, secret))
	secret.Data[GatewayAuthSecretKey] = []byte("rotated")
	require.NoError(t, fc.Update(ctx, secret))
	reconcileAgent(t, r, agent)

	after := agentWorkflowTemplate(t, fc, agent.Name, agent.Namespace).Annotations[langoplabels.LabelKeyLangopConfigHash]
	assert.NotEqual(t, before, after, "a new key must change the revision, so running Workflows are replaced")

	keySecret := &corev1.Secret{}
	require.NoError(t, fc.Get(ctx, types.NamespacedName{Name: "keyed-agent-gateway-key", Namespace: "default"}, keySecret))
	assert.Equal(t, expectedAgentKey("rotated", testAgentUID), string(keySecret.Data[ModelAPIKeyEnv]))
}

func TestAgentWaitsForTheGatewaySecret(t *testing.T) {
	ctx := context.Background()
	r, fc, agent := newKeyedAgentReconciler(t) // no gateway-auth yet
	res := reconcileAgent(t, r, agent)
	assert.Equal(t, 10*time.Second, res.RequeueAfter, "the agent is reconciled again once the cluster has a secret")

	err := fc.Get(ctx, types.NamespacedName{Name: "keyed-agent-gateway-key", Namespace: "default"}, &corev1.Secret{})
	assert.Error(t, err, "no key can be issued yet")
}
