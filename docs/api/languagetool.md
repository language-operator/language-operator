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

- Serve HTTP on configured port (default 8080)
- Expose `/tools` endpoint listing available tools
- Expose `/invoke` endpoint for tool execution

See [Tool Protocol](../architecture/tools.md) for the full specification.

### Deployment Modes

**Service Mode** (default):
- Standalone Deployment shared by multiple agents
- More efficient for stateless tools
- Scales independently

**Sidecar Mode** (future):
- Tool container injected into each agent pod
- Better for stateful or agent-specific tools
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

### Network Egress

Control what external resources tools can access:

```yaml
spec:
  networkEgress:
    - host: api.brave.com
      port: 443
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
  networkEgress:
    - host: api.brave.com
      port: 443
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
  networkEgress:
    - host: postgres.database.svc.cluster.local
      port: 5432
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
kubectl get languagetool web-search -o jsonpath='{.status.availableTools}'
```

Shows all tools exposed by the service.

## Related Resources

- [LanguageAgent](languageagent.md) - Reference tools in agents
- [Tool Protocol](../architecture/tools.md) - MCP specification

## Examples

See [Examples](../getting-started/examples.md) for common tool configurations.
