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

## Key Features

- **Native Kubernetes Integration** - Agents run as standard Kubernetes workloads with full lifecycle management
- **Network Isolation** - Cluster-scoped namespaces with NetworkPolicy enforcement
- **Model Abstraction** - Unified LiteLLM proxy handles provider diversity and credential management
- **MCP Tool Protocol** - Standard interface for extending agent capabilities
- **Configuration Injection** - Automated mounting of personas, instructions, and tool endpoints
- **Observability** - OpenTelemetry traces and metrics for all agent operations

## Quick Links

- [Installation Guide](getting-started/installation.md) - Install the operator via Helm
- [Quick Start](getting-started/quickstart.md) - Deploy your first agent in 5 minutes
- **[API Reference](api/reference.md)** - Auto-generated CRD field documentation
- [Architecture](architecture/overview.md) - System design and component interaction

## Project Status

**Pre-release** — not ready for production use.

## License

[MIT](https://github.com/language-operator/language-operator/blob/main/LICENSE)
