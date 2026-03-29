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
		models    []ModelReference
		expectErr bool
		errMsg    string
	}{
		{
			name:      "no model references is valid (models is optional)",
			models:    []ModelReference{},
			expectErr: false,
		},
		{
			name: "valid model reference should pass",
			models: []ModelReference{
				{Name: "test-model", Role: "primary"},
			},
			expectErr: false,
		},
		{
			name: "empty model name should fail",
			models: []ModelReference{
				{Name: "", Role: "primary"},
			},
			expectErr: true,
			errMsg:    "name cannot be empty",
		},
		{
			name: "no primary model should fail",
			models: []ModelReference{
				{Name: "test-model", Role: "fallback"},
			},
			expectErr: true,
			errMsg:    "at least one model must have role 'primary'",
		},
		{
			name: "negative priority should fail",
			models: []ModelReference{
				{Name: "test-model", Role: "primary", Priority: intPtr(-1)},
			},
			expectErr: true,
			errMsg:    "priority cannot be negative",
		},
		{
			name: "multiple models with primary should pass",
			models: []ModelReference{
				{Name: "model1", Role: "primary"},
				{Name: "model2", Role: "fallback"},
			},
			expectErr: false,
		},
		{
			name: "model with default role is primary",
			models: []ModelReference{
				{Name: "model1"}, // default role should be treated as primary
			},
			expectErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			agent := &LanguageAgent{
				Spec: LanguageAgentSpec{
					Models: tt.models,
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
