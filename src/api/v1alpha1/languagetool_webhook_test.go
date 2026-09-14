/*
Copyright 2025 Langop Team.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package v1alpha1

import (
	"context"
	"testing"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

// mockToolRegistryManager is a test double for RegistryManager.
type mockToolRegistryManager struct {
	registries []string
}

func (m *mockToolRegistryManager) GetRegistries() []string {
	return m.registries
}

func newToolWebhook(t *testing.T, registries []string) *LanguageToolWebhook {
	t.Helper()
	scheme := newWebhookTestScheme(t)
	// Include a LanguageCluster so cluster-membership validation passes.
	cluster := &LanguageCluster{ObjectMeta: metav1.ObjectMeta{Name: "default"}}
	c := fake.NewClientBuilder().WithScheme(scheme).WithObjects(cluster).Build()
	return &LanguageToolWebhook{
		Client:          c,
		RegistryManager: &mockToolRegistryManager{registries: registries},
	}
}

func makeTool(image string) *LanguageTool {
	return &LanguageTool{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-tool",
			Namespace: "default",
		},
		Spec: LanguageToolSpec{
			Image: image,
		},
	}
}

func TestLanguageToolWebhook_ValidateCreate_RegistryAllowed(t *testing.T) {
	h := newToolWebhook(t, []string{"ghcr.io", "docker.io"})
	tool := makeTool("ghcr.io/language-operator/tool:latest")

	_, err := h.ValidateCreate(context.Background(), tool)
	if err != nil {
		t.Errorf("expected no error for allowed registry, got: %v", err)
	}
}

func TestLanguageToolWebhook_ValidateCreate_RegistryDenied(t *testing.T) {
	h := newToolWebhook(t, []string{"docker.io"})
	tool := makeTool("ghcr.io/language-operator/tool:latest")

	_, err := h.ValidateCreate(context.Background(), tool)
	if err == nil {
		t.Error("expected error for denied registry, got nil")
	}
}

func TestLanguageToolWebhook_ValidateCreate_NoRegistryList(t *testing.T) {
	h := newToolWebhook(t, []string{})
	tool := makeTool("private.registry.example.com/tool:v1")

	_, err := h.ValidateCreate(context.Background(), tool)
	if err != nil {
		t.Errorf("expected no error when registry list is empty, got: %v", err)
	}
}

func TestLanguageToolWebhook_ValidateCreate_NilRegistryManager(t *testing.T) {
	scheme := newWebhookTestScheme(t)
	cluster := &LanguageCluster{ObjectMeta: metav1.ObjectMeta{Name: "default"}}
	c := fake.NewClientBuilder().WithScheme(scheme).WithObjects(cluster).Build()
	h := &LanguageToolWebhook{Client: c, RegistryManager: nil}
	tool := makeTool("private.registry.example.com/tool:v1")

	_, err := h.ValidateCreate(context.Background(), tool)
	if err != nil {
		t.Errorf("expected no error when RegistryManager is nil, got: %v", err)
	}
}

func TestLanguageToolWebhook_ValidateUpdate_RegistryDenied(t *testing.T) {
	h := newToolWebhook(t, []string{"docker.io"})
	old := makeTool("docker.io/library/tool:v1")
	updated := makeTool("ghcr.io/language-operator/tool:v2")

	_, err := h.ValidateUpdate(context.Background(), old, updated)
	if err == nil {
		t.Error("expected error when updated image uses denied registry, got nil")
	}
}

func TestLanguageToolWebhook_ValidateUpdate_RegistryAllowed(t *testing.T) {
	h := newToolWebhook(t, []string{"ghcr.io"})
	old := makeTool("ghcr.io/language-operator/tool:v1")
	updated := makeTool("ghcr.io/language-operator/tool:v2")

	_, err := h.ValidateUpdate(context.Background(), old, updated)
	if err != nil {
		t.Errorf("expected no error for allowed registry on update, got: %v", err)
	}
}

// TestLanguageToolWebhook_Default_IsNoOp guards against regressing the fix in
// #915: Transport, Deployment.Resources, and the stdio-tool Image workaround
// all moved to the CRD schema (+kubebuilder:default on Transport and
// Deployment, +kubebuilder:validation:XValidation relaxing Image's
// requirement for transport=stdio — see languagetool_types.go), so they apply
// even when this webhook — or all webhooks — are disabled. Default() must
// leave the object untouched; real defaulting behavior is covered by the
// envtest suite (suite_test.go), which exercises the actual CRD schema.
func TestLanguageToolWebhook_Default_IsNoOp(t *testing.T) {
	h := newToolWebhook(t, nil)
	tool := &LanguageTool{
		ObjectMeta: metav1.ObjectMeta{Name: "s", Namespace: "default"},
		Spec:       LanguageToolSpec{Transport: "stdio", Stdio: &StdioServerSpec{Command: []string{"npx", "-y", "x"}}},
	}
	before := tool.DeepCopy()
	if err := h.Default(context.Background(), tool); err != nil {
		t.Fatalf("Default: %v", err)
	}
	if tool.Spec.Transport != before.Spec.Transport {
		t.Errorf("Default() changed Transport: %q -> %q", before.Spec.Transport, tool.Spec.Transport)
	}
	if tool.Spec.Image != before.Spec.Image {
		t.Errorf("Default() changed Image (CRD relaxation now handles this): %q -> %q", before.Spec.Image, tool.Spec.Image)
	}
	if tool.Spec.Deployment.Resources.Requests != nil || tool.Spec.Deployment.Resources.Limits != nil {
		t.Errorf("expected Deployment.Resources to stay empty (CRD default now handles this), got %+v", tool.Spec.Deployment.Resources)
	}
}

func TestLanguageToolWebhook_Validate_StdioRequiresCommand(t *testing.T) {
	h := newToolWebhook(t, nil)
	tool := &LanguageTool{
		ObjectMeta: metav1.ObjectMeta{Name: "s", Namespace: "default"},
		Spec:       LanguageToolSpec{Image: DefaultMCPBridgeImage, Transport: "stdio"},
	}
	if _, err := h.ValidateCreate(context.Background(), tool); err == nil {
		t.Error("expected error when transport=stdio without stdio.command, got nil")
	}
}

func TestLanguageToolWebhook_Validate_StdioConfigOnNonStdio(t *testing.T) {
	h := newToolWebhook(t, nil)
	tool := makeTool("ghcr.io/x/y:1")
	tool.Spec.Transport = "streamable-http"
	tool.Spec.Stdio = &StdioServerSpec{Command: []string{"npx", "x"}}
	if _, err := h.ValidateCreate(context.Background(), tool); err == nil {
		t.Error("expected error when spec.stdio is set on a non-stdio transport, got nil")
	}
}

func TestLanguageToolWebhook_Validate_StdioSkipsRegistryAndWarnsOnImage(t *testing.T) {
	// Registry allowlist excludes ghcr.io; a stdio tool should still pass because the user
	// supplies a command, not an image.
	h := newToolWebhook(t, []string{"docker.io"})

	tool := &LanguageTool{
		ObjectMeta: metav1.ObjectMeta{Name: "s", Namespace: "default"},
		Spec: LanguageToolSpec{
			Image:     DefaultMCPBridgeImage, // ghcr.io — would be denied for a non-stdio tool
			Transport: "stdio",
			Stdio:     &StdioServerSpec{Command: []string{"npx", "-y", "x"}},
		},
	}
	if _, err := h.ValidateCreate(context.Background(), tool); err != nil {
		t.Errorf("stdio tool should skip registry validation, got: %v", err)
	}

	// A user-set custom image on a stdio tool produces a warning (it is ignored).
	tool.Spec.Image = "myregistry.example.com/custom:1"
	warnings, err := h.ValidateCreate(context.Background(), tool)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if len(warnings) == 0 {
		t.Error("expected a warning that spec.image is ignored for stdio tools")
	}
}
