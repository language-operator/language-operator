# LanguageCluster

The `LanguageCluster` CRD creates a managed namespace for AI agent deployments with shared infrastructure.

## Overview

A LanguageCluster provides:
- Dedicated namespace with network isolation
- Shared LiteLLM proxy for all models in the cluster
- Optional external ingress at `proxy.<domain>`
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

NetworkPolicy rules are defined via `spec.networkPolicies` (a list of `NetworkRule` objects). By default:

- Agents can communicate with each other on port 8080
- Agents can reach the shared proxy
- Agents can reach tools in the same namespace
- External ingress is controlled via the domain setting

### External Access

Configure `spec.domain` to expose the proxy externally:

```yaml
spec:
  domain: agents.example.com
```

Creates an Ingress/HTTPRoute at `proxy.agents.example.com` for external model access.

## Related Resources

- [LanguageAgent](languageagent.md) - Deploy agents in the cluster
- [LanguageModel](languagemodel.md) - Register models with the proxy
- [Architecture Overview](../architecture/overview.md) - System design

## Examples

See [Examples](../getting-started/examples.md) for multi-cluster patterns.
