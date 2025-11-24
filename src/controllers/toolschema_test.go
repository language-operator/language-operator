package controllers

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestDiscoverMCPToolSchemas_Success(t *testing.T) {
	// Create mock MCP server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/mcp" {
			t.Errorf("Expected request to /mcp, got %s", r.URL.Path)
		}
		
		if r.Method != "POST" {
			t.Errorf("Expected POST request, got %s", r.Method)
		}
		
		// Mock MCP response
		response := MCPResponse{
			JSONRpc: "2.0",
			ID:      1,
			Result: json.RawMessage(`{
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
								"path": {
									"type": "string",
									"description": "Path to the file"
								},
								"content": {
									"type": "string",
									"description": "File content"
								}
							},
							"required": ["path", "content"]
						}
					}
				]
			}`),
		}
		
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
	}))
	defer server.Close()
	
	// Create reconciler
	r := &LanguageToolReconciler{}
	
	// Extract host:port from test server URL
	endpoint := server.URL[7:] // Remove "http://" prefix
	
	// Test discovery
	schemas, err := r.discoverMCPToolSchemas(context.Background(), endpoint)
	
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}
	
	if len(schemas) != 2 {
		t.Fatalf("Expected 2 schemas, got %d", len(schemas))
	}
	
	// Verify first tool schema
	readSchema := schemas[0]
	if readSchema.Name != "read_file" {
		t.Errorf("Expected name 'read_file', got '%s'", readSchema.Name)
	}
	
	if readSchema.Description != "Read a file from the workspace" {
		t.Errorf("Expected description 'Read a file from the workspace', got '%s'", readSchema.Description)
	}
	
	if readSchema.InputSchema == nil {
		t.Fatal("Expected InputSchema to be set")
	}
	
	if readSchema.InputSchema.Type != "object" {
		t.Errorf("Expected type 'object', got '%s'", readSchema.InputSchema.Type)
	}
	
	if len(readSchema.InputSchema.Required) != 1 || readSchema.InputSchema.Required[0] != "path" {
		t.Errorf("Expected required field ['path'], got %v", readSchema.InputSchema.Required)
	}
	
	pathProp, exists := readSchema.InputSchema.Properties["path"]
	if !exists {
		t.Fatal("Expected 'path' property to exist")
	}
	
	if pathProp.Type != "string" {
		t.Errorf("Expected path type 'string', got '%s'", pathProp.Type)
	}
	
	if pathProp.Description != "Path to the file to read" {
		t.Errorf("Expected path description 'Path to the file to read', got '%s'", pathProp.Description)
	}
	
	// The example should be JSON-encoded
	if pathProp.Example != "\"/workspace/data.txt\"" {
		t.Errorf("Expected example '\"/workspace/data.txt\"', got '%s'", pathProp.Example)
	}
	
	// Verify second tool schema  
	writeSchema := schemas[1]
	if writeSchema.Name != "write_file" {
		t.Errorf("Expected name 'write_file', got '%s'", writeSchema.Name)
	}
	
	if len(writeSchema.InputSchema.Required) != 2 {
		t.Errorf("Expected 2 required fields, got %d", len(writeSchema.InputSchema.Required))
	}
}

func TestDiscoverMCPToolSchemas_ServerError(t *testing.T) {
	// Create mock server that returns an error
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer server.Close()
	
	r := &LanguageToolReconciler{}
	endpoint := server.URL[7:] // Remove "http://" prefix
	
	_, err := r.discoverMCPToolSchemas(context.Background(), endpoint)
	
	if err == nil {
		t.Fatal("Expected error for server error response")
	}
}

func TestDiscoverMCPToolSchemas_MCPError(t *testing.T) {
	// Create mock server that returns MCP error
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		response := MCPResponse{
			JSONRpc: "2.0",
			ID:      1,
			Error: &MCPError{
				Code:    -32601,
				Message: "Method not found",
			},
		}
		
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
	}))
	defer server.Close()
	
	r := &LanguageToolReconciler{}
	endpoint := server.URL[7:] // Remove "http://" prefix
	
	_, err := r.discoverMCPToolSchemas(context.Background(), endpoint)
	
	if err == nil {
		t.Fatal("Expected error for MCP error response")
	}
	
	expectedError := "MCP server error: Method not found (code -32601)"
	if err.Error() != expectedError {
		t.Errorf("Expected error '%s', got '%s'", expectedError, err.Error())
	}
}