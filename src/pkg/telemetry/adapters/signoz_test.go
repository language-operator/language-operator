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
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/language-operator/language-operator/pkg/telemetry"
)

func TestNewSignozAdapter(t *testing.T) {
	t.Run("Valid configuration", func(t *testing.T) {
		adapter, err := NewSignozAdapter("https://signoz.example.com", "test-api-key", 30*time.Second)

		require.NoError(t, err)
		assert.NotNil(t, adapter)
		assert.Equal(t, "https://signoz.example.com", adapter.endpoint)
		assert.Equal(t, "test-api-key", adapter.apiKey)
		assert.Equal(t, 30*time.Second, adapter.timeout)
		assert.NotNil(t, adapter.httpClient)
	})

	t.Run("Empty endpoint", func(t *testing.T) {
		_, err := NewSignozAdapter("", "test-api-key", 30*time.Second)

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "endpoint cannot be empty")
	})

	t.Run("Empty API key", func(t *testing.T) {
		_, err := NewSignozAdapter("https://signoz.example.com", "", 30*time.Second)

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "apiKey cannot be empty")
	})

	t.Run("Zero timeout", func(t *testing.T) {
		_, err := NewSignozAdapter("https://signoz.example.com", "test-api-key", 0)

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "timeout must be positive")
	})

	t.Run("Invalid URL", func(t *testing.T) {
		_, err := NewSignozAdapter("not-a-url", "test-api-key", 30*time.Second)

		require.NoError(t, err) // URL parsing is lenient, but endpoint would be invalid
		// Note: url.Parse doesn't fail for most strings, real validation happens during requests
	})

	t.Run("Trailing slash removal", func(t *testing.T) {
		adapter, err := NewSignozAdapter("https://signoz.example.com/", "test-api-key", 30*time.Second)

		require.NoError(t, err)
		assert.Equal(t, "https://signoz.example.com", adapter.endpoint)
	})
}

func TestNewSignozAdapterFromConfig(t *testing.T) {
	t.Run("Complete config", func(t *testing.T) {
		config := SignozConfig{
			Endpoint:        "https://signoz.example.com",
			APIKey:          "test-api-key",
			Timeout:         45 * time.Second,
			MaxResponseSize: 100 * 1024 * 1024, // 100MB
		}

		adapter, err := NewSignozAdapterFromConfig(config)

		require.NoError(t, err)
		assert.Equal(t, "https://signoz.example.com", adapter.endpoint)
		assert.Equal(t, "test-api-key", adapter.apiKey)
		assert.Equal(t, 45*time.Second, adapter.timeout)
		assert.Equal(t, int64(100*1024*1024), adapter.maxResponseSize)
	})

	t.Run("Default timeout", func(t *testing.T) {
		config := SignozConfig{
			Endpoint: "https://signoz.example.com",
			APIKey:   "test-api-key",
			// Timeout not specified
		}

		adapter, err := NewSignozAdapterFromConfig(config)

		require.NoError(t, err)
		assert.Equal(t, 30*time.Second, adapter.timeout)
	})

	t.Run("Default max response size", func(t *testing.T) {
		config := SignozConfig{
			Endpoint: "https://signoz.example.com",
			APIKey:   "test-api-key",
			// MaxResponseSize not specified
		}

		adapter, err := NewSignozAdapterFromConfig(config)

		require.NoError(t, err)
		assert.Equal(t, int64(DefaultMaxResponseSize), adapter.maxResponseSize)
	})
}

func TestSignozAdapter_QuerySpans(t *testing.T) {
	t.Run("Successful query", func(t *testing.T) {
		// Mock SigNoz server
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Verify request
			assert.Equal(t, "POST", r.Method)
			assert.Equal(t, "/api/v5/query_range", r.URL.Path)
			assert.Equal(t, "test-api-key", r.Header.Get("SIGNOZ-API-KEY"))
			assert.Equal(t, "application/json", r.Header.Get("Content-Type"))

			// Mock response
			response := map[string]interface{}{
				"data": map[string]interface{}{
					"result": []map[string]interface{}{
						{
							"spanID":        "span-123",
							"traceID":       "trace-456",
							"parentSpanID":  "parent-789",
							"operationName": "execute_task",
							"timestamp":     "2025-01-01T12:00:00Z",
							"duration":      float64(1000000000), // 1 second in nanoseconds
							"statusCode":    float64(200),
							"attributes": map[string]interface{}{
								"task.name":  "fetch_user",
								"agent.name": "test-agent",
							},
							"events": []interface{}{},
						},
					},
				},
			}

			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(response)
		}))
		defer server.Close()

		adapter, err := NewSignozAdapter(server.URL, "test-api-key", 30*time.Second)
		require.NoError(t, err)

		filter := telemetry.SpanFilter{
			TaskName: "fetch_user",
			TimeRange: telemetry.TimeRange{
				Start: time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC),
				End:   time.Date(2025, 1, 2, 0, 0, 0, 0, time.UTC),
			},
			Limit: 10,
		}

		spans, err := adapter.QuerySpans(context.Background(), filter)

		require.NoError(t, err)
		require.Len(t, spans, 1)

		span := spans[0]
		assert.Equal(t, "span-123", span.SpanID)
		assert.Equal(t, "trace-456", span.TraceID)
		assert.Equal(t, "parent-789", span.ParentSpanID)
		assert.Equal(t, "execute_task", span.OperationName)
		assert.Equal(t, "fetch_user", span.TaskName)
		assert.True(t, span.Status)
		assert.Equal(t, time.Second, span.Duration)
		assert.Equal(t, "test-agent", span.Attributes["agent.name"])
	})

	t.Run("Empty response", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			response := map[string]interface{}{
				"data": map[string]interface{}{
					"result": []map[string]interface{}{},
				},
			}

			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(response)
		}))
		defer server.Close()

		adapter, err := NewSignozAdapter(server.URL, "test-api-key", 30*time.Second)
		require.NoError(t, err)

		filter := telemetry.SpanFilter{
			TimeRange: telemetry.TimeRange{
				Start: time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC),
				End:   time.Date(2025, 1, 2, 0, 0, 0, 0, time.UTC),
			},
		}

		spans, err := adapter.QuerySpans(context.Background(), filter)

		require.NoError(t, err)
		assert.Empty(t, spans)
	})

	t.Run("API error", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusUnauthorized)
			w.Write([]byte("Invalid API key"))
		}))
		defer server.Close()

		adapter, err := NewSignozAdapter(server.URL, "invalid-key", 30*time.Second)
		require.NoError(t, err)

		filter := telemetry.SpanFilter{
			TimeRange: telemetry.TimeRange{
				Start: time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC),
				End:   time.Date(2025, 1, 2, 0, 0, 0, 0, time.UTC),
			},
		}

		_, err = adapter.QuerySpans(context.Background(), filter)

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "401")
		assert.Contains(t, err.Error(), "Invalid API key")
	})

	t.Run("Context timeout", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			time.Sleep(100 * time.Millisecond) // Delay response
			json.NewEncoder(w).Encode(map[string]interface{}{"data": map[string]interface{}{"result": []interface{}{}}})
		}))
		defer server.Close()

		adapter, err := NewSignozAdapter(server.URL, "test-api-key", 30*time.Second)
		require.NoError(t, err)

		// Use very short context timeout
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Millisecond)
		defer cancel()

		filter := telemetry.SpanFilter{
			TimeRange: telemetry.TimeRange{
				Start: time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC),
				End:   time.Date(2025, 1, 2, 0, 0, 0, 0, time.UTC),
			},
		}

		_, err = adapter.QuerySpans(ctx, filter)

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "context deadline exceeded")
	})
}

func TestSignozAdapter_QueryMetrics(t *testing.T) {
	t.Run("Successful query", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Verify request
			assert.Equal(t, "POST", r.Method)
			assert.Equal(t, "/api/v1/query_range", r.URL.Path)
			assert.Equal(t, "test-api-key", r.Header.Get("SIGNOZ-API-KEY"))

			// Mock Prometheus-compatible response
			response := map[string]interface{}{
				"status": "success",
				"data": map[string]interface{}{
					"resultType": "matrix",
					"result": []map[string]interface{}{
						{
							"metric": map[string]string{
								"__name__":  "task_duration_seconds",
								"task_name": "fetch_user",
								"status":    "success",
							},
							"values": [][]interface{}{
								{float64(1704110400), "1.5"}, // [timestamp, value]
								{float64(1704110460), "2.1"},
							},
						},
					},
				},
			}

			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(response)
		}))
		defer server.Close()

		adapter, err := NewSignozAdapter(server.URL, "test-api-key", 30*time.Second)
		require.NoError(t, err)

		filter := telemetry.MetricFilter{
			MetricName: "task_duration_seconds",
			Labels: map[string]string{
				"task_name": "fetch_user",
			},
			TimeRange: telemetry.TimeRange{
				Start: time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC),
				End:   time.Date(2025, 1, 2, 0, 0, 0, 0, time.UTC),
			},
			Aggregation: "avg",
			Limit:       10,
		}

		metrics, err := adapter.QueryMetrics(context.Background(), filter)

		require.NoError(t, err)
		require.Len(t, metrics, 2)

		// Check metrics are sorted by timestamp (newest first)
		assert.True(t, metrics[0].Time.After(metrics[1].Time))

		metric := metrics[1] // First chronologically
		assert.Equal(t, 1.5, metric.Value)
		assert.Equal(t, "seconds", metric.Unit)
		assert.Equal(t, "fetch_user", metric.Labels["task_name"])
		assert.Equal(t, "success", metric.Labels["status"])
	})

	t.Run("Failed query", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			response := map[string]interface{}{
				"status": "error",
				"error":  "Query failed",
			}

			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(response)
		}))
		defer server.Close()

		adapter, err := NewSignozAdapter(server.URL, "test-api-key", 30*time.Second)
		require.NoError(t, err)

		filter := telemetry.MetricFilter{
			MetricName: "invalid_metric",
			TimeRange: telemetry.TimeRange{
				Start: time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC),
				End:   time.Date(2025, 1, 2, 0, 0, 0, 0, time.UTC),
			},
		}

		_, err = adapter.QueryMetrics(context.Background(), filter)

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "SigNoz metrics query failed")
	})
}

func TestSignozAdapter_Available(t *testing.T) {
	t.Run("Available server", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			assert.Equal(t, "GET", r.Method)
			assert.Equal(t, "/api/v1/version", r.URL.Path)
			assert.Equal(t, "test-api-key", r.Header.Get("SIGNOZ-API-KEY"))

			w.WriteHeader(http.StatusOK)
			json.NewEncoder(w).Encode(map[string]string{"version": "0.12.0"})
		}))
		defer server.Close()

		adapter, err := NewSignozAdapter(server.URL, "test-api-key", 30*time.Second)
		require.NoError(t, err)

		available := adapter.Available()
		assert.True(t, available)

		// Second call should use cache
		available = adapter.Available()
		assert.True(t, available)
	})

	t.Run("Unavailable server", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte("Server error"))
		}))
		defer server.Close()

		adapter, err := NewSignozAdapter(server.URL, "test-api-key", 30*time.Second)
		require.NoError(t, err)

		available := adapter.Available()
		assert.False(t, available)
	})

	t.Run("Network error", func(t *testing.T) {
		// Use invalid endpoint to simulate network error
		adapter, err := NewSignozAdapter("http://invalid-host:12345", "test-api-key", 1*time.Second)
		require.NoError(t, err)

		available := adapter.Available()
		assert.False(t, available)
	})

	t.Run("Cache expiration", func(t *testing.T) {
		callCount := 0
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			callCount++
			w.WriteHeader(http.StatusOK)
			json.NewEncoder(w).Encode(map[string]string{"version": "0.12.0"})
		}))
		defer server.Close()

		adapter, err := NewSignozAdapter(server.URL, "test-api-key", 30*time.Second)
		require.NoError(t, err)

		// Set up deterministic time control for testing
		baseTime := time.Date(2025, 1, 1, 12, 0, 0, 0, time.UTC)
		currentTime := baseTime
		adapter.availabilityCache.ttl = 10 * time.Second
		adapter.availabilityCache.timeNow = func() time.Time {
			return currentTime
		}

		// First call
		available := adapter.Available()
		assert.True(t, available)
		assert.Equal(t, 1, callCount)

		// Second call immediately (should use cache)
		available = adapter.Available()
		assert.True(t, available)
		assert.Equal(t, 1, callCount)

		// Advance time beyond cache TTL
		currentTime = baseTime.Add(15 * time.Second)

		// Third call (should make new request due to expired cache)
		available = adapter.Available()
		assert.True(t, available)
		assert.Equal(t, 2, callCount)
	})

	t.Run("Concurrent cache access race condition", func(t *testing.T) {
		var callCount int
		var callCountMutex sync.Mutex

		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			callCountMutex.Lock()
			callCount++
			callCountMutex.Unlock()
			w.WriteHeader(http.StatusOK)
			json.NewEncoder(w).Encode(map[string]string{"version": "0.12.0"})
		}))
		defer server.Close()

		adapter, err := NewSignozAdapter(server.URL, "test-api-key", 30*time.Second)
		require.NoError(t, err)

		// Set up deterministic time and long cache TTL for testing
		baseTime := time.Date(2025, 1, 1, 12, 0, 0, 0, time.UTC)
		adapter.availabilityCache.ttl = 10 * time.Second // Long TTL
		adapter.availabilityCache.timeNow = func() time.Time {
			return baseTime
		}

		// First call to populate cache
		first := adapter.Available()
		assert.True(t, first)

		// Get initial call count
		callCountMutex.Lock()
		initialCalls := callCount
		callCountMutex.Unlock()

		// Run multiple goroutines concurrently to test for race conditions
		const numGoroutines = 20
		const numCallsPerGoroutine = 5

		var wg sync.WaitGroup
		results := make([]bool, numGoroutines*numCallsPerGoroutine)

		for i := 0; i < numGoroutines; i++ {
			wg.Add(1)
			go func(goroutineID int) {
				defer wg.Done()
				for j := 0; j < numCallsPerGoroutine; j++ {
					// All calls should return same result (true) due to caching
					result := adapter.Available()
					results[goroutineID*numCallsPerGoroutine+j] = result
				}
			}(i)
		}

		wg.Wait()

		// Verify all calls returned true (consistent cache behavior)
		for i, result := range results {
			assert.True(t, result, "Call %d should return true", i)
		}

		// Verify cache was effective (should be no additional calls after initial population)
		callCountMutex.Lock()
		finalCalls := callCount
		callCountMutex.Unlock()

		assert.Equal(t, initialCalls, finalCalls, "Should not make additional HTTP requests due to caching")
	})
}

func TestBuildSpanQuery(t *testing.T) {
	adapter, _ := NewSignozAdapter("https://example.com", "key", 30*time.Second)

	t.Run("Basic query", func(t *testing.T) {
		filter := telemetry.SpanFilter{
			TimeRange: telemetry.TimeRange{
				Start: time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC),
				End:   time.Date(2025, 1, 2, 0, 0, 0, 0, time.UTC),
			},
		}

		query := adapter.buildSpanQuery(filter)

		assert.Contains(t, query, "SELECT")
		assert.Contains(t, query, "FROM signoz_traces.signoz_spans")
		assert.Contains(t, query, "WHERE")
		assert.Contains(t, query, "timestamp >=")
		assert.Contains(t, query, "timestamp <=")
		assert.Contains(t, query, "ORDER BY timestamp DESC")
	})

	t.Run("With task name filter", func(t *testing.T) {
		filter := telemetry.SpanFilter{
			TaskName: "fetch_user",
			TimeRange: telemetry.TimeRange{
				Start: time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC),
				End:   time.Date(2025, 1, 2, 0, 0, 0, 0, time.UTC),
			},
		}

		query := adapter.buildSpanQuery(filter)

		assert.Contains(t, query, "attributes['task.name'] = 'fetch_user'")
		assert.Contains(t, query, "OR")
	})

	t.Run("With trace ID filter", func(t *testing.T) {
		filter := telemetry.SpanFilter{
			TraceID: "trace-123",
			TimeRange: telemetry.TimeRange{
				Start: time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC),
				End:   time.Date(2025, 1, 2, 0, 0, 0, 0, time.UTC),
			},
		}

		query := adapter.buildSpanQuery(filter)

		assert.Contains(t, query, "traceID = 'trace-123'")
	})

	t.Run("With custom attributes", func(t *testing.T) {
		filter := telemetry.SpanFilter{
			Attributes: map[string]string{
				"agent.name": "test-agent",
				"task.type":  "neural",
			},
			TimeRange: telemetry.TimeRange{
				Start: time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC),
				End:   time.Date(2025, 1, 2, 0, 0, 0, 0, time.UTC),
			},
		}

		query := adapter.buildSpanQuery(filter)

		assert.Contains(t, query, "attributes['agent.name'] = 'test-agent'")
		assert.Contains(t, query, "attributes['task.type'] = 'neural'")
		assert.Contains(t, query, "AND")
	})

	t.Run("With limit", func(t *testing.T) {
		filter := telemetry.SpanFilter{
			TimeRange: telemetry.TimeRange{
				Start: time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC),
				End:   time.Date(2025, 1, 2, 0, 0, 0, 0, time.UTC),
			},
			Limit: 50,
		}

		query := adapter.buildSpanQuery(filter)

		assert.Contains(t, query, "LIMIT 50")
	})
}

func TestBuildMetricQuery(t *testing.T) {
	adapter, _ := NewSignozAdapter("https://example.com", "key", 30*time.Second)

	t.Run("Basic metric query", func(t *testing.T) {
		filter := telemetry.MetricFilter{
			MetricName: "task_duration_seconds",
		}

		query := adapter.buildMetricQuery(filter)

		assert.Equal(t, "task_duration_seconds", query)
	})

	t.Run("With labels", func(t *testing.T) {
		filter := telemetry.MetricFilter{
			MetricName: "task_duration_seconds",
			Labels: map[string]string{
				"task_name": "fetch_user",
				"status":    "success",
			},
		}

		query := adapter.buildMetricQuery(filter)

		assert.Contains(t, query, "task_duration_seconds{")
		assert.Contains(t, query, "task_name=\"fetch_user\"")
		assert.Contains(t, query, "status=\"success\"")
		assert.Contains(t, query, "}")
	})

	t.Run("With aggregation", func(t *testing.T) {
		filter := telemetry.MetricFilter{
			MetricName:  "task_duration_seconds",
			Aggregation: "avg",
		}

		query := adapter.buildMetricQuery(filter)

		assert.Equal(t, "avg(task_duration_seconds)", query)
	})

	t.Run("With unknown aggregation", func(t *testing.T) {
		filter := telemetry.MetricFilter{
			MetricName:  "task_duration_seconds",
			Aggregation: "unknown",
		}

		query := adapter.buildMetricQuery(filter)

		assert.Equal(t, "avg(task_duration_seconds)", query)
	})
}

func TestExtractTaskName(t *testing.T) {
	adapter, _ := NewSignozAdapter("https://example.com", "key", 30*time.Second)

	testCases := []struct {
		operationName string
		attributes    map[string]string
		expected      string
	}{
		{
			operationName: "fetch_data",
			attributes: map[string]string{
				"task.name": "fetch_user",
			},
			expected: "fetch_user",
		},
		{
			operationName: "execute_task",
			attributes: map[string]string{
				"task": "process_data",
			},
			expected: "process_data",
		},
		{
			operationName: "some_operation",
			attributes: map[string]string{
				"task_name": "analyze_data",
			},
			expected: "analyze_data",
		},
		{
			operationName: "fallback_operation",
			attributes:    map[string]string{},
			expected:      "fallback_operation",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.expected, func(t *testing.T) {
			result := adapter.extractTaskName(tc.operationName, tc.attributes)
			assert.Equal(t, tc.expected, result)
		})
	}
}

func TestExtractMetricUnit(t *testing.T) {
	adapter, _ := NewSignozAdapter("https://example.com", "key", 30*time.Second)

	testCases := []struct {
		labels   map[string]string
		expected string
	}{
		{
			labels: map[string]string{
				"__name__": "task_duration_seconds",
			},
			expected: "seconds",
		},
		{
			labels: map[string]string{
				"__name__": "memory_usage_bytes",
			},
			expected: "bytes",
		},
		{
			labels: map[string]string{
				"__name__": "request_count",
			},
			expected: "count",
		},
		{
			labels: map[string]string{
				"__name__": "api_cost_dollars",
			},
			expected: "dollars",
		},
		{
			labels: map[string]string{
				"__name__": "custom_metric",
				"unit":     "percentage",
			},
			expected: "percentage",
		},
		{
			labels: map[string]string{
				"__name__": "unknown_metric",
			},
			expected: "",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.labels["__name__"], func(t *testing.T) {
			result := adapter.extractMetricUnit(tc.labels)
			assert.Equal(t, tc.expected, result)
		})
	}
}

func TestEscapeClickHouseString(t *testing.T) {
	testCases := []struct {
		input    string
		expected string
	}{
		{
			input:    "simple_string",
			expected: "simple_string",
		},
		{
			input:    "string'with'quotes",
			expected: "string''with''quotes",
		},
		{
			input:    "multiple'quotes'here'too",
			expected: "multiple''quotes''here''too",
		},
		{
			input:    "",
			expected: "",
		},
		{
			input:    "'",
			expected: "''",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.input, func(t *testing.T) {
			result := escapeClickHouseString(tc.input)
			assert.Equal(t, tc.expected, result)
		})
	}
}

// TestTelemetryAdapterInterface verifies that SignozAdapter implements TelemetryAdapter
func TestTelemetryAdapterInterface(t *testing.T) {
	adapter, err := NewSignozAdapter("https://example.com", "key", 30*time.Second)
	require.NoError(t, err)

	// This should compile if SignozAdapter properly implements TelemetryAdapter
	var _ telemetry.TelemetryAdapter = adapter

	// Test that all methods exist and return expected types
	ctx := context.Background()

	// Available should return bool
	available := adapter.Available()
	assert.IsType(t, true, available)

	// QuerySpans should return []Span and error
	spans, err := adapter.QuerySpans(ctx, telemetry.SpanFilter{})
	assert.IsType(t, []telemetry.Span{}, spans)
	assert.IsType(t, (*error)(nil), &err)

	// QueryMetrics should return []MetricPoint and error
	metrics, err := adapter.QueryMetrics(ctx, telemetry.MetricFilter{})
	assert.IsType(t, []telemetry.MetricPoint{}, metrics)
	assert.IsType(t, (*error)(nil), &err)
}

func TestNewSignozAdapterWithMaxSize(t *testing.T) {
	t.Run("Valid configuration", func(t *testing.T) {
		maxSize := int64(10 * 1024 * 1024) // 10MB
		adapter, err := NewSignozAdapterWithMaxSize("https://signoz.example.com", "test-api-key", 30*time.Second, maxSize)

		require.NoError(t, err)
		assert.NotNil(t, adapter)
		assert.Equal(t, "https://signoz.example.com", adapter.endpoint)
		assert.Equal(t, "test-api-key", adapter.apiKey)
		assert.Equal(t, 30*time.Second, adapter.timeout)
		assert.Equal(t, maxSize, adapter.maxResponseSize)
	})

	t.Run("Zero max response size", func(t *testing.T) {
		_, err := NewSignozAdapterWithMaxSize("https://signoz.example.com", "test-api-key", 30*time.Second, 0)

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "maxResponseSize must be positive")
	})

	t.Run("Negative max response size", func(t *testing.T) {
		_, err := NewSignozAdapterWithMaxSize("https://signoz.example.com", "test-api-key", 30*time.Second, -100)

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "maxResponseSize must be positive")
	})
}

func TestSignozAdapter_ResponseSizeLimiting(t *testing.T) {
	t.Run("Response within size limit", func(t *testing.T) {
		responseData := "small response data"
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(responseData))
		}))
		defer server.Close()

		// Set max size to 1KB (much larger than our test response)
		adapter, err := NewSignozAdapterWithMaxSize(server.URL, "test-api-key", 30*time.Second, 1024)
		require.NoError(t, err)

		// Test the makeRequest method directly (it's unexported, but we can test through QuerySpans)
		filter := telemetry.SpanFilter{
			TimeRange: telemetry.TimeRange{
				Start: time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC),
				End:   time.Date(2025, 1, 2, 0, 0, 0, 0, time.UTC),
			},
		}

		// This should succeed without errors
		_, err = adapter.QuerySpans(context.Background(), filter)
		// Note: We expect an error here due to JSON parsing of our simple string, but not a size limit error
		assert.Error(t, err)
		assert.NotContains(t, err.Error(), "exceeds maximum allowed size")
	})

	t.Run("Response exceeds size limit", func(t *testing.T) {
		// Create a large response (1MB of data)
		largeData := make([]byte, 1024*1024) // 1MB
		for i := range largeData {
			largeData[i] = 'A'
		}

		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			w.Write(largeData)
		}))
		defer server.Close()

		// Set max size to 500KB (smaller than our 1MB response)
		maxSize := int64(500 * 1024) // 500KB
		adapter, err := NewSignozAdapterWithMaxSize(server.URL, "test-api-key", 30*time.Second, maxSize)
		require.NoError(t, err)

		filter := telemetry.SpanFilter{
			TimeRange: telemetry.TimeRange{
				Start: time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC),
				End:   time.Date(2025, 1, 2, 0, 0, 0, 0, time.UTC),
			},
		}

		_, err = adapter.QuerySpans(context.Background(), filter)

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "exceeds maximum allowed size")
		assert.Contains(t, err.Error(), "512000 bytes") // 500KB in bytes
	})

	t.Run("Response exactly at size limit", func(t *testing.T) {
		// Create response data exactly at the limit
		maxSize := int64(1024) // 1KB
		exactData := make([]byte, maxSize)
		for i := range exactData {
			exactData[i] = 'B'
		}

		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			w.Write(exactData)
		}))
		defer server.Close()

		adapter, err := NewSignozAdapterWithMaxSize(server.URL, "test-api-key", 30*time.Second, maxSize)
		require.NoError(t, err)

		filter := telemetry.SpanFilter{
			TimeRange: telemetry.TimeRange{
				Start: time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC),
				End:   time.Date(2025, 1, 2, 0, 0, 0, 0, time.UTC),
			},
		}

		_, err = adapter.QuerySpans(context.Background(), filter)

		// Should succeed (no size limit error, though may have JSON parsing error)
		if err != nil {
			assert.NotContains(t, err.Error(), "exceeds maximum allowed size")
		}
	})
}

func TestDefaultMaxResponseSize(t *testing.T) {
	// Test that the default constant is reasonable (50MB)
	assert.Equal(t, 50*1024*1024, DefaultMaxResponseSize)

	// Test that NewSignozAdapter uses the default
	adapter, err := NewSignozAdapter("https://example.com", "key", 30*time.Second)
	require.NoError(t, err)
	assert.Equal(t, int64(DefaultMaxResponseSize), adapter.maxResponseSize)
}
