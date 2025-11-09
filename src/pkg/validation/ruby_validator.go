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
		"/usr/local/bin/validate-ruby-code.rb",                        // Docker container
		"scripts/validate-ruby-code.rb",                               // CI from src/ directory
		"../../scripts/validate-ruby-code.rb",                         // Test from src/pkg/validation
		filepath.Join("..", "..", "scripts", "validate-ruby-code.rb"), // Alternative path
		"../scripts/validate-ruby-code.rb",                            // From pkg/validation
		filepath.Join("src", "scripts", "validate-ruby-code.rb"),      // From repo root
	}

	for _, path := range locations {
		absPath, err := filepath.Abs(path)
		if err != nil {
			fmt.Fprintf(os.Stderr, "DEBUG: path %s failed to resolve: %v\n", path, err)
			continue
		}
		_, statErr := os.Stat(absPath)
		if statErr == nil {
			fmt.Fprintf(os.Stderr, "DEBUG: Found validator script at %s\n", absPath)
			return absPath
		}
		fmt.Fprintf(os.Stderr, "DEBUG: path %s -> %s (not found: %v)\n", path, absPath, statErr)
	}

	// Default to container location
	fmt.Fprintf(os.Stderr, "DEBUG: No script found, defaulting to /usr/local/bin/validate-ruby-code.rb\n")
	return "/usr/local/bin/validate-ruby-code.rb"
}

// ValidateRubyCode validates Ruby code using AST-based analysis
// It shells out to the Ruby gem's AST validator for accurate parsing
func ValidateRubyCode(code string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()

	// Find the validator script
	scriptPath := findValidatorScript()

	// Execute Ruby wrapper script that calls the gem's AST validator
	cmd := exec.CommandContext(ctx, "ruby", scriptPath)
	cmd.Stdin = strings.NewReader(code)

	output, err := cmd.CombinedOutput()

	// Check for timeout
	if ctx.Err() == context.DeadlineExceeded {
		return fmt.Errorf("validation timeout: code too large or complex (>1s)")
	}

	// Parse JSON output from validator
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
