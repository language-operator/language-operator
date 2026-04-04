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

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func boolPtr(b bool) *bool { return &b }

func TestLanguageAgentDefault(t *testing.T) {
	tests := []struct {
		name     string
		agent    *LanguageAgent
		expected *WorkspaceSpec
	}{
		{
			name: "workspace defaults to enabled when nil",
			agent: &LanguageAgent{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-agent",
					Namespace: "default",
				},
				Spec: LanguageAgentSpec{
					Image: "test:latest",
					Models: []ModelReference{
						{Name: "test-model"},
					},
					Instructions: "test instructions",
					// Workspace is nil
				},
			},
			expected: &WorkspaceSpec{
				Enabled:    boolPtr(true),
				Size:       "10Gi",
				AccessMode: "ReadWriteOnce",
				MountPath:  "/workspace",
			},
		},
		{
			name: "workspace not overridden when explicitly set",
			agent: &LanguageAgent{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-agent",
					Namespace: "default",
				},
				Spec: LanguageAgentSpec{
					Image: "test:latest",
					Models: []ModelReference{
						{Name: "test-model"},
					},
					Instructions: "test instructions",
					Workspace: &WorkspaceSpec{
						Enabled:    boolPtr(false),
						Size:       "5Gi",
						AccessMode: "ReadWriteMany",
						MountPath:  "/custom",
					},
				},
			},
			expected: &WorkspaceSpec{
				Enabled:    boolPtr(false),
				Size:       "5Gi",
				AccessMode: "ReadWriteMany",
				MountPath:  "/custom",
			},
		},
		{
			name: "workspace partially specified gets defaults applied by CRD",
			agent: &LanguageAgent{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-agent",
					Namespace: "default",
				},
				Spec: LanguageAgentSpec{
					Image: "test:latest",
					Models: []ModelReference{
						{Name: "test-model"},
					},
					Instructions: "test instructions",
					Workspace: &WorkspaceSpec{
						Size: "20Gi",
						// Other fields will get CRD defaults
					},
				},
			},
			expected: &WorkspaceSpec{
				Size: "20Gi",
				// enabled, accessMode, mountPath would be set by CRD defaults
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Call the Default method via the webhook handler (nil client: Default makes no API calls)
			_ = (&LanguageAgentWebhook{}).Default(context.Background(), tt.agent)

			// Check workspace was set appropriately
			if tt.agent.Spec.Workspace == nil {
				t.Errorf("Expected workspace to be set, got nil")
				return
			}

			// For the first test case, verify all fields
			if tt.name == "workspace defaults to enabled when nil" {
				gotEnabled := tt.agent.Spec.Workspace.Enabled != nil && *tt.agent.Spec.Workspace.Enabled
				wantEnabled := tt.expected.Enabled != nil && *tt.expected.Enabled
				if gotEnabled != wantEnabled {
					t.Errorf("Expected Enabled=%v, got %v", wantEnabled, gotEnabled)
				}
				if tt.agent.Spec.Workspace.Size != tt.expected.Size {
					t.Errorf("Expected Size=%s, got %s", tt.expected.Size, tt.agent.Spec.Workspace.Size)
				}
				if tt.agent.Spec.Workspace.AccessMode != tt.expected.AccessMode {
					t.Errorf("Expected AccessMode=%s, got %s", tt.expected.AccessMode, tt.agent.Spec.Workspace.AccessMode)
				}
				if tt.agent.Spec.Workspace.MountPath != tt.expected.MountPath {
					t.Errorf("Expected MountPath=%s, got %s", tt.expected.MountPath, tt.agent.Spec.Workspace.MountPath)
				}
			}

			// For the second test case, verify values weren't overridden
			if tt.name == "workspace not overridden when explicitly set" {
				gotEnabled := tt.agent.Spec.Workspace.Enabled != nil && *tt.agent.Spec.Workspace.Enabled
				wantEnabled := tt.expected.Enabled != nil && *tt.expected.Enabled
				if gotEnabled != wantEnabled {
					t.Errorf("Expected Enabled=%v, got %v", wantEnabled, gotEnabled)
				}
				if tt.agent.Spec.Workspace.Size != tt.expected.Size {
					t.Errorf("Expected Size=%s, got %s", tt.expected.Size, tt.agent.Spec.Workspace.Size)
				}
				if tt.agent.Spec.Workspace.AccessMode != tt.expected.AccessMode {
					t.Errorf("Expected AccessMode=%s, got %s", tt.expected.AccessMode, tt.agent.Spec.Workspace.AccessMode)
				}
				if tt.agent.Spec.Workspace.MountPath != tt.expected.MountPath {
					t.Errorf("Expected MountPath=%s, got %s", tt.expected.MountPath, tt.agent.Spec.Workspace.MountPath)
				}
			}

			// For the third test case, verify workspace wasn't replaced
			if tt.name == "workspace partially specified gets defaults applied by CRD" {
				if tt.agent.Spec.Workspace.Size != tt.expected.Size {
					t.Errorf("Expected Size=%s, got %s", tt.expected.Size, tt.agent.Spec.Workspace.Size)
				}
			}
		})
	}
}

func TestLanguageAgentValidateCreate(t *testing.T) {
	tests := []struct {
		name      string
		agent     *LanguageAgent
		expectErr bool
	}{
		{
			name: "valid agent",
			agent: &LanguageAgent{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-agent",
					Namespace: "default",
				},
				Spec: LanguageAgentSpec{
					Image: "test:latest",
					Models: []ModelReference{
						{Name: "test-model"},
					},
					Instructions: "test instructions",
				},
			},
			expectErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.agent.validateSpec()
			if (err != nil) != tt.expectErr {
				t.Errorf("validateSpec() error = %v, expectErr %v", err, tt.expectErr)
			}
		})
	}
}

func TestLanguageAgentValidateWorkspaceSize(t *testing.T) {
	tests := []struct {
		name      string
		size      string
		expectErr bool
		errMsg    string
	}{
		{
			name:      "valid size - integers",
			size:      "10Gi",
			expectErr: false,
		},
		{
			name:      "valid size - decimals",
			size:      "1.5Ti",
			expectErr: false,
		},
		{
			name:      "valid size - small unit",
			size:      "500Mi",
			expectErr: false,
		},
		{
			name:      "valid size - bytes",
			size:      "1024",
			expectErr: false,
		},
		{
			name:      "valid size - millicores format",
			size:      "100m",
			expectErr: false,
		},
		{
			name:      "invalid format - letters",
			size:      "abc",
			expectErr: true,
			errMsg:    "invalid format",
		},
		{
			name:      "invalid format - mixed",
			size:      "10XYZ",
			expectErr: true,
			errMsg:    "invalid format",
		},
		{
			name:      "zero size",
			size:      "0Gi",
			expectErr: true,
			errMsg:    "cannot be zero",
		},
		{
			name:      "zero bytes",
			size:      "0",
			expectErr: true,
			errMsg:    "cannot be zero",
		},
		{
			name:      "negative size",
			size:      "-5Gi",
			expectErr: true,
			errMsg:    "cannot be negative",
		},
		{
			name:      "empty string",
			size:      "",
			expectErr: true,
			errMsg:    "cannot be empty",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateWorkspaceSize(tt.size)

			if (err != nil) != tt.expectErr {
				t.Errorf("validateWorkspaceSize(%q) error = %v, expectErr %v", tt.size, err, tt.expectErr)
				return
			}

			if tt.expectErr && err != nil && tt.errMsg != "" {
				if !contains(err.Error(), tt.errMsg) {
					t.Errorf("validateWorkspaceSize(%q) error = %v, expected to contain %q", tt.size, err.Error(), tt.errMsg)
				}
			}
		})
	}
}

func TestLanguageAgentValidateCreateWithWorkspace(t *testing.T) {
	tests := []struct {
		name      string
		agent     *LanguageAgent
		expectErr bool
		errMsg    string
	}{
		{
			name: "valid workspace size",
			agent: &LanguageAgent{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-agent",
					Namespace: "default",
				},
				Spec: LanguageAgentSpec{
					Image:        "test:latest",
					Models:       []ModelReference{{Name: "test-model"}},
					Instructions: "test instructions",
					Workspace: &WorkspaceSpec{
						Size: "10Gi",
					},
				},
			},
			expectErr: false,
		},
		{
			name: "valid decimal workspace size",
			agent: &LanguageAgent{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-agent",
					Namespace: "default",
				},
				Spec: LanguageAgentSpec{
					Image:        "test:latest",
					Models:       []ModelReference{{Name: "test-model"}},
					Instructions: "test instructions",
					Workspace: &WorkspaceSpec{
						Size: "1.5Ti",
					},
				},
			},
			expectErr: false,
		},
		{
			name: "invalid workspace size - zero",
			agent: &LanguageAgent{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-agent",
					Namespace: "default",
				},
				Spec: LanguageAgentSpec{
					Image:        "test:latest",
					Models:       []ModelReference{{Name: "test-model"}},
					Instructions: "test instructions",
					Workspace: &WorkspaceSpec{
						Enabled: boolPtr(true),
						Size:    "0Gi",
					},
				},
			},
			expectErr: true,
			errMsg:    "workspace.size",
		},
		{
			name: "invalid workspace size - format",
			agent: &LanguageAgent{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-agent",
					Namespace: "default",
				},
				Spec: LanguageAgentSpec{
					Image:        "test:latest",
					Models:       []ModelReference{{Name: "test-model"}},
					Instructions: "test instructions",
					Workspace: &WorkspaceSpec{
						Enabled: boolPtr(true),
						Size:    "invalid",
					},
				},
			},
			expectErr: true,
			errMsg:    "workspace.size",
		},
		{
			name: "invalid workspace size - empty string",
			agent: &LanguageAgent{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-agent",
					Namespace: "default",
				},
				Spec: LanguageAgentSpec{
					Image:        "test:latest",
					Models:       []ModelReference{{Name: "test-model"}},
					Instructions: "test instructions",
					Workspace: &WorkspaceSpec{
						Enabled: boolPtr(true),
						Size:    "",
					},
				},
			},
			expectErr: true,
			errMsg:    "workspace.size",
		},
		{
			name: "workspace disabled - no size validation",
			agent: &LanguageAgent{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-agent",
					Namespace: "default",
				},
				Spec: LanguageAgentSpec{
					Image:        "test:latest",
					Models:       []ModelReference{{Name: "test-model"}},
					Instructions: "test instructions",
					Workspace: &WorkspaceSpec{
						Enabled: boolPtr(false),
						Size:    "", // Empty size should be fine when workspace is disabled
					},
				},
			},
			expectErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.agent.validateSpec()

			if (err != nil) != tt.expectErr {
				t.Errorf("validateSpec() error = %v, expectErr %v", err, tt.expectErr)
				return
			}

			if tt.expectErr && err != nil && tt.errMsg != "" {
				if !contains(err.Error(), tt.errMsg) {
					t.Errorf("validateSpec() error = %v, expected to contain %q", err.Error(), tt.errMsg)
				}
			}
		})
	}
}

// Helper function to check if a string contains a substring
func contains(s, substr string) bool {
	return len(substr) == 0 || (len(s) >= len(substr) && func() bool {
		for i := 0; i <= len(s)-len(substr); i++ {
			if s[i:i+len(substr)] == substr {
				return true
			}
		}
		return false
	}())
}

func intPtr(i int32) *int32 {
	return &i
}

func TestLanguageAgentSecurityWarnings(t *testing.T) {
	const ns = "default"
	ctx := context.Background()
	wh := &LanguageAgentWebhook{Client: makeClusterClient(t, ns)}

	zeroQty := resource.MustParse("0")

	baseAgent := func(image string) *LanguageAgent {
		return &LanguageAgent{
			ObjectMeta: metav1.ObjectMeta{Name: "agent", Namespace: ns},
			Spec: LanguageAgentSpec{
				Image:        image,
				Instructions: "test",
			},
		}
	}

	tests := []struct {
		name         string
		agent        *LanguageAgent
		wantWarnings []string
		wantErr      bool
	}{
		{
			name:         "latest tag without digest warns",
			agent:        baseAgent("myimage:latest"),
			wantWarnings: []string{"spec.image: using ':latest' tag"},
		},
		{
			name:         "digest-pinned latest suppresses warning",
			agent:        baseAgent("myimage:latest@sha256:abc123"),
			wantWarnings: nil,
		},
		{
			name:         "non-latest tag no warning",
			agent:        baseAgent("myimage:v1.2.3"),
			wantWarnings: nil,
		},
		{
			name: "cpu limit zero warns",
			agent: func() *LanguageAgent {
				a := baseAgent("myimage:v1.0.0")
				a.Spec.Deployment.Resources.Limits = corev1.ResourceList{
					corev1.ResourceCPU: zeroQty,
				}
				return a
			}(),
			wantWarnings: []string{"spec.deployment.resources.limits.cpu"},
		},
		{
			name: "memory limit zero warns",
			agent: func() *LanguageAgent {
				a := baseAgent("myimage:v1.0.0")
				a.Spec.Deployment.Resources.Limits = corev1.ResourceList{
					corev1.ResourceMemory: zeroQty,
				}
				return a
			}(),
			wantWarnings: []string{"spec.deployment.resources.limits.memory"},
		},
		{
			name: "both limits zero gives two warnings",
			agent: func() *LanguageAgent {
				a := baseAgent("myimage:v1.0.0")
				a.Spec.Deployment.Resources.Limits = corev1.ResourceList{
					corev1.ResourceCPU:    zeroQty,
					corev1.ResourceMemory: zeroQty,
				}
				return a
			}(),
			wantWarnings: []string{
				"spec.deployment.resources.limits.cpu",
				"spec.deployment.resources.limits.memory",
			},
		},
		{
			name: "non-zero limits no warning",
			agent: func() *LanguageAgent {
				a := baseAgent("myimage:v1.0.0")
				a.Spec.Deployment.Resources.Limits = corev1.ResourceList{
					corev1.ResourceCPU:    resource.MustParse("500m"),
					corev1.ResourceMemory: resource.MustParse("512Mi"),
				}
				return a
			}(),
			wantWarnings: nil,
		},
		{
			name: "runAsNonRoot false warns",
			agent: func() *LanguageAgent {
				a := baseAgent("myimage:v1.0.0")
				a.Spec.Deployment.SecurityContext = &corev1.PodSecurityContext{
					RunAsNonRoot: boolPtr(false),
				}
				return a
			}(),
			wantWarnings: []string{"spec.deployment.securityContext.runAsNonRoot"},
		},
		{
			name: "runAsNonRoot true no warning",
			agent: func() *LanguageAgent {
				a := baseAgent("myimage:v1.0.0")
				a.Spec.Deployment.SecurityContext = &corev1.PodSecurityContext{
					RunAsNonRoot: boolPtr(true),
				}
				return a
			}(),
			wantWarnings: nil,
		},
		{
			name:         "no security context no warning",
			agent:        baseAgent("myimage:v1.0.0"),
			wantWarnings: nil,
		},
		{
			name: "all three dangerous configs give three warnings and no error",
			agent: func() *LanguageAgent {
				a := baseAgent("myimage:latest")
				a.Spec.Deployment.Resources.Limits = corev1.ResourceList{
					corev1.ResourceCPU: zeroQty,
				}
				a.Spec.Deployment.SecurityContext = &corev1.PodSecurityContext{
					RunAsNonRoot: boolPtr(false),
				}
				return a
			}(),
			wantWarnings: []string{
				"spec.image: using ':latest' tag",
				"spec.deployment.resources.limits.cpu",
				"spec.deployment.securityContext.runAsNonRoot",
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			warns, err := wh.ValidateCreate(ctx, tt.agent)
			if (err != nil) != tt.wantErr {
				t.Fatalf("ValidateCreate() error = %v, wantErr %v", err, tt.wantErr)
			}
			if len(warns) != len(tt.wantWarnings) {
				t.Fatalf("ValidateCreate() warnings = %v, want %d warnings containing %v", warns, len(tt.wantWarnings), tt.wantWarnings)
			}
			for i, want := range tt.wantWarnings {
				if !contains(warns[i], want) {
					t.Errorf("warns[%d] = %q, want it to contain %q", i, warns[i], want)
				}
			}
		})
	}
}

func TestLanguageAgentValidateUpdate(t *testing.T) {
	const ns = "default"
	ctx := context.Background()

	makeAgent := func(size string) *LanguageAgent {
		return &LanguageAgent{
			ObjectMeta: metav1.ObjectMeta{Name: "agent", Namespace: ns},
			Spec: LanguageAgentSpec{
				Image:        "test:latest",
				Instructions: "test",
				Workspace:    &WorkspaceSpec{Size: size},
			},
		}
	}

	tests := []struct {
		name      string
		oldAgent  *LanguageAgent
		newAgent  *LanguageAgent
		expectErr bool
		errMsg    string
	}{
		{
			name:      "decrease rejected",
			oldAgent:  makeAgent("20Gi"),
			newAgent:  makeAgent("5Gi"),
			expectErr: true,
			errMsg:    "cannot decrease storage size",
		},
		{
			name:      "same size allowed",
			oldAgent:  makeAgent("10Gi"),
			newAgent:  makeAgent("10Gi"),
			expectErr: false,
		},
		{
			name:      "increase allowed",
			oldAgent:  makeAgent("10Gi"),
			newAgent:  makeAgent("20Gi"),
			expectErr: false,
		},
		{
			name: "old workspace nil - allowed",
			oldAgent: &LanguageAgent{
				ObjectMeta: metav1.ObjectMeta{Name: "agent", Namespace: ns},
				Spec:       LanguageAgentSpec{Image: "test:latest", Instructions: "test"},
			},
			newAgent:  makeAgent("10Gi"),
			expectErr: false,
		},
		{
			name:     "new workspace nil - allowed (validateSpec handles format)",
			oldAgent: makeAgent("10Gi"),
			newAgent: &LanguageAgent{
				ObjectMeta: metav1.ObjectMeta{Name: "agent", Namespace: ns},
				Spec:       LanguageAgentSpec{Image: "test:latest", Instructions: "test"},
			},
			expectErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			webhook := &LanguageAgentWebhook{Client: makeClusterClient(t, ns)}
			_, err := webhook.ValidateUpdate(ctx, tt.newAgent, tt.oldAgent)
			if (err != nil) != tt.expectErr {
				t.Errorf("ValidateUpdate() error = %v, expectErr %v", err, tt.expectErr)
				return
			}
			if tt.expectErr && err != nil && !contains(err.Error(), tt.errMsg) {
				t.Errorf("ValidateUpdate() error = %q, want it to contain %q", err.Error(), tt.errMsg)
			}
		})
	}
}
