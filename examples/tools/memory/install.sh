#!/usr/bin/env bash
# Registers the official knowledge-graph memory server as a sidecar LanguageTool and deploys a
# demo claude-code agent that uses it. The sidecar persists its state on the agent's /workspace
# PVC, so memory survives pod restarts with no external store.
#
# Usage: CLUSTER_NAME=my-cluster MODEL_NAME=my-model bash install.sh [--dry-run]
set -euo pipefail

: "${CLUSTER_NAME:?CLUSTER_NAME is required — set it to the name of your LanguageCluster}"
: "${MODEL_NAME:?MODEL_NAME is required — set it to the name of a LanguageModel in that cluster}"
export CLUSTER_NAME
export MODEL_NAME
export TOOL_NAME="${TOOL_NAME:-memory}"
export AGENT_NAME="${AGENT_NAME:-memory-demo}"
export WORKSPACE_SIZE="${WORKSPACE_SIZE:-10Gi}"

DRY_RUN=false
[[ "${1:-}" == "--dry-run" ]] && DRY_RUN=true

DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
TMPDIR="$(mktemp -d)"
trap 'rm -rf "$TMPDIR"' EXIT

# Restrict substitution to our placeholders so any literal $VARs in the manifests survive to be
# expanded inside the container at runtime.
for f in "$DIR"/*.yaml; do
    envsubst '$CLUSTER_NAME $MODEL_NAME $TOOL_NAME $AGENT_NAME $WORKSPACE_SIZE' < "$f" > "$TMPDIR/$(basename "$f")"
done

if $DRY_RUN; then
    kubectl kustomize "$TMPDIR"
    exit 0
fi

# No secret, no backend — the memory server has no credentials and writes only to /workspace.
kubectl kustomize "$TMPDIR" | kubectl apply -f -
