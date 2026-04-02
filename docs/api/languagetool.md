# LanguageTool

The `LanguageTool` CRD deploys MCP (Model Context Protocol) tool servers that extend agent capabilities.

## Overview

A LanguageTool provides:
- MCP-compatible tool server deployment
- Automatic endpoint injection into agents
- Tool schema discovery and validation
- Network egress policies for tool access

## Quick Example

```yaml
apiVersion: langop.io/v1alpha1
kind: LanguageTool
metadata:
  name: web-search
  namespace: my-cluster
spec:
  image: mcp/brave-search:latest
  deployment:
    env:
      - name: BRAVE_API_KEY
        valueFrom:
          secretKeyRef:
            name: brave-api-key
            key: api-key
```

## Complete API Reference

See the [Complete API Reference](reference.md#languagetool) for full field documentation including:

- **LanguageTool** - Top-level resource
- **LanguageToolSpec** - Specification fields
- **LanguageToolStatus** - Status and available tools

## Key Concepts

### MCP Protocol

Tools must implement the Model Context Protocol:

- Implement JSON-RPC 2.0 at `POST /mcp` on the configured port (default `8080`)
- Respond to `tools/list` method with available tools and their schemas
- Respond to `tools/call` method to execute a named tool
- Expose `GET /health` returning `{"status":"ok"}` when ready

See [Tool Protocol](../architecture/tools.md) for the full specification.

### Deployment Modes

**Service Mode** (default):
- Standalone Deployment shared by multiple agents
- More efficient for stateless tools
- Scales independently

**Sidecar Mode**:
- Tool container injected as a sidecar into each agent pod
- Endpoint injected as `http://localhost:<port>` (not a Service URL)
- Better for stateful or agent-specific tools that need workspace access
- Shares agent lifecycle

### Endpoint Injection

When an agent references a tool:

```yaml
spec:
  tools:
    - name: web-search
```

The operator injects:
- `MCP_SERVERS` environment variable with the tool's Service URL
- Tool metadata in `/etc/agent/config.yaml`

### Network Policies

Control what external resources tools can access:

```yaml
spec:
  networkPolicies:
    egress:
      - to:
          - cidr: "0.0.0.0/0"
        ports:
          - port: 443
            protocol: TCP
```

NetworkPolicy is generated to allow only specified destinations.

## Common Tool Examples

### Web Search

```yaml
apiVersion: langop.io/v1alpha1
kind: LanguageTool
metadata:
  name: web-search
spec:
  image: mcp/brave-search:latest
  deployment:
    env:
      - name: BRAVE_API_KEY
        valueFrom:
          secretKeyRef:
            name: brave-api-key
            key: api-key
  networkPolicies:
    egress:
      - to:
          - cidr: "0.0.0.0/0"
        ports:
          - port: 443
            protocol: TCP
```

### Database Access

```yaml
apiVersion: langop.io/v1alpha1
kind: LanguageTool
metadata:
  name: postgres-client
spec:
  image: mcp/postgres:latest
  deployment:
    env:
      - name: DATABASE_URL
        valueFrom:
          secretKeyRef:
            name: db-credentials
            key: url
  networkPolicies:
    egress:
      - to:
          - podSelector:
              matchLabels:
                app: postgres
        ports:
          - port: 5432
            protocol: TCP
```

### Custom Tool

```yaml
apiVersion: langop.io/v1alpha1
kind: LanguageTool
metadata:
  name: my-custom-tool
spec:
  image: my-registry/custom-tool:v1.0.0
  port: 8080
  deployment:
    replicas: 3
    resources:
      requests:
        cpu: 100m
        memory: 128Mi
```

## Tool Discovery

The operator queries tool schemas on startup:

```bash
kubectl get languagetool web-search -o jsonpath='{.status.toolSchemas}'
```

Shows all tools exposed by the service.

## Related Resources

- [LanguageAgent](languageagent.md) - Reference tools in agents
- [Tool Protocol](../architecture/tools.md) - MCP specification

## Examples

See [Examples](../getting-started/examples.md) for common tool configurations.
