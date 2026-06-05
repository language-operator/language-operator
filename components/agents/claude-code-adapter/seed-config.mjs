/**
 * claude-code-adapter init container
 *
 * Bridges the language-operator config injection model to Claude Code's native
 * config format. Reads /etc/agent/config.yaml (injected by the operator) and
 * translates models and tools into the files Claude Code expects.
 *
 * Writes (merge-safe with existing files on the PVC):
 *   $CLAUDE_CONFIG_DIR/settings.json — model selection
 *   $CLAUDE_CONFIG_DIR/.claude.json  — mcpServers + onboarding/trust markers
 *
 * Persona and instructions are NOT written here — the operator injects them as
 * AGENT_PERSONA and AGENT_INSTRUCTIONS env vars, which the runtime's launcher
 * (launch-claude) passes to claude on startup.
 *
 * Authentication is interactive — users run `/login` inside the agent terminal,
 * OR provide CLAUDE_CODE_OAUTH_TOKEN for headless auth. Credentials live in
 * $CLAUDE_CONFIG_DIR/.credentials.json and persist on the workspace PVC.
 */

import { readFileSync, writeFileSync, existsSync, mkdirSync } from 'fs'
import { parse as parseYaml } from 'yaml'
import { homedir } from 'os'
import { join } from 'path'

const home = process.env.HOME ?? homedir()
const claudeDir = process.env.CLAUDE_CONFIG_DIR ?? join(home, '.claude')
const agentName = process.env.AGENT_NAME ?? ''

mkdirSync(claudeDir, { recursive: true })

// -------------------------------------------------------------------
// Read /etc/agent/config.yaml (operator-injected)
// -------------------------------------------------------------------
let operatorConfig = null
const operatorConfigPath = '/etc/agent/config.yaml'
if (existsSync(operatorConfigPath)) {
  try {
    operatorConfig = parseYaml(readFileSync(operatorConfigPath, 'utf8')) ?? {}
    console.log('Read operator config from /etc/agent/config.yaml')
  } catch (err) {
    console.warn(`Failed to parse /etc/agent/config.yaml: ${err.message}`)
  }
}

// -------------------------------------------------------------------
// settings.json — model selection; merge-safe (operator manages `model` only)
// -------------------------------------------------------------------
const configModels = operatorConfig?.models ?? {}
let modelId = null
for (const [, model] of Object.entries(configModels)) {
  modelId ??= model.model ?? null
}
if (!modelId) {
  const modelNames = (process.env.LLM_MODEL ?? '').split(',').map(s => s.trim()).filter(Boolean)
  if (modelNames.length > 0) {
    modelId = modelNames[0]
  }
}

const settingsPath = join(claudeDir, 'settings.json')
let settings = {}
if (existsSync(settingsPath)) {
  try {
    settings = JSON.parse(readFileSync(settingsPath, 'utf8'))
  } catch (err) {
    console.warn(`Failed to parse existing settings.json: ${err.message} — overwriting`)
    settings = {}
  }
}
if (modelId) {
  settings.model = modelId
  console.log(`Set settings.model = ${modelId}`)
} else {
  delete settings.model
}

// Force terminal_bell notifications so claude emits BEL (\x07) when waiting
// for input. The xterm.js web terminal listens for BEL via term.onBell to
// prefix the browser tab title with a ✦ glyph. Without this, claude's default
// 'auto' channel emits no signal in our headless TTY and the title indicator
// never updates.
settings.preferredNotifChannel = 'terminal_bell'

writeFileSync(settingsPath, JSON.stringify(settings, null, 2))

// -------------------------------------------------------------------
// .claude.json — MCP server entries; merge-safe with existing
// Operator manages mcpServers key only; all other keys are preserved.
// -------------------------------------------------------------------
const configTools = operatorConfig?.tools ?? {}
const mcpServers = {}

for (const [toolName, tool] of Object.entries(configTools)) {
  if (!tool.endpoint) {
    console.warn(`Tool '${toolName}' has no endpoint — skipping`)
    continue
  }
  if (!tool.endpoint.startsWith('http://') && !tool.endpoint.startsWith('https://')) {
    console.warn(`Tool '${toolName}' endpoint '${tool.endpoint}' is not an HTTP URL — skipping`)
    continue
  }
  mcpServers[toolName] = { type: 'http', url: tool.endpoint }
  console.log(`Configured MCP server '${toolName}' → ${tool.endpoint}`)
}

const claudeJsonPath = process.env.CLAUDE_CONFIG_DIR
  ? join(claudeDir, '.claude.json')
  : join(home, '.claude.json')
let claudeJson = {}
if (existsSync(claudeJsonPath)) {
  try {
    claudeJson = JSON.parse(readFileSync(claudeJsonPath, 'utf8'))
    console.log('Merging into existing .claude.json')
  } catch (err) {
    console.warn(`Failed to parse existing .claude.json: ${err.message} — overwriting`)
    claudeJson = {}
  }
}

if (Object.keys(mcpServers).length > 0) {
  claudeJson.mcpServers = mcpServers
  console.log(`Updated mcpServers with ${Object.keys(mcpServers).length} tool(s)`)
} else {
  delete claudeJson.mcpServers
}

// Pre-trust /workspace so Claude Code doesn't prompt on first invocation. The
// trust check walks up the directory tree, so trusting /workspace covers any
// subdirectory the agent cd's into (e.g. /workspace/<repo> for the dev-team).
// In the operator model, the workspace was provisioned for this agent by the
// user who deployed the LanguageAgent — trust is implicit.
claudeJson.projects = claudeJson.projects ?? {}
claudeJson.projects['/workspace'] = {
  ...(claudeJson.projects['/workspace'] ?? {}),
  hasTrustDialogAccepted: true,
  hasCompletedProjectOnboarding: true,
}

// Pre-clear Claude Code's first-run wizard for headless agents. When
// CLAUDE_CODE_OAUTH_TOKEN authenticates (no /login flow ever runs), Claude Code's
// UI still treats the agent as "not onboarded" because .claude.json has no
// `oauthAccount` block. Seed both markers so the wizard skips and Claude drops
// straight into the prompt. The stubbed values are placeholders — Claude uses
// them for telemetry/display only; the real authentication is the OAuth token.
if (process.env.CLAUDE_CODE_OAUTH_TOKEN) {
  claudeJson.hasCompletedOnboarding = true
  if (!claudeJson.oauthAccount) {
    const now = new Date().toISOString()
    claudeJson.oauthAccount = {
      accountUuid: '00000000-0000-0000-0000-000000000000',
      emailAddress: `${agentName || 'agent'}@language-operator.local`,
      organizationUuid: '00000000-0000-0000-0000-000000000000',
      hasExtraUsageEnabled: false,
      billingType: 'stripe_subscription',
      accountCreatedAt: now,
      subscriptionCreatedAt: now,
      ccOnboardingFlags: {},
      claudeCodeTrialEndsAt: null,
      claudeCodeTrialDurationDays: null,
      seatTier: null,
      displayName: agentName || 'agent',
      organizationRole: 'admin',
      workspaceRole: null,
      organizationName: agentName || 'agent',
    }
    console.log('Seeded oauthAccount stub to skip onboarding wizard (CLAUDE_CODE_OAUTH_TOKEN auth)')
  }
}

writeFileSync(claudeJsonPath, JSON.stringify(claudeJson, null, 2))
