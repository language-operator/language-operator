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

func TestValidateExecution(t *testing.T) {
	i64 := func(v int64) *int64 { return &v }
	i32 := func(v int32) *int32 { return &v }

	tests := []struct {
		name   string
		exec   ExecutionSpec
		ports  []AgentPort
		errMsg string // empty means the spec must be accepted
	}{
		{name: "empty defaults to service", exec: ExecutionSpec{}},
		{name: "explicit service", exec: ExecutionSpec{Mode: ExecutionModeService}},
		{name: "task without schedule is manual-only", exec: ExecutionSpec{Mode: ExecutionModeTask}},
		{
			name: "scheduled task",
			exec: ExecutionSpec{Mode: ExecutionModeTask, Schedule: "*/5 * * * *", Timezone: "UTC"},
		},
		{
			name:   "schedule on a service agent",
			exec:   ExecutionSpec{Mode: ExecutionModeService, Schedule: "@daily"},
			errMsg: "spec.execution.schedule",
		},
		{
			name:   "deadline on a service agent",
			exec:   ExecutionSpec{ActiveDeadlineSeconds: i64(60)},
			errMsg: "spec.execution.activeDeadlineSeconds",
		},
		{
			name:   "ttl on a service agent",
			exec:   ExecutionSpec{TTLSecondsAfterFinished: i32(60)},
			errMsg: "spec.execution.ttlSecondsAfterFinished",
		},
		{
			name:   "retry limit on a service agent",
			exec:   ExecutionSpec{RetryLimit: i32(3)},
			errMsg: "spec.execution.retryLimit",
		},
		{
			name:   "timezone without a schedule",
			exec:   ExecutionSpec{Mode: ExecutionModeTask, Timezone: "UTC"},
			errMsg: "spec.execution.timezone",
		},
		{
			name:   "unknown timezone",
			exec:   ExecutionSpec{Mode: ExecutionModeTask, Schedule: "@daily", Timezone: "Mars/Olympus"},
			errMsg: "not a valid IANA timezone",
		},
		{
			name:   "ports on a task agent",
			exec:   ExecutionSpec{Mode: ExecutionModeTask},
			ports:  []AgentPort{{Name: "http", Port: 8080}},
			errMsg: "spec.ports",
		},
		{
			name:  "ports on a service agent",
			exec:  ExecutionSpec{Mode: ExecutionModeService},
			ports: []AgentPort{{Name: "http", Port: 8080}},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateExecution(&tt.exec, tt.ports)
			if tt.errMsg == "" {
				if err != nil {
					t.Errorf("validateExecution() unexpected error: %v", err)
				}
				return
			}
			if err == nil {
				t.Fatalf("validateExecution() expected error containing %q, got nil", tt.errMsg)
			}
			if !strings.Contains(err.Error(), tt.errMsg) {
				t.Errorf("validateExecution() error = %v, expected to contain %q", err, tt.errMsg)
			}
		})
	}
}

func TestValidateCronSchedule(t *testing.T) {
	valid := []string{
		"* * * * *",
		"*/5 * * * *",
		"0 3 * * *",
		"30 2 1 * *",
		"0 0 1-15 JAN-JUN MON",
		"0,15,30,45 * * * *",
		"0 9-17/2 * * mon-fri",
		"0 0 * * 7", // day-of-week 7 is Sunday
		"@daily", "@hourly", "@midnight", "@yearly", "@annually", "@monthly", "@weekly",
		"@every 1h30m",
	}
	for _, s := range valid {
		t.Run("valid/"+s, func(t *testing.T) {
			if err := validateCronSchedule(s); err != nil {
				t.Errorf("validateCronSchedule(%q) unexpected error: %v", s, err)
			}
		})
	}

	invalid := map[string]string{
		"":              "cannot be empty",
		"* * * *":       "must have 5 fields",
		"* * * * * *":   "must have 5 fields",
		"60 * * * *":    "out of range",
		"* 24 * * *":    "out of range",
		"* * 0 * *":     "out of range",
		"* * * 13 *":    "out of range",
		"* * * * 8":     "out of range",
		"*/0 * * * *":   "must be a positive integer",
		"*/abc * * * *": "must be a positive integer",
		"foo * * * *":   "is not a number",
		"@nope":         "not a recognized cron macro",
		"@every nope":   "invalid @every duration",
	}
	for s, want := range invalid {
		t.Run("invalid/"+s, func(t *testing.T) {
			err := validateCronSchedule(s)
			if err == nil {
				t.Fatalf("validateCronSchedule(%q) expected error containing %q, got nil", s, want)
			}
			if !strings.Contains(err.Error(), want) {
				t.Errorf("validateCronSchedule(%q) error = %v, expected to contain %q", s, err, want)
			}
		})
	}
}

func TestValidateSpec_RejectsReplicasAndAutoscaling(t *testing.T) {
	// Agents run as Argo Workflows, which have no replicas and no scale
	// subresource. Silently ignoring these would mislead the user.
	three := int32(3)
	agent := &LanguageAgent{Spec: LanguageAgentSpec{
		Image:      "ghcr.io/language-operator/agent:latest",
		Deployment: DeploymentSpec{Replicas: &three},
	}}
	err := agent.validateSpec()
	if err == nil || !strings.Contains(err.Error(), "spec.deployment.replicas") {
		t.Errorf("expected spec.deployment.replicas rejection, got %v", err)
	}

	agent = &LanguageAgent{Spec: LanguageAgentSpec{
		Image:      "ghcr.io/language-operator/agent:latest",
		Deployment: DeploymentSpec{Autoscaling: &AutoscalingSpec{MaxReplicas: three}},
	}}
	err = agent.validateSpec()
	if err == nil || !strings.Contains(err.Error(), "spec.deployment.autoscaling") {
		t.Errorf("expected spec.deployment.autoscaling rejection, got %v", err)
	}
}
