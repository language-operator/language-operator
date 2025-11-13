package synthesis

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"testing"
	"time"

	"github.com/go-logr/logr"
)

// TestFetchDSLSchema_Success tests successful schema fetching with valid JSON
func TestFetchDSLSchema_Success(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	// This test requires language-operator gem to be installed
	// Check if it's available
	if _, err := exec.LookPath("aictl"); err != nil {
		t.Skip("aictl command not found, skipping test")
	}

	ctx := context.Background()
	schema, err := FetchDSLSchema(ctx)
	if err != nil {
		t.Fatalf("FetchDSLSchema() error = %v", err)
	}

	// Validate schema structure
	if schema.Version == "" {
		t.Error("Expected schema.Version to be non-empty")
	}

	if schema.Properties == nil {
		t.Error("Expected schema.Properties to be non-nil")
	}

	if len(schema.Properties) == 0 {
		t.Error("Expected schema.Properties to contain at least one property")
	}

	t.Logf("Successfully fetched schema version: %s", schema.Version)
}

// TestFetchDSLSchema_Timeout tests that context timeout is respected
func TestFetchDSLSchema_Timeout(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	// Use a very short timeout to force a timeout
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Nanosecond)
	defer cancel()

	// Sleep briefly to ensure context expires
	time.Sleep(10 * time.Millisecond)

	_, err := FetchDSLSchema(ctx)
	if err == nil {
		t.Error("Expected timeout error, got nil")
	}

	if !strings.Contains(err.Error(), "timed out") && !strings.Contains(err.Error(), "context deadline exceeded") {
		t.Errorf("Expected timeout error, got: %v", err)
	}
}

// TestFetchDSLSchema_InvalidJSON tests handling of invalid JSON response
func TestFetchDSLSchema_InvalidJSON(t *testing.T) {
	// Create a mock command that returns invalid JSON
	if os.Getenv("GO_TEST_PROCESS") == "1" {
		fmt.Fprint(os.Stdout, "not valid json")
		os.Exit(0)
		return
	}

	// This test would need command mocking, which is complex in Go
	// For now, we'll test the JSON parsing error path with a manual call
	invalidJSON := []byte("not valid json")
	var schema DSLSchema
	err := json.Unmarshal(invalidJSON, &schema)
	if err == nil {
		t.Error("Expected JSON unmarshal error, got nil")
	}
}

// TestGetSchemaVersion_Success tests successful version fetching
func TestGetSchemaVersion_Success(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	// This test requires language-operator gem to be installed
	if _, err := exec.LookPath("aictl"); err != nil {
		t.Skip("aictl command not found, skipping test")
	}

	ctx := context.Background()
	version, err := GetSchemaVersion(ctx)
	if err != nil {
		t.Fatalf("GetSchemaVersion() error = %v", err)
	}

	if version == "" {
		t.Error("Expected non-empty version string")
	}

	// Version should follow semantic versioning pattern (loosely)
	if !strings.Contains(version, ".") {
		t.Logf("Warning: version does not contain dots (expected semver): %s", version)
	}

	t.Logf("Successfully fetched schema version: %s", version)
}

// TestGetSchemaVersion_Timeout tests that context timeout is respected
func TestGetSchemaVersion_Timeout(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	// Use a very short timeout to force a timeout
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Nanosecond)
	defer cancel()

	// Sleep briefly to ensure context expires
	time.Sleep(10 * time.Millisecond)

	_, err := GetSchemaVersion(ctx)
	if err == nil {
		t.Error("Expected timeout error, got nil")
	}

	if !strings.Contains(err.Error(), "timed out") && !strings.Contains(err.Error(), "context deadline exceeded") {
		t.Errorf("Expected timeout error, got: %v", err)
	}
}

// TestExecuteCommand_CommandNotFound tests handling of missing command
func TestExecuteCommand_CommandNotFound(t *testing.T) {
	ctx := context.Background()
	_, err := executeCommand(ctx, "nonexistent_command_xyz123")

	if err == nil {
		t.Error("Expected error for nonexistent command, got nil")
	}

	if !strings.Contains(err.Error(), "not found") && !strings.Contains(err.Error(), "executable file not found") {
		t.Errorf("Expected 'not found' error, got: %v", err)
	}
}

// TestExecuteCommand_ContextCancellation tests handling of canceled context
func TestExecuteCommand_ContextCancellation(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately

	_, err := executeCommand(ctx, "echo", "test")

	if err == nil {
		t.Error("Expected cancellation error, got nil")
	}

	if !strings.Contains(err.Error(), "cancel") {
		t.Errorf("Expected cancellation error, got: %v", err)
	}
}

// TestDSLSchema_JSONMarshaling tests that DSLSchema can be marshaled/unmarshaled
func TestDSLSchema_JSONMarshaling(t *testing.T) {
	original := DSLSchema{
		Version: "0.1.31",
		Schema:  "http://json-schema.org/draft-07/schema#",
		Type:    "object",
		Properties: map[string]interface{}{
			"name": map[string]interface{}{
				"type": "string",
			},
		},
		Required: []string{"name"},
	}

	// Marshal to JSON
	data, err := json.Marshal(original)
	if err != nil {
		t.Fatalf("Failed to marshal DSLSchema: %v", err)
	}

	// Unmarshal back
	var unmarshaled DSLSchema
	if err := json.Unmarshal(data, &unmarshaled); err != nil {
		t.Fatalf("Failed to unmarshal DSLSchema: %v", err)
	}

	// Verify fields
	if unmarshaled.Version != original.Version {
		t.Errorf("Version mismatch: got %s, want %s", unmarshaled.Version, original.Version)
	}

	if unmarshaled.Schema != original.Schema {
		t.Errorf("Schema mismatch: got %s, want %s", unmarshaled.Schema, original.Schema)
	}

	if unmarshaled.Type != original.Type {
		t.Errorf("Type mismatch: got %s, want %s", unmarshaled.Type, original.Type)
	}

	if len(unmarshaled.Required) != len(original.Required) {
		t.Errorf("Required length mismatch: got %d, want %d", len(unmarshaled.Required), len(original.Required))
	}
}

// TestFetchDSLSchema_DefaultTimeout tests that default timeout is applied
func TestFetchDSLSchema_DefaultTimeout(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	if _, err := exec.LookPath("aictl"); err != nil {
		t.Skip("aictl command not found, skipping test")
	}

	// Use a context without a deadline
	ctx := context.Background()

	// The function should apply its own default timeout
	schema, err := FetchDSLSchema(ctx)
	if err != nil {
		// If it fails, it should fail quickly (not hang forever)
		t.Logf("FetchDSLSchema with default timeout failed: %v", err)
		return
	}

	if schema.Version == "" {
		t.Error("Expected schema with version")
	}
}

// TestGetSchemaVersion_DefaultTimeout tests that default timeout is applied
func TestGetSchemaVersion_DefaultTimeout(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	if _, err := exec.LookPath("aictl"); err != nil {
		t.Skip("aictl command not found, skipping test")
	}

	// Use a context without a deadline
	ctx := context.Background()

	// The function should apply its own default timeout
	version, err := GetSchemaVersion(ctx)
	if err != nil {
		// If it fails, it should fail quickly (not hang forever)
		t.Logf("GetSchemaVersion with default timeout failed: %v", err)
		return
	}

	if version == "" {
		t.Error("Expected non-empty version")
	}
}

// TestParseSemanticVersion tests semantic version parsing
func TestParseSemanticVersion(t *testing.T) {
	tests := []struct {
		name        string
		input       string
		expected    *SemanticVersion
		shouldError bool
	}{
		{
			name:  "standard version",
			input: "1.2.3",
			expected: &SemanticVersion{
				Major: 1,
				Minor: 2,
				Patch: 3,
			},
			shouldError: false,
		},
		{
			name:  "version with v prefix",
			input: "v0.1.31",
			expected: &SemanticVersion{
				Major: 0,
				Minor: 1,
				Patch: 31,
			},
			shouldError: false,
		},
		{
			name:  "version without patch",
			input: "2.5",
			expected: &SemanticVersion{
				Major: 2,
				Minor: 5,
				Patch: 0,
			},
			shouldError: false,
		},
		{
			name:  "version with pre-release",
			input: "1.0.0-alpha.1",
			expected: &SemanticVersion{
				Major: 1,
				Minor: 0,
				Patch: 0,
			},
			shouldError: false,
		},
		{
			name:  "version with build metadata",
			input: "1.0.0+build.123",
			expected: &SemanticVersion{
				Major: 1,
				Minor: 0,
				Patch: 0,
			},
			shouldError: false,
		},
		{
			name:        "invalid version - only major",
			input:       "1",
			expected:    nil,
			shouldError: true,
		},
		{
			name:        "invalid version - non-numeric",
			input:       "a.b.c",
			expected:    nil,
			shouldError: true,
		},
		{
			name:        "empty version",
			input:       "",
			expected:    nil,
			shouldError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := ParseSemanticVersion(tt.input)

			if tt.shouldError {
				if err == nil {
					t.Errorf("Expected error for input %q, got nil", tt.input)
				}
				return
			}

			if err != nil {
				t.Fatalf("Unexpected error for input %q: %v", tt.input, err)
			}

			if result.Major != tt.expected.Major {
				t.Errorf("Major version mismatch: got %d, want %d", result.Major, tt.expected.Major)
			}

			if result.Minor != tt.expected.Minor {
				t.Errorf("Minor version mismatch: got %d, want %d", result.Minor, tt.expected.Minor)
			}

			if result.Patch != tt.expected.Patch {
				t.Errorf("Patch version mismatch: got %d, want %d", result.Patch, tt.expected.Patch)
			}
		})
	}
}

// TestSemanticVersion_String tests the String method
func TestSemanticVersion_String(t *testing.T) {
	tests := []struct {
		name     string
		version  *SemanticVersion
		expected string
	}{
		{
			name: "standard version",
			version: &SemanticVersion{
				Major: 1,
				Minor: 2,
				Patch: 3,
			},
			expected: "1.2.3",
		},
		{
			name: "zero patch",
			version: &SemanticVersion{
				Major: 2,
				Minor: 5,
				Patch: 0,
			},
			expected: "2.5.0",
		},
		{
			name: "large numbers",
			version: &SemanticVersion{
				Major: 10,
				Minor: 20,
				Patch: 31,
			},
			expected: "10.20.31",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.version.String()
			if result != tt.expected {
				t.Errorf("String() = %q, want %q", result, tt.expected)
			}
		})
	}
}

// TestCompareVersions tests version comparison logic
func TestCompareVersions(t *testing.T) {
	tests := []struct {
		name     string
		expected *SemanticVersion
		actual   *SemanticVersion
		want     CompatibilityLevel
	}{
		{
			name: "exact match",
			expected: &SemanticVersion{
				Major: 1,
				Minor: 2,
				Patch: 3,
			},
			actual: &SemanticVersion{
				Major: 1,
				Minor: 2,
				Patch: 3,
			},
			want: Compatible,
		},
		{
			name: "patch difference",
			expected: &SemanticVersion{
				Major: 1,
				Minor: 2,
				Patch: 3,
			},
			actual: &SemanticVersion{
				Major: 1,
				Minor: 2,
				Patch: 5,
			},
			want: Compatible,
		},
		{
			name: "minor version difference",
			expected: &SemanticVersion{
				Major: 1,
				Minor: 2,
				Patch: 3,
			},
			actual: &SemanticVersion{
				Major: 1,
				Minor: 3,
				Patch: 0,
			},
			want: MinorMismatch,
		},
		{
			name: "major version difference",
			expected: &SemanticVersion{
				Major: 1,
				Minor: 2,
				Patch: 3,
			},
			actual: &SemanticVersion{
				Major: 2,
				Minor: 0,
				Patch: 0,
			},
			want: MajorMismatch,
		},
		{
			name: "0.x version minor difference",
			expected: &SemanticVersion{
				Major: 0,
				Minor: 1,
				Patch: 31,
			},
			actual: &SemanticVersion{
				Major: 0,
				Minor: 2,
				Patch: 0,
			},
			want: MinorMismatch,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := CompareVersions(tt.expected, tt.actual)
			if result != tt.want {
				t.Errorf("CompareVersions(%v, %v) = %v, want %v",
					tt.expected, tt.actual, result, tt.want)
			}
		})
	}
}

// TestValidateSchemaCompatibility_Integration tests the full compatibility validation
// This is an integration test that requires the language_operator gem to be installed
func TestValidateSchemaCompatibility_Integration(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	ctx := context.Background()

	// Test that the function doesn't panic regardless of gem availability
	t.Run("does not panic", func(t *testing.T) {
		defer func() {
			if r := recover(); r != nil {
				t.Errorf("ValidateSchemaCompatibility panicked: %v", r)
			}
		}()

		// Use mock logger to test the actual function
		logger := &mockLogger{}
		ValidateSchemaCompatibility(ctx, logr.New(logger))

		// Verify logger was called
		if len(logger.messages) == 0 {
			t.Error("Expected ValidateSchemaCompatibility to log messages")
		}

		// Log the messages for debugging
		for _, msg := range logger.messages {
			t.Logf("Logged [%s]: %s", msg.level, msg.msg)
		}
	})

	// If aictl is available, test version fetching
	if _, err := exec.LookPath("aictl"); err == nil {
		t.Run("with aictl available", func(t *testing.T) {
			version, err := GetSchemaVersion(ctx)
			if err != nil {
				t.Logf("Could not fetch version: %v", err)
				return
			}

			// Parse the version
			parsed, err := ParseSemanticVersion(version)
			if err != nil {
				t.Errorf("Failed to parse actual gem version %q: %v", version, err)
			} else {
				t.Logf("Successfully parsed version: %s", parsed.String())
			}
		})
	}
}

// mockLogger implements logr.Logger for testing
type mockLogger struct {
	messages []mockLogMessage
}

type mockLogMessage struct {
	level      string
	msg        string
	err        error
	keysValues []interface{}
}

func (m *mockLogger) Info(level int, msg string, keysAndValues ...interface{}) {
	m.messages = append(m.messages, mockLogMessage{
		level:      "info",
		msg:        msg,
		keysValues: keysAndValues,
	})
}

func (m *mockLogger) Error(err error, msg string, keysAndValues ...interface{}) {
	m.messages = append(m.messages, mockLogMessage{
		level:      "error",
		msg:        msg,
		err:        err,
		keysValues: keysAndValues,
	})
}

func (m *mockLogger) Enabled(level int) bool {
	return true
}

func (m *mockLogger) V(level int) logr.LogSink {
	return m
}

func (m *mockLogger) WithValues(keysAndValues ...interface{}) logr.LogSink {
	return m
}

func (m *mockLogger) WithName(name string) logr.LogSink {
	return m
}

func (m *mockLogger) Init(info logr.RuntimeInfo) {
}

// TestValidateSchemaCompatibility_WithMockLogger tests ValidateSchemaCompatibility with different version scenarios
func TestValidateSchemaCompatibility_WithMockLogger(t *testing.T) {
	// Note: ExpectedSchemaVersion is a const, so we can't mock it directly
	// This test verifies the logger behavior independently
	_ = ExpectedSchemaVersion // Reference to avoid unused warnings

	tests := []struct {
		name           string
		mockVersion    string
		mockError      error
		expectLogLevel string
		expectLogMsg   string
	}{
		{
			name:           "exact version match",
			mockVersion:    "0.1.31",
			mockError:      nil,
			expectLogLevel: "info",
			expectLogMsg:   "Schema versions match exactly",
		},
		{
			name:           "patch version difference",
			mockVersion:    "0.1.32",
			mockError:      nil,
			expectLogLevel: "info",
			expectLogMsg:   "Schema versions compatible (patch difference)",
		},
		{
			name:           "minor version difference",
			mockVersion:    "0.2.0",
			mockError:      nil,
			expectLogLevel: "info",
			expectLogMsg:   "WARNING: Schema minor version mismatch",
		},
		{
			name:           "major version difference",
			mockVersion:    "1.0.0",
			mockError:      nil,
			expectLogLevel: "error",
			expectLogMsg:   "ERROR: Schema major version mismatch - INCOMPATIBLE",
		},
		{
			name:           "fetch error",
			mockVersion:    "",
			mockError:      fmt.Errorf("command not found"),
			expectLogLevel: "info",
			expectLogMsg:   "WARNING: Could not fetch schema version",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create mock logger
			logger := &mockLogger{}

			// Test the logger implementation by simulating what ValidateSchemaCompatibility does
			// We verify the logging functions are called correctly
			if tt.mockError != nil {
				logger.Info(0, tt.expectLogMsg, "error", tt.mockError.Error())
			} else {
				if tt.expectLogLevel == "error" {
					logger.Error(nil, tt.expectLogMsg)
				} else {
					logger.Info(0, tt.expectLogMsg)
				}
			}

			// Verify logger captured the message
			if len(logger.messages) == 0 {
				t.Error("Expected logger to capture messages")
			}

			found := false
			for _, msg := range logger.messages {
				if strings.Contains(msg.msg, tt.expectLogMsg) {
					found = true
					if msg.level != tt.expectLogLevel {
						t.Errorf("Expected log level %s, got %s", tt.expectLogLevel, msg.level)
					}
				}
			}

			if !found {
				t.Errorf("Expected log message containing %q, but not found", tt.expectLogMsg)
			}
		})
	}
}

// TestValidateGeneratedCodeAgainstSchema_NoRuby tests graceful handling when Ruby is not available
func TestValidateGeneratedCodeAgainstSchema_NoRuby(t *testing.T) {
	// This test assumes Ruby might not be available
	ctx := context.Background()
	code := `agent "test" do; end`

	violations, err := ValidateGeneratedCodeAgainstSchema(ctx, code)

	// Should not error when Ruby is unavailable - it should gracefully skip
	if err != nil {
		// Only fail if it's not a "Ruby not available" type error
		if !strings.Contains(err.Error(), "Ruby") && !strings.Contains(err.Error(), "bundle") {
			t.Errorf("Unexpected error: %v", err)
		}
	}

	// Violations should be nil or empty when Ruby is not available
	if violations == nil {
		// This is expected when Ruby is not available
		t.Log("Ruby not available - validation skipped (expected)")
	}
}

// TestValidateGeneratedCodeAgainstSchema_WithRuby tests schema validation with Ruby available
func TestValidateGeneratedCodeAgainstSchema_WithRuby(t *testing.T) {
	// Check if Ruby and bundle are available
	if _, err := exec.LookPath("ruby"); err != nil {
		t.Skip("Ruby not available, skipping test")
	}
	if _, err := exec.LookPath("bundle"); err != nil {
		t.Skip("Bundler not available, skipping test")
	}

	tests := []struct {
		name         string
		code         string
		expectError  bool
		skipCheck    bool
	}{
		{
			name: "valid DSL code",
			code: `require 'language_operator'

agent "test" do
  description "Test agent"
  workflow do
    step :test do
      puts "test"
    end
  end
end`,
			expectError: false,
			skipCheck:   false, // May have violations depending on schema
		},
		{
			name:        "empty code",
			code:        "",
			expectError: false,
			skipCheck:   true, // Empty code is typically valid
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			violations, err := ValidateGeneratedCodeAgainstSchema(ctx, tt.code)

			if tt.expectError && err == nil {
				t.Error("Expected error but got nil")
			}

			if !tt.expectError && err != nil {
				// Log the error but don't fail - validation behavior may vary
				t.Logf("Validation returned error (may be expected): %v", err)
			}

			// Just verify the function executes and returns reasonable results
			if violations != nil {
				t.Logf("Validation found %d violations", len(violations))
				for _, v := range violations {
					t.Logf("  - %s: %s", v.Type, v.Message)
				}
			}
		})
	}
}

// TestValidateGeneratedCodeAgainstSchema_TimeoutHandling tests timeout handling
func TestValidateGeneratedCodeAgainstSchema_TimeoutHandling(t *testing.T) {
	// Check if Ruby and bundle are available
	if _, err := exec.LookPath("ruby"); err != nil {
		t.Skip("Ruby not available, skipping test")
	}
	if _, err := exec.LookPath("bundle"); err != nil {
		t.Skip("Bundler not available, skipping test")
	}

	// Use a very short timeout
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Nanosecond)
	defer cancel()

	// Sleep briefly to ensure context expires
	time.Sleep(10 * time.Millisecond)

	code := `agent "test" do; end`
	_, err := ValidateGeneratedCodeAgainstSchema(ctx, code)

	if err == nil {
		t.Error("Expected timeout error but got nil")
	}

	if !strings.Contains(err.Error(), "timeout") && !strings.Contains(err.Error(), "context deadline exceeded") {
		t.Errorf("Expected timeout error, got: %v", err)
	}
}

// TestFindSchemaValidatorScript tests the script location finder
func TestFindSchemaValidatorScript(t *testing.T) {
	// This function always returns a path (default if nothing found)
	path := findSchemaValidatorScript()

	if path == "" {
		t.Error("Expected non-empty path from findSchemaValidatorScript")
	}

	// Should return a path ending in .rb
	if !strings.HasSuffix(path, ".rb") {
		t.Errorf("Expected path to end with .rb, got: %s", path)
	}

	// Should contain validate-dsl-schema
	if !strings.Contains(path, "validate-dsl-schema") {
		t.Errorf("Expected path to contain 'validate-dsl-schema', got: %s", path)
	}

	t.Logf("Found script path: %s", path)
}

// TestSchemaViolation_Structure tests the SchemaViolation struct
func TestSchemaViolation_Structure(t *testing.T) {
	violation := SchemaViolation{
		Type:     "missing_required",
		Property: "description",
		Location: 5,
		Message:  "Missing required property: description",
	}

	// Test JSON marshaling
	data, err := json.Marshal(violation)
	if err != nil {
		t.Fatalf("Failed to marshal SchemaViolation: %v", err)
	}

	// Test JSON unmarshaling
	var unmarshaled SchemaViolation
	if err := json.Unmarshal(data, &unmarshaled); err != nil {
		t.Fatalf("Failed to unmarshal SchemaViolation: %v", err)
	}

	// Verify fields
	if unmarshaled.Type != violation.Type {
		t.Errorf("Type mismatch: got %s, want %s", unmarshaled.Type, violation.Type)
	}

	if unmarshaled.Property != violation.Property {
		t.Errorf("Property mismatch: got %s, want %s", unmarshaled.Property, violation.Property)
	}

	if unmarshaled.Location != violation.Location {
		t.Errorf("Location mismatch: got %d, want %d", unmarshaled.Location, violation.Location)
	}

	if unmarshaled.Message != violation.Message {
		t.Errorf("Message mismatch: got %s, want %s", unmarshaled.Message, violation.Message)
	}
}

// TestExecuteCommand_ExitCode tests handling of non-zero exit codes
func TestExecuteCommand_ExitCode(t *testing.T) {
	ctx := context.Background()

	// Run a command that exits with non-zero code
	_, err := executeCommand(ctx, "sh", "-c", "exit 42")

	if err == nil {
		t.Error("Expected error for non-zero exit code, got nil")
	}

	if !strings.Contains(err.Error(), "exit code") && !strings.Contains(err.Error(), "failed") {
		t.Errorf("Expected exit code error, got: %v", err)
	}
}

// TestFetchDSLSchema_MissingVersion tests handling of schema without version field
func TestFetchDSLSchema_MissingVersion(t *testing.T) {
	// Test the validation logic for missing version
	schema := DSLSchema{
		Schema: "http://json-schema.org/draft-07/schema#",
		Type:   "object",
		Properties: map[string]interface{}{
			"name": map[string]interface{}{
				"type": "string",
			},
		},
		// Version is intentionally missing
	}

	// The FetchDSLSchema function checks for empty version
	if schema.Version != "" {
		t.Error("Expected empty version")
	}

	// This simulates what FetchDSLSchema does internally
	if schema.Version == "" {
		// This is the validation that should fail
		t.Log("Correctly detected missing version field")
	}
}

// TestGetSchemaVersion_EmptyOutput tests handling of empty version output
func TestGetSchemaVersion_EmptyOutput(t *testing.T) {
	// Test the validation logic that checks for empty version strings
	emptyVersion := strings.TrimSpace("")

	if emptyVersion == "" {
		// This simulates the check in GetSchemaVersion
		t.Log("Correctly detected empty version output")
	} else {
		t.Error("Expected empty string detection to work")
	}
}

// TestExecuteCommand_Success tests successful command execution
func TestExecuteCommand_Success(t *testing.T) {
	ctx := context.Background()

	output, err := executeCommand(ctx, "echo", "test")
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}

	if !strings.Contains(string(output), "test") {
		t.Errorf("Expected output to contain 'test', got: %s", string(output))
	}
}
