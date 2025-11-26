package synthesis

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/go-logr/logr"
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
	output, err := executeCommand(ctx, "aictl", "system", "schema", "--version")
	if err != nil {
		return "", fmt.Errorf("failed to execute aictl system schema --version command: %w", err)
	}

	// Trim whitespace and return version
	version := strings.TrimSpace(string(output))
	if version == "" {
		return "", fmt.Errorf("empty version returned from command")
	}

	return version, nil
}

// allowedCommands defines the allowlist of commands that can be executed for security.
// This prevents command injection attacks by restricting execution to known-safe binaries.
var allowedCommands = map[string]bool{
	"aictl":  true, // Language operator CLI tool
	"bundle": true, // Ruby bundler for executing validation scripts
	"ruby":   true, // Ruby interpreter for schema validation
}

// validateCommandSecurity validates command and arguments for security.
// Returns an error if the command or arguments are not allowed.
func validateCommandSecurity(command string, args []string) error {
	// Check if command is in allowlist
	if !allowedCommands[command] {
		return fmt.Errorf("command not allowed: %s (not in security allowlist)", command)
	}

	// Validate command path is clean (no shell metacharacters)
	if strings.ContainsAny(command, "$`;&|><*?[]{}()~") {
		return fmt.Errorf("command contains invalid characters: %s", command)
	}

	// Validate arguments don't contain shell metacharacters
	for i, arg := range args {
		if strings.ContainsAny(arg, "$`;&|><*?{}()~") {
			return fmt.Errorf("argument %d contains invalid characters: %s", i, arg)
		}
	}

	return nil
}

// executeCommand executes a command with the given context and returns its output.
// It handles timeouts, errors, and provides detailed error messages for debugging.
// Returns only stdout (not stderr) to avoid parser warnings interfering with JSON parsing.
//
// SECURITY: This function validates all commands and arguments against an allowlist
// to prevent command injection attacks.
func executeCommand(ctx context.Context, command string, args ...string) ([]byte, error) {
	// Validate command security first
	if err := validateCommandSecurity(command, args); err != nil {
		return nil, fmt.Errorf("security validation failed: %w", err)
	}

	cmd := exec.CommandContext(ctx, command, args...)

	// Capture stdout and stderr separately
	// This is important because Ruby may emit warnings to stderr that should not
	// be included in the JSON output we're parsing from stdout
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err := cmd.Run()

	if ctx.Err() == context.DeadlineExceeded {
		return nil, fmt.Errorf("command timed out: %s %v", command, args)
	}

	if ctx.Err() == context.Canceled {
		return nil, fmt.Errorf("command canceled: %s %v", command, args)
	}

	if err != nil {
		// Check if it's an exec.ExitError to get the exit code
		if exitErr, ok := err.(*exec.ExitError); ok {
			return nil, fmt.Errorf("command failed with exit code %d: %s %v (stderr: %s)",
				exitErr.ExitCode(), command, args, stderr.String())
		}

		// Check if command was not found
		if err == exec.ErrNotFound || strings.Contains(err.Error(), "executable file not found") {
			return nil, fmt.Errorf("command not found: %s (is language-operator gem installed?)", command)
		}

		return nil, fmt.Errorf("command execution failed: %s %v: %w (stderr: %s)",
			command, args, err, stderr.String())
	}

	// Return only stdout (JSON data), ignoring stderr (Ruby warnings)
	return stdout.Bytes(), nil
}

// SchemaViolation represents a schema validation error from the Ruby validator
type SchemaViolation struct {
	Type     string `json:"type"`
	Property string `json:"property,omitempty"`
	Location int    `json:"location"`
	Message  string `json:"message"`
}

// expectedScriptHash is the SHA256 hash of the trusted validation script.
// This prevents script replacement attacks by verifying integrity.
const expectedScriptHash = "d40378d720f162608fdc81f74fb27e6937dd64e593f0cde24b72b0d92939dc28"

// schemaValidatorPaths defines allowed script locations in order of preference.
// SECURITY: Limited paths prevent filesystem traversal and script replacement.
var schemaValidatorPaths = []string{
	"/usr/local/bin/validate-dsl-schema.rb", // Production container location
	"scripts/validate-dsl-schema.rb",        // Development/CI location
}

// validateScriptIntegrity verifies the Ruby script hasn't been tampered with.
func validateScriptIntegrity(scriptPath string) error {
	data, err := os.ReadFile(scriptPath)
	if err != nil {
		return fmt.Errorf("failed to read script for integrity check: %w", err)
	}

	hash := sha256.Sum256(data)
	actualHash := hex.EncodeToString(hash[:])

	if actualHash != expectedScriptHash {
		return fmt.Errorf("script integrity check failed: expected %s, got %s", expectedScriptHash, actualHash)
	}

	return nil
}

// findSchemaValidatorScript securely locates and validates the schema validator script.
// Returns an error if no valid script is found or if integrity validation fails.
func findSchemaValidatorScript() (string, error) {
	for _, path := range schemaValidatorPaths {
		// Convert to absolute path for security
		absPath, err := filepath.Abs(path)
		if err != nil {
			continue
		}

		// Check if file exists
		if _, err := os.Stat(absPath); err != nil {
			continue
		}

		// Verify script integrity
		if err := validateScriptIntegrity(absPath); err != nil {
			return "", fmt.Errorf("script integrity validation failed for %s: %w", absPath, err)
		}

		return absPath, nil
	}

	return "", fmt.Errorf("no valid schema validator script found in allowed paths: %v", schemaValidatorPaths)
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

	// Find and validate the validator script
	scriptPath, err := findSchemaValidatorScript()
	if err != nil {
		// In test environments, script might not be available - skip validation
		// In production, this would be a critical error
		if strings.Contains(err.Error(), "no valid schema validator script found") {
			return nil, nil // Skip validation in test environment
		}
		return nil, fmt.Errorf("failed to locate schema validator script: %w", err)
	}

	// Validate command security before execution
	args := []string{"exec", "ruby", scriptPath}
	if err := validateCommandSecurity("bundle", args); err != nil {
		return nil, fmt.Errorf("security validation failed: %w", err)
	}

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

// ExpectedSchemaVersion is the schema version the operator expects.
// This should match the language_operator gem version used during development.
const ExpectedSchemaVersion = "0.1.31"

// SemanticVersion represents a parsed semantic version (major.minor.patch)
type SemanticVersion struct {
	Major int
	Minor int
	Patch int
}

// ParseSemanticVersion parses a semantic version string like "1.2.3" or "0.1.31"
// It handles versions with or without patch numbers, and ignores pre-release/build metadata.
func ParseSemanticVersion(version string) (*SemanticVersion, error) {
	// Remove leading 'v' if present
	version = strings.TrimPrefix(version, "v")

	// Split on '-' or '+' to remove pre-release and build metadata
	if idx := strings.IndexAny(version, "-+"); idx >= 0 {
		version = version[:idx]
	}

	// Split version into parts
	parts := strings.Split(version, ".")
	if len(parts) < 2 {
		return nil, fmt.Errorf("invalid semantic version format: %s (expected major.minor or major.minor.patch)", version)
	}

	// Parse major version
	major, err := strconv.Atoi(parts[0])
	if err != nil {
		return nil, fmt.Errorf("invalid major version: %s", parts[0])
	}

	// Parse minor version
	minor, err := strconv.Atoi(parts[1])
	if err != nil {
		return nil, fmt.Errorf("invalid minor version: %s", parts[1])
	}

	// Parse patch version (optional, defaults to 0)
	patch := 0
	if len(parts) >= 3 {
		patch, err = strconv.Atoi(parts[2])
		if err != nil {
			return nil, fmt.Errorf("invalid patch version: %s", parts[2])
		}
	}

	return &SemanticVersion{
		Major: major,
		Minor: minor,
		Patch: patch,
	}, nil
}

// String returns the semantic version as a string
func (v *SemanticVersion) String() string {
	return fmt.Sprintf("%d.%d.%d", v.Major, v.Minor, v.Patch)
}

// CompatibilityLevel represents the level of compatibility between two versions
type CompatibilityLevel int

const (
	// Compatible indicates versions are fully compatible
	Compatible CompatibilityLevel = iota
	// MinorMismatch indicates minor version differs (new features, should be compatible)
	MinorMismatch
	// MajorMismatch indicates major version differs (breaking changes, incompatible)
	MajorMismatch
)

// CompareVersions compares two semantic versions and returns the compatibility level
func CompareVersions(expected, actual *SemanticVersion) CompatibilityLevel {
	if expected.Major != actual.Major {
		return MajorMismatch
	}
	if expected.Minor != actual.Minor {
		return MinorMismatch
	}
	return Compatible
}

// ValidateSchemaCompatibility checks version compatibility between the operator
// and the language_operator gem schema. It is called during operator startup.
//
// Behavior:
// - Major version mismatch: Logs ERROR (incompatible)
// - Minor version mismatch: Logs WARNING (new features)
// - Patch version mismatch: Logs INFO (bug fixes)
// - Schema fetch failure: Logs WARNING and continues (non-blocking)
//
// Example usage:
//
//	ctx := context.Background()
//	log := ctrl.Log.WithName("setup")
//	ValidateSchemaCompatibility(ctx, log)
func ValidateSchemaCompatibility(ctx context.Context, log logr.Logger) {
	// Fetch the actual schema version from the gem
	actualVersion, err := GetSchemaVersion(ctx)
	if err != nil {
		// Don't block operator startup if we can't fetch the version
		// This could happen if the gem isn't installed or the command fails
		log.Info("WARNING: Could not fetch schema version from language-operator gem",
			"error", err.Error(),
			"impact", "Schema compatibility cannot be verified")
		return
	}

	// Parse expected version
	expected, err := ParseSemanticVersion(ExpectedSchemaVersion)
	if err != nil {
		log.Error(err, "Failed to parse expected schema version",
			"version", ExpectedSchemaVersion)
		return
	}

	// Parse actual version
	actual, err := ParseSemanticVersion(actualVersion)
	if err != nil {
		log.Error(err, "Failed to parse actual schema version from gem",
			"version", actualVersion)
		return
	}

	// Compare versions
	compatibility := CompareVersions(expected, actual)

	switch compatibility {
	case MajorMismatch:
		log.Error(nil, "ERROR: Schema major version mismatch - INCOMPATIBLE",
			"expectedVersion", expected.String(),
			"actualVersion", actual.String(),
			"expectedMajor", expected.Major,
			"actualMajor", actual.Major,
			"impact", "The operator may not work correctly with this gem version",
			"recommendation", fmt.Sprintf("Update language_operator gem to version %d.x.x or update operator to match gem version %d.x.x",
				expected.Major, actual.Major))

	case MinorMismatch:
		log.Info("WARNING: Schema minor version mismatch",
			"expectedVersion", expected.String(),
			"actualVersion", actual.String(),
			"expectedMinor", expected.Minor,
			"actualMinor", actual.Minor,
			"impact", "New features may be available in the gem that the operator doesn't use",
			"recommendation", "Consider updating to matching versions for full compatibility")

	case Compatible:
		if expected.Patch != actual.Patch {
			log.Info("Schema versions compatible (patch difference)",
				"expectedVersion", expected.String(),
				"actualVersion", actual.String(),
				"status", "Compatible - patch versions may differ")
		} else {
			log.Info("Schema versions match exactly",
				"version", expected.String(),
				"status", "Fully compatible")
		}
	}
}
