/**
 * openclaw-adapter init container
 *
 * Bridges the language-operator config injection model to openclaw's native
 * config format. Reads /etc/agent/config.yaml (injected by the operator) and
 * translates models, tools, and personas into openclaw.json and workspace
 * bootstrap files.
 *
 * openclaw.json is written only on first run (skip-if-exists) to preserve
 * user runtime state across pod restarts. Bootstrap files (AGENTS.md, SOUL.md)
 * are always overwritten — they are operator-managed configuration, not user state.
 */

import { readFileSync, writeFileSync, existsSync, mkdirSync } from 'fs'
import { parse as parseYaml } from 'yaml'

const stateDir = process.env.OPENCLAW_STATE_DIR ?? '/workspace/.openclaw'
const configFile = `${stateDir}/openclaw.json`
const workspaceDir = `${stateDir}/workspace`
const agentName = process.env.AGENT_NAME ?? ''

mkdirSync(stateDir, { recursive: true })
mkdirSync(workspaceDir, { recursive: true })

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
// Bootstrap files: AGENTS.md and SOUL.md (always overwrite)
// These are operator-managed — persona changes should be reflected
// on every pod restart.
// -------------------------------------------------------------------
const personas = operatorConfig?.personas ?? []

if (personas.length > 0) {
  const agentsSections = personas.map((persona) => {
    const lines = []
    if (persona.systemPrompt) {
      lines.push(persona.systemPrompt)
    }
    if (persona.instructions?.length) {
      lines.push('\n## Instructions')
      for (const instruction of persona.instructions) {
        lines.push(`- ${instruction}`)
      }
    }
    if (persona.capabilities?.length) {
      lines.push('\n## Capabilities')
      for (const capability of persona.capabilities) {
        lines.push(`- ${capability}`)
      }
    }
    if (persona.limitations?.length) {
      lines.push('\n## Limitations')
      for (const limitation of persona.limitations) {
        lines.push(`- ${limitation}`)
      }
    }
    return lines.join('\n')
  })

  const agentsMd = `# Agent Instructions\n\n${agentsSections.join('\n\n---\n\n')}\n`
  writeFileSync(`${workspaceDir}/AGENTS.md`, agentsMd)
  console.log('Wrote AGENTS.md from persona configuration')

  // SOUL.md — tone and character, if any persona has description/tone
  const soulPersonas = personas.filter(p => p.tone || p.description)
  if (soulPersonas.length > 0) {
    const soulSections = soulPersonas.map(persona => {
      const lines = []
      const header = persona.displayName ?? persona.name ?? 'Persona'
      if (persona.description) {
        lines.push(`# ${header}\n\n${persona.description}`)
      } else {
        lines.push(`# ${header}`)
      }
      if (persona.tone) {
        lines.push(`\n**Tone:** ${persona.tone}`)
      }
      return lines.join('\n')
    })
    const soulMd = soulSections.join('\n\n---\n\n') + '\n'
    writeFileSync(`${workspaceDir}/SOUL.md`, soulMd)
    console.log('Wrote SOUL.md from persona tone/description')
  }
} else {
  console.log('No personas in config.yaml — skipping AGENTS.md / SOUL.md')
}

// -------------------------------------------------------------------
// openclaw.json — skip if already exists (preserve user state)
// -------------------------------------------------------------------
if (existsSync(configFile)) {
  console.log(`openclaw.json already exists at ${configFile}, skipping seed`)
  process.exit(0)
}

// -------------------------------------------------------------------
// Build models.providers from config.yaml models section.
// Fall back to MODEL_ENDPOINTS / LLM_MODEL env vars if absent.
// -------------------------------------------------------------------
const configModels = operatorConfig?.models ?? {}
const providers = {}

if (Object.keys(configModels).length > 0) {
  // Primary source: config.yaml models section
  // Each key is the LanguageModel CRD name; value has .provider, .model, .endpoint
  for (const [crdName, model] of Object.entries(configModels)) {
    if (!model.endpoint) {
      console.warn(`Model '${crdName}' has no endpoint — skipping`)
      continue
    }
    providers[crdName] = {
      baseUrl: model.endpoint,
      apiKey: 'sk-langop-proxy',  // placeholder; LiteLLM proxy handles real auth
      api: 'openai-completions',   // LiteLLM exposes OpenAI-compatible API
      models: [
        { id: model.model ?? crdName, name: model.model ?? crdName },
      ],
    }
    console.log(`Configured model provider '${crdName}' → ${model.endpoint}`)
  }
} else {
  // Fallback: zip MODEL_ENDPOINTS + LLM_MODEL env vars
  const endpoints = (process.env.MODEL_ENDPOINTS ?? '').split(',').map(s => s.trim()).filter(Boolean)
  const modelNames = (process.env.LLM_MODEL ?? '').split(',').map(s => s.trim()).filter(Boolean)

  if (endpoints.length === 0) {
    console.warn('MODEL_ENDPOINTS is not set and config.yaml has no models — seeding without model config')
  }

  for (let i = 0; i < endpoints.length; i++) {
    const providerKey = modelNames[i] ?? `model-${i}`
    const modelId = modelNames[i] ?? providerKey
    providers[providerKey] = {
      baseUrl: endpoints[i],
      apiKey: 'sk-langop-proxy',
      api: 'openai-completions',
      models: [{ id: modelId, name: modelId }],
    }
    console.log(`Configured model provider '${providerKey}' → ${endpoints[i]} (from env vars)`)
  }
}

// -------------------------------------------------------------------
// Build mcp.servers from config.yaml tools section
// -------------------------------------------------------------------
const configTools = operatorConfig?.tools ?? {}
const mcpServers = {}

for (const [toolName, tool] of Object.entries(configTools)) {
  if (!tool.endpoint) {
    console.warn(`Tool '${toolName}' has no endpoint — skipping`)
    continue
  }
  // Map all tools (default protocol is mcp; skip non-http entries)
  if (!tool.endpoint.startsWith('http://') && !tool.endpoint.startsWith('https://')) {
    console.warn(`Tool '${toolName}' endpoint '${tool.endpoint}' is not an HTTP URL — skipping`)
    continue
  }
  mcpServers[toolName] = { url: tool.endpoint }
  console.log(`Configured MCP server '${toolName}' → ${tool.endpoint}`)
}

// -------------------------------------------------------------------
// Assemble and write openclaw.json
// -------------------------------------------------------------------
const config = {}

if (Object.keys(providers).length > 0) {
  config.models = { providers }
}

if (Object.keys(mcpServers).length > 0) {
  config.mcp = { servers: mcpServers }
}

if (agentName) {
  config.agents = {
    defaults: {
      identity: { name: agentName },
    },
  }
}

writeFileSync(configFile, JSON.stringify(config, null, 2))
console.log(`Seeded openclaw.json at ${configFile}`)
