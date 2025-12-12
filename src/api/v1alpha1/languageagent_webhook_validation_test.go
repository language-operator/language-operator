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
	"strings"
	"testing"
)

func TestLanguageAgentValidateModelReferences(t *testing.T) {
	tests := []struct {
		name      string
		modelRefs []ModelReference
		expectErr bool
		errMsg    string
	}{
		{
			name:      "no model references should fail",
			modelRefs: []ModelReference{},
			expectErr: true,
			errMsg:    "at least one model reference is required",
		},
		{
			name: "valid model reference should pass",
			modelRefs: []ModelReference{
				{Name: "test-model", Role: "primary"},
			},
			expectErr: false,
		},
		{
			name: "empty model name should fail",
			modelRefs: []ModelReference{
				{Name: "", Role: "primary"},
			},
			expectErr: true,
			errMsg:    "name cannot be empty",
		},
		{
			name: "no primary model should fail",
			modelRefs: []ModelReference{
				{Name: "test-model", Role: "fallback"},
			},
			expectErr: true,
			errMsg:    "at least one model must have role 'primary'",
		},
		{
			name: "negative priority should fail",
			modelRefs: []ModelReference{
				{Name: "test-model", Role: "primary", Priority: intPtr(-1)},
			},
			expectErr: true,
			errMsg:    "priority cannot be negative",
		},
		{
			name: "multiple models with primary should pass",
			modelRefs: []ModelReference{
				{Name: "model1", Role: "primary"},
				{Name: "model2", Role: "fallback"},
			},
			expectErr: false,
		},
		{
			name: "model with default role is primary",
			modelRefs: []ModelReference{
				{Name: "model1"}, // default role should be treated as primary
			},
			expectErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			agent := &LanguageAgent{
				Spec: LanguageAgentSpec{
					ModelRefs: tt.modelRefs,
				},
			}

			err := agent.validateModelReferences()

			if (err != nil) != tt.expectErr {
				t.Errorf("validateModelReferences() error = %v, expectErr %v", err, tt.expectErr)
				return
			}

			if tt.expectErr && err != nil && tt.errMsg != "" {
				if !strings.Contains(err.Error(), tt.errMsg) {
					t.Errorf("validateModelReferences() error = %v, expected to contain %q", err.Error(), tt.errMsg)
				}
			}
		})
	}
}

func TestLanguageAgentValidateExecutionMode(t *testing.T) {
	tests := []struct {
		name          string
		executionMode string
		schedule      string
		eventTriggers []EventTriggerSpec
		expectErr     bool
		errMsg        string
	}{
		{
			name:          "scheduled mode requires schedule",
			executionMode: "scheduled",
			schedule:      "",
			expectErr:     true,
			errMsg:        "schedule is required when executionMode is 'scheduled'",
		},
		{
			name:          "scheduled mode with schedule should pass",
			executionMode: "scheduled",
			schedule:      "0 9 * * *",
			expectErr:     false,
		},
		{
			name:          "event-driven mode requires event triggers",
			executionMode: "event-driven",
			eventTriggers: []EventTriggerSpec{},
			expectErr:     true,
			errMsg:        "eventTriggers is required when executionMode is 'event-driven'",
		},
		{
			name:          "event-driven mode with triggers should pass",
			executionMode: "event-driven",
			eventTriggers: []EventTriggerSpec{
				{Type: "webhook"},
			},
			expectErr: false,
		},
		{
			name:          "autonomous mode should pass",
			executionMode: "autonomous",
			expectErr:     false,
		},
		{
			name:          "interactive mode should pass",
			executionMode: "interactive",
			expectErr:     false,
		},
		{
			name:          "empty execution mode should pass (default autonomous)",
			executionMode: "",
			expectErr:     false,
		},
		{
			name:          "invalid execution mode should fail",
			executionMode: "invalid",
			expectErr:     true,
			errMsg:        "invalid executionMode",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			agent := &LanguageAgent{
				Spec: LanguageAgentSpec{
					ExecutionMode: tt.executionMode,
					Schedule:      tt.schedule,
					EventTriggers: tt.eventTriggers,
				},
			}

			err := agent.validateExecutionMode()

			if (err != nil) != tt.expectErr {
				t.Errorf("validateExecutionMode() error = %v, expectErr %v", err, tt.expectErr)
				return
			}

			if tt.expectErr && err != nil && tt.errMsg != "" {
				if !strings.Contains(err.Error(), tt.errMsg) {
					t.Errorf("validateExecutionMode() error = %v, expected to contain %q", err.Error(), tt.errMsg)
				}
			}
		})
	}
}
