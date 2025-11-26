package integration

import (
	"testing"

	"github.com/go-logr/logr"
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