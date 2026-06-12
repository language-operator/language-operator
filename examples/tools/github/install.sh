#!/usr/bin/env bash
# Usage: CLUSTER_NAME=my-cluster GITHUB_TOKEN=ghp_... bash install.sh [--dry-run]
set -euo pipefail

: "${CLUSTER_NAME:?CLUSTER_NAME is required — set it to the name of your LanguageCluster}"
export CLUSTER_NAME
export TOOL_NAME="${TOOL_NAME:-github}"

DRY_RUN=false
[[ "${1:-}" == "--dry-run" ]] && DRY_RUN=true

# The token is only needed when actually applying to a cluster.
if ! $DRY_RUN; then
    : "${GITHUB_TOKEN:?GITHUB_TOKEN is required — a GitHub PAT for the MCP server to use}"
fi

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

kubectl create secret generic github-mcp-credentials \
    --from-literal=token="$GITHUB_TOKEN" \
    --namespace "$CLUSTER_NAME" \
    --dry-run=client -o yaml | kubectl apply -f -

kubectl kustomize "$TMPDIR" | kubectl apply -f -
