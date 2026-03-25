# Agent Runtime Container Specification

This document defines the contract between the Language Operator and agent runtime containers. It specifies what the operator guarantees to provide to every running agent, and what a compliant agent image must implement.

## Overview

A **LanguageAgent** runs as a standard Kubernetes Deployment (or CronJob for scheduled agents). The operator manages the pod lifecycle, injects configuration, and handles networking. The agent container is responsible for its own runtime logic: reading instructions, connecting to tools, and executing tasks.

## What the Operator Provides

### Mounted Files

The operator mounts exactly two files into every agent container:

| Path | Content | Source |
|------|---------|--------|
| `/etc/agent/instructions.txt` | Task instructions (plain text) | `spec.instructions` (inline) or `spec.instructionsFrom` (ConfigMap/Secret reference) |
| `/etc/agent/config.yaml` | Structured agent configuration (YAML) | Assembled by the operator from personas, tools, models, and agent metadata |

Files are read-only. The operator reconciles them on every change to the LanguageAgent spec or referenced resources.

### Environment Variables

The operator injects the following environment variables into every agent container:

| Variable | Value |
|----------|-------|
| `AGENT_NAME` | `metadata.name` of the LanguageAgent |
| `AGENT_NAMESPACE` | `metadata.namespace` of the LanguageAgent |
| `AGENT_UUID` | Stable UUID assigned to this agent (from `spec.uuid` or generated) |
| `AGENT_MODE` | Execution mode: `autonomous`, `interactive`, `scheduled`, or `event-driven` |
| `AGENT_CLUSTER_NAME` | Name of the LanguageCluster this agent belongs to (empty if none) |
| `AGENT_CLUSTER_UUID` | Stable UUID of the LanguageCluster (empty if none) |

Additional environment variables from `spec.env` and `spec.envFrom` are passed through unchanged.

### Networking

Every agent gets:
- A **ClusterIP Service** on port 8080 named `<agent-name>` in the agent's namespace
- An **HTTPRoute** for external access (if a Gateway is configured)
- **NetworkPolicy** permitting inbound traffic from:
  - Other agent pods (label `langop.io/kind=LanguageAgent`) on port 8080
  - The operator dashboard
  - Configured trigger pods

## What the Agent Must Implement

### Health Check (Required)

```
GET /health
```

Returns `200 OK` when the agent is ready to accept requests. Used by Kubernetes readiness and liveness probes.

### Port 8080 (Required)

The agent must listen on port 8080. What it serves there is up to the agent — HTTP, gRPC, A2A, OpenAI-compatible API, or anything else.

### Startup Behaviour

On startup, the agent should:

1. Read `/etc/agent/instructions.txt` for task definition
2. Read `/etc/agent/config.yaml` for all other configuration (personas, tools, models)
3. Start listening on port 8080

## File Formats

### Instructions (`/etc/agent/instructions.txt`)

Plain text. The task definition for this agent — what it should do, its role, and any behavioural directives.

```
You are a data analyst. Analyze CSV files and generate insights.
Focus on trends, anomalies, and actionable recommendations.
Always cite data sources and use structured output.
```

### Agent Config (`/etc/agent/config.yaml`)

A single YAML document assembled by the operator. Contains everything the agent needs to configure its runtime: persona(s), tool endpoints, and model credentials.

```yaml
# Agent identity (mirrors AGENT_* env vars for convenience)
agent:
  name: data-analyst
  namespace: default
  uuid: "550e8400-e29b-41d4-a716-446655440000"
  mode: autonomous
  clusterName: production-cluster
  clusterUUID: "6ba7b810-9dad-11d1-80b4-00c04fd430c8"

# Persona configuration — merged in order if multiple are specified.
# Each entry is the full spec of the referenced LanguagePersona resource.
personas:
  - name: analytical-persona
    displayName: Analytical Persona
    description: Precise, data-driven, cites sources
    systemPrompt: You are an analytical assistant. Be precise and always cite data.
    tone: professional
    language: en
    instructions:
      - Always cite data sources
      - Use structured output
    capabilities:
      - data analysis
      - statistical reasoning
    limitations:
      - Do not speculate without data

# Tool endpoints — keyed by tool name, resolved to in-cluster MCP service URLs.
tools:
  mem0-memory:
    endpoint: http://mem0-memory.tools.svc.cluster.local:8080
    protocol: mcp
  python-executor:
    endpoint: http://python-executor.tools.svc.cluster.local:8080
    protocol: mcp

# Model configuration — keyed by model name.
models:
  claude-sonnet:
    role: primary
    provider: anthropic
    endpoint: https://api.anthropic.com/v1
    model: claude-sonnet-4-6
    secretRef: claude-credentials
```

## Example LanguageAgent YAML

```yaml
apiVersion: langop.io/v1alpha1
kind: LanguageAgent
metadata:
  name: data-analyst
  namespace: default
spec:
  image: myregistry/agent-runtime:python-v1.0.0
  imagePullPolicy: Always

  instructions: |
    You are a data analyst. Analyze CSV files and generate insights.
    Focus on trends, anomalies, and actionable recommendations.

  personaRefs:
    - name: analytical-persona

  executionMode: autonomous

  toolRefs:
    - name: mem0-memory
    - name: python-executor

  modelRefs:
    - name: claude-sonnet

  replicas: 1
  resources:
    limits:
      memory: 1Gi
      cpu: 500m
```

## Compliance Checklist

A compliant agent image must:

- [ ] Listen on port **8080**
- [ ] Expose `GET /health` returning `200 OK` when ready
- [ ] Read task instructions from `/etc/agent/instructions.txt` on startup
- [ ] Read runtime configuration from `/etc/agent/config.yaml` on startup
- [ ] Respect `AGENT_NAME`, `AGENT_NAMESPACE`, `AGENT_UUID`, `AGENT_MODE`, `AGENT_CLUSTER_NAME`, `AGENT_CLUSTER_UUID` environment variables
