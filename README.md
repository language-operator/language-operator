# Language Operator

A [Kubernetes operator](https://kubernetes.io/docs/concepts/extend-kubernetes/operator/) for running AI agent clusters at scale.

## What It Does

Language Operator provides a purpose-built set of CRDs for deploying and managing agents in Kubernetes:

| Resource | Status | Purpose |
|----------|--------|---------|
| [`LanguageCluster`](https://langop.io/docs/api/reference/#languagecluster) | Alpha | Configures a managed namespace for agents |
| [`LanguageAgent`](https://langop.io/docs/api/reference/#languageagent) | Alpha | Deploy a goal-directed agent |
| [`LanguageAgentRuntime`](https://langop.io/docs/api/reference/#languageagentruntime) | Alpha | Presets for popular vendors (Claude, OpenClaw, OpenCode) |
| [`LanguageModel`](https://langop.io/docs/api/reference/#languagemodel) | Alpha | Model configuration (proxied through LiteLLM) |
| [`LanguageTool`](https://langop.io/docs/api/reference/#languagetool) | Alpha | MCP-compatible tool server |
| [`LanguagePersona`](https://langop.io/docs/api/reference/#languagepersona) | Development | Reusable behaviors and expertise |


## Getting Started

See the [installation guide](https://langop.io/docs/getting-started/installation/) and [quick start](https://langop.io/docs/getting-started/quickstart/) to deploy your first agent.

## Development

See the [development setup guide](docs/development/setup.md) for full instructions.

## Status

**Pre-release** — not ready for production.

## License

[MIT](LICENSE)
