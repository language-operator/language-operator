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
	"testing"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

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
					ModelRefs: []ModelReference{
						{Name: "test-model"},
					},
					Instructions: "test instructions",
					// Workspace is nil
				},
			},
			expected: &WorkspaceSpec{
				Enabled:    true,
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
					ModelRefs: []ModelReference{
						{Name: "test-model"},
					},
					Instructions: "test instructions",
					Workspace: &WorkspaceSpec{
						Enabled:    false,
						Size:       "5Gi",
						AccessMode: "ReadWriteMany",
						MountPath:  "/custom",
					},
				},
			},
			expected: &WorkspaceSpec{
				Enabled:    false,
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
					ModelRefs: []ModelReference{
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
			// Call the Default method
			tt.agent.Default()

			// Check workspace was set appropriately
			if tt.agent.Spec.Workspace == nil {
				t.Errorf("Expected workspace to be set, got nil")
				return
			}

			// For the first test case, verify all fields
			if tt.name == "workspace defaults to enabled when nil" {
				if tt.agent.Spec.Workspace.Enabled != tt.expected.Enabled {
					t.Errorf("Expected Enabled=%v, got %v", tt.expected.Enabled, tt.agent.Spec.Workspace.Enabled)
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
				if tt.agent.Spec.Workspace.Enabled != tt.expected.Enabled {
					t.Errorf("Expected Enabled=%v, got %v", tt.expected.Enabled, tt.agent.Spec.Workspace.Enabled)
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
					ModelRefs: []ModelReference{
						{Name: "test-model"},
					},
					Instructions: "test instructions",
				},
			},
			expectErr: false,
		},
		{
			name: "missing instructions",
			agent: &LanguageAgent{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-agent",
					Namespace: "default",
				},
				Spec: LanguageAgentSpec{
					Image: "test:latest",
					ModelRefs: []ModelReference{
						{Name: "test-model"},
					},
					Instructions: "",
				},
			},
			expectErr: true,
		},
		{
			name: "negative rate limit",
			agent: &LanguageAgent{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-agent",
					Namespace: "default",
				},
				Spec: LanguageAgentSpec{
					Image: "test:latest",
					ModelRefs: []ModelReference{
						{Name: "test-model"},
					},
					Instructions: "test instructions",
					RateLimits: &AgentRateLimitSpec{
						RequestsPerMinute: intPtr(-1),
					},
				},
			},
			expectErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := tt.agent.ValidateCreate()
			if (err != nil) != tt.expectErr {
				t.Errorf("ValidateCreate() error = %v, expectErr %v", err, tt.expectErr)
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
			errMsg:    "invalid format",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			agent := &LanguageAgent{}
			err := agent.validateWorkspaceSize(tt.size)

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
					ModelRefs:    []ModelReference{{Name: "test-model"}},
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
					ModelRefs:    []ModelReference{{Name: "test-model"}},
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
					ModelRefs:    []ModelReference{{Name: "test-model"}},
					Instructions: "test instructions",
					Workspace: &WorkspaceSpec{
						Size: "0Gi",
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
					ModelRefs:    []ModelReference{{Name: "test-model"}},
					Instructions: "test instructions",
					Workspace: &WorkspaceSpec{
						Size: "invalid",
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
					ModelRefs:    []ModelReference{{Name: "test-model"}},
					Instructions: "test instructions",
					Workspace: &WorkspaceSpec{
						Enabled: false,
						Size:    "", // Empty size should be fine when workspace is disabled
					},
				},
			},
			expectErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := tt.agent.ValidateCreate()

			if (err != nil) != tt.expectErr {
				t.Errorf("ValidateCreate() error = %v, expectErr %v", err, tt.expectErr)
				return
			}

			if tt.expectErr && err != nil && tt.errMsg != "" {
				if !contains(err.Error(), tt.errMsg) {
					t.Errorf("ValidateCreate() error = %v, expected to contain %q", err.Error(), tt.errMsg)
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

func TestLanguageAgentValidateSchedule(t *testing.T) {
	tests := []struct {
		name          string
		executionMode string
		schedule      string
		expectErr     bool
		errMsg        string
	}{
		// Valid standard cron expressions
		{
			name:          "valid hourly cron",
			executionMode: "scheduled",
			schedule:      "0 * * * *",
			expectErr:     false,
		},
		{
			name:          "valid daily cron",
			executionMode: "scheduled",
			schedule:      "0 9 * * *",
			expectErr:     false,
		},
		{
			name:          "valid weekly cron",
			executionMode: "scheduled",
			schedule:      "0 9 * * 1",
			expectErr:     false,
		},
		{
			name:          "valid complex cron",
			executionMode: "scheduled",
			schedule:      "*/15 8-17 * * 1-5",
			expectErr:     false,
		},
		// Valid special cron expressions
		{
			name:          "valid @hourly",
			executionMode: "scheduled",
			schedule:      "@hourly",
			expectErr:     false,
		},
		{
			name:          "valid @daily",
			executionMode: "scheduled",
			schedule:      "@daily",
			expectErr:     false,
		},
		{
			name:          "valid @weekly",
			executionMode: "scheduled",
			schedule:      "@weekly",
			expectErr:     false,
		},
		{
			name:          "valid @monthly",
			executionMode: "scheduled",
			schedule:      "@monthly",
			expectErr:     false,
		},
		{
			name:          "valid @yearly",
			executionMode: "scheduled",
			schedule:      "@yearly",
			expectErr:     false,
		},
		// Invalid cron expressions
		{
			name:          "invalid minute - too high",
			executionMode: "scheduled",
			schedule:      "60 * * * *",
			expectErr:     true,
			errMsg:        "invalid cron expression",
		},
		{
			name:          "invalid hour - too high",
			executionMode: "scheduled",
			schedule:      "0 25 * * *",
			expectErr:     true,
			errMsg:        "invalid cron expression",
		},
		{
			name:          "invalid day - too high",
			executionMode: "scheduled",
			schedule:      "0 0 32 * *",
			expectErr:     true,
			errMsg:        "invalid cron expression",
		},
		{
			name:          "invalid month - too high",
			executionMode: "scheduled",
			schedule:      "0 0 1 13 *",
			expectErr:     true,
			errMsg:        "invalid cron expression",
		},
		{
			name:          "invalid weekday - too high",
			executionMode: "scheduled",
			schedule:      "0 0 * * 8",
			expectErr:     true,
			errMsg:        "invalid cron expression",
		},
		{
			name:          "invalid format - wrong number of fields",
			executionMode: "scheduled",
			schedule:      "0 * *",
			expectErr:     true,
			errMsg:        "invalid cron expression",
		},
		{
			name:          "invalid format - random text",
			executionMode: "scheduled",
			schedule:      "invalid cron syntax",
			expectErr:     true,
			errMsg:        "invalid cron expression",
		},
		{
			name:          "invalid format - division by zero",
			executionMode: "scheduled",
			schedule:      "*/0 * * * *",
			expectErr:     true,
			errMsg:        "invalid cron expression",
		},
		// Schedule requirements based on execution mode
		{
			name:          "scheduled mode requires schedule",
			executionMode: "scheduled",
			schedule:      "",
			expectErr:     true,
			errMsg:        "schedule is required when executionMode is 'scheduled'",
		},
		{
			name:          "autonomous mode allows empty schedule",
			executionMode: "autonomous",
			schedule:      "",
			expectErr:     false,
		},
		{
			name:          "autonomous mode allows valid schedule",
			executionMode: "autonomous",
			schedule:      "0 9 * * *",
			expectErr:     false,
		},
		{
			name:          "autonomous mode rejects invalid schedule",
			executionMode: "autonomous",
			schedule:      "invalid",
			expectErr:     true,
			errMsg:        "invalid cron expression",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			agent := &LanguageAgent{
				Spec: LanguageAgentSpec{
					ExecutionMode: tt.executionMode,
					Schedule:      tt.schedule,
				},
			}

			err := agent.validateSchedule()

			if (err != nil) != tt.expectErr {
				t.Errorf("validateSchedule() error = %v, expectErr %v", err, tt.expectErr)
				return
			}

			if tt.expectErr && err != nil && tt.errMsg != "" {
				if !contains(err.Error(), tt.errMsg) {
					t.Errorf("validateSchedule() error = %v, expected to contain %q", err.Error(), tt.errMsg)
				}
			}
		})
	}
}

func TestLanguageAgentValidateCreateWithSchedule(t *testing.T) {
	tests := []struct {
		name      string
		agent     *LanguageAgent
		expectErr bool
		errMsg    string
	}{
		{
			name: "scheduled agent with valid schedule",
			agent: &LanguageAgent{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-agent",
					Namespace: "default",
				},
				Spec: LanguageAgentSpec{
					Image:         "test:latest",
					ModelRefs:     []ModelReference{{Name: "test-model"}},
					Instructions:  "test instructions",
					ExecutionMode: "scheduled",
					Schedule:      "0 9 * * *",
				},
			},
			expectErr: false,
		},
		{
			name: "scheduled agent with @daily syntax",
			agent: &LanguageAgent{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-agent",
					Namespace: "default",
				},
				Spec: LanguageAgentSpec{
					Image:         "test:latest",
					ModelRefs:     []ModelReference{{Name: "test-model"}},
					Instructions:  "test instructions",
					ExecutionMode: "scheduled",
					Schedule:      "@daily",
				},
			},
			expectErr: false,
		},
		{
			name: "scheduled agent missing schedule",
			agent: &LanguageAgent{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-agent",
					Namespace: "default",
				},
				Spec: LanguageAgentSpec{
					Image:         "test:latest",
					ModelRefs:     []ModelReference{{Name: "test-model"}},
					Instructions:  "test instructions",
					ExecutionMode: "scheduled",
					Schedule:      "",
				},
			},
			expectErr: true,
			errMsg:    "schedule is required when executionMode is 'scheduled'",
		},
		{
			name: "scheduled agent with invalid schedule",
			agent: &LanguageAgent{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-agent",
					Namespace: "default",
				},
				Spec: LanguageAgentSpec{
					Image:         "test:latest",
					ModelRefs:     []ModelReference{{Name: "test-model"}},
					Instructions:  "test instructions",
					ExecutionMode: "scheduled",
					Schedule:      "60 * * * *",
				},
			},
			expectErr: true,
			errMsg:    "spec.schedule",
		},
		{
			name: "autonomous agent without schedule",
			agent: &LanguageAgent{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-agent",
					Namespace: "default",
				},
				Spec: LanguageAgentSpec{
					Image:         "test:latest",
					ModelRefs:     []ModelReference{{Name: "test-model"}},
					Instructions:  "test instructions",
					ExecutionMode: "autonomous",
					Schedule:      "",
				},
			},
			expectErr: false,
		},
		{
			name: "autonomous agent with valid schedule",
			agent: &LanguageAgent{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-agent",
					Namespace: "default",
				},
				Spec: LanguageAgentSpec{
					Image:         "test:latest",
					ModelRefs:     []ModelReference{{Name: "test-model"}},
					Instructions:  "test instructions",
					ExecutionMode: "autonomous",
					Schedule:      "0 9 * * *",
				},
			},
			expectErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := tt.agent.ValidateCreate()

			if (err != nil) != tt.expectErr {
				t.Errorf("ValidateCreate() error = %v, expectErr %v", err, tt.expectErr)
				return
			}

			if tt.expectErr && err != nil && tt.errMsg != "" {
				if !contains(err.Error(), tt.errMsg) {
					t.Errorf("ValidateCreate() error = %v, expected to contain %q", err.Error(), tt.errMsg)
				}
			}
		})
	}
}
