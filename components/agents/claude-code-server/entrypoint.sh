#!/bin/sh
set -e

PORT="${PORT:-8080}"

if [ -n "${TTYD_USERNAME}" ] && [ -n "${TTYD_PASSWORD}" ]; then
    exec ttyd \
        -W \
        -p "${PORT}" \
        -c "${TTYD_USERNAME}:${TTYD_PASSWORD}" \
        -w /workspace \
        claude
else
    exec ttyd \
        -W \
        -p "${PORT}" \
        -w /workspace \
        claude
fi
