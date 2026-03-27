# CRD Reference Overview

Language Operator provides five Custom Resource Definitions (CRDs) for managing AI agent workloads on Kubernetes.

## Core Resources

### LanguageCluster

**Scope:** Cluster
**Purpose:** Managed namespace for AI clusters

A `LanguageCluster` creates an isolated environment for agents, models, and tools with:

- Dedicated namespace
- Shared LiteLLM proxy for all models
- Optional external ingress
- NetworkPolicy enforcement

**Common Use Cases:**

- Multi-tenant agent deployments
- Environment separation (dev/staging/prod)
- Team or project isolation

[Full LanguageCluster Reference →](languagecluster.md)

---

### LanguageAgent

**Scope:** Namespace
**Purpose:** Autonomous, scheduled, and reactive agents

A `LanguageAgent` represents an AI agent running as a Kubernetes workload with:

- Container image specification
- Model, tool, and persona references
- Execution mode (autonomous, scheduled, interactive)
- Configuration injection
- Workspace storage

**Common Use Cases:**

- Deploying AI assistants (openclaw, etc.)
- Scheduled automation tasks
- Interactive agent services
- Multi-agent systems

[Full LanguageAgent Reference →](languageagent.md)

---

### LanguageModel

**Scope:** Namespace
**Purpose:** LLM endpoint configuration

A `LanguageModel` configures access to a large language model through a managed LiteLLM proxy:

- Provider abstraction (Anthropic, OpenAI, Azure, etc.)
- Credential management
- Rate limiting and retry policies
- Cost tracking
- Regional endpoints

**Common Use Cases:**

- Claude, GPT-4, or other commercial models
- Self-hosted models (Ollama, vLLM)
- Azure OpenAI deployments
- Multi-region failover

[Full LanguageModel Reference →](languagemodel.md)

---

### LanguageTool

**Scope:** Namespace
**Purpose:** MCP-compatible tool servers

A `LanguageTool` deploys a Model Context Protocol (MCP) tool server that extends agent capabilities:

- Standalone service or per-agent sidecar
- Automatic endpoint injection
- Tool schema discovery
- Network egress policies

**Common Use Cases:**

- Web search (Brave, Google)
- Email and calendar integration
- Database query tools
- Custom business logic

[Full LanguageTool Reference →](languagetool.md)

---

### LanguagePersona

**Scope:** Namespace
**Purpose:** Reusable behavioral templates

A `LanguagePersona` defines reusable personality and instruction templates:

- System prompts
- Tone and style guidelines
- Capability preferences
- Behavioral constraints
- Template inheritance

**Common Use Cases:**

- Consistent brand voice across agents
- Role-based agent templates (analyst, writer, coder)
- Compliance and safety constraints
- A/B testing behavioral variants

[Full LanguagePersona Reference →](languagepersona.md)

---

## Resource Relationships

```mermaid
graph TD
    Cluster[LanguageCluster] -->|creates namespace| NS[Namespace]
    Cluster -->|manages| Proxy[LiteLLM Proxy]

    Agent[LanguageAgent] -->|deployed in| NS
    Model[LanguageModel] -->|deployed in| NS
    Tool[LanguageTool] -->|deployed in| NS
    Persona[LanguagePersona] -->|deployed in| NS

    Agent -->|references| Model
    Agent -->|references| Tool
    Agent -->|references| Persona

    Model -->|registered in| Proxy

    Agent -->|connects to| Proxy
    Agent -->|connects to| Tool
```

## Configuration Injection

The operator automatically injects configuration into agent pods:

| Injection | Source | Location |
|-----------|--------|----------|
| Instructions | `spec.instructions` or `spec.instructionsFrom` | `/etc/agent/instructions.txt` |
| Config | Assembled from personas, models, tools | `/etc/agent/config.yaml` |
| Model Endpoint | Shared proxy URL | `MODEL_ENDPOINTS` env var |
| Model Names | All `modelRefs` | `LLM_MODEL` env var |
| Tool Endpoints | All `toolRefs` | `TOOL_ENDPOINTS` env var |
| Agent Identity | Metadata | `AGENT_NAME`, `AGENT_NAMESPACE`, etc. |

See [Agent Runtime Contract](../architecture/agents.md) for complete injection spec.

## API Versions

All CRDs use API version `langop.io/v1alpha1`.

!!! warning "Alpha API"
    The v1alpha1 API is subject to breaking changes. Pin your Helm chart version and review release notes before upgrading.

## Common Patterns

### Cross-Namespace References

Agents can reference models, tools, and personas in other namespaces:

```yaml
spec:
  modelRefs:
    - name: claude-sonnet
      namespace: shared-models  # Cross-namespace reference
```

### Multiple References

Agents can reference multiple models, tools, or personas:

```yaml
spec:
  modelRefs:
    - name: claude-sonnet
    - name: claude-haiku
  toolRefs:
    - name: web-search
    - name: email-client
  personaRefs:
    - name: base-assistant
    - name: brand-voice
```

The operator merges all referenced configurations.

### Conditional Status

All resources follow Kubernetes conventions for status:

```yaml
status:
  phase: Running  # or Pending, Failed, Unknown
  conditions:
    - type: Ready
      status: "True"
      reason: ResourcesCreated
      message: All child resources created successfully
```

## Next Steps

- Dive into individual CRD references (sidebar navigation)
- Read [Architecture Overview](../architecture/overview.md)
- See [Examples](../getting-started/examples.md) for common patterns
