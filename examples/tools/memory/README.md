# tools/memory

An agent that **remembers across restarts** — with zero extra infrastructure.

This is **Rung 1** of the [tool-layer example curriculum](https://github.com/language-operator/language-operator/issues/890).
It registers the official [knowledge-graph memory server](https://www.npmjs.com/package/@modelcontextprotocol/server-memory)
(`@modelcontextprotocol/server-memory`) as a `LanguageTool` and attaches it to a demo agent.

## The concept

The memory server is a **stdio** MCP server, so the operator runs it under a pinned
stdio→Streamable-HTTP bridge — you don't wire up a bridge yourself. The one thing that makes this
example special is `spec.deploymentMode: sidecar`:

- In the default **`service`** mode a tool runs as a shared `Deployment`, and the bridge's scratch
  space is an ephemeral `EmptyDir` — gone on every restart.
- In **`sidecar`** mode the tool container is injected **into the agent's own pod**, where it can
  see the agent's `/workspace` PVC.

Point the memory file at that PVC (`MEMORY_FILE_PATH=/workspace/.memory/memory.jsonl`) and the
knowledge graph outlives any single pod. Tell the agent some facts, delete its pod, and after it
reschedules it still recalls them — no database, no StatefulSet, no external store.

> **Gotcha — sidecars need a workspace.** A sidecar tool can only persist if the agent has a
> `/workspace` PVC. The demo agent enables one (`spec.workspace`, default-enabled). If you strip the
> workspace, `MEMORY_FILE_PATH` lands on ephemeral storage and the demo no longer survives restarts.

## Prerequisites

- Language Operator [installed](https://langop.io/docs/getting-started/installation/)
- `kubectl` configured for your cluster
- `envsubst` (`brew install gettext` on macOS, pre-installed on most Linux distros)
- A `LanguageCluster` already applied in the target namespace (see [clusters/basic](../../clusters/basic/))
- A `LanguageModel` in that cluster for the agent to use (see [models/anthropic](../../models/anthropic/))
- Credentials for the `claude-code` runtime — an `anthropic-credentials` secret (`api-key`) **or**
  a `claude-code-oauth` secret (`token`) in the cluster namespace

## Variables

| Variable | Required | Default | Description |
|----------|----------|---------|-------------|
| `CLUSTER_NAME` | yes | — | Namespace of the target LanguageCluster |
| `MODEL_NAME` | yes | — | Name of a LanguageModel the demo agent should use |
| `TOOL_NAME` | no | `memory` | Name of the LanguageTool CR (the name the agent references) |
| `AGENT_NAME` | no | `memory-demo` | Name of the demo LanguageAgent CR |
| `WORKSPACE_SIZE` | no | `10Gi` | Size of the agent's `/workspace` PVC |

> **Egress:** the tool manifest grants outbound HTTPS (`0.0.0.0/0:443`) because `npx` fetches the
> server package from the npm registry at boot. The server itself makes no network calls — it only
> reads and writes the local memory file. The operator's default tool policy denies external egress,
> so this boot-time rule is required.

## Install

```bash
CLUSTER_NAME=my-cluster MODEL_NAME=my-model bash examples/tools/memory/install.sh
```

Dry-run (prints rendered YAML, applies nothing):
```bash
CLUSTER_NAME=my-cluster MODEL_NAME=my-model bash examples/tools/memory/install.sh --dry-run
```

## What's created

- `LanguageTool/memory` — the MCP tool CR (`transport: stdio`, `deploymentMode: sidecar`)
- `LanguageAgent/memory-demo` — a `claude-code` agent that references the tool and a model, with a
  `/workspace` PVC for the sidecar to write to

No secret and no backend are created — the memory server has no credentials.

Once the tool reaches `Ready`, the operator's MCP handshake populates `status.toolSchemas`, which
proves the server speaks MCP through the bridge:

```bash
kubectl get languagetool memory -n my-cluster -o jsonpath='{.status.toolSchemas}' ; echo
```

The memory container runs **inside the agent pod** (sidecar). Confirm it and its `/workspace` mount:

```bash
kubectl get pod -n my-cluster -l app.kubernetes.io/name=memory-demo,langop.io/component=agent \
  -o jsonpath='{.items[0].spec.containers[*].name}' ; echo
```

## The demo: memory that survives a pod delete

1. **Tell the agent some facts.** Talk to the `memory-demo` agent however your cluster exposes it
   and give it 2–3 things to remember, e.g. *"My name is James, my favorite language is Go, and our
   staging cluster is called `aurora`."* The agent records them via the memory tool.

2. **Delete the agent pod.**
   ```bash
   kubectl delete pod -n my-cluster -l app.kubernetes.io/name=memory-demo,langop.io/component=agent
   ```
   The Deployment reschedules a fresh pod. The new pod re-attaches the same `/workspace` PVC, so the
   `memory.jsonl` written earlier is still there.

3. **Ask it what it remembers.** *"What's my name and favorite language?"* The agent queries the
   memory tool and answers correctly — proof the state survived on the PVC, not in pod memory.

## Teardown

```bash
kubectl delete languageagent memory-demo -n my-cluster
kubectl delete languagetool memory -n my-cluster
```

Deleting the agent also removes its `/workspace` PVC (and the stored memory) per the cluster's
reclaim policy.

---

Part of the [tool-layer example curriculum (R1–R7)](https://github.com/language-operator/language-operator/issues/890).
