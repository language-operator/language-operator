#!/usr/bin/env bash
# Deploys a self-contained Postgres (StatefulSet) and registers it as a LanguageTool, auto-wiring
# generated credentials into both. No external database required.
#
# Usage: CLUSTER_NAME=my-cluster bash install.sh [--dry-run]
set -euo pipefail

: "${CLUSTER_NAME:?CLUSTER_NAME is required — set it to the name of your LanguageCluster}"
export CLUSTER_NAME
export TOOL_NAME="${TOOL_NAME:-postgres}"

POSTGRES_USER="${POSTGRES_USER:-app}"
POSTGRES_DB="${POSTGRES_DB:-app}"
SECRET_NAME="postgres-mcp-credentials"

DRY_RUN=false
[[ "${1:-}" == "--dry-run" ]] && DRY_RUN=true

DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
TMPDIR="$(mktemp -d)"
trap 'rm -rf "$TMPDIR"' EXIT

# Restrict substitution to our placeholders so the literal $DATABASE_URL in the tool's stdio
# command survives to be expanded inside the container at runtime. $TOOL_NAME also covers the
# bundled ${TOOL_NAME}-db StatefulSet/Service/ConfigMap names.
for f in "$DIR"/*.yaml; do
    envsubst '$CLUSTER_NAME $TOOL_NAME' < "$f" > "$TMPDIR/$(basename "$f")"
done

if $DRY_RUN; then
    kubectl kustomize "$TMPDIR"
    exit 0
fi

# Generate the password once and reuse it on re-runs, so it never drifts from the password baked
# into the persisted PVC on first init.
PASSWORD="$(kubectl get secret "$SECRET_NAME" -n "$CLUSTER_NAME" \
    -o jsonpath='{.data.password}' 2>/dev/null | base64 -d || true)"
if [[ -z "$PASSWORD" ]]; then
    PASSWORD="$(openssl rand -hex 16)"
fi

DATABASE_URL="postgresql://${POSTGRES_USER}:${PASSWORD}@${TOOL_NAME}-db.${CLUSTER_NAME}.svc.cluster.local:5432/${POSTGRES_DB}"

# One secret is the single source of truth: the StatefulSet reads username/password/dbname; the
# LanguageTool's DATABASE_URL reads url.
kubectl create secret generic "$SECRET_NAME" \
    --from-literal=username="$POSTGRES_USER" \
    --from-literal=password="$PASSWORD" \
    --from-literal=dbname="$POSTGRES_DB" \
    --from-literal=url="$DATABASE_URL" \
    --namespace "$CLUSTER_NAME" \
    --dry-run=client -o yaml | kubectl apply -f -

kubectl kustomize "$TMPDIR" | kubectl apply -f -
