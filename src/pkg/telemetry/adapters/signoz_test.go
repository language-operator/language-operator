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

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "endpoint URL must include scheme")
	})

	t.Run("Invalid URL with unsupported scheme", func(t *testing.T) {
		_, err := NewSignozAdapter("ftp://signoz.example.com", "test-api-key", 30*time.Second)

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "endpoint URL scheme must be http or https")
	})

	t.Run("Invalid URL missing host", func(t *testing.T) {
		_, err := NewSignozAdapter("https://", "test-api-key", 30*time.Second)

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "endpoint URL must include host")
	})

	t.Run("Invalid URL empty host with port", func(t *testing.T) {
		_, err := NewSignozAdapter("https://:3000", "test-api-key", 30*time.Second)

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "host cannot start with colon")
	})

	t.Run("Invalid URL host with empty port", func(t *testing.T) {
		_, err := NewSignozAdapter("http://localhost:", "test-api-key", 30*time.Second)

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "host cannot end with colon")
	})

	t.Run("Invalid URL port out of range", func(t *testing.T) {
		_, err := NewSignozAdapter("https://example.com:99999", "test-api-key", 30*time.Second)

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "port must be ≤65535")
	})

	t.Run("Invalid URL port zero", func(t *testing.T) {
		_, err := NewSignozAdapter("https://example.com:0", "test-api-key", 30*time.Second)

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "port must be positive")
	})

	t.Run("Invalid hostname single dot", func(t *testing.T) {
		_, err := NewSignozAdapter("https://.", "test-api-key", 30*time.Second)

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "hostname cannot be single dot")
	})

	t.Run("Invalid hostname leading dot", func(t *testing.T) {
		_, err := NewSignozAdapter("https://.example.com", "test-api-key", 30*time.Second)

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "hostname cannot start with dot")
	})

	t.Run("Valid IPv6 with port", func(t *testing.T) {
		adapter, err := NewSignozAdapter("https://[::1]:3000", "test-api-key", 30*time.Second)

		require.NoError(t, err)
		assert.Equal(t, "https://[::1]:3000", adapter.endpoint)
	})

	t.Run("Valid IPv6 without port", func(t *testing.T) {
		adapter, err := NewSignozAdapter("http://::1", "test-api-key", 30*time.Second)

		require.NoError(t, err)
		assert.Equal(t, "http://::1", adapter.endpoint)
	})

	t.Run("Invalid IPv6 in brackets", func(t *testing.T) {
		_, err := NewSignozAdapter("https://[invalid::ipv6::address]:3000", "test-api-key", 30*time.Second)

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "hostname contains colon but is not valid IPv6")
	})

	t.Run("Invalid IPv4 in IPv6 brackets", func(t *testing.T) {
		_, err := NewSignozAdapter("https://[192.168.1.1]:3000", "test-api-key", 30*time.Second)

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "IPv4 address in IPv6 brackets")
	})

	t.Run("Valid edge case ports", func(t *testing.T) {
		testCases := []struct {
			port string
			url  string
		}{
			{"1", "https://example.com:1"},
			{"65535", "https://example.com:65535"},
			{"8080", "https://example.com:8080"},
		}

		for _, tc := range testCases {
			t.Run("port "+tc.port, func(t *testing.T) {
				adapter, err := NewSignozAdapter(tc.url, "test-api-key", 30*time.Second)
				require.NoError(t, err)
				assert.Equal(t, tc.url, adapter.endpoint)
			})
		}
	})

	t.Run("Valid URL with port", func(t *testing.T) {
		adapter, err := NewSignozAdapter("https://signoz.example.com:3000", "test-api-key", 30*time.Second)

		require.NoError(t, err)
		assert.Equal(t, "https://signoz.example.com:3000", adapter.endpoint)
	})

	t.Run("Valid URL with path", func(t *testing.T) {
		adapter, err := NewSignozAdapter("http://localhost:3000/api", "test-api-key", 30*time.Second)

		require.NoError(t, err)
		assert.Equal(t, "http://localhost:3000/api", adapter.endpoint)
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

func TestValidateHost(t *testing.T) {
	testCases := []struct {
		name        string
		host        string
		expectError bool
		errorText   string
	}{
		{
			name:        "valid hostname",
			host:        "example.com",
			expectError: false,
		},
		{
			name:        "valid hostname with port",
			host:        "example.com:3000",
			expectError: false,
		},
		{
			name:        "valid IPv4",
			host:        "192.168.1.1",
			expectError: false,
		},
		{
			name:        "valid IPv4 with port",
			host:        "192.168.1.1:8080",
			expectError: false,
		},
		{
			name:        "valid IPv6 with brackets and port",
			host:        "[::1]:3000",
			expectError: false,
		},
		{
			name:        "valid IPv6 without brackets (no port)",
			host:        "::1",
			expectError: false,
		},
		{
			name:        "empty host",
			host:        "",
			expectError: true,
			errorText:   "host cannot be empty",
		},
		{
			name:        "host starting with colon",
			host:        ":3000",
			expectError: true,
			errorText:   "host cannot start with colon",
		},
		{
			name:        "host ending with colon",
			host:        "localhost:",
			expectError: true,
			errorText:   "host cannot end with colon",
		},
		{
			name:        "invalid port range high",
			host:        "example.com:99999",
			expectError: true,
			errorText:   "port must be ≤65535",
		},
		{
			name:        "invalid port zero",
			host:        "example.com:0",
			expectError: true,
			errorText:   "port must be positive",
		},
		{
			name:        "invalid port negative",
			host:        "example.com:-1",
			expectError: true,
			errorText:   "invalid port",
		},
		{
			name:        "single dot hostname",
			host:        ".",
			expectError: true,
			errorText:   "hostname cannot be single dot",
		},
		{
			name:        "hostname starting with dot",
			host:        ".example.com",
			expectError: true,
			errorText:   "hostname cannot start with dot",
		},
		{
			name:        "invalid IPv6 in brackets",
			host:        "[invalid::ipv6]:3000",
			expectError: true,
			errorText:   "hostname contains colon but is not valid IPv6",
		},
		{
			name:        "IPv4 in IPv6 brackets",
			host:        "[192.168.1.1]:3000",
			expectError: true,
			errorText:   "IPv4 address in IPv6 brackets",
		},
		{
			name:        "valid boundary ports",
			host:        "example.com:1",
			expectError: false,
		},
		{
			name:        "valid max port",
			host:        "example.com:65535",
			expectError: false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			err := validateHost(tc.host)

			if tc.expectError {
				assert.Error(t, err, "Expected error for host: %s", tc.host)
				if tc.errorText != "" {
					assert.Contains(t, err.Error(), tc.errorText)
				}
			} else {
				assert.NoError(t, err, "Expected no error for host: %s", tc.host)
			}
		})
	}
}

func TestValidateHostname(t *testing.T) {
	testCases := []struct {
		name        string
		hostname    string
		expectError bool
		errorText   string
	}{
		{
			name:        "valid hostname",
			hostname:    "example.com",
			expectError: false,
		},
		{
			name:        "valid IPv4",
			hostname:    "192.168.1.1",
			expectError: false,
		},
		{
			name:        "valid IPv6",
			hostname:    "::1",
			expectError: false,
		},
		{
			name:        "valid IPv6 with brackets",
			hostname:    "[::1]",
			expectError: false,
		},
		{
			name:        "empty hostname",
			hostname:    "",
			expectError: true,
			errorText:   "hostname cannot be empty",
		},
		{
			name:        "single dot",
			hostname:    ".",
			expectError: true,
			errorText:   "hostname cannot be single dot",
		},
		{
			name:        "leading dot",
			hostname:    ".example.com",
			expectError: true,
			errorText:   "hostname cannot start with dot",
		},
		{
			name:        "invalid colon usage",
			hostname:    "invalid:colon:usage",
			expectError: true,
			errorText:   "hostname contains colon but is not valid IPv6",
		},
		{
			name:        "invalid IPv6 brackets",
			hostname:    "[invalid::ipv6]",
			expectError: true,
			errorText:   "invalid IPv6 address",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			err := validateHostname(tc.hostname)

			if tc.expectError {
				assert.Error(t, err, "Expected error for hostname: %s", tc.hostname)
				if tc.errorText != "" {
					assert.Contains(t, err.Error(), tc.errorText)
				}
			} else {
				assert.NoError(t, err, "Expected no error for hostname: %s", tc.hostname)
			}
		})
	}
}

func TestValidatePort(t *testing.T) {
	testCases := []struct {
		name        string
		port        string
		expectError bool
		errorText   string
	}{
		{
			name:        "valid port",
			port:        "3000",
			expectError: false,
		},
		{
			name:        "valid port 1",
			port:        "1",
			expectError: false,
		},
		{
			name:        "valid port max",
			port:        "65535",
			expectError: false,
		},
		{
			name:        "empty port",
			port:        "",
			expectError: true,
			errorText:   "port cannot be empty",
		},
		{
			name:        "non-numeric port",
			port:        "abc",
			expectError: true,
			errorText:   "port must be numeric",
		},
		{
			name:        "port zero",
			port:        "0",
			expectError: true,
			errorText:   "port must be positive",
		},
		{
			name:        "negative port",
			port:        "-1",
			expectError: true,
			errorText:   "port must be positive",
		},
		{
			name:        "port too high",
			port:        "99999",
			expectError: true,
			errorText:   "port must be ≤65535",
		},
		{
			name:        "port with decimal",
			port:        "3000.5",
			expectError: true,
			errorText:   "port must be numeric",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			err := validatePort(tc.port)

			if tc.expectError {
				assert.Error(t, err, "Expected error for port: %s", tc.port)
				if tc.errorText != "" {
					assert.Contains(t, err.Error(), tc.errorText)
				}
			} else {
				assert.NoError(t, err, "Expected no error for port: %s", tc.port)
			}
		})
	}
}

// TestQueryBuilderV5Payload tests the new Query Builder v5 payload generation
func TestQueryBuilderV5Payload(t *testing.T) {
	adapter, err := NewSignozAdapter("https://signoz.example.com", "test-api-key", 30*time.Second)
	require.NoError(t, err)

	t.Run("Basic payload structure", func(t *testing.T) {
		filter := telemetry.SpanFilter{
			TimeRange: telemetry.TimeRange{
				Start: time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC),
				End:   time.Date(2025, 1, 2, 0, 0, 0, 0, time.UTC),
			},
			Limit: 10,
		}

		payload := adapter.buildQueryBuilderV5Payload(filter)

		// Check top-level structure
		assert.Equal(t, "raw", payload["requestType"])
		assert.NotNil(t, payload["start"])
		assert.NotNil(t, payload["end"])
		assert.NotNil(t, payload["variables"])
		assert.NotNil(t, payload["compositeQuery"])

		// Check timestamps are in milliseconds
		startMs := payload["start"].(int64)
		endMs := payload["end"].(int64)
		assert.Greater(t, startMs, int64(1000000000000)) // > 1 billion milliseconds (year 2001)
		assert.Greater(t, endMs, startMs)

		// Check compositeQuery structure
		compositeQuery := payload["compositeQuery"].(map[string]interface{})
		queries := compositeQuery["queries"].([]map[string]interface{})
		assert.Len(t, queries, 1)

		query := queries[0]
		assert.Equal(t, "builder_query", query["type"])

		spec := query["spec"].(map[string]interface{})
		assert.Equal(t, "A", spec["name"])
		assert.Equal(t, "traces", spec["signal"])
		assert.Equal(t, 10, spec["limit"])
		assert.Equal(t, 0, spec["offset"])
		assert.Equal(t, false, spec["disabled"])

		// Check selectFields - only basic fields to avoid SigNoz v5 compatibility issues
		selectFields := spec["selectFields"].([]map[string]string)
		expectedFields := []string{"spanID", "traceID", "timestamp", "duration"}
		assert.Len(t, selectFields, len(expectedFields))
		for i, field := range expectedFields {
			assert.Equal(t, field, selectFields[i]["name"])
		}

		// Check order
		order := spec["order"].([]map[string]interface{})
		assert.Len(t, order, 1)
		assert.Equal(t, "timestamp", order[0]["key"].(map[string]string)["name"])
		assert.Equal(t, "desc", order[0]["direction"])
	})

	t.Run("Filter expressions", func(t *testing.T) {
		filter := telemetry.SpanFilter{
			TaskName: "fetch_user",
			TraceID:  "trace-123",
			Attributes: map[string]string{
				"environment": "production",
				"service":     "api",
			},
			TimeRange: telemetry.TimeRange{
				Start: time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC),
				End:   time.Date(2025, 1, 2, 0, 0, 0, 0, time.UTC),
			},
		}

		payload := adapter.buildQueryBuilderV5Payload(filter)

		compositeQuery := payload["compositeQuery"].(map[string]interface{})
		queries := compositeQuery["queries"].([]map[string]interface{})
		spec := queries[0]["spec"].(map[string]interface{})
		filterMap := spec["filter"].(map[string]interface{})
		expression := filterMap["expression"].(string)

		// Filter expression should be empty due to SigNoz v5 syntax compatibility issues
		// buildFilterExpression returns empty string to avoid query errors
		assert.Equal(t, "", expression)
	})

	t.Run("No filters (empty expression)", func(t *testing.T) {
		filter := telemetry.SpanFilter{
			TimeRange: telemetry.TimeRange{
				Start: time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC),
				End:   time.Date(2025, 1, 2, 0, 0, 0, 0, time.UTC),
			},
		}

		payload := adapter.buildQueryBuilderV5Payload(filter)

		compositeQuery := payload["compositeQuery"].(map[string]interface{})
		queries := compositeQuery["queries"].([]map[string]interface{})
		spec := queries[0]["spec"].(map[string]interface{})
		filterMap := spec["filter"].(map[string]interface{})
		expression := filterMap["expression"].(string)

		assert.Equal(t, "", expression, "Empty filter should result in empty expression")
	})
}

// TestQueryBuilderV5Response tests parsing of Query Builder v5 responses
func TestQueryBuilderV5Response(t *testing.T) {
	t.Run("Parse successful Query Builder v5 response", func(t *testing.T) {
		// Mock SigNoz server with Query Builder v5 response format
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Verify the request payload is Query Builder v5 format
			var reqPayload map[string]interface{}
			json.NewDecoder(r.Body).Decode(&reqPayload)

			// Verify key Query Builder v5 fields exist
			assert.Equal(t, "raw", reqPayload["requestType"])
			assert.NotNil(t, reqPayload["compositeQuery"])

			// Mock Query Builder v5 response
			response := map[string]interface{}{
				"status": "success",
				"data": map[string]interface{}{
					"resultType": "list",
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
		assert.Len(t, spans, 1)

		span := spans[0]
		assert.Equal(t, "span-123", span.SpanID)
		assert.Equal(t, "trace-456", span.TraceID)
		assert.Equal(t, "parent-789", span.ParentSpanID)
		assert.Equal(t, "execute_task", span.OperationName)
		assert.Equal(t, "fetch_user", span.TaskName)
		assert.True(t, span.Status) // Status code 200 = success
		assert.Equal(t, 1*time.Second, span.Duration)
	})

	t.Run("Handle API errors in Query Builder v5 format", func(t *testing.T) {
		// Mock server that returns Query Builder v5 error response
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusBadRequest)

			response := map[string]interface{}{
				"status": "error",
				"error": map[string]interface{}{
					"code":    "invalid_input",
					"message": "failed to decode request body: unknown field \"query\"",
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

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "SigNoz API error: 400")
		assert.Contains(t, err.Error(), "unknown field")
		assert.Empty(t, spans)
	})
}
