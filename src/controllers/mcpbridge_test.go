package controllers

import (
	"context"
	"strings"
	"testing"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	langopv1alpha1 "github.com/language-operator/language-operator/api/v1alpha1"
	"github.com/language-operator/language-operator/controllers/testutil"
	"github.com/language-operator/language-operator/internal/testutil/gen"
)

// getToolDeployment reconciles a LanguageTool twice (finalizer, then resources) and returns
// the created Deployment.
func getToolDeployment(t *testing.T, r *LanguageToolReconciler, tool *langopv1alpha1.LanguageTool) *appsv1.Deployment {
	t.Helper()
	ctx := context.Background()
	for i := 0; i < 2; i++ {
		if _, err := r.Reconcile(ctx, ctrl.Request{NamespacedName: types.NamespacedName{Name: tool.Name, Namespace: tool.Namespace}}); err != nil {
			t.Fatalf("Reconcile %d failed: %v", i+1, err)
		}
	}
	deployment := &appsv1.Deployment{}
	if err := r.Get(ctx, types.NamespacedName{Name: tool.Name, Namespace: tool.Namespace}, deployment); err != nil {
		t.Fatalf("Expected Deployment to exist: %v", err)
	}
	return deployment
}

func volumeNames(vols []corev1.Volume) map[string]bool {
	out := make(map[string]bool, len(vols))
	for _, v := range vols {
		out[v.Name] = true
	}
	return out
}

func mountPaths(mounts []corev1.VolumeMount) map[string]string {
	out := make(map[string]string, len(mounts))
	for _, m := range mounts {
		out[m.Name] = m.MountPath
	}
	return out
}

func envValue(envs []corev1.EnvVar, name string) (string, bool) {
	for _, e := range envs {
		if e.Name == name {
			return e.Value, true
		}
	}
	return "", false
}

func TestToolDeployment_StdioInjectsBridge(t *testing.T) {
	scheme := testutil.SetupTestScheme(t)

	tool := gen.LanguageTool("ctx7", "default",
		gen.SetToolImage("ignored:latest"),
		gen.SetToolPort(8080),
		gen.SetToolTransport("stdio"),
		gen.SetToolStdioCommand("npx", "-y", "@upstash/context7-mcp"),
	)

	fakeClient := fake.NewClientBuilder().
		WithScheme(scheme).
		WithObjects(gen.ReadyCluster("default"), tool).
		WithStatusSubresource(tool).
		Build()

	r := &LanguageToolReconciler{Client: fakeClient, Scheme: scheme}
	deployment := getToolDeployment(t, r, tool)

	podSpec := deployment.Spec.Template.Spec
	c := podSpec.Containers[0]

	// Bridge image (default, since MCPBridgeImage unset) replaces the user image.
	if c.Image != langopv1alpha1.DefaultMCPBridgeImage {
		t.Errorf("container image = %q, want default bridge %q", c.Image, langopv1alpha1.DefaultMCPBridgeImage)
	}
	if len(c.Command) != 1 || c.Command[0] != "supergateway" {
		t.Errorf("command = %v, want [supergateway]", c.Command)
	}
	args := strings.Join(c.Args, " ")
	for _, want := range []string{"--stdio", "npx -y @upstash/context7-mcp", "--stateful", "--streamableHttpPath /mcp", "--healthEndpoint /health", "--port 8080"} {
		if !strings.Contains(args, want) {
			t.Errorf("bridge args %q missing %q", args, want)
		}
	}

	// Writable scratch: HOME + cache env, cache + tmp volumes/mounts.
	if v, ok := envValue(c.Env, "HOME"); !ok || v != mcpBridgeHome {
		t.Errorf("HOME env = %q (present=%v), want %q", v, ok, mcpBridgeHome)
	}
	vols := volumeNames(podSpec.Volumes)
	if !vols[bridgeCacheVolumeName(mcpBridgeVolumePrefix)] || !vols[bridgeTmpVolumeName(mcpBridgeVolumePrefix)] {
		t.Errorf("pod volumes missing bridge cache/tmp: %v", podSpec.Volumes)
	}
	mounts := mountPaths(c.VolumeMounts)
	if mounts[bridgeCacheVolumeName(mcpBridgeVolumePrefix)] != mcpBridgeHome || mounts[bridgeTmpVolumeName(mcpBridgeVolumePrefix)] != mcpBridgeTmpPath {
		t.Errorf("bridge mounts wrong: %v", c.VolumeMounts)
	}

	// Operator-injected /health readiness probe.
	if c.ReadinessProbe == nil || c.ReadinessProbe.HTTPGet == nil || c.ReadinessProbe.HTTPGet.Path != "/health" {
		t.Errorf("readiness probe not /health httpGet: %+v", c.ReadinessProbe)
	}

	// Hardened security context preserved.
	if c.SecurityContext == nil || c.SecurityContext.ReadOnlyRootFilesystem == nil || !*c.SecurityContext.ReadOnlyRootFilesystem {
		t.Errorf("expected readOnlyRootFilesystem=true, got %+v", c.SecurityContext)
	}
}

func TestToolDeployment_StreamableHTTPUnchanged(t *testing.T) {
	scheme := testutil.SetupTestScheme(t)

	tool := gen.LanguageTool("http-tool", "default",
		gen.SetToolImage("myregistry/tool:1.2.3"),
		gen.SetToolPort(8080),
		gen.SetToolTransport("streamable-http"),
	)
	tool.Spec.Deployment.Command = []string{"/server"}
	tool.Spec.Deployment.Args = []string{"--serve"}

	fakeClient := fake.NewClientBuilder().
		WithScheme(scheme).
		WithObjects(gen.ReadyCluster("default"), tool).
		WithStatusSubresource(tool).
		Build()

	r := &LanguageToolReconciler{Client: fakeClient, Scheme: scheme}
	deployment := getToolDeployment(t, r, tool)
	c := deployment.Spec.Template.Spec.Containers[0]

	if c.Image != "myregistry/tool:1.2.3" {
		t.Errorf("image = %q, want the user image (no bridge)", c.Image)
	}
	if len(c.Command) != 1 || c.Command[0] != "/server" {
		t.Errorf("command = %v, want user command [/server]", c.Command)
	}
	// No bridge scratch volumes for non-stdio tools.
	if volumeNames(deployment.Spec.Template.Spec.Volumes)[bridgeCacheVolumeName(mcpBridgeVolumePrefix)] {
		t.Error("streamable-http tool should not get bridge cache volume")
	}
	if _, ok := envValue(c.Env, "HOME"); ok {
		t.Error("streamable-http tool should not get bridge HOME env")
	}
}

func TestToolDeployment_StdioRespectsBridgeImageOverride(t *testing.T) {
	scheme := testutil.SetupTestScheme(t)

	tool := gen.LanguageTool("ctx7", "default",
		gen.SetToolTransport("stdio"),
		gen.SetToolStdioCommand("uvx", "mcp-server-git"),
	)
	fakeClient := fake.NewClientBuilder().
		WithScheme(scheme).
		WithObjects(gen.ReadyCluster("default"), tool).
		WithStatusSubresource(tool).
		Build()

	r := &LanguageToolReconciler{
		Client:                   fakeClient,
		Scheme:                   scheme,
		MCPBridgeImage:           "ghcr.io/language-operator/mcp-bridge:v9",
		MCPBridgeImagePullPolicy: corev1.PullIfNotPresent,
	}
	deployment := getToolDeployment(t, r, tool)
	c := deployment.Spec.Template.Spec.Containers[0]

	if c.Image != "ghcr.io/language-operator/mcp-bridge:v9" {
		t.Errorf("image = %q, want the configured bridge override", c.Image)
	}
	if c.ImagePullPolicy != corev1.PullIfNotPresent {
		t.Errorf("pull policy = %q, want IfNotPresent", c.ImagePullPolicy)
	}
}
