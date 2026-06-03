#!/usr/bin/env bash
# Usage: CLUSTER_NAME=my-cluster GITHUB_TOKEN=ghp_... PROJECT_REPOSITORY=owner/repo bash install.sh [--dry-run]
set -euo pipefail

: "${CLUSTER_NAME:?CLUSTER_NAME is required — set it to the name of your LanguageCluster}"
: "${GITHUB_TOKEN:?GITHUB_TOKEN is required (gh PAT with repo, issues scopes)}"
: "${PROJECT_REPOSITORY:?PROJECT_REPOSITORY is required (owner/repo, e.g. language-operator/language-operator)}"
export CLUSTER_NAME
export PROJECT_REPOSITORY
export PROJECT_REPO_NAME="${PROJECT_REPOSITORY##*/}"
export PROJECT_NAME="${PROJECT_NAME:-$PROJECT_REPO_NAME}"

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

kubectl create secret generic github-credentials \
    --from-literal=token="$GITHUB_TOKEN" \
    --namespace "$CLUSTER_NAME" \
    --dry-run=client -o yaml | kubectl apply -f -

if [[ -n "${ANTHROPIC_API_KEY:-}" ]]; then
    kubectl create secret generic anthropic-credentials \
        --from-literal=api-key="$ANTHROPIC_API_KEY" \
        --namespace "$CLUSTER_NAME" \
        --dry-run=client -o yaml | kubectl apply -f -
fi

if [[ -n "${CLAUDE_CODE_OAUTH_TOKEN:-}" ]]; then
    kubectl create secret generic claude-code-oauth \
        --from-literal=token="$CLAUDE_CODE_OAUTH_TOKEN" \
        --namespace "$CLUSTER_NAME" \
        --dry-run=client -o yaml | kubectl apply -f -
fi

kubectl kustomize "$TMPDIR" | kubectl apply -f -
