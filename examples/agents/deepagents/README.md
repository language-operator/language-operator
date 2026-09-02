# agents/deepagents

Deploys a single [deepagents](https://github.com/langchain-ai/deepagents) agent that
**autonomously runs a task** — here, researching a library through the `context7` MCP tool
and writing a cited summary to its workspace.

Unlike the coding-CLI runtimes (claude-code, opencode, openclaw), deepagents is a *framework*
runtime: it is not an interactive terminal you drive. On startup it reads its
`spec.instructions`, builds a deep agent (planning, sub-agents, a `/workspace` filesystem,
and any referenced MCP tools) pointed at the cluster gateway, runs the task once, then idles.
**`kubectl logs` is the primary UI** — every step streams to stdout. A thin web view at `/`
mirrors the stream and adds Approve/Reject buttons when human-in-the-loop is enabled.

Because the model is reached through the cluster gateway, this example references a
`LanguageModel` by name (`claude-sonnet` by default) rather than bundling it — register the
model separately first (see [models/anthropic](../../models/anthropic/)). The `context7` tool
*is* bundled here so the example is self-contained.

## Prerequisites

- Language Operator [installed](https://langop.io/docs/getting-started/installation/)
- `language-operator-runtimes` chart installed (provides the `deepagents` runtime)
- `kubectl` configured for your cluster
- `envsubst` (`brew install gettext` on macOS, pre-installed on most Linux distros)
- A `LanguageCluster` already applied in the target namespace (see [clusters/basic](../../clusters/basic/))
- A `LanguageModel` named `claude-sonnet` registered in that namespace (see [models/anthropic](../../models/anthropic/))

## Variables

| Variable | Required | Default | Description |
|----------|----------|---------|-------------|
| `CLUSTER_NAME` | yes | — | Namespace of the target LanguageCluster |
| `AGENT_NAME` | no | `researcher` | Name of the LanguageAgent |
| `MODEL_NAME` | no | `claude-sonnet` | Name of the LanguageModel CR the agent references (must already exist) |
| `WORKSPACE_SIZE` | no | `10Gi` | Persistent workspace PVC size |

## Install

Register a model first, then deploy the agent + tool:

```bash
# 1. register the model (latest Sonnet + Opus) and its credentials secret
CLUSTER_NAME=my-cluster ANTHROPIC_API_KEY=sk-ant-... bash examples/models/anthropic/install.sh

# 2. deploy the deepagents agent and the context7 tool
CLUSTER_NAME=my-cluster bash examples/agents/deepagents/install.sh
```

Dry-run (prints rendered YAML):
```bash
CLUSTER_NAME=my-cluster bash examples/agents/deepagents/install.sh --dry-run
```

## Watch it run

The agent starts working as soon as the pod is `Running` — there is nothing to log into.

```bash
argo logs @latest -n my-cluster -f
```

Or, without the `argo` CLI:

```bash
kubectl logs -n my-cluster -f -l app.kubernetes.io/name=researcher -c main
```

You'll see the agent plan, call the `context7` tool, and write
`/workspace/deepagents-summary.md`.

For the live web view (status, streaming output, and HITL controls):

```bash
kubectl port-forward -n my-cluster svc/researcher 8080:8080
# open http://localhost:8080
```

## Human-in-the-loop

By default the runtime pauses before side-effecting tools (`write_file`/`edit_file` and every
MCP tool) and waits for approval. This example sets `HITL_TOOLS=none` in the agent's
`deployment.env` for a hands-off first run. To require approvals instead, remove that env var
(or set it to `"*"`); the run will pause as `interrupted`, and you approve/reject from the web
view at `/` or via `POST /resume`.

## What's created

- `LanguageAgent/researcher` — the agent CR
- `LanguageTool/context7` — the bundled MCP documentation tool
- `Deployment/researcher` — runs the deepagents container on port 8080
- `Service/researcher` — exposes the live view / HITL endpoints
- `NetworkPolicy/researcher` — operator-managed; in-cluster access to the gateway and tool
- `PersistentVolumeClaim/researcher` — workspace at `/workspace` (agent files + checkpoint DB)
- `ConfigMap/researcher-config` — assembled agent configuration

The agent itself needs no external egress — it reaches the model through the in-cluster gateway
and the tool through its in-cluster MCP service.

## Teardown

```bash
kubectl delete languageagent researcher -n my-cluster
kubectl delete languagetool context7 -n my-cluster
```

The referenced `LanguageModel` and its secret are managed separately — tear them down with
[models/anthropic](../../models/anthropic/) if you no longer need them.
