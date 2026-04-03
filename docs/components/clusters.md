# Clusters

A `LanguageCluster` is the top-level organizational unit in Language Operator. It creates a managed Kubernetes namespace and deploys the shared infrastructure that all agents, models, and tools in that namespace depend on: a LiteLLM gateway, NetworkPolicies, and optional external ingress.

## How It Works

Creating a `LanguageCluster` triggers the cluster controller to:

1. Create a Kubernetes namespace with the same name as the resource
2. Deploy the shared LiteLLM gateway (`gateway` Deployment + Service)
3. Set up default RBAC for agent pods
4. Create an Ingress at `gateway.<spec.domain>` (when `spec.domain` is set)
5. Watch for `LanguageModel` changes in the namespace and reconcile the gateway config on each change

The cluster is the reconciliation boundary: the controller watches `LanguageModel` CRs in the namespace directly, so adding or removing a model triggers a gateway config update without touching any agent.

## Namespace Mapping

Each `LanguageCluster` maps 1:1 to a namespace. The namespace is created and owned by the cluster resource — deleting the `LanguageCluster` deletes the namespace and everything in it.

```yaml
apiVersion: langop.io/v1alpha1
kind: LanguageCluster
metadata:
  name: production-agents   # → namespace: production-agents
spec: {}
```

All namespace-scoped resources (`LanguageAgent`, `LanguageModel`, `LanguageTool`, `LanguagePersona`) are deployed into this namespace.

## The Shared Gateway

Every cluster runs exactly one LiteLLM proxy. It aggregates all `LanguageModel` resources in the namespace into a single OpenAI-compatible endpoint. Agents connect to it via the `MODEL_ENDPOINTS` environment variable:

```
MODEL_ENDPOINTS=http://gateway.<namespace>.svc.cluster.local:8000
```

Credentials never leave the gateway pod. Agents send model names and prompts; the gateway holds the API keys and routes to the correct upstream provider.

When the model list changes — a `LanguageModel` is added, updated, or deleted — the cluster controller regenerates the `gateway-config` ConfigMap and triggers a rolling restart of the gateway Deployment. No agent redeploy is required.

## External Access

Set `spec.domain` to expose the gateway outside the cluster:

```yaml
spec:
  domain: agents.example.com
```

This creates an Ingress at `gateway.agents.example.com`. TLS is handled by cert-manager using the issuer configured in Helm values (`config.tls.certificateIssuerName`). Per-cluster overrides are available via `spec.ingress`:

```yaml
spec:
  domain: agents.example.com
  ingress:
    className: nginx
    tls:
      issuerRef:
        name: letsencrypt-production
        kind: ClusterIssuer
```

## Network Isolation

By default, agents in a cluster can communicate with each other and with the shared gateway, but not with arbitrary external hosts. Additional ingress and egress rules are configured on the cluster and applied as built-in rules to every agent's `NetworkPolicy`.

Allow HTTPS egress to upstream APIs:

```yaml
spec:
  networkPolicies:
    egress:
      - to:
          - cidr: "0.0.0.0/0"
        ports:
          - port: 443
```

Allow egress by DNS name (requires Cilium):

```yaml
spec:
  networkPolicies:
    egress:
      - to:
          - dns:
              - "api.anthropic.com"
              - "api.openai.com"
        ports:
          - port: 443
```

Restrict ingress to agents in a named group:

```yaml
spec:
  networkPolicies:
    ingress:
      - from:
          - group: data-pipeline
        ports:
          - port: 8080
```

See [LanguageCluster API Reference](../api/languagecluster.md#network-isolation) for the full `NetworkPeer` field reference.

## Capacity Limits

Enforce hard limits on how many resources can be created in the namespace:

```yaml
spec:
  capacity:
    maxAgents: 10
    maxModels: 5
    maxCPU: "8"
    maxMemory: 16Gi
```

The operator creates a `ResourceQuota` named `langop-quota` in the namespace. Current usage is reflected in `status.capacity`.

## Gateway Customization

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
```

All standard Kubernetes deployment fields are supported: `nodeSelector`, `affinity`, `tolerations`, `topologySpreadConstraints`, `env`, `envFrom`, `volumes`, and `volumeMounts`.

## Related

- [LanguageModel](../api/languagemodel.md) — register LLM endpoints with the cluster gateway
- [LanguageAgent](../api/languageagent.md) — deploy agents into the cluster namespace
- [LanguageCluster API Reference](../api/languagecluster.md) — full field documentation
- [Models](models.md) — how the shared gateway and model registration works
