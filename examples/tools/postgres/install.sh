#!/usr/bin/env bash
# Usage: CLUSTER_NAME=my-cluster DATABASE_URL=postgresql://... bash install.sh [--dry-run]
set -euo pipefail

: "${CLUSTER_NAME:?CLUSTER_NAME is required — set it to the name of your LanguageCluster}"
export CLUSTER_NAME
export TOOL_NAME="${TOOL_NAME:-postgres}"

DRY_RUN=false
[[ "${1:-}" == "--dry-run" ]] && DRY_RUN=true

# The connection URL is only needed when actually applying to a cluster.
if ! $DRY_RUN; then
    : "${DATABASE_URL:?DATABASE_URL is required — a postgres connection string, e.g. postgresql://user:pass@host:5432/db}"
fi

DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
TMPDIR="$(mktemp -d)"
trap 'rm -rf "$TMPDIR"' EXIT

# Restrict substitution to our placeholders so the literal $DATABASE_URL in the
# tool's stdio command survives to be expanded inside the container at runtime.
for f in "$DIR"/*.yaml; do
    envsubst '$CLUSTER_NAME $TOOL_NAME' < "$f" > "$TMPDIR/$(basename "$f")"
done

if $DRY_RUN; then
    kubectl kustomize "$TMPDIR"
    exit 0
fi

kubectl create secret generic postgres-mcp-credentials \
    --from-literal=url="$DATABASE_URL" \
    --namespace "$CLUSTER_NAME" \
    --dry-run=client -o yaml | kubectl apply -f -

kubectl kustomize "$TMPDIR" | kubectl apply -f -
