# LanguageTool Implementation Specification

This document defines the contract for implementing a `LanguageTool` — an MCP-compatible service that agents use for memory, knowledge retrieval, code execution, and other capabilities.

## Overview

A **LanguageTool** is a Kubernetes-managed service that exposes the [Model Context Protocol (MCP)](https://modelcontextprotocol.io/). The operator discovers tools via `LanguageTool` CRDs, resolves their endpoints, and injects connection details into agent containers via the `MCP_SERVERS` environment variable and the `tools:` section of `/etc/agent/config.yaml`.

Agents connect to tools over MCP. The operator does not proxy or inspect tool traffic — it only handles discovery and configuration injection.

## What the Operator Provides

When a `LanguageTool` resource is created, the operator:

1. Creates and manages the tool's Deployment and Service
2. Resolves the tool endpoint: `http://<service-name>.<namespace>.svc.cluster.local:<port>`
3. Syncs the MCP tool schema by calling `tools/list` on the endpoint
4. Injects the resolved endpoint into each referencing agent via the `MCP_SERVERS` env var and the `tools:` section of `/etc/agent/config.yaml`
5. Reconciles NetworkPolicy to allow agent pods to reach the tool service

## What the Tool Must Implement

### MCP Protocol (Required)

The tool must implement the [MCP specification](https://spec.modelcontextprotocol.io/) and respond on the port defined in `spec.port` (default: `8080`).

#### Tool Listing

```
POST /mcp
Content-Type: application/json

{"jsonrpc": "2.0", "id": 1, "method": "tools/list", "params": {}}
```

Response:

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
      },
      {
        "name": "recall",
        "description": "Retrieve information from long-term memory",
        "inputSchema": {
          "type": "object",
          "properties": {
            "query": { "type": "string", "description": "What to search for" }
          },
          "required": ["query"]
        }
      }
    ]
  }
}
```

#### Tool Invocation

```
POST /mcp
Content-Type: application/json

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

Response:

```json
{
  "jsonrpc": "2.0",
  "id": 2,
  "result": {
    "content": [{ "type": "text", "text": "Stored successfully." }]
  }
}
```

### Health Check (Required)

```
GET /health
```

Returns `200 OK` with a JSON body when the tool is ready to serve requests:

```json
{ "status": "ok" }
```

The operator polls this endpoint to determine tool availability before injecting the endpoint into agent configs. A tool that fails its health check will not be included in the agent's `tools.json`.

## LanguageTool CRD Reference

```yaml
apiVersion: langop.io/v1alpha1
kind: LanguageTool
metadata:
  name: mem0-memory
  namespace: language-operator-myapp
spec:
  image: myregistry/mem0-mcp:latest

  # Protocol type — only "mcp" is currently supported (default)
  type: mcp

  # Deployment mode:
  # - "service": standalone Deployment+Service shared across agents (default)
  # - "sidecar": injected as a sidecar into each agent pod (dedicated, with workspace access)
  deploymentMode: service

  # Port the tool listens on (default: 8080)
  port: 8080
```

## Agent Connection

The operator injects tool endpoints into each referencing agent two ways:

1. **`MCP_SERVERS` env var** — a comma-separated list of tool endpoint URLs (e.g. `http://mem0-memory.language-operator-myapp.svc.cluster.local:8080`)
2. **`/etc/agent/config.yaml`** — a `tools:` map keyed by tool name:

```yaml
tools:
  mem0-memory:
    endpoint: http://mem0-memory.language-operator-myapp.svc.cluster.local:8080
    protocol: mcp
```

Agents can use either source; `MCP_SERVERS` is convenient for simple single-tool lookup, while `config.yaml` provides structured access to all tools by name.

## Deployment Pattern

A `LanguageTool` is a self-contained resource — the operator creates the Deployment and Service automatically from `spec.image`. No separate Deployment or Service manifest is needed:

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

The operator creates a `Deployment` named `mem0-memory` and a `ClusterIP` Service named `mem0-memory` in the same namespace. The tool endpoint `http://mem0-memory.language-operator-myapp.svc.cluster.local:8080` is injected into every agent that references this tool.

## Compliance Checklist

A compliant tool implementation must:

- [ ] Implement the MCP JSON-RPC protocol at the configured port
- [ ] Respond to `tools/list` with a complete list of available tools and their input schemas
- [ ] Respond to `tools/call` with the result of invoking the named tool
- [ ] Expose `GET /health` returning `200 OK` with `{"status":"ok"}` when ready
- [ ] Expose an MCP-compliant HTTP endpoint on `spec.port` (the operator creates the Kubernetes Service automatically)
- [ ] Handle errors gracefully with MCP error responses (not HTTP 5xx)

## Error Responses

Tool errors must use MCP error format, not HTTP error codes:

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

Standard MCP error codes:

| Code | Meaning |
|------|---------|
| `-32700` | Parse error |
| `-32600` | Invalid request |
| `-32601` | Method not found |
| `-32602` | Invalid params |
| `-32603` | Internal error |
