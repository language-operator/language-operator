package validation

import (
	"context"
	"encoding/json"
	"fmt"
	"os/exec"
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

// ValidateRubyCode validates Ruby code using AST-based analysis
// It shells out to the Ruby gem's AST validator for accurate parsing
func ValidateRubyCode(code string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()

	// Execute Ruby wrapper script that calls the gem's AST validator
	cmd := exec.CommandContext(ctx, "ruby", "/usr/local/bin/validate-ruby-code.rb")
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
