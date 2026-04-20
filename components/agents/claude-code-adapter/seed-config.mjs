/**
 * claude-code-adapter init container
 *
 * Bridges the language-operator config injection model to Claude Code's native
 * config format. Reads /etc/agent/config.yaml (injected by the operator) and
 * translates models, tools, and personas into the files Claude Code expects.
 *
 * Writes:
 *   $HOME/.claude.json          — MCP server entries; merge-safe with existing
 *   $HOME/.claude/settings.json — model and ANTHROPIC_BASE_URL; always overwrite
 *   /workspace/CLAUDE.md        — persona/instructions as markdown; always overwrite
 *   $HOME/.claude/claude-a2a.config.json — agent card for A2A server; always overwrite
 */

import { readFileSync, writeFileSync, existsSync, mkdirSync } from 'fs'
import { parse as parseYaml } from 'yaml'
import { homedir } from 'os'
import { join } from 'path'

const home = process.env.HOME ?? homedir()
const claudeDir = join(home, '.claude')
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
// $HOME/.claude/settings.json — model selection and gateway URL
// Always overwrite — this is operator-managed configuration.
// -------------------------------------------------------------------
const configModels = operatorConfig?.models ?? {}
let modelId = null
let gatewayEndpoint = null

if (Object.keys(configModels).length > 0) {
  for (const [, model] of Object.entries(configModels)) {
    if (!model.endpoint) continue
    gatewayEndpoint ??= model.endpoint
    modelId ??= model.model ?? null
  }
} else {
  // Fallback: MODEL_ENDPOINT / LLM_MODEL env vars
  const endpoints = (process.env.MODEL_ENDPOINT ?? '').split(',').map(s => s.trim()).filter(Boolean)
  const modelNames = (process.env.LLM_MODEL ?? '').split(',').map(s => s.trim()).filter(Boolean)
  if (endpoints.length > 0) {
    gatewayEndpoint = endpoints[0]
  }
  if (modelNames.length > 0) {
    modelId = modelNames[0]
  }
}

const settings = {}
if (modelId) {
  settings.model = modelId
}
if (gatewayEndpoint) {
  // Claude Code expects /anthropic path on the base URL so the SDK hits
  // gateway/anthropic/v1/messages — LiteLLM routes this to real Anthropic.
  // A placeholder API key is required to satisfy SDK validation; the gateway
  // substitutes the real credential so the value sent here doesn't matter.
  const base = gatewayEndpoint.replace(/\/+$/, '')
  settings.env = {
    ANTHROPIC_BASE_URL: `${base}/anthropic`,
    ANTHROPIC_API_KEY: 'sk-langop-proxy',
  }
  console.log(`Configured ANTHROPIC_BASE_URL → ${settings.env.ANTHROPIC_BASE_URL}`)
} else {
  // No gateway — use a directly-injected API key if available.
  // The operator injects claudeCode.apiKeyRef secrets via envFrom on the main
  // container; for direct-API mode without a gateway, pass the key as an
  // explicit env var named ANTHROPIC_API_KEY on the agent's deployment.env.
  const apiKey = process.env.ANTHROPIC_API_KEY ?? ''
  if (apiKey) {
    settings.env = { ANTHROPIC_API_KEY: apiKey }
    console.log('Configured ANTHROPIC_API_KEY for direct API access')
  } else {
    console.warn('No gateway and no ANTHROPIC_API_KEY — Claude Code will need credentials another way')
  }
}

writeFileSync(join(claudeDir, 'settings.json'), JSON.stringify(settings, null, 2))
console.log(`Wrote settings.json to ${claudeDir}/settings.json`)

// -------------------------------------------------------------------
// $HOME/.claude.json — MCP server entries; merge-safe with existing
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

const claudeJsonPath = join(home, '.claude.json')
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
  console.log('No tools in config.yaml — cleared mcpServers')
}

writeFileSync(claudeJsonPath, JSON.stringify(claudeJson, null, 2))
console.log(`Wrote .claude.json to ${claudeJsonPath}`)

// -------------------------------------------------------------------
// /workspace/CLAUDE.md — persona instructions and agent context
// Always overwrite — operator-managed, persona changes reflected on restart.
// -------------------------------------------------------------------
const personas = operatorConfig?.personas ?? []
const instructions = operatorConfig?.agent?.instructions ?? operatorConfig?.instructions ?? null

const claudeMdLines = []

if (agentName) {
  claudeMdLines.push(`# ${agentName}`)
  claudeMdLines.push('')
}

if (instructions) {
  claudeMdLines.push(instructions)
  claudeMdLines.push('')
}

if (personas.length > 0) {
  for (const persona of personas) {
    if (persona.systemPrompt) {
      claudeMdLines.push(persona.systemPrompt)
      claudeMdLines.push('')
    }
    if (persona.instructions?.length) {
      claudeMdLines.push('## Instructions')
      for (const instruction of persona.instructions) {
        claudeMdLines.push(`- ${instruction}`)
      }
      claudeMdLines.push('')
    }
    if (persona.capabilities?.length) {
      claudeMdLines.push('## Capabilities')
      for (const capability of persona.capabilities) {
        claudeMdLines.push(`- ${capability}`)
      }
      claudeMdLines.push('')
    }
    if (persona.limitations?.length) {
      claudeMdLines.push('## Limitations')
      for (const limitation of persona.limitations) {
        claudeMdLines.push(`- ${limitation}`)
      }
      claudeMdLines.push('')
    }
  }
}

if (claudeMdLines.length > 0) {
  mkdirSync('/workspace', { recursive: true })
  writeFileSync('/workspace/CLAUDE.md', claudeMdLines.join('\n'))
  console.log('Wrote /workspace/CLAUDE.md from persona/instructions configuration')
} else {
  console.log('No persona/instructions in config.yaml — skipping CLAUDE.md')
}

// -------------------------------------------------------------------
// $HOME/.claude/.credentials.json — OAuth credentials for subscription billing
// Written when CLAUDE_OAUTH_CREDENTIALS env var is set (JSON blob from secret).
// When present, the SDK uses OAuth flow instead of ANTHROPIC_API_KEY.
// -------------------------------------------------------------------
const oauthCredentials = process.env.CLAUDE_OAUTH_CREDENTIALS ?? ''
if (oauthCredentials) {
  try {
    JSON.parse(oauthCredentials) // validate before writing
    writeFileSync(join(claudeDir, '.credentials.json'), oauthCredentials)
    console.log('Wrote .credentials.json for OAuth subscription billing')
  } catch (err) {
    console.warn(`Failed to write .credentials.json: ${err.message}`)
  }
}

// -------------------------------------------------------------------
// $HOME/.claude/claude-a2a.config.json — agent card for A2A server
// Always overwrite — operator-managed.
// -------------------------------------------------------------------
const agentInstructions = operatorConfig?.agent?.instructions ?? operatorConfig?.instructions ?? ''
const agentCard = {
  name: agentName || 'claude-code-agent',
  description: (personas[0]?.description ?? agentInstructions.slice(0, 200)) || 'Claude Code agent',
}

writeFileSync(join(claudeDir, 'claude-a2a.config.json'), JSON.stringify(agentCard, null, 2))
console.log(`Wrote claude-a2a.config.json for agent '${agentCard.name}'`)
