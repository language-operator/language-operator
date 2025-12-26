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

package handlers

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/gorilla/mux"
	"github.com/language-operator/language-operator/pkg/telemetry"
)

// DashboardHandler provides HTTP endpoints for the dashboard to query telemetry data.
//
// This handler bridges the Next.js dashboard with the ClickHouse telemetry adapter,
// converting HTTP requests to telemetry queries and formatting responses appropriately.
//
// All endpoints return JSON responses with the following structure:
//   - success: boolean indicating if the request was successful
//   - data: the actual response data (if successful)
//   - error: error message (if unsuccessful)
type DashboardHandler struct {
	adapter telemetry.TelemetryAdapter
}

// NewDashboardHandler creates a new dashboard handler with the given telemetry adapter.
func NewDashboardHandler(adapter telemetry.TelemetryAdapter) *DashboardHandler {
	return &DashboardHandler{
		adapter: adapter,
	}
}

// AgentExecution represents an agent execution for the dashboard API.
type AgentExecution struct {
	TraceID      string    `json:"traceId"`
	ExecutionID  string    `json:"executionId"`
	StartTime    time.Time `json:"startTime"`
	EndTime      time.Time `json:"endTime"`
	Duration     int64     `json:"duration"` // milliseconds
	Status       string    `json:"status"`
	RootSpanName string    `json:"rootSpanName"`
	SpanCount    int       `json:"spanCount"`
}

// TraceSpan represents a trace span for the dashboard API.
type TraceSpan struct {
	SpanID       string                 `json:"spanId"`
	ParentSpanID *string                `json:"parentSpanId,omitempty"`
	SpanName     string                 `json:"spanName"`
	StartTime    time.Time              `json:"startTime"`
	EndTime      time.Time              `json:"endTime"`
	Duration     int64                  `json:"duration"` // milliseconds
	Status       string                 `json:"status"`
	Attributes   map[string]interface{} `json:"attributes"`
	Events       []TraceSpanEvent       `json:"events"`
}

// TraceSpanEvent represents a span event for the dashboard API.
type TraceSpanEvent struct {
	Time       time.Time              `json:"time"`
	Name       string                 `json:"name"`
	Attributes map[string]interface{} `json:"attributes"`
}

// TraceData represents the complete trace data for a specific execution.
type TraceData struct {
	TraceID     string      `json:"traceId"`
	ExecutionID string      `json:"executionId"`
	Spans       []TraceSpan `json:"spans"`
}

// RegisterRoutes registers all dashboard telemetry routes with the given router.
func (h *DashboardHandler) RegisterRoutes(router *mux.Router) {
	// Agent execution endpoints
	router.HandleFunc("/api/clusters/{clusterName}/agents/{agentName}/executions", h.GetAgentExecutions).Methods("GET")
	router.HandleFunc("/api/clusters/{clusterName}/agents/{agentName}/executions/{executionId}/traces", h.GetExecutionTraces).Methods("GET")
}

// GetAgentExecutions returns a list of recent executions for a specific agent.
//
// Query parameters:
//   - limit: maximum number of executions to return (default: 50, max: 1000)
//   - timeRange: time range in milliseconds from now (default: 24h)
//
// Response format:
//
//	{
//	  "success": true,
//	  "data": [
//	    {
//	      "traceId": "abc123...",
//	      "executionId": "exec_001",
//	      "startTime": "2024-01-15T10:30:00Z",
//	      "endTime": "2024-01-15T10:30:45Z",
//	      "duration": 45000,
//	      "status": "success",
//	      "rootSpanName": "agent_execution",
//	      "spanCount": 5
//	    }
//	  ]
//	}
func (h *DashboardHandler) GetAgentExecutions(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), 30*time.Second)
	defer cancel()

	// Extract URL parameters
	vars := mux.Vars(r)
	clusterName := vars["clusterName"]
	agentName := vars["agentName"]

	// Parse query parameters
	limit := parseQueryInt(r, "limit", 50, 1000)
	timeRangeMs := parseQueryInt(r, "timeRange", 24*60*60*1000, 0) // 24 hours default

	// Check if adapter is available - for NoOp adapter, return empty results instead of error
	if !h.adapter.Available() {
		// Return empty executions list for unavailable adapters (like NoOp)
		writeSuccessResponse(w, []AgentExecution{})
		return
	}

	// Build telemetry query filter
	filter := telemetry.SpanFilter{
		Attributes: map[string]string{
			"agent.name":   agentName,
			"cluster.name": clusterName,
		},
		TimeRange: telemetry.TimeRange{
			Start: time.Now().Add(-time.Duration(timeRangeMs) * time.Millisecond),
			End:   time.Now(),
		},
		Limit: limit * 10, // Query more spans to group into executions
	}

	// Query spans from ClickHouse
	spans, err := h.adapter.QuerySpans(ctx, filter)
	if err != nil {
		writeErrorResponse(w, http.StatusInternalServerError, fmt.Sprintf("Failed to query spans: %v", err))
		return
	}

	// Group spans into executions (by trace ID)
	executions := h.groupSpansIntoExecutions(spans, limit)

	// Write successful response
	writeSuccessResponse(w, executions)
}

// GetExecutionTraces returns detailed trace data for a specific execution.
//
// Path parameters:
//   - clusterName: name of the cluster
//   - agentName: name of the agent
//   - executionId: ID of the execution (derived from trace ID)
//
// Response format:
//
//	{
//	  "success": true,
//	  "data": {
//	    "traceId": "abc123...",
//	    "executionId": "exec_001",
//	    "spans": [
//	      {
//	        "spanId": "span123",
//	        "parentSpanId": null,
//	        "spanName": "agent_execution",
//	        "startTime": "2024-01-15T10:30:00Z",
//	        "endTime": "2024-01-15T10:30:45Z",
//	        "duration": 45000,
//	        "status": "success",
//	        "attributes": {...},
//	        "events": [...]
//	      }
//	    ]
//	  }
//	}
func (h *DashboardHandler) GetExecutionTraces(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), 30*time.Second)
	defer cancel()

	// Extract URL parameters
	vars := mux.Vars(r)
	executionId := vars["executionId"]

	// Check if adapter is available - for NoOp adapter, return empty trace data
	if !h.adapter.Available() {
		// Return empty trace data for unavailable adapters (like NoOp)
		emptyTrace := TraceData{
			TraceID:     h.executionIdToTraceId(executionId),
			ExecutionID: executionId,
			Spans:       []TraceSpan{},
		}
		writeSuccessResponse(w, emptyTrace)
		return
	}

	// Convert execution ID back to trace ID
	// For now, assume they're the same or derive from a known pattern
	traceId := h.executionIdToTraceId(executionId)

	// Build telemetry query filter for specific trace
	filter := telemetry.SpanFilter{
		TraceID: traceId,
		Limit:   1000, // Get all spans for this trace
	}

	// Query spans from ClickHouse
	spans, err := h.adapter.QuerySpans(ctx, filter)
	if err != nil {
		writeErrorResponse(w, http.StatusInternalServerError, fmt.Sprintf("Failed to query trace spans: %v", err))
		return
	}

	// Convert to dashboard format
	traceData := h.convertSpansToTraceData(traceId, executionId, spans)

	// Write successful response
	writeSuccessResponse(w, traceData)
}

// groupSpansIntoExecutions groups spans by trace ID to create execution summaries.
func (h *DashboardHandler) groupSpansIntoExecutions(spans []telemetry.Span, limit int) []AgentExecution {
	executionMap := make(map[string]*AgentExecution)

	for _, span := range spans {
		execution, exists := executionMap[span.TraceID]
		if !exists {
			execution = &AgentExecution{
				TraceID:      span.TraceID,
				ExecutionID:  h.traceIdToExecutionId(span.TraceID),
				StartTime:    span.StartTime,
				EndTime:      span.EndTime,
				Duration:     span.Duration.Milliseconds(),
				Status:       h.spanStatusToString(span.Status),
				RootSpanName: span.OperationName,
				SpanCount:    0,
			}
			executionMap[span.TraceID] = execution
		}

		// Update execution bounds and status
		if span.StartTime.Before(execution.StartTime) {
			execution.StartTime = span.StartTime
		}
		if span.EndTime.After(execution.EndTime) {
			execution.EndTime = span.EndTime
		}
		execution.Duration = execution.EndTime.Sub(execution.StartTime).Milliseconds()

		// Update status - if any span failed, mark execution as failed
		if !span.Status && execution.Status != "error" {
			execution.Status = "error"
		}

		// Update root span name if this is a root span
		if span.ParentSpanID == "" {
			execution.RootSpanName = span.OperationName
		}

		execution.SpanCount++
	}

	// Convert map to slice and sort by start time (newest first)
	executions := make([]AgentExecution, 0, len(executionMap))
	for _, execution := range executionMap {
		executions = append(executions, *execution)
	}

	// Sort by start time (newest first)
	for i := 0; i < len(executions); i++ {
		for j := i + 1; j < len(executions); j++ {
			if executions[j].StartTime.After(executions[i].StartTime) {
				executions[i], executions[j] = executions[j], executions[i]
			}
		}
	}

	// Limit results
	if len(executions) > limit {
		executions = executions[:limit]
	}

	return executions
}

// convertSpansToTraceData converts telemetry spans to dashboard trace data format.
func (h *DashboardHandler) convertSpansToTraceData(traceId, executionId string, spans []telemetry.Span) TraceData {
	dashboardSpans := make([]TraceSpan, len(spans))

	for i, span := range spans {
		// Convert attributes to interface{} map
		attributes := make(map[string]interface{})
		for k, v := range span.Attributes {
			attributes[k] = v
		}

		// Convert events
		events := make([]TraceSpanEvent, len(span.Events))
		for j, event := range span.Events {
			eventAttributes := make(map[string]interface{})
			for k, v := range event.Attributes {
				eventAttributes[k] = v
			}
			events[j] = TraceSpanEvent{
				Time:       event.Time,
				Name:       event.Name,
				Attributes: eventAttributes,
			}
		}

		// Handle parent span ID (convert empty string to nil pointer)
		var parentSpanId *string
		if span.ParentSpanID != "" {
			parentSpanId = &span.ParentSpanID
		}

		dashboardSpans[i] = TraceSpan{
			SpanID:       span.SpanID,
			ParentSpanID: parentSpanId,
			SpanName:     span.OperationName,
			StartTime:    span.StartTime,
			EndTime:      span.EndTime,
			Duration:     span.Duration.Milliseconds(),
			Status:       h.spanStatusToString(span.Status),
			Attributes:   attributes,
			Events:       events,
		}
	}

	return TraceData{
		TraceID:     traceId,
		ExecutionID: executionId,
		Spans:       dashboardSpans,
	}
}

// Utility functions
func (h *DashboardHandler) traceIdToExecutionId(traceId string) string {
	// For now, use a simple truncated version
	// In production, might use a different mapping strategy
	if len(traceId) > 8 {
		return "exec_" + traceId[:8]
	}
	return "exec_" + traceId
}

func (h *DashboardHandler) executionIdToTraceId(executionId string) string {
	// For now, this is a simple reverse mapping
	// In production, you might need a proper lookup table
	if len(executionId) > 5 && executionId[:5] == "exec_" {
		return executionId[5:]
	}
	return executionId
}

func (h *DashboardHandler) spanStatusToString(status bool) string {
	if status {
		return "success"
	}
	return "error"
}

// HTTP utility functions
func parseQueryInt(r *http.Request, param string, defaultValue, maxValue int) int {
	value := r.URL.Query().Get(param)
	if value == "" {
		return defaultValue
	}

	parsed, err := strconv.Atoi(value)
	if err != nil {
		return defaultValue
	}

	if maxValue > 0 && parsed > maxValue {
		return maxValue
	}

	return parsed
}

func writeSuccessResponse(w http.ResponseWriter, data interface{}) {
	response := map[string]interface{}{
		"success": true,
		"data":    data,
	}
	writeJSONResponse(w, http.StatusOK, response)
}

func writeErrorResponse(w http.ResponseWriter, statusCode int, message string) {
	response := map[string]interface{}{
		"success": false,
		"error":   message,
	}
	writeJSONResponse(w, statusCode, response)
}

func writeJSONResponse(w http.ResponseWriter, statusCode int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	json.NewEncoder(w).Encode(data)
}
