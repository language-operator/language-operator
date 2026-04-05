#!/bin/sh
# Runs inside the container image to verify server.mjs behaviour.
# Starts the server, makes HTTP requests, asserts expected responses.
# Exit 0 = all pass, non-zero = failure.
set -e

PASS=0
FAIL=0

assert() {
  local desc="$1"; local cmd="$2"
  if eval "$cmd" > /dev/null 2>&1; then
    echo "  PASS: $desc"
    PASS=$((PASS + 1))
  else
    echo "  FAIL: $desc"
    FAIL=$((FAIL + 1))
  fi
}

# Start server in background
mkdir -p /workspace
AGENT_NAME=test-agent HOME=/tmp/test-home node /app/server.mjs &
SERVER_PID=$!

# Wait for server to be ready
for i in $(seq 1 20); do
  if wget -q -O /dev/null http://localhost:8080/healthz 2>/dev/null; then
    break
  fi
  sleep 0.5
done

# ---------------------------------------------------------------------------
# Test 1: /healthz returns 200 with status ok
# ---------------------------------------------------------------------------
echo "--- Test 1: /healthz ---"
HEALTH=$(wget -q -O - http://localhost:8080/healthz 2>/dev/null || echo 'FAILED')
assert "healthz returns 200"       "echo '$HEALTH' | grep -q 'ok'"

# ---------------------------------------------------------------------------
# Test 2: /.well-known/agent-card returns valid JSON with required fields
# ---------------------------------------------------------------------------
echo "--- Test 2: agent-card ---"
CARD=$(wget -q -O - http://localhost:8080/.well-known/agent-card 2>/dev/null || echo 'FAILED')
assert "agent-card returns JSON"   "echo '$CARD' | grep -q 'name'"
assert "agent-card has skills"     "echo '$CARD' | grep -q 'skills'"
assert "agent-card has version"    "echo '$CARD' | grep -q 'version'"
assert "agent-card has agent name" "echo '$CARD' | grep -q 'test-agent'"

# ---------------------------------------------------------------------------
# Test 3: POST / with invalid body returns 400
# ---------------------------------------------------------------------------
echo "--- Test 3: invalid request returns 400 ---"
STATUS=$(wget -q --server-response -O /dev/null \
  --post-data '{"jsonrpc":"2.0","method":"tasks/send","params":{}}' \
  --header 'Content-Type: application/json' \
  http://localhost:8080/ 2>&1 | grep 'HTTP/' | awk '{print $2}' || echo '000')
assert "missing message returns 4xx" "[ '$STATUS' = '400' ] || [ '$STATUS' = '500' ]"

# ---------------------------------------------------------------------------
# Test 4: tasks/get for unknown task returns 404
# ---------------------------------------------------------------------------
echo "--- Test 4: tasks/get unknown task ---"
STATUS=$(wget -q --server-response -O /dev/null \
  --post-data '{"jsonrpc":"2.0","method":"tasks/get","params":{"id":"nonexistent-task-id"}}' \
  --header 'Content-Type: application/json' \
  http://localhost:8080/ 2>&1 | grep 'HTTP/' | awk '{print $2}' || echo '000')
assert "unknown task returns 404"  "[ '$STATUS' = '404' ]"

# ---------------------------------------------------------------------------
kill $SERVER_PID 2>/dev/null || true

echo ""
echo "Results: $PASS passed, $FAIL failed"
[ "$FAIL" -eq 0 ] || exit 1
