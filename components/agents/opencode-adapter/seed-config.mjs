/**
 * opencode-adapter init container
 *
 * Bridges the language-operator config injection model to opencode's native
 * config format. Reads /etc/agent/config.yaml (injected by the operator) and
 * translates models and tools into /etc/opencode/opencode.jsonc.
 *
 * The config file is always overwritten — it is operator-managed configuration,
 * not user state (unlike openclaw where the JSON config holds runtime state).
 */

import { readFileSync, writeFileSync, existsSync, mkdirSync } from 'fs'
import { parse as parseYaml } from 'yaml'

const outputDir = process.env.OPENCODE_CONFIG_DIR ?? '/etc/opencode'
const outputFile = `${outputDir}/opencode.jsonc`

mkdirSync(outputDir, { recursive: true })

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
// Build provider config from config.yaml models section.
// Fall back to MODEL_ENDPOINTS / LLM_MODEL env vars if absent.
// -------------------------------------------------------------------
const configModels = operatorConfig?.models ?? {}
const provider = {}

if (Object.keys(configModels).length > 0) {
  // Primary source: config.yaml models section
  for (const [crdName, model] of Object.entries(configModels)) {
    if (!model.endpoint) {
      console.warn(`Model '${crdName}' has no endpoint — skipping`)
      continue
    }
    const modelId = model.model ?? crdName
    provider[crdName] = {
      options: {
        baseURL: model.endpoint,
        apiKey: 'sk-langop-proxy',  // placeholder; LiteLLM proxy handles real auth
      },
      models: { [modelId]: {} },
    }
    console.log(`Configured provider '${crdName}' → ${model.endpoint} (model: ${modelId})`)
  }
} else {
  // Fallback: zip MODEL_ENDPOINTS + LLM_MODEL env vars
  const endpoints = (process.env.MODEL_ENDPOINTS ?? '').split(',').map(s => s.trim()).filter(Boolean)
  const modelNames = (process.env.LLM_MODEL ?? '').split(',').map(s => s.trim()).filter(Boolean)

  if (endpoints.length === 0) {
    console.warn('MODEL_ENDPOINTS is not set and config.yaml has no models — seeding without provider config')
  }

  for (let i = 0; i < endpoints.length; i++) {
    const key = modelNames[i] ?? `model-${i}`
    provider[key] = {
      options: {
        baseURL: endpoints[i],
        apiKey: 'sk-langop-proxy',
      },
      models: { [key]: {} },
    }
    console.log(`Configured provider '${key}' → ${endpoints[i]} (from env vars)`)
  }
}

// -------------------------------------------------------------------
// Build MCP server config from config.yaml tools section
// -------------------------------------------------------------------
const configTools = operatorConfig?.tools ?? {}
const mcp = {}

for (const [toolName, tool] of Object.entries(configTools)) {
  if (!tool.endpoint) {
    console.warn(`Tool '${toolName}' has no endpoint — skipping`)
    continue
  }
  if (!tool.endpoint.startsWith('http://') && !tool.endpoint.startsWith('https://')) {
    console.warn(`Tool '${toolName}' endpoint '${tool.endpoint}' is not an HTTP URL — skipping`)
    continue
  }
  mcp[toolName] = { type: 'remote', url: tool.endpoint }
  console.log(`Configured MCP server '${toolName}' → ${tool.endpoint}`)
}

// -------------------------------------------------------------------
// Assemble and write opencode.jsonc
// -------------------------------------------------------------------
const config = {
  autoupdate: false,
}

if (Object.keys(provider).length > 0) {
  config.provider = provider
}

if (Object.keys(mcp).length > 0) {
  config.mcp = mcp
}

writeFileSync(outputFile, JSON.stringify(config, null, 2))
console.log(`Wrote opencode config to ${outputFile}`)
