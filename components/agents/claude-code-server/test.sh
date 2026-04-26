#!/bin/sh
# Smoke test: verify ttyd and claude binaries are present and executable.
set -e

ttyd --version >/dev/null 2>&1 || { echo "FAIL: ttyd not found"; exit 1; }
echo "PASS: ttyd present"

claude --version >/dev/null 2>&1 || { echo "FAIL: claude not found"; exit 1; }
echo "PASS: claude present"

echo "All tests passed"
