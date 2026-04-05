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
  instructions: You are an expert code reviewer.
personas:
  - name: p
    displayName: Test Persona
    description: A test persona.
    systemPrompt: You are a test agent.
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
    endpoint: http://gateway.default.svc.cluster.local:8000
EOF

HOME=/tmp/t1/home AGENT_NAME=test-agent \
  node /app/seed-config.mjs > /tmp/t1/out.txt 2>&1
clear_config

assert "settings.json created"          "[ -f /tmp/t1/home/.claude/settings.json ]"
assert "settings has model"             "grep -q 'claude-sonnet-4-5' /tmp/t1/home/.claude/settings.json"
assert "settings has ANTHROPIC_BASE_URL" "grep -q 'ANTHROPIC_BASE_URL' /tmp/t1/home/.claude/settings.json"
assert "base URL points to gateway"     "grep -q 'gateway.default.svc' /tmp/t1/home/.claude/settings.json"
assert "base URL has /anthropic path"   "grep -q '/anthropic' /tmp/t1/home/.claude/settings.json"
assert ".claude.json created"           "[ -f /tmp/t1/home/.claude.json ]"
assert "mcpServers present"             "grep -q 'mcpServers' /tmp/t1/home/.claude.json"
assert "tool endpoint in .claude.json"  "grep -q 'my-tool.default.svc' /tmp/t1/home/.claude.json"
assert "MCP type is http"               "grep -q '\"type\": \"http\"' /tmp/t1/home/.claude.json"
assert "CLAUDE.md created"             "[ -f /workspace/CLAUDE.md ]"
assert "CLAUDE.md has agent name"       "grep -q 'test-agent' /workspace/CLAUDE.md"
assert "CLAUDE.md has systemPrompt"    "grep -q 'You are a test agent' /workspace/CLAUDE.md"
assert "CLAUDE.md has instructions"    "grep -q 'Do the thing' /workspace/CLAUDE.md"
assert "CLAUDE.md has capabilities"    "grep -q 'research' /workspace/CLAUDE.md"
assert "CLAUDE.md has limitations"     "grep -q 'No speculation' /workspace/CLAUDE.md"
assert "agent card created"            "[ -f /tmp/t1/home/.claude/claude-a2a.config.json ]"
assert "agent card has name"           "grep -q 'test-agent' /tmp/t1/home/.claude/claude-a2a.config.json"

# ---------------------------------------------------------------------------
# Test 2: MCP servers merge-safe (existing .claude.json preserved)
# ---------------------------------------------------------------------------
echo "--- Test 2: merge-safe .claude.json ---"

set_config << 'EOF'
tools:
  new-tool:
    endpoint: http://new-tool.default.svc.cluster.local:8080
EOF

mkdir -p /tmp/t2/home/.claude
echo '{"userKey":"preserve-me","mcpServers":{"old-tool":{"type":"http","url":"http://old"}}}' \
  > /tmp/t2/home/.claude.json

HOME=/tmp/t2/home AGENT_NAME=test-agent \
  node /app/seed-config.mjs > /tmp/t2/out.txt 2>&1
clear_config

assert ".claude.json still exists"      "[ -f /tmp/t2/home/.claude.json ]"
assert "user key preserved"             "grep -q 'preserve-me' /tmp/t2/home/.claude.json"
assert "new tool present"               "grep -q 'new-tool' /tmp/t2/home/.claude.json"
assert "old mcpServer replaced"         "! grep -q 'old-tool' /tmp/t2/home/.claude.json"

# ---------------------------------------------------------------------------
# Test 3: env var fallback (no config.yaml)
# ---------------------------------------------------------------------------
echo "--- Test 3: env var fallback ---"

mkdir -p /tmp/t3/home/.claude

HOME=/tmp/t3/home AGENT_NAME=test-agent \
  MODEL_ENDPOINT=http://proxy.default.svc.cluster.local:8000 \
  LLM_MODEL=claude-sonnet-4-5 \
  node /app/seed-config.mjs > /tmp/t3/out.txt 2>&1

assert "settings.json created"          "[ -f /tmp/t3/home/.claude/settings.json ]"
assert "model from LLM_MODEL"           "grep -q 'claude-sonnet-4-5' /tmp/t3/home/.claude/settings.json"
assert "endpoint from MODEL_ENDPOINT"   "grep -q 'proxy.default.svc' /tmp/t3/home/.claude/settings.json"
assert "no CLAUDE.md (no personas)"     "[ ! -f /workspace/CLAUDE.md ] || ! grep -q 'test-agent' /workspace/CLAUDE.md"

# ---------------------------------------------------------------------------
# Test 4: no config.yaml, no env vars → graceful empty config
# ---------------------------------------------------------------------------
echo "--- Test 4: graceful empty (no config, no env vars) ---"

mkdir -p /tmp/t4/home/.claude

HOME=/tmp/t4/home AGENT_NAME=empty-agent \
  node /app/seed-config.mjs > /tmp/t4/out.txt 2>&1

assert "settings.json still created"    "[ -f /tmp/t4/home/.claude/settings.json ]"
assert "agent card still created"       "[ -f /tmp/t4/home/.claude/claude-a2a.config.json ]"
assert "agent card has name"            "grep -q 'empty-agent' /tmp/t4/home/.claude/claude-a2a.config.json"

# ---------------------------------------------------------------------------
echo ""
echo "Results: $PASS passed, $FAIL failed"
[ "$FAIL" -eq 0 ] || exit 1
