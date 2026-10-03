package controllers

import (
	"context"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"

	langopv1alpha1 "github.com/language-operator/language-operator/api/v1alpha1"
	"github.com/language-operator/language-operator/controllers/testutil"
	"github.com/language-operator/language-operator/internal/testutil/gen"
)

func TestGatewayModelName(t *testing.T) {
	named := gen.LanguageModel("gpt", "ns", gen.SetModelName("gpt-4o"))
	wildcard := gen.LanguageModel("openrouter", "ns", gen.SetModelName("*"))

	name, err := gatewayModelName(named, langopv1alpha1.ModelReference{Name: "gpt"})
	require.NoError(t, err)
	assert.Equal(t, "gpt-4o", name)

	name, err = gatewayModelName(wildcard, langopv1alpha1.ModelReference{Name: "openrouter", Model: "anthropic/claude-sonnet-4.5"})
	require.NoError(t, err)
	assert.Equal(t, "openrouter/anthropic/claude-sonnet-4.5", name, "the gateway registers a wildcard as <name>/*")

	_, err = gatewayModelName(wildcard, langopv1alpha1.ModelReference{Name: "openrouter"})
	assert.ErrorContains(t, err, "is a wildcard")

	_, err = gatewayModelName(named, langopv1alpha1.ModelReference{Name: "gpt", Model: "gpt-4o-mini"})
	assert.ErrorContains(t, err, "is not a wildcard")
}

// An agent using a wildcard LanguageModel gets "<name>/<model>" everywhere it
// learns model names: LLM_MODEL and config.yaml.
func TestAgentUsesAModelFromAWildcard(t *testing.T) {
	scheme := testutil.SetupTestScheme(t)
	model := gen.LanguageModel("openrouter", "default", gen.SetModelProvider("openai-compatible"),
		gen.SetModelName("*"), gen.SetModelEndpoint("https://openrouter.ai/api/v1"))
	agent := gen.LanguageAgent("router-agent", "default")
	agent.Spec.Models = []langopv1alpha1.ModelReference{{Name: "openrouter", Model: "anthropic/claude-sonnet-4.5"}}

	cfg := parseAgentConfigMap(t, scheme, gen.ReadyCluster("default"), model, agent)
	require.Contains(t, cfg.Models, "openrouter")
	assert.Equal(t, "openrouter/anthropic/claude-sonnet-4.5", cfg.Models["openrouter"].Model)

	// parseAgentConfigMap stored these in its own client; start from clean copies.
	agent, model = agent.DeepCopy(), model.DeepCopy()
	agent.ResourceVersion, model.ResourceVersion = "", ""
	r, fc := newModeReconciler(t, agent)
	require.NoError(t, fc.Create(context.Background(), model))
	reconcileTwice(t, r, agent.Name, agent.Namespace)
	podSpec, _ := agentPodView(t, fc, agent.Name, agent.Namespace)
	var llm string
	for _, e := range podSpec.Containers[0].Env {
		if e.Name == "LLM_MODEL" {
			llm = e.Value
		}
	}
	assert.Equal(t, "openrouter/anthropic/claude-sonnet-4.5", llm)
}

func TestAgentReferencingAWildcardWithoutAModelFails(t *testing.T) {
	model := gen.LanguageModel("openrouter", "default", gen.SetModelProvider("openai-compatible"),
		gen.SetModelName("*"), gen.SetModelEndpoint("https://openrouter.ai/api/v1"))
	agent := gen.LanguageAgent("no-model-agent", "default")
	agent.Spec.Models = []langopv1alpha1.ModelReference{{Name: "openrouter"}}
	r, fc := newModeReconciler(t, agent)
	require.NoError(t, fc.Create(context.Background(), model))

	req := ctrl.Request{NamespacedName: types.NamespacedName{Name: agent.Name, Namespace: agent.Namespace}}
	_, _ = r.Reconcile(context.Background(), req) // adds the finalizer
	_, err := r.Reconcile(context.Background(), req)
	require.Error(t, err)
	assert.True(t, strings.Contains(err.Error(), "is a wildcard"), err.Error())
}
