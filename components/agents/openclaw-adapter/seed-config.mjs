/**
 * openclaw-adapter init container
 *
 * Bridges the language-operator config injection model to openclaw's native
 * config format. Reads /etc/agent/config.yaml (injected by the operator) and
 * translates models, tools, and personas into openclaw.json and workspace
 * bootstrap files.
 *
 * On subsequent runs (openclaw.json already exists), only operator-managed sections
 * are updated: gateway config and mcp.servers. All other user runtime state is
 * preserved. Bootstrap files (AGENTS.md, SOUL.md) are always overwritten.
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
// Gateway config — always written (operator-managed, not user state).
// When the gateway binds to 0.0.0.0 (--bind lan), openclaw requires
// either explicit allowedOrigins or dangerouslyAllowHostHeaderOriginFallback.
// We use the Host-header fallback: safe in-cluster because only Traefik
// reaches the pod (enforced by NetworkPolicy) and Traefik always sends
// the correct Host header.
// -------------------------------------------------------------------
const gatewayConfig = {
  gateway: {
    controlUi: {
      dangerouslyAllowHostHeaderOriginFallback: true,
      dangerouslyDisableDeviceAuth: true,
    },
  },
}

// -------------------------------------------------------------------
// Build mcp.servers from config.yaml tools section.
// Done before the existsSync branch so it is available in both the
// merge path (existing file) and the new-file path below.
// mcp.servers is operator-managed — always overwrite, never preserve.
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
  mcpServers[toolName] = { url: tool.endpoint }
  console.log(`Configured MCP server '${toolName}' → ${tool.endpoint}`)
}

if (existsSync(configFile)) {
  console.log(`openclaw.json already exists at ${configFile}, merging gateway config and mcp.servers`)
  let existing = {}
  try {
    existing = JSON.parse(readFileSync(configFile, 'utf8'))
  } catch (err) {
    console.warn(`Failed to parse existing openclaw.json: ${err.message} — overwriting`)
  }
  // Deep-merge only the gateway section; preserve all other user state.
  existing.gateway = {
    ...(existing.gateway ?? {}),
    ...gatewayConfig.gateway,
    controlUi: {
      ...((existing.gateway ?? {}).controlUi ?? {}),
      ...gatewayConfig.gateway.controlUi,
    },
  }
  // Always overwrite mcp.servers — operator-managed, not user state.
  if (Object.keys(mcpServers).length > 0) {
    existing.mcp = { servers: mcpServers }
    console.log(`Updated mcp.servers with ${Object.keys(mcpServers).length} tool(s)`)
  } else {
    delete existing.mcp
    console.log('No tools in config.yaml — cleared mcp.servers')
  }
  writeFileSync(configFile, JSON.stringify(existing, null, 2))
  console.log('Merged gateway config and mcp.servers into existing openclaw.json')
  process.exit(0)
}

// -------------------------------------------------------------------
// Build models.providers from config.yaml models section.
// Fall back to MODEL_ENDPOINT / LLM_MODEL env vars if absent.
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
  // Fallback: zip MODEL_ENDPOINT + LLM_MODEL env vars
  const endpoints = (process.env.MODEL_ENDPOINT ?? '').split(',').map(s => s.trim()).filter(Boolean)
  const modelNames = (process.env.LLM_MODEL ?? '').split(',').map(s => s.trim()).filter(Boolean)

  if (endpoints.length === 0) {
    console.warn('MODEL_ENDPOINT is not set and config.yaml has no models — seeding without model config')
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
// Assemble and write openclaw.json
// -------------------------------------------------------------------
const config = { ...gatewayConfig }

if (Object.keys(providers).length > 0) {
  config.models = { providers }
}

if (Object.keys(mcpServers).length > 0) {
  config.mcp = { servers: mcpServers }
}

if (agentName) {
  config.agents = {
    list: [
      { id: agentName, identity: { name: agentName }, default: true },
    ],
  }
}

writeFileSync(configFile, JSON.stringify(config, null, 2))
console.log(`Seeded openclaw.json at ${configFile}`)
