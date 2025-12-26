/*
Copyright 2025 Langop Team.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package adapters

import (
	"context"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/language-operator/language-operator/pkg/telemetry"
)

func TestNewClickhouseAdapter(t *testing.T) {
	tests := []struct {
		name          string
		envVars       map[string]string
		expectError   bool
		expectedError string
	}{
		{
			name:          "missing endpoint",
			envVars:       map[string]string{},
			expectError:   true,
			expectedError: "ClickHouse adapter requires TELEMETRY_ADAPTER_ENDPOINT",
		},
		{
			name: "valid configuration with defaults",
			envVars: map[string]string{
				"TELEMETRY_ADAPTER_ENDPOINT": "http://localhost:8123",
			},
			expectError: true, // Will fail to connect to non-existent DB
		},
		{
			name: "valid configuration with custom values",
			envVars: map[string]string{
				"TELEMETRY_ADAPTER_ENDPOINT":       "http://clickhouse:8123",
				"TELEMETRY_ADAPTER_DATABASE":       "test_db",
				"TELEMETRY_ADAPTER_USERNAME":       "test_user",
				"TELEMETRY_ADAPTER_PASSWORD":       "test_pass",
				"TELEMETRY_ADAPTER_TIMEOUT":        "10s",
				"TELEMETRY_ADAPTER_RETRY_ATTEMPTS": "5",
				"TELEMETRY_ADAPTER_RETRY_BACKOFF":  "2s",
			},
			expectError: true, // Will fail to connect to non-existent DB
		},
		{
			name: "invalid timeout",
			envVars: map[string]string{
				"TELEMETRY_ADAPTER_ENDPOINT": "http://localhost:8123",
				"TELEMETRY_ADAPTER_TIMEOUT":  "invalid",
			},
			expectError:   true,
			expectedError: "invalid timeout",
		},
		{
			name: "invalid retry attempts",
			envVars: map[string]string{
				"TELEMETRY_ADAPTER_ENDPOINT":       "http://localhost:8123",
				"TELEMETRY_ADAPTER_RETRY_ATTEMPTS": "invalid",
			},
			expectError:   true,
			expectedError: "invalid retry attempts",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Clear environment
			clearEnv()

			// Set test environment variables
			for key, value := range tt.envVars {
				os.Setenv(key, value)
			}

			adapter, err := NewClickhouseAdapter()

			if tt.expectError {
				if err == nil {
					t.Errorf("Expected error, but got none")
				}
				if tt.expectedError != "" && err != nil {
					if !contains(err.Error(), tt.expectedError) {
						t.Errorf("Expected error containing %q, got %q", tt.expectedError, err.Error())
					}
				}
			} else {
				if err != nil {
					t.Errorf("Unexpected error: %v", err)
				}
				if adapter == nil {
					t.Errorf("Expected adapter to be created")
				}
			}

			if adapter != nil {
				adapter.Close()
			}
		})
	}

	// Clean up
	clearEnv()
}

func TestClickhouseAdapter_buildDSN(t *testing.T) {
	adapter := &ClickhouseAdapter{
		endpoint: "http://clickhouse:8123",
		database: "otel",
		username: "default",
		password: "secret",
		timeout:  30 * time.Second,
	}

	expected := "tcp://clickhouse:9000?database=otel&username=default&password=secret&read_timeout=30&write_timeout=30"
	actual := adapter.buildDSN()

	if actual != expected {
		t.Errorf("Expected DSN %q, got %q", expected, actual)
	}
}

func TestClickhouseAdapter_buildSpanQuery(t *testing.T) {
	adapter := &ClickhouseAdapter{}

	tests := []struct {
		name     string
		filter   telemetry.SpanFilter
		expected string
	}{
		{
			name:   "basic query",
			filter: telemetry.SpanFilter{},
			expected: `
		SELECT 
			SpanId,
			TraceId,
			ParentSpanId,
			SpanName,
			SpanKind,
			Timestamp,
			Duration,
			StatusCode,
			StatusMessage,
			SpanAttributes
		FROM otel_traces
		WHERE 1=1 ORDER BY Timestamp DESC`,
		},
		{
			name: "with task name",
			filter: telemetry.SpanFilter{
				TaskName: "test_task",
			},
			expected: `
		SELECT 
			SpanId,
			TraceId,
			ParentSpanId,
			SpanName,
			SpanKind,
			Timestamp,
			Duration,
			StatusCode,
			StatusMessage,
			SpanAttributes
		FROM otel_traces
		WHERE 1=1 AND (SpanAttributes['task.name'] = ? OR SpanName LIKE ?) ORDER BY Timestamp DESC`,
		},
		{
			name: "with limit",
			filter: telemetry.SpanFilter{
				Limit: 100,
			},
			expected: `
		SELECT 
			SpanId,
			TraceId,
			ParentSpanId,
			SpanName,
			SpanKind,
			Timestamp,
			Duration,
			StatusCode,
			StatusMessage,
			SpanAttributes
		FROM otel_traces
		WHERE 1=1 ORDER BY Timestamp DESC LIMIT ?`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			query := adapter.buildSpanQuery(tt.filter)
			query = normalizeWhitespace(query)
			expected := normalizeWhitespace(tt.expected)

			if query != expected {
				t.Errorf("Expected query:\n%s\nGot:\n%s", expected, query)
			}
		})
	}
}

func TestClickhouseAdapter_buildSpanArgs(t *testing.T) {
	adapter := &ClickhouseAdapter{}

	start := time.Date(2023, 1, 1, 0, 0, 0, 0, time.UTC)
	end := time.Date(2023, 1, 2, 0, 0, 0, 0, time.UTC)

	filter := telemetry.SpanFilter{
		TaskName: "test_task",
		TimeRange: telemetry.TimeRange{
			Start: start,
			End:   end,
		},
		TraceID: "trace123",
		Attributes: map[string]string{
			"service.name": "test-service",
		},
		Limit: 50,
	}

	args := adapter.buildSpanArgs(filter)

	expectedArgs := []interface{}{
		start,                    // TimeRange.Start
		end,                      // TimeRange.End
		"test_task",              // TaskName
		"%test_task%",            // TaskName (LIKE pattern)
		"trace123",               // TraceID
		"test-service",           // Attributes["service.name"]
		50,                       // Limit
	}

	if len(args) != len(expectedArgs) {
		t.Errorf("Expected %d args, got %d", len(expectedArgs), len(args))
		return
	}

	for i, expected := range expectedArgs {
		if args[i] != expected {
			t.Errorf("Arg %d: expected %v, got %v", i, expected, args[i])
		}
	}
}

func TestClickhouseAdapter_Available(t *testing.T) {
	adapter := &ClickhouseAdapter{
		healthy:   true,
		lastCheck: time.Now(),
	}

	if !adapter.Available() {
		t.Errorf("Expected adapter to be available")
	}

	adapter.healthy = false
	if adapter.Available() {
		t.Errorf("Expected adapter to be unavailable")
	}
}

// Helper functions

func clearEnv() {
	envVars := []string{
		"TELEMETRY_ADAPTER_ENDPOINT",
		"TELEMETRY_ADAPTER_DATABASE",
		"TELEMETRY_ADAPTER_USERNAME",
		"TELEMETRY_ADAPTER_PASSWORD",
		"TELEMETRY_ADAPTER_TIMEOUT",
		"TELEMETRY_ADAPTER_RETRY_ATTEMPTS",
		"TELEMETRY_ADAPTER_RETRY_BACKOFF",
	}

	for _, env := range envVars {
		os.Unsetenv(env)
	}
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && s[:len(substr)] == substr ||
		len(s) > len(substr) && s[len(s)-len(substr):] == substr ||
		len(substr) < len(s) && findSubstring(s, substr)
}

func findSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

func normalizeWhitespace(s string) string {
	// Simple whitespace normalization for test comparison
	lines := strings.Split(s, "\n")
	var normalized []string
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if trimmed != "" {
			normalized = append(normalized, trimmed)
		}
	}
	return strings.Join(normalized, " ")
}

