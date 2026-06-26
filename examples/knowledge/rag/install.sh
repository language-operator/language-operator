#!/usr/bin/env bash
# Deploys a self-contained Qdrant vector DB (StatefulSet), seeds it from a demo corpus via a
# one-shot ingestion Job, and registers a read-only mcp-server-qdrant LanguageTool plus a demo
# claude-code agent that answers questions grounded in that corpus. No external vector DB required.
#
# Usage: CLUSTER_NAME=my-cluster MODEL_NAME=my-model bash install.sh [--dry-run]
set -euo pipefail

: "${CLUSTER_NAME:?CLUSTER_NAME is required — set it to the name of your LanguageCluster}"
: "${MODEL_NAME:?MODEL_NAME is required — set it to the name of a LanguageModel in that cluster}"
export CLUSTER_NAME
export MODEL_NAME
export TOOL_NAME="${TOOL_NAME:-kb}"
export AGENT_NAME="${AGENT_NAME:-rag-demo}"
# One model drives both the ingestion Job and the tool's query embedding — they MUST match, so a
# single variable feeds both manifests.
export EMBEDDING_MODEL="${EMBEDDING_MODEL:-sentence-transformers/all-MiniLM-L6-v2}"

SECRET_NAME="qdrant-credentials"

DRY_RUN=false
[[ "${1:-}" == "--dry-run" ]] && DRY_RUN=true

DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
TMPDIR="$(mktemp -d)"
trap 'rm -rf "$TMPDIR"' EXIT

# Restrict substitution to our placeholders so any literal $VARs in the manifests survive to be
# expanded inside the container at runtime.
for f in "$DIR"/*.yaml; do
    envsubst '$CLUSTER_NAME $MODEL_NAME $TOOL_NAME $AGENT_NAME $EMBEDDING_MODEL' \
        < "$f" > "$TMPDIR/$(basename "$f")"
done

if $DRY_RUN; then
    kubectl kustomize "$TMPDIR"
    exit 0
fi

# Generate the API key once and reuse it on re-runs, so it never drifts from the key baked into the
# persisted Qdrant PVC on first boot. This single secret is the source of truth: the StatefulSet
# reads it as QDRANT__SERVICE__API_KEY; the tool and ingestion Job read it as QDRANT_API_KEY.
API_KEY="$(kubectl get secret "$SECRET_NAME" -n "$CLUSTER_NAME" \
    -o jsonpath='{.data.api-key}' 2>/dev/null | base64 -d || true)"
if [[ -z "$API_KEY" ]]; then
    API_KEY="$(openssl rand -hex 32)"
fi

kubectl create secret generic "$SECRET_NAME" \
    --from-literal=api-key="$API_KEY" \
    --namespace "$CLUSTER_NAME" \
    --dry-run=client -o yaml | kubectl apply -f -

# A Job's pod template is immutable, so a re-run of install.sh can't update an existing kb-ingest
# Job in place. Delete it first so it actually re-runs — deterministic point IDs make that safe
# (the second ingest upserts the same points, never duplicating them).
kubectl delete job kb-ingest -n "$CLUSTER_NAME" --ignore-not-found

kubectl kustomize "$TMPDIR" | kubectl apply -f -
