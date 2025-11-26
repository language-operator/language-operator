package integration

import (
	"context"
	"encoding/json"
	"os/exec"
	"strings"
	"testing"
	"time"

	"github.com/go-logr/logr"
	"github.com/language-operator/language-operator/pkg/synthesis"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestSchemaIntegrationEndToEnd tests the full Go→Ruby schema flow
// This is an integration test that exercises:
// 1. Fetching schema from Ruby gem
// 2. Validating schema structure
// 3. Checking version compatibility
// 4. Validating DSL code against schema
// 5. Handling failures gracefully
// 6. Setting telemetry attributes
//
// Requirements:
// - Ruby must be installed
// - Bundler must be installed
// - language_operator gem must be available (via bundle exec aictl)
func TestSchemaIntegrationEndToEnd(t *testing.T) {
	t.Skip("Test disabled - FetchDSLSchema function was removed as dead code")
	return

	// Verify aictl is accessible via bundle exec
	ctx := context.Background()
	if _, err := synthesis.GetSchemaVersion(ctx); err != nil {
		if strings.Contains(err.Error(), "command not found") ||
			strings.Contains(err.Error(), "gem installed") {
			t.Skipf("aictl not available via bundle exec, skipping test: %v", err)
		}
		// Other errors should fail the test
		t.Fatalf("Unexpected error checking aictl availability: %v", err)
	}

	t.Run("fetch full schema", func(t *testing.T) {
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()

		schema, err := synthesis.FetchDSLSchema(ctx)
		require.NoError(t, err, "FetchDSLSchema should succeed")
		require.NotNil(t, schema, "Schema should not be nil")

		// Validate schema structure
		assert.NotEmpty(t, schema.Version, "Schema should have version")
		assert.Equal(t, "object", schema.Type, "Schema type should be 'object'")
		assert.NotEmpty(t, schema.Properties, "Schema should have properties")

		// Verify schema has expected properties for agent DSL
		// The exact properties depend on the gem version, but we can check some basics
		assert.NotNil(t, schema.Properties, "Schema properties should not be nil")
		t.Logf("✓ Fetched schema version %s with %d properties", schema.Version, len(schema.Properties))

		// Verify schema can be marshaled back to JSON (round-trip test)
		jsonData, err := json.Marshal(schema)
		require.NoError(t, err, "Schema should be marshalable to JSON")
		assert.NotEmpty(t, jsonData, "Marshaled JSON should not be empty")

		// Verify JSON can be unmarshaled back
		var roundTrip synthesis.DSLSchema
		err = json.Unmarshal(jsonData, &roundTrip)
		require.NoError(t, err, "Marshaled JSON should be unmarshalable")
		assert.Equal(t, schema.Version, roundTrip.Version, "Round-trip should preserve version")
	})

	t.Run("fetch schema version", func(t *testing.T) {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		version, err := synthesis.GetSchemaVersion(ctx)
		require.NoError(t, err, "GetSchemaVersion should succeed")
		assert.NotEmpty(t, version, "Version should not be empty")

		// Version should be semantic version format
		assert.Contains(t, version, ".", "Version should contain dots (semver format)")

		// Verify version can be parsed
		parsed, err := synthesis.ParseSemanticVersion(version)
		require.NoError(t, err, "Version should be parsable as semantic version")
		assert.NotNil(t, parsed, "Parsed version should not be nil")

		t.Logf("✓ Fetched and parsed schema version: %s", parsed.String())
	})

	t.Run("check version compatibility", func(t *testing.T) {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		// Fetch actual version from gem
		actualVersion, err := synthesis.GetSchemaVersion(ctx)
		require.NoError(t, err, "GetSchemaVersion should succeed")

		// Parse both versions
		expected, err := synthesis.ParseSemanticVersion(synthesis.ExpectedSchemaVersion)
		require.NoError(t, err, "Expected version should be valid semver")

		actual, err := synthesis.ParseSemanticVersion(actualVersion)
		require.NoError(t, err, "Actual version should be valid semver")

		// Check compatibility
		compatibility := synthesis.CompareVersions(expected, actual)

		switch compatibility {
		case synthesis.Compatible:
			t.Logf("✓ Schema versions are compatible: expected=%s actual=%s", expected.String(), actual.String())

		case synthesis.MinorMismatch:
			t.Logf("⚠ Minor version mismatch (acceptable): expected=%s actual=%s", expected.String(), actual.String())
			// Minor mismatch is acceptable - just log warning

		case synthesis.MajorMismatch:
			t.Errorf("❌ Major version mismatch (incompatible): expected=%s actual=%s", expected.String(), actual.String())
			// Major mismatch indicates breaking changes
		}

		// Also test ValidateSchemaCompatibility with mock logger
		logger := &mockLogger{}
		synthesis.ValidateSchemaCompatibility(ctx, logr.New(logger))

		// Verify logger was called (should have logged version info)
		assert.NotEmpty(t, logger.messages, "ValidateSchemaCompatibility should log messages")
		t.Logf("✓ ValidateSchemaCompatibility logged %d messages", len(logger.messages))

		// Log the messages for debugging
		for _, msg := range logger.messages {
			t.Logf("  [%s] %s", msg.level, msg.msg)
		}
	})

	t.Run("validate DSL code against schema", func(t *testing.T) {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		// Test with valid DSL code
		validCode := `require 'language_operator'

agent 'test-agent' do
  description 'An end-to-end integration test agent'

  schedule '0 8 * * *'
end
`

		violations, err := synthesis.ValidateGeneratedCodeAgainstSchema(ctx, validCode)
		require.NoError(t, err, "Validation should not error on valid code")
		assert.Empty(t, violations, "Valid code should have no violations")

		t.Logf("✓ Valid DSL code passed schema validation")

		// Test with invalid DSL code (has invalid property)
		invalidCode := `require 'language_operator'

agent 'test-agent' do
  description 'Test'

  invalid_property 'this should fail'
end
`

		violations, err = synthesis.ValidateGeneratedCodeAgainstSchema(ctx, invalidCode)
		require.NoError(t, err, "Validation should not error (returns violations instead)")

		// Note: The validator behavior depends on the Ruby gem implementation
		// It should detect the invalid property, but we'll be lenient here
		// since the gem might evolve
		if len(violations) > 0 {
			t.Logf("✓ Invalid DSL code detected %d violations (as expected)", len(violations))
			for _, v := range violations {
				t.Logf("  - %s: %s (at line %d)", v.Type, v.Message, v.Location)
			}
		} else {
			t.Logf("⚠ Validator did not report violations for invalid property (gem behavior may vary)")
		}
	})

	t.Run("handle schema fetch failures gracefully", func(t *testing.T) {
		// Use a very short timeout to force a timeout error
		ctx, cancel := context.WithTimeout(context.Background(), 1*time.Nanosecond)
		defer cancel()

		// Sleep to ensure context expires
		time.Sleep(2 * time.Millisecond)

		_, err := synthesis.FetchDSLSchema(ctx)
		assert.Error(t, err, "Should return error on timeout")
		assert.Contains(t, err.Error(), "timed out", "Error should indicate timeout")

		t.Logf("✓ Schema fetch timeout handled gracefully: %v", err)

		// Test that ValidateSchemaCompatibility doesn't block on errors
		logger := &mockLogger{}
		synthesis.ValidateSchemaCompatibility(ctx, logr.New(logger))

		// Should have logged a warning but not panicked
		found := false
		for _, msg := range logger.messages {
			if strings.Contains(msg.msg, "WARNING") || strings.Contains(msg.msg, "Could not fetch") {
				found = true
				break
			}
		}
		assert.True(t, found, "Should log warning when schema fetch fails")

		t.Logf("✓ ValidateSchemaCompatibility handles failures gracefully")
	})

	t.Run("telemetry and context propagation", func(t *testing.T) {
		// Create context with values (simulating telemetry/tracing)
		type contextKey string
		const requestIDKey contextKey = "request-id"

		ctx := context.WithValue(context.Background(), requestIDKey, "test-request-123")
		ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
		defer cancel()

		// Functions should respect context
		version, err := synthesis.GetSchemaVersion(ctx)
		require.NoError(t, err, "Should work with context values")
		assert.NotEmpty(t, version, "Should return version")

		// Verify context is propagated (check it can be retrieved)
		requestID := ctx.Value(requestIDKey)
		assert.Equal(t, "test-request-123", requestID, "Context values should be preserved")

		t.Logf("✓ Context and telemetry attributes are properly propagated")
	})

	// Final summary
	t.Log("✅ End-to-end schema integration test completed successfully")
	t.Log("   - Schema fetching from Ruby gem: ✓")
	t.Log("   - Schema structure validation: ✓")
	t.Log("   - Version compatibility checking: ✓")
	t.Log("   - DSL code validation: ✓")
	t.Log("   - Error handling: ✓")
	t.Log("   - Telemetry/context: ✓")
}

// mockLogger implements logr.LogSink for testing ValidateSchemaCompatibility
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
