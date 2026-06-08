#!/usr/bin/env bash
# Usage: CLUSTER_NAME=my-cluster ENDPOINT=http://my-model.ns.svc:8000/v1 MODEL_ID=my-model \
#        [API_KEY=sk-...] bash install.sh [--dry-run]
set -euo pipefail

: "${CLUSTER_NAME:?CLUSTER_NAME is required — set it to the name of your LanguageCluster}"
export CLUSTER_NAME
export MODEL_NAME="${MODEL_NAME:-generic-model}"
export ENDPOINT="${ENDPOINT:-http://my-model.my-namespace.svc.cluster.local:8000/v1}"
export MODEL_ID="${MODEL_ID:-my-model}"

DRY_RUN=false
[[ "${1:-}" == "--dry-run" ]] && DRY_RUN=true

# An API key is optional — many self-hosted servers (Ollama, vLLM, …) need no auth.
# When provided, render the apiKeySecretRef block and create the backing secret.
if [[ -n "${API_KEY:-}" ]]; then
    export API_KEY_BLOCK=$'  apiKeySecretRef:\n    name: generic-model-credentials\n    key: api-key'
else
    export API_KEY_BLOCK=""
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

if [[ -n "${API_KEY:-}" ]]; then
    kubectl create secret generic generic-model-credentials \
        --from-literal=api-key="$API_KEY" \
        --namespace "$CLUSTER_NAME" \
        --dry-run=client -o yaml | kubectl apply -f -
fi

kubectl kustomize "$TMPDIR" | kubectl apply -f -
