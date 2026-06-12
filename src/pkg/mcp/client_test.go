package mcp

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"
)

// rpcEnvelope is a minimal decoder for the incoming request method/id.
type rpcEnvelope struct {
	ID     int    `json:"id"`
	Method string `json:"method"`
}

const toolsListJSON = `{"tools":[{"name":"get-library-docs","description":"Fetch docs","inputSchema":{"type":"object","properties":{"id":{"type":"string","description":"library id"}},"required":["id"]}}]}`

func sseFrame(payload string) string {
	return "event: message\ndata: " + payload + "\n\n"
}

func decodeMethod(t *testing.T, r *http.Request) rpcEnvelope {
	t.Helper()
	body, err := io.ReadAll(r.Body)
	if err != nil {
		t.Fatalf("read body: %v", err)
	}
	var env rpcEnvelope
	if err := json.Unmarshal(body, &env); err != nil {
		t.Fatalf("decode body %q: %v", string(body), err)
	}
	return env
}

func rpcResult(id int, result string) string {
	return fmt.Sprintf(`{"jsonrpc":"2.0","id":%d,"result":%s}`, id, result)
}

// fullHandshakeServer implements initialize → notifications/initialized → tools/list,
// optionally framing the tools/list response as SSE, and asserting session threading.
func fullHandshakeServer(t *testing.T, useSSE bool, sessionID string) *httptest.Server {
	t.Helper()
	var initialized bool
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/mcp" {
			t.Errorf("unexpected path %q", r.URL.Path)
		}
		if got := r.Header.Get("Accept"); !strings.Contains(got, "text/event-stream") {
			t.Errorf("missing SSE Accept header, got %q", got)
		}
		env := decodeMethod(t, r)
		switch env.Method {
		case "initialize":
			w.Header().Set("Mcp-Session-Id", sessionID)
			w.Header().Set("Content-Type", "application/json")
			_, _ = io.WriteString(w, rpcResult(env.ID, `{"protocolVersion":"2024-11-05","capabilities":{}}`))
		case "notifications/initialized":
			if r.Header.Get("Mcp-Session-Id") != sessionID {
				t.Errorf("notifications/initialized missing session id; got %q", r.Header.Get("Mcp-Session-Id"))
			}
			initialized = true
			w.WriteHeader(http.StatusAccepted)
		case "tools/list":
			if !initialized {
				t.Errorf("tools/list called before notifications/initialized")
			}
			if r.Header.Get("Mcp-Session-Id") != sessionID {
				t.Errorf("tools/list missing session id; got %q", r.Header.Get("Mcp-Session-Id"))
			}
			payload := rpcResult(env.ID, toolsListJSON)
			if useSSE {
				w.Header().Set("Content-Type", "text/event-stream")
				_, _ = io.WriteString(w, sseFrame(payload))
			} else {
				w.Header().Set("Content-Type", "application/json")
				_, _ = io.WriteString(w, payload)
			}
		default:
			t.Errorf("unexpected method %q", env.Method)
		}
	}))
}

func assertContext7Tools(t *testing.T, tools []Tool) {
	t.Helper()
	if len(tools) != 1 {
		t.Fatalf("want 1 tool, got %d", len(tools))
	}
	if tools[0].Name != "get-library-docs" {
		t.Errorf("tool name = %q", tools[0].Name)
	}
	if tools[0].InputSchema == nil || tools[0].InputSchema.Properties["id"].Type != "string" {
		t.Errorf("input schema not parsed: %+v", tools[0].InputSchema)
	}
	if len(tools[0].InputSchema.Required) != 1 || tools[0].InputSchema.Required[0] != "id" {
		t.Errorf("required not parsed: %+v", tools[0].InputSchema.Required)
	}
}

func TestListTools_JSONHandshake(t *testing.T) {
	srv := fullHandshakeServer(t, false, "sess-123")
	defer srv.Close()

	tools, err := (&Client{HTTP: srv.Client()}).ListTools(context.Background(), srv.URL)
	if err != nil {
		t.Fatalf("ListTools: %v", err)
	}
	assertContext7Tools(t, tools)
}

func TestListTools_SSEHandshake(t *testing.T) {
	srv := fullHandshakeServer(t, true, "sess-sse")
	defer srv.Close()

	tools, err := (&Client{HTTP: srv.Client()}).ListTools(context.Background(), srv.URL)
	if err != nil {
		t.Fatalf("ListTools: %v", err)
	}
	assertContext7Tools(t, tools)
}

func TestListTools_FallbackOnMethodNotFound(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		env := decodeMethod(t, r)
		w.Header().Set("Content-Type", "application/json")
		if env.Method == "initialize" {
			_, _ = io.WriteString(w, fmt.Sprintf(`{"jsonrpc":"2.0","id":%d,"error":{"code":-32601,"message":"method not found"}}`, env.ID))
			return
		}
		// bare tools/list (the legacy fallback) succeeds
		_, _ = io.WriteString(w, rpcResult(env.ID, toolsListJSON))
	}))
	defer srv.Close()

	tools, err := (&Client{HTTP: srv.Client()}).ListTools(context.Background(), srv.URL)
	if err != nil {
		t.Fatalf("ListTools fallback: %v", err)
	}
	assertContext7Tools(t, tools)
}

func TestListTools_FallbackOn4xx(t *testing.T) {
	var calls int
	var mu sync.Mutex
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		env := decodeMethod(t, r)
		mu.Lock()
		calls++
		mu.Unlock()
		if env.Method == "initialize" {
			http.Error(w, "bad request", http.StatusBadRequest)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = io.WriteString(w, rpcResult(env.ID, toolsListJSON))
	}))
	defer srv.Close()

	tools, err := (&Client{HTTP: srv.Client()}).ListTools(context.Background(), srv.URL)
	if err != nil {
		t.Fatalf("ListTools 4xx fallback: %v", err)
	}
	assertContext7Tools(t, tools)
}

func TestListTools_ServerError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		env := decodeMethod(t, r)
		w.Header().Set("Content-Type", "application/json")
		switch env.Method {
		case "initialize":
			_, _ = io.WriteString(w, rpcResult(env.ID, `{}`))
		case "tools/list":
			_, _ = io.WriteString(w, fmt.Sprintf(`{"jsonrpc":"2.0","id":%d,"error":{"code":-32603,"message":"boom"}}`, env.ID))
		default:
			w.WriteHeader(http.StatusAccepted)
		}
	}))
	defer srv.Close()

	_, err := (&Client{HTTP: srv.Client()}).ListTools(context.Background(), srv.URL)
	if err == nil || !strings.Contains(err.Error(), "boom") {
		t.Fatalf("want MCP server error, got %v", err)
	}
}

func TestListTools_ContextCancellation(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(2 * time.Second)
		w.Header().Set("Content-Type", "application/json")
		_, _ = io.WriteString(w, rpcResult(1, `{}`))
	}))
	defer srv.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()
	_, err := (&Client{HTTP: srv.Client()}).ListTools(ctx, srv.URL)
	if err == nil {
		t.Fatal("want context deadline error, got nil")
	}
}

func TestParseSSEResponse_MultiLineAndComments(t *testing.T) {
	body := ": this is a comment\nevent: message\nid: 1\ndata: " + rpcResult(2, toolsListJSON) + "\n\n"
	resp, err := parseSSEResponse([]byte(body))
	if err != nil {
		t.Fatalf("parseSSEResponse: %v", err)
	}
	tools, err := toolsFromResponse(resp)
	if err != nil {
		t.Fatalf("toolsFromResponse: %v", err)
	}
	assertContext7Tools(t, tools)
}
