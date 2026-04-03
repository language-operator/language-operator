# Language Operator

A Kubernetes operator for running AI agent clusters as native workloads.

## What It Does

Language Operator provides a purpose-built set of CRDs for deploying and managing agents in Kubernetes:

| Resource | Purpose |
|----------|---------|
| `LanguageCluster` | Managed namespace for agents |
| `LanguageAgent` | Free-form agents like OpenClaw or OpenCode |
| `LanguageAgentRuntime` | Agent runtime presets |
| `LanguageModel` | An LLM configuration (proxied through LiteLLM) |
| `LanguageTool` | A MCP-compatible server |
| `LanguagePersona` | Define tone, personality and expertise |


## Getting Started

See the [installation guide](https://language-operator.github.io/language-operator/docs/getting-started/installation/) and [quick start](https://language-operator.github.io/language-operator/docs/getting-started/quickstart/) to deploy your first agent.

## Development

See the [development setup guide](docs/development/setup.md) for full instructions.

## Status

**Pre-release** — not ready for production.

## License

[MIT](LICENSE)
