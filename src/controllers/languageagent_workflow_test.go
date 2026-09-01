package controllers

import (
	"context"
	"slices"
	"strings"
	"testing"
	"time"

	wfv1 "github.com/argoproj/argo-workflows/v4/pkg/apis/workflow/v1alpha1"
	"github.com/go-logr/logr"
	langopv1alpha1 "github.com/language-operator/language-operator/api/v1alpha1"
	"github.com/language-operator/language-operator/controllers/testutil"
	"github.com/language-operator/language-operator/internal/testutil/gen"
	"github.com/language-operator/language-operator/pkg/events"
	langoplabels "github.com/language-operator/language-operator/pkg/labels"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/tools/record"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

func TestLanguageAgentController_WorkflowTemplateCreation(t *testing.T) {
	scheme := testutil.SetupTestScheme(t)

	agent := &langopv1alpha1.LanguageAgent{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-workflow-agent",
			Namespace: "default",
		},
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
	_, err := reconciler.Reconcile(ctx, ctrl.Request{
		NamespacedName: types.NamespacedName{
			Name:      agent.Name,
			Namespace: agent.Namespace,
		},
	})
	if err != nil {
		t.Fatalf("Reconcile failed: %v", err)
	}

	// Verify Deployment was created
	podSpec, _ := agentPodView(t, fakeClient, agent.Name, agent.Namespace)

	// Verify Deployment has correct image
	if len(podSpec.Containers) != 1 {
		t.Errorf("Expected 1 container, got %d", len(podSpec.Containers))
	}
	if podSpec.Containers[0].Image != agent.Spec.Image {
		t.Errorf("Expected image '%s', got '%s'", agent.Spec.Image, podSpec.Containers[0].Image)
	}
}

func TestLanguageAgentController_PodSecurityContext(t *testing.T) {
	scheme := testutil.SetupTestScheme(t)

	agent := &langopv1alpha1.LanguageAgent{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-security-agent",
			Namespace: "default",
		},
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
	_, err := reconciler.Reconcile(ctx, ctrl.Request{
		NamespacedName: types.NamespacedName{
			Name:      agent.Name,
			Namespace: agent.Namespace,
		},
	})
	if err != nil {
		t.Fatalf("Reconcile failed: %v", err)
	}

	podSpec, _ := agentPodView(t, fakeClient, agent.Name, agent.Namespace)

	// Verify Pod security context
	podSec := podSpec.SecurityContext
	if podSec == nil {
		t.Fatal("Pod SecurityContext is nil")
	}

	if podSec.RunAsNonRoot == nil || !*podSec.RunAsNonRoot {
		t.Error("Expected RunAsNonRoot to be true")
	}

	if podSec.RunAsUser == nil || *podSec.RunAsUser != 1000 {
		t.Errorf("Expected RunAsUser to be 1000, got %v", podSec.RunAsUser)
	}

	if podSec.FSGroup == nil || *podSec.FSGroup != 101 {
		t.Errorf("Expected FSGroup to be 101, got %v", podSec.FSGroup)
	}

	if podSec.SeccompProfile == nil || podSec.SeccompProfile.Type != corev1.SeccompProfileTypeRuntimeDefault {
		t.Error("Expected SeccompProfile type to be RuntimeDefault")
	}
}

func TestLanguageAgentController_ContainerSecurityContext(t *testing.T) {
	scheme := testutil.SetupTestScheme(t)

	agent := &langopv1alpha1.LanguageAgent{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-container-security-agent",
			Namespace: "default",
		},
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
	_, err := reconciler.Reconcile(ctx, ctrl.Request{
		NamespacedName: types.NamespacedName{
			Name:      agent.Name,
			Namespace: agent.Namespace,
		},
	})
	if err != nil {
		t.Fatalf("Reconcile failed: %v", err)
	}

	podSpec, _ := agentPodView(t, fakeClient, agent.Name, agent.Namespace)

	// Verify container security context
	if len(podSpec.Containers) == 0 {
		t.Fatal("No containers found in workflow template")
	}

	containerSec := podSpec.Containers[0].SecurityContext
	if containerSec == nil {
		t.Fatal("Container SecurityContext is nil")
	}

	if containerSec.AllowPrivilegeEscalation == nil || *containerSec.AllowPrivilegeEscalation {
		t.Error("Expected AllowPrivilegeEscalation to be false")
	}

	if containerSec.RunAsNonRoot == nil || !*containerSec.RunAsNonRoot {
		t.Error("Expected RunAsNonRoot to be true")
	}

	if containerSec.RunAsUser == nil || *containerSec.RunAsUser != 1000 {
		t.Errorf("Expected RunAsUser to be 1000, got %v", containerSec.RunAsUser)
	}

	if containerSec.ReadOnlyRootFilesystem == nil || !*containerSec.ReadOnlyRootFilesystem {
		t.Error("Expected ReadOnlyRootFilesystem to be true")
	}

	if containerSec.Capabilities == nil {
		t.Fatal("Capabilities is nil")
	}

	if len(containerSec.Capabilities.Drop) != 1 || containerSec.Capabilities.Drop[0] != "ALL" {
		t.Errorf("Expected capabilities to drop ALL, got %v", containerSec.Capabilities.Drop)
	}
}

func TestLanguageAgentController_TmpfsVolumes(t *testing.T) {
	scheme := testutil.SetupTestScheme(t)

	agent := &langopv1alpha1.LanguageAgent{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-tmpfs-agent",
			Namespace: "default",
		},
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
	_, err := reconciler.Reconcile(ctx, ctrl.Request{
		NamespacedName: types.NamespacedName{
			Name:      agent.Name,
			Namespace: agent.Namespace,
		},
	})
	if err != nil {
		t.Fatalf("Reconcile failed: %v", err)
	}

	podSpec, _ := agentPodView(t, fakeClient, agent.Name, agent.Namespace)

	// Check for tmpfs volumes
	expectedVolumes := map[string]string{
		"tmp": "/tmp",
	}

	volumes := podSpec.Volumes
	volumeNames := make(map[string]bool)
	for _, vol := range volumes {
		volumeNames[vol.Name] = true
		// Verify it's an EmptyDir with Memory medium
		if vol.EmptyDir != nil && vol.EmptyDir.Medium == corev1.StorageMediumMemory {
			// Good - it's a tmpfs volume
		} else if _, ok := expectedVolumes[vol.Name]; ok {
			t.Errorf("Volume %s should be EmptyDir with Memory medium", vol.Name)
		}
	}

	// Check all expected volumes exist
	for volName := range expectedVolumes {
		if !volumeNames[volName] {
			t.Errorf("Expected volume %s to exist", volName)
		}
	}

	// Check volume mounts on container
	if len(podSpec.Containers) == 0 {
		t.Fatal("No containers found in workflow template")
	}

	volumeMounts := podSpec.Containers[0].VolumeMounts
	mountPaths := make(map[string]string)
	for _, mount := range volumeMounts {
		mountPaths[mount.Name] = mount.MountPath
	}

	// Verify all expected mounts
	for volName, expectedPath := range expectedVolumes {
		if actualPath, ok := mountPaths[volName]; ok {
			if actualPath != expectedPath {
				t.Errorf("Volume %s expected to be mounted at %s, got %s", volName, expectedPath, actualPath)
			}
		} else {
			t.Errorf("Expected volume mount for %s at %s", volName, expectedPath)
		}
	}
}

func TestLanguageAgentController_ResolveTools(t *testing.T) {
	scheme := testutil.SetupTestScheme(t)

	t.Run("service_mode_tool", func(t *testing.T) {
		tool := gen.LanguageTool("my-tool", "default",
			gen.SetToolDeploymentMode("service"),
		)
		agent := &langopv1alpha1.LanguageAgent{
			ObjectMeta: metav1.ObjectMeta{Name: "agent", Namespace: "default"},
			Spec: langopv1alpha1.LanguageAgentSpec{
				Tools: []langopv1alpha1.ToolReference{{Name: "my-tool"}},
			},
		}

		fakeClient := fake.NewClientBuilder().WithScheme(scheme).WithObjects(tool).Build()
		r := &LanguageAgentReconciler{Client: fakeClient,
			Scheme: scheme, Log: logr.Discard()}

		urls, err := r.resolveTools(context.Background(), agent)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(urls) != 1 {
			t.Fatalf("expected 1 URL, got %d", len(urls))
		}
		expected := "http://my-tool.default.svc.cluster.local:8080/mcp"
		if urls[0] != expected {
			t.Errorf("expected %q, got %q", expected, urls[0])
		}
	})

	t.Run("sidecar_mode_tool", func(t *testing.T) {
		tool := gen.LanguageTool("sidecar-tool", "default",
			gen.SetToolDeploymentMode("sidecar"),
			gen.SetToolPort(9090),
		)
		agent := &langopv1alpha1.LanguageAgent{
			ObjectMeta: metav1.ObjectMeta{Name: "agent", Namespace: "default"},
			Spec: langopv1alpha1.LanguageAgentSpec{
				Tools: []langopv1alpha1.ToolReference{{Name: "sidecar-tool"}},
			},
		}

		fakeClient := fake.NewClientBuilder().WithScheme(scheme).WithObjects(tool).Build()
		r := &LanguageAgentReconciler{Client: fakeClient,
			Scheme: scheme, Log: logr.Discard()}

		urls, err := r.resolveTools(context.Background(), agent)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(urls) != 1 {
			t.Fatalf("expected 1 URL, got %d", len(urls))
		}
		expected := "http://localhost:9090/mcp"
		if urls[0] != expected {
			t.Errorf("expected %q, got %q", expected, urls[0])
		}
	})

	t.Run("missing_tool", func(t *testing.T) {
		agent := &langopv1alpha1.LanguageAgent{
			ObjectMeta: metav1.ObjectMeta{Name: "agent", Namespace: "default"},
			Spec: langopv1alpha1.LanguageAgentSpec{
				Tools: []langopv1alpha1.ToolReference{{Name: "nonexistent"}},
			},
		}
		fakeClient := fake.NewClientBuilder().WithScheme(scheme).Build()
		r := &LanguageAgentReconciler{Client: fakeClient,
			Scheme: scheme, Log: logr.Discard()}

		_, err := r.resolveTools(context.Background(), agent)
		if err == nil {
			t.Error("expected error for missing tool")
		}
	})

	t.Run("custom_port", func(t *testing.T) {
		tool := gen.LanguageTool("port-tool", "default",
			gen.SetToolDeploymentMode("service"),
			gen.SetToolPort(9999),
		)
		agent := &langopv1alpha1.LanguageAgent{
			ObjectMeta: metav1.ObjectMeta{Name: "agent", Namespace: "default"},
			Spec: langopv1alpha1.LanguageAgentSpec{
				Tools: []langopv1alpha1.ToolReference{{Name: "port-tool"}},
			},
		}

		fakeClient := fake.NewClientBuilder().WithScheme(scheme).WithObjects(tool).Build()
		r := &LanguageAgentReconciler{Client: fakeClient,
			Scheme: scheme, Log: logr.Discard()}

		urls, err := r.resolveTools(context.Background(), agent)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		expected := "http://port-tool.default.svc.cluster.local:9999/mcp"
		if len(urls) != 1 || urls[0] != expected {
			t.Errorf("expected %q, got %v", expected, urls)
		}
	})

	t.Run("disabled_tool_skipped", func(t *testing.T) {
		tool := gen.LanguageTool("disabled-tool", "default",
			gen.SetToolDeploymentMode("service"),
		)
		disabled := false
		agent := &langopv1alpha1.LanguageAgent{
			ObjectMeta: metav1.ObjectMeta{Name: "agent", Namespace: "default"},
			Spec: langopv1alpha1.LanguageAgentSpec{
				Tools: []langopv1alpha1.ToolReference{{Name: "disabled-tool", Enabled: &disabled}},
			},
		}

		fakeClient := fake.NewClientBuilder().WithScheme(scheme).WithObjects(tool).Build()
		r := &LanguageAgentReconciler{Client: fakeClient,
			Scheme: scheme, Log: logr.Discard()}

		urls, err := r.resolveTools(context.Background(), agent)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(urls) != 0 {
			t.Errorf("expected 0 URLs for disabled tool, got %d: %v", len(urls), urls)
		}
	})

	t.Run("explicitly_enabled_tool_included", func(t *testing.T) {
		tool := gen.LanguageTool("enabled-tool", "default",
			gen.SetToolDeploymentMode("service"),
		)
		enabled := true
		agent := &langopv1alpha1.LanguageAgent{
			ObjectMeta: metav1.ObjectMeta{Name: "agent", Namespace: "default"},
			Spec: langopv1alpha1.LanguageAgentSpec{
				Tools: []langopv1alpha1.ToolReference{{Name: "enabled-tool", Enabled: &enabled}},
			},
		}

		fakeClient := fake.NewClientBuilder().WithScheme(scheme).WithObjects(tool).Build()
		r := &LanguageAgentReconciler{Client: fakeClient,
			Scheme: scheme, Log: logr.Discard()}

		urls, err := r.resolveTools(context.Background(), agent)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(urls) != 1 {
			t.Errorf("expected 1 URL for explicitly enabled tool, got %d", len(urls))
		}
	})
}

func TestLanguageAgentController_ResolveModels(t *testing.T) {
	scheme := testutil.SetupTestScheme(t)

	t.Run("single_model", func(t *testing.T) {
		model := gen.LanguageModel("my-model", "default",
			gen.SetModelName("claude-3-sonnet"),
		)
		agent := &langopv1alpha1.LanguageAgent{
			ObjectMeta: metav1.ObjectMeta{Name: "agent", Namespace: "default"},
			Spec: langopv1alpha1.LanguageAgentSpec{
				Models: []langopv1alpha1.ModelReference{{Name: "my-model"}},
			},
		}

		fakeClient := fake.NewClientBuilder().WithScheme(scheme).WithObjects(model).Build()
		r := &LanguageAgentReconciler{Client: fakeClient,
			Scheme: scheme, Log: logr.Discard()}

		urls, names, err := r.resolveModels(context.Background(), agent)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		// All models in a namespace share the single cluster gateway
		expectedURL := "http://gateway.default.svc.cluster.local:8000"
		if len(urls) != 1 || urls[0] != expectedURL {
			t.Errorf("unexpected URLs: %v", urls)
		}
		if len(names) != 1 || names[0] != "claude-3-sonnet" {
			t.Errorf("unexpected names: %v", names)
		}
	})

	t.Run("multiple_models", func(t *testing.T) {
		m1 := gen.LanguageModel("model-a", "default", gen.SetModelName("gpt-4"))
		m2 := gen.LanguageModel("model-b", "default", gen.SetModelName("claude-3"))
		agent := &langopv1alpha1.LanguageAgent{
			ObjectMeta: metav1.ObjectMeta{Name: "agent", Namespace: "default"},
			Spec: langopv1alpha1.LanguageAgentSpec{
				Models: []langopv1alpha1.ModelReference{
					{Name: "model-a"},
					{Name: "model-b"},
				},
			},
		}

		fakeClient := fake.NewClientBuilder().WithScheme(scheme).WithObjects(m1, m2).Build()
		r := &LanguageAgentReconciler{Client: fakeClient,
			Scheme: scheme, Log: logr.Discard()}

		urls, names, err := r.resolveModels(context.Background(), agent)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		// Multiple models in the same namespace → single shared gateway URL, two model names
		if len(urls) != 1 {
			t.Fatalf("expected 1 gateway URL (deduplicated), got %d: %v", len(urls), urls)
		}
		if len(names) != 2 {
			t.Fatalf("expected 2 names, got %d: %v", len(names), names)
		}
	})

	t.Run("missing_model", func(t *testing.T) {
		agent := &langopv1alpha1.LanguageAgent{
			ObjectMeta: metav1.ObjectMeta{Name: "agent", Namespace: "default"},
			Spec: langopv1alpha1.LanguageAgentSpec{
				Models: []langopv1alpha1.ModelReference{{Name: "nonexistent"}},
			},
		}
		fakeClient := fake.NewClientBuilder().WithScheme(scheme).Build()
		r := &LanguageAgentReconciler{Client: fakeClient,
			Scheme: scheme, Log: logr.Discard()}

		_, _, err := r.resolveModels(context.Background(), agent)
		if err == nil {
			t.Error("expected error for missing model")
		}
	})
}

func TestLanguageAgentController_ResolveSidecarTools(t *testing.T) {
	scheme := testutil.SetupTestScheme(t)

	t.Run("sidecar_tool_added", func(t *testing.T) {
		tool := gen.LanguageTool("my-sidecar", "default",
			gen.SetToolDeploymentMode("sidecar"),
			gen.SetToolImage("my-tool:v1"),
		)
		agent := &langopv1alpha1.LanguageAgent{
			ObjectMeta: metav1.ObjectMeta{Name: "agent", Namespace: "default"},
			Spec: langopv1alpha1.LanguageAgentSpec{
				Tools: []langopv1alpha1.ToolReference{{Name: "my-sidecar"}},
			},
		}

		fakeClient := fake.NewClientBuilder().WithScheme(scheme).WithObjects(tool).Build()
		r := &LanguageAgentReconciler{Client: fakeClient,
			Scheme: scheme, Log: logr.Discard()}

		containers, _, err := r.resolveSidecarTools(context.Background(), agent)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(containers) != 1 {
			t.Fatalf("expected 1 container, got %d", len(containers))
		}
		if containers[0].Image != "my-tool:v1" {
			t.Errorf("expected image 'my-tool:v1', got %q", containers[0].Image)
		}
	})

	t.Run("stdio_sidecar_injects_bridge", func(t *testing.T) {
		tool := gen.LanguageTool("my-stdio-sidecar", "default",
			gen.SetToolDeploymentMode("sidecar"),
			gen.SetToolPort(8080),
			gen.SetToolTransport("stdio"),
			gen.SetToolStdioCommand("npx", "-y", "@upstash/context7-mcp"),
		)
		agent := &langopv1alpha1.LanguageAgent{
			ObjectMeta: metav1.ObjectMeta{Name: "agent", Namespace: "default"},
			Spec: langopv1alpha1.LanguageAgentSpec{
				Tools: []langopv1alpha1.ToolReference{{Name: "my-stdio-sidecar"}},
			},
		}

		fakeClient := fake.NewClientBuilder().WithScheme(scheme).WithObjects(tool).Build()
		r := &LanguageAgentReconciler{Client: fakeClient, Scheme: scheme, Log: logr.Discard()}

		containers, volumes, err := r.resolveSidecarTools(context.Background(), agent)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(containers) != 1 {
			t.Fatalf("expected 1 container, got %d", len(containers))
		}
		c := containers[0]
		if c.Image != langopv1alpha1.DefaultMCPBridgeImage {
			t.Errorf("sidecar image = %q, want bridge %q", c.Image, langopv1alpha1.DefaultMCPBridgeImage)
		}
		if len(c.Command) != 1 || c.Command[0] != "supergateway" {
			t.Errorf("command = %v, want [supergateway]", c.Command)
		}
		if c.ReadinessProbe == nil || c.ReadinessProbe.HTTPGet == nil || c.ReadinessProbe.HTTPGet.Path != "/health" {
			t.Errorf("expected /health httpGet readiness probe, got %+v", c.ReadinessProbe)
		}
		// Per-tool (pod-unique) scratch volume names, returned for the agent pod.
		wantCache := "tool-my-stdio-sidecar-cache"
		var haveCache bool
		for _, v := range volumes {
			if v.Name == wantCache {
				haveCache = true
			}
		}
		if !haveCache {
			t.Errorf("expected sidecar volume %q in %v", wantCache, volumes)
		}
	})

	t.Run("service_mode_skipped", func(t *testing.T) {
		tool := gen.LanguageTool("svc-tool", "default",
			gen.SetToolDeploymentMode("service"),
		)
		agent := &langopv1alpha1.LanguageAgent{
			ObjectMeta: metav1.ObjectMeta{Name: "agent", Namespace: "default"},
			Spec: langopv1alpha1.LanguageAgentSpec{
				Tools: []langopv1alpha1.ToolReference{{Name: "svc-tool"}},
			},
		}

		fakeClient := fake.NewClientBuilder().WithScheme(scheme).WithObjects(tool).Build()
		r := &LanguageAgentReconciler{Client: fakeClient,
			Scheme: scheme, Log: logr.Discard()}

		containers, _, err := r.resolveSidecarTools(context.Background(), agent)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(containers) != 0 {
			t.Errorf("expected 0 containers for service-mode tool, got %d", len(containers))
		}
	})

	t.Run("default_resources_applied", func(t *testing.T) {
		tool := gen.LanguageTool("def-sidecar", "default",
			gen.SetToolDeploymentMode("sidecar"),
		)
		agent := &langopv1alpha1.LanguageAgent{
			ObjectMeta: metav1.ObjectMeta{Name: "agent", Namespace: "default"},
			Spec: langopv1alpha1.LanguageAgentSpec{
				Tools: []langopv1alpha1.ToolReference{{Name: "def-sidecar"}},
			},
		}

		fakeClient := fake.NewClientBuilder().WithScheme(scheme).WithObjects(tool).Build()
		r := &LanguageAgentReconciler{Client: fakeClient,
			Scheme: scheme, Log: logr.Discard()}

		containers, _, err := r.resolveSidecarTools(context.Background(), agent)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(containers) != 1 {
			t.Fatalf("expected 1 container, got %d", len(containers))
		}
		if containers[0].Resources.Requests.Cpu().IsZero() {
			t.Error("expected default CPU request to be set on sidecar container")
		}
	})

	t.Run("disabled_sidecar_tool_skipped", func(t *testing.T) {
		tool := gen.LanguageTool("disabled-sidecar", "default",
			gen.SetToolDeploymentMode("sidecar"),
			gen.SetToolImage("my-tool:v1"),
		)
		disabled := false
		agent := &langopv1alpha1.LanguageAgent{
			ObjectMeta: metav1.ObjectMeta{Name: "agent", Namespace: "default"},
			Spec: langopv1alpha1.LanguageAgentSpec{
				Tools: []langopv1alpha1.ToolReference{{Name: "disabled-sidecar", Enabled: &disabled}},
			},
		}

		fakeClient := fake.NewClientBuilder().WithScheme(scheme).WithObjects(tool).Build()
		r := &LanguageAgentReconciler{Client: fakeClient,
			Scheme: scheme, Log: logr.Discard()}

		containers, _, err := r.resolveSidecarTools(context.Background(), agent)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(containers) != 0 {
			t.Errorf("expected 0 containers for disabled sidecar tool, got %d", len(containers))
		}
	})
}

func TestLanguageAgentController_SidecarToolInjectedIntoWorkflow(t *testing.T) {
	scheme := testutil.SetupTestScheme(t)

	tool := gen.LanguageTool("my-sidecar", "default",
		gen.SetToolDeploymentMode("sidecar"),
		gen.SetToolImage("ghcr.io/language-operator/tool:latest"),
		gen.SetToolPort(8080),
	)
	agent := gen.LanguageAgent("sidecar-agent", "default",
		gen.SetAgentTool("my-sidecar", nil),
	)

	fakeClient := fake.NewClientBuilder().
		WithScheme(scheme).
		WithObjects(gen.ReadyCluster("default"), tool, agent).
		WithStatusSubresource(agent).
		Build()

	r := &LanguageAgentReconciler{
		Client:          fakeClient,
		Scheme:          scheme,
		Log:             logr.Discard(),
		Recorder:        record.NewFakeRecorder(10),
		EventManager:    events.NewEventManager(record.NewFakeRecorder(10)),
		RegistryManager: &mockRegistryManager{},
	}

	ctx := context.Background()
	req := ctrl.Request{NamespacedName: types.NamespacedName{Name: agent.Name, Namespace: agent.Namespace}}

	// First reconcile: adds finalizer.
	_, err := r.Reconcile(ctx, req)
	require.NoError(t, err)

	// Second reconcile: creates resources.
	_, err = r.Reconcile(ctx, req)
	require.NoError(t, err)

	tmpl := agentWorkflowTemplate(t, fakeClient, agent.Name, agent.Namespace)

	// Tool sidecars become Argo sidecars: they run alongside the agent for its
	// whole life and Argo tears them down when the agent container exits.
	sidecars := tmpl.Spec.Templates[0].Sidecars
	require.Len(t, sidecars, 1, "expected exactly one sidecar from sidecar tool")
	assert.Equal(t, "tool-my-sidecar", sidecars[0].Name)
	assert.Equal(t, "ghcr.io/language-operator/tool:latest", sidecars[0].Image)
	assert.Contains(t, tmpl.Spec.PodSpecPatch, "shareProcessNamespace")
}

func TestLanguageAgentController_AgentConfigVolume(t *testing.T) {
	scheme := testutil.SetupTestScheme(t)

	agent := &langopv1alpha1.LanguageAgent{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "config-vol-agent",
			Namespace: "default",
		},
		Spec: langopv1alpha1.LanguageAgentSpec{
			Image:        "ghcr.io/language-operator/agent:latest",
			Instructions: "test instructions",
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

	expectedConfigMapName := GenerateConfigMapName(agent.Name, "agent")

	var foundVolume bool
	for _, vol := range podSpec.Volumes {
		if vol.Name == "agent-config" {
			foundVolume = true
			if vol.ConfigMap == nil {
				t.Fatal("agent-config volume has no ConfigMap source")
			}
			if vol.ConfigMap.Name != expectedConfigMapName {
				t.Errorf("agent-config volume points to %q, want %q", vol.ConfigMap.Name, expectedConfigMapName)
			}
			break
		}
	}
	if !foundVolume {
		t.Error("expected agent-config volume in workflow template, not found")
	}

	containers := podSpec.Containers
	if len(containers) == 0 {
		t.Fatal("no containers in workflow template")
	}
	var foundMount bool
	for _, vm := range containers[0].VolumeMounts {
		if vm.Name == "agent-config" {
			foundMount = true
			if vm.MountPath != "/etc/agent" {
				t.Errorf("agent-config mount path is %q, want /etc/agent", vm.MountPath)
			}
			if !vm.ReadOnly {
				t.Error("agent-config volume mount should be read-only")
			}
			break
		}
	}
	if !foundMount {
		t.Error("expected agent-config volume mount in agent container, not found")
	}
}

func TestLanguageAgentController_AgentConfigMapKeys(t *testing.T) {
	scheme := testutil.SetupTestScheme(t)

	agent := &langopv1alpha1.LanguageAgent{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "configmap-keys-agent",
			Namespace: "default",
		},
		Spec: langopv1alpha1.LanguageAgentSpec{
			Image:        "ghcr.io/language-operator/agent:latest",
			Instructions: "You are a helpful assistant.",
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

	cm := &corev1.ConfigMap{}
	if err := fakeClient.Get(ctx, types.NamespacedName{
		Name:      GenerateConfigMapName(agent.Name, "agent"),
		Namespace: agent.Namespace,
	}, cm); err != nil {
		t.Fatalf("Expected agent ConfigMap to exist: %v", err)
	}

	if _, ok := cm.Data["config.yaml"]; !ok {
		t.Error("ConfigMap missing config.yaml key")
	}
	if _, ok := cm.Data["instructions.txt"]; ok {
		t.Error("ConfigMap must not contain instructions.txt key")
	}

	configYAML := cm.Data["config.yaml"]
	if !strings.Contains(configYAML, agent.Name) {
		t.Errorf("config.yaml does not contain agent name %q: %s", agent.Name, configYAML)
	}
	if !strings.Contains(configYAML, agent.Namespace) {
		t.Errorf("config.yaml does not contain agent namespace %q: %s", agent.Namespace, configYAML)
	}
	if !strings.Contains(configYAML, agent.Spec.Instructions) {
		t.Errorf("config.yaml does not contain instructions %q: %s", agent.Spec.Instructions, configYAML)
	}
}

func TestLanguageAgentController_SchedulingFields(t *testing.T) {
	scheme := testutil.SetupTestScheme(t)

	agent := &langopv1alpha1.LanguageAgent{
		ObjectMeta: metav1.ObjectMeta{Name: "test-sched-agent", Namespace: "default"},
		Spec: langopv1alpha1.LanguageAgentSpec{
			Image: "ghcr.io/language-operator/agent:latest",
			Deployment: langopv1alpha1.DeploymentSpec{
				NodeSelector: map[string]string{"kubernetes.io/arch": "amd64"},
				Tolerations: []corev1.Toleration{
					{Key: "gpu", Operator: corev1.TolerationOpExists, Effect: corev1.TaintEffectNoSchedule},
				},
				TopologySpreadConstraints: []corev1.TopologySpreadConstraint{
					{MaxSkew: 1, TopologyKey: "topology.kubernetes.io/zone", WhenUnsatisfiable: corev1.DoNotSchedule},
				},
				Affinity: &corev1.Affinity{
					NodeAffinity: &corev1.NodeAffinity{
						RequiredDuringSchedulingIgnoredDuringExecution: &corev1.NodeSelector{
							NodeSelectorTerms: []corev1.NodeSelectorTerm{
								{MatchExpressions: []corev1.NodeSelectorRequirement{
									{Key: "node-type", Operator: corev1.NodeSelectorOpIn, Values: []string{"gpu"}},
								}},
							},
						},
					},
				},
				ImagePullSecrets: []corev1.LocalObjectReference{{Name: "my-registry-secret"}},
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
		t.Fatalf("First reconcile failed: %v", err)
	}
	_, err = reconciler.Reconcile(ctx, ctrl.Request{
		NamespacedName: types.NamespacedName{Name: agent.Name, Namespace: agent.Namespace},
	})
	if err != nil {
		t.Fatalf("Second reconcile failed: %v", err)
	}

	podSpec, _ := agentPodView(t, fakeClient, agent.Name, agent.Namespace)

	if podSpec.NodeSelector["kubernetes.io/arch"] != "amd64" {
		t.Errorf("Expected NodeSelector kubernetes.io/arch=amd64, got %v", podSpec.NodeSelector)
	}
	if len(podSpec.Tolerations) == 0 || podSpec.Tolerations[0].Key != "gpu" {
		t.Errorf("Expected Toleration gpu, got %v", podSpec.Tolerations)
	}
	// WorkflowSpec has no topologySpreadConstraints field, so the operator carries
	// them over as a strategic-merge patch on the pod Argo generates.
	tmpl := agentWorkflowTemplate(t, fakeClient, agent.Name, agent.Namespace)
	assert.Contains(t, tmpl.Spec.PodSpecPatch, "topology.kubernetes.io/zone",
		"expected topology spread constraints in podSpecPatch, got %q", tmpl.Spec.PodSpecPatch)
	if podSpec.Affinity == nil || podSpec.Affinity.NodeAffinity == nil {
		t.Error("Expected Affinity to be set")
	}
	if len(podSpec.ImagePullSecrets) == 0 || podSpec.ImagePullSecrets[0].Name != "my-registry-secret" {
		t.Errorf("Expected ImagePullSecret my-registry-secret, got %v", podSpec.ImagePullSecrets)
	}
}

func TestLanguageAgentController_PodLabelsAndAnnotations(t *testing.T) {
	scheme := testutil.SetupTestScheme(t)

	agent := &langopv1alpha1.LanguageAgent{
		ObjectMeta: metav1.ObjectMeta{Name: "test-labels-agent", Namespace: "default"},
		Spec: langopv1alpha1.LanguageAgentSpec{
			Image: "ghcr.io/language-operator/agent:latest",
			Deployment: langopv1alpha1.DeploymentSpec{
				// "app.kubernetes.io/name" is also set by the operator; operator must take precedence.
				PodLabels: map[string]string{
					"cost-center":            "team-a",
					"app.kubernetes.io/name": "user-override-should-be-ignored",
				},
				PodAnnotations: map[string]string{
					"prometheus.io/scrape": "true",
					"prometheus.io/port":   "8080",
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
		t.Fatalf("First reconcile failed: %v", err)
	}
	_, err = reconciler.Reconcile(ctx, ctrl.Request{
		NamespacedName: types.NamespacedName{Name: agent.Name, Namespace: agent.Namespace},
	})
	if err != nil {
		t.Fatalf("Second reconcile failed: %v", err)
	}

	_, podMeta := agentPodView(t, fakeClient, agent.Name, agent.Namespace)

	// User label should be present
	if podMeta.Labels["cost-center"] != "team-a" {
		t.Errorf("Expected pod label cost-center=team-a, got %q", podMeta.Labels["cost-center"])
	}
	// Operator label must not be overridden by user; GetCommonLabels sets it to the agent name.
	if podMeta.Labels["app.kubernetes.io/name"] != agent.Name {
		t.Errorf("Operator label app.kubernetes.io/name should be %q, got %q", agent.Name, podMeta.Labels["app.kubernetes.io/name"])
	}
	// The Service selector must stay operator-labels-only, or a user pod label
	// could silently detach the Service from the agent's pods.
	svc := &corev1.Service{}
	require.NoError(t, fakeClient.Get(ctx, types.NamespacedName{Name: agent.Name, Namespace: agent.Namespace}, svc))
	if _, hasUserLabel := svc.Spec.Selector["cost-center"]; hasUserLabel {
		t.Error("User pod label should not appear in the Service selector")
	}
	// PodAnnotations
	if podMeta.Annotations["prometheus.io/scrape"] != "true" {
		t.Errorf("Expected pod annotation prometheus.io/scrape=true, got %q", podMeta.Annotations["prometheus.io/scrape"])
	}
	if podMeta.Annotations["prometheus.io/port"] != "8080" {
		t.Errorf("Expected pod annotation prometheus.io/port=8080, got %q", podMeta.Annotations["prometheus.io/port"])
	}
}

func TestLanguageAgentController_ConfigHashAnnotation(t *testing.T) {
	scheme := testutil.SetupTestScheme(t)

	agent := &langopv1alpha1.LanguageAgent{
		ObjectMeta: metav1.ObjectMeta{Name: "test-hash-agent", Namespace: "default"},
		Spec: langopv1alpha1.LanguageAgentSpec{
			Image: "ghcr.io/language-operator/agent:latest",
			Deployment: langopv1alpha1.DeploymentSpec{
				PodAnnotations: map[string]string{
					"prometheus.io/scrape": "true",
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
	for i := range 2 {
		_, err := reconciler.Reconcile(ctx, ctrl.Request{
			NamespacedName: types.NamespacedName{Name: agent.Name, Namespace: agent.Namespace},
		})
		if err != nil {
			t.Fatalf("Reconcile %d failed: %v", i+1, err)
		}
	}

	_, podMeta := agentPodView(t, fakeClient, agent.Name, agent.Namespace)

	annotations := podMeta.Annotations
	hash, ok := annotations[langoplabels.LabelKeyLangopConfigHash]
	if !ok || hash == "" {
		t.Errorf("Expected pod annotation %q to be set, got annotations: %v", langoplabels.LabelKeyLangopConfigHash, annotations)
	}
	// User annotations must still be present alongside the operator-managed hash.
	if annotations["prometheus.io/scrape"] != "true" {
		t.Errorf("Expected user annotation prometheus.io/scrape=true to be preserved, got %q", annotations["prometheus.io/scrape"])
	}
}

func TestLanguageAgentController_UserVolumesAndMounts(t *testing.T) {
	scheme := testutil.SetupTestScheme(t)

	agent := &langopv1alpha1.LanguageAgent{
		ObjectMeta: metav1.ObjectMeta{Name: "test-vols-agent", Namespace: "default"},
		Spec: langopv1alpha1.LanguageAgentSpec{
			Image: "ghcr.io/language-operator/agent:latest",
			Deployment: langopv1alpha1.DeploymentSpec{
				Volumes: []corev1.Volume{
					{
						Name: "user-secret",
						VolumeSource: corev1.VolumeSource{
							Secret: &corev1.SecretVolumeSource{SecretName: "my-secret"},
						},
					},
				},
				VolumeMounts: []corev1.VolumeMount{
					{Name: "user-secret", MountPath: "/etc/user-secret", ReadOnly: true},
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
		t.Fatalf("First reconcile failed: %v", err)
	}
	_, err = reconciler.Reconcile(ctx, ctrl.Request{
		NamespacedName: types.NamespacedName{Name: agent.Name, Namespace: agent.Namespace},
	})
	if err != nil {
		t.Fatalf("Second reconcile failed: %v", err)
	}

	podSpec, _ := agentPodView(t, fakeClient, agent.Name, agent.Namespace)

	// Operator-managed volumes (tmp, agent-config) must still be present
	volumeNames := map[string]bool{}
	for _, v := range podSpec.Volumes {
		volumeNames[v.Name] = true
	}
	if !volumeNames["tmp"] {
		t.Error("Expected operator-managed tmp volume to be present")
	}
	if !volumeNames["agent-config"] {
		t.Error("Expected operator-managed agent-config volume to be present")
	}
	if !volumeNames["user-secret"] {
		t.Error("Expected user-supplied user-secret volume to be appended")
	}

	// User mount must also be present
	mountPaths := map[string]string{}
	for _, m := range podSpec.Containers[0].VolumeMounts {
		mountPaths[m.Name] = m.MountPath
	}
	if mountPaths["user-secret"] != "/etc/user-secret" {
		t.Errorf("Expected user-secret mounted at /etc/user-secret, got %q", mountPaths["user-secret"])
	}
	// Operator mount (tmp) must still be present
	if mountPaths["tmp"] != "/tmp" {
		t.Errorf("Expected operator tmp mount at /tmp, got %q", mountPaths["tmp"])
	}
}

func TestLanguageAgentController_StartupProbe(t *testing.T) {
	scheme := testutil.SetupTestScheme(t)

	agent := &langopv1alpha1.LanguageAgent{
		ObjectMeta: metav1.ObjectMeta{Name: "test-startup-agent", Namespace: "default"},
		Spec: langopv1alpha1.LanguageAgentSpec{
			Image: "ghcr.io/language-operator/agent:latest",
			Deployment: langopv1alpha1.DeploymentSpec{
				StartupProbe: &corev1.Probe{
					ProbeHandler: corev1.ProbeHandler{
						HTTPGet: &corev1.HTTPGetAction{
							Path: "/healthz",
						},
					},
					FailureThreshold:    30,
					PeriodSeconds:       10,
					InitialDelaySeconds: 5,
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
		t.Fatalf("First reconcile failed: %v", err)
	}
	_, err = reconciler.Reconcile(ctx, ctrl.Request{
		NamespacedName: types.NamespacedName{Name: agent.Name, Namespace: agent.Namespace},
	})
	if err != nil {
		t.Fatalf("Second reconcile failed: %v", err)
	}

	podSpec, _ := agentPodView(t, fakeClient, agent.Name, agent.Namespace)

	container := podSpec.Containers[0]
	if container.StartupProbe == nil {
		t.Fatal("Expected StartupProbe to be set")
	}
	if container.StartupProbe.FailureThreshold != 30 {
		t.Errorf("Expected StartupProbe.FailureThreshold=30, got %d", container.StartupProbe.FailureThreshold)
	}
	if container.StartupProbe.PeriodSeconds != 10 {
		t.Errorf("Expected StartupProbe.PeriodSeconds=10, got %d", container.StartupProbe.PeriodSeconds)
	}
}

func TestLanguageAgentController_UserPodSecurityContext(t *testing.T) {
	scheme := testutil.SetupTestScheme(t)

	customUID := int64(2000)
	customGID := int64(3000)
	agent := &langopv1alpha1.LanguageAgent{
		ObjectMeta: metav1.ObjectMeta{Name: "test-custom-sec-agent", Namespace: "default"},
		Spec: langopv1alpha1.LanguageAgentSpec{
			Image: "ghcr.io/language-operator/agent:latest",
			Deployment: langopv1alpha1.DeploymentSpec{
				SecurityContext: &corev1.PodSecurityContext{
					RunAsUser:  &customUID,
					RunAsGroup: &customGID,
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
		t.Fatalf("First reconcile failed: %v", err)
	}
	_, err = reconciler.Reconcile(ctx, ctrl.Request{
		NamespacedName: types.NamespacedName{Name: agent.Name, Namespace: agent.Namespace},
	})
	if err != nil {
		t.Fatalf("Second reconcile failed: %v", err)
	}

	podSpec, _ := agentPodView(t, fakeClient, agent.Name, agent.Namespace)

	podSec := podSpec.SecurityContext
	if podSec == nil {
		t.Fatal("Pod SecurityContext is nil")
	}
	if podSec.RunAsUser == nil || *podSec.RunAsUser != customUID {
		t.Errorf("Expected RunAsUser=%d (user-supplied), got %v", customUID, podSec.RunAsUser)
	}
	if podSec.RunAsGroup == nil || *podSec.RunAsGroup != customGID {
		t.Errorf("Expected RunAsGroup=%d (user-supplied), got %v", customGID, podSec.RunAsGroup)
	}
	// Container-level security context should still be the operator-managed one
	containerSec := podSpec.Containers[0].SecurityContext
	if containerSec == nil {
		t.Fatal("Container SecurityContext should still be set by operator")
	}
	if containerSec.AllowPrivilegeEscalation == nil || *containerSec.AllowPrivilegeEscalation {
		t.Error("Container SecurityContext.AllowPrivilegeEscalation should be false")
	}
}

// agentWorkflowTemplate fetches the WorkflowTemplate the operator rendered for an
// agent, failing the test if it is absent or malformed.
func agentWorkflowTemplate(t *testing.T, c client.Client, name, namespace string) *wfv1.WorkflowTemplate {
	t.Helper()
	tmpl := &wfv1.WorkflowTemplate{}
	require.NoError(t, c.Get(context.Background(), types.NamespacedName{Name: name, Namespace: namespace}, tmpl),
		"expected a WorkflowTemplate for agent %s/%s", namespace, name)
	require.Len(t, tmpl.Spec.Templates, 1, "expected exactly one Argo template")
	return tmpl
}

// agentPodView reassembles an agent's WorkflowTemplate into the pod spec and pod
// metadata Argo will generate from it, so tests can assert on pod shape without
// caring how Argo splits it across WorkflowSpec and Template.
//
// It deliberately does not fold in PodSpecPatch — assert on that field directly
// for the handful of settings (shareProcessNamespace, topology spread) carried there.
func agentPodView(t *testing.T, c client.Client, name, namespace string) (corev1.PodSpec, metav1.ObjectMeta) {
	t.Helper()
	tmpl := agentWorkflowTemplate(t, c, name, namespace)
	spec := tmpl.Spec
	argoTmpl := spec.Templates[0]

	pod := corev1.PodSpec{
		ServiceAccountName: spec.ServiceAccountName,
		Volumes:            spec.Volumes,
		SecurityContext:    spec.SecurityContext,
		ImagePullSecrets:   spec.ImagePullSecrets,
		NodeSelector:       spec.NodeSelector,
		Tolerations:        spec.Tolerations,
		Affinity:           spec.Affinity,
	}
	if argoTmpl.Container != nil {
		pod.Containers = []corev1.Container{*argoTmpl.Container}
	}
	for _, ic := range argoTmpl.InitContainers {
		pod.InitContainers = append(pod.InitContainers, ic.Container)
	}
	for _, sc := range argoTmpl.Sidecars {
		pod.Containers = append(pod.Containers, sc.Container)
	}

	meta := metav1.ObjectMeta{}
	if spec.PodMetadata != nil {
		meta.Labels = spec.PodMetadata.Labels
		meta.Annotations = spec.PodMetadata.Annotations
	}
	return pod, meta
}

// newSeedReconciler creates a reconciler and fake client for workspace seeding tests.
func newSeedReconciler(t *testing.T, objs ...client.Object) (*LanguageAgentReconciler, client.Client) {
	t.Helper()
	scheme := testutil.SetupTestScheme(t)
	fakeClient := fake.NewClientBuilder().
		WithScheme(scheme).
		WithObjects(objs...).
		WithStatusSubresource(objs[0]).
		Build()
	return &LanguageAgentReconciler{
		Client:          fakeClient,
		Scheme:          scheme,
		Log:             logr.Discard(),
		Recorder:        &record.FakeRecorder{},
		RegistryManager: &mockRegistryManager{},
	}, fakeClient
}

// reconcileTwice runs two reconcile passes (first adds finalizer, second creates resources).
func reconcileTwice(t *testing.T, r *LanguageAgentReconciler, name, ns string) {
	t.Helper()
	req := ctrl.Request{NamespacedName: types.NamespacedName{Name: name, Namespace: ns}}
	_, err := r.Reconcile(context.Background(), req)
	require.NoError(t, err)
	_, err = r.Reconcile(context.Background(), req)
	require.NoError(t, err)
}

// --- execution modes ---

// newModeReconciler builds a reconciler for an agent whose execution mode is under test.
func newModeReconciler(t *testing.T, agent *langopv1alpha1.LanguageAgent) (*LanguageAgentReconciler, client.Client) {
	t.Helper()
	scheme := testutil.SetupTestScheme(t)
	fakeClient := fake.NewClientBuilder().
		WithScheme(scheme).
		WithObjects(gen.ReadyCluster("default"), agent).
		WithStatusSubresource(agent).
		Build()
	return &LanguageAgentReconciler{
		Client:          fakeClient,
		Scheme:          scheme,
		Log:             logr.Discard(),
		Recorder:        &record.FakeRecorder{},
		RegistryManager: &mockRegistryManager{},
	}, fakeClient
}

func TestLanguageAgentController_ServiceModeCreatesWorkflow(t *testing.T) {
	agent := gen.LanguageAgent("svc-mode-agent", "default")
	r, fc := newModeReconciler(t, agent)
	reconcileTwice(t, r, agent.Name, agent.Namespace)
	ctx := context.Background()
	key := types.NamespacedName{Name: agent.Name, Namespace: agent.Namespace}

	require.NoError(t, fc.Get(ctx, key, &wfv1.WorkflowTemplate{}))

	wf := &wfv1.Workflow{}
	require.NoError(t, fc.Get(ctx, key, wf), "service mode must create a long-lived Workflow")
	require.NotNil(t, wf.Spec.WorkflowTemplateRef)
	assert.Equal(t, agent.Name, wf.Spec.WorkflowTemplateRef.Name)

	// It must restart forever, the way a Deployment would have restarted the pod,
	// and never be garbage-collected while the agent exists.
	require.NotNil(t, wf.Spec.RetryStrategy)
	assert.Equal(t, wfv1.RetryPolicyAlways, wf.Spec.RetryStrategy.RetryPolicy)
	require.NotNil(t, wf.Spec.RetryStrategy.Limit)
	assert.Equal(t, serviceRetryLimit, wf.Spec.RetryStrategy.Limit.IntValue())
	assert.Nil(t, wf.Spec.TTLStrategy, "a service agent's Workflow must not expire")
	require.NotNil(t, wf.Spec.PodGC)
	assert.Equal(t, wfv1.PodGCOnPodNone, wf.Spec.PodGC.Strategy)

	assert.True(t, errors.IsNotFound(fc.Get(ctx, key, &wfv1.CronWorkflow{})),
		"service mode must not create a CronWorkflow")

	// A service agent is addressable.
	require.NoError(t, fc.Get(ctx, key, &corev1.Service{}))
}

func TestLanguageAgentController_TaskModeCreatesCronWorkflow(t *testing.T) {
	agent := gen.LanguageAgent("task-mode-agent", "default")
	agent.Spec.Execution = langopv1alpha1.ExecutionSpec{
		Mode:              langopv1alpha1.ExecutionModeTask,
		Schedule:          "*/5 * * * *",
		Timezone:          "America/New_York",
		ConcurrencyPolicy: "Replace",
	}
	r, fc := newModeReconciler(t, agent)
	reconcileTwice(t, r, agent.Name, agent.Namespace)
	ctx := context.Background()
	key := types.NamespacedName{Name: agent.Name, Namespace: agent.Namespace}

	require.NoError(t, fc.Get(ctx, key, &wfv1.WorkflowTemplate{}))

	cron := &wfv1.CronWorkflow{}
	require.NoError(t, fc.Get(ctx, key, cron), "a scheduled task agent must create a CronWorkflow")
	assert.Equal(t, []string{"*/5 * * * *"}, cron.Spec.Schedules)
	assert.Equal(t, "America/New_York", cron.Spec.Timezone)
	assert.Equal(t, wfv1.ReplaceConcurrent, cron.Spec.ConcurrencyPolicy)
	assert.False(t, cron.Spec.Suspend)

	// Each run references the template and is cleaned up after its TTL.
	require.NotNil(t, cron.Spec.WorkflowSpec.WorkflowTemplateRef)
	assert.Equal(t, agent.Name, cron.Spec.WorkflowSpec.WorkflowTemplateRef.Name)
	require.NotNil(t, cron.Spec.WorkflowSpec.TTLStrategy)
	require.NotNil(t, cron.Spec.WorkflowSpec.TTLStrategy.SecondsAfterCompletion)
	assert.Equal(t, defaultTaskTTLSeconds, *cron.Spec.WorkflowSpec.TTLStrategy.SecondsAfterCompletion)

	assert.True(t, errors.IsNotFound(fc.Get(ctx, key, &wfv1.Workflow{})),
		"task mode must not create a long-lived Workflow")

	// A task agent's pods come and go, so it gets no Service.
	assert.True(t, errors.IsNotFound(fc.Get(ctx, key, &corev1.Service{})),
		"task mode must not create a Service")
}

func TestLanguageAgentController_UnscheduledTaskIsTemplateOnly(t *testing.T) {
	agent := gen.LanguageAgent("manual-task-agent", "default")
	agent.Spec.Execution = langopv1alpha1.ExecutionSpec{Mode: langopv1alpha1.ExecutionModeTask}
	r, fc := newModeReconciler(t, agent)
	reconcileTwice(t, r, agent.Name, agent.Namespace)
	ctx := context.Background()
	key := types.NamespacedName{Name: agent.Name, Namespace: agent.Namespace}

	// The template is what `argo submit --from workflowtemplate/<agent>` targets.
	require.NoError(t, fc.Get(ctx, key, &wfv1.WorkflowTemplate{}))
	assert.True(t, errors.IsNotFound(fc.Get(ctx, key, &wfv1.CronWorkflow{})))
	assert.True(t, errors.IsNotFound(fc.Get(ctx, key, &wfv1.Workflow{})))
}

func TestLanguageAgentController_ModeFlipRemovesStaleObject(t *testing.T) {
	ctx := context.Background()

	t.Run("service_to_task_deletes_workflow", func(t *testing.T) {
		agent := gen.LanguageAgent("flip-to-task", "default")
		r, fc := newModeReconciler(t, agent)
		reconcileTwice(t, r, agent.Name, agent.Namespace)
		key := types.NamespacedName{Name: agent.Name, Namespace: agent.Namespace}
		require.NoError(t, fc.Get(ctx, key, &wfv1.Workflow{}))

		stored := &langopv1alpha1.LanguageAgent{}
		require.NoError(t, fc.Get(ctx, key, stored))
		stored.Spec.Execution = langopv1alpha1.ExecutionSpec{
			Mode:     langopv1alpha1.ExecutionModeTask,
			Schedule: "@hourly",
		}
		require.NoError(t, fc.Update(ctx, stored))
		reconcileTwice(t, r, agent.Name, agent.Namespace)

		assert.True(t, errors.IsNotFound(fc.Get(ctx, key, &wfv1.Workflow{})),
			"the service Workflow must be removed when the agent becomes a task")
		require.NoError(t, fc.Get(ctx, key, &wfv1.CronWorkflow{}))
	})

	t.Run("task_to_service_deletes_cronworkflow", func(t *testing.T) {
		agent := gen.LanguageAgent("flip-to-service", "default")
		agent.Spec.Execution = langopv1alpha1.ExecutionSpec{
			Mode:     langopv1alpha1.ExecutionModeTask,
			Schedule: "@hourly",
		}
		r, fc := newModeReconciler(t, agent)
		reconcileTwice(t, r, agent.Name, agent.Namespace)
		key := types.NamespacedName{Name: agent.Name, Namespace: agent.Namespace}
		require.NoError(t, fc.Get(ctx, key, &wfv1.CronWorkflow{}))

		stored := &langopv1alpha1.LanguageAgent{}
		require.NoError(t, fc.Get(ctx, key, stored))
		stored.Spec.Execution = langopv1alpha1.ExecutionSpec{Mode: langopv1alpha1.ExecutionModeService}
		require.NoError(t, fc.Update(ctx, stored))
		reconcileTwice(t, r, agent.Name, agent.Namespace)

		assert.True(t, errors.IsNotFound(fc.Get(ctx, key, &wfv1.CronWorkflow{})),
			"the CronWorkflow must be removed when the agent becomes a service")
		require.NoError(t, fc.Get(ctx, key, &wfv1.Workflow{}))
	})

	t.Run("clearing_schedule_deletes_cronworkflow", func(t *testing.T) {
		agent := gen.LanguageAgent("clear-schedule", "default")
		agent.Spec.Execution = langopv1alpha1.ExecutionSpec{
			Mode:     langopv1alpha1.ExecutionModeTask,
			Schedule: "@daily",
		}
		r, fc := newModeReconciler(t, agent)
		reconcileTwice(t, r, agent.Name, agent.Namespace)
		key := types.NamespacedName{Name: agent.Name, Namespace: agent.Namespace}
		require.NoError(t, fc.Get(ctx, key, &wfv1.CronWorkflow{}))

		stored := &langopv1alpha1.LanguageAgent{}
		require.NoError(t, fc.Get(ctx, key, stored))
		stored.Spec.Execution.Schedule = ""
		require.NoError(t, fc.Update(ctx, stored))
		reconcileTwice(t, r, agent.Name, agent.Namespace)

		assert.True(t, errors.IsNotFound(fc.Get(ctx, key, &wfv1.CronWorkflow{})))
		require.NoError(t, fc.Get(ctx, key, &wfv1.WorkflowTemplate{}), "the template survives")
	})
}

func TestLanguageAgentController_SuspendTearsDownWorkflow(t *testing.T) {
	ctx := context.Background()

	t.Run("service_workflow_removed", func(t *testing.T) {
		agent := gen.LanguageAgent("suspend-svc", "default")
		r, fc := newModeReconciler(t, agent)
		reconcileTwice(t, r, agent.Name, agent.Namespace)
		key := types.NamespacedName{Name: agent.Name, Namespace: agent.Namespace}
		require.NoError(t, fc.Get(ctx, key, &wfv1.Workflow{}))

		stored := &langopv1alpha1.LanguageAgent{}
		require.NoError(t, fc.Get(ctx, key, stored))
		suspend := true
		stored.Spec.Execution.Suspend = &suspend
		require.NoError(t, fc.Update(ctx, stored))
		reconcileTwice(t, r, agent.Name, agent.Namespace)

		assert.True(t, errors.IsNotFound(fc.Get(ctx, key, &wfv1.Workflow{})),
			"suspending a service agent must tear its Workflow down")
		// The template stays so the agent can still be invoked by hand.
		require.NoError(t, fc.Get(ctx, key, &wfv1.WorkflowTemplate{}))

		updated := &langopv1alpha1.LanguageAgent{}
		require.NoError(t, fc.Get(ctx, key, updated))
		assert.Equal(t, events.PhaseStatusSuspended, updated.Status.Phase)
	})

	t.Run("cronworkflow_suspended_not_deleted", func(t *testing.T) {
		agent := gen.LanguageAgent("suspend-cron", "default")
		suspend := true
		agent.Spec.Execution = langopv1alpha1.ExecutionSpec{
			Mode:     langopv1alpha1.ExecutionModeTask,
			Schedule: "@daily",
			Suspend:  &suspend,
		}
		r, fc := newModeReconciler(t, agent)
		reconcileTwice(t, r, agent.Name, agent.Namespace)
		key := types.NamespacedName{Name: agent.Name, Namespace: agent.Namespace}

		cron := &wfv1.CronWorkflow{}
		require.NoError(t, fc.Get(ctx, key, cron))
		assert.True(t, cron.Spec.Suspend, "a suspended task agent's CronWorkflow must stop firing")
	})
}

func TestLanguageAgentController_ServiceWorkflowReplacedOnConfigChange(t *testing.T) {
	ctx := context.Background()
	agent := gen.LanguageAgent("replace-agent", "default")
	r, fc := newModeReconciler(t, agent)
	reconcileTwice(t, r, agent.Name, agent.Namespace)
	key := types.NamespacedName{Name: agent.Name, Namespace: agent.Namespace}

	before := &wfv1.Workflow{}
	require.NoError(t, fc.Get(ctx, key, before))
	firstHash := before.Annotations[langoplabels.LabelKeyLangopConfigHash]
	require.NotEmpty(t, firstHash)

	// Change something that lands in the agent's config.yaml. A Workflow spec cannot
	// be updated in place, so the operator must replace it rather than leave the
	// agent running against a stale config.
	stored := &langopv1alpha1.LanguageAgent{}
	require.NoError(t, fc.Get(ctx, key, stored))
	stored.Spec.Instructions = "completely new instructions"
	require.NoError(t, fc.Update(ctx, stored))
	reconcileTwice(t, r, agent.Name, agent.Namespace)

	after := &wfv1.Workflow{}
	require.NoError(t, fc.Get(ctx, key, after))
	assert.NotEqual(t, firstHash, after.Annotations[langoplabels.LabelKeyLangopConfigHash],
		"the Workflow must be re-created with the new config hash")
}

func TestLanguageAgentController_TaskRunStatusFromLatestRun(t *testing.T) {
	ctx := context.Background()
	agent := gen.LanguageAgent("run-history-agent", "default")
	agent.Spec.Execution = langopv1alpha1.ExecutionSpec{
		Mode:     langopv1alpha1.ExecutionModeTask,
		Schedule: "@daily",
	}
	r, fc := newModeReconciler(t, agent)
	reconcileTwice(t, r, agent.Name, agent.Namespace)

	// Stand in for Argo: two runs the CronWorkflow fired, the newer one still going.
	older := metav1.NewTime(time.Now().Add(-2 * time.Hour))
	newer := metav1.NewTime(time.Now().Add(-10 * time.Minute))
	for _, run := range []struct {
		name     string
		phase    wfv1.WorkflowPhase
		started  metav1.Time
		finished metav1.Time
	}{
		{"run-history-agent-aaa", wfv1.WorkflowSucceeded, older, metav1.NewTime(older.Add(time.Minute))},
		{"run-history-agent-bbb", wfv1.WorkflowRunning, newer, metav1.Time{}},
	} {
		wf := &wfv1.Workflow{
			ObjectMeta: metav1.ObjectMeta{
				Name:      run.name,
				Namespace: agent.Namespace,
				Labels:    GetCommonLabels(agent.Name, "LanguageAgent"),
			},
		}
		wf.Status.Phase = run.phase
		wf.Status.StartedAt = run.started
		wf.Status.FinishedAt = run.finished
		require.NoError(t, fc.Create(ctx, wf))
	}

	reconcileTwice(t, r, agent.Name, agent.Namespace)

	updated := &langopv1alpha1.LanguageAgent{}
	require.NoError(t, fc.Get(ctx, types.NamespacedName{Name: agent.Name, Namespace: agent.Namespace}, updated))
	assert.Equal(t, "run-history-agent-bbb", updated.Status.LastRunName, "the newest run wins")
	assert.Equal(t, string(wfv1.WorkflowRunning), updated.Status.LastRunPhase)
	assert.Equal(t, events.PhaseStatusRunning, updated.Status.Phase)
	assert.Nil(t, updated.Status.LastRunFinishedAt, "a running run has not finished")
	assert.Empty(t, updated.Status.ActiveWorkflowName, "task mode has no long-lived Workflow")
}

func TestLanguageAgentController_AgentServiceAccountCanReportTaskResults(t *testing.T) {
	// The Argo executor writes a WorkflowTaskResult with the pod's own
	// ServiceAccount; without this rule every agent run fails at completion.
	agent := gen.LanguageAgent("taskresult-agent", "default")
	r, fc := newModeReconciler(t, agent)
	reconcileTwice(t, r, agent.Name, agent.Namespace)

	role := &rbacv1.Role{}
	require.NoError(t, fc.Get(context.Background(), types.NamespacedName{
		Name:      GenerateServiceAccountName(agent.Name),
		Namespace: agent.Namespace,
	}, role))

	var found bool
	for _, rule := range role.Rules {
		if slices.Contains(rule.APIGroups, "argoproj.io") && slices.Contains(rule.Resources, "workflowtaskresults") {
			assert.Contains(t, rule.Verbs, "create")
			assert.Contains(t, rule.Verbs, "patch")
			found = true
		}
	}
	assert.True(t, found, "agent Role must allow writing workflowtaskresults, got %+v", role.Rules)
}
