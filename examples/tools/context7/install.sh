#!/usr/bin/env bash
# Usage: CLUSTER_NAME=my-cluster [CONTEXT7_API_KEY=ctx7_...] bash install.sh [--dry-run]
set -euo pipefail

: "${CLUSTER_NAME:?CLUSTER_NAME is required — set it to the name of your LanguageCluster}"
export CLUSTER_NAME
export TOOL_NAME="${TOOL_NAME:-context7}"

DRY_RUN=false
[[ "${1:-}" == "--dry-run" ]] && DRY_RUN=true

DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
TMPDIR="$(mktemp -d)"
trap 'rm -rf "$TMPDIR"' EXIT

for f in "$DIR"/*.yaml; do
    envsubst '$CLUSTER_NAME $TOOL_NAME' < "$f" > "$TMPDIR/$(basename "$f")"
done

if $DRY_RUN; then
    kubectl kustomize "$TMPDIR"
    exit 0
fi

# The API key is optional — Context7 works without it at a lower rate limit.
# Only create the secret when a key is provided; the tool references it with
# optional:true so the pod starts either way.
if [[ -n "${CONTEXT7_API_KEY:-}" ]]; then
    kubectl create secret generic context7-mcp-credentials \
        --from-literal=api-key="$CONTEXT7_API_KEY" \
        --namespace "$CLUSTER_NAME" \
        --dry-run=client -o yaml | kubectl apply -f -
fi

kubectl kustomize "$TMPDIR" | kubectl apply -f -
