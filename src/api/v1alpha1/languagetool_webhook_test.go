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
