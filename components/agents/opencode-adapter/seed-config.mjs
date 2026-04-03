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
// Fall back to MODEL_ENDPOINT / LLM_MODEL env vars if absent.
// -------------------------------------------------------------------
const configModels = operatorConfig?.models ?? {}
const provider = {}

// LiteLLM exposes an OpenAI-compatible API, so we always use "openai" as the
// provider key regardless of what the CRD models are named. All models are
// aggregated under the single gateway endpoint.

if (Object.keys(configModels).length > 0) {
  // Primary source: config.yaml models section
  // All models share the same LiteLLM gateway — use the first endpoint found.
  const models = {}
  let gatewayEndpoint = null

  for (const [crdName, model] of Object.entries(configModels)) {
    if (!model.endpoint) {
      console.warn(`Model '${crdName}' has no endpoint — skipping`)
      continue
    }
    gatewayEndpoint ??= model.endpoint
    const modelId = model.model ?? crdName
    models[modelId] = {}
    console.log(`Registered model '${modelId}' via gateway ${model.endpoint}`)
  }

  if (gatewayEndpoint) {
    const baseURL = gatewayEndpoint.replace(/\/+$/, '') + '/v1'
    provider['openai'] = {
      options: {
        baseURL,
        apiKey: 'sk-langop-proxy',  // placeholder; LiteLLM proxy handles real auth
      },
      models,
    }
  }
} else {
  // Fallback: zip MODEL_ENDPOINT + LLM_MODEL env vars
  const endpoints = (process.env.MODEL_ENDPOINT ?? '').split(',').map(s => s.trim()).filter(Boolean)
  const modelNames = (process.env.LLM_MODEL ?? '').split(',').map(s => s.trim()).filter(Boolean)

  if (endpoints.length === 0) {
    console.warn('MODEL_ENDPOINT is not set and config.yaml has no models — seeding without provider config')
  } else {
    // All models share the same LiteLLM gateway — use the first endpoint.
    const models = {}
    for (let i = 0; i < modelNames.length; i++) {
      models[modelNames[i]] = {}
    }
    const baseURL = endpoints[0].replace(/\/+$/, '') + '/v1'
    provider['openai'] = {
      options: {
        baseURL,
        apiKey: 'sk-langop-proxy',
      },
      models: Object.keys(models).length > 0 ? models : undefined,
    }
    console.log(`Configured openai provider → ${baseURL} (models: ${modelNames.join(', ') || 'none'})`)
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
