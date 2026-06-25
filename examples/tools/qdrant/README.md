# tools/qdrant

An agent that **recalls by meaning** — and survives restarts — with zero extra infrastructure and
**no embedding API**.

This is **Rung 2** of the [tool-layer example curriculum](https://github.com/language-operator/language-operator/issues/890).
It registers the official [Qdrant MCP server](https://github.com/qdrant/mcp-server-qdrant)
(`mcp-server-qdrant`, pinned to `v0.8.1`) as a `LanguageTool` and attaches it to a demo agent. It
exposes two tools: `qdrant-store` (save a note) and `qdrant-find` (retrieve by semantic similarity).

## The concept

Rung 1 ([tools/memory](../memory/)) gave the agent an **exact-match** knowledge graph. This rung
gives it **semantic recall**: store *"the launch retro is on Friday"*, then ask *"when's the
postmortem?"* and `qdrant-find` returns the note — different words, same meaning. R1's graph
**cannot** do that; matching meaning instead of strings is the whole point of R2.

Three things make this work, all with **zero external backend**:

- **`uvx` over stdio.** `mcp-server-qdrant` is a Python package, so the operator runs it under its
  pinned stdio→Streamable-HTTP bridge using `uvx` — you don't wire up a bridge yourself. (R1 proved
  the `npx` half of the bridge toolchain; this proves the `uvx` half.)
- **Embedded Qdrant on the PVC.** `QDRANT_LOCAL_PATH=/workspace/.qdrant` runs Qdrant **embedded**
  (in-process, no separate server) and writes the store to the agent's `/workspace` PVC. Combined
  with `spec.deploymentMode: sidecar` — which injects the tool container **into the agent's own
  pod** so it can see that PVC — the vectors outlive any single pod.
- **Local FastEmbed embeddings.** `EMBEDDING_MODEL=sentence-transformers/all-MiniLM-L6-v2` computes
  embeddings **in-process** with [FastEmbed](https://github.com/qdrant/fastembed) — no API key, no
  model gateway, no per-token cost. The model is downloaded once at boot and cached on the PVC.

### Embedded vs server mode

This example uses **embedded** mode: Qdrant is a library inside the sidecar, the store is a local
file on the PVC, and there is exactly **one writer**. That is perfect for a single agent's private
memory and needs no infrastructure — but it is **not shareable** across agents. The
**server** mode (a standalone Qdrant reachable over the network, shared by many agents) is the
subject of a later rung — see **R3** in the [curriculum](https://github.com/language-operator/language-operator/issues/890).

> **Gotcha — sidecars need a workspace.** A sidecar tool can only persist if the agent has a
> `/workspace` PVC. The demo agent enables one (`spec.workspace`, default-enabled). If you strip the
> workspace, `QDRANT_LOCAL_PATH` lands on ephemeral storage and the demo no longer survives restarts.

> **Gotcha — slow first start.** The first boot downloads both the `mcp-server-qdrant` package
> (via `uvx`) and the FastEmbed model over HTTPS, so the tool can take a minute or two to reach
> `Ready`. No custom probe is needed: the operator's bridge readiness probe already allows ~120s of
> cold-start grace. Subsequent starts reuse the model cached on the PVC.

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
| `TOOL_NAME` | no | `qdrant` | Name of the LanguageTool CR (the name the agent references) |
| `AGENT_NAME` | no | `qdrant-demo` | Name of the demo LanguageAgent CR |
| `WORKSPACE_SIZE` | no | `10Gi` | Size of the agent's `/workspace` PVC |

> **Egress:** the tool manifest grants outbound HTTPS (`0.0.0.0/0:443`) because the server fetches
> two things at boot: its own package (via `uvx`) and the FastEmbed embedding model. There is no
> runtime egress — once booted, store and search are fully local. The operator's default tool policy
> denies external egress, so this boot-time rule is required.

## Install

```bash
CLUSTER_NAME=my-cluster MODEL_NAME=my-model bash examples/tools/qdrant/install.sh
```

Dry-run (prints rendered YAML, applies nothing):
```bash
CLUSTER_NAME=my-cluster MODEL_NAME=my-model bash examples/tools/qdrant/install.sh --dry-run
```

## What's created

- `LanguageTool/qdrant` — the MCP tool CR (`transport: stdio`, `deploymentMode: sidecar`)
- `LanguageAgent/qdrant-demo` — a `claude-code` agent that references the tool and a model, with a
  `/workspace` PVC for the sidecar to write to

No secret and no backend are created — embedded Qdrant has no credentials and FastEmbed embeds locally.

Once the tool reaches `Ready`, the operator's MCP handshake populates `status.toolSchemas`, which
proves the server speaks MCP through the bridge — you should see `qdrant-store` and `qdrant-find`:

```bash
kubectl get languagetool qdrant -n my-cluster -o jsonpath='{.status.toolSchemas}' ; echo
```

The qdrant container runs **inside the agent pod** (sidecar). Confirm it and its `/workspace` mount:

```bash
kubectl get pod -n my-cluster -l app.kubernetes.io/name=qdrant-demo,langop.io/component=agent \
  -o jsonpath='{.items[0].spec.containers[*].name}' ; echo
```

## The demo: recall by meaning, not by words

1. **Tell the agent a note.** Talk to the `qdrant-demo` agent however your cluster exposes it and
   give it something to remember in specific words, e.g. *"The launch retro is on Friday at 3pm."*
   The agent stores it via `qdrant-store`.

2. **Ask with different words.** *"When's the postmortem?"* — note that you never said "retro" or
   "Friday". The agent calls `qdrant-find`, the vector search matches on *meaning*, and it answers
   from the stored note. This is the headline contrast with R1: an exact-match store would find
   nothing here.

3. **(Bonus) Survive a pod delete.** The embedded store lives on the PVC, so it outlives the pod:
   ```bash
   kubectl delete pod -n my-cluster -l app.kubernetes.io/name=qdrant-demo,langop.io/component=agent
   ```
   After the Deployment reschedules a fresh pod (which re-attaches the same `/workspace` PVC and
   reuses the cached embedding model), ask again — the note is still there.

## Teardown

```bash
kubectl delete languageagent qdrant-demo -n my-cluster
kubectl delete languagetool qdrant -n my-cluster
```

Deleting the agent also removes its `/workspace` PVC (and the stored vectors) per the cluster's
reclaim policy.

---

Part of the [tool-layer example curriculum (R1–R7)](https://github.com/language-operator/language-operator/issues/890).
