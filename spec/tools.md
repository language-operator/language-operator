# LanguageTool Implementation Specification

This document defines the contract for implementing a `LanguageTool` — an MCP-compatible service that agents use for memory, knowledge retrieval, code execution, and other capabilities.

## Overview

A **LanguageTool** is a Kubernetes-managed service that exposes the [Model Context Protocol (MCP)](https://modelcontextprotocol.io/). The operator discovers tools via `LanguageTool` CRDs, resolves their endpoints, and injects connection details into agent containers via `/etc/agent/tools/config.json`.

Agents connect to tools over MCP. The operator does not proxy or inspect tool traffic — it only handles discovery and configuration injection.

## What the Operator Provides

When a `LanguageTool` resource is created, the operator:

1. Creates a Deployment and Service for the tool image
2. Resolves the tool endpoint: `http://<tool-name>.<namespace>.svc.cluster.local:<port>`
3. Syncs the MCP tool schema by calling `tools/list` on the endpoint
4. Writes the resolved endpoint into the `tools.json` ConfigMap for each agent referencing this tool
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
  namespace: my-cluster
spec:
  # Container image to run (required)
  image: myregistry/mem0-mcp:latest

  # Port the MCP service listens on (default: 8080)
  port: 8080

  # Deployment mode: "service" (default, shared) or "sidecar" (per-agent)
  deploymentMode: service

  # Pod and container configuration
  deployment:
    env:
      - name: MEM0_API_KEY
        valueFrom:
          secretKeyRef:
            name: mem0-secret
            key: api-key
    replicas: 1
    resources:
      requests:
        cpu: 100m
        memory: 128Mi

  # Network egress rules (default: full cluster access, no external egress)
  networkPolicies:
    - ports:
        - port: 443
      to:
        - cidr: "0.0.0.0/0"
```

## Agent Connection

The operator resolves the tool endpoint and writes it to each agent's `/etc/agent/tools/config.json`:

```json
{
  "mem0-memory": {
    "endpoint": "http://mem0-memory.tools.svc.cluster.local:8080",
    "protocol": "mcp"
  }
}
```

The agent reads this file on startup and uses the endpoint URL to make MCP JSON-RPC calls.

## Deployment Pattern

The operator creates the Deployment and Service automatically from `spec.image`. A single manifest is all that's needed:

```yaml
apiVersion: langop.io/v1alpha1
kind: LanguageTool
metadata:
  name: mem0-memory
  namespace: my-cluster
spec:
  image: myregistry/mem0-mcp:latest
  port: 8080
  deployment:
    env:
      - name: MEM0_API_KEY
        valueFrom:
          secretKeyRef:
            name: mem0-secret
            key: api-key
```

The operator reconciles a `Deployment` and a `Service` (both named after the `LanguageTool`) in the same namespace. There is no need to create these resources separately.

## Compliance Checklist

A compliant tool implementation must:

- [ ] Implement the MCP JSON-RPC protocol at the configured port
- [ ] Respond to `tools/list` with a complete list of available tools and their input schemas
- [ ] Respond to `tools/call` with the result of invoking the named tool
- [ ] Expose `GET /health` returning `200 OK` with `{"status":"ok"}` when ready
- [ ] Be packaged as a container image specified in `spec.image` (the operator creates the Service)
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
