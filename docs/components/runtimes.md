# Runtimes

A `LanguageAgentRuntime` is a cluster-scoped preset that packages the defaults for a specific agent type: container image, ports, workspace configuration, resource limits, probes, and init containers. It is analogous to a Kubernetes `StorageClass` — admins install runtimes once, users reference them by name.

```yaml
spec:
  runtime: openclaw
```

## How It Works

At reconcile time, the operator merges the runtime's defaults into the agent's effective spec before creating any Kubernetes resources. The agent's own fields always take precedence for scalar values; lists are runtime-first, then agent-appended.

This means a minimal agent spec like:

```yaml
apiVersion: langop.io/v1alpha1
kind: LanguageAgent
metadata:
  name: my-agent
spec:
  runtime: openclaw
  models:
    - name: claude-sonnet
```

produces the same Deployment as if the agent had explicitly specified the OpenClaw image, port, workspace settings, init containers, and probes — because the runtime supplies those defaults.

## Bundled Runtimes

Three runtimes are shipped via the `language-operator-runtimes` Helm chart, installed independently after the operator:

| Name | Image | Port | Interface |
|------|-------|------|-----------|
| `openclaw` | `ghcr.io/openclaw/openclaw:latest` | 18789 | WebSocket gateway |
| `opencode` | `ghcr.io/anomalyco/opencode:latest` | 3000 | HTTP / browser UI |
| `claude-code` | `ghcr.io/language-operator/claude-code-adapter:latest` | 8080 | HTTP / WebSocket terminal |

Install them with:

```bash
helm install language-operator-runtimes \
  language-operator/language-operator-runtimes \
  --namespace language-operator
```

Any runtime can be disabled at install time via its `values.yaml` key:

```yaml
runtimes:
  claudeCode:
    enabled: false
```

See [Installation](../getting-started/installation.md#3-install-the-runtimes) for the full install sequence.

## Merge Semantics

| Field type | Behaviour |
|------------|-----------|
| Scalars (`image`, `resources`, probes) | Runtime provides the default; agent overrides if set |
| `ports` | Replace semantics — runtime ports apply only when the agent specifies no ports |
| Lists (`env`, `envFrom`, `volumes`, `volumeMounts`, `initContainers`) | Runtime entries prepended; agent entries appended |

**Example:** An OpenClaw runtime defines an init container that adapts `/etc/agent/config.yaml` into OpenClaw's native format. An agent using `runtime: openclaw` can add its own init containers; they run after the runtime's adapter.

## Credentials

Runtimes provision credentials for every agent that uses them through the generic `credentials` list. Each entry's `name` is both the environment variable name and the key in the operator-managed Secret. Entries are resolved in priority order:

- **`valueFrom` set** — the referenced Secret's keys are injected via `envFrom`; the operator creates no Secret of its own.
- **`value` set** — the literal is stored in an operator-managed Secret named `{agent-name}-runtime` and injected via `envFrom`.
- **neither set** — the operator auto-generates a random value once, persists it in the `{agent-name}-runtime` Secret, and preserves it across reconciles (it is never rotated).

The bundled runtimes declare the credentials their images need:

- `openclaw` declares `OPENCLAW_GATEWAY_TOKEN` (auto-generated).
- `opencode` declares `OPENCODE_SERVER_PASSWORD` (auto-generated) and sets `OPENCODE_SERVER_USERNAME=opencode` as a plain `deployment.env` variable.
- `claude-code` declares no credentials — authentication is interactive via `/login`.

Runtime-declared entries merge first, then any entries the agent adds in its own `spec.credentials`; entries are deduplicated by `name`, with the agent's entry winning on a collision.

## Authentication

Authentication is a runtime trait, not an agent setting. A runtime's `spec.auth.enabled: true` gates whether agents using it sit behind the cluster's OIDC proxy. An agent is proxied **only when both** the cluster has `auth.enabled: true` **and** its runtime has `auth.enabled: true`. The three bundled runtimes all enable auth since they serve web UIs. See [Clusters](clusters.md#authentication) for the cluster-side configuration.

## Custom Runtimes

Any container image can be packaged as a runtime:

```yaml
apiVersion: langop.io/v1alpha1
kind: LanguageAgentRuntime
metadata:
  name: my-python-agent
spec:
  image: ghcr.io/my-org/python-agent:v1.0.0
  ports:
    - name: http
      port: 8080
  workspace:
    size: 5Gi
    mountPath: /workspace
  deployment:
    resources:
      requests:
        memory: 512Mi
        cpu: 200m
      limits:
        memory: 2Gi
        cpu: 1000m
    livenessProbe:
      httpGet:
        path: /health
        port: 8080
      initialDelaySeconds: 15
      periodSeconds: 30
    initContainers:
      - name: config-adapter
        image: ghcr.io/my-org/config-adapter:v1.0.0
```

`LanguageAgentRuntime` is cluster-scoped — one runtime can be referenced by agents in any namespace.

## Related

- [LanguageAgent](../api/languageagent.md) — references runtimes via `spec.runtime`
- [LanguageAgentRuntime API Reference](../api/languageagentruntime.md) — full field documentation
- [OpenClaw Guide](../guides/openclaw.md)
- [OpenCode Guide](../guides/opencode.md)
