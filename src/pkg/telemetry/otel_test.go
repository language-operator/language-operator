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

	sdktrace "go.opentelemetry.io/otel/sdk/trace"
)

func TestInitTracer_NoEndpoint(t *testing.T) {
	t.Setenv("OTEL_EXPORTER_OTLP_ENDPOINT", "")
	tp, err := InitTracer(context.Background())
	if err != nil {
		t.Errorf("expected nil error, got %v", err)
	}
	if tp != nil {
		t.Errorf("expected nil TracerProvider when endpoint is unset, got %v", tp)
	}
}

func TestShutdown_NilProvider(t *testing.T) {
	if err := Shutdown(context.Background(), nil); err != nil {
		t.Errorf("expected nil error for nil provider, got %v", err)
	}
}

func TestParseResourceAttributes(t *testing.T) {
	tests := []struct {
		input   string
		wantLen int
	}{
		{"", 0},
		{"key=value", 1},
		{"k1=v1,k2=v2", 2},
		{"bad", 0},
		{"=nokey", 0},
		{"novalue=", 0},
	}
	for _, tt := range tests {
		got := parseResourceAttributes(tt.input)
		if len(got) != tt.wantLen {
			t.Errorf("parseResourceAttributes(%q): got %d attrs, want %d", tt.input, len(got), tt.wantLen)
		}
	}
}

func TestConfigureSampler(t *testing.T) {
	tests := []struct {
		name        string
		samplerType string
		samplerArg  string
		wantDesc    string
	}{
		{
			name:        "always_on",
			samplerType: "always_on",
			wantDesc:    sdktrace.AlwaysSample().Description(),
		},
		{
			name:        "always_off",
			samplerType: "always_off",
			wantDesc:    sdktrace.NeverSample().Description(),
		},
		{
			name:        "traceidratio with valid arg",
			samplerType: "traceidratio",
			samplerArg:  "0.5",
			wantDesc:    sdktrace.TraceIDRatioBased(0.5).Description(),
		},
		{
			name:        "traceidratio clamps below zero",
			samplerType: "traceidratio",
			samplerArg:  "-0.1",
			wantDesc:    sdktrace.TraceIDRatioBased(0.0).Description(),
		},
		{
			name:        "traceidratio clamps above one",
			samplerType: "traceidratio",
			samplerArg:  "1.5",
			wantDesc:    sdktrace.TraceIDRatioBased(1.0).Description(),
		},
		{
			name:        "traceidratio with invalid arg defaults to 1.0",
			samplerType: "traceidratio",
			samplerArg:  "notanumber",
			wantDesc:    sdktrace.TraceIDRatioBased(1.0).Description(),
		},
		{
			name:        "traceidratio with empty arg defaults to 1.0",
			samplerType: "traceidratio",
			wantDesc:    sdktrace.TraceIDRatioBased(1.0).Description(),
		},
		{
			name:        "unknown sampler defaults to always_on",
			samplerType: "unknown_sampler",
			wantDesc:    sdktrace.AlwaysSample().Description(),
		},
		{
			name:     "unset sampler defaults to always_on",
			wantDesc: sdktrace.AlwaysSample().Description(),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Setenv("OTEL_TRACES_SAMPLER", tt.samplerType)
			t.Setenv("OTEL_TRACES_SAMPLER_ARG", tt.samplerArg)
			got := configureSampler()
			if got.Description() != tt.wantDesc {
				t.Errorf("configureSampler() = %q, want %q", got.Description(), tt.wantDesc)
			}
		})
	}
}

func TestInitTracer_WithEndpoint(t *testing.T) {
	// gRPC dial is lazy by default; pointing at a non-existent host is fine for unit tests.
	t.Setenv("OTEL_EXPORTER_OTLP_ENDPOINT", "localhost:14317")
	tp, err := InitTracer(context.Background())
	if err != nil {
		t.Fatalf("InitTracer with endpoint: unexpected error: %v", err)
	}
	if tp == nil {
		t.Fatal("InitTracer with endpoint: expected non-nil TracerProvider")
	}
	// Clean up: shut down the provider so it doesn't leak goroutines.
	_ = Shutdown(context.Background(), tp)
}

func TestShutdown_WithSDKProvider(t *testing.T) {
	tp := sdktrace.NewTracerProvider()
	if err := Shutdown(context.Background(), tp); err != nil {
		t.Errorf("Shutdown(sdkProvider): unexpected error: %v", err)
	}
}
