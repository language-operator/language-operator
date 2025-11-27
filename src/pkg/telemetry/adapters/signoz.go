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
	"io"
	"net"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/language-operator/language-operator/pkg/telemetry"
)

const (
	// DefaultMaxResponseSize is the default maximum size for HTTP response bodies (50MB)
	// This prevents memory exhaustion from large telemetry datasets
	DefaultMaxResponseSize = 50 * 1024 * 1024 // 50MB
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

	// maxResponseSize is the maximum allowed size for HTTP response bodies
	// Prevents memory exhaustion from large telemetry responses
	maxResponseSize int64

	// availabilityCache caches the result of Available() checks
	// to avoid frequent health checks
	availabilityCache struct {
		sync.RWMutex
		value     bool
		timestamp time.Time
		ttl       time.Duration
		timeNow   func() time.Time // Injectable for testing
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

	// MaxResponseSize is the maximum allowed size for HTTP response bodies
	// Defaults to 50MB (50 * 1024 * 1024 bytes) if not specified
	// Prevents memory exhaustion from large telemetry datasets
	MaxResponseSize int64
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
	return NewSignozAdapterWithMaxSize(endpoint, apiKey, timeout, DefaultMaxResponseSize)
}

// NewSignozAdapterWithMaxSize creates a new SignozAdapter with custom response size limit.
func NewSignozAdapterWithMaxSize(endpoint, apiKey string, timeout time.Duration, maxResponseSize int64) (*SignozAdapter, error) {
	if endpoint == "" {
		return nil, fmt.Errorf("endpoint cannot be empty")
	}

	if apiKey == "" {
		return nil, fmt.Errorf("apiKey cannot be empty")
	}

	if timeout <= 0 {
		return nil, fmt.Errorf("timeout must be positive, got %v", timeout)
	}

	if maxResponseSize <= 0 {
		return nil, fmt.Errorf("maxResponseSize must be positive, got %d", maxResponseSize)
	}

	// Validate endpoint is a proper URL
	parsedURL, err := url.Parse(endpoint)
	if err != nil {
		return nil, fmt.Errorf("invalid endpoint URL: %w", err)
	}

	// Validate URL has required scheme and host
	if parsedURL.Scheme == "" {
		return nil, fmt.Errorf("endpoint URL must include scheme (http or https): %s", endpoint)
	}
	if parsedURL.Scheme != "http" && parsedURL.Scheme != "https" {
		return nil, fmt.Errorf("endpoint URL scheme must be http or https, got: %s", parsedURL.Scheme)
	}
	if parsedURL.Host == "" {
		return nil, fmt.Errorf("endpoint URL must include host: %s", endpoint)
	}

	// Enhanced validation to prevent runtime panics
	if err := validateHost(parsedURL.Host); err != nil {
		return nil, fmt.Errorf("invalid endpoint host: %w", err)
	}

	// Remove trailing slash from endpoint for consistent URL building
	endpoint = strings.TrimSuffix(endpoint, "/")

	adapter := &SignozAdapter{
		endpoint:        endpoint,
		apiKey:          apiKey,
		timeout:         timeout,
		maxResponseSize: maxResponseSize,
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
	adapter.availabilityCache.timeNow = time.Now

	return adapter, nil
}

// NewSignozAdapterFromConfig creates a SignozAdapter from a config struct.
//
// Applies default values for optional fields:
//   - Timeout: 30 seconds if not specified
//   - MaxResponseSize: 50MB if not specified
//   - RetryCount: 3 if not specified (currently unused)
func NewSignozAdapterFromConfig(config SignozConfig) (*SignozAdapter, error) {
	timeout := config.Timeout
	if timeout == 0 {
		timeout = 30 * time.Second
	}

	maxResponseSize := config.MaxResponseSize
	if maxResponseSize == 0 {
		maxResponseSize = DefaultMaxResponseSize
	}

	return NewSignozAdapterWithMaxSize(config.Endpoint, config.APIKey, timeout, maxResponseSize)
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

	// Read response body with size limit to prevent memory exhaustion
	limitedReader := io.LimitReader(resp.Body, s.maxResponseSize)
	respBody, err := io.ReadAll(limitedReader)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	// Check if response was truncated due to size limit
	if int64(len(respBody)) == s.maxResponseSize {
		// Try to read one more byte to confirm truncation
		buf := make([]byte, 1)
		n, _ := resp.Body.Read(buf)
		if n > 0 {
			return nil, fmt.Errorf("response body exceeds maximum allowed size of %d bytes", s.maxResponseSize)
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
//  1. Building Query Builder v5 payload with filter expressions
//  2. Calling SigNoz /api/v5/query_range endpoint
//  3. Parsing response and converting to standard Span format
//  4. Handling pagination and result limits
//
// The method constructs Query Builder v5 requests to filter spans by:
//   - Time range (start/end timestamps in milliseconds)
//   - Task name (extracted from span attributes)
//   - Trace ID (exact match)
//   - Custom attributes (key-value pairs)
//
// Returns spans ordered by timestamp (newest first) up to filter.Limit.
// Returns empty slice (not error) if no spans match criteria.
func (s *SignozAdapter) QuerySpans(ctx context.Context, filter telemetry.SpanFilter) ([]telemetry.Span, error) {
	// Build Query Builder v5 payload
	reqPayload := s.buildQueryBuilderV5Payload(filter)

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

// buildQueryBuilderV5Payload constructs a Query Builder v5 payload for SigNoz trace queries.
//
// SigNoz Query Builder v5 uses a structured JSON format instead of raw ClickHouse SQL.
// The payload structure follows the pattern:
//   - start/end: timestamps in milliseconds
//   - requestType: "raw" for span data retrieval
//   - compositeQuery: contains query specifications
//   - queries[].spec: defines filtering, ordering, and result selection
//
// This replaces the old ClickHouse SQL-based approach with expression-based filtering.
func (s *SignozAdapter) buildQueryBuilderV5Payload(filter telemetry.SpanFilter) map[string]interface{} {
	// Convert timestamps to milliseconds (SigNoz v5 requirement)
	startMs := filter.TimeRange.Start.UnixMilli()
	endMs := filter.TimeRange.End.Unix() * 1000 // Handle cases where UnixMilli() might not be available
	if filter.TimeRange.End.UnixNano() > 0 {
		endMs = filter.TimeRange.End.UnixNano() / 1000000
	}

	// Build filter expression based on SpanFilter criteria
	filterExpression := s.buildFilterExpression(filter)

	// Define select fields for span data
	selectFields := []map[string]string{
		{"name": "spanID"},
		{"name": "traceID"},
		{"name": "parentSpanID"},
		{"name": "operationName"},
		{"name": "timestamp"},
		{"name": "duration"},
		{"name": "statusCode"},
		{"name": "attributes"},
		{"name": "events"},
	}

	// Build Query Builder v5 payload structure
	payload := map[string]interface{}{
		"start":       startMs,
		"end":         endMs,
		"requestType": "raw",
		"variables":   map[string]interface{}{},
		"compositeQuery": map[string]interface{}{
			"queries": []map[string]interface{}{
				{
					"type": "builder_query",
					"spec": map[string]interface{}{
						"name":   "A",
						"signal": "traces",
						"filter": map[string]interface{}{
							"expression": filterExpression,
						},
						"selectFields": selectFields,
						"order": []map[string]interface{}{
							{
								"key":       map[string]string{"name": "timestamp"},
								"direction": "desc",
							},
						},
						"limit":    filter.Limit,
						"offset":   0,
						"disabled": false,
					},
				},
			},
		},
	}

	return payload
}

// buildFilterExpression constructs a SigNoz Query Builder v5 filter expression.
//
// Converts SpanFilter criteria to expression syntax that SigNoz understands.
// Combines multiple conditions with AND operators.
func (s *SignozAdapter) buildFilterExpression(filter telemetry.SpanFilter) string {
	var conditions []string

	// Task name filter - check multiple possible attribute keys
	if filter.TaskName != "" {
		taskConditions := []string{
			fmt.Sprintf("attributes['task.name'] = '%s'", s.escapeFilterValue(filter.TaskName)),
			fmt.Sprintf("attributes['task_name'] = '%s'", s.escapeFilterValue(filter.TaskName)),
			fmt.Sprintf("operationName = '%s'", s.escapeFilterValue(filter.TaskName)),
		}
		conditions = append(conditions, "("+strings.Join(taskConditions, " OR ")+")")
	}

	// Trace ID filter
	if filter.TraceID != "" {
		conditions = append(conditions, fmt.Sprintf("traceID = '%s'", s.escapeFilterValue(filter.TraceID)))
	}

	// Custom attributes filter
	for key, value := range filter.Attributes {
		conditions = append(conditions, fmt.Sprintf(
			"attributes['%s'] = '%s'",
			s.escapeFilterValue(key),
			s.escapeFilterValue(value)))
	}

	// Combine all conditions with AND
	if len(conditions) == 0 {
		return "" // No specific filters, return all spans in time range
	}

	return strings.Join(conditions, " AND ")
}

// escapeFilterValue escapes special characters for SigNoz filter expressions.
//
// SigNoz filter expressions need single quotes escaped to prevent injection.
func (s *SignozAdapter) escapeFilterValue(value string) string {
	// Escape single quotes for filter expression safety
	return strings.ReplaceAll(value, "'", "''")
}

// buildSpanQuery constructs a ClickHouse SQL query for span filtering.
//
// DEPRECATED: This method is kept for backward compatibility but should not be used
// with SigNoz v0.103.0+. Use buildQueryBuilderV5Payload instead.
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

// parseSpanResponse parses SigNoz Query Builder v5 response into standard Span format.
//
// SigNoz Query Builder v5 returns query results in JSON format with structure:
//
//	{
//	  "status": "success",
//	  "data": {
//	    "resultType": "list",
//	    "result": [
//	      {
//	        "spanID": "...",
//	        "traceID": "...",
//	        "timestamp": "2023-11-27T10:00:00Z",
//	        "attributes": {...},
//	        ...
//	      }
//	    ]
//	  }
//	}
//
// This handles the new Query Builder v5 response format for trace queries.
func (s *SignozAdapter) parseSpanResponse(respBody []byte, limit int) ([]telemetry.Span, error) {
	// First, try to parse as Query Builder v5 response format
	var v5Response struct {
		Status string `json:"status"`
		Data   struct {
			ResultType string                   `json:"resultType"`
			Result     []map[string]interface{} `json:"result"`
		} `json:"data"`
	}

	if err := json.Unmarshal(respBody, &v5Response); err != nil {
		return nil, fmt.Errorf("failed to unmarshal v5 response: %w", err)
	}

	// Check if it's a successful Query Builder v5 response
	if v5Response.Status == "success" && v5Response.Data.ResultType == "list" {
		return s.parseV5SpanResponse(v5Response.Data.Result, limit)
	}

	// Fallback: try to parse as old response format for backward compatibility
	var legacyResponse struct {
		Data struct {
			Result []map[string]interface{} `json:"result"`
		} `json:"data"`
	}

	if err := json.Unmarshal(respBody, &legacyResponse); err != nil {
		return nil, fmt.Errorf("failed to unmarshal legacy response: %w", err)
	}

	return s.parseV5SpanResponse(legacyResponse.Data.Result, limit)
}

// parseV5SpanResponse processes Query Builder v5 span result data.
//
// Converts the raw Query Builder v5 span data to our standard telemetry.Span format.
func (s *SignozAdapter) parseV5SpanResponse(result []map[string]interface{}, limit int) ([]telemetry.Span, error) {
	spans := make([]telemetry.Span, 0, len(result))

	for _, row := range result {
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
	now := s.availabilityCache.timeNow()

	// Check cache first with read lock
	s.availabilityCache.RLock()
	if now.Sub(s.availabilityCache.timestamp) < s.availabilityCache.ttl {
		value := s.availabilityCache.value
		s.availabilityCache.RUnlock()
		return value
	}
	s.availabilityCache.RUnlock()

	// Perform health check
	available := s.checkAvailability()

	// Update cache with write lock
	s.availabilityCache.Lock()
	s.availabilityCache.value = available
	s.availabilityCache.timestamp = now
	s.availabilityCache.Unlock()

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

// validateHost performs comprehensive host validation to prevent runtime panics.
//
// Validates that the host portion of a URL is properly formatted and will not
// cause issues during HTTP client operations. This catches edge cases that
// url.Parse() accepts but cause panics in http.Client.
//
// Common problematic patterns caught:
//   - Empty host with port: ":3000"
//   - Host with trailing colon: "localhost:"
//   - Invalid port numbers: "localhost:99999"
//   - Malformed IPv6: "[::1"
//   - Empty hostname: "."
//
// Returns nil if host is valid, error describing the problem otherwise.
func validateHost(host string) error {
	if host == "" {
		return fmt.Errorf("host cannot be empty")
	}

	// Check for common malformed patterns
	// If starts with colon, check if it's just a port (invalid) or valid IPv6
	if strings.HasPrefix(host, ":") {
		// If it's just a port number, that's invalid
		if len(host) > 1 && host[1] >= '0' && host[1] <= '9' {
			return fmt.Errorf("host cannot start with colon (empty hostname): %s", host)
		}
		// Otherwise it might be IPv6, let normal validation handle it
	}

	if strings.HasSuffix(host, ":") {
		return fmt.Errorf("host cannot end with colon (missing port): %s", host)
	}

	// Split host and port for individual validation
	hostname, port, err := net.SplitHostPort(host)
	hadBrackets := strings.HasPrefix(host, "[") && strings.Contains(host, "]:")
	if err != nil {
		// If SplitHostPort fails, treat the entire host as hostname (no port)
		hostname = host
		port = ""
		hadBrackets = strings.HasPrefix(host, "[") && strings.HasSuffix(host, "]")
	}

	// Validate hostname if present
	if hostname != "" {
		if err := validateHostname(hostname); err != nil {
			return fmt.Errorf("invalid hostname: %w", err)
		}
		// Special check: if hostname came from IPv6 brackets, ensure it's actually IPv6
		if hadBrackets {
			if ip := net.ParseIP(hostname); ip == nil || ip.To4() != nil {
				return fmt.Errorf("IPv4 address in IPv6 brackets: [%s]", hostname)
			}
		}
	} else if port != "" {
		// Port specified but no hostname
		return fmt.Errorf("port specified but hostname is empty: %s", host)
	}

	// Validate port if present
	if port != "" {
		if err := validatePort(port); err != nil {
			return fmt.Errorf("invalid port: %w", err)
		}
	}

	return nil
}

// validateHostname validates that a hostname is properly formatted.
//
// This performs basic validation to catch common issues that could cause
// DNS resolution or connection failures.
func validateHostname(hostname string) error {
	if hostname == "" {
		return fmt.Errorf("hostname cannot be empty")
	}

	// Check for obviously invalid patterns
	if hostname == "." {
		return fmt.Errorf("hostname cannot be single dot")
	}

	if strings.HasPrefix(hostname, ".") {
		return fmt.Errorf("hostname cannot start with dot: %s", hostname)
	}

	// IPv6 addresses should be bracketed when used with ports
	// If we see unbracketed IPv6, it's fine for hostname-only
	if strings.Contains(hostname, ":") && !strings.HasPrefix(hostname, "[") {
		// Could be IPv6 without brackets - check if it parses as IP
		if ip := net.ParseIP(hostname); ip == nil {
			return fmt.Errorf("hostname contains colon but is not valid IPv6: %s", hostname)
		}
		// It's valid IPv6, which is fine for hostname-only (will work with http.Client)
		return nil
	}

	// Additional IPv6 validation for bracketed form (only when no port splitting occurred)
	if strings.HasPrefix(hostname, "[") && strings.HasSuffix(hostname, "]") {
		ipv6 := hostname[1 : len(hostname)-1]
		ip := net.ParseIP(ipv6)
		if ip == nil {
			return fmt.Errorf("invalid IPv6 address: %s", hostname)
		}
		if ip.To4() != nil {
			return fmt.Errorf("IPv4 address in IPv6 brackets: %s", hostname)
		}
	}

	return nil
}

// validatePort validates that a port number is within valid range.
//
// Port numbers must be in range 1-65535. Port 0 is technically valid in some
// contexts but causes issues with HTTP clients, so we reject it.
func validatePort(portStr string) error {
	if portStr == "" {
		return fmt.Errorf("port cannot be empty")
	}

	port, err := strconv.Atoi(portStr)
	if err != nil {
		return fmt.Errorf("port must be numeric: %s", portStr)
	}

	if port < 1 {
		return fmt.Errorf("port must be positive (1-65535), got: %d", port)
	}

	if port > 65535 {
		return fmt.Errorf("port must be ≤65535, got: %d", port)
	}

	return nil
}
