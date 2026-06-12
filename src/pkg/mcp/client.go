// Package mcp implements a minimal Model Context Protocol (MCP) client used by the operator
// to discover the tools an MCP server exposes. It speaks the Streamable HTTP transport: a
// proper initialize → notifications/initialized → tools/list handshake (with session and
// protocol-version headers), parsing both plain-JSON and SSE-framed responses, and falling
// back to a bare tools/list for simplified servers that don't implement initialize.
package mcp

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
)

// DefaultProtocolVersion is the MCP protocol version the client advertises during the
// handshake. 2024-11-05 is broadly supported, including by the supergateway bridge.
const DefaultProtocolVersion = "2024-11-05"

// ErrInitializeUnsupported indicates the server rejected the MCP initialize handshake — e.g.
// a simplified server that only answers a bare tools/list. Callers fall back to ListToolsLegacy.
var ErrInitializeUnsupported = errors.New("mcp: server does not support the initialize handshake")

// Tool is a discovered MCP tool and its input schema.
type Tool struct {
	Name        string
	Description string
	InputSchema *InputSchema
}

// InputSchema is the JSON-schema description of a tool's arguments.
type InputSchema struct {
	Type       string
	Properties map[string]Property
	Required   []string
}

// Property is a single field within an InputSchema.
type Property struct {
	Type        string
	Description string
	Examples    []string
}

// Client is a minimal MCP Streamable HTTP client for schema discovery.
type Client struct {
	// HTTP is the HTTP client used for requests. When nil, http.DefaultClient is used.
	HTTP *http.Client
	// ProtocolVersion overrides the advertised MCP protocol version. Defaults to DefaultProtocolVersion.
	ProtocolVersion string
}

// ListTools performs the full MCP handshake against baseURL's /mcp endpoint and returns the
// discovered tools. If the server rejects the initialize handshake, it transparently falls
// back to a bare tools/list. baseURL is like "http://host:port" (no /mcp suffix).
func (c *Client) ListTools(ctx context.Context, baseURL string) ([]Tool, error) {
	tools, err := c.listToolsHandshake(ctx, baseURL)
	if errors.Is(err, ErrInitializeUnsupported) {
		return c.ListToolsLegacy(ctx, baseURL)
	}
	return tools, err
}

// ListToolsLegacy posts a bare tools/list with no handshake — the pre-Streamable-HTTP dialect
// some simplified operator-native tools implement.
func (c *Client) ListToolsLegacy(ctx context.Context, baseURL string) ([]Tool, error) {
	resp, _, err := c.do(ctx, mcpEndpoint(baseURL), "", rpcRequest{JSONRPC: "2.0", ID: 1, Method: "tools/list"})
	if err != nil {
		return nil, err
	}
	return toolsFromResponse(resp)
}

func (c *Client) listToolsHandshake(ctx context.Context, baseURL string) ([]Tool, error) {
	endpoint := mcpEndpoint(baseURL)

	// 1. initialize — establish the session.
	initResp, sessionID, err := c.do(ctx, endpoint, "", rpcRequest{
		JSONRPC: "2.0",
		ID:      1,
		Method:  "initialize",
		Params: map[string]any{
			"protocolVersion": c.protocolVersion(),
			"capabilities":    map[string]any{},
			"clientInfo":      map[string]any{"name": "language-operator", "version": "v1alpha1"},
		},
	})
	if err != nil {
		// A 4xx means the server doesn't speak the handshake (e.g. simplified tools/list-only
		// server, or one demanding a different shape) — fall back rather than fail discovery.
		var se *statusError
		if errors.As(err, &se) && se.code >= 400 && se.code < 500 {
			return nil, ErrInitializeUnsupported
		}
		return nil, err
	}
	if initResp != nil && initResp.Error != nil {
		if initResp.Error.Code == -32601 { // method not found
			return nil, ErrInitializeUnsupported
		}
		return nil, fmt.Errorf("mcp initialize error: %s (code %d)", initResp.Error.Message, initResp.Error.Code)
	}

	// 2. notifications/initialized — required by the spec before normal requests. It's a
	// notification (no id, no response); ignore transport-level hiccups.
	_ = c.notify(ctx, endpoint, sessionID, rpcRequest{JSONRPC: "2.0", Method: "notifications/initialized"})

	// 3. tools/list — within the established session.
	listResp, _, err := c.do(ctx, endpoint, sessionID, rpcRequest{JSONRPC: "2.0", ID: 2, Method: "tools/list"})
	if err != nil {
		return nil, err
	}
	return toolsFromResponse(listResp)
}

// do sends a JSON-RPC request and returns the parsed response plus any session id the server
// assigned (from the Mcp-Session-Id response header).
func (c *Client) do(ctx context.Context, endpoint, sessionID string, req rpcRequest) (*rpcResponse, string, error) {
	httpReq, err := c.newRequest(ctx, endpoint, sessionID, req)
	if err != nil {
		return nil, "", err
	}
	resp, err := c.httpClient().Do(httpReq)
	if err != nil {
		return nil, "", err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, "", err
	}
	sid := resp.Header.Get("Mcp-Session-Id")
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, sid, &statusError{code: resp.StatusCode, body: string(body)}
	}
	parsed, err := parseJSONRPCResponse(resp.Header.Get("Content-Type"), body)
	if err != nil {
		return nil, sid, err
	}
	return parsed, sid, nil
}

// notify sends a JSON-RPC notification (no response expected).
func (c *Client) notify(ctx context.Context, endpoint, sessionID string, req rpcRequest) error {
	httpReq, err := c.newRequest(ctx, endpoint, sessionID, req)
	if err != nil {
		return err
	}
	resp, err := c.httpClient().Do(httpReq)
	if err != nil {
		return err
	}
	_ = resp.Body.Close()
	return nil
}

func (c *Client) newRequest(ctx context.Context, endpoint, sessionID string, req rpcRequest) (*http.Request, error) {
	body, err := json.Marshal(req)
	if err != nil {
		return nil, err
	}
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Accept", "application/json, text/event-stream")
	httpReq.Header.Set("MCP-Protocol-Version", c.protocolVersion())
	if sessionID != "" {
		httpReq.Header.Set("Mcp-Session-Id", sessionID)
	}
	return httpReq, nil
}

func (c *Client) protocolVersion() string {
	if c.ProtocolVersion != "" {
		return c.ProtocolVersion
	}
	return DefaultProtocolVersion
}

func (c *Client) httpClient() *http.Client {
	if c.HTTP != nil {
		return c.HTTP
	}
	return http.DefaultClient
}

// mcpEndpoint appends the /mcp path to a base URL.
func mcpEndpoint(baseURL string) string {
	return strings.TrimRight(baseURL, "/") + "/mcp"
}

// parseJSONRPCResponse decodes a JSON-RPC response that may be either a plain JSON body or an
// SSE stream (Streamable HTTP lets the server answer a POST with text/event-stream).
func parseJSONRPCResponse(contentType string, body []byte) (*rpcResponse, error) {
	if strings.Contains(contentType, "text/event-stream") {
		return parseSSEResponse(body)
	}
	var r rpcResponse
	if err := json.Unmarshal(body, &r); err != nil {
		// Some servers stream SSE without the matching content-type; try that before giving up.
		if sse, sseErr := parseSSEResponse(body); sseErr == nil {
			return sse, nil
		}
		return nil, fmt.Errorf("mcp: failed to parse response: %w", err)
	}
	return &r, nil
}

// parseSSEResponse extracts the first JSON-RPC response message from an SSE stream. Per the
// SSE spec, an event's data lines (joined with newlines) form the payload; a blank line ends
// the event. We return the first event whose payload is a JSON-RPC response (has result/error).
func parseSSEResponse(body []byte) (*rpcResponse, error) {
	scanner := bufio.NewScanner(bytes.NewReader(body))
	scanner.Buffer(make([]byte, 0, 64*1024), 4*1024*1024)

	var data []string
	tryParse := func() (*rpcResponse, bool) {
		if len(data) == 0 {
			return nil, false
		}
		var r rpcResponse
		if err := json.Unmarshal([]byte(strings.Join(data, "\n")), &r); err == nil && (r.Result != nil || r.Error != nil) {
			return &r, true
		}
		return nil, false
	}

	for scanner.Scan() {
		line := scanner.Text()
		switch {
		case strings.HasPrefix(line, "data:"):
			data = append(data, strings.TrimPrefix(strings.TrimPrefix(line, "data:"), " "))
		case line == "":
			if r, ok := tryParse(); ok {
				return r, nil
			}
			data = data[:0]
		default:
			// ignore event:, id:, retry:, and comment lines
		}
	}
	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("mcp: error reading SSE stream: %w", err)
	}
	if r, ok := tryParse(); ok {
		return r, nil
	}
	return nil, fmt.Errorf("mcp: no JSON-RPC message found in SSE stream")
}

func toolsFromResponse(resp *rpcResponse) ([]Tool, error) {
	if resp == nil {
		return nil, fmt.Errorf("mcp: empty response")
	}
	if resp.Error != nil {
		return nil, fmt.Errorf("mcp server error: %s (code %d)", resp.Error.Message, resp.Error.Code)
	}
	var result toolsListResult
	if err := json.Unmarshal(resp.Result, &result); err != nil {
		return nil, fmt.Errorf("mcp: failed to parse tools/list result: %w", err)
	}
	tools := make([]Tool, 0, len(result.Tools))
	for _, wt := range result.Tools {
		t := Tool{Name: wt.Name, Description: wt.Description}
		if wt.InputSchema != nil {
			is := &InputSchema{Type: wt.InputSchema.Type, Required: wt.InputSchema.Required}
			if len(wt.InputSchema.Properties) > 0 {
				is.Properties = make(map[string]Property, len(wt.InputSchema.Properties))
				for name, wp := range wt.InputSchema.Properties {
					is.Properties[name] = Property{Type: wp.Type, Description: wp.Description, Examples: wp.Examples}
				}
			}
			t.InputSchema = is
		}
		tools = append(tools, t)
	}
	return tools, nil
}

// statusError is returned by do for non-2xx HTTP responses.
type statusError struct {
	code int
	body string
}

func (e *statusError) Error() string {
	return fmt.Sprintf("mcp: server returned status %d", e.code)
}

// JSON-RPC wire types.

type rpcRequest struct {
	JSONRPC string `json:"jsonrpc"`
	ID      int    `json:"id,omitempty"`
	Method  string `json:"method"`
	Params  any    `json:"params,omitempty"`
}

type rpcResponse struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      int             `json:"id"`
	Result  json.RawMessage `json:"result,omitempty"`
	Error   *rpcError       `json:"error,omitempty"`
}

type rpcError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

type toolsListResult struct {
	Tools []wireTool `json:"tools"`
}

type wireTool struct {
	Name        string           `json:"name"`
	Description string           `json:"description,omitempty"`
	InputSchema *wireInputSchema `json:"inputSchema,omitempty"`
}

type wireInputSchema struct {
	Type       string                  `json:"type,omitempty"`
	Properties map[string]wireProperty `json:"properties,omitempty"`
	Required   []string                `json:"required,omitempty"`
}

type wireProperty struct {
	Type        string   `json:"type"`
	Description string   `json:"description,omitempty"`
	Examples    []string `json:"examples,omitempty"`
}
