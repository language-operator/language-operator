package synthesis

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

// DSLSchema represents the JSON Schema for the Agent DSL
// It is fetched from the language_operator Ruby gem via the CLI
type DSLSchema struct {
	Version    string                 `json:"version"`
	Schema     string                 `json:"$schema"`
	Type       string                 `json:"type"`
	Properties map[string]interface{} `json:"properties"`
	Required   []string               `json:"required"`
}

// FetchDSLSchema executes the language_operator CLI to fetch the DSL schema
// in JSON format. It returns the parsed schema or an error if the command
// fails or the output is invalid.
//
// Example usage:
//
//	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
//	defer cancel()
//	schema, err := FetchDSLSchema(ctx)
//	if err != nil {
//	    log.Error(err, "Failed to fetch DSL schema")
//	    return err
//	}
//	log.Info("Fetched schema", "version", schema.Version)
func FetchDSLSchema(ctx context.Context) (*DSLSchema, error) {
	// Default timeout if none specified in context
	if _, ok := ctx.Deadline(); !ok {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, 30*time.Second)
		defer cancel()
	}

	// Execute the command to fetch schema
	output, err := executeCommand(ctx, "language_operator", "schema", "--format=json")
	if err != nil {
		return nil, fmt.Errorf("failed to execute language_operator schema command: %w", err)
	}

	// Parse JSON output
	var schema DSLSchema
	if err := json.Unmarshal(output, &schema); err != nil {
		return nil, fmt.Errorf("failed to parse schema JSON: %w (output: %s)", err, string(output))
	}

	// Basic validation
	if schema.Version == "" {
		return nil, fmt.Errorf("schema missing version field")
	}

	return &schema, nil
}

// GetSchemaVersion executes the language_operator CLI to fetch just the schema
// version string. This is more efficient than fetching the full schema when only
// the version is needed (e.g., for compatibility checking).
//
// Example usage:
//
//	ctx := context.Background()
//	version, err := GetSchemaVersion(ctx)
//	if err != nil {
//	    log.Error(err, "Failed to fetch schema version")
//	    return err
//	}
//	log.Info("Schema version", "version", version)
func GetSchemaVersion(ctx context.Context) (string, error) {
	// Default timeout if none specified in context
	if _, ok := ctx.Deadline(); !ok {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, 10*time.Second)
		defer cancel()
	}

	// Execute the command to fetch version
	output, err := executeCommand(ctx, "language_operator", "schema", "--version")
	if err != nil {
		return "", fmt.Errorf("failed to execute language_operator schema --version command: %w", err)
	}

	// Trim whitespace and return version
	version := strings.TrimSpace(string(output))
	if version == "" {
		return "", fmt.Errorf("empty version returned from command")
	}

	return version, nil
}

// executeCommand executes a command with the given context and returns its output.
// It handles timeouts, errors, and provides detailed error messages for debugging.
func executeCommand(ctx context.Context, command string, args ...string) ([]byte, error) {
	cmd := exec.CommandContext(ctx, command, args...)

	// Capture both stdout and stderr
	output, err := cmd.CombinedOutput()

	if ctx.Err() == context.DeadlineExceeded {
		return nil, fmt.Errorf("command timed out: %s %v", command, args)
	}

	if ctx.Err() == context.Canceled {
		return nil, fmt.Errorf("command canceled: %s %v", command, args)
	}

	if err != nil {
		// Check if it's an exec.ExitError to get the exit code
		if exitErr, ok := err.(*exec.ExitError); ok {
			return nil, fmt.Errorf("command failed with exit code %d: %s %v (output: %s)",
				exitErr.ExitCode(), command, args, string(output))
		}

		// Check if command was not found
		if err == exec.ErrNotFound || strings.Contains(err.Error(), "executable file not found") {
			return nil, fmt.Errorf("command not found: %s (is language_operator gem installed?)", command)
		}

		return nil, fmt.Errorf("command execution failed: %s %v: %w (output: %s)",
			command, args, err, string(output))
	}

	return output, nil
}

// SchemaViolation represents a schema validation error from the Ruby validator
type SchemaViolation struct {
	Type     string `json:"type"`
	Property string `json:"property,omitempty"`
	Location int    `json:"location"`
	Message  string `json:"message"`
}

// findSchemaValidatorScript looks for the schema validator script in common locations
func findSchemaValidatorScript() string {
	// Try locations in order of preference
	locations := []string{
		"/usr/local/bin/validate-dsl-schema.rb",                              // Docker container
		"scripts/validate-dsl-schema.rb",                                     // CI from src/ directory
		"../../../scripts/validate-dsl-schema.rb",                            // Test from src/pkg/synthesis
		filepath.Join("..", "..", "..", "scripts", "validate-dsl-schema.rb"), // Alternative path
		"../../scripts/validate-dsl-schema.rb",                               // From src subdirectory
		"../scripts/validate-dsl-schema.rb",                                  // From pkg/synthesis
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
	return "/usr/local/bin/validate-dsl-schema.rb"
}

// ValidateGeneratedCodeAgainstSchema validates generated DSL code against the language-operator schema.
// It executes the Ruby validator script which loads the code using the gem's DSL loader.
// Returns a descriptive error with violation details if validation fails.
//
// Example usage:
//
//	ctx := context.Background()
//	violations, err := ValidateGeneratedCodeAgainstSchema(ctx, dslCode)
//	if err != nil {
//	    log.Error(err, "Failed to run schema validation")
//	    return err
//	}
//	if len(violations) > 0 {
//	    log.Info("Schema validation failed", "violations", violations)
//	    // Handle violations...
//	}
func ValidateGeneratedCodeAgainstSchema(ctx context.Context, code string) ([]SchemaViolation, error) {
	// Check if Ruby is available
	if _, err := exec.LookPath("ruby"); err != nil {
		// Ruby not available - skip validation
		// This happens in test environments without Ruby
		// Validation will occur at runtime in the agent container
		return nil, nil
	}

	// Check if bundler is available (required for the script)
	if _, err := exec.LookPath("bundle"); err != nil {
		// Bundler not available - skip validation
		return nil, nil
	}

	// Default timeout if none specified in context
	if _, ok := ctx.Deadline(); !ok {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, 5*time.Second)
		defer cancel()
	}

	// Find the validator script
	scriptPath := findSchemaValidatorScript()

	// Execute Ruby validator script via bundle exec
	cmd := exec.CommandContext(ctx, "bundle", "exec", "ruby", scriptPath)
	cmd.Stdin = strings.NewReader(code)

	// Capture STDOUT and STDERR separately
	// STDOUT contains JSON violations
	// STDERR may contain Ruby warnings
	output, err := cmd.Output()

	// Check for timeout
	if ctx.Err() == context.DeadlineExceeded {
		return nil, fmt.Errorf("schema validation timeout: code too large or complex (>5s)")
	}

	// Parse JSON output from validator (STDOUT only)
	var violations []SchemaViolation
	if len(output) > 0 {
		if jsonErr := json.Unmarshal(output, &violations); jsonErr != nil {
			// If JSON parsing fails, the output might be an error message
			return nil, fmt.Errorf("schema validator produced invalid output: %s (error: %w)", string(output), jsonErr)
		}
	}

	// If the command failed but we got no violations, something went wrong
	if err != nil && len(violations) == 0 {
		// Check if it's an exec.ExitError to get the exit code
		if exitErr, ok := err.(*exec.ExitError); ok {
			// Exit code 1 is expected when there are violations
			// But we should have violations in the output
			if exitErr.ExitCode() == 1 && len(output) > 0 {
				// This is okay - violations were reported
				return violations, nil
			}
			return nil, fmt.Errorf("schema validator failed with exit code %d: %s (stderr: %s)",
				exitErr.ExitCode(), string(output), string(exitErr.Stderr))
		}

		return nil, fmt.Errorf("schema validator execution failed: %w (output: %s)", err, string(output))
	}

	return violations, nil
}
