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

### Self-Configuration

`spec.selfConfigure` controls whether the agent pod may submit [`LanguageAgentSelfConfig`](languageagentselfconfig.md) requests to modify its own spec at runtime. When enabled, the operator grants the agent's ServiceAccount permission to create `LanguageAgentSelfConfig` resources in the same namespace.

```yaml
spec:
  selfConfigure:
    enabled: true
    allowedActions:
      - tools
      - envVars
```

**`spec.selfConfigure` fields (`SelfConfigureSpec`):**

| Field | Type | Default | Description |
|-------|------|---------|-------------|
| `enabled` | *bool | `false` | Gate for all self-configuration. When `false`, any `LanguageAgentSelfConfig` targeting this agent is immediately denied. |
| `allowedActions` | []string | `[]` | Allowlist of self-config categories. When `enabled` is `true` but this list is empty, all actions are denied. Valid values: `tools`, `models`, `envVars`, `instructions`, `roleRules`. |

See [LanguageAgentSelfConfig](languageagentselfconfig.md) for the full self-config request API.

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
| `AGENT_REPO_DIR` | Absolute path to the cloned repository (the agent container's working directory). Only injected when `spec.repository` is set. See [Repository](#repository). |
| `OTEL_EXPORTER_OTLP_ENDPOINT` | Propagated from the operator environment when configured |
| `OTEL_SERVICE_NAME` | Set to `agent-<name>` when `OTEL_EXPORTER_OTLP_ENDPOINT` is configured |
| `OTEL_RESOURCE_ATTRIBUTES` | Propagated from the operator environment (conditional on OTEL endpoint) |
| `OTEL_TRACES_SAMPLER` | Propagated from the operator environment (conditional on OTEL endpoint) |
| `OTEL_TRACES_SAMPLER_ARG` | Propagated from the operator environment (conditional on OTEL endpoint) |

Additional variables from `spec.deployment.env` and `spec.deployment.envFrom` are passed through unchanged. See [Environment Variables](../components/agents.md#environment-variables) in the architecture docs for the full reference.

### Workspace

When `spec.workspace.enabled` is true (the default), the operator provisions a PersistentVolumeClaim and mounts it into the agent container and all init containers. The PVC is normally deleted when the LanguageAgent is deleted; set `retain: true` to preserve it across agent deletions.

**`spec.workspace` fields (`WorkspaceSpec`):**

| Field | Type | Default | Description |
|-------|------|---------|-------------|
| `enabled` | *bool | `true` | Create and mount a workspace PVC. Set to `false` to disable without removing the workspace config. |
| `size` | string | `10Gi` | PVC storage request (e.g. `"10Gi"`, `"500Mi"`) |
| `mountPath` | string | `/workspace` | Mount path in the container |
| `storageClassName` | *string | cluster default | StorageClass for the PVC |
| `accessMode` | string | `ReadWriteOnce` | PVC access mode: `ReadWriteOnce` or `ReadWriteMany` |
| `retain` | *bool | `false` | When `true`, the PVC's ownerReference is removed on agent deletion so Kubernetes GC does not collect it. The orphaned PVC name is surfaced in `status.workspacePVCName`. |
| `initialFiles` | map[string]string | — | Files seeded into the workspace on first boot only. Keys are filenames; values are file contents. Files are not overwritten if they already exist. |
| `seedConfigMapRef` | *LocalObjectReference | — | External ConfigMap whose keys are filenames and values are file contents. Merged with `initialFiles`; `initialFiles` wins on key collision. |

When `retain` is `true` and the agent is deleted, `status.workspacePVCName` records the name of the orphaned PVC so it can be located and reattached later.

### Repository

When `spec.repository` is set, the operator adds a `repository` init container that clones a git repository into the workspace before the agent starts. The agent container's working directory is set to the clone path, and `AGENT_REPO_DIR` is injected into every container so the runtime can locate it.

The clone is **clone-once**: if the target directory already contains a `.git` directory the clone is skipped, so edits and commits made by the agent survive pod restarts. Because the clone lands in the workspace, declaring `spec.repository` automatically defaults `spec.workspace.enabled` to `true`; declaring a repository while `spec.workspace.enabled` is explicitly `false` is rejected by the webhook.

**`spec.repository` fields (`RepositorySpec`):**

| Field | Type | Default | Description |
|-------|------|---------|-------------|
| `url` | string | — | Git repository to clone, either HTTPS (`https://...`) or SSH (`git@host:org/repo.git`). **Required.** |
| `ref` | string | default branch | Branch, tag, or commit SHA to check out. |
| `path` | string | repo name from URL | Subdirectory under the workspace `mountPath` to clone into. Must be relative (no leading `/`, no `..` segments). |
| `depth` | int | `0` (full clone) | When > 0, performs a shallow clone with this history depth. |
| `secretRef` | *LocalObjectReference | — | Secret holding git credentials for private repositories. Recognized keys: `token`, or `username` + `password` (HTTPS); `ssh-privatekey` (SSH). |

The clone target is `<workspace mountPath>/<path>` — e.g. with the default `mountPath: /workspace` and `path: app`, the repository is cloned to `/workspace/app` and `AGENT_REPO_DIR` is set to `/workspace/app`. When `path` is omitted, the directory name is derived from the URL (e.g. `https://github.com/org/repo.git` → `/workspace/repo`).

The operator never reads the credentials Secret; it is mounted read-only into the `repository` init container, which selects SSH or HTTPS auth based on which keys are present.

**Private repository example (HTTPS token):**

```yaml
apiVersion: langop.io/v1alpha1
kind: LanguageAgent
metadata:
  name: code-agent
  namespace: default
spec:
  image: myregistry/agent-runtime:latest
  repository:
    url: https://github.com/myorg/private-repo.git
    ref: main
    path: app
    depth: 1
    secretRef:
      name: git-credentials
---
apiVersion: v1
kind: Secret
metadata:
  name: git-credentials
  namespace: default
type: Opaque
stringData:
  token: ghp_xxxxxxxxxxxxxxxxxxxx   # a personal access token (HTTPS)
  # For SSH instead, provide an ssh-privatekey key and a git@host:... url.
```

### Resource Management

Agents are deployed as standard Kubernetes Deployments with:

- Configurable replicas (`spec.deployment.replicas`)
- Resource limits and requests (`spec.deployment.resources`)
- Node selectors, tolerations, and affinity rules
- Custom liveness, readiness, and startup probes

### Monitoring

`spec.monitoring` integrates the agent with Prometheus Operator. The operator silently skips this if prometheus-operator is not installed.

```yaml
spec:
  monitoring:
    serviceMonitor:
      enabled: true
      path: /metrics
      interval: 30s
    rules:
      - name: agent-alerts
        rules:
          - alert: AgentDown
            expr: up{job="my-agent"} == 0
            for: 5m
            labels:
              severity: critical
```

**`spec.monitoring.serviceMonitor` fields (`AgentServiceMonitorSpec`):**

| Field | Type | Default | Description |
|-------|------|---------|-------------|
| `enabled` | bool | — | Create a ServiceMonitor for this agent. **Required.** |
| `port` | string | first port name, or `"http"` | Name of the service port to scrape |
| `path` | string | `/metrics` | HTTP path to scrape for metrics |
| `interval` | string | Prometheus default | Scrape interval (e.g. `"30s"`) |
| `scrapeTimeout` | string | Prometheus default | Per-scrape timeout |
| `labels` | map[string]string | — | Additional labels added to the ServiceMonitor metadata |

**`spec.monitoring.rules[]` fields (`PrometheusRuleGroup`):**

| Field | Type | Description |
|-------|------|-------------|
| `name` | string | Rule group name. **Required.** |
| `interval` | string | Evaluation interval for this group. Uses Prometheus default when omitted. |
| `rules` | []PrometheusAlertingRule | Alerting or recording rules in this group. At least one required. |

**`spec.monitoring.rules[].rules[]` fields (`PrometheusAlertingRule`):**

| Field | Type | Description |
|-------|------|-------------|
| `alert` | string | Alert name (leave empty for recording rules) |
| `record` | string | Output metric name for recording rules (leave empty for alerting rules) |
| `expr` | string | PromQL expression. **Required.** |
| `for` | string | Duration condition must be true before alert fires (alerting rules only) |
| `labels` | map[string]string | Labels attached to the alert or recording rule |
| `annotations` | map[string]string | Annotations attached to the alert (alerting rules only) |

!!! note "Requires prometheus-operator"
    The operator creates `ServiceMonitor` and `PrometheusRule` resources only when prometheus-operator CRDs are present in the cluster. If prometheus-operator is not installed, `spec.monitoring` is silently ignored.

## Related Resources

- [LanguageModel](languagemodel.md) - Configure LLM access
- [LanguageTool](languagetool.md) - Add tool capabilities
- [LanguagePersona](languagepersona.md) - Define behavioral templates
- [LanguageAgentSelfConfig](languageagentselfconfig.md) - Runtime self-modification requests
- [Agent Runtime Contract](../components/agents.md) - What the operator injects

