# LanguageAgent

The `LanguageAgent` CRD represents an autonomous AI agent deployment in Kubernetes.

## Overview

A LanguageAgent runs a container image with:
- LLM access through the shared cluster proxy
- Tool endpoints for extended capabilities
- Persona configuration for behavioral templates
- Instructions for tasks and goals
- Workspace storage for persistent state

## Quick Example

```yaml
apiVersion: langop.io/v1alpha1
kind: LanguageAgent
metadata:
  name: my-agent
  namespace: my-cluster
spec:
  image: ghcr.io/my-org/my-agent:latest
  modelRefs:
    - name: claude-sonnet
  instructions: |
    You are a helpful AI assistant.
  workspace:
    size: 10Gi
```

## Complete API Reference

See the [Complete API Reference](reference.md#languageagent) for full field documentation including:

- **LanguageAgent** - Top-level resource
- **LanguageAgentSpec** - Specification fields
- **LanguageAgentStatus** - Status and conditions

## Key Concepts

### Execution Modes

- `autonomous` - Continuously running agent
- `scheduled` - Cron-based execution
- `interactive` - User-triggered execution
- `event-driven` - Responds to Kubernetes events

### Configuration Injection

The operator automatically mounts:

- `/etc/agent/instructions.txt` - Task instructions
- `/etc/agent/config.yaml` - Personas, models, tools

Environment variables:

- `MODEL_ENDPOINTS` - Shared proxy URL
- `LLM_MODEL` - Comma-separated model names
- `TOOL_ENDPOINTS` - MCP tool server URLs
- `AGENT_NAME`, `AGENT_NAMESPACE`, `AGENT_UUID` - Identity

### Resource Management

Agents are deployed as standard Kubernetes Deployments with:

- Configurable replicas
- Resource limits and requests
- Horizontal Pod Autoscaling support
- PodDisruptionBudgets for high availability

## Related Resources

- [LanguageModel](languagemodel.md) - Configure LLM access
- [LanguageTool](languagetool.md) - Add tool capabilities
- [LanguagePersona](languagepersona.md) - Define behavioral templates
- [Agent Runtime Contract](../architecture/agents.md) - What the operator injects

## Examples

See [Examples](../getting-started/examples.md) for common deployment patterns.
