#!/usr/bin/env bash
# Usage: DOMAIN=example.com GITHUB_CLIENT_ID=<id> GITHUB_CLIENT_SECRET=<secret> \
#          [CLUSTER_NAME=my-cluster] bash install.sh [--dry-run]
#
# Prerequisites:
#   - GitHub OAuth App with callback URL: https://auth.${DOMAIN}/callback
#     Create one at https://github.com/settings/developers
set -euo pipefail

: "${DOMAIN:?DOMAIN is required (e.g. DOMAIN=example.com)}"
: "${GITHUB_CLIENT_ID:?GITHUB_CLIENT_ID is required}"
: "${GITHUB_CLIENT_SECRET:?GITHUB_CLIENT_SECRET is required}"
export CLUSTER_NAME="${CLUSTER_NAME:-my-cluster}"
export DOMAIN GITHUB_CLIENT_ID GITHUB_CLIENT_SECRET

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
else
    kubectl kustomize "$TMPDIR" | kubectl apply -f -
fi
