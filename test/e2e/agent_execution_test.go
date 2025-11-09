package e2e

import (
	"fmt"
	"strings"
	"testing"

	langopv1alpha1 "github.com/based/language-operator/api/v1alpha1"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// TestAgentExecution tests the full lifecycle of a LanguageAgent
func TestAgentExecution(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping e2e test in short mode")
	}

	// Setup test environment
	env := SetupTestEnvironment(t)
	defer env.Teardown(t)

	// Start mock LLM service
	mockLLM := NewMockLLMService(t)
	defer mockLLM.Close()

	// Create mock chat model and set synthesizer
	mockChatModel := NewMockChatModel(mockLLM)
	env.SetSynthesizer(t, mockChatModel)

	// Create test namespace
	namespace := "test-agent-execution"
	env.CreateNamespace(t, namespace)
	defer env.DeleteNamespace(t, namespace)

	// Create test model (required by agent)
	env.CreateTestModel(t, namespace, "test-model")

	// Create LanguageAgent
	agent := NewTestLanguageAgent(namespace, "test-agent", langopv1alpha1.LanguageAgentSpec{
		Instructions:  "Check the health of https://api.example.com every 5 minutes",
		ExecutionMode: "scheduled",
	})

	env.CreateLanguageAgent(t, agent)

	// Wait for synthesis to complete
	t.Log("Waiting for synthesis to complete...")
	err := env.WaitForCondition(t, namespace, "test-agent", "Synthesized", metav1.ConditionTrue)
	require.NoError(t, err, "Synthesis should complete successfully")

	// Verify ConfigMap was created
	t.Log("Verifying ConfigMap creation...")
	err = env.WaitForConfigMap(t, namespace, "test-agent-code")
	require.NoError(t, err, "ConfigMap should be created")

	cm := env.GetConfigMap(t, namespace, "test-agent-code")
	code, ok := cm.Data["agent.rb"]
	assert.True(t, ok, "ConfigMap should contain agent.rb")
	assert.NotEmpty(t, code, "Agent code should not be empty")

	// Verify code quality
	assert.Contains(t, code, "agent", "Code should contain agent definition")
	assert.Contains(t, code, "schedule", "Code should contain schedule")
	assert.Contains(t, code, "*/5 * * * *", "Code should contain correct schedule")

	// Note: In envtest, we cannot verify actual Deployment/Pod creation
	// as envtest only runs the API server, not the full cluster control plane.
	// We verify the synthesis and ConfigMap creation which is what the controller manages.

	// Verify agent status
	updatedAgent := env.GetLanguageAgent(t, namespace, "test-agent")
	assert.NotNil(t, updatedAgent.Status.Conditions, "Status conditions should be set")

	// Check for Synthesized condition
	hasSynthesized := false
	for _, cond := range updatedAgent.Status.Conditions {
		if cond.Type == "Synthesized" && cond.Status == metav1.ConditionTrue {
			hasSynthesized = true
			break
		}
	}
	assert.True(t, hasSynthesized, "Agent should have Synthesized condition set to True")

	t.Logf("Generated code:\n%s", code)
}

// TestAgentWithWorkspace tests agent execution with persistent workspace
func TestAgentWithWorkspace(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping e2e test in short mode")
	}

	// Setup test environment
	env := SetupTestEnvironment(t)
	defer env.Teardown(t)

	// Start mock LLM service
	mockLLM := NewMockLLMService(t)
	defer mockLLM.Close()

	// Create mock chat model and set synthesizer
	mockChatModel := NewMockChatModel(mockLLM)
	env.SetSynthesizer(t, mockChatModel)

	// Create test namespace
	namespace := "test-agent-workspace"
	env.CreateNamespace(t, namespace)
	defer env.DeleteNamespace(t, namespace)

	// Create test model (required by agent)
	env.CreateTestModel(t, namespace, "test-model")

	// Create LanguageAgent with workspace
	agent := NewTestLanguageAgent(namespace, "test-agent-ws", langopv1alpha1.LanguageAgentSpec{
		Instructions:  "Process data and save results to workspace",
		ExecutionMode: "autonomous",
		Workspace: &langopv1alpha1.WorkspaceSpec{
			Enabled: true,
			Size:    "1Gi",
		},
	})

	env.CreateLanguageAgent(t, agent)

	// Wait for synthesis
	err := env.WaitForCondition(t, namespace, "test-agent-ws", "Synthesized", metav1.ConditionTrue)
	require.NoError(t, err, "Synthesis should complete")

	// Verify PVC was created (workspace enabled)
	// Note: In a real cluster, we would check for PVC creation
	// In envtest, we just verify the agent spec was accepted

	updatedAgent := env.GetLanguageAgent(t, namespace, "test-agent-ws")
	assert.NotNil(t, updatedAgent.Spec.Workspace, "Workspace config should be present")
	assert.True(t, updatedAgent.Spec.Workspace.Enabled, "Workspace should be enabled")
}

// TestAgentScheduleModes tests different execution modes
func TestAgentScheduleModes(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping e2e test in short mode")
	}

	// Setup test environment
	env := SetupTestEnvironment(t)
	defer env.Teardown(t)

	// Start mock LLM service
	mockLLM := NewMockLLMService(t)
	defer mockLLM.Close()

	// Create mock chat model and set synthesizer
	mockChatModel := NewMockChatModel(mockLLM)
	env.SetSynthesizer(t, mockChatModel)

	testCases := []struct {
		name         string
		mode         string
		instructions string
		checkSchedule bool
	}{
		{
			name:         "scheduled mode",
			mode:         "scheduled",
			instructions: "Run daily at 9am",
			checkSchedule: true,
		},
		{
			name:         "autonomous mode",
			mode:         "autonomous",
			instructions: "Process tasks autonomously",
			checkSchedule: false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Create test namespace
			namespace := fmt.Sprintf("test-%s", strings.ReplaceAll(tc.name, " ", "-"))
			env.CreateNamespace(t, namespace)
			defer env.DeleteNamespace(t, namespace)

			// Create test model (required by agent)
			env.CreateTestModel(t, namespace, "test-model")

			// Create agent
			agent := NewTestLanguageAgent(namespace, "test-agent", langopv1alpha1.LanguageAgentSpec{
				Instructions:  tc.instructions,
				ExecutionMode: tc.mode,
			})

			env.CreateLanguageAgent(t, agent)

			// Wait for synthesis
			err := env.WaitForCondition(t, namespace, "test-agent", "Synthesized", metav1.ConditionTrue)
			require.NoError(t, err, "Synthesis should complete for mode: %s", tc.mode)

			// Verify ConfigMap
			cm := env.GetConfigMap(t, namespace, "test-agent-code")
			code := cm.Data["agent.rb"]

			if tc.checkSchedule {
				assert.Contains(t, code, "schedule", "Scheduled mode should have schedule in code")
			}

			t.Logf("Mode: %s\nGenerated code:\n%s", tc.mode, code)
		})
	}
}

// TestAgentStatusUpdates tests that agent status is updated correctly
func TestAgentStatusUpdates(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping e2e test in short mode")
	}

	// Setup test environment
	env := SetupTestEnvironment(t)
	defer env.Teardown(t)

	// Start mock LLM service
	mockLLM := NewMockLLMService(t)
	defer mockLLM.Close()

	// Create mock chat model and set synthesizer
	mockChatModel := NewMockChatModel(mockLLM)
	env.SetSynthesizer(t, mockChatModel)

	// Create test namespace
	namespace := "test-status-updates"
	env.CreateNamespace(t, namespace)
	defer env.DeleteNamespace(t, namespace)

	// Create test model (required by agent)
	env.CreateTestModel(t, namespace, "test-model")

	// Create agent
	agent := NewTestLanguageAgent(namespace, "test-agent", langopv1alpha1.LanguageAgentSpec{
		Instructions:  "Simple test agent",
		ExecutionMode: "autonomous",
	})

	env.CreateLanguageAgent(t, agent)

	// Wait for synthesis
	err := env.WaitForCondition(t, namespace, "test-agent", "Synthesized", metav1.ConditionTrue)
	require.NoError(t, err, "Synthesis should complete")

	// Check updated state
	updatedAgent := env.GetLanguageAgent(t, namespace, "test-agent")
	assert.NotEmpty(t, updatedAgent.Status.Conditions, "Status conditions should be populated")

	// Verify specific conditions
	hasSynthesized := false
	for _, cond := range updatedAgent.Status.Conditions {
		t.Logf("Condition: Type=%s Status=%s Reason=%s Message=%s",
			cond.Type, cond.Status, cond.Reason, cond.Message)

		if cond.Type == "Synthesized" && cond.Status == metav1.ConditionTrue {
			hasSynthesized = true
		}
	}

	assert.True(t, hasSynthesized, "Agent should have Synthesized condition set to True")
}
