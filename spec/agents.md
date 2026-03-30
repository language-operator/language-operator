# Agent Runtime Container Specification

This document defines the contract between the Language Operator and agent runtime containers. It specifies what the operator guarantees to provide to every running agent, and what a compliant agent image must implement.

## Overview

A **LanguageAgent** runs as a standard Kubernetes Deployment (or CronJob for scheduled agents). The operator manages the pod lifecycle, injects configuration, and handles networking. The agent container is responsible for its own runtime logic: reading instructions, connecting to tools, and executing tasks.

## What the Operator Provides

### Mounted Files

The operator mounts exactly two files into every agent container:

| Path | Content | Source |
|------|---------|--------|
| `/etc/agent/instructions.txt` | Task instructions (plain text) | `spec.instructions` (inline string) |
| `/etc/agent/config.yaml` | Structured agent configuration (YAML) | Assembled by the operator from personas, tools, models, and agent metadata |

Files are read-only. The operator reconciles them on every change to the LanguageAgent spec or referenced resources.

### Workspace Storage

When `spec.workspace.enabled` is true (the default), the operator creates a PersistentVolumeClaim named `<agent-name>-workspace` and mounts it read-write into the agent container and all init containers:

| Path | Content |
|------|---------|
| `spec.workspace.mountPath` (default `/workspace`) | Read-write persistent volume, backed by a PVC |

The workspace survives pod restarts and redeployments. It does not survive deletion of the LanguageAgent.

Relevant spec fields:

| Field | Default | Description |
|-------|---------|-------------|
| `spec.workspace.enabled` | `true` | Create and mount a workspace PVC |
| `spec.workspace.size` | `10Gi` | PVC storage request |
| `spec.workspace.mountPath` | `/workspace` | Mount path in the container |
| `spec.workspace.storageClassName` | cluster default | StorageClass for the PVC |
| `spec.workspace.accessMode` | `ReadWriteOnce` | PVC access mode |

The volume name inside the pod spec is `workspace`. Init containers that need to pre-seed the workspace (e.g. config adapters) should mount it explicitly by this name:

```yaml
initContainers:
  - name: seed-config
    image: myregistry/config-adapter:latest
    volumeMounts:
      - name: workspace
        mountPath: /workspace
```

### Environment Variables

The operator injects the following environment variables into every agent container and all init containers:

| Variable | Value |
|----------|-------|
| `AGENT_NAME` | `metadata.name` of the LanguageAgent |
| `AGENT_NAMESPACE` | `metadata.namespace` of the LanguageAgent |
| `AGENT_UUID` | Stable UUID assigned to this agent (from `status.uuid`) |
| `AGENT_MODE` | Execution mode from `spec.executionMode` (omitted if not set) |
| `AGENT_CLUSTER_NAME` | Name of the LanguageCluster this agent belongs to |
| `AGENT_CLUSTER_UUID` | Kubernetes UID of the LanguageCluster |
| `MODEL_ENDPOINTS` | Single shared LiteLLM gateway URL (`http://gateway.<namespace>.svc.cluster.local:8000`). The same URL is used regardless of how many models are referenced. |
| `LLM_MODEL` | Comma-separated list of model names for all referenced models |
| `MCP_SERVERS` | Comma-separated MCP tool server URLs for each referenced LanguageTool (only injected when at least one service-mode tool is resolved) |
| `AGENT_INSTRUCTIONS` | Content of `spec.instructions`; only set when instructions are non-empty. Identical to the content of `/etc/agent/instructions.txt`. |
| `OTEL_EXPORTER_OTLP_ENDPOINT` | Propagated from the operator environment when configured; enables agent-side OTEL tracing |
| `OTEL_SERVICE_NAME` | Set to `agent-<name>` when `OTEL_EXPORTER_OTLP_ENDPOINT` is configured |
| `OTEL_RESOURCE_ATTRIBUTES` | Propagated from the operator environment when set (conditional on `OTEL_EXPORTER_OTLP_ENDPOINT`) |
| `OTEL_TRACES_SAMPLER` | Propagated from the operator environment when set (conditional on `OTEL_EXPORTER_OTLP_ENDPOINT`) |
| `OTEL_TRACES_SAMPLER_ARG` | Propagated from the operator environment when set (conditional on `OTEL_EXPORTER_OTLP_ENDPOINT`) |

Additional environment variables from `spec.deployment.env` and `spec.deployment.envFrom` are passed through unchanged.

### Networking

Every agent gets:
- A **ClusterIP Service** on `spec.port` (default `8080`) named `<agent-name>` in the agent's namespace
- An **HTTPRoute** for external access (if a Gateway is configured)
- **NetworkPolicy** permitting inbound traffic from other agent pods in the same cluster namespace

## What the Agent Must Implement

### Port

The agent listens on `spec.port` (default `8080`). The operator creates a ClusterIP Service on this port. What the agent serves there is up to the image — HTTP, gRPC, OpenAI-compatible API, or anything else.

### Probes

Liveness and readiness probes are configured via `spec.livenessProbe` and `spec.readinessProbe`. If not set, no probes are configured. The operator does not require any specific health endpoint — probe configuration is entirely up to the agent author.

### Startup Behaviour

On startup, the agent should:

1. Read `/etc/agent/instructions.txt` for task definition (if present)
2. Read `/etc/agent/config.yaml` for all other configuration (personas, tools, models)
3. Start listening on `spec.port`

## File Formats

### Instructions (`/etc/agent/instructions.txt`)

Plain text. The task definition for this agent — what it should do, its role, and any behavioural directives.

```
You are a data analyst. Analyze CSV files and generate insights.
Focus on trends, anomalies, and actionable recommendations.
Always cite data sources and use structured output.
```

### Agent Config (`/etc/agent/config.yaml`)

A single YAML document assembled by the operator. Contains everything the agent needs to configure its runtime: persona(s), tool endpoints, and model configuration.

```yaml
# Agent identity (mirrors AGENT_* env vars for convenience)
agent:
  name: data-analyst
  namespace: default
  uuid: "550e8400-e29b-41d4-a716-446655440000"
  mode: autonomous
  clusterName: production-cluster
  clusterUUID: "6ba7b810-9dad-11d1-80b4-00c04fd430c8"

# Persona configuration — merged in order if multiple are specified.
# Each entry is the full spec of the referenced LanguagePersona resource.
personas:
  - name: analytical-persona
    tone: professional
    personality: Analytical and precise, always cites data sources and uses structured output
    expertise: Data analysis, statistical reasoning, and visualization

# Tool endpoints — keyed by tool name, resolved to in-cluster MCP service URLs.
tools:
  mem0-memory:
    endpoint: http://mem0-memory.tools.svc.cluster.local:8080
    protocol: mcp
  python-executor:
    endpoint: http://python-executor.tools.svc.cluster.local:8080
    protocol: mcp

# Model configuration — keyed by model name.
# All LLM traffic routes through the shared namespace gateway.
# Agents never hold real API credentials.
models:
  claude-sonnet:
    role: primary
    provider: anthropic
    model: claude-sonnet-4-5
    endpoint: http://gateway.default.svc.cluster.local:8000
```

The `MODEL_ENDPOINTS` env var also carries this gateway URL for runtimes that prefer environment-variable-based configuration.

## Example LanguageAgent YAML

```yaml
apiVersion: langop.io/v1alpha1
kind: LanguageAgent
metadata:
  name: data-analyst
  namespace: default
spec:
  image: myregistry/agent-runtime:python-v1.0.0
  port: 8080

  instructions: |
    You are a data analyst. Analyze CSV files and generate insights.
    Focus on trends, anomalies, and actionable recommendations.

  persona: analytical-persona

  executionMode: autonomous

  tools:
    - name: mem0-memory
    - name: python-executor

  models:
    - name: claude-sonnet

  workspace:
    size: 10Gi
    mountPath: /workspace   # agents can read/write freely; survives restarts

  deployment:
    imagePullPolicy: Always
    replicas: 1
    resources:
      limits:
        memory: 1Gi
        cpu: 500m
    livenessProbe:
      httpGet:
        path: /health
        port: 8080
      initialDelaySeconds: 10
      periodSeconds: 30
    readinessProbe:
      httpGet:
        path: /health
        port: 8080
      initialDelaySeconds: 5
      periodSeconds: 10
```

### Init Container Pattern (Config Adapters)

For agent runtimes that require configuration in a format other than `/etc/agent/config.yaml`, use an init container to translate before the agent starts. The init container shares the workspace volume:

```yaml
spec:
  image: ghcr.io/myorg/my-agent:latest
  port: 18789

  initContainers:
    - name: seed-config
      image: myregistry/config-adapter:latest
      env:
        - name: STATE_DIR
          value: /workspace/.config
      volumeMounts:
        - name: workspace
          mountPath: /workspace
        - name: agent-config
          mountPath: /etc/agent
          readOnly: true

  workspace:
    size: 10Gi
    mountPath: /workspace
```

The init container runs to completion before the agent container starts. On subsequent pod restarts, it can check for existing state and skip re-seeding.

## Compliance Checklist

A well-behaved agent image should:

- [ ] Listen on the port specified by `spec.port` (default `8080`)
- [ ] Read task instructions from `/etc/agent/instructions.txt` on startup (if present)
- [ ] Read runtime configuration from `/etc/agent/config.yaml` on startup (if present)
- [ ] Respect `AGENT_NAME`, `AGENT_NAMESPACE`, `AGENT_UUID`, `AGENT_MODE`, `AGENT_CLUSTER_NAME`, `AGENT_CLUSTER_UUID` environment variables
- [ ] Route LLM traffic through `MODEL_ENDPOINTS` proxy URLs rather than connecting to model APIs directly
- [ ] Use `spec.workspace.mountPath` (default `/workspace`) for persistent state — do not assume local container storage survives restarts
