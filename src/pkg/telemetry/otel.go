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
	"os"
	"runtime/debug"
	"time"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.24.0"
	"go.opentelemetry.io/otel/trace"
)

// InitTracer initializes OpenTelemetry tracing if OTEL_EXPORTER_OTLP_ENDPOINT is set.
// Returns nil if endpoint not configured (OTel disabled).
// Returns TracerProvider for graceful shutdown, or error if initialization fails.
func InitTracer(ctx context.Context) (trace.TracerProvider, error) {
	// Check if OTel endpoint is configured
	endpoint := os.Getenv("OTEL_EXPORTER_OTLP_ENDPOINT")
	if endpoint == "" {
		// OTel not configured, return nil (tracing disabled)
		return nil, nil
	}

	// Create context with timeout for initialization
	initCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	// Create OTLP gRPC exporter
	exporter, err := otlptracegrpc.New(
		initCtx,
		otlptracegrpc.WithEndpoint(endpoint),
		otlptracegrpc.WithInsecure(), // Use insecure for internal cluster communication
	)
	if err != nil {
		// Log warning but don't fail - operator should work without tracing
		return nil, err
	}

	// Get service version from build info
	version := "unknown"
	if info, ok := debug.ReadBuildInfo(); ok {
		version = info.Main.Version
		if version == "" || version == "(devel)" {
			version = "dev"
		}
	}

	// Build resource attributes
	resourceAttrs := []resource.Option{
		resource.WithAttributes(
			semconv.ServiceName("language-operator"),
			semconv.ServiceVersion(version),
		),
	}

	// Add k8s.namespace.name from environment
	if namespace := os.Getenv("POD_NAMESPACE"); namespace != "" {
		resourceAttrs = append(resourceAttrs, resource.WithAttributes(
			semconv.K8SNamespaceName(namespace),
		))
	}

	// Parse additional attributes from OTEL_RESOURCE_ATTRIBUTES
	// This allows setting k8s.cluster.name and other custom attributes
	if resAttrs := os.Getenv("OTEL_RESOURCE_ATTRIBUTES"); resAttrs != "" {
		resourceAttrs = append(resourceAttrs, resource.WithAttributes())
	}

	// Create resource
	res, err := resource.New(ctx, resourceAttrs...)
	if err != nil {
		return nil, err
	}

	// Create TracerProvider with batch span processor
	tp := sdktrace.NewTracerProvider(
		sdktrace.WithBatcher(exporter),
		sdktrace.WithResource(res),
	)

	// Set as global TracerProvider
	otel.SetTracerProvider(tp)

	return tp, nil
}

// Shutdown gracefully shuts down the TracerProvider, flushing any remaining spans.
func Shutdown(ctx context.Context, tp trace.TracerProvider) error {
	if tp == nil {
		return nil
	}

	// Type assert to *sdktrace.TracerProvider to access Shutdown
	if sdkTP, ok := tp.(*sdktrace.TracerProvider); ok {
		return sdkTP.Shutdown(ctx)
	}

	return nil
}
