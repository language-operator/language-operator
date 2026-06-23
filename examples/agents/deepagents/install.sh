#!/usr/bin/env bash
# Usage: CLUSTER_NAME=my-cluster bash install.sh [--dry-run]
#
# Requires a LanguageModel named "$MODEL_NAME" to already exist in the cluster
# namespace — deploy one first, e.g. examples/models/anthropic (registers
# claude-sonnet / claude-opus and their credentials secret).
set -euo pipefail

: "${CLUSTER_NAME:?CLUSTER_NAME is required — set it to the name of your LanguageCluster}"
export AGENT_NAME="${AGENT_NAME:-researcher}"
export CLUSTER_NAME
export MODEL_NAME="${MODEL_NAME:-claude-sonnet}"
export WORKSPACE_SIZE="${WORKSPACE_SIZE:-10Gi}"

DRY_RUN=false
[[ "${1:-}" == "--dry-run" ]] && DRY_RUN=true

DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
TMPDIR="$(mktemp -d)"
trap 'rm -rf "$TMPDIR"' EXIT

for f in "$DIR"/*.yaml; do
    envsubst < "$f" > "$TMPDIR/$(basename "$f")"
done

if $DRY_RUN; then
    kubectl kustomize "$TMPDIR"
    exit 0
fi

kubectl kustomize "$TMPDIR" | kubectl apply -f -
