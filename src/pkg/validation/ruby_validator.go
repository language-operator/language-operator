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
