package controllers

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// toolsListResponseJSON is a tools/list result the mock server returns to every request
// (including initialize), which is enough for discoverMCPToolSchemas' handshake to resolve.
const toolsListResponseJSON = `{
	"jsonrpc": "2.0",
	"id": 1,
	"result": {
		"tools": [
			{
				"name": "read_file",
				"description": "Read a file from the workspace",
				"inputSchema": {
					"type": "object",
					"properties": {
						"path": {
							"type": "string",
							"description": "Path to the file to read",
							"examples": ["/workspace/data.txt"]
						}
					},
					"required": ["path"]
				}
			},
			{
				"name": "write_file",
				"description": "Write content to a file",
				"inputSchema": {
					"type": "object",
					"properties": {
						"path": {"type": "string", "description": "Path to the file"},
						"content": {"type": "string", "description": "File content"}
					},
					"required": ["path", "content"]
				}
			}
		]
	}
}`

func TestDiscoverMCPToolSchemas_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/mcp" {
			t.Errorf("Expected request to /mcp, got %s", r.URL.Path)
		}
		if r.Method != http.MethodPost {
			t.Errorf("Expected POST request, got %s", r.Method)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(toolsListResponseJSON))
	}))
	defer server.Close()

	r := &LanguageToolReconciler{}
	endpoint := server.URL[len("http://"):]

	schemas, err := r.discoverMCPToolSchemas(context.Background(), endpoint)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}
	if len(schemas) != 2 {
		t.Fatalf("Expected 2 schemas, got %d", len(schemas))
	}

	readSchema := schemas[0]
	if readSchema.Name != "read_file" {
		t.Errorf("Expected name 'read_file', got '%s'", readSchema.Name)
	}
	if readSchema.Description != "Read a file from the workspace" {
		t.Errorf("Unexpected description %q", readSchema.Description)
	}
	if readSchema.InputSchema == nil || readSchema.InputSchema.Type != "object" {
		t.Fatalf("Expected object InputSchema, got %+v", readSchema.InputSchema)
	}
	if len(readSchema.InputSchema.Required) != 1 || readSchema.InputSchema.Required[0] != "path" {
		t.Errorf("Expected required ['path'], got %v", readSchema.InputSchema.Required)
	}
	pathProp, ok := readSchema.InputSchema.Properties["path"]
	if !ok {
		t.Fatal("Expected 'path' property")
	}
	if pathProp.Type != "string" || pathProp.Description != "Path to the file to read" {
		t.Errorf("Unexpected path property %+v", pathProp)
	}
	// The first example is preserved as a JSON-encoded string.
	if pathProp.Example != "\"/workspace/data.txt\"" {
		t.Errorf("Expected example '\"/workspace/data.txt\"', got '%s'", pathProp.Example)
	}

	if len(schemas[1].InputSchema.Required) != 2 {
		t.Errorf("Expected 2 required fields on write_file, got %d", len(schemas[1].InputSchema.Required))
	}
}

func TestDiscoverMCPToolSchemas_ServerError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer server.Close()

	r := &LanguageToolReconciler{}
	_, err := r.discoverMCPToolSchemas(context.Background(), server.URL[len("http://"):])
	if err == nil {
		t.Fatal("Expected error for server error response")
	}
}

func TestDiscoverMCPToolSchemas_MCPError(t *testing.T) {
	// Server answers every request (initialize and the legacy fallback tools/list) with a
	// JSON-RPC error. Discovery should surface it after falling back.
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"jsonrpc":"2.0","id":1,"error":{"code":-32601,"message":"Method not found"}}`))
	}))
	defer server.Close()

	r := &LanguageToolReconciler{}
	_, err := r.discoverMCPToolSchemas(context.Background(), server.URL[len("http://"):])
	if err == nil {
		t.Fatal("Expected error for MCP error response")
	}
	if !strings.Contains(err.Error(), "Method not found") || !strings.Contains(err.Error(), "-32601") {
		t.Errorf("Unexpected error: %v", err)
	}
}

func TestDiscoverMCPToolSchemas_ContextCancelled(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		select {}
	}))
	defer server.Close()

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // pre-cancel so the request is rejected immediately

	r := &LanguageToolReconciler{HTTPClient: server.Client()}
	_, err := r.discoverMCPToolSchemas(ctx, server.URL[len("http://"):])
	if err == nil {
		t.Fatal("Expected error for cancelled context, got nil")
	}
}
