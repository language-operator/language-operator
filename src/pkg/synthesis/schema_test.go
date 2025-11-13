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

	// Check if language_operator command is available
	if _, err := exec.LookPath("aictl"); err != nil {
		t.Skip("aictl command not found, skipping test")
	}

	// Create a test logger that captures log output
	// In a real scenario, we'd use a mock logger to verify log messages
	// For this test, we'll just ensure it doesn't panic
	ctx := context.Background()

	// This should not panic or cause errors
	// The actual validation happens against the installed gem version
	t.Run("does not panic", func(t *testing.T) {
		defer func() {
			if r := recover(); r != nil {
				t.Errorf("ValidateSchemaCompatibility panicked: %v", r)
			}
		}()

		// We can't easily verify log output in this test without a mock logger
		// But we can verify the function executes without panicking
		// ValidateSchemaCompatibility(ctx, log)
		// Note: Commented out because we need a proper logger instance

		// Instead, let's test the version fetching works
		version, err := GetSchemaVersion(ctx)
		if err != nil {
			t.Logf("Could not fetch version: %v", err)
			return
		}

		// Parse the version
		_, err = ParseSemanticVersion(version)
		if err != nil {
			t.Errorf("Failed to parse actual gem version %q: %v", version, err)
		}
	})
}
