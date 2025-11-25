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

package telemetry

import (
	"context"
	"time"
)

// TelemetryAdapter provides an interface for querying observability backends
// to retrieve historical execution data for the learning system.
//
// Implementations should handle backend-specific protocols (gRPC, HTTP, etc.)
// and data formats while presenting a unified interface to the learning controller.
//
// The adapter is responsible for:
// - Querying trace data for task execution patterns
// - Retrieving metrics for performance analysis
// - Converting backend-specific formats to standardized structs
// - Handling errors gracefully with appropriate timeouts
//
// Example implementations:
// - SigNozAdapter: Query ClickHouse database via SigNoz API
// - JaegerAdapter: Query Jaeger via gRPC or HTTP API
// - PrometheusAdapter: Query metrics via PromQL
type TelemetryAdapter interface {
	// QuerySpans retrieves execution spans matching the given filter criteria.
	//
	// Returns spans ordered by timestamp (newest first) up to filter.Limit.
	// If no spans match the criteria, returns empty slice (not an error).
	//
	// Context should include reasonable timeout (5-30s recommended).
	// Implementations should respect context cancellation.
	//
	// Example:
	//   spans, err := adapter.QuerySpans(ctx, SpanFilter{
	//     TaskName: "fetch_user",
	//     TimeRange: TimeRange{
	//       Start: time.Now().Add(-24*time.Hour),
	//       End:   time.Now(),
	//     },
	//     Limit: 50,
	//   })
	QuerySpans(ctx context.Context, filter SpanFilter) ([]Span, error)

	// QueryMetrics retrieves metric data points matching the given filter.
	//
	// Returns metrics ordered by timestamp (newest first) up to filter.Limit.
	// Useful for cost analysis, performance trends, and error rates.
	//
	// Example:
	//   metrics, err := adapter.QueryMetrics(ctx, MetricFilter{
	//     MetricName: "task_duration_seconds",
	//     Labels: map[string]string{"task_name": "fetch_user"},
	//     TimeRange: TimeRange{Start: yesterday, End: now},
	//     Limit: 100,
	//   })
	QueryMetrics(ctx context.Context, filter MetricFilter) ([]MetricPoint, error)

	// Available returns true if the telemetry backend is reachable and healthy.
	//
	// Should perform a lightweight health check (e.g., ping, version query).
	// Used by learning controller to decide whether to attempt queries.
	//
	// If this returns false, learning controller should gracefully degrade
	// rather than failing hard.
	Available() bool
}

// SpanFilter specifies criteria for querying execution spans.
type SpanFilter struct {
	// TaskName filters spans for a specific task (e.g., "fetch_user").
	// If empty, returns spans for all tasks.
	TaskName string

	// TimeRange restricts spans to a specific time window.
	// Both Start and End are inclusive.
	TimeRange TimeRange

	// TraceID filters spans belonging to a specific trace.
	// Useful for debugging individual agent executions.
	TraceID string

	// Attributes filters spans by custom attributes.
	// Matches spans where ALL specified attributes match exactly.
	//
	// Common attributes:
	// - "agent.name": Agent instance name
	// - "task.type": "neural" or "symbolic"
	// - "synthesis.version": ConfigMap version
	Attributes map[string]string

	// Limit restricts the maximum number of spans returned.
	// Backends may enforce their own lower limits.
	// Recommended: 10-1000 depending on use case.
	Limit int
}

// MetricFilter specifies criteria for querying metric data.
type MetricFilter struct {
	// MetricName is the name of the metric to query.
	// Examples: "task_duration_seconds", "llm_cost_dollars", "error_rate"
	MetricName string

	// Labels filters metrics by label values.
	// Matches metrics where ALL specified labels match exactly.
	Labels map[string]string

	// TimeRange restricts metrics to a specific time window.
	TimeRange TimeRange

	// Aggregation specifies how to aggregate metric values over time.
	// Examples: "avg", "sum", "max", "min", "count"
	Aggregation string

	// Limit restricts the maximum number of data points returned.
	Limit int
}

// TimeRange represents a time window for queries.
type TimeRange struct {
	// Start is the beginning of the time window (inclusive).
	Start time.Time

	// End is the end of the time window (inclusive).
	End time.Time
}

// IsValid returns true if the time range is valid (Start <= End).
func (tr TimeRange) IsValid() bool {
	return !tr.Start.After(tr.End)
}

// Duration returns the duration of the time range.
func (tr TimeRange) Duration() time.Duration {
	return tr.End.Sub(tr.Start)
}

// Contains returns true if the given time falls within the range (inclusive).
func (tr TimeRange) Contains(t time.Time) bool {
	return !t.Before(tr.Start) && !t.After(tr.End)
}

// Span represents a single execution span from the telemetry backend.
//
// This is a simplified, learning-focused view of OpenTelemetry spans.
// It contains only the fields needed for pattern detection and analysis.
type Span struct {
	// SpanID uniquely identifies this span within its trace.
	SpanID string

	// TraceID identifies the trace this span belongs to.
	TraceID string

	// ParentSpanID identifies the parent span (empty if root span).
	ParentSpanID string

	// OperationName describes what this span represents.
	// Examples: "execute_task", "tool_call", "llm_request"
	OperationName string

	// TaskName is the name of the task being executed (if applicable).
	// Extracted from span attributes or derived from operation name.
	TaskName string

	// StartTime is when the operation started.
	StartTime time.Time

	// EndTime is when the operation completed.
	EndTime time.Time

	// Duration is the execution time (EndTime - StartTime).
	Duration time.Duration

	// Status indicates whether the operation succeeded.
	// True = success, False = error/failure
	Status bool

	// ErrorMessage contains error details if Status is false.
	ErrorMessage string

	// Attributes contains key-value pairs from the span.
	// Common learning-relevant attributes:
	// - "task.inputs": JSON string of task inputs
	// - "task.outputs": JSON string of task outputs  
	// - "tool.name": Name of tool called
	// - "tool.method": Method/function called on tool
	// - "llm.model": Model name for LLM calls
	// - "llm.tokens.input": Input token count
	// - "llm.tokens.output": Output token count
	// - "synthesis.version": ConfigMap version
	Attributes map[string]string

	// Events contains timestamped log events within the span.
	// Useful for understanding execution flow and debugging.
	Events []SpanEvent
}

// SpanEvent represents a timestamped event within a span.
type SpanEvent struct {
	// Time when the event occurred.
	Time time.Time

	// Name of the event.
	// Examples: "tool_call_started", "llm_response_received", "error_occurred"
	Name string

	// Attributes contains event-specific data.
	Attributes map[string]string
}

// MetricPoint represents a single metric measurement at a point in time.
type MetricPoint struct {
	// Time when the metric was recorded.
	Time time.Time

	// Value is the metric value (float64 for maximum precision).
	Value float64

	// Labels contains the metric labels.
	Labels map[string]string

	// Unit describes what the value represents.
	// Examples: "seconds", "bytes", "count", "dollars"
	Unit string
}

// MockAdapter provides a no-op implementation for testing and development.
//
// Always returns empty results and reports as available.
// Useful for unit testing learning controller logic without requiring
// a real telemetry backend.
type MockAdapter struct {
	// AvailableReturn controls what Available() returns.
	// Defaults to true.
	AvailableReturn bool

	// SpanResults contains pre-configured spans to return from QuerySpans.
	// If nil, QuerySpans returns empty slice.
	SpanResults []Span

	// MetricResults contains pre-configured metrics to return from QueryMetrics.
	// If nil, QueryMetrics returns empty slice.
	MetricResults []MetricPoint
}

// NewMockAdapter creates a new MockAdapter with default settings.
func NewMockAdapter() *MockAdapter {
	return &MockAdapter{
		AvailableReturn: true,
		SpanResults:     []Span{},
		MetricResults:   []MetricPoint{},
	}
}

// QuerySpans returns the pre-configured span results.
func (m *MockAdapter) QuerySpans(ctx context.Context, filter SpanFilter) ([]Span, error) {
	if m.SpanResults == nil {
		return []Span{}, nil
	}
	return m.SpanResults, nil
}

// QueryMetrics returns the pre-configured metric results.
func (m *MockAdapter) QueryMetrics(ctx context.Context, filter MetricFilter) ([]MetricPoint, error) {
	if m.MetricResults == nil {
		return []MetricPoint{}, nil
	}
	return m.MetricResults, nil
}

// Available returns the configured availability status.
func (m *MockAdapter) Available() bool {
	return m.AvailableReturn
}

// NoOpAdapter provides an adapter that always returns empty results.
//
// Reports as unavailable (Available() returns false).
// Useful as a fallback when no real telemetry backend is configured.
type NoOpAdapter struct{}

// NewNoOpAdapter creates a new NoOpAdapter.
func NewNoOpAdapter() *NoOpAdapter {
	return &NoOpAdapter{}
}

// QuerySpans always returns empty results.
func (n *NoOpAdapter) QuerySpans(ctx context.Context, filter SpanFilter) ([]Span, error) {
	return []Span{}, nil
}

// QueryMetrics always returns empty results.
func (n *NoOpAdapter) QueryMetrics(ctx context.Context, filter MetricFilter) ([]MetricPoint, error) {
	return []MetricPoint{}, nil
}

// Available always returns false.
func (n *NoOpAdapter) Available() bool {
	return false
}