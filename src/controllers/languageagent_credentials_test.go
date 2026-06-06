package controllers

import (
	"context"
	"testing"

	"github.com/go-logr/logr"
	langopv1alpha1 "github.com/language-operator/language-operator/api/v1alpha1"
	"github.com/language-operator/language-operator/controllers/testutil"
	"github.com/language-operator/language-operator/internal/testutil/gen"
	"github.com/language-operator/language-operator/pkg/events"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/tools/record"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

// newCredentialsReconciler builds a reconciler and working agent for credential tests.
func newCredentialsReconciler(t *testing.T, agent *langopv1alpha1.LanguageAgent) (*LanguageAgentReconciler, *langopv1alpha1.LanguageAgent) {
	t.Helper()
	scheme := testutil.SetupTestScheme(t)
	cluster := gen.ReadyCluster(agent.Namespace)
	recorder := record.NewFakeRecorder(10)
	fakeClient := fake.NewClientBuilder().WithScheme(scheme).WithObjects(cluster, agent).WithStatusSubresource(agent).Build()
	r := &LanguageAgentReconciler{
		Client:          fakeClient,
		Scheme:          scheme,
		Log:             logr.Discard(),
		Recorder:        recorder,
		EventManager:    events.NewEventManager(recorder),
		RegistryManager: &mockRegistryManager{},
	}
	return r, agent.DeepCopy()
}

func baseAgent(name string, creds ...langopv1alpha1.CredentialSpec) *langopv1alpha1.LanguageAgent {
	disabled := false
	return &langopv1alpha1.LanguageAgent{
		ObjectMeta: metav1.ObjectMeta{Name: name, Namespace: "default"},
		Spec: langopv1alpha1.LanguageAgentSpec{
			Image:       "ghcr.io/language-operator/agent:latest",
			Workspace:   &langopv1alpha1.WorkspaceSpec{Enabled: &disabled},
			Credentials: creds,
		},
	}
}

// TestReconcileRuntimeSecret_Generated verifies an auto-generated credential is stored in the
// managed Secret and the Secret is wired into the agent's envFrom.
func TestReconcileRuntimeSecret_Generated(t *testing.T) {
	agent := baseAgent("cred-gen", langopv1alpha1.CredentialSpec{Name: "OPENCLAW_GATEWAY_TOKEN"})
	r, working := newCredentialsReconciler(t, agent)

	require.NoError(t, r.reconcileRuntimeSecret(context.Background(), agent, working))

	secret := &corev1.Secret{}
	require.NoError(t, r.Get(context.Background(), types.NamespacedName{Name: "cred-gen-runtime", Namespace: "default"}, secret))
	assert.NotEmpty(t, secret.Data["OPENCLAW_GATEWAY_TOKEN"], "token should be generated")

	assert.True(t, hasSecretEnvFrom(working, "cred-gen-runtime"), "managed Secret should be wired via envFrom")
}

// TestReconcileRuntimeSecret_GeneratedPreserved verifies a generated value is not rotated across reconciles.
func TestReconcileRuntimeSecret_GeneratedPreserved(t *testing.T) {
	agent := baseAgent("cred-keep", langopv1alpha1.CredentialSpec{Name: "TOKEN"})
	r, working := newCredentialsReconciler(t, agent)

	require.NoError(t, r.reconcileRuntimeSecret(context.Background(), agent, working))
	first := &corev1.Secret{}
	require.NoError(t, r.Get(context.Background(), types.NamespacedName{Name: "cred-keep-runtime", Namespace: "default"}, first))
	original := string(first.Data["TOKEN"])
	require.NotEmpty(t, original)

	working2 := agent.DeepCopy()
	require.NoError(t, r.reconcileRuntimeSecret(context.Background(), agent, working2))
	second := &corev1.Secret{}
	require.NoError(t, r.Get(context.Background(), types.NamespacedName{Name: "cred-keep-runtime", Namespace: "default"}, second))
	assert.Equal(t, original, string(second.Data["TOKEN"]), "generated value must be preserved, not rotated")
}

// TestReconcileRuntimeSecret_InlineValue verifies an inline value is stored verbatim.
func TestReconcileRuntimeSecret_InlineValue(t *testing.T) {
	agent := baseAgent("cred-inline", langopv1alpha1.CredentialSpec{Name: "PASSWORD", Value: "s3cret"})
	r, working := newCredentialsReconciler(t, agent)

	require.NoError(t, r.reconcileRuntimeSecret(context.Background(), agent, working))
	secret := &corev1.Secret{}
	require.NoError(t, r.Get(context.Background(), types.NamespacedName{Name: "cred-inline-runtime", Namespace: "default"}, secret))
	assert.Equal(t, "s3cret", string(secret.Data["PASSWORD"]))
}

// TestReconcileRuntimeSecret_ValueFrom verifies a referenced Secret is injected via envFrom and no
// managed Secret is created.
func TestReconcileRuntimeSecret_ValueFrom(t *testing.T) {
	agent := baseAgent("cred-ref", langopv1alpha1.CredentialSpec{
		Name:      "TOKEN",
		ValueFrom: &langopv1alpha1.RuntimeSecretRef{Name: "external-secret"},
	})
	r, working := newCredentialsReconciler(t, agent)

	require.NoError(t, r.reconcileRuntimeSecret(context.Background(), agent, working))

	// No managed Secret should exist.
	managed := &corev1.Secret{}
	err := r.Get(context.Background(), types.NamespacedName{Name: "cred-ref-runtime", Namespace: "default"}, managed)
	assert.Error(t, err, "no managed Secret should be created for a ValueFrom credential")

	assert.True(t, hasSecretEnvFrom(working, "external-secret"), "referenced Secret should be wired via envFrom")
}

func hasSecretEnvFrom(agent *langopv1alpha1.LanguageAgent, secretName string) bool {
	for _, ef := range agent.Spec.Deployment.EnvFrom {
		if ef.SecretRef != nil && ef.SecretRef.Name == secretName {
			return true
		}
	}
	return false
}
