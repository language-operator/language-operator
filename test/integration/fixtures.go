package integration

import (
	langopv1alpha1 "github.com/language-operator/language-operator/api/v1alpha1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// TestScenario defines a test case for agent synthesis
type TestScenario struct {
	Name             string
	Instructions     string
	ExecutionMode    string
	ExpectedSchedule string
	ExpectedTools    []string
	ShouldContain    []string
	ShouldFail       bool
}

// Common test scenarios that can be reused across tests
var TestScenarios = []TestScenario{
	{
		Name:             "daily spreadsheet review",
		Instructions:     "review my spreadsheet at 4pm daily and email me errors",
		ExecutionMode:    "scheduled",
		ExpectedSchedule: "0 16 * * *",
		ExpectedTools:    []string{"google-sheets", "email"},
		ShouldContain:    []string{"schedule", "agent"},
	},
	{
		Name:             "health check every 5 minutes",
		Instructions:     "check https://api.example.com/health every 5 minutes",
		ExecutionMode:    "scheduled",
		ExpectedSchedule: "*/5 * * * *",
		ExpectedTools:    []string{"web-fetch"},
		ShouldContain:    []string{"schedule", "agent"},
	},
	{
		Name:             "hourly data sync",
		Instructions:     "sync data from API every hour",
		ExecutionMode:    "scheduled",
		ExpectedSchedule: "0 * * * *",
		ExpectedTools:    []string{"web-fetch"},
		ShouldContain:    []string{"schedule", "agent"},
	},
	{
		Name:             "daily morning report",
		Instructions:     "send me a report at 9am every day",
		ExecutionMode:    "scheduled",
		ExpectedSchedule: "0 9 * * *",
		ExpectedTools:    []string{"email"},
		ShouldContain:    []string{"schedule", "agent"},
	},
	{
		Name:          "autonomous processing",
		Instructions:  "process tasks autonomously",
		ExecutionMode: "autonomous",
		ShouldContain: []string{"agent"},
	},
	{
		Name:         "empty instructions",
		Instructions: "",
		ShouldFail:   true,
	},
	{
		Name:         "whitespace only",
		Instructions: "   \n\t  ",
		ShouldFail:   true,
	},
}

// NewTestAgent creates a LanguageAgent for testing with sensible defaults
func NewTestAgent(namespace, name string, spec langopv1alpha1.LanguageAgentSpec) *langopv1alpha1.LanguageAgent {
	// Set defaults if not provided
	if spec.Image == "" {
		spec.Image = "ghcr.io/language-operator/agent:latest"
	}
	if len(spec.ModelRefs) == 0 {
		spec.ModelRefs = []langopv1alpha1.ModelReference{
			{Name: "test-model"},
		}
	}
	if spec.ExecutionMode == "" {
		spec.ExecutionMode = "autonomous"
	}

	return &langopv1alpha1.LanguageAgent{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
		},
		Spec: spec,
	}
}

// NewTestModel creates a LanguageModel for testing
func NewTestModel(namespace, name string) *langopv1alpha1.LanguageModel {
	return &langopv1alpha1.LanguageModel{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
		},
		Spec: langopv1alpha1.LanguageModelSpec{
			Provider:  "openai-compatible",
			ModelName: "test-model",
			Endpoint:  "http://test-endpoint",
		},
	}
}
