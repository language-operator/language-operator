/**
 * openclaw-adapter init container
 *
 * Bridges the language-operator config injection model to openclaw's native
 * config format. The operator injects MODEL_ENDPOINTS (a comma-separated list
 * of LiteLLM proxy URLs) and LLM_MODEL into every init container. This script
 * reads those values and seeds openclaw.json so openclaw routes all LLM traffic
 * through the operator-managed proxy rather than connecting to model APIs directly.
 *
 * On subsequent runs (PVC already has openclaw.json), exits immediately so the
 * agent's runtime config is preserved across restarts.
 */

import { writeFileSync, existsSync, mkdirSync } from 'fs'

const stateDir = process.env.OPENCLAW_STATE_DIR ?? '/home/node/.openclaw'
const configFile = `${stateDir}/openclaw.json`

if (existsSync(configFile)) {
  console.log(`openclaw.json already exists at ${configFile}, skipping seed`)
  process.exit(0)
}

mkdirSync(stateDir, { recursive: true })

// MODEL_ENDPOINTS is a comma-separated list of LiteLLM proxy base URLs
// injected by the language-operator into all init containers.
// LiteLLM exposes an OpenAI-compatible API, so we configure openclaw's
// OpenAI provider to point at the first proxy endpoint.
const modelEndpoints = process.env.MODEL_ENDPOINTS ?? ''
const llmModel = process.env.LLM_MODEL ?? ''

const proxyUrl = modelEndpoints.split(',')[0]?.trim()

if (!proxyUrl) {
  console.warn('MODEL_ENDPOINTS is not set — seeding empty openclaw.json. ' +
    'LLM calls will use openclaw defaults.')
}

const config = {}

if (proxyUrl) {
  // Configure openclaw to use the LiteLLM proxy as its OpenAI-compatible provider.
  // The proxy handles authentication with the real model API; a placeholder key
  // satisfies openclaw's validation without being used for auth.
  config.providers = {
    openai: {
      apiKey: 'sk-langop-proxy',
      baseUrl: proxyUrl,
    },
  }

  if (llmModel) {
    config.defaultModel = llmModel
  }

  console.log(`Configured LiteLLM proxy at ${proxyUrl}`)
}

writeFileSync(configFile, JSON.stringify(config, null, 2))
console.log(`Seeded openclaw.json at ${configFile}`)
