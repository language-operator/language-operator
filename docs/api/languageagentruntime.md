# LanguageAgentRuntime

A `LanguageAgentRuntime` is a cluster-scoped preset that packages the image, port, init containers, probes, and env vars for a specific agent type. It is analogous to a Kubernetes `StorageClass` or `IngressClass` — admins install runtimes once, users reference them by name.

## Overview

Instead of specifying every container detail in each `LanguageAgent`, reference a runtime:

```yaml
spec:
  runtime: openclaw
```

The operator merges the runtime's defaults into the agent's effective spec at reconcile time. Agent fields always win over runtime defaults for scalar values; lists (env, volumes, init containers) are runtime-first, then agent-appended.

## Bundled Runtimes

The standard runtimes are installed automatically with the Helm chart:

| Name | Image | Port | Use case |
|------|-------|------|----------|
| `openclaw` | `ghcr.io/openclaw/openclaw:latest` | 18789 | AI coding assistant (WebSocket gateway) |
| `opencode` | `ghcr.io/anomalyco/opencode:latest` | 3000 | AI coding assistant (HTTP/browser UI) |

Disable a bundled runtime in `values.yaml`:

```yaml
runtimes:
  opencode:
    enabled: false
```

## Custom Runtimes

Create your own runtime for any agent image:

```yaml
apiVersion: langop.io/v1alpha1
kind: LanguageAgentRuntime
metadata:
  name: my-runtime
spec:
  image: ghcr.io/my-org/my-agent:latest
  ports:
    - name: http
      port: 8080
  workspace:
    size: 5Gi
    mountPath: /workspace
  deployment:
    resources:
      requests:
        memory: 256Mi
        cpu: 100m
      limits:
        memory: 1Gi
        cpu: 500m
```

## Spec Fields

| Field | Type | Description |
|-------|------|-------------|
| `image` | string | Default container image for agents using this runtime |
| `ports` | `[]AgentPort` | Default port list; see merge semantics below |
| `workspace` | `WorkspaceSpec` | Default workspace configuration (size, mountPath) |
| `deployment` | `DeploymentSpec` | Default deployment settings (resources, probes, initContainers, env, …) |
| `openclaw` | `OpenclawConfig` | When `openclaw.enabled: true`, the operator auto-generates `OPENCLAW_GATEWAY_TOKEN` for every agent referencing this runtime (without the agent needing `spec.openclaw`) |
| `opencode` | `OpencodeConfig` | When `opencode.enabled: true`, the operator auto-generates `OPENCODE_SERVER_PASSWORD` for every agent referencing this runtime |

### Automatic Credential Generation

Setting `spec.openclaw.enabled: true` or `spec.opencode.enabled: true` on a runtime causes the operator to create a `{agent-name}-runtime` Secret and inject the generated credential env var into every agent that references the runtime. This is how the bundled runtimes enable auto-credential management without requiring each `LanguageAgent` to configure credentials manually.

Custom runtimes can use the same mechanism:

```yaml
apiVersion: langop.io/v1alpha1
kind: LanguageAgentRuntime
metadata:
  name: my-runtime
spec:
  image: ghcr.io/my-org/my-agent:latest
  openclaw:
    enabled: true   # auto-generates OPENCLAW_GATEWAY_TOKEN for all agents using this runtime
  ports:
    - name: http
      port: 8080
```

## Merge Semantics

| Field type | Behaviour |
|------------|-----------|
| Scalars (`image`, `resources`, probes) | Runtime provides default; agent overrides if set |
| `ports` | **Replace semantics** — runtime ports apply only when the agent defines no ports of its own |
| Other lists (`env`, `envFrom`, `volumes`, `volumeMounts`, `initContainers`) | Runtime entries prepended; agent entries appended |

## Status

`LanguageAgentRuntime` has no status subresource. It is a static configuration object — the operator reads it at reconcile time but does not track its health.

```bash
kubectl get languageagentruntimes
# NAME       AGE
# openclaw   5m
# opencode   5m
```

## Related

- [LanguageAgent](languageagent.md) — references runtimes via `spec.runtime`
