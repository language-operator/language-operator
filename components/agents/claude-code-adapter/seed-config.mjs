/**
 * claude-code-adapter init container
 *
 * Bridges the language-operator config injection model to Claude Code's native
 * config format. Reads /etc/agent/config.yaml (injected by the operator) and
 * translates models, tools, and personas into the files Claude Code expects.
 *
 * Writes (merge-safe with existing files on the PVC):
 *   $CLAUDE_CONFIG_DIR/settings.json — model selection
 *   $CLAUDE_CONFIG_DIR/.claude.json  — mcpServers entries
 *   /workspace/CLAUDE.md             — persona/instructions as markdown
 *
 * Authentication is interactive — users run `/login` inside the agent terminal.
 * Credentials live in $CLAUDE_CONFIG_DIR/.credentials.json (written by Claude
 * Code itself) and persist on the workspace PVC.
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

writeFileSync(claudeJsonPath, JSON.stringify(claudeJson, null, 2))

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
}
