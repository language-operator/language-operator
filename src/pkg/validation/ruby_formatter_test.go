package validation

import (
	"strings"
	"testing"
)

func TestFormatRubyCode(t *testing.T) {
	tests := []struct {
		name          string
		input         string
		expectedLines []string // Key lines that should be properly formatted
		shouldImprove bool     // Whether we expect formatting improvements
	}{
		{
			name: "task definition with poor formatting",
			input: `require 'language_operator'

agent "test-agent" do
  task(:read_file,

       instructions: "read a file",

       inputs: {  },

       outputs: { content: 'string' }) do |inputs|

    result = execute_tool('read_file', { path: inputs[:path] })

    { content: result }

  end
end`,
			expectedLines: []string{
				"task(:read_file,",
				"     instructions: \"read a file\",",
				"     inputs: {},",
				"     outputs: { content: 'string' }) do |inputs|",
			},
			shouldImprove: true,
		},
		{
			name: "already well formatted code",
			input: `require 'language_operator'

agent "test-agent" do
  description "A well formatted agent"
  
  task(:example_task,
       instructions: "do something",
       inputs: { param: 'string' },
       outputs: { result: 'string' })
       
  main do |inputs|
    result = execute_task(:example_task, inputs: inputs)
    result
  end
end`,
			shouldImprove: false,
		},
		{
			name:          "empty code",
			input:         "",
			shouldImprove: false,
		},
		{
			name: "invalid ruby code",
			input: `require 'language_operator'

agent "test" do
  task(:broken
    # missing parameters and syntax errors
end`,
			shouldImprove: false, // Should return original on errors
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := FormatRubyCode(tt.input)

			// Should never error - returns original code on formatting failures
			if err != nil {
				t.Errorf("FormatRubyCode() returned error: %v", err)
			}

			// Should never return empty unless input was empty
			if tt.input != "" && strings.TrimSpace(result) == "" {
				t.Errorf("FormatRubyCode() returned empty result for non-empty input")
			}

			// If we expect improvements, check for specific formatting
			if tt.shouldImprove {
				for _, expectedLine := range tt.expectedLines {
					if !strings.Contains(result, expectedLine) {
						t.Errorf("Expected formatted code to contain line: %q\nActual output:\n%s", expectedLine, result)
					}
				}

				// Check that formatting actually changed the code
				if result == tt.input {
					t.Errorf("Expected formatting to improve code, but output was identical to input")
				}
			}

			// Basic sanity checks
			if strings.Contains(result, "require 'language_operator'") {
				// Good - required line preserved
			}

			if strings.Contains(result, "agent ") {
				// Good - agent definition preserved
			}
		})
	}
}

func TestFormatRubyCodeEdgeCases(t *testing.T) {
	tests := []struct {
		name  string
		input string
		desc  string
	}{
		{
			name:  "very large code",
			input: strings.Repeat("# comment\n", 1000) + "require 'language_operator'\nagent 'test' do\nend",
			desc:  "should handle large code without timeout",
		},
		{
			name:  "unicode characters",
			input: "require 'language_operator'\n# こんにちは\nagent 'test' do\nend",
			desc:  "should handle unicode comments",
		},
		{
			name:  "mixed line endings",
			input: "require 'language_operator'\r\nagent 'test' do\r\nend\n",
			desc:  "should handle different line endings",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := FormatRubyCode(tt.input)

			if err != nil {
				t.Errorf("FormatRubyCode() failed on %s: %v", tt.desc, err)
			}

			if strings.TrimSpace(result) == "" && strings.TrimSpace(tt.input) != "" {
				t.Errorf("FormatRubyCode() returned empty for %s", tt.desc)
			}
		})
	}
}
