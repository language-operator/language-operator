package synthesis

import (
	"testing"

	langopv1alpha1 "github.com/language-operator/language-operator/api/v1alpha1"
)

func TestBuildToolsList_EmptyRequest(t *testing.T) {
	s := &Synthesizer{}
	req := AgentSynthesisRequest{}

	result := s.buildToolsList(req)

	if result != "None" {
		t.Errorf("Expected 'None', got '%s'", result)
	}
}

func TestBuildToolsList_LegacyTools(t *testing.T) {
	s := &Synthesizer{}
	req := AgentSynthesisRequest{
		Tools: []string{"workspace", "github"},
	}

	result := s.buildToolsList(req)
	expected := "  - workspace\n  - github\n"

	if result != expected {
		t.Errorf("Expected '%s', got '%s'", expected, result)
	}
}

func TestBuildToolsList_ToolSchemas(t *testing.T) {
	s := &Synthesizer{}
	req := AgentSynthesisRequest{
		Tools: []string{"workspace"}, // Should be ignored in favor of ToolSchemas
		ToolSchemas: []langopv1alpha1.ToolSchema{
			{
				Name:        "read_file",
				Description: "Read contents of a file",
				InputSchema: &langopv1alpha1.ToolSchemaDefinition{
					Type:     "object",
					Required: []string{"path"},
					Properties: map[string]langopv1alpha1.ToolProperty{
						"path": {
							Type:        "string",
							Description: "Path to the file to read",
							Example:     "\"/workspace/data.txt\"",
						},
					},
				},
			},
			{
				Name:        "write_file",
				Description: "Write content to a file",
				InputSchema: &langopv1alpha1.ToolSchemaDefinition{
					Type:     "object",
					Required: []string{"path", "content"},
					Properties: map[string]langopv1alpha1.ToolProperty{
						"path": {
							Type:        "string",
							Description: "Path to the file to write",
						},
						"content": {
							Type:        "string",
							Description: "Content to write to the file",
						},
					},
				},
			},
		},
	}

	result := s.buildToolsList(req)

	// Check that it contains the expected tool information
	expected := []string{
		"### read_file",
		"Read contents of a file",
		"**Parameters:**",
		"- `path`: string (required) - Path to the file to read (e.g., \"/workspace/data.txt\")",
		"### write_file",
		"Write content to a file",
		"- `path`: string (required) - Path to the file to write",
		"- `content`: string (required) - Content to write to the file",
	}

	for _, expectedSubstring := range expected {
		if !stringContains(result, expectedSubstring) {
			t.Errorf("Expected result to contain '%s', but it didn't. Full result:\n%s", expectedSubstring, result)
		}
	}
}

func TestFormatToolSchemas_EmptySchemas(t *testing.T) {
	s := &Synthesizer{}

	result := s.formatToolSchemas([]langopv1alpha1.ToolSchema{})

	if result != "None" {
		t.Errorf("Expected 'None', got '%s'", result)
	}
}

func TestContainsString(t *testing.T) {
	tests := []struct {
		name     string
		slice    []string
		item     string
		expected bool
	}{
		{
			name:     "item exists",
			slice:    []string{"a", "b", "c"},
			item:     "b",
			expected: true,
		},
		{
			name:     "item does not exist",
			slice:    []string{"a", "b", "c"},
			item:     "d",
			expected: false,
		},
		{
			name:     "empty slice",
			slice:    []string{},
			item:     "a",
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := containsString(tt.slice, tt.item)
			if result != tt.expected {
				t.Errorf("containsString(%v, %q) = %v; want %v", tt.slice, tt.item, result, tt.expected)
			}
		})
	}
}

// Helper function to check if a string contains a substring
func stringContains(s, substr string) bool {
	return len(s) >= len(substr) && findSubstring(s, substr)
}

func findSubstring(s, substr string) bool {
	if len(substr) == 0 {
		return true
	}
	if len(s) < len(substr) {
		return false
	}
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
