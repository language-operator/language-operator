#!/usr/bin/env bash
# Usage: CLUSTER_NAME=my-cluster bash install.sh [--dry-run]
set -euo pipefail

: "${CLUSTER_NAME:?CLUSTER_NAME is required — set it to the name of your LanguageCluster}"
export AGENT_NAME="${AGENT_NAME:-my-agent}"
export CLUSTER_NAME
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
