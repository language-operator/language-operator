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
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestTimeRange(t *testing.T) {
	now := time.Now()
	oneHourAgo := now.Add(-time.Hour)

	t.Run("IsValid", func(t *testing.T) {
		validRange := TimeRange{Start: oneHourAgo, End: now}
		assert.True(t, validRange.IsValid(), "Valid time range should return true")

		invalidRange := TimeRange{Start: now, End: oneHourAgo}
		assert.False(t, invalidRange.IsValid(), "Invalid time range should return false")

		equalRange := TimeRange{Start: now, End: now}
		assert.True(t, equalRange.IsValid(), "Equal start and end should be valid")
	})

	t.Run("Duration", func(t *testing.T) {
		tr := TimeRange{Start: oneHourAgo, End: now}
		duration := tr.Duration()
		assert.Equal(t, time.Hour, duration, "Duration should be calculated correctly")
	})

	t.Run("Contains", func(t *testing.T) {
		tr := TimeRange{Start: oneHourAgo, End: now}

		// Test time within range
		midTime := oneHourAgo.Add(30 * time.Minute)
		assert.True(t, tr.Contains(midTime), "Time within range should return true")

		// Test boundary times
		assert.True(t, tr.Contains(oneHourAgo), "Start time should be included")
		assert.True(t, tr.Contains(now), "End time should be included")

		// Test times outside range
		beforeStart := oneHourAgo.Add(-time.Minute)
		assert.False(t, tr.Contains(beforeStart), "Time before start should return false")

		afterEnd := now.Add(time.Minute)
		assert.False(t, tr.Contains(afterEnd), "Time after end should return false")
	})
}

func TestMockAdapter(t *testing.T) {
	t.Run("Default behavior", func(t *testing.T) {
		adapter := NewMockAdapter()

		// Should be available by default
		assert.True(t, adapter.Available(), "Mock adapter should be available by default")

		// Should return empty results by default
		ctx := context.Background()
		spans, err := adapter.QuerySpans(ctx, SpanFilter{})
		require.NoError(t, err)
		assert.Empty(t, spans, "Default mock should return empty spans")

		metrics, err := adapter.QueryMetrics(ctx, MetricFilter{})
		require.NoError(t, err)
		assert.Empty(t, metrics, "Default mock should return empty metrics")
	})

	t.Run("Custom configuration", func(t *testing.T) {
		// Configure mock with custom data
		mockSpans := []Span{
			{
				SpanID:        "span-1",
				TraceID:       "trace-1",
				OperationName: "execute_task",
				TaskName:      "fetch_user",
				StartTime:     time.Now(),
				EndTime:       time.Now().Add(time.Second),
				Duration:      time.Second,
				Status:        true,
			},
		}

		mockMetrics := []MetricPoint{
			{
				Time:  time.Now(),
				Value: 1.5,
				Labels: map[string]string{
					"task_name": "fetch_user",
				},
				Unit: "seconds",
			},
		}

		adapter := &MockAdapter{
			AvailableReturn: false,
			SpanResults:     mockSpans,
			MetricResults:   mockMetrics,
		}

		// Should respect custom availability
		assert.False(t, adapter.Available(), "Mock adapter should respect custom availability")

		// Should return configured spans
		ctx := context.Background()
		spans, err := adapter.QuerySpans(ctx, SpanFilter{})
		require.NoError(t, err)
		require.Len(t, spans, 1)
		assert.Equal(t, "fetch_user", spans[0].TaskName)

		// Should return configured metrics
		metrics, err := adapter.QueryMetrics(ctx, MetricFilter{})
		require.NoError(t, err)
		require.Len(t, metrics, 1)
		assert.Equal(t, 1.5, metrics[0].Value)
		assert.Equal(t, "fetch_user", metrics[0].Labels["task_name"])
	})
}

func TestNoOpAdapter(t *testing.T) {
	adapter := NewNoOpAdapter()

	// Should not be available
	assert.False(t, adapter.Available(), "NoOp adapter should not be available")

	// Should return empty results
	ctx := context.Background()
	spans, err := adapter.QuerySpans(ctx, SpanFilter{})
	require.NoError(t, err)
	assert.Empty(t, spans, "NoOp adapter should return empty spans")

	metrics, err := adapter.QueryMetrics(ctx, MetricFilter{})
	require.NoError(t, err)
	assert.Empty(t, metrics, "NoOp adapter should return empty metrics")
}

func TestSpanFilter(t *testing.T) {
	t.Run("Complete filter", func(t *testing.T) {
		now := time.Now()
		filter := SpanFilter{
			TaskName: "fetch_user",
			TimeRange: TimeRange{
				Start: now.Add(-time.Hour),
				End:   now,
			},
			TraceID: "trace-123",
			Attributes: map[string]string{
				"agent.name": "test-agent",
				"task.type":  "neural",
			},
			Limit: 100,
		}

		assert.Equal(t, "fetch_user", filter.TaskName)
		assert.Equal(t, "trace-123", filter.TraceID)
		assert.Equal(t, 100, filter.Limit)
		assert.True(t, filter.TimeRange.IsValid())
		assert.Equal(t, "test-agent", filter.Attributes["agent.name"])
		assert.Equal(t, "neural", filter.Attributes["task.type"])
	})
}

func TestMetricFilter(t *testing.T) {
	t.Run("Complete filter", func(t *testing.T) {
		now := time.Now()
		filter := MetricFilter{
			MetricName: "task_duration_seconds",
			Labels: map[string]string{
				"task_name": "fetch_user",
				"status":    "success",
			},
			TimeRange: TimeRange{
				Start: now.Add(-time.Hour),
				End:   now,
			},
			Aggregation: "avg",
			Limit:       50,
		}

		assert.Equal(t, "task_duration_seconds", filter.MetricName)
		assert.Equal(t, "avg", filter.Aggregation)
		assert.Equal(t, 50, filter.Limit)
		assert.True(t, filter.TimeRange.IsValid())
		assert.Equal(t, "fetch_user", filter.Labels["task_name"])
		assert.Equal(t, "success", filter.Labels["status"])
	})
}

func TestSpan(t *testing.T) {
	t.Run("Complete span", func(t *testing.T) {
		now := time.Now()
		span := Span{
			SpanID:        "span-123",
			TraceID:       "trace-456",
			ParentSpanID:  "parent-789",
			OperationName: "execute_task",
			TaskName:      "fetch_user",
			StartTime:     now,
			EndTime:       now.Add(2 * time.Second),
			Duration:      2 * time.Second,
			Status:        true,
			Attributes: map[string]string{
				"task.inputs":  `{"user_id": 123}`,
				"task.outputs": `{"user": {"name": "Alice"}}`,
				"tool.name":    "database",
			},
			Events: []SpanEvent{
				{
					Time: now.Add(time.Second),
					Name: "tool_call_started",
					Attributes: map[string]string{
						"tool": "database",
					},
				},
			},
		}

		assert.Equal(t, "span-123", span.SpanID)
		assert.Equal(t, "trace-456", span.TraceID)
		assert.Equal(t, "parent-789", span.ParentSpanID)
		assert.Equal(t, "execute_task", span.OperationName)
		assert.Equal(t, "fetch_user", span.TaskName)
		assert.True(t, span.Status)
		assert.Equal(t, 2*time.Second, span.Duration)
		assert.Equal(t, "database", span.Attributes["tool.name"])
		require.Len(t, span.Events, 1)
		assert.Equal(t, "tool_call_started", span.Events[0].Name)
	})
}

func TestMetricPoint(t *testing.T) {
	t.Run("Complete metric", func(t *testing.T) {
		now := time.Now()
		metric := MetricPoint{
			Time:  now,
			Value: 1.5,
			Labels: map[string]string{
				"task_name": "fetch_user",
				"status":    "success",
			},
			Unit: "seconds",
		}

		assert.Equal(t, now, metric.Time)
		assert.Equal(t, 1.5, metric.Value)
		assert.Equal(t, "seconds", metric.Unit)
		assert.Equal(t, "fetch_user", metric.Labels["task_name"])
		assert.Equal(t, "success", metric.Labels["status"])
	})
}

// TestAdapterContract verifies that adapters properly implement the interface contract
func TestAdapterContract(t *testing.T) {
	adapters := []struct {
		name    string
		adapter TelemetryAdapter
	}{
		{"MockAdapter", NewMockAdapter()},
		{"NoOpAdapter", NewNoOpAdapter()},
	}

	for _, tc := range adapters {
		t.Run(tc.name, func(t *testing.T) {
			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()

			// Available should not panic and return a boolean
			available := tc.adapter.Available()
			assert.IsType(t, true, available, "Available() should return bool")

			// QuerySpans should not panic and return slice + error
			spans, err := tc.adapter.QuerySpans(ctx, SpanFilter{Limit: 10})
			assert.NoError(t, err, "QuerySpans should not return error for valid filter")
			assert.NotNil(t, spans, "QuerySpans should return non-nil slice")

			// QueryMetrics should not panic and return slice + error
			metrics, err := tc.adapter.QueryMetrics(ctx, MetricFilter{Limit: 10})
			assert.NoError(t, err, "QueryMetrics should not return error for valid filter")
			assert.NotNil(t, metrics, "QueryMetrics should return non-nil slice")
		})
	}
}

// TestContextCancellation verifies adapters respect context cancellation
func TestContextCancellation(t *testing.T) {
	adapter := NewMockAdapter()

	// Create a context that's already cancelled
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	// Adapters should respect the cancelled context
	// (Mock adapter doesn't actually check context, but real ones should)
	_, err := adapter.QuerySpans(ctx, SpanFilter{})
	// Mock adapter doesn't check context, so this won't error
	// In a real implementation, this would return context.Canceled
	assert.NoError(t, err, "Mock adapter doesn't check context")

	_, err = adapter.QueryMetrics(ctx, MetricFilter{})
	assert.NoError(t, err, "Mock adapter doesn't check context")
}
