package integration

import (
	"context"
	"strings"
	"testing"

	"github.com/language-operator/language-operator/pkg/synthesis"
	"github.com/go-logr/logr"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestSynthesisQuality tests synthesis with various instructions
// This is a fast unit test - no Kubernetes required
func TestSynthesisQuality(t *testing.T) {
	// Start mock LLM service
	mockLLM := NewMockLLMService(t)
	defer mockLLM.Close()

	mockChatModel := NewMockChatModel(mockLLM)
	synthesizer := synthesis.NewSynthesizer(mockChatModel, logr.Discard())

	for _, scenario := range TestScenarios {
		// Skip scenarios meant to fail
		if scenario.ShouldFail {
			continue
		}

		t.Run(scenario.Name, func(t *testing.T) {
			req := synthesis.AgentSynthesisRequest{
				Instructions: scenario.Instructions,
				Tools:        scenario.ExpectedTools,
				AgentName:    "test-agent",
				Namespace:    "default",
			}

			ctx := context.Background()
			resp, err := synthesizer.SynthesizeAgent(ctx, req)

			require.NoError(t, err, "Synthesis should succeed")
			require.NotNil(t, resp, "Response should not be nil")

			code := resp.DSLCode

			// Verify code quality
			assert.NotEmpty(t, code, "Generated code should not be empty")
			assert.True(t,
				strings.HasPrefix(strings.TrimSpace(code), "require"),
				"Code should start with require statement")
			assert.Contains(t, code, "agent", "Code should contain agent definition")
			
			// Verify DSL v1 task/main model
			assert.Contains(t, code, "task :", "Code should contain task definition")
			assert.Contains(t, code, "main do", "Code should contain main block")
			assert.Contains(t, code, "execute_task", "Code should use execute_task")
			assert.Contains(t, code, "inputs:", "Code should have task inputs schema")
			assert.Contains(t, code, "outputs:", "Code should have task outputs schema")

			// Verify expected schedule
			if scenario.ExpectedSchedule != "" {
				assert.Contains(t, code, scenario.ExpectedSchedule,
					"Code should contain schedule: %s", scenario.ExpectedSchedule)
			}

			// Verify expected content
			for _, expected := range scenario.ShouldContain {
				assert.Contains(t, code, expected,
					"Code should contain '%s'", expected)
			}

			// Note: Tool verification removed because the workflow feature was removed in v0.1.36
			// Agents now receive tools via the controller's environment/configuration,
			// not through explicit workflow/step definitions in the DSL

			t.Logf("✓ Generated valid code for: %s", scenario.Name)
		})
	}
}

// TestSynthesisValidation tests that invalid inputs are rejected
func TestSynthesisValidation(t *testing.T) {
	mockLLM := NewMockLLMService(t)
	defer mockLLM.Close()

	mockChatModel := NewMockChatModel(mockLLM)
	synthesizer := synthesis.NewSynthesizer(mockChatModel, logr.Discard())

	tests := []struct {
		name         string
		instructions string
		expectError  bool
	}{
		{
			name:         "empty instructions",
			instructions: "",
			expectError:  true,
		},
		{
			name:         "whitespace only",
			instructions: "   \n\t  ",
			expectError:  true,
		},
		{
			name:         "very short valid",
			instructions: "test",
			expectError:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := synthesis.AgentSynthesisRequest{
				Instructions: tt.instructions,
				AgentName:    "test-agent",
				Namespace:    "default",
			}

			ctx := context.Background()
			resp, err := synthesizer.SynthesizeAgent(ctx, req)

			if tt.expectError {
				// Either error or empty response is acceptable for invalid input
				if err == nil && resp != nil {
					assert.Empty(t, resp.DSLCode, "Should not generate code for invalid input")
				}
			} else {
				require.NoError(t, err, "Valid input should not error")
				require.NotNil(t, resp, "Response should not be nil")
			}
		})
	}
}

// TestSynthesisErrorHandling tests that synthesis handles LLM errors gracefully
func TestSynthesisErrorHandling(t *testing.T) {
	// Create mock that returns errors
	mockLLM := NewMockLLMServiceWithError(t, "LLM service unavailable")
	defer mockLLM.Close()

	mockChatModel := NewMockChatModel(mockLLM)
	synthesizer := synthesis.NewSynthesizer(mockChatModel, logr.Discard())

	req := synthesis.AgentSynthesisRequest{
		Instructions: "This should fail",
		AgentName:    "test-agent",
		Namespace:    "default",
	}

	ctx := context.Background()
	_, err := synthesizer.SynthesizeAgent(ctx, req)

	assert.Error(t, err, "Should return error when LLM fails")
	assert.Contains(t, err.Error(), "LLM service unavailable",
		"Error should include LLM error message")
}

// TestSynthesisWithContextTimeout tests timeout handling
func TestSynthesisWithContextTimeout(t *testing.T) {
	t.Skip("TODO: Implement timeout testing")
}
