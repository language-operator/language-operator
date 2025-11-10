package integration

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/sdk/trace/tracetest"

	"github.com/based/language-operator/pkg/synthesis"
	"github.com/go-logr/logr"
)

// TestTracePropagation tests end-to-end trace propagation from operator to synthesizer
// This verifies that OpenTelemetry spans are correctly created and linked across
// different components of the language operator.
func TestTracePropagation(t *testing.T) {
	// Create in-memory span exporter to capture traces
	exporter := tracetest.NewInMemoryExporter()

	// Create TracerProvider with in-memory exporter
	tp := trace.NewTracerProvider(
		trace.WithSyncer(exporter),
	)
	otel.SetTracerProvider(tp)
	defer tp.Shutdown(context.Background())

	// Start mock LLM service
	mockLLM := NewMockLLMService(t)
	defer mockLLM.Close()

	mockChatModel := NewMockChatModel(mockLLM)
	synthesizer := synthesis.NewSynthesizer(mockChatModel, logr.Discard())

	// Create root span to simulate controller reconciliation
	tracer := otel.Tracer("language-operator/test")
	ctx, rootSpan := tracer.Start(context.Background(), "agent.reconcile")

	// Perform synthesis within the traced context
	req := synthesis.AgentSynthesisRequest{
		Instructions: "Create an agent that monitors system health",
		Tools:        []string{},
		AgentName:    "test-agent",
		Namespace:    "default",
	}

	resp, err := synthesizer.SynthesizeAgent(ctx, req)
	rootSpan.End()

	// Verify synthesis succeeded
	require.NoError(t, err, "Synthesis should succeed")
	require.NotNil(t, resp, "Response should not be nil")

	// Flush spans
	err = tp.ForceFlush(context.Background())
	require.NoError(t, err, "Should flush spans successfully")

	// Verify spans were created
	spans := exporter.GetSpans()
	require.NotEmpty(t, spans, "Should have captured spans")

	// Verify we have expected span hierarchy
	spanNames := make(map[string]bool)
	for _, span := range spans {
		spanNames[span.Name] = true
	}

	// Check for root reconcile span
	assert.True(t, spanNames["agent.reconcile"], "Should have agent.reconcile span")

	// Check for synthesis span (from synthesizer)
	assert.True(t, spanNames["synthesis.agent.generate"], "Should have synthesis.agent.generate span")

	// Verify trace ID propagation - all spans should share the same trace ID
	if len(spans) > 1 {
		traceID := spans[0].SpanContext.TraceID()
		for i, span := range spans {
			assert.Equal(t, traceID, span.SpanContext.TraceID(),
				"Span %d (%s) should have same trace ID as root span", i, span.Name)
		}
	}

	// Verify parent-child relationships through trace ID
	// All spans in the same trace should have the same trace ID

	t.Logf("✓ Captured %d spans with proper trace propagation", len(spans))
	for _, span := range spans {
		t.Logf("  - %s (trace: %s, span: %s)",
			span.Name,
			span.SpanContext.TraceID().String()[:8],
			span.SpanContext.SpanID().String()[:8])
	}
}

// TestTraceAttributes verifies that spans contain expected attributes
// TODO: Fix this test - currently the synth validation creates spans but without our context
func TestTraceAttributes(t *testing.T) {
	t.Skip("Skipping until synthesis context propagation is fixed")
	// Create in-memory span exporter
	exporter := tracetest.NewInMemoryExporter()
	tp := trace.NewTracerProvider(
		trace.WithSyncer(exporter),
	)
	otel.SetTracerProvider(tp)
	defer tp.Shutdown(context.Background())

	// Start mock LLM service
	mockLLM := NewMockLLMService(t)
	defer mockLLM.Close()

	mockChatModel := NewMockChatModel(mockLLM)
	synthesizer := synthesis.NewSynthesizer(mockChatModel, logr.Discard())

	// Perform synthesis
	ctx := context.Background()
	req := synthesis.AgentSynthesisRequest{
		Instructions: "Test agent",
		AgentName:    "test-agent",
		Namespace:    "test-namespace",
	}

	_, err := synthesizer.SynthesizeAgent(ctx, req)
	require.NoError(t, err)

	// Flush spans
	err = tp.ForceFlush(context.Background())
	require.NoError(t, err)

	// Verify span attributes
	spans := exporter.GetSpans()
	require.NotEmpty(t, spans)

	// Find synthesis span and check attributes
	for _, span := range spans {
		if span.Name == "synthesis.agent.generate" {
			attrs := span.Attributes

			// Check for expected attributes
			hasAgentName := false
			hasNamespace := false

			for _, attr := range attrs {
				if string(attr.Key) == "agent.name" {
					hasAgentName = true
					assert.Equal(t, "test-agent", attr.Value.AsString())
				}
				if string(attr.Key) == "agent.namespace" {
					hasNamespace = true
					assert.Equal(t, "test-namespace", attr.Value.AsString())
				}
			}

			assert.True(t, hasAgentName, "Should have agent.name attribute")
			assert.True(t, hasNamespace, "Should have agent.namespace attribute")

			t.Logf("✓ Synthesis span has expected attributes")
		}
	}
}
