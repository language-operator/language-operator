#!/bin/sh
# Runs inside the container image to verify seed-config.mjs behaviour.
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

set_config() {
  mkdir -p /etc/agent
  cat > /etc/agent/config.yaml
}

clear_config() {
  rm -f /etc/agent/config.yaml
}

# ---------------------------------------------------------------------------
# Test 1: full config.yaml mapping
# ---------------------------------------------------------------------------
echo "--- Test 1: full config.yaml mapping ---"

set_config << 'EOF'
agent:
  name: test-agent
tools:
  my-tool:
    endpoint: http://my-tool.default.svc.cluster.local:8080
    protocol: mcp
models:
  claude-sonnet:
    model: claude-sonnet-4-5
    endpoint: http://gateway.default.svc.cluster.local:8000
EOF

mkdir -p /tmp/t1
OPENCODE_CONFIG_DIR=/tmp/t1 \
  node /app/seed-config.mjs > /tmp/t1/out.txt 2>&1
clear_config

assert "opencode.jsonc created"          "[ -f /tmp/t1/opencode.jsonc ]"
assert "provider present"                "grep -q 'provider' /tmp/t1/opencode.jsonc"
assert "provider key is openai"          "grep -q '\"openai\"' /tmp/t1/opencode.jsonc"
assert "options.baseURL set"             "grep -q 'baseURL' /tmp/t1/opencode.jsonc"
assert "correct baseURL value"           "grep -q 'gateway.default.svc.*8000/v1' /tmp/t1/opencode.jsonc"
assert "model id as record key"          "grep -q 'claude-sonnet-4-5' /tmp/t1/opencode.jsonc"
assert "placeholder apiKey"              "grep -q 'sk-langop-proxy' /tmp/t1/opencode.jsonc"
assert "mcp section present"             "grep -q 'mcp' /tmp/t1/opencode.jsonc"
assert "mcp tool key"                    "grep -q 'my-tool' /tmp/t1/opencode.jsonc"
assert "mcp type: remote"                "grep -q 'remote' /tmp/t1/opencode.jsonc"
assert "mcp tool url"                    "grep -q 'my-tool.default.svc' /tmp/t1/opencode.jsonc"
assert "autoupdate: false"               "grep -q 'autoupdate' /tmp/t1/opencode.jsonc"

# ---------------------------------------------------------------------------
# Test 2: always overwrites (no skip-if-exists — config is operator-managed)
# ---------------------------------------------------------------------------
echo "--- Test 2: always overwrites existing config ---"

set_config << 'EOF'
models:
  new-model:
    model: gpt-4o
    endpoint: http://gateway.default.svc.cluster.local:8000
EOF

mkdir -p /tmp/t2
echo '{"old": true}' > /tmp/t2/opencode.jsonc

OPENCODE_CONFIG_DIR=/tmp/t2 \
  node /app/seed-config.mjs > /tmp/t2/out.txt 2>&1
clear_config

assert "config overwritten"              "! grep -q '\"old\"' /tmp/t2/opencode.jsonc"
assert "new model present"               "grep -q 'gpt-4o' /tmp/t2/opencode.jsonc"

# ---------------------------------------------------------------------------
# Test 3: env var fallback (no config.yaml)
# ---------------------------------------------------------------------------
echo "--- Test 3: env var fallback ---"

mkdir -p /tmp/t3

MODEL_ENDPOINT=http://gateway.default.svc.cluster.local:8000 \
  LLM_MODEL=claude-sonnet-4-5 \
  OPENCODE_CONFIG_DIR=/tmp/t3 \
  node /app/seed-config.mjs > /tmp/t3/out.txt 2>&1

assert "opencode.jsonc created"          "[ -f /tmp/t3/opencode.jsonc ]"
assert "provider key is openai"          "grep -q '\"openai\"' /tmp/t3/opencode.jsonc"
assert "provider populated from env"     "grep -q 'gateway.default.svc.*8000/v1' /tmp/t3/opencode.jsonc"
assert "model name from LLM_MODEL"       "grep -q 'claude-sonnet-4-5' /tmp/t3/opencode.jsonc"

# ---------------------------------------------------------------------------
# Test 4: no config.yaml, no env vars → graceful minimal config
# ---------------------------------------------------------------------------
echo "--- Test 4: graceful empty (no config, no env vars) ---"

mkdir -p /tmp/t4

OPENCODE_CONFIG_DIR=/tmp/t4 \
  node /app/seed-config.mjs > /tmp/t4/out.txt 2>&1

assert "opencode.jsonc still created"    "[ -f /tmp/t4/opencode.jsonc ]"
assert "autoupdate: false present"       "grep -q 'autoupdate' /tmp/t4/opencode.jsonc"

# ---------------------------------------------------------------------------
echo ""
echo "Results: $PASS passed, $FAIL failed"
[ "$FAIL" -eq 0 ] || exit 1
