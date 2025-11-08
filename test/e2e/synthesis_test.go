package e2e

import (
	"context"
	"os"
	"strings"
	"testing"

	"github.com/based/language-operator/pkg/synthesis"
	"github.com/go-logr/logr"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestSynthesisQuality tests the quality of synthesized code
func TestSynthesisQuality(t *testing.T) {
	// Start mock LLM service
	mockLLM := NewMockLLMService(t)
	defer mockLLM.Close()

	// Create mock chat model that uses the mock LLM service
	mockChatModel := NewMockChatModel(mockLLM)

	testCases := []struct {
		name             string
		instructions     string
		expectedSchedule string
		expectedTools    []string
		shouldContain    []string
	}{
		{
			name:             "daily spreadsheet review",
			instructions:     "review my spreadsheet at 4pm daily and email me errors",
			expectedSchedule: "0 16 * * *",
			expectedTools:    []string{"google-sheets", "email"},
			shouldContain:    []string{"schedule", "workflow", "step"},
		},
		{
			name:             "health monitoring",
			instructions:     "check https://api.example.com/health every 5 minutes",
			expectedSchedule: "*/5 * * * *",
			expectedTools:    []string{"web-fetch"},
			shouldContain:    []string{"schedule", "workflow", "web-fetch"},
		},
		{
			name:             "hourly sync",
			instructions:     "sync data from API every hour",
			expectedSchedule: "0 * * * *",
			expectedTools:    []string{"web-fetch"},
			shouldContain:    []string{"schedule", "workflow"},
		},
		{
			name:             "daily morning report",
			instructions:     "send me a report at 9am every day",
			expectedSchedule: "0 9 * * *",
			expectedTools:    []string{"email"},
			shouldContain:    []string{"schedule", "workflow", "email"},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Create synthesizer
			synthesizer := synthesis.NewSynthesizer(mockChatModel, testLogger(t))

			// Create synthesis request
			req := synthesis.AgentSynthesisRequest{
				Instructions: tc.instructions,
				Tools:        tc.expectedTools,
				AgentName:    "test-agent",
				Namespace:    "default",
			}

			// Synthesize code
			ctx := context.Background()
			resp, err := synthesizer.SynthesizeAgent(ctx, req)
			require.NoError(t, err, "Synthesis should not fail")
			require.NotNil(t, resp, "Response should not be nil")

			code := resp.DSLCode

			// Verify code is not empty
			assert.NotEmpty(t, code, "Generated code should not be empty")

			// Verify code starts with require statement
			assert.True(t,
				strings.HasPrefix(strings.TrimSpace(code), "require"),
				"Code should start with require statement")

			// Verify code contains agent definition
			assert.Contains(t, code, "agent", "Code should contain agent definition")

			// Verify code is valid Ruby (basic checks)
			assert.True(t,
				strings.Count(code, "do") <= strings.Count(code, "end"),
				"Code should have balanced do/end blocks")

			// Verify expected content
			for _, expected := range tc.shouldContain {
				assert.Contains(t, code, expected,
					"Code should contain '%s'", expected)
			}

			// Verify schedule if expected
			if tc.expectedSchedule != "" {
				assert.Contains(t, code, tc.expectedSchedule,
					"Code should contain schedule: %s", tc.expectedSchedule)
			}

			// Verify tools are referenced
			for _, tool := range tc.expectedTools {
				// Tool might be referenced as 'google-sheets' or 'google_sheets'
				normalized := strings.ReplaceAll(tool, "-", "_")
				hasHyphen := strings.Contains(code, tool)
				hasUnderscore := strings.Contains(code, normalized)

				assert.True(t, hasHyphen || hasUnderscore,
					"Code should reference tool: %s", tool)
			}

			t.Logf("Generated code:\n%s", code)
		})
	}
}

// TestSynthesisWithExistingFixtures tests synthesis with existing test fixtures
func TestSynthesisWithExistingFixtures(t *testing.T) {
	// Start mock LLM service
	mockLLM := NewMockLLMService(t)
	defer mockLLM.Close()

	// Create mock chat model that uses the mock LLM service
	mockChatModel := NewMockChatModel(mockLLM)

	// Test with a few existing fixtures
	fixtures := []string{
		"../../test/instructions/devops/health-check-simple.txt",
		"../../test/instructions/communication/meeting-notes.txt",
		"../../test/instructions/financial/spreadsheet-review.txt",
	}

	for _, fixture := range fixtures {
		t.Run(fixture, func(t *testing.T) {
			// Read fixture file
			content, err := os.ReadFile(fixture)
			if err != nil {
				t.Skipf("Skipping fixture %s: %v", fixture, err)
				return
			}

			// Parse YAML frontmatter
			parts := strings.Split(string(content), "---")
			if len(parts) < 3 {
				t.Skipf("Skipping fixture %s: invalid format", fixture)
				return
			}

			instructions := strings.TrimSpace(parts[2])

			// Create synthesizer
			synthesizer := synthesis.NewSynthesizer(mockChatModel, testLogger(t))

			// Create synthesis request
			req := synthesis.AgentSynthesisRequest{
				Instructions: instructions,
				AgentName:    "test-agent",
				Namespace:    "default",
			}

			// Synthesize code
			ctx := context.Background()
			resp, err := synthesizer.SynthesizeAgent(ctx, req)
			require.NoError(t, err, "Synthesis should not fail for fixture: %s", fixture)
			require.NotNil(t, resp, "Response should not be nil")

			code := resp.DSLCode

			// Basic validation
			assert.NotEmpty(t, code, "Generated code should not be empty")
			assert.Contains(t, code, "agent", "Code should contain agent definition")

			t.Logf("Fixture: %s\nGenerated code:\n%s", fixture, code)
		})
	}
}

// TestSynthesisValidation tests that synthesis validates output correctly
// Note: Detailed validation tests are in pkg/synthesis/synthesizer_test.go
// This e2e test ensures validation is actually applied during synthesis
func TestSynthesisValidation(t *testing.T) {
	// Start mock LLM service
	mockLLM := NewMockLLMService(t)
	defer mockLLM.Close()

	// Create mock chat model that uses the mock LLM service
	mockChatModel := NewMockChatModel(mockLLM)

	// Create synthesizer
	synthesizer := synthesis.NewSynthesizer(mockChatModel, testLogger(t))

	// Test that synthesis applies validation
	req := synthesis.AgentSynthesisRequest{
		Instructions: "test validation",
		AgentName:    "test-agent",
		Namespace:    "default",
	}

	ctx := context.Background()
	resp, err := synthesizer.SynthesizeAgent(ctx, req)

	// The mock should generate valid code
	require.NoError(t, err, "Synthesis should succeed with valid mock output")
	require.NotNil(t, resp, "Response should not be nil")
	assert.NotEmpty(t, resp.DSLCode, "Generated code should not be empty")
	assert.Contains(t, resp.DSLCode, "agent", "Code should contain agent definition")
}

// testLogger creates a logger for testing
func testLogger(t *testing.T) logr.Logger {
	return logr.Discard()
}
