# agents/claude-code

Deploys a single Claude Code agent with a persistent workspace and unrestricted egress — a good starting point for interactive development or as a base for custom agents.

Authentication is interactive: after deploying, open the agent terminal and run `/login`. Credentials, sessions, and project history persist on the workspace PVC, so subsequent pod restarts don't re-prompt.

## Prerequisites

- Language Operator [installed](https://langop.io/docs/getting-started/installation/)
- `language-operator-runtimes` chart installed (provides the `claude-code` runtime)
- `kubectl` configured for your cluster
- `envsubst` (`brew install gettext` on macOS, pre-installed on most Linux distros)
- A `LanguageCluster` already applied in the target namespace (see [clusters/basic](../../clusters/basic/))

## Variables

| Variable | Required | Default | Description |
|----------|----------|---------|-------------|
| `CLUSTER_NAME` | yes | — | Namespace of the target LanguageCluster |
| `AGENT_NAME` | no | `my-agent` | Name of the LanguageAgent |
| `WORKSPACE_SIZE` | no | `10Gi` | Persistent workspace PVC size |

## Install

```bash
CLUSTER_NAME=my-cluster bash examples/agents/claude-code/install.sh
```

Dry-run (prints rendered YAML):
```bash
CLUSTER_NAME=my-cluster bash examples/agents/claude-code/install.sh --dry-run
```

## First-time setup

After the pod is `Running`, open the agent terminal and run `/login` inside Claude Code. Sign in with your Claude account through the browser flow. Credentials are written to `/workspace/.claude/.credentials.json` and persist across pod restarts.

## What's created

- `LanguageAgent/my-agent` — the agent CR
- `Deployment/my-agent` — runs the claude-code container
- `Service/my-agent` — exposes the WebSocket terminal
- `NetworkPolicy/my-agent` — restricts ingress; allows unrestricted egress
- `PersistentVolumeClaim/my-agent-workspace` — workspace at `/workspace` (also holds Claude config)
- `ConfigMap/my-agent-agent` — assembled agent configuration

## Access

### With OIDC auth (cluster has a domain)

```
https://<AGENT_NAME>.<your-domain>
```

Sign in through the OIDC provider configured on the cluster.

### Local access (port-forward)

```bash
kubectl port-forward -n my-cluster svc/my-agent 8080:8080
# open http://localhost:8080
```

## Teardown

```bash
kubectl delete languageagent my-agent -n my-cluster
```
