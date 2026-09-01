# Network Policies

Language Operator gives every workload a Kubernetes `NetworkPolicy`. The operator generates the
rules each workload needs to function — agent-to-agent traffic, the shared gateway, DNS — and you
add anything else (egress to public APIs, tighter ingress) through `spec.networkPolicies`. This
applies to `LanguageAgent`, `LanguageTool`, and `LanguageCluster`.

Network isolation is on by default (`networkIsolation.enabled: true` in the operator's Helm
values) and **requires a NetworkPolicy-capable CNI** — see [Requirements](#requirements-and-gotchas).

## What the operator allows by default

Each agent gets a `NetworkPolicy` named after it (`<agent-name>`), owned by the agent so it is
deleted alongside it. With no `spec.networkPolicies` of your own, it permits:

**Ingress** — restricted to:

- other agents in the namespace (label `langop.io/kind=LanguageAgent`)
- the ingress controller's namespace, when the operator is configured with one

...on the agent's port(s) (`spec.ports`, default `8080`).

**Egress** — restricted to:

- any pod in the same namespace — this is how an agent reaches the shared gateway, MCP tools, and other agents
- cluster DNS (kube-dns, UDP/TCP 53)
- the Kubernetes API server
- the OpenTelemetry collector, when one is configured on the operator

The net effect: an agent can talk to the gateway, tools, other agents, and DNS — but **not to
arbitrary external hosts**. Any direct public-API call needs an explicit egress rule.

!!! note "Tools are different"

    A `LanguageTool`'s policy restricts egress the same way but leaves **ingress open** by default,
    so agents in the namespace can reach it. Adding `spec.networkPolicies.ingress` to a tool locks
    its ingress down to the sources you list.

## Allowing outbound traffic (the common case)

LLM traffic flows through the in-cluster gateway, which is already allowed. But runtimes that call
public endpoints directly need egress — Claude Code reaching `claude.ai` / `api.anthropic.com`,
`git clone` over HTTPS, `npx` pulling a package. The blanket pattern used across the examples:

```yaml
spec:
  networkPolicies:
    egress:
      - to:
          - cidr: "0.0.0.0/0"
        ports:
          - port: 443
            protocol: TCP   # protocol is optional; defaults to TCP
          - port: 80
            protocol: TCP
```

Scope it down by replacing `0.0.0.0/0` with specific CIDRs or DNS names (below).

## Peer types

Each `to` / `from` entry is a `NetworkPeer`, paired with `ports` (a required `port` 1–65535 and an
optional `protocol`: `TCP`, `UDP`, or `SCTP`, default `TCP`).

| Field | Selects | Example |
|-------|---------|---------|
| `cidr` | An IP range | `cidr: "203.0.113.0/24"` |
| `dns` | Hostnames, resolved to IPs at reconcile time (wildcards allowed) | `dns: ["api.anthropic.com", "*.googleapis.com"]` |
| `service` | A Kubernetes Service (by name, optional namespace) | `service: {name: postgres, namespace: data}` |
| `group` | Agents tagged with a shared `langop.io/group` label | `group: data-pipeline` |
| `namespaceSelector` / `podSelector` | Standard label selectors | `namespaceSelector: {matchLabels: {kubernetes.io/metadata.name: shared}}` |

Egress by DNS name:

```yaml
spec:
  networkPolicies:
    egress:
      - to:
          - dns: ["api.anthropic.com", "api.openai.com"]
        ports:
          - port: 443
```

Grouping — label the member agents, then reference the group:

```yaml
# on each member agent
spec:
  deployment:
    podLabels:
      langop.io/group: data-pipeline
---
# allow only that group to reach this agent
spec:
  networkPolicies:
    ingress:
      - from:
          - group: data-pipeline
        ports:
          - port: 8080
```

!!! note "DNS rules are a point-in-time snapshot"

    The operator resolves `dns:` peers to IP blocks **when it reconciles the policy** and bakes them
    into a standard `NetworkPolicy`, so it works on any NetworkPolicy-capable CNI — no Cilium FQDN
    policy required. Because it's a snapshot, hostnames behind rotating or CDN IPs can drift between
    reconciles; prefer `cidr` for stable ranges, or a CNI with native FQDN support for high-churn
    endpoints. A failed lookup **fails closed** (the peer contributes no IPs rather than opening
    everything).

## Where to set rules

Rules you add are **appended** to the operator's defaults — they widen access, they don't replace
the baseline.

- **Per agent** — `spec.networkPolicies` on the `LanguageAgent` (most specific).
- **Per tool** — `spec.networkPolicies` on the `LanguageTool`. For example, the `context7` tool
  needs `0.0.0.0/0:443` to fetch its package from npm at boot and call the Context7 API at runtime.
- **Cluster-wide** — `spec.networkPolicies` on the `LanguageCluster` applies to every agent in the
  namespace; see [Clusters → Network Isolation](../components/clusters.md#network-isolation). Use
  this for rules every agent needs, such as egress to your model providers.

## Disabling isolation

Set `networkIsolation.enabled: false` in the operator's Helm values (equivalently
`LANGOP_NETWORK_ISOLATION_ENABLED=false`) to skip `NetworkPolicy` creation entirely — every
workload gets unrestricted network access. Useful on a local cluster or a CNI without NetworkPolicy
support; not recommended in shared or production clusters.

## Requirements and gotchas

- **A NetworkPolicy-capable CNI is required.** Plain flannel / kindnet ignore NetworkPolicies, so
  the rules become silent no-ops and isolation is *not* enforced. Cilium, Calico, Antrea, and Weave
  Net all work. The operator detects support at startup and logs a warning if it's missing; start it
  with `--require-network-policy` to hard-fail instead.
- **DNS egress is allowed by default and required** — don't strip it, or in-cluster name resolution
  breaks for every workload.
- **`dns:` peers are point-in-time** and fail closed (see the note above).

For the full `NetworkPeer` field reference, see the
[LanguageAgent API](../api/languageagent.md) and [LanguageCluster API](../api/languagecluster.md#network-isolation).
