# Agents

A `LanguageAgent` is a container image running as a Kubernetes Deployment. The operator handles everything around it — lifecycle, configuration, networking, persistent storage — so the image itself can focus on doing work.

This page covers what the operator injects into every agent pod, what the agent is expected to do with it, and how to wire everything together.

## Configuration

Every agent pod gets one file mounted at `/etc/agent/config.yaml`. The operator assembles it from the agent's `spec.instructions`, referenced personas, resolved tool endpoints, and model configuration. It's reconciled on every change to the LanguageAgent or any resource it references.

```yaml
# Agent identity
agent:
  name: data-analyst
  namespace: default

# From spec.instructions — omitted when empty
instructions: |
  You are a data analyst. Analyze CSV files and generate insights.

# Resolved from referenced LanguagePersona CRDs
personas:
  - name: analytical-persona
    tone: professional
    personality: "precise, data-driven, always cites sources before drawing conclusions"
    expertise: "data analyst specialising in statistical reasoning and business intelligence"

# Resolved tool endpoints, keyed by tool name
tools:
  mem0-memory:
    endpoint: http://mem0-memory.tools.svc.cluster.local:8080/mcp
    protocol: mcp
  python-executor:
    endpoint: http://python-executor.tools.svc.cluster.local:8080/mcp
    protocol: mcp

# Model configuration — all LLM traffic routes through the shared gateway
models:
  claude-sonnet:
    role: primary
    provider: anthropic
    model: claude-sonnet-4-5
    endpoint: http://gateway.default.svc.cluster.local:8000
```

The file is read-only. Agents should read it on startup to configure their runtime.

## Environment Variables

The operator injects these into the agent container and all init containers:

| Variable | Value |
|----------|-------|
| `AGENT_NAME` | `metadata.name` of the LanguageAgent |
| `AGENT_NAMESPACE` | `metadata.namespace` of the LanguageAgent |
| `AGENT_UUID` | Stable UUID assigned to this agent |
| `AGENT_CLUSTER_NAME` | Name of the LanguageCluster this agent belongs to |
| `AGENT_CLUSTER_UUID` | Kubernetes UID of the LanguageCluster |
| `MODEL_ENDPOINT` | Shared LiteLLM gateway URL — the same URL regardless of how many models are referenced |
| `LLM_MODEL` | Comma-separated list of model names for all referenced models |
| `MCP_SERVERS` | Comma-separated tool endpoint URLs — service-mode tools use in-cluster DNS; sidecar-mode tools use `http://localhost:<port>/mcp` |
| `AGENT_INSTRUCTIONS` | Content of `spec.instructions`; only set when non-empty |
| `AGENT_REPO_DIR` | Absolute path to the cloned repository; also the agent's working directory. Only injected when `spec.repository` is set |
| `OTEL_EXPORTER_OTLP_ENDPOINT` | Propagated from the operator when configured |
| `OTEL_SERVICE_NAME` | Set to `agent-<name>` when OTEL is configured |

Additional variables from `spec.deployment.env` and `spec.deployment.envFrom` are passed through unchanged.

## Workspace

When `spec.workspace.enabled` is true (the default), the operator provisions a PersistentVolumeClaim named `<agent-name>-workspace` and mounts it into the agent container and all init containers. The workspace survives pod restarts and redeployments. By default the PVC is deleted when the LanguageAgent is deleted; set `spec.workspace.retain: true` to preserve it.

| Field | Default | Description |
|-------|---------|-------------|
| `spec.workspace.enabled` | `true` | Create and mount a workspace PVC |
| `spec.workspace.size` | `10Gi` | PVC storage request |
| `spec.workspace.mountPath` | `/workspace` | Mount path in the container |
| `spec.workspace.storageClassName` | cluster default | StorageClass for the PVC |
| `spec.workspace.accessMode` | `ReadWriteOnce` | PVC access mode (`ReadWriteOnce` or `ReadWriteMany`) |
| `spec.workspace.retain` | `false` | When `true`, the PVC is preserved after agent deletion; the orphaned PVC name is recorded in `status.workspacePVCName` |
| `spec.workspace.initialFiles` | — | Files seeded into the workspace on first boot (keys = filenames, values = file contents; not overwritten if already present) |
| `spec.workspace.seedConfigMapRef` | — | External ConfigMap whose keys/values are seeded as files; merged with `initialFiles` (`initialFiles` wins on collision) |

The volume is named `workspace` in the pod spec. Init containers that need to pre-seed it should mount it by that name.

## Repository

When `spec.repository` is set, the operator adds a `repository` init container (image `alpine/git:latest`) that runs after the workspace seeder and clones a git repository into the workspace. The agent container's working directory is set to the clone path, and `AGENT_REPO_DIR` is injected into every container.

The clone is **clone-once**: it is skipped when the target directory already contains a `.git` directory, so the agent's edits and commits survive pod restarts. Declaring `spec.repository` defaults `spec.workspace.enabled` to `true`, since the clone needs the workspace PVC to land in.

| Field | Default | Description |
|-------|---------|-------------|
| `spec.repository.url` | — | Git repository to clone — HTTPS (`https://...`) or SSH (`git@host:org/repo.git`). Required |
| `spec.repository.ref` | default branch | Branch, tag, or commit SHA to check out |
| `spec.repository.path` | repo name from URL | Subdirectory under the workspace `mountPath` to clone into (relative path only) |
| `spec.repository.depth` | `0` | When > 0, shallow-clone to this history depth |
| `spec.repository.secretRef` | — | Secret with git credentials for private repos. Keys: `token` or `username`+`password` (HTTPS), `ssh-privatekey` (SSH) |

When `secretRef` is set, the Secret is mounted read-only into the `repository` init container at `/git-secret`; the operator never reads its contents. The init container picks SSH or HTTPS authentication based on which keys are present. See the [LanguageAgent API reference](../api/languageagent.md#repository) for a full private-repo example.

## Networking

The operator creates:

- A **ClusterIP Service** for each port in `spec.ports`, named after the agent
- An **HTTPRoute** for external access when a Gateway is configured on the LanguageCluster
- A **NetworkPolicy** allowing inbound traffic from other agents in the same cluster namespace

The agent listens on the port(s) defined in `spec.ports`. What it serves there is up to the image — HTTP, WebSocket, OpenAI-compatible API, anything.

## Init Containers

For runtimes that need configuration in a format other than `config.yaml`, use an init container to translate before the agent starts. The operator automatically mounts `/etc/agent/config.yaml` into every init container, so adapters can read the operator config and write whatever format the runtime expects.

```yaml
spec:
  image: ghcr.io/myorg/my-agent:latest
  ports:
    - name: http
      port: 18789

  workspace:
    size: 10Gi

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
```

The init container runs to completion before the agent container starts. On subsequent restarts it can check for existing state and skip re-seeding user data while still applying any operator-managed config changes.

## Example

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
    mountPath: /workspace

  deployment:
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

## Checklist

A well-behaved agent image should:

- [ ] Listen on the port(s) defined in `spec.ports` (default: one port named `http` on `8080`)
- [ ] Read `/etc/agent/config.yaml` on startup for instructions, personas, tools, and models
- [ ] Respect the `AGENT_*` identity env vars
- [ ] Route all LLM traffic through `MODEL_ENDPOINT` — agents never hold real API credentials
- [ ] Use `spec.workspace.mountPath` (default `/workspace`) for persistent state
