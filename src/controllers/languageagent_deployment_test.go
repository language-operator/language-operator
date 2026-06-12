package controllers

import (
	"context"
	"strings"
	"testing"

	"github.com/go-logr/logr"
	langopv1alpha1 "github.com/language-operator/language-operator/api/v1alpha1"
	"github.com/language-operator/language-operator/controllers/testutil"
	"github.com/language-operator/language-operator/internal/testutil/gen"
	"github.com/language-operator/language-operator/pkg/events"
	langoplabels "github.com/language-operator/language-operator/pkg/labels"
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
)

func TestLanguageAgentController_DeploymentCreation(t *testing.T) {
	scheme := testutil.SetupTestScheme(t)

	agent := &langopv1alpha1.LanguageAgent{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-deployment-agent",
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
	deployment := &appsv1.Deployment{}
	err = fakeClient.Get(ctx, types.NamespacedName{
		Name:      agent.Name,
		Namespace: agent.Namespace,
	}, deployment)
	if err != nil {
		t.Fatalf("Expected Deployment to exist for autonomous agent, but got error: %v", err)
	}

	// Verify Deployment has correct image
	if len(deployment.Spec.Template.Spec.Containers) != 1 {
		t.Errorf("Expected 1 container, got %d", len(deployment.Spec.Template.Spec.Containers))
	}
	if deployment.Spec.Template.Spec.Containers[0].Image != agent.Spec.Image {
		t.Errorf("Expected image '%s', got '%s'", agent.Spec.Image, deployment.Spec.Template.Spec.Containers[0].Image)
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

	// Verify Deployment was created
	deployment := &appsv1.Deployment{}
	err = fakeClient.Get(ctx, types.NamespacedName{
		Name:      agent.Name,
		Namespace: agent.Namespace,
	}, deployment)
	if err != nil {
		t.Fatalf("Expected Deployment to exist, but got error: %v", err)
	}

	// Verify Pod security context
	podSec := deployment.Spec.Template.Spec.SecurityContext
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

	// Verify Deployment was created
	deployment := &appsv1.Deployment{}
	err = fakeClient.Get(ctx, types.NamespacedName{
		Name:      agent.Name,
		Namespace: agent.Namespace,
	}, deployment)
	if err != nil {
		t.Fatalf("Expected Deployment to exist, but got error: %v", err)
	}

	// Verify container security context
	if len(deployment.Spec.Template.Spec.Containers) == 0 {
		t.Fatal("No containers found in deployment")
	}

	containerSec := deployment.Spec.Template.Spec.Containers[0].SecurityContext
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

	// Verify Deployment was created
	deployment := &appsv1.Deployment{}
	err = fakeClient.Get(ctx, types.NamespacedName{
		Name:      agent.Name,
		Namespace: agent.Namespace,
	}, deployment)
	if err != nil {
		t.Fatalf("Expected Deployment to exist, but got error: %v", err)
	}

	// Check for tmpfs volumes
	expectedVolumes := map[string]string{
		"tmp": "/tmp",
	}

	volumes := deployment.Spec.Template.Spec.Volumes
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
	if len(deployment.Spec.Template.Spec.Containers) == 0 {
		t.Fatal("No containers found in deployment")
	}

	volumeMounts := deployment.Spec.Template.Spec.Containers[0].VolumeMounts
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
		expected := "http://my-tool.default.svc.cluster.local:8080"
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
		expected := "http://localhost:9090"
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
		expected := "http://port-tool.default.svc.cluster.local:9999"
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

func TestLanguageAgentController_SidecarToolInjectedIntoDeployment(t *testing.T) {
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

	dep := &appsv1.Deployment{}
	require.NoError(t, fakeClient.Get(ctx, req.NamespacedName, dep))

	initContainers := dep.Spec.Template.Spec.InitContainers
	require.Len(t, initContainers, 1, "expected exactly one init container from sidecar tool")
	assert.Equal(t, "tool-my-sidecar", initContainers[0].Name)
	assert.Equal(t, "ghcr.io/language-operator/tool:latest", initContainers[0].Image)
	require.NotNil(t, dep.Spec.Template.Spec.ShareProcessNamespace)
	assert.True(t, *dep.Spec.Template.Spec.ShareProcessNamespace)
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

	deployment := &appsv1.Deployment{}
	if err := fakeClient.Get(ctx, types.NamespacedName{Name: agent.Name, Namespace: agent.Namespace}, deployment); err != nil {
		t.Fatalf("Expected Deployment to exist: %v", err)
	}

	expectedConfigMapName := GenerateConfigMapName(agent.Name, "agent")

	var foundVolume bool
	for _, vol := range deployment.Spec.Template.Spec.Volumes {
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
		t.Error("expected agent-config volume in deployment, not found")
	}

	containers := deployment.Spec.Template.Spec.Containers
	if len(containers) == 0 {
		t.Fatal("no containers in deployment")
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

	deployment := &appsv1.Deployment{}
	if err := fakeClient.Get(ctx, types.NamespacedName{Name: agent.Name, Namespace: agent.Namespace}, deployment); err != nil {
		t.Fatalf("Expected Deployment to exist: %v", err)
	}

	podSpec := deployment.Spec.Template.Spec

	if podSpec.NodeSelector["kubernetes.io/arch"] != "amd64" {
		t.Errorf("Expected NodeSelector kubernetes.io/arch=amd64, got %v", podSpec.NodeSelector)
	}
	if len(podSpec.Tolerations) == 0 || podSpec.Tolerations[0].Key != "gpu" {
		t.Errorf("Expected Toleration gpu, got %v", podSpec.Tolerations)
	}
	if len(podSpec.TopologySpreadConstraints) == 0 || podSpec.TopologySpreadConstraints[0].TopologyKey != "topology.kubernetes.io/zone" {
		t.Errorf("Expected TopologySpreadConstraint, got %v", podSpec.TopologySpreadConstraints)
	}
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

	deployment := &appsv1.Deployment{}
	if err := fakeClient.Get(ctx, types.NamespacedName{Name: agent.Name, Namespace: agent.Namespace}, deployment); err != nil {
		t.Fatalf("Expected Deployment to exist: %v", err)
	}

	podMeta := deployment.Spec.Template.ObjectMeta

	// User label should be present
	if podMeta.Labels["cost-center"] != "team-a" {
		t.Errorf("Expected pod label cost-center=team-a, got %q", podMeta.Labels["cost-center"])
	}
	// Operator label must not be overridden by user; GetCommonLabels sets it to the agent name.
	if podMeta.Labels["app.kubernetes.io/name"] != agent.Name {
		t.Errorf("Operator label app.kubernetes.io/name should be %q, got %q", agent.Name, podMeta.Labels["app.kubernetes.io/name"])
	}
	// Selector labels must remain unchanged (operator labels only)
	selectorLabels := deployment.Spec.Selector.MatchLabels
	if _, hasUserLabel := selectorLabels["cost-center"]; hasUserLabel {
		t.Error("User pod label should not appear in selector MatchLabels")
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

	deployment := &appsv1.Deployment{}
	if err := fakeClient.Get(ctx, types.NamespacedName{Name: agent.Name, Namespace: agent.Namespace}, deployment); err != nil {
		t.Fatalf("Expected Deployment to exist: %v", err)
	}

	annotations := deployment.Spec.Template.Annotations
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

	deployment := &appsv1.Deployment{}
	if err := fakeClient.Get(ctx, types.NamespacedName{Name: agent.Name, Namespace: agent.Namespace}, deployment); err != nil {
		t.Fatalf("Expected Deployment to exist: %v", err)
	}

	// Operator-managed volumes (tmp, agent-config) must still be present
	volumeNames := map[string]bool{}
	for _, v := range deployment.Spec.Template.Spec.Volumes {
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
	for _, m := range deployment.Spec.Template.Spec.Containers[0].VolumeMounts {
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

	deployment := &appsv1.Deployment{}
	if err := fakeClient.Get(ctx, types.NamespacedName{Name: agent.Name, Namespace: agent.Namespace}, deployment); err != nil {
		t.Fatalf("Expected Deployment to exist: %v", err)
	}

	container := deployment.Spec.Template.Spec.Containers[0]
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

	deployment := &appsv1.Deployment{}
	if err := fakeClient.Get(ctx, types.NamespacedName{Name: agent.Name, Namespace: agent.Namespace}, deployment); err != nil {
		t.Fatalf("Expected Deployment to exist: %v", err)
	}

	podSec := deployment.Spec.Template.Spec.SecurityContext
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
	containerSec := deployment.Spec.Template.Spec.Containers[0].SecurityContext
	if containerSec == nil {
		t.Fatal("Container SecurityContext should still be set by operator")
	}
	if containerSec.AllowPrivilegeEscalation == nil || *containerSec.AllowPrivilegeEscalation {
		t.Error("Container SecurityContext.AllowPrivilegeEscalation should be false")
	}
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
