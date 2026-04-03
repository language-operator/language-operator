# Language Operator

A [Kubernetes operator](https://kubernetes.io/docs/concepts/extend-kubernetes/operator/) for running AI agent clusters as native workloads.

## What It Does

Language Operator provides a purpose-built set of CRDs for deploying and managing agents in Kubernetes:

| Resource | Purpose |
|----------|---------|
| `LanguageCluster` | Provisions a managed namespace for agents |
| `LanguageAgent` | Deploys arbitrary agents like OpenClaw or OpenCode |
| `LanguageAgentRuntime` | Defines agent runtime presets |
| `LanguageModel` | Defines an LLM configuration (proxied through LiteLLM) |
| `LanguageTool` | Deploys an MCP-compatible server |
| `LanguagePersona` | Defines tone, personality and expertise |


## Getting Started

See the [installation guide](https://language-operator.github.io/language-operator/docs/getting-started/installation/) and [quick start](https://language-operator.github.io/language-operator/docs/getting-started/quickstart/) to deploy your first agent.

## Development

See the [development setup guide](docs/development/setup.md) for full instructions.

## Status

**Pre-release** — not ready for production.

## License

[MIT](LICENSE)
