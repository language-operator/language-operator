package synthesis

import (
	"context"
	"testing"

	"github.com/go-logr/logr"
)

func TestTaskValidator_ValidateTaskAgent(t *testing.T) {
	logger := logr.Discard()
	validator := NewTaskValidator(logger)
	ctx := context.Background()

	tests := []struct {
		name         string
		code         string
		expectErrors bool
		errorTypes   []string
	}{
		{
			name: "valid task agent",
			code: `require 'language_operator'

agent "test-agent" do
  description "Test agent"
  
  task :fetch_data,
    instructions: "fetch some data from API",
    inputs: { source: 'string' },
    outputs: { data: 'array', count: 'integer' }
  
  task :process_data,
    instructions: "process the fetched data",
    inputs: { data: 'array' },
    outputs: { result: 'hash' }
  
  main do |inputs|
    raw_data = execute_task(:fetch_data, inputs: { source: inputs[:source] })
    result = execute_task(:process_data, inputs: { data: raw_data[:data] })
    result
  end
end`,
			expectErrors: false,
		},
		{
			name: "agent missing main block",
			code: `require 'language_operator'

agent "test-agent" do
  description "Test agent"
  
  task :fetch_data,
    instructions: "fetch some data",
    inputs: {},
    outputs: { data: 'array' }
end`,
			expectErrors: true,
			errorTypes:   []string{"main_block"},
		},
		{
			name: "task with invalid type",
			code: `require 'language_operator'

agent "test-agent" do
  description "Test agent"
  
  task :fetch_data,
    instructions: "fetch some data",
    inputs: { source: 'invalid_type' },
    outputs: { data: 'array' }
  
  main do |inputs|
    result = execute_task(:fetch_data, inputs: inputs)
    result
  end
end`,
			expectErrors: true,
			errorTypes:   []string{"type_definition"},
		},
		{
			name: "calling undefined task",
			code: `require 'language_operator'

agent "test-agent" do
  description "Test agent"
  
  task :fetch_data,
    instructions: "fetch some data",
    inputs: {},
    outputs: { data: 'array' }
  
  main do |inputs|
    result = execute_task(:process_data, inputs: inputs)
    result
  end
end`,
			expectErrors: true,
			errorTypes:   []string{"task_call"},
		},
		{
			name: "symbolic task with safe code",
			code: `require 'language_operator'

agent "test-agent" do
  description "Test agent"
  
  task :add_numbers do |inputs|
    result = inputs[:a] + inputs[:b]
    { sum: result }
  end
  
  main do |inputs|
    result = execute_task(:add_numbers, inputs: inputs)
    result
  end
end`,
			expectErrors: false,
		},
		{
			name: "symbolic task with dangerous code",
			code: `require 'language_operator'

agent "test-agent" do
  description "Test agent"
  
  task :dangerous_task do |inputs|
    system("rm -rf /")
    { result: "done" }
  end
  
  main do |inputs|
    result = execute_task(:dangerous_task, inputs: inputs)
    result
  end
end`,
			expectErrors: true,
			errorTypes:   []string{"security"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			errors, err := validator.ValidateTaskAgent(ctx, tt.code)
			if err != nil {
				t.Fatalf("ValidateTaskAgent() failed: %v", err)
			}

			// Filter to only error-level violations
			errorCount := 0
			foundErrorTypes := make(map[string]bool)
			for _, validationErr := range errors {
				if validationErr.Severity == "error" {
					errorCount++
					foundErrorTypes[validationErr.Type] = true
				}
			}

			hasErrors := errorCount > 0
			if hasErrors != tt.expectErrors {
				t.Errorf("Expected errors: %v, got errors: %v (found %d errors)", 
					tt.expectErrors, hasErrors, errorCount)
				for _, err := range errors {
					t.Logf("  - %s [%s]: %s", err.Type, err.Severity, err.Message)
				}
				
				// Debug: show parsed structure for failing test
				if tt.name == "calling_undefined_task" {
					agent, parseErr := validator.ParseAgentStructure(tt.code)
					if parseErr == nil {
						t.Logf("DEBUG: Parsed %d tasks", len(agent.Tasks))
						for name := range agent.Tasks {
							t.Logf("  - Task: %s", name)
						}
						if agent.MainBlock != nil {
							t.Logf("DEBUG: Main block found with %d task calls", len(agent.MainBlock.TaskCalls))
							for _, call := range agent.MainBlock.TaskCalls {
								t.Logf("  - Calls: %s", call.TaskName)
							}
							t.Logf("DEBUG: Main block code: %q", agent.MainBlock.CodeBlock)
						} else {
							t.Logf("DEBUG: No main block found")
						}
					} else {
						t.Logf("DEBUG: Parse error: %v", parseErr)
					}
				}
			}

			// Check that expected error types are present
			for _, expectedType := range tt.errorTypes {
				if !foundErrorTypes[expectedType] {
					t.Errorf("Expected error type '%s' not found. Found types: %v", 
						expectedType, foundErrorTypes)
				}
			}
		})
	}
}

func TestTaskValidator_ParseTypeHash(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected map[string]string
	}{
		{
			name:  "simple type hash",
			input: "name: 'string', count: 'integer'",
			expected: map[string]string{
				"name":  "string",
				"count": "integer",
			},
		},
		{
			name:  "empty string",
			input: "",
			expected: map[string]string{},
		},
		{
			name:  "single type",
			input: "data: 'array'",
			expected: map[string]string{
				"data": "array",
			},
		},
		{
			name:  "double quotes",
			input: `source: "string", result: "hash"`,
			expected: map[string]string{
				"source": "string", 
				"result": "hash",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := parseTypeHash(tt.input)
			
			if len(result) != len(tt.expected) {
				t.Errorf("Expected %d items, got %d", len(tt.expected), len(result))
			}
			
			for key, expectedValue := range tt.expected {
				if actualValue, exists := result[key]; !exists {
					t.Errorf("Expected key '%s' not found", key)
				} else if actualValue != expectedValue {
					t.Errorf("For key '%s': expected '%s', got '%s'", 
						key, expectedValue, actualValue)
				}
			}
		})
	}
}

func TestTaskValidator_IsValidDSLType(t *testing.T) {
	validTypes := []string{"string", "integer", "number", "boolean", "array", "hash", "any"}
	invalidTypes := []string{"String", "int", "float", "object", "list", "invalid_type"}

	for _, validType := range validTypes {
		t.Run("valid_type_"+validType, func(t *testing.T) {
			if !isValidDSLType(validType) {
				t.Errorf("Expected '%s' to be valid DSL type", validType)
			}
		})
	}

	for _, invalidType := range invalidTypes {
		t.Run("invalid_type_"+invalidType, func(t *testing.T) {
			if isValidDSLType(invalidType) {
				t.Errorf("Expected '%s' to be invalid DSL type", invalidType)
			}
		})
	}
}