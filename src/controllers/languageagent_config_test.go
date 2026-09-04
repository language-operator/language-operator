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
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/tools/record"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/yaml"
)

func TestLanguageAgentController_EnvVarInjection(t *testing.T) {
	scheme := testutil.SetupTestScheme(t)

	agent := &langopv1alpha1.LanguageAgent{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "env-agent",
			Namespace: "default",
		},
		Spec: langopv1alpha1.LanguageAgentSpec{
			Image:        "ghcr.io/language-operator/agent:latest",
			Instructions: "test instructions",
		},
	}

	cluster := gen.ReadyCluster("default")
	cluster.UID = "test-cluster-uid-1234"

	fakeClient := fake.NewClientBuilder().
		WithScheme(scheme).
		WithObjects(cluster, agent).
		WithStatusSubresource(agent).
		Build()

	reconciler := &LanguageAgentReconciler{
		Client:          fakeClient,
		Scheme:          scheme,
		Log:             logr.Discard(),
		Recorder:        &record.FakeRecorder{},
		RegistryManager: &mockRegistryManager{},
	}

	ctx := context.Background()
	_, err := reconciler.Reconcile(ctx, ctrl.Request{
		NamespacedName: types.NamespacedName{Name: agent.Name, Namespace: agent.Namespace},
	})
	if err != nil {
		t.Fatalf("Reconcile failed: %v", err)
	}

	podSpec, _ := agentPodView(t, fakeClient, agent.Name, agent.Namespace)

	if len(podSpec.Containers) == 0 {
		t.Fatal("No containers in workflow template")
	}

	envMap := make(map[string]string)
	for _, e := range podSpec.Containers[0].Env {
		envMap[e.Name] = e.Value
	}

	if envMap["AGENT_NAME"] != agent.Name {
		t.Errorf("Expected AGENT_NAME=%s, got %s", agent.Name, envMap["AGENT_NAME"])
	}
	if envMap["AGENT_NAMESPACE"] != agent.Namespace {
		t.Errorf("Expected AGENT_NAMESPACE=%s, got %s", agent.Namespace, envMap["AGENT_NAMESPACE"])
	}
	// AGENT_UUID is empty until status is populated; key must still be present
	if _, ok := envMap["AGENT_UUID"]; !ok {
		t.Error("Expected AGENT_UUID env var to be present")
	}
	if envMap["AGENT_CLUSTER_NAME"] != "default" {
		t.Errorf("Expected AGENT_CLUSTER_NAME=default, got %s", envMap["AGENT_CLUSTER_NAME"])
	}
	if envMap["AGENT_CLUSTER_UUID"] != "test-cluster-uid-1234" {
		t.Errorf("Expected AGENT_CLUSTER_UUID=test-cluster-uid-1234, got %s", envMap["AGENT_CLUSTER_UUID"])
	}
}

func TestLanguageAgentController_ContractEnvVars(t *testing.T) {
	agentRequest := func(name string) ctrl.Request {
		return ctrl.Request{NamespacedName: types.NamespacedName{Name: name, Namespace: "default"}}
	}

	t.Run("AGENT_INSTRUCTIONS set from spec.instructions", func(t *testing.T) {
		scheme := testutil.SetupTestScheme(t)
		agent := &langopv1alpha1.LanguageAgent{
			ObjectMeta: metav1.ObjectMeta{Name: "instr-agent", Namespace: "default"},
			Spec: langopv1alpha1.LanguageAgentSpec{
				Image:        "ghcr.io/language-operator/agent:latest",
				Instructions: "do the thing",
			},
		}
		fakeClient := fake.NewClientBuilder().
			WithScheme(scheme).
			WithObjects(gen.ReadyCluster("default"), agent).
			WithStatusSubresource(agent).
			Build()
		reconciler := &LanguageAgentReconciler{
			Client:          fakeClient,
			Scheme:          scheme,
			Log:             logr.Discard(),
			Recorder:        &record.FakeRecorder{},
			RegistryManager: &mockRegistryManager{},
		}
		ctx := context.Background()
		_, err := reconciler.Reconcile(ctx, agentRequest(agent.Name))
		require.NoError(t, err)

		podSpec, _ := agentPodView(t, fakeClient, agent.Name, agent.Namespace)
		envMap := make(map[string]string)
		for _, e := range podSpec.Containers[0].Env {
			envMap[e.Name] = e.Value
		}
		assert.Equal(t, "do the thing", envMap["AGENT_INSTRUCTIONS"], "AGENT_INSTRUCTIONS must equal spec.instructions")
	})

	t.Run("MODEL_ENDPOINT and LLM_MODEL set from spec.models", func(t *testing.T) {
		scheme := testutil.SetupTestScheme(t)
		model := gen.LanguageModel("claude-sonnet", "default")
		agent := &langopv1alpha1.LanguageAgent{
			ObjectMeta: metav1.ObjectMeta{Name: "model-agent", Namespace: "default"},
			Spec: langopv1alpha1.LanguageAgentSpec{
				Image:  "ghcr.io/language-operator/agent:latest",
				Models: []langopv1alpha1.ModelReference{{Name: "claude-sonnet"}},
			},
		}
		fakeClient := fake.NewClientBuilder().
			WithScheme(scheme).
			WithObjects(gen.ReadyCluster("default"), agent, model).
			WithStatusSubresource(agent).
			Build()
		reconciler := &LanguageAgentReconciler{
			Client:          fakeClient,
			Scheme:          scheme,
			Log:             logr.Discard(),
			Recorder:        &record.FakeRecorder{},
			RegistryManager: &mockRegistryManager{},
		}
		ctx := context.Background()
		_, err := reconciler.Reconcile(ctx, agentRequest(agent.Name))
		require.NoError(t, err)

		podSpec, _ := agentPodView(t, fakeClient, agent.Name, agent.Namespace)
		envMap := make(map[string]string)
		for _, e := range podSpec.Containers[0].Env {
			envMap[e.Name] = e.Value
		}
		assert.Equal(t, "http://gateway.default.svc.cluster.local:8000", envMap["MODEL_ENDPOINT"])
		assert.Equal(t, model.Spec.ModelName, envMap["LLM_MODEL"])
	})

	t.Run("MCP_SERVERS set from spec.tools", func(t *testing.T) {
		scheme := testutil.SetupTestScheme(t)
		tool := gen.LanguageTool("mem0", "default")
		agent := &langopv1alpha1.LanguageAgent{
			ObjectMeta: metav1.ObjectMeta{Name: "tool-agent", Namespace: "default"},
			Spec: langopv1alpha1.LanguageAgentSpec{
				Image: "ghcr.io/language-operator/agent:latest",
				Tools: []langopv1alpha1.ToolReference{{Name: "mem0"}},
			},
		}
		fakeClient := fake.NewClientBuilder().
			WithScheme(scheme).
			WithObjects(gen.ReadyCluster("default"), agent, tool).
			WithStatusSubresource(agent).
			Build()
		reconciler := &LanguageAgentReconciler{
			Client:          fakeClient,
			Scheme:          scheme,
			Log:             logr.Discard(),
			Recorder:        &record.FakeRecorder{},
			RegistryManager: &mockRegistryManager{},
		}
		ctx := context.Background()
		_, err := reconciler.Reconcile(ctx, agentRequest(agent.Name))
		require.NoError(t, err)

		podSpec, _ := agentPodView(t, fakeClient, agent.Name, agent.Namespace)
		envMap := make(map[string]string)
		for _, e := range podSpec.Containers[0].Env {
			envMap[e.Name] = e.Value
		}
		// Default port is 0 → resolved to 8080 in resolveTools
		assert.Equal(t, "http://mem0.default.svc.cluster.local:8080/mcp", envMap["MCP_SERVERS"])
	})

	t.Run("AGENT_INSTRUCTIONS absent when spec.instructions is empty", func(t *testing.T) {
		scheme := testutil.SetupTestScheme(t)
		agent := &langopv1alpha1.LanguageAgent{
			ObjectMeta: metav1.ObjectMeta{Name: "no-instr-agent", Namespace: "default"},
			Spec: langopv1alpha1.LanguageAgentSpec{
				Image: "ghcr.io/language-operator/agent:latest",
				// Instructions intentionally empty
			},
		}
		fakeClient := fake.NewClientBuilder().
			WithScheme(scheme).
			WithObjects(gen.ReadyCluster("default"), agent).
			WithStatusSubresource(agent).
			Build()
		reconciler := &LanguageAgentReconciler{
			Client:          fakeClient,
			Scheme:          scheme,
			Log:             logr.Discard(),
			Recorder:        &record.FakeRecorder{},
			RegistryManager: &mockRegistryManager{},
		}
		ctx := context.Background()
		_, err := reconciler.Reconcile(ctx, agentRequest(agent.Name))
		require.NoError(t, err)

		podSpec, _ := agentPodView(t, fakeClient, agent.Name, agent.Namespace)
		envMap := make(map[string]string)
		for _, e := range podSpec.Containers[0].Env {
			envMap[e.Name] = e.Value
		}
		_, present := envMap["AGENT_INSTRUCTIONS"]
		assert.False(t, present, "AGENT_INSTRUCTIONS must be absent when spec.instructions is empty")
	})

	t.Run("OTEL env vars injected when OTEL_EXPORTER_OTLP_ENDPOINT is set", func(t *testing.T) {
		t.Setenv("OTEL_EXPORTER_OTLP_ENDPOINT", "http://otel:4317")
		t.Setenv("OTEL_RESOURCE_ATTRIBUTES", "k=v")
		t.Setenv("OTEL_TRACES_SAMPLER", "parentbased_traceidratio")
		t.Setenv("OTEL_TRACES_SAMPLER_ARG", "0.1")

		scheme := testutil.SetupTestScheme(t)
		agent := &langopv1alpha1.LanguageAgent{
			ObjectMeta: metav1.ObjectMeta{Name: "otel-agent", Namespace: "default"},
			Spec: langopv1alpha1.LanguageAgentSpec{
				Image: "ghcr.io/language-operator/agent:latest",
			},
		}
		fakeClient := fake.NewClientBuilder().
			WithScheme(scheme).
			WithObjects(gen.ReadyCluster("default"), agent).
			WithStatusSubresource(agent).
			Build()
		reconciler := &LanguageAgentReconciler{
			Client:          fakeClient,
			Scheme:          scheme,
			Log:             logr.Discard(),
			Recorder:        &record.FakeRecorder{},
			RegistryManager: &mockRegistryManager{},
		}
		ctx := context.Background()
		_, err := reconciler.Reconcile(ctx, agentRequest(agent.Name))
		require.NoError(t, err)

		podSpec, _ := agentPodView(t, fakeClient, agent.Name, agent.Namespace)
		envMap := make(map[string]string)
		for _, e := range podSpec.Containers[0].Env {
			envMap[e.Name] = e.Value
		}
		assert.Equal(t, "http://otel:4317", envMap["OTEL_EXPORTER_OTLP_ENDPOINT"])
		assert.Equal(t, "agent-otel-agent", envMap["OTEL_SERVICE_NAME"])
		assert.Equal(t, "k=v", envMap["OTEL_RESOURCE_ATTRIBUTES"])
		assert.Equal(t, "parentbased_traceidratio", envMap["OTEL_TRACES_SAMPLER"])
		assert.Equal(t, "0.1", envMap["OTEL_TRACES_SAMPLER_ARG"])
	})

	t.Run("OTEL env vars absent when OTEL_EXPORTER_OTLP_ENDPOINT is unset", func(t *testing.T) {
		scheme := testutil.SetupTestScheme(t)
		agent := &langopv1alpha1.LanguageAgent{
			ObjectMeta: metav1.ObjectMeta{Name: "no-otel-agent", Namespace: "default"},
			Spec: langopv1alpha1.LanguageAgentSpec{
				Image: "ghcr.io/language-operator/agent:latest",
			},
		}
		fakeClient := fake.NewClientBuilder().
			WithScheme(scheme).
			WithObjects(gen.ReadyCluster("default"), agent).
			WithStatusSubresource(agent).
			Build()
		reconciler := &LanguageAgentReconciler{
			Client:          fakeClient,
			Scheme:          scheme,
			Log:             logr.Discard(),
			Recorder:        &record.FakeRecorder{},
			RegistryManager: &mockRegistryManager{},
		}
		ctx := context.Background()
		_, err := reconciler.Reconcile(ctx, agentRequest(agent.Name))
		require.NoError(t, err)

		podSpec, _ := agentPodView(t, fakeClient, agent.Name, agent.Namespace)
		envMap := make(map[string]string)
		for _, e := range podSpec.Containers[0].Env {
			envMap[e.Name] = e.Value
		}
		_, present := envMap["OTEL_EXPORTER_OTLP_ENDPOINT"]
		assert.False(t, present, "OTEL_EXPORTER_OTLP_ENDPOINT must be absent when not set in operator env")
	})

	t.Run("MCP_SERVERS absent when no tools resolved", func(t *testing.T) {
		scheme := testutil.SetupTestScheme(t)
		agent := &langopv1alpha1.LanguageAgent{
			ObjectMeta: metav1.ObjectMeta{Name: "no-tools-agent", Namespace: "default"},
			Spec: langopv1alpha1.LanguageAgentSpec{
				Image: "ghcr.io/language-operator/agent:latest",
				// No Tools specified
			},
		}
		fakeClient := fake.NewClientBuilder().
			WithScheme(scheme).
			WithObjects(gen.ReadyCluster("default"), agent).
			WithStatusSubresource(agent).
			Build()
		reconciler := &LanguageAgentReconciler{
			Client:          fakeClient,
			Scheme:          scheme,
			Log:             logr.Discard(),
			Recorder:        &record.FakeRecorder{},
			RegistryManager: &mockRegistryManager{},
		}
		ctx := context.Background()
		_, err := reconciler.Reconcile(ctx, agentRequest(agent.Name))
		require.NoError(t, err)

		podSpec, _ := agentPodView(t, fakeClient, agent.Name, agent.Namespace)
		envMap := make(map[string]string)
		for _, e := range podSpec.Containers[0].Env {
			envMap[e.Name] = e.Value
		}
		_, present := envMap["MCP_SERVERS"]
		assert.False(t, present, "MCP_SERVERS must be absent when no tools are resolved")
	})

	t.Run("init container receives MODEL_ENDPOINT and LLM_MODEL", func(t *testing.T) {
		scheme := testutil.SetupTestScheme(t)
		model := gen.LanguageModel("claude-sonnet", "default")
		agent := &langopv1alpha1.LanguageAgent{
			ObjectMeta: metav1.ObjectMeta{Name: "init-env-agent", Namespace: "default"},
			Spec: langopv1alpha1.LanguageAgentSpec{
				Image:  "ghcr.io/language-operator/agent:latest",
				Models: []langopv1alpha1.ModelReference{{Name: "claude-sonnet"}},
				Deployment: langopv1alpha1.DeploymentSpec{
					InitContainers: []corev1.Container{
						{Name: "setup", Image: "busybox:latest"},
					},
				},
			},
		}
		fakeClient := fake.NewClientBuilder().
			WithScheme(scheme).
			WithObjects(gen.ReadyCluster("default"), agent, model).
			WithStatusSubresource(agent).
			Build()
		reconciler := &LanguageAgentReconciler{
			Client:          fakeClient,
			Scheme:          scheme,
			Log:             logr.Discard(),
			Recorder:        &record.FakeRecorder{},
			RegistryManager: &mockRegistryManager{},
		}
		ctx := context.Background()
		_, err := reconciler.Reconcile(ctx, agentRequest(agent.Name))
		require.NoError(t, err)

		podSpec, _ := agentPodView(t, fakeClient, agent.Name, agent.Namespace)

		var setupContainer *corev1.Container
		for i := range podSpec.InitContainers {
			if podSpec.InitContainers[i].Name == "setup" {
				setupContainer = &podSpec.InitContainers[i]
				break
			}
		}
		require.NotNil(t, setupContainer, "expected init container 'setup' in workflow template")

		initEnvMap := make(map[string]string)
		for _, e := range setupContainer.Env {
			initEnvMap[e.Name] = e.Value
		}
		assert.Equal(t, "http://gateway.default.svc.cluster.local:8000", initEnvMap["MODEL_ENDPOINT"])
		assert.Equal(t, model.Spec.ModelName, initEnvMap["LLM_MODEL"])
	})
}

func TestLanguageAgentController_ResourceRequests(t *testing.T) {
	scheme := testutil.SetupTestScheme(t)

	agent := &langopv1alpha1.LanguageAgent{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "resource-agent",
			Namespace: "default",
		},
		Spec: langopv1alpha1.LanguageAgentSpec{
			Image: "ghcr.io/language-operator/agent:latest",
			Deployment: langopv1alpha1.DeploymentSpec{
				Resources: corev1.ResourceRequirements{
					Requests: corev1.ResourceList{
						corev1.ResourceCPU:    resource.MustParse("200m"),
						corev1.ResourceMemory: resource.MustParse("256Mi"),
					},
					Limits: corev1.ResourceList{
						corev1.ResourceCPU:    resource.MustParse("500m"),
						corev1.ResourceMemory: resource.MustParse("512Mi"),
					},
				},
			},
		},
	}

	fakeClient := fake.NewClientBuilder().
		WithScheme(scheme).
		WithObjects(gen.ReadyCluster("default"), agent).
		WithStatusSubresource(agent).
		Build()

	reconciler := &LanguageAgentReconciler{
		Client:          fakeClient,
		Scheme:          scheme,
		Log:             logr.Discard(),
		Recorder:        &record.FakeRecorder{},
		RegistryManager: &mockRegistryManager{},
	}

	ctx := context.Background()
	_, err := reconciler.Reconcile(ctx, ctrl.Request{
		NamespacedName: types.NamespacedName{Name: agent.Name, Namespace: agent.Namespace},
	})
	if err != nil {
		t.Fatalf("Reconcile failed: %v", err)
	}

	podSpec, _ := agentPodView(t, fakeClient, agent.Name, agent.Namespace)

	if len(podSpec.Containers) == 0 {
		t.Fatal("No containers in workflow template")
	}

	res := podSpec.Containers[0].Resources
	if res.Requests.Cpu().Cmp(resource.MustParse("200m")) != 0 {
		t.Errorf("Expected CPU request 200m, got %s", res.Requests.Cpu().String())
	}
	if res.Requests.Memory().Cmp(resource.MustParse("256Mi")) != 0 {
		t.Errorf("Expected memory request 256Mi, got %s", res.Requests.Memory().String())
	}
	if res.Limits.Cpu().Cmp(resource.MustParse("500m")) != 0 {
		t.Errorf("Expected CPU limit 500m, got %s", res.Limits.Cpu().String())
	}
	if res.Limits.Memory().Cmp(resource.MustParse("512Mi")) != 0 {
		t.Errorf("Expected memory limit 512Mi, got %s", res.Limits.Memory().String())
	}
}

// --- Group 1: Pure/stateless functions ---

func TestLanguageAgentController_HashString(t *testing.T) {
	h1 := hashString("hello")
	h2 := hashString("hello")
	h3 := hashString("world")

	if h1 != h2 {
		t.Errorf("hashString not deterministic: %s vs %s", h1, h2)
	}
	if h1 == h3 {
		t.Error("hashString should differ for different inputs")
	}
	if h1 == "" {
		t.Error("hashString returned empty string")
	}
}

func TestLanguageAgentController_GetNames(t *testing.T) {
	r := &LanguageAgentReconciler{}

	t.Run("get_tool_names", func(t *testing.T) {
		agent := &langopv1alpha1.LanguageAgent{
			Spec: langopv1alpha1.LanguageAgentSpec{
				Tools: []langopv1alpha1.ToolReference{
					{Name: "tool-a"},
					{Name: "tool-b"},
				},
			},
		}
		names := r.getToolNames(agent)
		if len(names) != 2 || names[0] != "tool-a" || names[1] != "tool-b" {
			t.Errorf("getToolNames: got %v", names)
		}
	})

	t.Run("get_model_names", func(t *testing.T) {
		agent := &langopv1alpha1.LanguageAgent{
			Spec: langopv1alpha1.LanguageAgentSpec{
				Models: []langopv1alpha1.ModelReference{
					{Name: "model-x"},
				},
			},
		}
		names := r.getModelNames(agent)
		if len(names) != 1 || names[0] != "model-x" {
			t.Errorf("getModelNames: got %v", names)
		}
	})

	t.Run("get_persona_names", func(t *testing.T) {
		agent := &langopv1alpha1.LanguageAgent{
			Spec: langopv1alpha1.LanguageAgentSpec{
				Persona: "persona-1",
			},
		}
		names := r.getPersonaNames(agent)
		if len(names) != 1 || names[0] != "persona-1" {
			t.Errorf("getPersonaNames: got %v", names)
		}
	})

	t.Run("empty_refs", func(t *testing.T) {
		agent := &langopv1alpha1.LanguageAgent{}
		if got := r.getToolNames(agent); len(got) != 0 {
			t.Errorf("expected empty tool names, got %v", got)
		}
		if got := r.getModelNames(agent); len(got) != 0 {
			t.Errorf("expected empty model names, got %v", got)
		}
		if got := r.getPersonaNames(agent); len(got) != 0 {
			t.Errorf("expected empty persona names, got %v", got)
		}
	})
}

func TestLanguageAgentController_ConfigMapContent(t *testing.T) {
	scheme := testutil.SetupTestScheme(t)

	t.Run("model_section", func(t *testing.T) {
		model := gen.LanguageModel("claude-model", "default",
			gen.SetModelProvider("anthropic"),
			gen.SetModelName("claude-sonnet-4-5"),
		)
		agent := gen.LanguageAgent("model-agent", "default",
			gen.SetAgentInstructions("do the thing"),
		)
		agent.Spec.Models = []langopv1alpha1.ModelReference{
			{Name: "claude-model", Role: "primary"},
		}

		cfg := parseAgentConfigMap(t, scheme, gen.ReadyCluster("default"), model, agent)

		assert.Equal(t, "do the thing", cfg.Instructions)
		require.Contains(t, cfg.Models, "claude-model", "config.yaml missing model entry")
		m := cfg.Models["claude-model"]
		assert.Equal(t, "http://gateway.default.svc.cluster.local:8000", m.Endpoint)
		assert.Equal(t, "anthropic", m.Provider)
		assert.Equal(t, "claude-sonnet-4-5", m.Model)
		assert.Equal(t, "primary", m.Role)
	})

	t.Run("tool_service_mode", func(t *testing.T) {
		tool := gen.LanguageTool("search-tool", "default") // service mode by default, port 0 → 8080
		agent := gen.LanguageAgent("tool-agent", "default")
		agent.Spec.Tools = []langopv1alpha1.ToolReference{{Name: "search-tool"}}

		cfg := parseAgentConfigMap(t, scheme, gen.ReadyCluster("default"), tool, agent)

		require.Contains(t, cfg.Tools, "search-tool", "config.yaml missing tool entry")
		tool_cfg := cfg.Tools["search-tool"]
		assert.Equal(t, "http://search-tool.default.svc.cluster.local:8080/mcp", tool_cfg.Endpoint)
		assert.Equal(t, "mcp", tool_cfg.Protocol)
	})

	t.Run("tool_sidecar_mode", func(t *testing.T) {
		tool := gen.LanguageTool("sidecar-tool", "default",
			gen.SetToolDeploymentMode("sidecar"),
		)
		agent := gen.LanguageAgent("sidecar-agent", "default")
		agent.Spec.Tools = []langopv1alpha1.ToolReference{{Name: "sidecar-tool"}}

		cfg := parseAgentConfigMap(t, scheme, gen.ReadyCluster("default"), tool, agent)

		require.Contains(t, cfg.Tools, "sidecar-tool", "config.yaml missing sidecar tool entry")
		assert.Equal(t, "http://localhost:8080/mcp", cfg.Tools["sidecar-tool"].Endpoint)
		assert.Equal(t, "mcp", cfg.Tools["sidecar-tool"].Protocol)
	})

	t.Run("persona_section", func(t *testing.T) {
		persona := gen.LanguagePersona("my-persona", "default",
			gen.SetPersonaTone("professional"),
			gen.SetPersonaPersonality("concise and precise"),
		)
		persona.Status.Phase = events.PhaseStatusReady

		agent := gen.LanguageAgent("persona-agent", "default")
		agent.Spec.Persona = "my-persona"

		cfg := parseAgentConfigMap(t, scheme, gen.ReadyCluster("default"), persona, agent)

		require.Len(t, cfg.Personas, 1, "config.yaml should contain exactly one persona")
		p := cfg.Personas[0]
		assert.Equal(t, "my-persona", p.Name)
		assert.Equal(t, "professional", p.Tone)
		assert.Equal(t, "concise and precise", p.Personality)
	})
}

func TestLanguageAgentController_FetchPersona(t *testing.T) {
	scheme := testutil.SetupTestScheme(t)

	t.Run("no_refs", func(t *testing.T) {
		agent := &langopv1alpha1.LanguageAgent{
			ObjectMeta: metav1.ObjectMeta{Name: "agent", Namespace: "default"},
		}
		fakeClient := fake.NewClientBuilder().WithScheme(scheme).Build()
		r := &LanguageAgentReconciler{Client: fakeClient,
			Scheme: scheme, Log: logr.Discard()}

		persona, err := r.fetchPersona(context.Background(), agent)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if persona != nil {
			t.Error("expected nil persona when no refs")
		}
	})

	t.Run("single_ready_persona", func(t *testing.T) {
		p := gen.LanguagePersona("my-persona", "default")
		p.Status.Phase = events.PhaseStatusReady

		agent := &langopv1alpha1.LanguageAgent{
			ObjectMeta: metav1.ObjectMeta{Name: "agent", Namespace: "default"},
			Spec: langopv1alpha1.LanguageAgentSpec{
				Persona: "my-persona",
			},
		}

		fakeClient := fake.NewClientBuilder().WithScheme(scheme).WithObjects(p).Build()
		r := &LanguageAgentReconciler{Client: fakeClient,
			Scheme: scheme, Log: logr.Discard()}

		persona, err := r.fetchPersona(context.Background(), agent)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if persona == nil {
			t.Fatal("expected persona to be returned")
		}
	})

	t.Run("persona_not_ready", func(t *testing.T) {
		p := gen.LanguagePersona("pending-persona", "default")
		p.Status.Phase = events.PhaseStatusPending

		agent := &langopv1alpha1.LanguageAgent{
			ObjectMeta: metav1.ObjectMeta{Name: "agent", Namespace: "default"},
			Spec: langopv1alpha1.LanguageAgentSpec{
				Persona: "pending-persona",
			},
		}

		fakeClient := fake.NewClientBuilder().WithScheme(scheme).WithObjects(p).Build()
		r := &LanguageAgentReconciler{Client: fakeClient,
			Scheme: scheme, Log: logr.Discard()}

		_, err := r.fetchPersona(context.Background(), agent)
		if err == nil {
			t.Error("expected error when persona not ready")
		}
	})

	t.Run("persona_not_found", func(t *testing.T) {
		agent := &langopv1alpha1.LanguageAgent{
			ObjectMeta: metav1.ObjectMeta{Name: "agent", Namespace: "default"},
			Spec: langopv1alpha1.LanguageAgentSpec{
				Persona: "missing",
			},
		}

		fakeClient := fake.NewClientBuilder().WithScheme(scheme).Build()
		r := &LanguageAgentReconciler{Client: fakeClient,
			Scheme: scheme, Log: logr.Discard()}

		_, err := r.fetchPersona(context.Background(), agent)
		if err == nil {
			t.Error("expected error when persona not found")
		}
	})
}

func TestLanguageAgentController_WorkspaceSeed_InitialFiles(t *testing.T) {
	agent := gen.LanguageAgent("seed-agent", "default",
		gen.SetAgentWorkspace("5Gi"),
		gen.SetAgentWorkspaceInitialFiles(map[string]string{
			"AGENT.md":    "# Hello",
			"memory.json": "{}",
		}),
	)
	r, fakeClient := newSeedReconciler(t, agent, gen.ReadyCluster("default"))
	reconcileTwice(t, r, agent.Name, agent.Namespace)

	ctx := context.Background()

	// Seed ConfigMap should exist with the initial files as data
	cm := &corev1.ConfigMap{}
	require.NoError(t, fakeClient.Get(ctx, types.NamespacedName{
		Name:      agent.Name + "-workspace-seed",
		Namespace: agent.Namespace,
	}, cm))
	assert.Equal(t, "# Hello", cm.Data["AGENT.md"])
	assert.Equal(t, "{}", cm.Data["memory.json"])

	// Deployment should have a workspace-seeder init container
	podSpec, _ := agentPodView(t, fakeClient, agent.Name, agent.Namespace)

	var seeder *corev1.Container
	for i := range podSpec.InitContainers {
		if podSpec.InitContainers[i].Name == "workspace-seeder" {
			seeder = &podSpec.InitContainers[i]
			break
		}
	}
	require.NotNil(t, seeder, "expected workspace-seeder init container")
	assert.Equal(t, "busybox:latest", seeder.Image)

	// Init container must mount the workspace PVC
	var hasWorkspace bool
	for _, vm := range seeder.VolumeMounts {
		if vm.Name == "workspace" {
			hasWorkspace = true
		}
	}
	assert.True(t, hasWorkspace, "workspace-seeder must mount workspace PVC")

	// seed-init volume mount must be present
	var hasSeedInit bool
	for _, vm := range seeder.VolumeMounts {
		if vm.Name == "workspace-seed-init" {
			hasSeedInit = true
		}
	}
	assert.True(t, hasSeedInit, "workspace-seeder must mount workspace-seed-init volume")

	// Pod volumes must include workspace-seed-init
	var hasVol bool
	for _, v := range podSpec.Volumes {
		if v.Name == "workspace-seed-init" {
			hasVol = true
		}
	}
	assert.True(t, hasVol, "pod volumes must include workspace-seed-init")

	// Seed init container runs before user init containers (it must be index 0 when no user containers)
	assert.Equal(t, "workspace-seeder", podSpec.InitContainers[0].Name)
}

func TestLanguageAgentController_WorkspaceSeed_SeedConfigMapRef(t *testing.T) {
	refCM := &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{Name: "my-seed", Namespace: "default"},
		Data:       map[string]string{"prompt.md": "be helpful"},
	}
	agent := gen.LanguageAgent("ref-agent", "default",
		gen.SetAgentWorkspace("5Gi"),
		gen.SetAgentWorkspaceSeedConfigMapRef("my-seed"),
	)
	r, fakeClient := newSeedReconciler(t, agent, gen.ReadyCluster("default"), refCM)
	reconcileTwice(t, r, agent.Name, agent.Namespace)

	ctx := context.Background()

	// No operator-managed seed ConfigMap (no InitialFiles)
	cm := &corev1.ConfigMap{}
	err := fakeClient.Get(ctx, types.NamespacedName{Name: agent.Name + "-workspace-seed", Namespace: agent.Namespace}, cm)
	assert.True(t, errors.IsNotFound(err), "seed ConfigMap should not be created when only SeedConfigMapRef is set")

	podSpec, _ := agentPodView(t, fakeClient, agent.Name, agent.Namespace)

	var seeder *corev1.Container
	for i := range podSpec.InitContainers {
		if podSpec.InitContainers[i].Name == "workspace-seeder" {
			seeder = &podSpec.InitContainers[i]
			break
		}
	}
	require.NotNil(t, seeder, "expected workspace-seeder init container for SeedConfigMapRef")

	var hasSeedRef bool
	for _, vm := range seeder.VolumeMounts {
		if vm.Name == "workspace-seed-ref" {
			hasSeedRef = true
		}
	}
	assert.True(t, hasSeedRef, "workspace-seeder must mount workspace-seed-ref volume")
}

func TestLanguageAgentController_WorkspaceSeed_Both(t *testing.T) {
	refCM := &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{Name: "extra-seed", Namespace: "default"},
		Data:       map[string]string{"extra.md": "extra content"},
	}
	agent := gen.LanguageAgent("both-agent", "default",
		gen.SetAgentWorkspace("5Gi"),
		gen.SetAgentWorkspaceInitialFiles(map[string]string{"AGENT.md": "# Agent"}),
		gen.SetAgentWorkspaceSeedConfigMapRef("extra-seed"),
	)
	r, fakeClient := newSeedReconciler(t, agent, gen.ReadyCluster("default"), refCM)
	reconcileTwice(t, r, agent.Name, agent.Namespace)

	podSpec, _ := agentPodView(t, fakeClient, agent.Name, agent.Namespace)

	var seeder *corev1.Container
	for i := range podSpec.InitContainers {
		if podSpec.InitContainers[i].Name == "workspace-seeder" {
			seeder = &podSpec.InitContainers[i]
			break
		}
	}
	require.NotNil(t, seeder)

	mountNames := map[string]bool{}
	for _, vm := range seeder.VolumeMounts {
		mountNames[vm.Name] = true
	}
	assert.True(t, mountNames["workspace"], "must mount workspace")
	assert.True(t, mountNames["workspace-seed-init"], "must mount workspace-seed-init")
	assert.True(t, mountNames["workspace-seed-ref"], "must mount workspace-seed-ref")
}

func TestLanguageAgentController_WorkspaceSeed_NoWorkspace(t *testing.T) {
	agent := gen.LanguageAgent("no-ws-agent", "default")
	r, fakeClient := newSeedReconciler(t, agent, gen.ReadyCluster("default"))
	reconcileTwice(t, r, agent.Name, agent.Namespace)

	ctx := context.Background()

	// No seed ConfigMap
	cm := &corev1.ConfigMap{}
	err := fakeClient.Get(ctx, types.NamespacedName{Name: agent.Name + "-workspace-seed", Namespace: agent.Namespace}, cm)
	assert.True(t, errors.IsNotFound(err), "no seed ConfigMap without workspace")

	podSpec, _ := agentPodView(t, fakeClient, agent.Name, agent.Namespace)
	for _, c := range podSpec.InitContainers {
		assert.NotEqual(t, "workspace-seeder", c.Name, "workspace-seeder must not be injected without workspace")
	}
}

func TestLanguageAgentController_WorkspaceSeed_ConditionSet(t *testing.T) {
	agent := gen.LanguageAgent("cond-seed-agent", "default",
		gen.SetAgentWorkspace("5Gi"),
		gen.SetAgentWorkspaceInitialFiles(map[string]string{"AGENT.md": "# Hello"}),
	)
	agent.Generation = 1
	r, fakeClient := newSeedReconciler(t, agent, gen.ReadyCluster("default"))
	reconcileTwice(t, r, agent.Name, agent.Namespace)

	ctx := context.Background()
	updated := &langopv1alpha1.LanguageAgent{}
	require.NoError(t, fakeClient.Get(ctx, types.NamespacedName{Name: agent.Name, Namespace: agent.Namespace}, updated))

	var seededCond *metav1.Condition
	for i := range updated.Status.Conditions {
		if updated.Status.Conditions[i].Type == langopv1alpha1.ConditionWorkspaceSeeded {
			seededCond = &updated.Status.Conditions[i]
			break
		}
	}
	require.NotNil(t, seededCond, "WorkspaceSeeded condition must be set")
	assert.Equal(t, metav1.ConditionTrue, seededCond.Status)
	assert.Equal(t, langopv1alpha1.ReasonWorkspaceSeedReady, seededCond.Reason)
}

// findInitContainer returns the named init container from a pod spec, or nil.
func findInitContainer(podSpec corev1.PodSpec, name string) *corev1.Container {
	for i := range podSpec.InitContainers {
		if podSpec.InitContainers[i].Name == name {
			return &podSpec.InitContainers[i]
		}
	}
	return nil
}

func TestDeriveRepoName(t *testing.T) {
	cases := map[string]string{
		"https://github.com/foo/bar.git":  "bar",
		"https://github.com/foo/bar":      "bar",
		"https://github.com/foo/bar.git/": "bar",
		"git@github.com:foo/bar.git":      "bar",
		"git@github.com:bar.git":          "bar",
		"ssh://git@host/foo/baz":          "baz",
	}
	for url, want := range cases {
		assert.Equal(t, want, deriveRepoName(url), "deriveRepoName(%q)", url)
	}
}

func TestLanguageAgentController_Repository_InitContainerAndWorkingDir(t *testing.T) {
	agent := gen.LanguageAgent("repo-agent", "default",
		gen.SetAgentWorkspace("5Gi"),
		gen.SetAgentRepository("https://github.com/foo/bar.git", "main", "", ""),
		gen.SetAgentRepositoryDepth(1),
	)
	r, fakeClient := newSeedReconciler(t, agent, gen.ReadyCluster("default"))
	reconcileTwice(t, r, agent.Name, agent.Namespace)

	podSpec, _ := agentPodView(t, fakeClient, agent.Name, agent.Namespace)

	repo := findInitContainer(podSpec, "repository")
	require.NotNil(t, repo, "expected repository init container")
	assert.Equal(t, "alpine/git:latest", repo.Image)

	// Clone-once semantics and ref/depth reflected in the script.
	require.Len(t, repo.Command, 3)
	script := repo.Command[2]
	assert.Contains(t, script, `if [ -d "$TARGET/.git" ]; then`, "clone-once guard must be present")
	assert.Contains(t, script, "/workspace/bar", "target dir derived from repo name")
	assert.Contains(t, script, "--branch", "ref must be honored")
	assert.Contains(t, script, "--depth 1", "depth must be honored")

	// Must mount the workspace PVC.
	var hasWorkspace bool
	for _, vm := range repo.VolumeMounts {
		if vm.Name == "workspace" {
			hasWorkspace = true
		}
	}
	assert.True(t, hasWorkspace, "repository init container must mount workspace")

	// Main container opens inside the cloned repo.
	assert.Equal(t, "/workspace/bar", podSpec.Containers[0].WorkingDir)

	// Init container order: workspace-seeder is absent (no seed), repository runs first.
	require.NotEmpty(t, podSpec.InitContainers)
	assert.Equal(t, "repository", podSpec.InitContainers[0].Name)
}

func TestLanguageAgentController_Repository_AgentRepoDirInjected(t *testing.T) {
	agent := gen.LanguageAgent("repo-env-agent", "default",
		gen.SetAgentWorkspace("5Gi"),
		gen.SetAgentWorkspaceMountPath("/data"),
		gen.SetAgentRepository("https://github.com/foo/bar.git", "", "src/app", ""),
	)
	r, fakeClient := newSeedReconciler(t, agent, gen.ReadyCluster("default"))
	reconcileTwice(t, r, agent.Name, agent.Namespace)

	podSpec, _ := agentPodView(t, fakeClient, agent.Name, agent.Namespace)

	hasRepoDir := func(env []corev1.EnvVar) bool {
		for _, e := range env {
			if e.Name == "AGENT_REPO_DIR" {
				assert.Equal(t, "/data/src/app", e.Value)
				return true
			}
		}
		return false
	}

	assert.True(t, hasRepoDir(podSpec.Containers[0].Env), "AGENT_REPO_DIR must be set on the main container")
	repo := findInitContainer(podSpec, "repository")
	require.NotNil(t, repo)
	// Explicit path overrides the derived name for WorkingDir too.
	assert.Equal(t, "/data/src/app", podSpec.Containers[0].WorkingDir)
}

func TestLanguageAgentController_Repository_SecretRefMounted(t *testing.T) {
	secret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{Name: "git-creds", Namespace: "default"},
		Data:       map[string][]byte{"token": []byte("abc")},
	}
	agent := gen.LanguageAgent("repo-secret-agent", "default",
		gen.SetAgentWorkspace("5Gi"),
		gen.SetAgentRepository("https://github.com/foo/bar.git", "", "", "git-creds"),
	)
	r, fakeClient := newSeedReconciler(t, agent, gen.ReadyCluster("default"), secret)
	reconcileTwice(t, r, agent.Name, agent.Namespace)

	podSpec, _ := agentPodView(t, fakeClient, agent.Name, agent.Namespace)

	repo := findInitContainer(podSpec, "repository")
	require.NotNil(t, repo)
	var hasCredMount bool
	for _, vm := range repo.VolumeMounts {
		if vm.Name == "repository-credentials" {
			assert.Equal(t, "/git-secret", vm.MountPath)
			hasCredMount = true
		}
	}
	assert.True(t, hasCredMount, "repository init container must mount git credentials")

	// Pod volume must reference the Secret.
	var credVol *corev1.Volume
	for i := range podSpec.Volumes {
		if podSpec.Volumes[i].Name == "repository-credentials" {
			credVol = &podSpec.Volumes[i]
		}
	}
	require.NotNil(t, credVol, "pod volumes must include repository-credentials")
	require.NotNil(t, credVol.Secret)
	assert.Equal(t, "git-creds", credVol.Secret.SecretName)
}

func TestLanguageAgentController_Repository_AbsentWhenNoRepository(t *testing.T) {
	agent := gen.LanguageAgent("no-repo-agent", "default", gen.SetAgentWorkspace("5Gi"))
	r, fakeClient := newSeedReconciler(t, agent, gen.ReadyCluster("default"))
	reconcileTwice(t, r, agent.Name, agent.Namespace)

	podSpec, _ := agentPodView(t, fakeClient, agent.Name, agent.Namespace)

	assert.Nil(t, findInitContainer(podSpec, "repository"), "no repository init container without spec.repository")
	assert.Empty(t, podSpec.Containers[0].WorkingDir, "WorkingDir must be unset without a repository")
	for _, e := range podSpec.Containers[0].Env {
		assert.NotEqual(t, "AGENT_REPO_DIR", e.Name, "AGENT_REPO_DIR must not be injected without a repository")
	}
}

func TestLanguageAgentController_Repository_ConditionSet(t *testing.T) {
	agent := gen.LanguageAgent("repo-cond-agent", "default",
		gen.SetAgentWorkspace("5Gi"),
		gen.SetAgentRepository("https://github.com/foo/bar.git", "", "", ""),
	)
	agent.Generation = 1
	r, fakeClient := newSeedReconciler(t, agent, gen.ReadyCluster("default"))
	reconcileTwice(t, r, agent.Name, agent.Namespace)

	ctx := context.Background()
	updated := &langopv1alpha1.LanguageAgent{}
	require.NoError(t, fakeClient.Get(ctx, types.NamespacedName{Name: agent.Name, Namespace: agent.Namespace}, updated))

	var cond *metav1.Condition
	for i := range updated.Status.Conditions {
		if updated.Status.Conditions[i].Type == langopv1alpha1.ConditionRepositoryCloned {
			cond = &updated.Status.Conditions[i]
			break
		}
	}
	require.NotNil(t, cond, "RepositoryCloned condition must be set")
	assert.Equal(t, metav1.ConditionTrue, cond.Status)
	assert.Equal(t, langopv1alpha1.ReasonRepositoryConfigured, cond.Reason)
}

// parseAgentConfigMap reconciles the agent once and returns the parsed config.yaml from the agent ConfigMap.
func parseAgentConfigMap(t *testing.T, scheme *runtime.Scheme, objects ...client.Object) agentConfigYAML {
	t.Helper()

	// The agent must be the last object passed so we can extract its name/namespace.
	agentObj := objects[len(objects)-1]
	agent := agentObj.(*langopv1alpha1.LanguageAgent)

	fakeClient := fake.NewClientBuilder().
		WithScheme(scheme).
		WithObjects(objects...).
		WithStatusSubresource(agent).
		Build()

	reconciler := &LanguageAgentReconciler{
		Client:          fakeClient,
		Scheme:          scheme,
		Log:             logr.Discard(),
		Recorder:        &record.FakeRecorder{},
		RegistryManager: &mockRegistryManager{},
	}

	ctx := context.Background()
	_, err := reconciler.Reconcile(ctx, ctrl.Request{
		NamespacedName: types.NamespacedName{Name: agent.Name, Namespace: agent.Namespace},
	})
	require.NoError(t, err, "Reconcile failed")

	cm := &corev1.ConfigMap{}
	require.NoError(t, fakeClient.Get(ctx, types.NamespacedName{
		Name:      GenerateConfigMapName(agent.Name, "agent"),
		Namespace: agent.Namespace,
	}, cm), "agent ConfigMap not found")

	var cfg agentConfigYAML
	require.NoError(t, yaml.Unmarshal([]byte(cm.Data["config.yaml"]), &cfg), "failed to parse config.yaml")

	return cfg
}
