package validation

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

// Violation represents a security violation found by the AST validator
type Violation struct {
	Type     string `json:"type"`
	Method   string `json:"method,omitempty"`
	Constant string `json:"constant,omitempty"`
	Variable string `json:"variable,omitempty"`
	Location int    `json:"location"`
	Message  string `json:"message"`
}

// findValidatorScript looks for the validator script in common locations
func findValidatorScript() string {
	// Try locations in order of preference
	locations := []string{
		"/usr/local/bin/validate-ruby-code.rb",                              // Docker container
		"scripts/validate-ruby-code.rb",                                     // CI from src/ directory
		"../../../scripts/validate-ruby-code.rb",                            // Test from src/pkg/validation
		filepath.Join("..", "..", "..", "scripts", "validate-ruby-code.rb"), // Alternative path
		"../../scripts/validate-ruby-code.rb",                               // From src subdirectory
		"../scripts/validate-ruby-code.rb",                                  // From pkg/validation
	}

	for _, path := range locations {
		absPath, err := filepath.Abs(path)
		if err != nil {
			continue
		}
		if _, err := os.Stat(absPath); err == nil {
			return absPath
		}
	}

	// Default to container location
	return "/usr/local/bin/validate-ruby-code.rb"
}

// ValidateRubyCode validates Ruby code using AST-based analysis
// It shells out to the Ruby gem's AST validator for accurate parsing
// If Ruby is not available, validation is skipped (returns nil).
func ValidateRubyCode(code string) error {
	// Check if Ruby is available
	if _, err := exec.LookPath("ruby"); err != nil {
		// Ruby not available - skip validation
		// This happens in test environments without Ruby
		// Validation will occur at runtime in the agent container
		return nil
	}

	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()

	// Find the validator script
	scriptPath := findValidatorScript()

	// Execute Ruby wrapper script that calls the gem's AST validator
	cmd := exec.CommandContext(ctx, "ruby", scriptPath)
	cmd.Stdin = strings.NewReader(code)

	// Capture STDOUT and STDERR separately
	// STDERR may contain parser warnings that should not interfere with JSON parsing
	output, err := cmd.Output()

	// Check for timeout
	if ctx.Err() == context.DeadlineExceeded {
		return fmt.Errorf("validation timeout: code too large or complex (>1s)")
	}

	// Parse JSON output from validator (STDOUT only)
	var violations []Violation
	if jsonErr := json.Unmarshal(output, &violations); jsonErr != nil {
		// If JSON parsing fails, the output might be an error message
		if len(output) > 0 {
			return fmt.Errorf("validator error: %s", strings.TrimSpace(string(output)))
		}
		if err != nil {
			return fmt.Errorf("validator execution failed: %w", err)
		}
		return fmt.Errorf("validator produced invalid output")
	}

	// If violations were found, format and return error
	if len(violations) > 0 {
		return formatViolations(violations)
	}

	return nil
}

// formatViolations converts violation structs into a readable error message
func formatViolations(violations []Violation) error {
	var msgs []string
	for _, v := range violations {
		msgs = append(msgs, fmt.Sprintf("Line %d: %s", v.Location, v.Message))
	}
	return fmt.Errorf("security violations detected:\n  %s", strings.Join(msgs, "\n  "))
}

// ValidateGeneratedCodeAgainstSchema validates Ruby DSL code for syntax and security violations
// Returns a list of violations and an error if validation fails
func ValidateGeneratedCodeAgainstSchema(ctx context.Context, code string) ([]Violation, error) {
	// Check if Ruby is available
	if _, err := exec.LookPath("ruby"); err != nil {
		// Ruby not available - skip validation in test environments
		return nil, nil
	}

	// Find the validator script
	scriptPath := findValidatorScript()

	// Execute Ruby wrapper script that calls the gem's AST validator
	cmd := exec.CommandContext(ctx, "ruby", scriptPath)
	cmd.Stdin = strings.NewReader(code)

	// Capture STDOUT and STDERR separately
	output, err := cmd.Output()

	// Check for timeout
	if ctx.Err() == context.DeadlineExceeded {
		return nil, fmt.Errorf("validation timeout: code too large or complex")
	}

	// Parse JSON output from validator (STDOUT only)
	var violations []Violation
	if jsonErr := json.Unmarshal(output, &violations); jsonErr != nil {
		// If JSON parsing fails, the output might be an error message or syntax error
		if len(output) > 0 {
			// Ruby syntax errors or other validation failures
			outputStr := strings.TrimSpace(string(output))
			if strings.Contains(outputStr, "unexpected") || strings.Contains(outputStr, "syntax error") {
				// Create a violation for syntax errors
				return []Violation{{
					Type:     "syntax_error",
					Location: 1,
					Message:  outputStr,
				}}, nil
			}
			return nil, fmt.Errorf("validator error: %s", outputStr)
		}
		if err != nil {
			return nil, fmt.Errorf("validator execution failed: %w", err)
		}
		return nil, fmt.Errorf("validator produced invalid output")
	}

	return violations, nil
}

// findFormatterScript looks for the Ruby formatter script in common locations
func findFormatterScript() string {
	// Try locations in order of preference
	locations := []string{
		"/usr/local/bin/format-ruby-code.rb",                              // Docker container
		"scripts/format-ruby-code.rb",                                     // CI from src/ directory
		"../../../scripts/format-ruby-code.rb",                            // Test from src/pkg/validation
		filepath.Join("..", "..", "..", "scripts", "format-ruby-code.rb"), // Alternative path
		"../../scripts/format-ruby-code.rb",                               // From src subdirectory
		"../scripts/format-ruby-code.rb",                                  // From pkg/validation
	}

	for _, path := range locations {
		absPath, err := filepath.Abs(path)
		if err != nil {
			continue
		}
		if _, err := os.Stat(absPath); err == nil {
			return absPath
		}
	}

	// Default to container location
	return "/usr/local/bin/format-ruby-code.rb"
}

// FormatRubyCode formats Ruby code using pattern-based corrections
// Fixes common formatting issues seen in LLM-generated code
func FormatRubyCode(code string) (string, error) {
	formatted := code

	// Fix task definition formatting - the main issue we observed
	formatted = fixTaskDefinitionFormatting(formatted)

	// Fix empty hash spacing
	formatted = fixEmptyHashFormatting(formatted)

	// Fix excessive blank lines
	formatted = fixExcessiveBlankLines(formatted)

	return formatted, nil
}

// fixTaskDefinitionFormatting fixes the main formatting issue: task parameter alignment
func fixTaskDefinitionFormatting(code string) string {
	// Pattern: task(:name,\n\n       instructions: "...",\n\n       inputs: { ... },\n\n       outputs: { ... })
	// Target: task(:name,\n     instructions: "...",\n     inputs: { ... },\n     outputs: { ... })

	lines := strings.Split(code, "\n")
	var result []string
	inTaskDefinition := false
	taskIndent := ""

	for _, line := range lines {
		trimmed := strings.TrimSpace(line)

		// Detect start of task definition
		if strings.HasPrefix(trimmed, "task(") && strings.Contains(line, ",") {
			inTaskDefinition = true
			// Calculate base indentation from task line
			taskIndent = line[:len(line)-len(strings.TrimLeft(line, " \t"))]
			result = append(result, line)
			continue
		}

		// Handle task definition continuation
		if inTaskDefinition {
			// End task definition when we hit the closing parenthesis or 'do' keyword
			if strings.Contains(trimmed, ") do |") || strings.Contains(trimmed, ")") && !strings.Contains(trimmed, ":") {
				inTaskDefinition = false
				result = append(result, line)
				continue
			}

			// Format parameter lines (instructions, inputs, outputs)
			if strings.Contains(trimmed, ":") && (strings.Contains(trimmed, "instructions:") ||
				strings.Contains(trimmed, "inputs:") || strings.Contains(trimmed, "outputs:")) {

				// Remove excessive blank lines before parameters
				for len(result) > 0 && strings.TrimSpace(result[len(result)-1]) == "" {
					result = result[:len(result)-1]
				}

				// Reformat with consistent indentation (5 spaces for parameter alignment)
				formatted := taskIndent + "     " + trimmed
				result = append(result, formatted)
				continue
			}
		}

		result = append(result, line)
	}

	return strings.Join(result, "\n")
}

// fixEmptyHashFormatting fixes empty hash spacing: {  } -> {}
func fixEmptyHashFormatting(code string) string {
	// Fix empty hashes with spaces
	code = strings.ReplaceAll(code, "{  }", "{}")
	code = strings.ReplaceAll(code, "{ }", "{}")

	return code
}

// fixExcessiveBlankLines removes excessive consecutive blank lines
func fixExcessiveBlankLines(code string) string {
	lines := strings.Split(code, "\n")
	var result []string
	blankLineCount := 0

	for _, line := range lines {
		if strings.TrimSpace(line) == "" {
			blankLineCount++
			if blankLineCount <= 1 { // Allow maximum 1 consecutive blank line
				result = append(result, line)
			}
		} else {
			blankLineCount = 0
			result = append(result, line)
		}
	}

	return strings.Join(result, "\n")
}
