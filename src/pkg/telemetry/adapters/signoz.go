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
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/language-operator/language-operator/pkg/telemetry"
)

// SignozAdapter implements TelemetryAdapter for SigNoz observability platform.
//
// SigNoz stores traces in ClickHouse database and provides REST API for querying.
// This adapter translates TelemetryAdapter interface calls to SigNoz API requests
// and converts responses back to standardized format.
//
// Key features:
// - Query spans via /api/v5/query_range endpoint
// - Support complex filters with ClickHouse WHERE clauses
// - Handle trace ID collection and tool span queries
// - Normalize response data to standard Span format
// - Proper authentication with SIGNOZ-API-KEY header
//
// Example usage:
//
//	adapter := NewSignozAdapter("https://signoz.example.com", "api-key", 30*time.Second)
//	spans, err := adapter.QuerySpans(ctx, telemetry.SpanFilter{
//	  TaskName: "fetch_user",
//	  TimeRange: telemetry.TimeRange{Start: yesterday, End: now},
//	  Limit: 50,
//	})
type SignozAdapter struct {
	// endpoint is the base URL of the SigNoz instance
	// Example: "https://signoz.example.com" or "http://localhost:3301"
	endpoint string

	// apiKey is the SigNoz API key for authentication
	// Used in SIGNOZ-API-KEY header for all requests
	apiKey string

	// httpClient is the HTTP client for making requests
	httpClient *http.Client

	// timeout is the default timeout for HTTP requests
	timeout time.Duration

	// availabilityCache caches the result of Available() checks
	// to avoid frequent health checks
	availabilityCache struct {
		value     bool
		timestamp time.Time
		ttl       time.Duration
	}
}

// SignozConfig contains configuration options for SignozAdapter.
type SignozConfig struct {
	// Endpoint is the base URL of the SigNoz instance
	Endpoint string

	// APIKey is the SigNoz API key for authentication
	APIKey string

	// Timeout is the HTTP request timeout
	// Defaults to 30 seconds if not specified
	Timeout time.Duration

	// RetryCount is the number of retries for failed requests
	// Defaults to 3 if not specified
	RetryCount int
}

// NewSignozAdapter creates a new SignozAdapter with the given configuration.
//
// Parameters:
//   - endpoint: Base URL of SigNoz instance (e.g., "https://signoz.example.com")
//   - apiKey: SigNoz API key for authentication
//   - timeout: HTTP request timeout (use 30*time.Second for default)
//
// Returns error if:
//   - endpoint is empty or invalid URL
//   - apiKey is empty
//   - timeout is zero or negative
//
// Example:
//
//	adapter, err := NewSignozAdapter(
//	  "https://signoz.example.com",
//	  "your-api-key",
//	  30*time.Second,
//	)
func NewSignozAdapter(endpoint, apiKey string, timeout time.Duration) (*SignozAdapter, error) {
	if endpoint == "" {
		return nil, fmt.Errorf("endpoint cannot be empty")
	}

	if apiKey == "" {
		return nil, fmt.Errorf("apiKey cannot be empty")
	}

	if timeout <= 0 {
		return nil, fmt.Errorf("timeout must be positive, got %v", timeout)
	}

	// Validate endpoint is a proper URL
	if _, err := url.Parse(endpoint); err != nil {
		return nil, fmt.Errorf("invalid endpoint URL: %w", err)
	}

	// Remove trailing slash from endpoint for consistent URL building
	endpoint = strings.TrimSuffix(endpoint, "/")

	adapter := &SignozAdapter{
		endpoint: endpoint,
		apiKey:   apiKey,
		timeout:  timeout,
		httpClient: &http.Client{
			Timeout: timeout,
			Transport: &http.Transport{
				MaxIdleConns:        10,
				IdleConnTimeout:     90 * time.Second,
				DisableCompression:  false,
				MaxIdleConnsPerHost: 2,
			},
		},
	}

	// Initialize availability cache with 30 second TTL
	adapter.availabilityCache.ttl = 30 * time.Second

	return adapter, nil
}

// NewSignozAdapterFromConfig creates a SignozAdapter from a config struct.
//
// Applies default values for optional fields:
//   - Timeout: 30 seconds if not specified
//   - RetryCount: 3 if not specified (currently unused)
func NewSignozAdapterFromConfig(config SignozConfig) (*SignozAdapter, error) {
	timeout := config.Timeout
	if timeout == 0 {
		timeout = 30 * time.Second
	}

	return NewSignozAdapter(config.Endpoint, config.APIKey, timeout)
}

// makeRequest performs an HTTP request with proper authentication and error handling.
//
// Adds SIGNOZ-API-KEY header and handles common error conditions.
// Returns the response body as bytes on success.
func (s *SignozAdapter) makeRequest(ctx context.Context, method, path string, body []byte) ([]byte, error) {
	url := s.endpoint + path

	var req *http.Request
	var err error

	if body != nil {
		req, err = http.NewRequestWithContext(ctx, method, url, bytes.NewReader(body))
	} else {
		req, err = http.NewRequestWithContext(ctx, method, url, nil)
	}
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	// Add authentication header
	req.Header.Set("SIGNOZ-API-KEY", s.apiKey)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")

	resp, err := s.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	// Read response body
	respBody := make([]byte, 0, 4096)
	buf := make([]byte, 1024)
	for {
		n, err := resp.Body.Read(buf)
		if n > 0 {
			respBody = append(respBody, buf[:n]...)
		}
		if err != nil {
			break
		}
	}

	// Check response status
	if resp.StatusCode >= 400 {
		return nil, fmt.Errorf("SigNoz API error: %d %s, body: %s",
			resp.StatusCode, resp.Status, string(respBody))
	}

	return respBody, nil
}

// QuerySpans retrieves execution spans from SigNoz matching the given filter criteria.
//
// Implements TelemetryAdapter.QuerySpans by:
//  1. Building ClickHouse SQL query with appropriate WHERE clauses
//  2. Calling SigNoz /api/v5/query_range endpoint
//  3. Parsing response and converting to standard Span format
//  4. Handling pagination and result limits
//
// The method constructs complex ClickHouse queries to filter spans by:
//   - Time range (start/end timestamps)
//   - Task name (extracted from span attributes)
//   - Trace ID (exact match)
//   - Custom attributes (key-value pairs)
//
// Returns spans ordered by timestamp (newest first) up to filter.Limit.
// Returns empty slice (not error) if no spans match criteria.
func (s *SignozAdapter) QuerySpans(ctx context.Context, filter telemetry.SpanFilter) ([]telemetry.Span, error) {
	// Build ClickHouse query based on filter
	query := s.buildSpanQuery(filter)

	// Prepare request payload for SigNoz API
	reqPayload := map[string]interface{}{
		"query": query,
		"start": filter.TimeRange.Start.Unix(),
		"end":   filter.TimeRange.End.Unix(),
		"step":  60, // 1 minute step for aggregation (not used for span queries)
	}

	reqBody, err := json.Marshal(reqPayload)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	// Make request to SigNoz
	respBody, err := s.makeRequest(ctx, "POST", "/api/v5/query_range", reqBody)
	if err != nil {
		return nil, fmt.Errorf("failed to query SigNoz: %w", err)
	}

	// Parse SigNoz response
	spans, err := s.parseSpanResponse(respBody, filter.Limit)
	if err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	return spans, nil
}

// buildSpanQuery constructs a ClickHouse SQL query for span filtering.
//
// SigNoz stores spans in ClickHouse with the following relevant columns:
//   - timestamp: Span start time (DateTime64)
//   - traceID: Trace identifier (String)
//   - spanID: Span identifier (String)
//   - parentSpanID: Parent span identifier (String)
//   - operationName: Operation name (String)
//   - duration: Span duration in nanoseconds (UInt64)
//   - statusCode: HTTP-like status code (UInt16)
//   - attributes: Key-value attributes (Map(String, String))
//   - events: Span events (Array)
//
// Builds WHERE clauses based on SpanFilter fields.
func (s *SignozAdapter) buildSpanQuery(filter telemetry.SpanFilter) string {
	var conditions []string
	var orderBy string = "ORDER BY timestamp DESC"
	var limitClause string

	// Base SELECT with required fields
	query := `SELECT 
		spanID,
		traceID,
		parentSpanID,
		operationName,
		timestamp,
		duration,
		statusCode,
		attributes,
		events
	FROM signoz_traces.signoz_spans`

	// Time range filter (always present)
	if !filter.TimeRange.Start.IsZero() && !filter.TimeRange.End.IsZero() {
		startUnix := filter.TimeRange.Start.Unix()
		endUnix := filter.TimeRange.End.Unix()
		conditions = append(conditions, fmt.Sprintf(
			"timestamp >= toDateTime64(%d, 3) AND timestamp <= toDateTime64(%d, 3)",
			startUnix, endUnix))
	}

	// Task name filter (look in attributes map)
	if filter.TaskName != "" {
		// Task name could be stored in different attribute keys
		taskConditions := []string{
			fmt.Sprintf("attributes['task.name'] = '%s'", escapeClickHouseString(filter.TaskName)),
			fmt.Sprintf("attributes['task_name'] = '%s'", escapeClickHouseString(filter.TaskName)),
			fmt.Sprintf("operationName = '%s'", escapeClickHouseString(filter.TaskName)),
		}
		conditions = append(conditions, "("+strings.Join(taskConditions, " OR ")+")")
	}

	// Trace ID filter
	if filter.TraceID != "" {
		conditions = append(conditions, fmt.Sprintf("traceID = '%s'", escapeClickHouseString(filter.TraceID)))
	}

	// Custom attributes filter
	for key, value := range filter.Attributes {
		conditions = append(conditions, fmt.Sprintf(
			"attributes['%s'] = '%s'",
			escapeClickHouseString(key),
			escapeClickHouseString(value)))
	}

	// Combine conditions
	if len(conditions) > 0 {
		query += " WHERE " + strings.Join(conditions, " AND ")
	}

	// Add ordering and limit
	query += " " + orderBy

	if filter.Limit > 0 {
		limitClause = fmt.Sprintf(" LIMIT %d", filter.Limit)
		query += limitClause
	}

	return query
}

// parseSpanResponse parses SigNoz ClickHouse query response into standard Span format.
//
// SigNoz returns query results in JSON format with structure:
//
//	{
//	  "data": {
//	    "result": [
//	      {
//	        "metric": {},
//	        "values": [[timestamp, value], ...]
//	      }
//	    ]
//	  }
//	}
//
// However, for span queries, the format is different and contains actual row data.
func (s *SignozAdapter) parseSpanResponse(respBody []byte, limit int) ([]telemetry.Span, error) {
	var response struct {
		Data struct {
			Result []map[string]interface{} `json:"result"`
		} `json:"data"`
	}

	if err := json.Unmarshal(respBody, &response); err != nil {
		return nil, fmt.Errorf("failed to unmarshal response: %w", err)
	}

	spans := make([]telemetry.Span, 0, len(response.Data.Result))

	for _, row := range response.Data.Result {
		span, err := s.convertRowToSpan(row)
		if err != nil {
			// Log error but continue processing other spans
			continue
		}
		spans = append(spans, span)

		// Respect limit
		if limit > 0 && len(spans) >= limit {
			break
		}
	}

	return spans, nil
}

// convertRowToSpan converts a single ClickHouse row to a standard Span.
//
// Maps SigNoz span fields to telemetry.Span struct fields.
func (s *SignozAdapter) convertRowToSpan(row map[string]interface{}) (telemetry.Span, error) {
	span := telemetry.Span{}

	// Extract basic fields
	if spanID, ok := row["spanID"].(string); ok {
		span.SpanID = spanID
	}

	if traceID, ok := row["traceID"].(string); ok {
		span.TraceID = traceID
	}

	if parentSpanID, ok := row["parentSpanID"].(string); ok {
		span.ParentSpanID = parentSpanID
	}

	if operationName, ok := row["operationName"].(string); ok {
		span.OperationName = operationName
	}

	// Parse timestamp
	if timestampStr, ok := row["timestamp"].(string); ok {
		if timestamp, err := time.Parse(time.RFC3339, timestampStr); err == nil {
			span.StartTime = timestamp
		}
	}

	// Parse duration and calculate end time
	if durationNano, ok := row["duration"].(float64); ok {
		span.Duration = time.Duration(int64(durationNano))
		span.EndTime = span.StartTime.Add(span.Duration)
	}

	// Parse status code
	if statusCode, ok := row["statusCode"].(float64); ok {
		// SigNoz uses HTTP-like status codes
		// 2xx = success, 4xx/5xx = error
		span.Status = int(statusCode) < 400
		if !span.Status {
			span.ErrorMessage = fmt.Sprintf("Status code: %d", int(statusCode))
		}
	}

	// Parse attributes
	span.Attributes = make(map[string]string)
	if attributesRaw, ok := row["attributes"]; ok {
		if attributesMap, ok := attributesRaw.(map[string]interface{}); ok {
			for key, value := range attributesMap {
				if valueStr, ok := value.(string); ok {
					span.Attributes[key] = valueStr
				} else {
					// Convert non-string values to string
					span.Attributes[key] = fmt.Sprintf("%v", value)
				}
			}
		}
	}

	// Extract task name from attributes or operation name
	span.TaskName = s.extractTaskName(span.OperationName, span.Attributes)

	// Parse events (simplified)
	span.Events = []telemetry.SpanEvent{}
	if eventsRaw, ok := row["events"]; ok {
		if eventsList, ok := eventsRaw.([]interface{}); ok {
			for _, eventRaw := range eventsList {
				if eventMap, ok := eventRaw.(map[string]interface{}); ok {
					event := telemetry.SpanEvent{}
					if name, ok := eventMap["name"].(string); ok {
						event.Name = name
					}
					if timestampStr, ok := eventMap["timestamp"].(string); ok {
						if timestamp, err := time.Parse(time.RFC3339, timestampStr); err == nil {
							event.Time = timestamp
						}
					}
					event.Attributes = make(map[string]string)
					if attrs, ok := eventMap["attributes"].(map[string]interface{}); ok {
						for k, v := range attrs {
							event.Attributes[k] = fmt.Sprintf("%v", v)
						}
					}
					span.Events = append(span.Events, event)
				}
			}
		}
	}

	return span, nil
}

// extractTaskName attempts to extract task name from operation name or attributes.
//
// Looks for task name in common attribute keys:
//   - task.name
//   - task_name
//   - function_name
//
// Falls back to operation name if no task-specific attribute found.
func (s *SignozAdapter) extractTaskName(operationName string, attributes map[string]string) string {
	// Try common task name attribute keys
	taskNameKeys := []string{"task.name", "task_name", "function_name", "method_name"}

	for _, key := range taskNameKeys {
		if taskName, exists := attributes[key]; exists && taskName != "" {
			return taskName
		}
	}

	// Special handling for execute_task operations
	if operationName == "execute_task" {
		if taskName, exists := attributes["task"]; exists && taskName != "" {
			return taskName
		}
	}

	// Fall back to operation name
	return operationName
}

// QueryMetrics retrieves metric data points from SigNoz matching the given filter.
//
// Implements TelemetryAdapter.QueryMetrics by:
//  1. Building PromQL-style query for metrics
//  2. Calling SigNoz /api/v1/query_range endpoint
//  3. Parsing response and converting to standard MetricPoint format
//  4. Handling aggregations and time series data
//
// Returns metrics ordered by timestamp (newest first) up to filter.Limit.
// Returns empty slice (not error) if no metrics match criteria.
func (s *SignozAdapter) QueryMetrics(ctx context.Context, filter telemetry.MetricFilter) ([]telemetry.MetricPoint, error) {
	// Build PromQL query based on filter
	query := s.buildMetricQuery(filter)

	// Prepare request payload for SigNoz metrics API
	reqPayload := map[string]interface{}{
		"query": query,
		"start": filter.TimeRange.Start.Unix(),
		"end":   filter.TimeRange.End.Unix(),
		"step":  "60s", // 1 minute step for aggregation
	}

	reqBody, err := json.Marshal(reqPayload)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	// Make request to SigNoz metrics endpoint
	respBody, err := s.makeRequest(ctx, "POST", "/api/v1/query_range", reqBody)
	if err != nil {
		return nil, fmt.Errorf("failed to query SigNoz metrics: %w", err)
	}

	// Parse SigNoz response
	metrics, err := s.parseMetricResponse(respBody, filter.Limit)
	if err != nil {
		return nil, fmt.Errorf("failed to parse metrics response: %w", err)
	}

	return metrics, nil
}

// buildMetricQuery constructs a PromQL query for metric filtering.
//
// SigNoz supports PromQL for querying metrics. This method builds
// PromQL queries based on MetricFilter criteria.
//
// Examples:
//   - Basic metric: "task_duration_seconds"
//   - With labels: "task_duration_seconds{task_name='fetch_user'}"
//   - With aggregation: "avg(task_duration_seconds{task_name='fetch_user'})"
func (s *SignozAdapter) buildMetricQuery(filter telemetry.MetricFilter) string {
	query := filter.MetricName

	// Add label filters
	if len(filter.Labels) > 0 {
		var labelFilters []string
		for key, value := range filter.Labels {
			// Escape special characters for PromQL
			escapedValue := strconv.Quote(value)
			labelFilters = append(labelFilters, fmt.Sprintf("%s=%s", key, escapedValue))
		}
		query += "{" + strings.Join(labelFilters, ",") + "}"
	}

	// Add aggregation if specified
	if filter.Aggregation != "" {
		switch strings.ToLower(filter.Aggregation) {
		case "avg", "sum", "max", "min", "count":
			query = fmt.Sprintf("%s(%s)", filter.Aggregation, query)
		default:
			// Default to avg for unknown aggregations
			query = fmt.Sprintf("avg(%s)", query)
		}
	}

	return query
}

// parseMetricResponse parses SigNoz PromQL query response into standard MetricPoint format.
//
// SigNoz returns PromQL query results in Prometheus-compatible format:
//
//	{
//	  "status": "success",
//	  "data": {
//	    "resultType": "matrix",
//	    "result": [
//	      {
//	        "metric": {"__name__": "metric_name", "label1": "value1"},
//	        "values": [[timestamp, "value"], ...]
//	      }
//	    ]
//	  }
//	}
func (s *SignozAdapter) parseMetricResponse(respBody []byte, limit int) ([]telemetry.MetricPoint, error) {
	var response struct {
		Status string `json:"status"`
		Data   struct {
			ResultType string `json:"resultType"`
			Result     []struct {
				Metric map[string]string `json:"metric"`
				Values [][]interface{}   `json:"values"`
			} `json:"result"`
		} `json:"data"`
	}

	if err := json.Unmarshal(respBody, &response); err != nil {
		return nil, fmt.Errorf("failed to unmarshal response: %w", err)
	}

	if response.Status != "success" {
		return nil, fmt.Errorf("SigNoz metrics query failed: %s", response.Status)
	}

	metrics := make([]telemetry.MetricPoint, 0)

	for _, series := range response.Data.Result {
		for _, value := range series.Values {
			if len(value) != 2 {
				continue // Invalid value format
			}

			// Parse timestamp (first element)
			timestampFloat, ok := value[0].(float64)
			if !ok {
				continue
			}
			timestamp := time.Unix(int64(timestampFloat), 0)

			// Parse value (second element)
			var metricValue float64
			switch v := value[1].(type) {
			case string:
				if parsedValue, err := strconv.ParseFloat(v, 64); err == nil {
					metricValue = parsedValue
				} else {
					continue // Skip unparseable values
				}
			case float64:
				metricValue = v
			default:
				continue // Skip non-numeric values
			}

			// Extract unit from metric name or labels
			unit := s.extractMetricUnit(series.Metric)

			metric := telemetry.MetricPoint{
				Time:   timestamp,
				Value:  metricValue,
				Labels: series.Metric,
				Unit:   unit,
			}

			metrics = append(metrics, metric)

			// Respect limit
			if limit > 0 && len(metrics) >= limit {
				break
			}
		}

		// Break outer loop if limit reached
		if limit > 0 && len(metrics) >= limit {
			break
		}
	}

	// Sort by timestamp (newest first)
	// Simple bubble sort for small datasets
	for i := 0; i < len(metrics); i++ {
		for j := i + 1; j < len(metrics); j++ {
			if metrics[i].Time.Before(metrics[j].Time) {
				metrics[i], metrics[j] = metrics[j], metrics[i]
			}
		}
	}

	return metrics, nil
}

// extractMetricUnit attempts to extract unit information from metric metadata.
//
// Looks for common unit patterns in metric names and labels.
func (s *SignozAdapter) extractMetricUnit(labels map[string]string) string {
	// Check for explicit unit label
	if unit, exists := labels["unit"]; exists {
		return unit
	}

	// Infer unit from metric name
	metricName := labels["__name__"]
	if metricName == "" {
		return ""
	}

	// Common unit patterns
	unitPatterns := map[string]string{
		"_seconds":      "seconds",
		"_milliseconds": "milliseconds",
		"_bytes":        "bytes",
		"_count":        "count",
		"_total":        "count",
		"_ratio":        "ratio",
		"_percent":      "percent",
		"_dollars":      "dollars",
		"_cost":         "dollars",
	}

	for suffix, unit := range unitPatterns {
		if strings.Contains(metricName, suffix) {
			return unit
		}
	}

	return ""
}

// Available returns true if the SigNoz backend is reachable and healthy.
//
// Performs a lightweight health check by calling the SigNoz version endpoint.
// Uses caching to avoid frequent health checks (30 second TTL).
//
// If this returns false, learning controller should gracefully degrade
// rather than failing hard.
func (s *SignozAdapter) Available() bool {
	now := time.Now()

	// Check cache first
	if now.Sub(s.availabilityCache.timestamp) < s.availabilityCache.ttl {
		return s.availabilityCache.value
	}

	// Perform health check
	available := s.checkAvailability()

	// Update cache
	s.availabilityCache.value = available
	s.availabilityCache.timestamp = now

	return available
}

// checkAvailability performs the actual availability check.
//
// Calls SigNoz version endpoint with a short timeout to verify connectivity.
func (s *SignozAdapter) checkAvailability() bool {
	// Create a short-timeout context for health check
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Try to get version information
	_, err := s.makeRequest(ctx, "GET", "/api/v1/version", nil)
	if err != nil {
		return false
	}

	return true
}

// escapeClickHouseString escapes special characters for ClickHouse string literals.
//
// ClickHouse string literals need single quotes escaped.
func escapeClickHouseString(s string) string {
	return strings.ReplaceAll(s, "'", "''")
}
