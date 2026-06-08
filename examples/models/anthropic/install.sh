#!/usr/bin/env bash
# Usage: CLUSTER_NAME=my-cluster ANTHROPIC_API_KEY=sk-ant-... bash install.sh [--dry-run]
set -euo pipefail

: "${CLUSTER_NAME:?CLUSTER_NAME is required — set it to the name of your LanguageCluster}"
export CLUSTER_NAME
export OPUS_NAME="${OPUS_NAME:-claude-opus}"
export OPUS_MODEL_ID="${OPUS_MODEL_ID:-claude-opus-4-8}"
export SONNET_NAME="${SONNET_NAME:-claude-sonnet}"
export SONNET_MODEL_ID="${SONNET_MODEL_ID:-claude-sonnet-4-6}"

DRY_RUN=false
[[ "${1:-}" == "--dry-run" ]] && DRY_RUN=true

# The API key is only needed when actually applying to a cluster.
if ! $DRY_RUN; then
    : "${ANTHROPIC_API_KEY:?ANTHROPIC_API_KEY is required — set it to your Anthropic API key}"
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

kubectl create secret generic anthropic-credentials \
    --from-literal=api-key="$ANTHROPIC_API_KEY" \
    --namespace "$CLUSTER_NAME" \
    --dry-run=client -o yaml | kubectl apply -f -

kubectl kustomize "$TMPDIR" | kubectl apply -f -
