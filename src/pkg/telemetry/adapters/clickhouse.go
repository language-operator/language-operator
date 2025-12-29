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
	"database/sql"
	"fmt"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"

	_ "github.com/ClickHouse/clickhouse-go/v2"
	"github.com/language-operator/language-operator/pkg/telemetry"
)

// ClickhouseAdapter implements TelemetryAdapter for direct ClickHouse queries.
//
// This adapter connects directly to ClickHouse database where OpenTelemetry
// Collector stores telemetry data. It bypasses external services like SigNoz
// and provides faster, more direct access to trace/metric data.
//
// Key features:
// - Direct ClickHouse SQL queries for traces/metrics
// - Support for OpenTelemetry schema tables
// - Connection pooling and health checks
// - Configurable timeouts and retry logic
type ClickhouseAdapter struct {
	// Database connection
	db *sql.DB

	// Configuration
	endpoint string
	database string
	username string
	password string

	// Health check state
	mu        sync.RWMutex
	lastCheck time.Time
	healthy   bool

	// Configuration from environment
	timeout       time.Duration
	retryAttempts int
	retryBackoff  time.Duration
}

// NewClickhouseAdapter creates a new ClickhouseAdapter from environment variables.
//
// Required environment variables:
//   - TELEMETRY_ADAPTER_ENDPOINT: ClickHouse HTTP endpoint (e.g., "http://clickhouse:8123")
//   - TELEMETRY_ADAPTER_DATABASE: Database name (default: "otel")
//   - TELEMETRY_ADAPTER_USERNAME: Username (default: "default")
//
// Optional environment variables:
//   - TELEMETRY_ADAPTER_PASSWORD: Password (if authentication required)
//   - TELEMETRY_ADAPTER_TIMEOUT: Connection timeout (default: "30s")
//   - TELEMETRY_ADAPTER_RETRY_ATTEMPTS: Retry attempts (default: 3)
//   - TELEMETRY_ADAPTER_RETRY_BACKOFF: Retry backoff (default: "1s")
func NewClickhouseAdapter() (*ClickhouseAdapter, error) {
	endpoint := os.Getenv("TELEMETRY_ADAPTER_ENDPOINT")
	if endpoint == "" {
		return nil, fmt.Errorf("ClickHouse adapter requires TELEMETRY_ADAPTER_ENDPOINT")
	}

	database := os.Getenv("TELEMETRY_ADAPTER_DATABASE")
	if database == "" {
		database = "langop"
	}

	username := os.Getenv("TELEMETRY_ADAPTER_USERNAME")
	if username == "" {
		username = "default"
	}

	password := os.Getenv("TELEMETRY_ADAPTER_PASSWORD")

	// Parse timeout
	timeoutStr := os.Getenv("TELEMETRY_ADAPTER_TIMEOUT")
	if timeoutStr == "" {
		timeoutStr = "30s"
	}
	timeout, err := time.ParseDuration(timeoutStr)
	if err != nil {
		return nil, fmt.Errorf("invalid timeout %s: %w", timeoutStr, err)
	}

	// Parse retry attempts
	retryAttemptsStr := os.Getenv("TELEMETRY_ADAPTER_RETRY_ATTEMPTS")
	if retryAttemptsStr == "" {
		retryAttemptsStr = "3"
	}
	retryAttempts, err := strconv.Atoi(retryAttemptsStr)
	if err != nil {
		return nil, fmt.Errorf("invalid retry attempts %s: %w", retryAttemptsStr, err)
	}

	// Parse retry backoff
	retryBackoffStr := os.Getenv("TELEMETRY_ADAPTER_RETRY_BACKOFF")
	if retryBackoffStr == "" {
		retryBackoffStr = "1s"
	}
	retryBackoff, err := time.ParseDuration(retryBackoffStr)
	if err != nil {
		return nil, fmt.Errorf("invalid retry backoff %s: %w", retryBackoffStr, err)
	}

	adapter := &ClickhouseAdapter{
		endpoint:      endpoint,
		database:      database,
		username:      username,
		password:      password,
		timeout:       timeout,
		retryAttempts: retryAttempts,
		retryBackoff:  retryBackoff,
		healthy:       false,
	}

	// Initialize database connection
	if err := adapter.connect(); err != nil {
		return nil, fmt.Errorf("failed to connect to ClickHouse: %w", err)
	}

	return adapter, nil
}

// connect establishes the database connection
func (c *ClickhouseAdapter) connect() error {
	// Build DSN for ClickHouse
	dsn := c.buildDSN()

	db, err := sql.Open("clickhouse", dsn)
	if err != nil {
		return fmt.Errorf("failed to open ClickHouse connection: %w", err)
	}

	// Configure connection pool
	db.SetMaxOpenConns(10)
	db.SetMaxIdleConns(5)
	db.SetConnMaxLifetime(5 * time.Minute)

	// Test connection
	ctx, cancel := context.WithTimeout(context.Background(), c.timeout)
	defer cancel()

	if err := db.PingContext(ctx); err != nil {
		db.Close()
		return fmt.Errorf("failed to ping ClickHouse: %w", err)
	}

	c.db = db
	c.updateHealthStatus(true)

	return nil
}

// buildDSN constructs the ClickHouse data source name
func (c *ClickhouseAdapter) buildDSN() string {
	// Convert HTTP endpoint to TCP for clickhouse-go driver
	// http://clickhouse:8123 -> tcp://clickhouse:9000
	tcpEndpoint := strings.Replace(c.endpoint, "http://", "tcp://", 1)
	tcpEndpoint = strings.Replace(tcpEndpoint, ":8123", ":9000", 1)

	dsn := fmt.Sprintf("%s?database=%s&username=%s", tcpEndpoint, c.database, c.username)

	if c.password != "" {
		dsn += "&password=" + c.password
	}

	// Skip timeout settings in DSN to avoid parsing issues
	// ClickHouse driver will use default timeout values

	return dsn
}

// QuerySpans retrieves execution spans matching the given filter criteria.
func (c *ClickhouseAdapter) QuerySpans(ctx context.Context, filter telemetry.SpanFilter) ([]telemetry.Span, error) {
	if !c.Available() {
		return nil, fmt.Errorf("ClickHouse adapter is not available")
	}

	query := c.buildSpanQuery(filter)
	args := c.buildSpanArgs(filter)

	rows, err := c.queryWithRetry(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to query spans: %w", err)
	}
	defer rows.Close()

	var spans []telemetry.Span
	for rows.Next() {
		span, err := c.scanSpan(rows)
		if err != nil {
			return nil, fmt.Errorf("failed to scan span: %w", err)
		}
		spans = append(spans, span)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating spans: %w", err)
	}

	return spans, nil
}

// buildSpanQuery constructs the SQL query for spans
func (c *ClickhouseAdapter) buildSpanQuery(filter telemetry.SpanFilter) string {
	query := `
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
		FROM langop.otel_traces
		WHERE 1=1`

	// Add time range filter
	if !filter.TimeRange.Start.IsZero() {
		query += " AND Timestamp >= ?"
	}
	if !filter.TimeRange.End.IsZero() {
		query += " AND Timestamp <= ?"
	}

	// Add task name filter
	if filter.TaskName != "" {
		query += " AND (SpanAttributes['task.name'] = ? OR SpanName LIKE ?)"
	}

	// Add trace ID filter
	if filter.TraceID != "" {
		query += " AND TraceId = ?"
	}

	// Add attribute filters
	for key := range filter.Attributes {
		// Special case: service.name should filter on ServiceName column, not SpanAttributes
		if key == "service.name" {
			query += " AND ServiceName = ?"
		} else {
			query += fmt.Sprintf(" AND SpanAttributes['%s'] = ?", key)
		}
	}

	// Order by timestamp (newest first)
	query += " ORDER BY Timestamp DESC"

	// Add limit
	if filter.Limit > 0 {
		query += " LIMIT ?"
	}

	return query
}

// buildSpanArgs constructs the query arguments for spans
func (c *ClickhouseAdapter) buildSpanArgs(filter telemetry.SpanFilter) []interface{} {
	var args []interface{}

	// Add time range args
	if !filter.TimeRange.Start.IsZero() {
		args = append(args, filter.TimeRange.Start)
	}
	if !filter.TimeRange.End.IsZero() {
		args = append(args, filter.TimeRange.End)
	}

	// Add task name args
	if filter.TaskName != "" {
		args = append(args, filter.TaskName, "%"+filter.TaskName+"%")
	}

	// Add trace ID arg
	if filter.TraceID != "" {
		args = append(args, filter.TraceID)
	}

	// Add attribute args
	for _, value := range filter.Attributes {
		args = append(args, value)
	}

	// Add limit arg
	if filter.Limit > 0 {
		args = append(args, filter.Limit)
	}

	return args
}

// scanSpan scans a database row into a telemetry.Span
func (c *ClickhouseAdapter) scanSpan(rows *sql.Rows) (telemetry.Span, error) {
	var span telemetry.Span
	var spanId, traceId, parentSpanId, spanName, spanKind string
	var timestamp time.Time
	var duration int64
	var statusCode int
	var statusMessage string
	var attributesJson string

	err := rows.Scan(
		&spanId, &traceId, &parentSpanId, &spanName, &spanKind,
		&timestamp, &duration, &statusCode, &statusMessage, &attributesJson,
	)
	if err != nil {
		return span, err
	}

	span.SpanID = spanId
	span.TraceID = traceId
	span.ParentSpanID = parentSpanId
	span.OperationName = spanName
	span.StartTime = timestamp
	span.Duration = time.Duration(duration) * time.Nanosecond
	span.EndTime = timestamp.Add(span.Duration)
	span.Status = statusCode == 1 // StatusCode 1 = OK
	span.ErrorMessage = statusMessage

	// Parse attributes from JSON (simplified - could use JSON parsing)
	span.Attributes = map[string]string{}
	if attributesJson != "" {
		// Basic attribute parsing - in production, use proper JSON parsing
		if strings.Contains(attributesJson, "task.name") {
			// Extract task name from attributes
			span.TaskName = extractTaskName(attributesJson)
		}
	}

	return span, nil
}

// extractTaskName extracts task name from span attributes (simplified)
func extractTaskName(attributesJson string) string {
	// Simplified extraction - in production, use proper JSON parsing
	if strings.Contains(attributesJson, "task.name") {
		// This is a basic implementation - should use JSON parsing
		return "extracted_task_name"
	}
	return ""
}

// QueryMetrics retrieves metric data points matching the given filter.
func (c *ClickhouseAdapter) QueryMetrics(ctx context.Context, filter telemetry.MetricFilter) ([]telemetry.MetricPoint, error) {
	if !c.Available() {
		return nil, fmt.Errorf("ClickHouse adapter is not available")
	}

	// Build metrics query (simplified implementation)
	query := `
		SELECT 
			TimeUnix,
			MetricName,
			Value,
			Attributes
		FROM langop.otel_metrics_gauge
		WHERE MetricName = ?
		AND TimeUnix >= ? AND TimeUnix <= ?
		ORDER BY TimeUnix DESC
		LIMIT ?`

	args := []interface{}{
		filter.MetricName,
		filter.TimeRange.Start.Unix(),
		filter.TimeRange.End.Unix(),
		filter.Limit,
	}

	rows, err := c.queryWithRetry(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to query metrics: %w", err)
	}
	defer rows.Close()

	var metrics []telemetry.MetricPoint
	for rows.Next() {
		var timeUnix int64
		var metricName string
		var value float64
		var attributesJson string

		if err := rows.Scan(&timeUnix, &metricName, &value, &attributesJson); err != nil {
			return nil, fmt.Errorf("failed to scan metric: %w", err)
		}

		metric := telemetry.MetricPoint{
			Time:   time.Unix(timeUnix, 0),
			Value:  value,
			Labels: map[string]string{}, // Parse from attributesJson in production
			Unit:   "",                  // Extract from metric metadata
		}

		metrics = append(metrics, metric)
	}

	return metrics, nil
}

// Available returns true if the ClickHouse database is reachable and healthy.
func (c *ClickhouseAdapter) Available() bool {
	c.mu.RLock()
	// Check if we need to refresh health status
	if time.Since(c.lastCheck) > time.Minute {
		c.mu.RUnlock()
		c.checkHealth()
		c.mu.RLock()
	}
	healthy := c.healthy
	c.mu.RUnlock()
	return healthy
}

// checkHealth performs a health check against ClickHouse
func (c *ClickhouseAdapter) checkHealth() {
	if c.db == nil {
		c.updateHealthStatus(false)
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Simple ping check
	err := c.db.PingContext(ctx)
	c.updateHealthStatus(err == nil)
}

// updateHealthStatus updates the health status with proper locking
func (c *ClickhouseAdapter) updateHealthStatus(healthy bool) {
	c.mu.Lock()
	c.healthy = healthy
	c.lastCheck = time.Now()
	c.mu.Unlock()
}

// queryWithRetry executes a query with retry logic
func (c *ClickhouseAdapter) queryWithRetry(ctx context.Context, query string, args ...interface{}) (*sql.Rows, error) {
	var rows *sql.Rows
	var err error

	for attempt := 0; attempt <= c.retryAttempts; attempt++ {
		if attempt > 0 {
			// Wait before retry
			select {
			case <-time.After(c.retryBackoff * time.Duration(attempt)):
			case <-ctx.Done():
				return nil, ctx.Err()
			}
		}

		queryCtx, cancel := context.WithTimeout(ctx, c.timeout)
		rows, err = c.db.QueryContext(queryCtx, query, args...)
		cancel()

		if err == nil {
			c.updateHealthStatus(true)
			return rows, nil
		}

		// Update health status on persistent errors
		if attempt == c.retryAttempts {
			c.updateHealthStatus(false)
		}
	}

	return nil, fmt.Errorf("query failed after %d attempts: %w", c.retryAttempts+1, err)
}

// Close closes the database connection
func (c *ClickhouseAdapter) Close() error {
	if c.db != nil {
		return c.db.Close()
	}
	return nil
}
