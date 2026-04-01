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

## Merge Semantics

| Field type | Behaviour |
|------------|-----------|
| Scalars (`image`, `port`, `resources`, probes) | Runtime provides default; agent overrides if set |
| Lists (`env`, `envFrom`, `volumes`, `volumeMounts`, `initContainers`) | Runtime entries prepended; agent entries appended |

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
