# Releases

This document tracks releases of the Language Operator project.

---

## v0.2.0 - 2025-11-09

**Initial Public Release**

Language Operator is a Kubernetes operator that transforms natural language descriptions into autonomous agents that execute tasks on your behalf.

### Core Capabilities

- **Natural Language Interface**: Describe tasks in plain English via `aictl` CLI
- **Autonomous Agent Synthesis**: Automatically generates Ruby code from task descriptions
- **Kubernetes-Native**: Deploys agents as CRDs (LanguageAgent, LanguageCluster, LanguageModel, LanguageTool)
- **Scheduled Execution**: Cron-based scheduling for recurring tasks
- **Tool Integration**: Built-in support for email, spreadsheets, web scraping, and custom tools
- **Multi-LLM Support**: Integration with Anthropic Claude and other language models via LiteLLM
- **Network Isolation**: NetworkPolicy enforcement for secure agent execution (requires compatible CNI like Cilium)
- **Private Registry Support**: Container image whitelist and authentication for air-gapped deployments

### Architecture

- **Operator Namespace**: `kube-system`
- **Base Images**: Alpine-based with `langop` user for security
- **SDK**: Published `language-operator` gem for Ruby components
- **Infrastructure**: Tested on k3s with Cilium CNI

### Getting Started

See [docs/quickstart.md](docs/quickstart.md) for installation and usage instructions.
