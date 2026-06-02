#!/usr/bin/env bash
# Usage: DOMAIN=example.com [ADMIN_PASSWORD=...] [ADMIN_EMAIL=...] bash install.sh [--dry-run]
#
# Requires: python3 with the 'bcrypt' package (pip install bcrypt) to hash ADMIN_PASSWORD.
# Escape hatch: set ADMIN_PASSWORD_HASH directly to skip hashing (useful in CI or when
#   python3/bcrypt is unavailable).
set -euo pipefail

: "${DOMAIN:?DOMAIN is required (e.g. DOMAIN=example.com)}"
export CLUSTER_NAME="${CLUSTER_NAME:-my-cluster}"
export DOMAIN
export ADMIN_EMAIL="${ADMIN_EMAIL:-admin@example.com}"
export ADMIN_USERNAME="${ADMIN_USERNAME:-admin}"

if [[ -z "${ADMIN_PASSWORD_HASH:-}" ]]; then
    export ADMIN_PASSWORD="${ADMIN_PASSWORD:-password}"
    if ! python3 -c "import bcrypt" 2>/dev/null; then
        echo "error: python3 with the 'bcrypt' package is required to hash ADMIN_PASSWORD."
        echo "  Install: pip install bcrypt"
        echo "  Or pre-compute the hash and pass ADMIN_PASSWORD_HASH directly."
        exit 1
    fi
    ADMIN_PASSWORD_HASH="$(python3 - <<'PYEOF'
import bcrypt, os
pw = os.environ['ADMIN_PASSWORD'].encode()
print(bcrypt.hashpw(pw, bcrypt.gensalt(10)).decode())
PYEOF
)"
fi
export ADMIN_PASSWORD_HASH

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
