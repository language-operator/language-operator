# LanguageTool Implementation Specification

This document defines the contract for a `LanguageTool` — an MCP-compatible service that agents
use for memory, knowledge retrieval, code execution, and other capabilities.

## Overview

A **LanguageTool** exposes the [Model Context Protocol (MCP)](https://modelcontextprotocol.io/)
over HTTP. The operator discovers tools via `LanguageTool` CRDs, resolves their endpoints, and
injects connection details into agent containers via the `MCP_SERVERS` environment variable and
the `tools:` section of `/etc/agent/config.yaml`. Agents connect to the tool's `/mcp` endpoint
directly — the operator does not proxy tool traffic.

## Transports

`spec.transport` selects how the tool's MCP endpoint is produced:

| Transport | Meaning |
|-----------|---------|
| `streamable-http` (default) | `spec.image` already serves [Streamable HTTP](https://modelcontextprotocol.io/specification/2025-06-18/basic/transports) MCP at `/mcp`. |
| `sse` | `spec.image` serves the legacy MCP HTTP+SSE transport. |
| `stdio` | The tool is a stdio MCP program. You supply `spec.stdio.command`; the **operator** runs it under a pinned, persistent stdio→Streamable-HTTP bridge that serves `/mcp` (and `/health`) on `spec.port`. |

Most MCP servers in the ecosystem ship as **stdio** programs (npx/uvx packages). `transport:
stdio` is the bulletproof way to run them: the operator injects a bridge that maintains **one
long-lived child process** (no per-request spawn), plus a writable cache and `/tmp` so `npx`/`uvx`
can run under the hardened, read-only-root pod securityContext. You don't manage the bridge.

## What the operator provides

1. Creates and manages the tool's Deployment and Service (for `deploymentMode: service`).
2. Resolves the endpoint: `http://<service-name>.<namespace>.svc.cluster.local:<port>`.
3. Discovers the tool's schema by performing an MCP handshake against `/mcp` (see below) and
   records it in `status.toolSchemas`.
4. Injects the resolved endpoint into each referencing agent via `MCP_SERVERS` and the `tools:`
   section of `/etc/agent/config.yaml`.
5. Reconciles a NetworkPolicy allowing agent pods to reach the tool service.
6. For `transport: stdio`, injects the bridge image, the writable cache + `/tmp` volumes, and a
   `/health` readiness probe.

## Schema discovery (how the operator calls a tool)

When the tool's Deployment is ready (`ReadyReplicas > 0`), the operator discovers its tools with
a spec-compliant Streamable HTTP handshake to `POST /mcp`:

1. `initialize` — establishes the session; the operator captures the `Mcp-Session-Id` response
   header and sends `MCP-Protocol-Version`.
2. `notifications/initialized`.
3. `tools/list` — within the session.

Responses may be either `application/json` or an SSE stream (`text/event-stream`); the operator
parses both. If a server does not implement `initialize` (returns JSON-RPC `-32601` or HTTP 4xx),
the operator falls back to a bare `tools/list` for backward compatibility with simplified tools.

Discovery is **best-effort**: tool endpoints are injected into agents regardless of whether schema
discovery succeeds. The timeout is configurable via the operator's `--mcp-discovery-timeout`
(default 30s).

## What a `streamable-http` tool must implement

A natively-HTTP tool (the default transport) must serve MCP Streamable HTTP at `spec.port`:

- `POST /mcp` handling `initialize`, `notifications/initialized`, `tools/list`, and `tools/call`,
  honoring the `Mcp-Session-Id` and `MCP-Protocol-Version` headers, and replying with either JSON
  or an SSE stream.
- Errors returned as MCP/JSON-RPC errors (not HTTP 5xx).

> Simplified tools that answer a bare `tools/list` (no `initialize`) still work via the
> discovery fallback, but implementing the full handshake is recommended.

## What a `stdio` tool requires

Set `spec.transport: stdio` and provide the stdio command:

```yaml
spec:
  transport: stdio
  port: 8080
  stdio:
    command: ["npx", "-y", "@upstash/context7-mcp"]
  deployment:
    env:
      - name: CONTEXT7_API_KEY
        valueFrom:
          secretKeyRef: { name: context7-mcp-credentials, key: api-key, optional: true }
  # stdio tools that fetch their package (npx/uvx) need outbound egress — the default tool
  # policy denies it. Egress is not auto-injected (it stays explicit for security).
  networkPolicies:
    egress:
      - to: [{ cidr: "0.0.0.0/0" }]
        ports: [{ port: 443, protocol: TCP }]
```

Notes:
- `spec.image` is ignored for stdio tools (the operator injects the bridge image); it may be
  omitted — the defaulting webhook fills it.
- The bridge runs the command through a shell, so environment variables interpolate at runtime
  (e.g. a connection string passed as `"$DATABASE_URL"`).
- Prefer pinned, prebuilt images over `npx -y`/`uvx` at boot where you can (immutability and
  supply-chain), but the bridge supports the package-fetch pattern for convenience.

## LanguageTool CRD reference

```yaml
apiVersion: langop.io/v1alpha1
kind: LanguageTool
metadata:
  name: mem0-memory
  namespace: language-operator-myapp
spec:
  # Transport: streamable-http (default) | sse | stdio
  transport: streamable-http

  # For streamable-http/sse: the image that serves MCP at /mcp.
  image: myregistry/mem0-mcp:latest

  # For transport=stdio instead: the stdio command the operator wraps with a bridge.
  # stdio:
  #   command: ["npx", "-y", "@upstash/context7-mcp"]

  # Protocol type — only "mcp" is currently supported (default)
  type: mcp

  # Deployment mode:
  # - "service": standalone Deployment+Service shared across agents (default)
  # - "sidecar": injected as a sidecar into each agent pod (dedicated, with workspace access)
  deploymentMode: service

  # Port the tool (or bridge) listens on (default: 8080)
  port: 8080
```

## Agent connection

The operator injects tool endpoints into each referencing agent two ways:

1. **`MCP_SERVERS` env var** — a comma-separated list of tool endpoint URLs (e.g.
   `http://mem0-memory.language-operator-myapp.svc.cluster.local:8080`)
2. **`/etc/agent/config.yaml`** — a `tools:` map keyed by tool name:

```yaml
tools:
  mem0-memory:
    endpoint: http://mem0-memory.language-operator-myapp.svc.cluster.local:8080
    protocol: mcp
```

The endpoint is identical regardless of transport — agents always speak Streamable HTTP MCP to
`<endpoint>/mcp`.

## Compliance checklist

A compliant `streamable-http` tool must:

- [ ] Serve MCP Streamable HTTP on `spec.port` at `/mcp` (the operator creates the Service)
- [ ] Handle the `initialize` → `notifications/initialized` → `tools/list` handshake (or, at
      minimum, answer a bare `tools/list` to support discovery fallback)
- [ ] Respond to `tools/call` with the result of invoking the named tool
- [ ] Reply with `application/json` or `text/event-stream` as appropriate
- [ ] Return errors as MCP/JSON-RPC errors (not HTTP 5xx)

A `stdio` tool must:

- [ ] Set `spec.transport: stdio` and provide `spec.stdio.command`
- [ ] Declare any outbound `networkPolicies.egress` its command needs (npm/registry, the
      service it talks to)

## Error responses

Tool errors must use MCP/JSON-RPC error format, not HTTP error codes:

```json
{
  "jsonrpc": "2.0",
  "id": 2,
  "error": { "code": -32603, "message": "Memory store unavailable", "data": { "reason": "connection refused" } }
}
```

| Code | Meaning |
|------|---------|
| `-32700` | Parse error |
| `-32600` | Invalid request |
| `-32601` | Method not found |
| `-32602` | Invalid params |
| `-32603` | Internal error |
