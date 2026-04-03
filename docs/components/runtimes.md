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

The Helm chart installs two runtimes automatically:

| Name | Image | Port | Interface |
|------|-------|------|-----------|
| `openclaw` | `ghcr.io/openclaw/openclaw:latest` | 18789 | WebSocket gateway |
| `opencode` | `ghcr.io/anomalyco/opencode:latest` | 3000 | HTTP / browser UI |

Both can be disabled in `values.yaml`:

```yaml
runtimes:
  opencode:
    enabled: false
```

## Merge Semantics

| Field type | Behaviour |
|------------|-----------|
| Scalars (`image`, `resources`, probes) | Runtime provides the default; agent overrides if set |
| `ports` | Replace semantics — runtime ports apply only when the agent specifies no ports |
| Lists (`env`, `envFrom`, `volumes`, `volumeMounts`, `initContainers`) | Runtime entries prepended; agent entries appended |

**Example:** An OpenClaw runtime defines an init container that adapts `/etc/agent/config.yaml` into OpenClaw's native format. An agent using `runtime: openclaw` can add its own init containers; they run after the runtime's adapter.

## Auto-Generated Credentials

Runtimes can instruct the operator to auto-generate credentials for every agent that uses them. Setting `spec.openclaw.enabled: true` or `spec.opencode.enabled: true` on a runtime causes the operator to create a `{agent-name}-runtime` Secret containing the generated credential and inject it into the agent via `envFrom` — no manual `kubectl create secret` needed.

The bundled runtimes use this mechanism. OpenClaw generates `OPENCLAW_GATEWAY_TOKEN`; OpenCode generates `OPENCODE_SERVER_PASSWORD`.

An agent can still override with its own `spec.openclaw.token` or `spec.openclaw.tokenRef` — the per-agent value takes precedence.

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

## Status

`LanguageAgentRuntime` has no status subresource. It is a static configuration object; the operator reads it at reconcile time but does not track its health or readiness.

```bash
kubectl get languageagentruntimes
# NAME       AGE
# openclaw   5m
# opencode   5m
```

## Related

- [LanguageAgent](../api/languageagent.md) — references runtimes via `spec.runtime`
- [LanguageAgentRuntime API Reference](../api/languageagentruntime.md) — full field documentation
- [OpenClaw Guide](../guides/openclaw.md)
- [OpenCode Guide](../guides/opencode.md)
