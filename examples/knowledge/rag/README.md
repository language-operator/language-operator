# knowledge/rag

An agent that **answers from a private knowledge base and cites its source** — RAG over a
**seeded corpus**, with the vector store living in a real, bundled Qdrant server. Zero external
infrastructure and **no embedding API**.

This is **Rung 3** of the [tool-layer example curriculum](https://github.com/language-operator/language-operator/issues/890),
and the first entry in the `examples/knowledge/` family. It deploys a self-contained
[Qdrant](https://github.com/qdrant/qdrant) `StatefulSet`, loads a demo corpus into it with a
one-shot ingestion `Job`, and registers the official
[Qdrant MCP server](https://github.com/qdrant/mcp-server-qdrant) (`mcp-server-qdrant`, pinned to
`v0.8.1`) as a **read-only** `LanguageTool` the agent uses to retrieve.

## The concept

R1 ([tools/memory](../../tools/memory/)) and R2 ([tools/qdrant](../../tools/qdrant/)) gave an agent
a memory it **writes to itself**, persisted on its own `/workspace` PVC via a **sidecar** tool. RAG
is the opposite shape: a **curated corpus the agent only reads**. Two things change because of that:

- **sidecar → service.** A service-mode stdio bridge has only ephemeral scratch, so the corpus
  can't live "inside" the tool — it must sit in a real backend. This rung stands up a Qdrant
  `StatefulSet` with its own PVC, exactly as [tools/postgres](../../tools/postgres/) does for SQL.
  One standalone tool (`deploymentMode: service`) can then be shared by many agents.
- **read-only → retrieval, not memory.** `QDRANT_READ_ONLY=true` makes `mcp-server-qdrant` expose
  only `qdrant-find` (no `qdrant-store`). That is the headline contrast with R2: agents *retrieve*
  from the corpus, they can't mutate it. The corpus is seeded once, out of band, by the Job.

Everything still runs with **no embedding API**: [FastEmbed](https://github.com/qdrant/fastembed)
computes embeddings locally, both when the Job ingests and when the tool answers a query.

### How the corpus is seeded

```
configmap.corpus.yaml ──▶ Job (uv + qdrant-client[fastembed]) ──embeds & upserts──▶ Qdrant "kb"
                                                                                         ▲
                          LanguageTool (mcp-server-qdrant, read-only) ──qdrant-find──────┘
                                                                                         │
                                                          LanguageAgent (claude-code) ───┘
```

The `Job` embeds each doc with FastEmbed and writes points into collection `kb` in the **exact wire
format** `mcp-server-qdrant` reads back — named vector `fast-<model>`, payload
`{"document", "metadata"}`, cosine distance. It is **idempotent**: each point's ID is derived
deterministically from the doc's `id` (`uuid5`), so re-running upserts in place and never
duplicates. `install.sh` deletes any prior `kb-ingest` Job before re-applying (a Job's pod template
is immutable), so a second `install.sh` genuinely re-ingests — and still ends with one point per doc.

### ⚠️ The embedding model must match

The ingestion Job and the tool **must** use the same `EMBEDDING_MODEL`. Different models produce
different vectors *and* a different vector name (`fast-<model>`), so a mismatch makes `qdrant-find`
query an empty vector and **silently return nothing** — no error, just bad answers. This example
feeds **one** `${EMBEDDING_MODEL}` variable into both manifests so they can't drift; keep it that
way if you swap the model.

## Single-secret wiring

One secret, `qdrant-credentials` (key `api-key`), is the single source of truth. `install.sh`
generates it once (and reuses it on re-runs, so it never drifts from the key baked into the
persisted PVC) and wires it into all three consumers:

| Consumer | Reads the key as |
|----------|------------------|
| Qdrant `StatefulSet` | `QDRANT__SERVICE__API_KEY` (turns on auth) |
| `LanguageTool` (mcp-server-qdrant) | `QDRANT_API_KEY` |
| Ingestion `Job` | `QDRANT_API_KEY` |

## Prerequisites

- Language Operator [installed](https://langop.io/docs/getting-started/installation/)
- `kubectl` configured for your cluster
- `envsubst` (`brew install gettext` on macOS, pre-installed on most Linux distros)
- `openssl` (for API-key generation — pre-installed on macOS and most Linux distros)
- A `LanguageCluster` already applied in the target namespace (see [clusters/basic](../../clusters/basic/))
- A `LanguageModel` in that cluster for the demo agent (see [models/anthropic](../../models/anthropic/))
- A default StorageClass (the Qdrant StatefulSet requests a 1Gi PVC)
- Credentials for the `claude-code` runtime — an `anthropic-credentials` secret (`api-key`) **or**
  a `claude-code-oauth` secret (`token`) in the cluster namespace

## Variables

| Variable | Required | Default | Description |
|----------|----------|---------|-------------|
| `CLUSTER_NAME` | yes | — | Namespace of the target LanguageCluster |
| `MODEL_NAME` | yes | — | Name of a LanguageModel the demo agent should use |
| `TOOL_NAME` | no | `kb` | Name of the LanguageTool CR (the name the agent references); also prefixes the bundled `${TOOL_NAME}-db` Qdrant resources |
| `AGENT_NAME` | no | `rag-demo` | Name of the demo LanguageAgent CR |
| `EMBEDDING_MODEL` | no | `sentence-transformers/all-MiniLM-L6-v2` | FastEmbed model — fed to **both** the Job and the tool |

> **Egress:** the tool manifest grants outbound HTTPS (`0.0.0.0/0:443`) for the boot-time `uvx`
> package fetch and FastEmbed model download. Qdrant is in the same namespace, which the operator's
> default tool policy already allows, so no backend egress rule is needed.

## Install

```bash
CLUSTER_NAME=my-cluster MODEL_NAME=my-model bash examples/knowledge/rag/install.sh
```

Dry-run (prints rendered YAML, applies nothing):
```bash
CLUSTER_NAME=my-cluster MODEL_NAME=my-model bash examples/knowledge/rag/install.sh --dry-run
```

## What's created

- `Secret/qdrant-credentials` — auto-generated `api-key`, the single source of truth
- `StatefulSet/kb-db` + its 1Gi `PersistentVolumeClaim` — the bundled Qdrant server
- `Service/kb-db` — headless service giving Qdrant stable in-cluster DNS (REST 6333, gRPC 6334)
- `ConfigMap/kb-corpus` — the demo corpus (`corpus.json`)
- `ConfigMap/kb-ingest-script` + `Job/kb-ingest` — the one-shot, idempotent ingestion
- `LanguageTool/kb` — the read-only MCP tool CR (`transport: stdio`, `deploymentMode: service`)
- `Deployment/kb` + `Service/kb` — the operator-injected bridge running `mcp-server-qdrant`, exposing
  `/mcp` in-cluster on port 8080
- `LanguageAgent/rag-demo` — a `claude-code` agent that retrieves from `kb` and cites sources
- `WorkflowTemplate/rag-demo` + `Workflow/rag-demo` — the agent runs as an Argo Workflow in the
  default `execution.mode: service`, so it is always on and addressable
- `Service/rag-demo` — exposes the agent's terminal

Watch the ingestion finish, then confirm the tool exposes **only** `qdrant-find` (read-only → no
`qdrant-store`):

```bash
kubectl wait --for=condition=complete job/kb-ingest -n my-cluster --timeout=300s
kubectl get languagetool kb -n my-cluster -o jsonpath='{.status.toolSchemas}' ; echo
```

> **Gotcha — slow first start.** The Job and the tool each download the FastEmbed model and their
> packages over HTTPS on first boot, so both can take a minute or two. The operator's bridge
> readiness probe already allows ~120s of cold-start grace; no custom probe is needed.

## The demo: grounded answers with citations

Talk to the `rag-demo` agent however your cluster exposes it.

1. **Ask something in the corpus.** *"How many PTO days do engineers get, and what happens to unused
   days?"* The agent calls `qdrant-find`, retrieves the **Time Off Policy** doc, and answers from it
   — *"26 days, accrued monthly; up to 10 roll over, the rest is paid out in January (source: Time
   Off Policy)."* The facts are fictional (about "Meridian Robotics"), so a correct, specific answer
   can only have come from retrieval.

2. **Ask something outside the corpus.** *"What's Meridian Robotics' stock price?"* Nothing in the
   knowledge base grounds it, so the agent declines — *"That isn't in the knowledge base."* — rather
   than guessing from world knowledge. That refusal is the point of a RAG agent.

## Swap in your own corpus

Replace `corpus.json` in [configmap.corpus.yaml](configmap.corpus.yaml) with your own docs (keep the
`{"id", "title", "text"}` shape — `id` drives the deterministic point ID, `title` is the citation
label), then re-run `install.sh`. Keep `EMBEDDING_MODEL` identical between runs, or re-ingest into a
fresh collection.

## Teardown

```bash
kubectl delete languageagent rag-demo -n my-cluster
kubectl delete languagetool kb -n my-cluster
kubectl delete job kb-ingest -n my-cluster
kubectl delete statefulset kb-db -n my-cluster
kubectl delete service kb-db -n my-cluster
kubectl delete configmap kb-corpus kb-ingest-script -n my-cluster
kubectl delete secret qdrant-credentials -n my-cluster
# The PVC is retained by default — delete it to wipe the corpus (and allow a fresh API key):
kubectl delete pvc storage-kb-db-0 -n my-cluster
```

---

Part of the [tool-layer example curriculum (R1–R7)](https://github.com/language-operator/language-operator/issues/890).
