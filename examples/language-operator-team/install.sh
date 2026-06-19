#!/usr/bin/env bash
# Usage: CLUSTER_NAME=my-cluster GITHUB_TOKEN=ghp_... PROJECT_REPOSITORY=https://github.com/owner/repo.git bash install.sh [--dry-run]
set -euo pipefail

: "${CLUSTER_NAME:?CLUSTER_NAME is required — set it to the name of your LanguageCluster}"
: "${GITHUB_TOKEN:?GITHUB_TOKEN is required (gh PAT with repo, issues scopes)}"
: "${PROJECT_REPOSITORY:?PROJECT_REPOSITORY is required (clone URL, e.g. https://github.com/language-operator/language-operator.git)}"
export CLUSTER_NAME
export PROJECT_REPOSITORY
# Derive a human-readable default project name from the URL basename (strip trailing slash and .git).
_repo_base="${PROJECT_REPOSITORY%/}"; _repo_base="${_repo_base##*/}"; _repo_base="${_repo_base%.git}"
export PROJECT_NAME="${PROJECT_NAME:-$_repo_base}"

DRY_RUN=false
[[ "${1:-}" == "--dry-run" ]] && DRY_RUN=true

DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
TMPDIR="$(mktemp -d)"
trap 'rm -rf "$TMPDIR"' EXIT

# Restrict substitution to our install vars so literal shell references in agent
# instructions (e.g. $AGENT_REPO_DIR, injected by the operator at runtime) survive.
for f in "$DIR"/*.yaml; do
    envsubst '${CLUSTER_NAME} ${PROJECT_NAME} ${PROJECT_REPOSITORY}' < "$f" > "$TMPDIR/$(basename "$f")"
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

# Context7 API key is optional — the tool works without it at a lower rate limit
# and references the secret with optional:true, so the pod starts either way.
if [[ -n "${CONTEXT7_API_KEY:-}" ]]; then
    kubectl create secret generic context7-mcp-credentials \
        --from-literal=api-key="$CONTEXT7_API_KEY" \
        --namespace "$CLUSTER_NAME" \
        --dry-run=client -o yaml | kubectl apply -f -
fi

kubectl kustomize "$TMPDIR" | kubectl apply -f -
