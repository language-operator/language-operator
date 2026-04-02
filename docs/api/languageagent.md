# LanguageAgent

The `LanguageAgent` CRD represents an autonomous AI agent deployment in Kubernetes.

## Overview

A LanguageAgent runs a container image with:
- LLM access through the shared cluster proxy
- Tool endpoints for extended capabilities
- Persona configuration for behavioral templates
- Instructions for tasks and goals
- Workspace storage for persistent state

## Quick Example

```yaml
apiVersion: langop.io/v1alpha1
kind: LanguageAgent
metadata:
  name: my-agent
  namespace: my-cluster
spec:
  runtime: openclaw       # use a bundled LanguageAgentRuntime
  openclaw:
    token: changeme       # operator creates the credential Secret automatically
  models:
    - name: claude-sonnet
  workspace:
    size: 10Gi
```

Or with a custom image and no runtime:

```yaml
spec:
  image: ghcr.io/my-org/my-agent:latest
  models:
    - name: claude-sonnet
  instructions: |
    You are a helpful AI assistant.
```

## Complete API Reference

See the [Complete API Reference](reference.md#languageagent) for full field documentation including:

- **LanguageAgent** - Top-level resource
- **LanguageAgentSpec** - Specification fields
- **LanguageAgentStatus** - Status and conditions

## Key Concepts

### Runtimes

A `LanguageAgentRuntime` is a cluster-scoped preset that packages image, port, init containers, probes, and env vars for a specific agent type. Reference one with `spec.runtime`:

```yaml
spec:
  runtime: opencode
```

The standard runtimes (`openclaw`, `opencode`) are bundled with the Helm chart. See [LanguageAgentRuntime](languageagentruntime.md) for details.

### Runtime-Specific Configuration

Each standard runtime has a corresponding config block for inline credential injection. The operator creates a managed Secret and injects it via `envFrom` — no manual `kubectl create secret` needed.

**openclaw:**

```yaml
spec:
  runtime: openclaw
  openclaw:
    token: changeme           # inline — operator creates {agent}-runtime Secret
    # tokenRef:               # or reference a pre-existing Secret
    #   name: my-secret       # must contain OPENCLAW_GATEWAY_TOKEN
```

**opencode:**

```yaml
spec:
  runtime: opencode
  opencode:
    username: demo            # sets OPENCODE_SERVER_USERNAME (default: "opencode")
    password: changeme        # inline — operator creates {agent}-runtime Secret
    # passwordRef:            # or reference a pre-existing Secret
    #   name: my-secret       # must contain OPENCODE_SERVER_PASSWORD
```

### Execution Modes

- `autonomous` - Continuously running agent
- `scheduled` - Cron-based execution
- `interactive` - User-triggered execution
- `event-driven` - Responds to Kubernetes events

### Model References

Each entry in `spec.models` is a `ModelReference` with the following fields:

| Field | Type | Default | Description |
|-------|------|---------|-------------|
| `name` | string | required | Name of a `LanguageModel` resource |
| `role` | string | `primary` | Hint for the agent runtime. Valid values: `primary`, `fallback`, `reasoning`, `tool-calling`, `summarization` |
| `priority` | integer | — | Optional selection priority hint; lower value = higher priority |

The `role` and `priority` fields are surfaced in `/etc/agent/config.yaml` under each model entry. The operator does not enforce them — they are hints for the agent runtime's model selection logic.

Example:

```yaml
models:
  - name: claude-sonnet
    role: primary
  - name: claude-haiku
    role: fallback
    priority: 2
```

### Tool References

Each entry in `spec.tools` is a `ToolReference` with the following fields:

| Field | Type | Default | Description |
|-------|------|---------|-------------|
| `name` | string | required | Name of a `LanguageTool` resource |
| `enabled` | boolean | `true` | Set to `false` to temporarily disable a tool without removing the reference |

When `enabled` is `false`, the tool endpoint is not injected into `/etc/agent/config.yaml` and not included in `MCP_SERVERS`.

Example:

```yaml
tools:
  - name: web-search
    enabled: true
  - name: code-executor
    enabled: false   # disabled — endpoint not injected
```

### Configuration Injection

The operator automatically mounts:

- `/etc/agent/config.yaml` - Instructions, personas, models, tools

Environment variables injected into every agent container and all init containers:

| Variable | Value |
|----------|-------|
| `AGENT_NAME` | `metadata.name` of the LanguageAgent |
| `AGENT_NAMESPACE` | `metadata.namespace` of the LanguageAgent |
| `AGENT_UUID` | Stable UUID assigned to this agent (from `status.uuid`) |
| `AGENT_MODE` | Execution mode from `spec.executionMode` (omitted if not set) |
| `AGENT_CLUSTER_NAME` | Name of the LanguageCluster this agent belongs to |
| `AGENT_CLUSTER_UUID` | Kubernetes UID of the LanguageCluster |
| `MODEL_ENDPOINTS` | Shared LiteLLM gateway URL (`http://gateway.<namespace>.svc.cluster.local:8000`) |
| `LLM_MODEL` | Comma-separated list of model names for all referenced models |
| `MCP_SERVERS` | Comma-separated MCP tool server URLs (only injected when at least one tool is resolved) |
| `AGENT_INSTRUCTIONS` | Content of `spec.instructions`; only set when instructions are non-empty |
| `OTEL_EXPORTER_OTLP_ENDPOINT` | Propagated from the operator environment when configured |
| `OTEL_SERVICE_NAME` | Set to `agent-<name>` when `OTEL_EXPORTER_OTLP_ENDPOINT` is configured |
| `OTEL_RESOURCE_ATTRIBUTES` | Propagated from the operator environment (conditional on OTEL endpoint) |
| `OTEL_TRACES_SAMPLER` | Propagated from the operator environment (conditional on OTEL endpoint) |
| `OTEL_TRACES_SAMPLER_ARG` | Propagated from the operator environment (conditional on OTEL endpoint) |

Additional variables from `spec.deployment.env` and `spec.deployment.envFrom` are passed through unchanged. See [Environment Variables](../architecture/agents.md#environment-variables) in the architecture docs for the full reference.

### Resource Management

Agents are deployed as standard Kubernetes Deployments with:

- Configurable replicas (`spec.deployment.replicas`)
- Resource limits and requests (`spec.deployment.resources`)
- Node selectors, tolerations, and affinity rules
- Custom liveness, readiness, and startup probes

## Related Resources

- [LanguageModel](languagemodel.md) - Configure LLM access
- [LanguageTool](languagetool.md) - Add tool capabilities
- [LanguagePersona](languagepersona.md) - Define behavioral templates
- [Agent Runtime Contract](../architecture/agents.md) - What the operator injects

## Examples

See [Examples](../getting-started/examples.md) for common deployment patterns.
