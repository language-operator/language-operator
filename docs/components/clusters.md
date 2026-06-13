# Clusters

A `LanguageCluster` is the top-level organizational unit in Language Operator. It creates a managed Kubernetes namespace and deploys the shared infrastructure that all agents, models, and tools in that namespace depend on: a LiteLLM gateway, NetworkPolicies, and optional external ingress.

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

## Shared Gateway

Every cluster runs exactly one LiteLLM proxy. It aggregates all `LanguageModel` resources in the namespace into a single OpenAI-compatible endpoint. Agents connect to it via the `MODEL_ENDPOINT` environment variable:

```
MODEL_ENDPOINT=http://gateway.<namespace>.svc.cluster.local:8000
```

Credentials never leave the gateway pod. Agents send model names and prompts; the gateway holds the API keys and routes to the correct upstream provider.

When the model list changes, the gateway restarts with the updated configuration. No agent redeploy is required.

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

## Authentication

`LanguageCluster.spec.auth` is the cluster-wide authentication switch. Setting `auth.enabled: true` provisions OIDC authentication for the namespace. There are two mutually exclusive paths under `auth.oidc`:

**Embedded Dex** — the operator deploys a Dex instance alongside the gateway. Connectors go under `oidc.dex.connectors`:

```yaml
spec:
  auth:
    enabled: true
    oidc:
      emailDomain: example.com
      dex:
        connectors:
          - type: github
            id: github
            name: GitHub
            config:
              clientID: my-github-app-client-id
              clientSecret: my-github-app-client-secret
```

**External OIDC provider** — skip Dex and point at an existing issuer directly. Use `oidc.externalIssuerURL` together with `clientID` and `clientSecretRef`:

```yaml
spec:
  auth:
    enabled: true
    oidc:
      externalIssuerURL: https://accounts.google.com
      clientID: language-operator
      clientSecretRef:
        name: oidc-client-secret
      emailDomain: example.com
```

Enabling auth on the cluster does not, by itself, put any agent behind the proxy. An agent is placed behind the OIDC proxy **only when both** the cluster has `auth.enabled: true` **and** the agent's runtime has `auth.enabled: true`. An agent with no runtime, or whose runtime does not enable auth, is never proxied — there is no per-agent auth override.

The three bundled runtimes (`openclaw`, `opencode`, `claude-code`) all set `auth.enabled: true` because they serve web UIs, so once the cluster enables auth they are automatically proxied. See [Runtimes](runtimes.md#authentication) for the runtime side of this gate.

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
