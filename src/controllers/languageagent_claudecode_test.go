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
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

// buildClaudeCodeReconciler builds a reconciler with a fake client containing the given agent.
func buildClaudeCodeReconciler(t *testing.T, agent *langopv1alpha1.LanguageAgent) (*LanguageAgentReconciler, *fake.ClientBuilder) {
	t.Helper()
	scheme := testutil.SetupTestScheme(t)
	cluster := gen.ReadyCluster(agent.Namespace)
	recorder := record.NewFakeRecorder(10)
	fakeClient := fake.NewClientBuilder().
		WithScheme(scheme).
		WithObjects(cluster, agent).
		WithStatusSubresource(agent).
		Build()
	reconciler := &LanguageAgentReconciler{
		Client:          fakeClient,
		Scheme:          scheme,
		Log:             logr.Discard(),
		Recorder:        recorder,
		EventManager:    events.NewEventManager(recorder),
		RegistryManager: &mockRegistryManager{},
	}
	return reconciler, fake.NewClientBuilder().WithScheme(scheme)
}

// reconcileClaudeCodeAgent runs two reconciles (first adds finalizer, second creates resources)
// and returns the updated agent.
func reconcileClaudeCodeAgent(t *testing.T, agent *langopv1alpha1.LanguageAgent) (*LanguageAgentReconciler, *langopv1alpha1.LanguageAgent) {
	t.Helper()
	scheme := testutil.SetupTestScheme(t)
	cluster := gen.ReadyCluster(agent.Namespace)
	recorder := record.NewFakeRecorder(10)
	fakeClient := fake.NewClientBuilder().
		WithScheme(scheme).
		WithObjects(cluster, agent).
		WithStatusSubresource(agent).
		Build()
	reconciler := &LanguageAgentReconciler{
		Client:          fakeClient,
		Scheme:          scheme,
		Log:             logr.Discard(),
		Recorder:        recorder,
		EventManager:    events.NewEventManager(recorder),
		RegistryManager: &mockRegistryManager{},
	}
	ctx := context.Background()
	req := ctrl.Request{NamespacedName: types.NamespacedName{Name: agent.Name, Namespace: agent.Namespace}}
	_, err := reconciler.Reconcile(ctx, req)
	require.NoError(t, err)
	_, err = reconciler.Reconcile(ctx, req)
	require.NoError(t, err)
	updated := &langopv1alpha1.LanguageAgent{}
	require.NoError(t, fakeClient.Get(ctx, req.NamespacedName, updated))
	return reconciler, updated
}

func TestReconcileRuntimeSecret_ClaudeCode_DisabledConfig(t *testing.T) {
	// claudeCode present but Enabled=false → no managed secret, no ANTHROPIC_API_KEY injected
	agent := &langopv1alpha1.LanguageAgent{
		ObjectMeta: metav1.ObjectMeta{Name: "cc-disabled", Namespace: "default"},
		Spec: langopv1alpha1.LanguageAgentSpec{
			Image: "ghcr.io/language-operator/agent:latest",
			Workspace: &langopv1alpha1.WorkspaceSpec{
				Enabled: func() *bool { b := false; return &b }(),
			},
			ClaudeCode: &langopv1alpha1.ClaudeCodeConfig{
				Enabled: false,
			},
		},
	}
	reconciler, updated := reconcileClaudeCodeAgent(t, agent)

	hasMR := func(kind, name string) bool {
		for _, r := range updated.Status.ManagedResources {
			if r.Kind == kind && r.Name == name {
				return true
			}
		}
		return false
	}
	assert.False(t, hasMR("Secret", "cc-disabled-runtime"), "no runtime Secret when Enabled=false")

	// Direct call to reconcileRuntimeSecret must not inject ANTHROPIC_API_KEY
	working := agent.DeepCopy()
	err := reconciler.reconcileRuntimeSecret(context.Background(), agent, working)
	require.NoError(t, err)
	for _, e := range working.Spec.Deployment.Env {
		assert.NotEqual(t, "ANTHROPIC_API_KEY", e.Name, "ANTHROPIC_API_KEY must not be injected when Enabled=false")
	}
}

func TestReconcileRuntimeSecret_ClaudeCode_NilConfig(t *testing.T) {
	// No claudeCode config → no managed secret, no ANTHROPIC_API_KEY
	agent := &langopv1alpha1.LanguageAgent{
		ObjectMeta: metav1.ObjectMeta{Name: "cc-nil", Namespace: "default"},
		Spec: langopv1alpha1.LanguageAgentSpec{
			Image: "ghcr.io/language-operator/agent:latest",
			Workspace: &langopv1alpha1.WorkspaceSpec{
				Enabled: func() *bool { b := false; return &b }(),
			},
		},
	}
	_, updated := reconcileClaudeCodeAgent(t, agent)

	hasMR := func(kind, name string) bool {
		for _, r := range updated.Status.ManagedResources {
			if r.Kind == kind && r.Name == name {
				return true
			}
		}
		return false
	}

	assert.False(t, hasMR("Secret", "cc-nil-runtime"), "no runtime Secret without claudeCode config")
}

func TestReconcileRuntimeSecret_ClaudeCode_GatewayMode(t *testing.T) {
	// claudeCode.enabled=true, no APIKey → ANTHROPIC_API_KEY=sk-langop-proxy literal env var, no managed secret
	agent := &langopv1alpha1.LanguageAgent{
		ObjectMeta: metav1.ObjectMeta{Name: "cc-gw", Namespace: "default"},
		Spec: langopv1alpha1.LanguageAgentSpec{
			Image: "ghcr.io/language-operator/agent:latest",
			Workspace: &langopv1alpha1.WorkspaceSpec{
				Enabled: func() *bool { b := false; return &b }(),
			},
			ClaudeCode: &langopv1alpha1.ClaudeCodeConfig{
				Enabled: true,
			},
		},
	}
	reconciler, _ := reconcileClaudeCodeAgent(t, agent)

	// Managed secret must not be created (placeholder is injected as literal env var)
	secret := &corev1.Secret{}
	err := reconciler.Client.Get(context.Background(), types.NamespacedName{
		Name:      "cc-gw-runtime",
		Namespace: "default",
	}, secret)
	assert.True(t, err != nil, "no managed runtime Secret in gateway mode")

	// Verify env var injection by calling reconcileRuntimeSecret directly
	working := agent.DeepCopy()
	err = reconciler.reconcileRuntimeSecret(context.Background(), agent, working)
	require.NoError(t, err)

	var found bool
	for _, e := range working.Spec.Deployment.Env {
		if e.Name == "ANTHROPIC_API_KEY" && e.Value == "sk-langop-proxy" {
			found = true
		}
	}
	assert.True(t, found, "ANTHROPIC_API_KEY=sk-langop-proxy injected in gateway mode")
}

func TestReconcileRuntimeSecret_ClaudeCode_InlineAPIKey(t *testing.T) {
	// claudeCode.apiKey set → managed secret with ANTHROPIC_API_KEY
	agent := &langopv1alpha1.LanguageAgent{
		ObjectMeta: metav1.ObjectMeta{Name: "cc-inline", Namespace: "default"},
		Spec: langopv1alpha1.LanguageAgentSpec{
			Image: "ghcr.io/language-operator/agent:latest",
			Workspace: &langopv1alpha1.WorkspaceSpec{
				Enabled: func() *bool { b := false; return &b }(),
			},
			ClaudeCode: &langopv1alpha1.ClaudeCodeConfig{
				Enabled: true,
				APIKey:  "sk-ant-real-key-abc123",
			},
		},
	}
	reconciler, _ := reconcileClaudeCodeAgent(t, agent)

	secret := &corev1.Secret{}
	require.NoError(t, reconciler.Client.Get(context.Background(), types.NamespacedName{
		Name:      "cc-inline-runtime",
		Namespace: "default",
	}, secret), "managed runtime Secret must be created for inline APIKey")

	assert.Equal(t, "sk-ant-real-key-abc123", string(secret.Data["ANTHROPIC_API_KEY"]))
}

func TestReconcileRuntimeSecret_ClaudeCode_APIKeyRef(t *testing.T) {
	// claudeCode.apiKeyRef → envFrom reference added, no managed secret
	agent := &langopv1alpha1.LanguageAgent{
		ObjectMeta: metav1.ObjectMeta{Name: "cc-ref", Namespace: "default"},
		Spec: langopv1alpha1.LanguageAgentSpec{
			Image: "ghcr.io/language-operator/agent:latest",
			Workspace: &langopv1alpha1.WorkspaceSpec{
				Enabled: func() *bool { b := false; return &b }(),
			},
			ClaudeCode: &langopv1alpha1.ClaudeCodeConfig{
				Enabled:   true,
				APIKeyRef: &langopv1alpha1.RuntimeSecretRef{Name: "my-anthropic-secret"},
			},
		},
	}
	reconciler, _ := reconcileClaudeCodeAgent(t, agent)

	// No managed secret when using APIKeyRef
	secret := &corev1.Secret{}
	err := reconciler.Client.Get(context.Background(), types.NamespacedName{
		Name:      "cc-ref-runtime",
		Namespace: "default",
	}, secret)
	assert.True(t, err != nil, "no managed runtime Secret when using APIKeyRef")

	// envFrom reference is injected into workingAgent
	working := agent.DeepCopy()
	err = reconciler.reconcileRuntimeSecret(context.Background(), agent, working)
	require.NoError(t, err)

	var found bool
	for _, ef := range working.Spec.Deployment.EnvFrom {
		if ef.SecretRef != nil && ef.SecretRef.Name == "my-anthropic-secret" {
			found = true
		}
	}
	assert.True(t, found, "envFrom references the APIKeyRef secret")
}

func TestReconcileRuntimeSecret_ClaudeCode_MaxTurns(t *testing.T) {
	// claudeCode.maxTurns set → CLAUDE_CODE_MAX_TURNS literal env var
	maxTurns := int32(5)
	agent := &langopv1alpha1.LanguageAgent{
		ObjectMeta: metav1.ObjectMeta{Name: "cc-maxturns", Namespace: "default"},
		Spec: langopv1alpha1.LanguageAgentSpec{
			Image: "ghcr.io/language-operator/agent:latest",
			Workspace: &langopv1alpha1.WorkspaceSpec{
				Enabled: func() *bool { b := false; return &b }(),
			},
			ClaudeCode: &langopv1alpha1.ClaudeCodeConfig{
				Enabled:  true,
				MaxTurns: &maxTurns,
			},
		},
	}
	reconciler, _ := reconcileClaudeCodeAgent(t, agent)

	working := agent.DeepCopy()
	err := reconciler.reconcileRuntimeSecret(context.Background(), agent, working)
	require.NoError(t, err)

	var found bool
	for _, e := range working.Spec.Deployment.Env {
		if e.Name == "CLAUDE_CODE_MAX_TURNS" && e.Value == "5" {
			found = true
		}
	}
	assert.True(t, found, "CLAUDE_CODE_MAX_TURNS=5 injected when maxTurns is set")
}
