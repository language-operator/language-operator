# Tools

A `LanguageTool` is an MCP-compatible service that agents call to do things beyond generating text — query a database, search the web, run code, read files. The operator manages the Deployment and Service, discovers what the tool can do, and injects its endpoint into every agent that references it.

## How it works

When you create a `LanguageTool`, the operator:

1. Deploys the tool as a Kubernetes Deployment and Service. For `transport: streamable-http` or `sse` it runs `spec.image` directly; for `transport: stdio` it injects a bridge image that wraps the stdio command and serves Streamable HTTP at `/mcp`.
2. Waits for the pod to become ready, then calls `tools/list` to discover what the tool exposes.
3. Stores the discovered schemas in `status.toolSchemas`.
4. Injects the tool's endpoint (including `/mcp`) into every referencing agent via `/etc/agent/config.yaml`.

Agents connect to tools directly over MCP — the operator doesn't proxy or inspect tool traffic.

## Transports

`spec.transport` selects how the tool's MCP endpoint is exposed:

| Transport | Description |
|-----------|-------------|
| `streamable-http` (default) | `spec.image` serves Streamable HTTP at `/mcp` on `spec.port`. |
| `sse` | `spec.image` serves the legacy MCP HTTP+SSE transport on `spec.port`. |
| `stdio` | The operator injects a bridge that runs `spec.stdio.command` as a persistent child and serves Streamable HTTP at `/mcp`. `spec.image` is ignored. |

### stdio tools

For `transport: stdio` you provide the full argv of the stdio MCP server in `spec.stdio.command`. The operator injects a pinned bridge (`ghcr.io/language-operator/mcp-bridge:latest`) that bundles Node and Python+uv, so both `npx` and `uvx` commands work without a custom image:

```yaml
spec:
  transport: stdio
  port: 8080
  stdio:
    command:
      - npx
      - -y
      - "@upstash/context7-mcp"
```

The bridge keeps one long-lived child process — there is no per-request spawn. Writable scratch volumes (`HOME` and `/tmp`) are injected automatically so npm/uv caches work even with `readOnlyRootFilesystem`.

## Deploying a tool

A `LanguageTool` is self-contained. You provide an image (or a stdio command); the operator creates the Deployment and Service:

```yaml
apiVersion: langop.io/v1alpha1
kind: LanguageTool
metadata:
  name: mem0-memory
  namespace: language-operator-myapp
spec:
  image: myregistry/mem0-mcp:latest
  port: 8080
```

That's enough to get a running tool with an in-cluster endpoint at `http://mem0-memory.language-operator-myapp.svc.cluster.local:8080`.

The `deploymentMode` field controls whether the tool runs as a shared service (default) or as a sidecar inside each agent pod:

- **`service`** — one Deployment shared across all agents that reference the tool. Good for stateless tools.
- **`sidecar`** — injected into each agent pod. Gets access to the agent's workspace volume. Good for tools that need per-agent state.

## MCP protocol

The tool must implement [MCP](https://spec.modelcontextprotocol.io/) on the port defined in `spec.port`. The operator calls two methods: `tools/list` for schema discovery, and agents call `tools/call` at runtime.

### Tool listing

```
POST /mcp
Content-Type: application/json
Accept: application/json, text/event-stream

{"jsonrpc": "2.0", "id": 1, "method": "tools/list", "params": {}}
```

```json
{
  "jsonrpc": "2.0",
  "id": 1,
  "result": {
    "tools": [
      {
        "name": "remember",
        "description": "Store information in long-term memory",
        "inputSchema": {
          "type": "object",
          "properties": {
            "content": { "type": "string", "description": "Information to remember" },
            "tags": { "type": "array", "items": { "type": "string" } }
          },
          "required": ["content"]
        }
      }
    ]
  }
}
```

### Tool invocation

```
POST /mcp
Content-Type: application/json
Accept: application/json, text/event-stream

{
  "jsonrpc": "2.0",
  "id": 2,
  "method": "tools/call",
  "params": {
    "name": "remember",
    "arguments": { "content": "User prefers metric units", "tags": ["preferences"] }
  }
}
```

```json
{
  "jsonrpc": "2.0",
  "id": 2,
  "result": {
    "content": [{ "type": "text", "text": "Stored successfully." }]
  }
}
```

### Health check

The operator uses `GET /health` to determine when the tool is ready before attempting schema discovery:

```json
{ "status": "ok" }
```

### Error responses

Tool errors should use MCP error format rather than HTTP error status codes:

```json
{
  "jsonrpc": "2.0",
  "id": 2,
  "error": {
    "code": -32603,
    "message": "Memory store unavailable",
    "data": { "reason": "connection refused" }
  }
}
```

| Code | Meaning |
|------|---------|
| `-32700` | Parse error |
| `-32600` | Invalid request |
| `-32601` | Method not found |
| `-32602` | Invalid params |
| `-32603` | Internal error |

## How agents receive tool endpoints

The operator writes each tool's endpoint into the referencing agent's `/etc/agent/config.yaml` under the `tools:` key:

```yaml
tools:
  mem0-memory:
    endpoint: http://mem0-memory.language-operator-myapp.svc.cluster.local:8080/mcp
    protocol: mcp
```

Runtime adapters (like `openclaw-adapter` and `opencode-adapter`) read this file and translate it into their runtime's native tool configuration before the agent starts.

## Checklist

For `transport: streamable-http` or `transport: sse`, the tool image must:

- [ ] Implement MCP JSON-RPC at the port defined in `spec.port`
- [ ] Respond to `tools/list` with all available tools and their input schemas
- [ ] Respond to `tools/call` with the result of invoking the named tool
- [ ] Expose `GET /health` returning `200 OK` with `{"status":"ok"}` when ready
- [ ] Return MCP error objects for failures, not HTTP error status codes

For `transport: stdio`, the operator-injected bridge handles all of the above — the stdio command only needs to speak the [MCP stdio transport](https://spec.modelcontextprotocol.io/specification/2025-11-05/basic/transports/#stdio).
