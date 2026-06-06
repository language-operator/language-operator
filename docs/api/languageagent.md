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

### Credentials

Agents inject credentials through the generic `spec.credentials` list. Each entry's `name` is both the environment variable name and the key in the operator-managed Secret:

```yaml
spec:
  runtime: openclaw
  credentials:
    - name: OPENCLAW_GATEWAY_TOKEN     # auto-generated once, persisted, never rotated
    - name: MY_API_KEY
      value: "literal-value"           # stored in the {agent}-runtime Secret
    - name: SHARED_SECRET
      valueFrom:
        name: my-existing-secret       # all keys injected via envFrom; operator manages nothing
```

Each entry is resolved in priority order:

- **`valueFrom` set** — the referenced Secret's keys are injected via `envFrom`; the operator creates no Secret of its own.
- **`value` set** — the literal is stored in an operator-managed Secret named `{agent}-runtime` and injected via `envFrom`.
- **neither set** — the operator auto-generates a random value once, persists it in the `{agent}-runtime` Secret, and preserves it across reconciles (it is never rotated).

Entries declared by the agent's runtime are merged first, then the agent's own entries are appended. Entries are deduplicated by `name`, with the agent's entry winning on a collision. Runtimes typically declare the credentials their image needs (for example, the `openclaw` runtime declares `OPENCLAW_GATEWAY_TOKEN`), so most agents need no `credentials` block at all.

### Authentication

Agents cannot configure authentication directly. Whether an agent sits behind the cluster's OIDC proxy is determined by its runtime's `auth.enabled` setting combined with the cluster's `auth.enabled` setting. See [LanguageAgentRuntime](languageagentruntime.md#authentication) and [Clusters](../components/clusters.md#authentication) for the effective model.

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

### Port References

Each entry in `spec.ports` is an `AgentPort` with the following fields:

| Field | Type | Default | Description |
|-------|------|---------|-------------|
| `name` | string | required | Port name; used as the Service port name. Must match `^[a-z][a-z0-9-]*$`, max 15 characters |
| `port` | int32 | required | Container port number (1–65535) |
| `protocol` | string | `TCP` | Transport protocol: `TCP`, `UDP`, or `SCTP` |
| `expose` | bool | `false` | When `true`, the HTTPRoute targets this port for external access. If no port has `expose: true`, the first port is used. At most one port should have `expose: true` |

When `spec.ports` is empty, the operator defaults to a single port named `http` on port `8080`.

Example:

```yaml
ports:
  - name: http
    port: 8080
    expose: true
  - name: metrics
    port: 9090   # internal only — not exposed via HTTPRoute
  - name: data
    port: 5000
    protocol: UDP
```

### Network Policies

Control what traffic agents can send and receive:

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

Each peer in `ingress[].from` and `egress[].to` is a `NetworkPeer`. See [NetworkPeer fields](languagecluster.md#networkpeer-fields) for the full field reference including `dns` (FQDN-based egress) and `group` (langop label selector).

### Configuration Injection

The operator automatically mounts:

- `/etc/agent/config.yaml` - Instructions, personas, models, tools

Environment variables injected into every agent container and all init containers:

| Variable | Value |
|----------|-------|
| `AGENT_NAME` | `metadata.name` of the LanguageAgent |
| `AGENT_NAMESPACE` | `metadata.namespace` of the LanguageAgent |
| `AGENT_UUID` | Stable UUID assigned to this agent (from `status.uuid`) |
| `AGENT_CLUSTER_NAME` | Name of the LanguageCluster this agent belongs to |
| `AGENT_CLUSTER_UUID` | Kubernetes UID of the LanguageCluster |
| `MODEL_ENDPOINT` | Shared LiteLLM gateway URL (`http://gateway.<namespace>.svc.cluster.local:8000`) |
| `LLM_MODEL` | Comma-separated list of model names for all referenced models |
| `MCP_SERVERS` | Comma-separated MCP tool server URLs (only injected when at least one tool is resolved) |
| `AGENT_INSTRUCTIONS` | Content of `spec.instructions`; only set when instructions are non-empty |
| `OTEL_EXPORTER_OTLP_ENDPOINT` | Propagated from the operator environment when configured |
| `OTEL_SERVICE_NAME` | Set to `agent-<name>` when `OTEL_EXPORTER_OTLP_ENDPOINT` is configured |
| `OTEL_RESOURCE_ATTRIBUTES` | Propagated from the operator environment (conditional on OTEL endpoint) |
| `OTEL_TRACES_SAMPLER` | Propagated from the operator environment (conditional on OTEL endpoint) |
| `OTEL_TRACES_SAMPLER_ARG` | Propagated from the operator environment (conditional on OTEL endpoint) |

Additional variables from `spec.deployment.env` and `spec.deployment.envFrom` are passed through unchanged. See [Environment Variables](../components/agents.md#environment-variables) in the architecture docs for the full reference.

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
- [Agent Runtime Contract](../components/agents.md) - What the operator injects

