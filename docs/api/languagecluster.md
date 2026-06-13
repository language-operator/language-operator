# LanguageCluster

The `LanguageCluster` CRD creates a managed namespace for AI agent deployments with shared infrastructure.

## Overview

A LanguageCluster provides:
- Dedicated namespace with network isolation
- Shared LiteLLM proxy for all models in the cluster
- Optional external ingress at `gateway.<domain>`
- NetworkPolicy enforcement for security

## Quick Example

```yaml
apiVersion: langop.io/v1alpha1
kind: LanguageCluster
metadata:
  name: production-agents
spec:
  domain: agents.example.com
```

This creates a `production-agents` namespace with the shared gateway accessible at `http://gateway.production-agents.svc.cluster.local:8000`.

## Complete API Reference

See the [Complete API Reference](reference.md#languagecluster) for full field documentation including:

- **LanguageCluster** - Top-level resource
- **LanguageClusterSpec** - Specification fields
- **LanguageClusterStatus** - Status and proxy information

## Key Concepts

### Shared Proxy Architecture

Each cluster has exactly one LiteLLM proxy that:

- Aggregates all `LanguageModel` resources in the namespace
- Provides unified endpoint for all agents
- Handles credential management centrally
- Enables cross-model cost tracking

All agents in the cluster connect to this shared proxy via the `MODEL_ENDPOINT` environment variable.

### Network Isolation

Network isolation for agents in this cluster is configured via `spec.networkPolicies`, an object with `ingress` and `egress` rule lists. Rules mirror the native Kubernetes NetworkPolicy shape — see `AgentNetworkPolicies` in the API reference. By default:

- Agents can communicate with each other on port 8080
- Agents can reach the shared proxy
- Agents can reach tools in the same namespace
- External ingress is controlled via the domain setting

Example — allow HTTPS egress from all agents in the cluster:

```yaml
spec:
  networkPolicies:
    egress:
      - to:
          - cidr: "0.0.0.0/0"
        ports:
          - port: 443
```

#### NetworkPeer fields

Each entry in an `ingress[].from` or `egress[].to` list is a `NetworkPeer`. All fields are optional and can be combined:

| Field | Type | Description |
|-------|------|-------------|
| `group` | string | Selects pods with the matching `langop.io/group` label value |
| `cidr` | string | CIDR block (e.g. `"10.0.0.0/8"`, `"0.0.0.0/0"`) |
| `dns` | []string | DNS names; supports `*` wildcards (e.g. `"*.openai.com"`). Requires a CNI that supports FQDN-based egress (e.g. Cilium). |
| `service` | object | Kubernetes Service reference (`name`, optional `namespace`) |
| `namespaceSelector` | LabelSelector | Selects namespaces for cross-namespace rules |
| `podSelector` | LabelSelector | Selects pods within the (current or selected) namespace |

**`dns` example** — allow egress to an external API without maintaining CIDR lists:

```yaml
spec:
  networkPolicies:
    egress:
      - to:
          - dns:
              - "api.openai.com"
              - "*.googleapis.com"
        ports:
          - port: 443
```

**`group` example** — allow agents in group `"data-pipeline"` to reach each other:

```yaml
spec:
  networkPolicies:
    ingress:
      - from:
          - group: data-pipeline
        ports:
          - port: 8080
```

### External Access

By default the gateway is **in-cluster only** — agents reach it via its Service at
`gateway.<namespace>.svc.cluster.local:8000` and no external Ingress is created. To expose the
proxy externally, set `spec.domain` **and** opt in with `spec.ingress.enabled: true`:

```yaml
spec:
  domain: agents.example.com
  ingress:
    enabled: true
```

Creates an Ingress/HTTPRoute at `gateway.agents.example.com` for external model access.

!!! note "Behavior change"
    Earlier versions created this Ingress automatically whenever `spec.domain` was set. The gateway
    is now opt-in for external exposure; set `spec.ingress.enabled: true` to restore an external
    Ingress on upgrade.

### Gateway Deployment Configuration

Use `spec.gateway.deployment` to customize the shared LiteLLM gateway pod. All fields are optional.

```yaml
spec:
  gateway:
    deployment:
      replicas: 2
      resources:
        requests:
          cpu: 100m
          memory: 256Mi
        limits:
          cpu: 500m
          memory: 512Mi
      nodeSelector:
        kubernetes.io/arch: amd64
```

**`spec.gateway.deployment` fields:**

| Field | Type | Description |
|-------|------|-------------|
| `replicas` | integer | Number of gateway pod replicas (default: 1) |
| `imagePullPolicy` | string | `Always`, `Never`, or `IfNotPresent` |
| `imagePullSecrets` | []LocalObjectReference | Image pull secrets |
| `env` | []EnvVar | Extra environment variables for the gateway container |
| `envFrom` | []EnvFromSource | Environment variables sourced from ConfigMap or Secret |
| `resources` | ResourceRequirements | CPU/memory requests and limits |
| `nodeSelector` | map[string]string | Node selector labels |
| `affinity` | Affinity | Pod affinity and anti-affinity rules |
| `tolerations` | []Toleration | Tolerations for tainted nodes |
| `topologySpreadConstraints` | []TopologySpreadConstraint | Pod topology spread |
| `serviceAccountName` | string | Service account for the gateway pod |
| `securityContext` | PodSecurityContext | Pod-level security attributes |
| `volumeMounts` | []VolumeMount | Extra volume mounts |
| `volumes` | []Volume | Extra volumes |

### Ingress Configuration

Use `spec.ingress` to control how the gateway is exposed externally. This requires `spec.domain` to be set.

```yaml
spec:
  domain: agents.example.com
  ingress:
    enabled: true
    className: nginx
    tls:
      enabled: true
```

**`spec.ingress` fields:**

| Field | Type | Description |
|-------|------|-------------|
| `enabled` | *bool | Create an external Ingress for the gateway (default: false — gateway is in-cluster only; requires `spec.domain` when set to true) |
| `className` | string | IngressClass name — per-cluster override of `config.gateway.ingressClassName` |
| `tls` | *IngressTLSConfig | TLS configuration (see below) |

**`spec.ingress.tls` fields:**

| Field | Type | Description |
|-------|------|-------------|
| `enabled` | *bool | Enable TLS on the Ingress (default: true) |
| `secretName` | string | Name of an existing TLS Secret. When set, the operator uses this secret directly and skips cert-manager integration. |

cert-manager issuer selection is operator-wide, not per-cluster. Configure it via the operator Helm chart's `config.tls.certificateIssuerName` and `config.tls.certificateIssuerKind` values.

**Example — manual TLS secret:**

```yaml
spec:
  domain: agents.example.com
  ingress:
    tls:
      secretName: gateway-tls
```

**Example — disable TLS:**

```yaml
spec:
  domain: agents.example.com
  ingress:
    tls:
      enabled: false
```

### Capacity and Quotas

Use `spec.capacity` to enforce hard resource limits on the cluster's namespace. When set, the operator creates a `ResourceQuota` named `langop-quota` in the namespace. When removed, the quota is deleted.

```yaml
spec:
  capacity:
    maxAgents: 10
    maxModels: 5
    maxTools: 20
    maxPersonas: 20
    maxCPU: "8"
    maxMemory: 16Gi
```

**`spec.capacity` fields:**

| Field | Type | Description |
|-------|------|-------------|
| `maxAgents` | integer | Maximum number of `LanguageAgent` objects |
| `maxModels` | integer | Maximum number of `LanguageModel` objects |
| `maxTools` | integer | Maximum number of `LanguageTool` objects |
| `maxPersonas` | integer | Maximum number of `LanguagePersona` objects |
| `maxCPU` | quantity | Aggregate `limits.cpu` across all pods (e.g. `"8"`, `"2500m"`) |
| `maxMemory` | quantity | Aggregate `limits.memory` across all pods (e.g. `"16Gi"`, `"512Mi"`) |

All fields are optional. Omit a field to leave that dimension unrestricted.

**`status.capacity` fields** report observed usage:

| Field | Description |
|-------|-------------|
| `agentCount` | Current number of `LanguageAgent` objects |
| `modelCount` | Current number of `LanguageModel` objects |
| `toolCount` | Current number of `LanguageTool` objects |
| `personaCount` | Current number of `LanguagePersona` objects |
| `totalCPULimits` | Sum of `limits.cpu` across all agent pod specs |
| `totalMemoryLimits` | Sum of `limits.memory` across all agent pod specs |

## Related Resources

- [LanguageAgent](languageagent.md) - Deploy agents in the cluster
- [LanguageModel](languagemodel.md) - Register models with the proxy
- [Clusters](../components/clusters.md) - How LanguageCluster works

