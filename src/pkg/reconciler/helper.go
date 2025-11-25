package reconciler

import (
	"context"
	"fmt"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

// ReconcileHelper provides common reconciliation functionality for all controllers
type ReconcileHelper[T client.Object] struct {
	Client       client.Client
	TracerName   string
	ResourceType string
}

// ReconcileResult contains the results of the reconciliation start
type ReconcileResult[T client.Object] struct {
	Ctx      context.Context
	Span     trace.Span
	Resource T
	Log      interface{} // logr.Logger
}

// StartReconcile initializes reconciliation with standard tracing and resource fetching
func (h *ReconcileHelper[T]) StartReconcile(ctx context.Context, req ctrl.Request, resource T) (*ReconcileResult[T], error) {
	// Start OpenTelemetry span for reconciliation
	tracer := otel.Tracer(h.TracerName)
	ctx, span := tracer.Start(ctx, fmt.Sprintf("%s.reconcile", h.ResourceType))

	// Add basic span attributes from request
	span.SetAttributes(
		attribute.String(fmt.Sprintf("%s.name", h.ResourceType), req.Name),
		attribute.String(fmt.Sprintf("%s.namespace", h.ResourceType), req.Namespace),
	)

	logger := log.FromContext(ctx)

	// Fetch the resource instance
	if err := h.Client.Get(ctx, req.NamespacedName, resource); err != nil {
		if errors.IsNotFound(err) {
			// Resource not found, likely deleted - this is not an error
			span.SetStatus(codes.Ok, "Resource not found (deleted)")
			return nil, client.IgnoreNotFound(err)
		}
		logger.Error(err, fmt.Sprintf("Failed to get %s", h.ResourceType))
		span.RecordError(err)
		span.SetStatus(codes.Error, fmt.Sprintf("Failed to get %s", h.ResourceType))
		return nil, err
	}

	// Add resource-specific attributes to span
	if obj, ok := any(resource).(metav1.Object); ok {
		span.SetAttributes(
			attribute.Int64(fmt.Sprintf("%s.generation", h.ResourceType), obj.GetGeneration()),
		)
	}

	// Add any additional type-specific attributes
	h.addTypeSpecificAttributes(span, resource)

	return &ReconcileResult[T]{
		Ctx:      ctx,
		Span:     span,
		Resource: resource,
		Log:      logger,
	}, nil
}

// addTypeSpecificAttributes allows adding custom attributes based on resource type
func (h *ReconcileHelper[T]) addTypeSpecificAttributes(span trace.Span, resource T) {
	// This is a hook for type-specific attributes
	// Controllers can extend this by embedding ReconcileHelper
	// and overriding this method if needed
}

// CompleteReconcile finishes the reconciliation with proper span closure
func (r *ReconcileResult[T]) CompleteReconcile(err error) {
	if r.Span != nil {
		if err != nil {
			r.Span.RecordError(err)
			r.Span.SetStatus(codes.Error, err.Error())
		} else {
			r.Span.SetStatus(codes.Ok, "Reconciliation completed successfully")
		}
		r.Span.End()
	}
}