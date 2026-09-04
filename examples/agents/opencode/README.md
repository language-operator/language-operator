# agents/opencode

Deploys a single OpenCode coding agent with a persistent workspace and unrestricted egress,
wired to a `LanguageModel` already registered in the cluster — a good starting point for
browser-based AI pair programming.

Unlike Claude Code, OpenCode talks to an LLM through the cluster gateway, so it needs a
`LanguageModel` to exist in the namespace. This example references one by name (`claude-sonnet`
by default) rather than bundling it — register the model separately first (see
[models/anthropic](../../models/anthropic/), which registers `claude-sonnet` on the latest
Sonnet plus `claude-opus`). Access to the OpenCode web UI is gated by the cluster's OIDC proxy
when the cluster has auth enabled — there is no separate opencode password.

## Prerequisites

- Language Operator [installed](https://langop.io/docs/getting-started/installation/)
- `language-operator-runtimes` chart installed (provides the `opencode` runtime)
- `kubectl` configured for your cluster
- `envsubst` (`brew install gettext` on macOS, pre-installed on most Linux distros)
- A `LanguageCluster` already applied in the target namespace (see [clusters/basic](../../clusters/basic/))
- A `LanguageModel` named `claude-sonnet` registered in that namespace (see [models/anthropic](../../models/anthropic/))

## Variables

| Variable | Required | Default | Description |
|----------|----------|---------|-------------|
| `CLUSTER_NAME` | yes | — | Namespace of the target LanguageCluster |
| `AGENT_NAME` | no | `opencode` | Name of the LanguageAgent |
| `MODEL_NAME` | no | `claude-sonnet` | Name of the LanguageModel CR the agent references (must already exist) |
| `WORKSPACE_SIZE` | no | `10Gi` | Persistent workspace PVC size |

## Install

Register a model first, then deploy the agent:

```bash
# 1. register the model (latest Sonnet + Opus) and its credentials secret
CLUSTER_NAME=my-cluster ANTHROPIC_API_KEY=sk-ant-... bash examples/models/anthropic/install.sh

# 2. deploy the agent
CLUSTER_NAME=my-cluster bash examples/agents/opencode/install.sh
```

Dry-run (prints rendered YAML):
```bash
CLUSTER_NAME=my-cluster bash examples/agents/opencode/install.sh --dry-run
```

## Authentication

No first-time setup is required. Access is gated by the cluster's OIDC proxy: when the
`LanguageCluster` has [auth enabled](https://langop.io/docs/components/clusters/#authentication), the
operator injects an oauth2-proxy sidecar in front of OpenCode and you sign in through the cluster's
OIDC provider.

> **Note:** OpenCode has no built-in authentication. When the cluster does **not** enable auth, the
> web UI is exposed unauthenticated on its ingress. Enable cluster auth to protect it.

## What's created

- `LanguageAgent/opencode` — the agent CR
- `WorkflowTemplate/opencode` — the agent's pod spec
- `Workflow/opencode` — the long-lived run; runs the opencode container on port 3000
- `Service/opencode` — exposes the web UI
- `NetworkPolicy/opencode` — restricts ingress; allows unrestricted egress
- `PersistentVolumeClaim/opencode-workspace` — workspace at `/workspace`
- `ConfigMap/opencode-agent` — assembled agent configuration

## Access

### With OIDC auth (cluster has a domain)

```
https://<AGENT_NAME>.<your-domain>
```

Sign in through the OIDC provider configured on the cluster.

### Local access (port-forward)

Port-forward bypasses the OIDC proxy, so no login is required:

```bash
kubectl port-forward -n my-cluster svc/opencode 3000:3000
# open http://localhost:3000, or attach the TUI:
opencode attach http://localhost:3000
```

## Teardown

```bash
kubectl delete languageagent opencode -n my-cluster
```

The referenced `LanguageModel` and its secret are managed separately — tear them down with
[models/anthropic](../../models/anthropic/) if you no longer need them.
