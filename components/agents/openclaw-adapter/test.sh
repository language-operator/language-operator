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
personas:
  - name: p
    displayName: Test Persona
    systemPrompt: You are a test agent.
    tone: professional
    description: A test persona.
    instructions:
      - Do the thing
    capabilities:
      - research
    limitations:
      - No speculation
tools:
  my-tool:
    endpoint: http://my-tool.default.svc.cluster.local:8080
    protocol: mcp
models:
  claude-sonnet:
    model: claude-sonnet-4-5
    endpoint: http://claude-sonnet.default.svc.cluster.local:8000
EOF

mkdir -p /tmp/t1/state
AGENT_NAME=test-agent OPENCLAW_STATE_DIR=/tmp/t1/state \
  node /app/seed-config.mjs > /tmp/t1/out.txt 2>&1
clear_config

assert "openclaw.json created"         "[ -f /tmp/t1/state/openclaw.json ]"
assert "models.providers present"       "grep -q 'claude-sonnet' /tmp/t1/state/openclaw.json"
assert "correct baseUrl"               "grep -q 'claude-sonnet.default.svc' /tmp/t1/state/openclaw.json"
assert "api: openai-completions"       "grep -q 'openai-completions' /tmp/t1/state/openclaw.json"
assert "models array with id"          "grep -q 'claude-sonnet-4-5' /tmp/t1/state/openclaw.json"
assert "placeholder apiKey"            "grep -q 'sk-langop-proxy' /tmp/t1/state/openclaw.json"
assert "mcp.servers present"           "grep -q 'my-tool' /tmp/t1/state/openclaw.json"
assert "mcp server url"                "grep -q 'my-tool.default.svc' /tmp/t1/state/openclaw.json"
assert "agent identity name"           "grep -q 'test-agent' /tmp/t1/state/openclaw.json"
assert "AGENTS.md created"             "[ -f /tmp/t1/state/workspace/AGENTS.md ]"
assert "AGENTS.md has systemPrompt"    "grep -q 'You are a test agent' /tmp/t1/state/workspace/AGENTS.md"
assert "AGENTS.md has instructions"    "grep -q 'Do the thing' /tmp/t1/state/workspace/AGENTS.md"
assert "AGENTS.md has capabilities"    "grep -q 'research' /tmp/t1/state/workspace/AGENTS.md"
assert "AGENTS.md has limitations"     "grep -q 'No speculation' /tmp/t1/state/workspace/AGENTS.md"
assert "SOUL.md created"               "[ -f /tmp/t1/state/workspace/SOUL.md ]"
assert "SOUL.md has tone"              "grep -q 'professional' /tmp/t1/state/workspace/SOUL.md"
assert "SOUL.md has description"       "grep -q 'Test Persona' /tmp/t1/state/workspace/SOUL.md"

# ---------------------------------------------------------------------------
# Test 2: skip-if-exists (openclaw.json preserved across restarts)
# ---------------------------------------------------------------------------
echo "--- Test 2: skip-if-exists ---"

set_config << 'EOF'
personas:
  - name: p
    displayName: New Persona
    systemPrompt: New system prompt.
    tone: casual
EOF

mkdir -p /tmp/t2/state/workspace
echo '{"preserved":true}' > /tmp/t2/state/openclaw.json
echo "Old AGENTS" > /tmp/t2/state/workspace/AGENTS.md

AGENT_NAME=test-agent OPENCLAW_STATE_DIR=/tmp/t2/state \
  node /app/seed-config.mjs > /tmp/t2/out.txt 2>&1
clear_config

assert "openclaw.json not overwritten"  "grep -q 'preserved' /tmp/t2/state/openclaw.json"
assert "AGENTS.md overwritten"          "! grep -q 'Old AGENTS' /tmp/t2/state/workspace/AGENTS.md"

# ---------------------------------------------------------------------------
# Test 3: env var fallback (no config.yaml)
# ---------------------------------------------------------------------------
echo "--- Test 3: env var fallback ---"

mkdir -p /tmp/t3/state

AGENT_NAME=test-agent \
  MODEL_ENDPOINT=http://proxy.default.svc.cluster.local:8000 \
  LLM_MODEL=claude-sonnet-4-5 \
  OPENCLAW_STATE_DIR=/tmp/t3/state \
  node /app/seed-config.mjs > /tmp/t3/out.txt 2>&1

assert "openclaw.json created"          "[ -f /tmp/t3/state/openclaw.json ]"
assert "providers populated from env"   "grep -q 'proxy.default.svc' /tmp/t3/state/openclaw.json"
assert "model name from LLM_MODEL"      "grep -q 'claude-sonnet-4-5' /tmp/t3/state/openclaw.json"
assert "no AGENTS.md (no personas)"     "[ ! -f /tmp/t3/state/workspace/AGENTS.md ]"

# ---------------------------------------------------------------------------
# Test 4: no config.yaml, no env vars → graceful empty config
# ---------------------------------------------------------------------------
echo "--- Test 4: graceful empty (no config, no env vars) ---"

mkdir -p /tmp/t4/state

AGENT_NAME=test-agent OPENCLAW_STATE_DIR=/tmp/t4/state \
  node /app/seed-config.mjs > /tmp/t4/out.txt 2>&1

assert "openclaw.json still created"    "[ -f /tmp/t4/state/openclaw.json ]"
assert "only identity in config"        "grep -q 'test-agent' /tmp/t4/state/openclaw.json"

# ---------------------------------------------------------------------------
echo ""
echo "Results: $PASS passed, $FAIL failed"
[ "$FAIL" -eq 0 ] || exit 1
