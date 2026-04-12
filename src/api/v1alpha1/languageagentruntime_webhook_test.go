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
	"strings"
	"testing"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// mockRuntimeRegistryManager is a test double for RegistryManager.
type mockRuntimeRegistryManager struct {
	registries []string
}

func (m *mockRuntimeRegistryManager) GetRegistries() []string {
	return m.registries
}

func TestLanguageAgentRuntimeValidateSpec(t *testing.T) {
	h := &LanguageAgentRuntimeWebhook{}

	t.Run("empty spec is valid", func(t *testing.T) {
		rt := &LanguageAgentRuntime{ObjectMeta: metav1.ObjectMeta{Name: "test-runtime"}}
		if err := h.validateSpec(rt); err != nil {
			t.Errorf("validateSpec() unexpected error: %v", err)
		}
	})

	t.Run("valid image is accepted", func(t *testing.T) {
		rt := &LanguageAgentRuntime{
			ObjectMeta: metav1.ObjectMeta{Name: "test-runtime"},
			Spec:       LanguageAgentRuntimeSpec{Image: "ghcr.io/test/agent:v1"},
		}
		if err := h.validateSpec(rt); err != nil {
			t.Errorf("validateSpec() unexpected error: %v", err)
		}
	})
}

func TestLanguageAgentRuntimeWorkspaceValidation(t *testing.T) {
	tests := []struct {
		name    string
		size    string
		wantErr bool
		errMsg  string
	}{
		{
			name:    "valid workspace size",
			size:    "10Gi",
			wantErr: false,
		},
		{
			name:    "valid decimal workspace size",
			size:    "1.5Ti",
			wantErr: false,
		},
		{
			name:    "zero workspace size rejected",
			size:    "0",
			wantErr: true,
			errMsg:  "cannot be zero",
		},
		{
			name:    "empty workspace size rejected",
			size:    "",
			wantErr: true,
			errMsg:  "cannot be empty",
		},
		{
			name:    "invalid format rejected",
			size:    "notasize",
			wantErr: true,
			errMsg:  "invalid format",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h := &LanguageAgentRuntimeWebhook{}
			rt := &LanguageAgentRuntime{
				ObjectMeta: metav1.ObjectMeta{Name: "test-runtime"},
				Spec: LanguageAgentRuntimeSpec{
					Workspace: &WorkspaceSpec{Size: tt.size},
				},
			}
			err := h.validateSpec(rt)
			if (err != nil) != tt.wantErr {
				t.Errorf("validateSpec() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if tt.wantErr && !strings.Contains(err.Error(), tt.errMsg) {
				t.Errorf("validateSpec() error = %q, want to contain %q", err.Error(), tt.errMsg)
			}
		})
	}
}

func TestLanguageAgentRuntimePortValidation(t *testing.T) {
	boolPtr := func(b bool) *bool { return &b }

	tests := []struct {
		name    string
		ports   []AgentPort
		wantErr bool
		errMsg  string
	}{
		{
			name:    "valid single port",
			ports:   []AgentPort{{Name: "http", Port: 8080}},
			wantErr: false,
		},
		{
			name: "duplicate port name rejected",
			ports: []AgentPort{
				{Name: "http", Port: 8080},
				{Name: "http", Port: 9090},
			},
			wantErr: true,
			errMsg:  "not unique",
		},
		{
			name: "duplicate port number rejected",
			ports: []AgentPort{
				{Name: "http", Port: 8080},
				{Name: "grpc", Port: 8080},
			},
			wantErr: true,
			errMsg:  "not unique",
		},
		{
			name: "multiple expose ports rejected",
			ports: []AgentPort{
				{Name: "http", Port: 8080, Expose: boolPtr(true)},
				{Name: "grpc", Port: 9090, Expose: boolPtr(true)},
			},
			wantErr: true,
			errMsg:  "at most one port",
		},
		{
			name: "single expose port allowed",
			ports: []AgentPort{
				{Name: "http", Port: 8080, Expose: boolPtr(true)},
				{Name: "grpc", Port: 9090},
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h := &LanguageAgentRuntimeWebhook{}
			rt := &LanguageAgentRuntime{
				ObjectMeta: metav1.ObjectMeta{Name: "test-runtime"},
				Spec:       LanguageAgentRuntimeSpec{Ports: tt.ports},
			}
			err := h.validateSpec(rt)
			if (err != nil) != tt.wantErr {
				t.Errorf("validateSpec() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if tt.wantErr && !strings.Contains(err.Error(), tt.errMsg) {
				t.Errorf("validateSpec() error = %q, want to contain %q", err.Error(), tt.errMsg)
			}
		})
	}
}

func TestLanguageAgentRuntimeWebhookValidateCreate(t *testing.T) {
	h := &LanguageAgentRuntimeWebhook{}

	t.Run("valid runtime passes", func(t *testing.T) {
		rt := &LanguageAgentRuntime{
			ObjectMeta: metav1.ObjectMeta{Name: "test-runtime"},
			Spec:       LanguageAgentRuntimeSpec{Image: "ghcr.io/test/agent:v1"},
		}
		_, err := h.ValidateCreate(context.Background(), rt)
		if err != nil {
			t.Errorf("ValidateCreate() unexpected error: %v", err)
		}
	})

	t.Run("invalid workspace size rejected", func(t *testing.T) {
		rt := &LanguageAgentRuntime{
			ObjectMeta: metav1.ObjectMeta{Name: "test-runtime"},
			Spec: LanguageAgentRuntimeSpec{
				Workspace: &WorkspaceSpec{Size: "0"},
			},
		}
		_, err := h.ValidateCreate(context.Background(), rt)
		if err == nil {
			t.Error("ValidateCreate() expected error for zero workspace size, got nil")
		}
	})
}

func TestLanguageAgentRuntimeWebhookValidateUpdate(t *testing.T) {
	h := &LanguageAgentRuntimeWebhook{}

	t.Run("valid update passes", func(t *testing.T) {
		old := &LanguageAgentRuntime{ObjectMeta: metav1.ObjectMeta{Name: "test-runtime"}}
		rt := &LanguageAgentRuntime{
			ObjectMeta: metav1.ObjectMeta{Name: "test-runtime"},
			Spec:       LanguageAgentRuntimeSpec{Image: "ghcr.io/test/agent:v2"},
		}
		_, err := h.ValidateUpdate(context.Background(), old, rt)
		if err != nil {
			t.Errorf("ValidateUpdate() unexpected error: %v", err)
		}
	})

	t.Run("update during deletion is skipped", func(t *testing.T) {
		now := metav1.Now()
		old := &LanguageAgentRuntime{ObjectMeta: metav1.ObjectMeta{Name: "test-runtime"}}
		rt := &LanguageAgentRuntime{
			ObjectMeta: metav1.ObjectMeta{
				Name:              "test-runtime",
				DeletionTimestamp: &now,
			},
			Spec: LanguageAgentRuntimeSpec{
				Workspace: &WorkspaceSpec{Size: "0"},
			},
		}
		_, err := h.ValidateUpdate(context.Background(), old, rt)
		if err != nil {
			t.Errorf("ValidateUpdate() during deletion should skip validation, got error: %v", err)
		}
	})
}

func TestLanguageAgentRuntimeWebhookValidateDelete(t *testing.T) {
	h := &LanguageAgentRuntimeWebhook{}
	rt := &LanguageAgentRuntime{ObjectMeta: metav1.ObjectMeta{Name: "test-runtime"}}
	_, err := h.ValidateDelete(context.Background(), rt)
	if err != nil {
		t.Errorf("ValidateDelete() unexpected error: %v", err)
	}
}

func TestLanguageAgentRuntimeWebhook_RegistryValidation(t *testing.T) {
	tests := []struct {
		name        string
		image       string
		registries  []string
		nilRM       bool
		wantErr     bool
		errContains string
	}{
		{
			name:       "allowed registry accepted",
			image:      "ghcr.io/language-operator/agent:v1",
			registries: []string{"ghcr.io", "docker.io"},
			wantErr:    false,
		},
		{
			name:        "denied registry rejected",
			image:       "private.example.com/agent:v1",
			registries:  []string{"ghcr.io", "docker.io"},
			wantErr:     true,
			errContains: "spec.image",
		},
		{
			name:       "empty registry list allows any image",
			image:      "private.example.com/agent:v1",
			registries: []string{},
			wantErr:    false,
		},
		{
			name:    "nil RegistryManager allows any image",
			image:   "private.example.com/agent:v1",
			nilRM:   true,
			wantErr: false,
		},
		{
			name:       "no image set skips registry check",
			image:      "",
			registries: []string{"ghcr.io"},
			wantErr:    false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var rm RegistryManager
			if !tt.nilRM {
				rm = &mockRuntimeRegistryManager{registries: tt.registries}
			}
			h := &LanguageAgentRuntimeWebhook{RegistryManager: rm}
			rt := &LanguageAgentRuntime{
				ObjectMeta: metav1.ObjectMeta{Name: "test-runtime"},
				Spec:       LanguageAgentRuntimeSpec{Image: tt.image},
			}
			_, err := h.ValidateCreate(context.Background(), rt)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateCreate() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if tt.wantErr && !strings.Contains(err.Error(), tt.errContains) {
				t.Errorf("ValidateCreate() error = %q, want to contain %q", err.Error(), tt.errContains)
			}
		})
	}
}

func TestLanguageAgentRuntimeWebhook_ResourcesValidation(t *testing.T) {
	tests := []struct {
		name        string
		resources   corev1.ResourceRequirements
		wantErr     bool
		errContains string
	}{
		{
			name: "valid requests and limits accepted",
			resources: corev1.ResourceRequirements{
				Requests: corev1.ResourceList{
					corev1.ResourceCPU:    resource.MustParse("100m"),
					corev1.ResourceMemory: resource.MustParse("256Mi"),
				},
				Limits: corev1.ResourceList{
					corev1.ResourceCPU:    resource.MustParse("1"),
					corev1.ResourceMemory: resource.MustParse("1Gi"),
				},
			},
			wantErr: false,
		},
		{
			name:      "empty resources accepted",
			resources: corev1.ResourceRequirements{},
			wantErr:   false,
		},
		{
			name: "zero CPU request rejected",
			resources: corev1.ResourceRequirements{
				Requests: corev1.ResourceList{
					corev1.ResourceCPU: resource.MustParse("0"),
				},
			},
			wantErr:     true,
			errContains: "spec.deployment.resources.requests",
		},
		{
			name: "zero memory limit rejected",
			resources: corev1.ResourceRequirements{
				Limits: corev1.ResourceList{
					corev1.ResourceMemory: resource.MustParse("0"),
				},
			},
			wantErr:     true,
			errContains: "spec.deployment.resources.limits",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h := &LanguageAgentRuntimeWebhook{}
			rt := &LanguageAgentRuntime{
				ObjectMeta: metav1.ObjectMeta{Name: "test-runtime"},
				Spec: LanguageAgentRuntimeSpec{
					Deployment: DeploymentSpec{
						Resources: tt.resources,
					},
				},
			}
			_, err := h.ValidateCreate(context.Background(), rt)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateCreate() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if tt.wantErr && !strings.Contains(err.Error(), tt.errContains) {
				t.Errorf("ValidateCreate() error = %q, want to contain %q", err.Error(), tt.errContains)
			}
		})
	}
}
