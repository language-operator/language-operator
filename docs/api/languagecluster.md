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

All agents in the cluster connect to this shared proxy via the `MODEL_ENDPOINTS` environment variable.

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

### External Access

Configure `spec.domain` to expose the proxy externally:

```yaml
spec:
  domain: agents.example.com
```

Creates an Ingress/HTTPRoute at `gateway.agents.example.com` for external model access.

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
- [Architecture Overview](../architecture/overview.md) - System design

## Examples

See [Examples](../getting-started/examples.md) for multi-cluster patterns.
