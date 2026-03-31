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
