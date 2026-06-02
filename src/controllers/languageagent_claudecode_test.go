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
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/tools/record"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

func TestReconcileRuntimeSecret_ClaudeCode_MaxTurns(t *testing.T) {
	maxTurns := int32(5)
	agent := &langopv1alpha1.LanguageAgent{
		ObjectMeta: metav1.ObjectMeta{Name: "cc-maxturns", Namespace: "default"},
		Spec: langopv1alpha1.LanguageAgentSpec{
			Image: "ghcr.io/language-operator/agent:latest",
			Workspace: &langopv1alpha1.WorkspaceSpec{
				Enabled: func() *bool { b := false; return &b }(),
			},
			ClaudeCode: &langopv1alpha1.ClaudeCodeConfig{
				MaxTurns: &maxTurns,
			},
		},
	}

	scheme := testutil.SetupTestScheme(t)
	cluster := gen.ReadyCluster(agent.Namespace)
	recorder := record.NewFakeRecorder(10)
	fakeClient := fake.NewClientBuilder().WithScheme(scheme).WithObjects(cluster, agent).WithStatusSubresource(agent).Build()
	reconciler := &LanguageAgentReconciler{
		Client:          fakeClient,
		Scheme:          scheme,
		Log:             logr.Discard(),
		Recorder:        recorder,
		EventManager:    events.NewEventManager(recorder),
		RegistryManager: &mockRegistryManager{},
	}

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
