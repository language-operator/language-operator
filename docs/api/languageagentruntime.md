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
| `openclaw` | `ghcr.io/openclaw/openclaw:latest` | 18789 | AI personal assistant (WebSocket gateway) |
| `opencode` | `ghcr.io/anomalyco/opencode:latest` | 3000 | AI coding assistant (HTTP/browser UI) |
| `claude-code` | `ghcr.io/language-operator/claude-code-adapter:latest` | 8080 | AI coding assistant (HTTP/WebSocket terminal) |

Disable a bundled runtime in `values.yaml`:

```yaml
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
| `deployment` | `DeploymentSpec` | Default pod and container settings (resources, probes, initContainers, env, …). The name is historical — agents run as Argo Workflow pods, so `replicas` and `autoscaling` have no effect and are rejected on the agent |
| `execution` | `ExecutionSpec` | Default execution model for agents using this runtime — `mode`, `schedule`, `timezone`, `concurrencyPolicy`, `activeDeadlineSeconds`, `ttlSecondsAfterFinished`, `retryLimit`, `suspend`. See [Execution Modes](../guides/execution-modes.md) |
| `credentials` | `[]Credential` | Credentials provisioned for every agent referencing this runtime; see below |
| `auth` | `RuntimeAuth` | Gates whether agents using this runtime sit behind the cluster's OIDC proxy; see [Authentication](#authentication) |

### Credentials

A runtime declares the credentials its image needs through the generic `credentials` list. Each entry's `name` is both the environment variable name and the key in the operator-managed Secret. Entries are resolved in priority order:

- **`valueFrom` set** — the referenced Secret's keys are injected via `envFrom`; the operator creates no Secret of its own.
- **`value` set** — the literal is stored in an operator-managed Secret named `{agent}-runtime` and injected via `envFrom`.
- **neither set** — the operator auto-generates a random value once, persists it in the `{agent}-runtime` Secret, and preserves it across reconciles (it is never rotated).

Runtime-declared entries are merged ahead of any entries the agent adds in its own `spec.credentials`; entries are deduplicated by `name`, with the agent's entry winning on a collision. This is how the bundled runtimes provision credentials without requiring each `LanguageAgent` to configure them manually:

- `openclaw` declares `OPENCLAW_GATEWAY_TOKEN` (auto-generated).
- `opencode` declares no credentials — access is gated by the cluster OIDC proxy (`auth.enabled: true`).
- `claude-code` declares no credentials — its authentication is interactive via `/login`.

Custom runtimes use the same mechanism:

```yaml
apiVersion: langop.io/v1alpha1
kind: LanguageAgentRuntime
metadata:
  name: my-runtime
spec:
  image: ghcr.io/my-org/my-agent:latest
  credentials:
    - name: MY_RUNTIME_TOKEN   # auto-generated for every agent using this runtime
  ports:
    - name: http
      port: 8080
```

### Authentication

`spec.auth.enabled` (bool) gates whether agents using this runtime are placed behind the cluster's OIDC proxy. An agent is proxied **only when both** the cluster has `auth.enabled: true` **and** its runtime has `auth.enabled: true`. An agent with no runtime, or whose runtime does not enable auth, is never proxied.

The three bundled runtimes (`openclaw`, `opencode`, `claude-code`) all set `auth.enabled: true`, since they serve web UIs. The cluster-wide switch and OIDC connection config live on the `LanguageCluster` — see [Clusters](../components/clusters.md#authentication).

```yaml
apiVersion: langop.io/v1alpha1
kind: LanguageAgentRuntime
metadata:
  name: my-runtime
spec:
  image: ghcr.io/my-org/my-agent:latest
  auth:
    enabled: true   # agents using this runtime sit behind the cluster OIDC proxy
  ports:
    - name: http
      port: 8080
```

## Merge Semantics

| Field type | Behaviour |
|------------|-----------|
| Scalars (`image`, `resources`, probes, `execution.*`) | Runtime provides default; agent overrides if set |
| `ports` | **Replace semantics** — runtime ports apply only when the agent defines no ports of its own |
| `credentials` | Runtime entries merged first, then agent entries appended; deduplicated by `name`, agent wins on collision |
| Other lists (`env`, `envFrom`, `volumes`, `volumeMounts`, `initContainers`) | Runtime entries prepended; agent entries appended |
| `deployment.replicas`, `deployment.autoscaling` | **Not merged.** Agents run as Argo Workflows, which have neither; the agent webhook rejects both fields |

A runtime whose image only makes sense as a one-shot job can default the mode for every agent
that uses it:

```yaml
spec:
  execution:
    mode: task
```

An agent may still override it.

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
