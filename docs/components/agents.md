# Agent Runtime Container Specification

This document defines the contract between the Language Operator and agent runtime containers. It specifies what the operator guarantees to provide to every running agent, and what a compliant agent image must implement.

## Overview

A **LanguageAgent** runs as a standard Kubernetes Deployment. The operator manages the pod lifecycle, injects configuration, and handles networking. The agent container is responsible for its own runtime logic: reading instructions, connecting to tools, and executing tasks.

## What the Operator Provides

### Mounted Files

The operator mounts exactly one file into every agent container:

| Path | Content | Source |
|------|---------|--------|
| `/etc/agent/config.yaml` | Structured agent configuration (YAML) | Assembled by the operator from `spec.instructions`, personas, tools, models, and agent metadata |

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
| `AGENT_CLUSTER_NAME` | Name of the LanguageCluster this agent belongs to |
| `AGENT_CLUSTER_UUID` | Kubernetes UID of the LanguageCluster |
| `MODEL_ENDPOINT` | Single shared LiteLLM gateway URL (`http://gateway.<namespace>.svc.cluster.local:8000`). The same URL is used regardless of how many models are referenced. |
| `LLM_MODEL` | Comma-separated list of model names for all referenced models |
| `MCP_SERVERS` | Comma-separated MCP tool server URLs for all resolved tools — service-mode tools use `http://<name>.<ns>.svc.cluster.local:<port>`; sidecar-mode tools use `http://localhost:<port>`. Only injected when at least one tool is resolved. |
| `AGENT_INSTRUCTIONS` | Content of `spec.instructions`; only set when instructions are non-empty. Identical to the `instructions` field in `/etc/agent/config.yaml`. |
| `OTEL_EXPORTER_OTLP_ENDPOINT` | Propagated from the operator environment when configured; enables agent-side OTEL tracing |
| `OTEL_SERVICE_NAME` | Set to `agent-<name>` when `OTEL_EXPORTER_OTLP_ENDPOINT` is configured |
| `OTEL_RESOURCE_ATTRIBUTES` | Propagated from the operator environment when set (conditional on `OTEL_EXPORTER_OTLP_ENDPOINT`) |
| `OTEL_TRACES_SAMPLER` | Propagated from the operator environment when set (conditional on `OTEL_EXPORTER_OTLP_ENDPOINT`) |
| `OTEL_TRACES_SAMPLER_ARG` | Propagated from the operator environment when set (conditional on `OTEL_EXPORTER_OTLP_ENDPOINT`) |

Additional environment variables from `spec.deployment.env` and `spec.deployment.envFrom` are passed through unchanged.

### Networking

Every agent gets:
- A **ClusterIP Service** for each port in `spec.ports` named `<agent-name>` in the agent's namespace (the port with `expose: true`, or the first port if none, is used for the HTTPRoute)
- An **HTTPRoute** for external access (if a Gateway is configured)
- **NetworkPolicy** permitting inbound traffic from other agent pods in the same cluster namespace

## What the Agent Must Implement

### Ports

The agent listens on the port(s) defined in `spec.ports`. The operator creates a ClusterIP Service with one port entry per `AgentPort`. What the agent serves there is up to the image — HTTP, gRPC, OpenAI-compatible API, or anything else.

Each `AgentPort` has four fields: `name` (required, used as the Service port name), `port` (required, the container port number), `protocol` (optional — `TCP`, `UDP`, or `SCTP`; defaults to `TCP`), and `expose` (optional bool — when `true`, the HTTPRoute targets this port; if no port has `expose: true`, the first port is used).

### Probes

Liveness and readiness probes are configured via `spec.deployment.livenessProbe` and `spec.deployment.readinessProbe`. If not set, no probes are configured. The operator does not require any specific health endpoint — probe configuration is entirely up to the agent author.

### Startup Behaviour

On startup, the agent should:

1. Read `/etc/agent/config.yaml` for all configuration (instructions, personas, tools, models)
2. Start listening on the port(s) defined in `spec.ports`

## File Formats

### Agent Config (`/etc/agent/config.yaml`)

A single YAML document assembled by the operator. Contains everything the agent needs to configure its runtime: persona(s), tool endpoints, and model configuration.

```yaml
# Agent identity
agent:
  name: data-analyst
  namespace: default

# Task instructions from spec.instructions — what this agent should do.
# Omitted if spec.instructions is empty.
instructions: |
  You are a data analyst. Analyze CSV files and generate insights.

# Persona configuration — merged in order if multiple are specified.
# Each entry reflects the tone/personality/expertise fields of the referenced LanguagePersona.
personas:
  - name: analytical-persona
    tone: professional
    personality: "precise, data-driven, always cites sources before drawing conclusions"
    expertise: "data analyst specialising in statistical reasoning and business intelligence"

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
    # priority: 1   # only present when spec.models[].priority is set
```

The `MODEL_ENDPOINT` env var also carries this gateway URL for runtimes that prefer environment-variable-based configuration.

## Example LanguageAgent YAML

```yaml
apiVersion: langop.io/v1alpha1
kind: LanguageAgent
metadata:
  name: data-analyst
  namespace: default
spec:
  image: myregistry/agent-runtime:python-v1.0.0
  ports:
    - name: http
      port: 8080

  instructions: |
    You are a data analyst. Analyze CSV files and generate insights.
    Focus on trends, anomalies, and actionable recommendations.

  persona: analytical-persona

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
  ports:
    - name: http
      port: 18789

  workspace:
    size: 10Gi
    mountPath: /workspace

  deployment:
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
```

The init container runs to completion before the agent container starts. On subsequent pod restarts, it can check for existing state and skip re-seeding.

## Compliance Checklist

A well-behaved agent image should:

- [ ] Listen on the port(s) defined in `spec.ports` (default: one port named `http` on `8080`)
- [ ] Read runtime configuration from `/etc/agent/config.yaml` on startup (if present); task instructions are in the top-level `instructions` field
- [ ] Respect `AGENT_NAME`, `AGENT_NAMESPACE`, `AGENT_UUID`, `AGENT_MODE`, `AGENT_CLUSTER_NAME`, `AGENT_CLUSTER_UUID` environment variables
- [ ] Route LLM traffic through `MODEL_ENDPOINT` proxy URLs rather than connecting to model APIs directly
- [ ] Use `spec.workspace.mountPath` (default `/workspace`) for persistent state — do not assume local container storage survives restarts
