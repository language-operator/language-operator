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
| `AGENT_CLUSTER_NAME` | Name of the LanguageCluster this agent belongs to (from `spec.clusterRef`) |
| `AGENT_CLUSTER_UUID` | Stable UUID of the LanguageCluster (from the referenced LanguageCluster's `spec.uuid`) |
| `AGENT_OPERATOR_WEBHOOK_URL` | The operator's push notification endpoint for task state changes |
| `AGENT_OPERATOR_WEBHOOK_TOKEN` | Token the agent includes in push notification requests for operator-side validation |

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

### A2A Protocol (Required)

Agents must implement the [Agent-to-Agent (A2A) protocol](https://a2a-protocol.org/latest/specification/) on port 8080. This enables agent discovery and task delegation without custom orchestration. The A2A spec defines paths without a version prefix — do not add `/v1/` to these endpoints.

#### Agent Discovery

```
GET /.well-known/agent.json   # Minimal Agent Card (public, no auth required)
GET /agentCard                # Extended Agent Card (may require auth)
```

The Agent Card describes this agent's identity, capabilities, and skills:

```json
{
  "name": "data-analyst",
  "description": "Analyzes CSV files and generates insights",
  "version": "1.0.0",
  "url": "http://data-analyst.default.svc.cluster.local:8080",
  "capabilities": {
    "streaming": true,
    "pushNotifications": false
  },
  "skills": [
    {
      "id": "analyze-csv",
      "name": "Analyze CSV",
      "description": "Load and analyze a CSV file, returning statistical insights"
    }
  ]
}
```

#### Message Sending

```
POST /messages                # Send a message, receive response synchronously
POST /messages:stream         # Send a message, receive response as SSE stream
```

Request:

```json
{
  "message": {
    "role": "user",
    "parts": [{ "text": "Analyze sales_data.csv and summarize the trends." }]
  }
}
```

#### Task Management

```
GET  /tasks                              # List tasks
GET  /tasks/{taskId}                     # Get task status and results
POST /tasks/{taskId}:cancel              # Cancel a task
GET  /tasks/{taskId}:subscribe           # Subscribe to task updates (SSE)
```

Task states: `submitted`, `working`, `completed`, `failed`, `canceled`.

#### Push Notification Configuration

```
POST   /tasks/{taskId}/pushNotificationConfigs          # Create notification config
GET    /tasks/{taskId}/pushNotificationConfigs          # List notification configs
GET    /tasks/{taskId}/pushNotificationConfigs/{id}     # Get specific config
DELETE /tasks/{taskId}/pushNotificationConfigs/{id}     # Delete config
```

### Operator Push Notification Registration (Required)

The operator registers itself as a push notification subscriber for every agent on startup. Agents **must** accept this registration and send state-change notifications to the operator throughout the task lifecycle.

The operator registers by calling:

```
POST /tasks/{taskId}/pushNotificationConfigs
Content-Type: application/json

{
  "url": "http://language-operator.language-operator.svc.cluster.local:8081/hooks/tasks",
  "token": "<per-agent-token injected via AGENT_OPERATOR_WEBHOOK_TOKEN env var>"
}
```

The operator webhook URL and token are injected as environment variables:

| Variable | Value |
|----------|-------|
| `AGENT_OPERATOR_WEBHOOK_URL` | The operator's push notification endpoint |
| `AGENT_OPERATOR_WEBHOOK_TOKEN` | Token the agent must include in push notification requests for validation |

Agents must send push notifications to the operator on **every task state transition**, including:

- `submitted` → `working` (task started)
- `working` → `input-required` (agent is blocked, needs caller input)
- `working` → `auth-required` (agent is blocked, needs credentials)
- Any state → `completed`, `failed`, `canceled`, `rejected` (terminal)
- Artifact generation (intermediate results)

**`input-required` and `auth-required` are not errors.** The operator will create or update a `LanguageAgentTask` Kubernetes resource surfacing the blocked state so external actors can observe and resolve it. The task remains alive on the agent — the operator will `POST /messages` with the resolution when one is provided.

Agents must **not** time out or discard a task in `input-required` or `auth-required` state without emitting a final `canceled` or `failed` notification.

### Health Check (Required)

```
GET /health
```

Returns `200 OK` when the agent is ready to accept tasks. Used by Kubernetes readiness probes.

### Startup Behaviour

On startup, the agent must:

1. Read `/etc/agent/instructions.txt` for task definition
2. Read `/etc/agent/config.yaml` for all other configuration (personas, tools, models)
3. Start listening on port 8080 and expose the A2A endpoints and `/health`
4. Register the operator as a push notification subscriber for all future tasks using `AGENT_OPERATOR_WEBHOOK_URL` and `AGENT_OPERATOR_WEBHOOK_TOKEN`

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
  gpt-4:
    role: fallback
    provider: openai
    endpoint: https://api.openai.com/v1
    model: gpt-4o
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

  mode: autonomous

  tools:
    - name: mem0-memory
      enabled: true
    - name: python-executor
      enabled: true

  models:
    - name: claude-sonnet
      role: primary

  replicas: 1
  resources:
    limits:
      memory: 1Gi
      cpu: 500m
```

## Compliance Checklist

A compliant agent image must:

- [ ] Expose `GET /.well-known/agent.json` and `GET /agentCard` returning valid A2A Agent Cards
- [ ] Expose `POST /messages` and `POST /messages:stream` for synchronous and streaming message sending
- [ ] Expose `GET /tasks`, `GET /tasks/{id}`, `POST /tasks/{id}:cancel`, `GET /tasks/{id}:subscribe` for task management
- [ ] Expose `GET /health` returning `200 OK` when ready
- [ ] Listen on port **8080**
- [ ] Read task instructions from `/etc/agent/instructions.txt` on startup
- [ ] Read runtime configuration from `/etc/agent/config.yaml` on startup
- [ ] Register the operator as a push notification subscriber on startup using `AGENT_OPERATOR_WEBHOOK_URL` and `AGENT_OPERATOR_WEBHOOK_TOKEN`
- [ ] Send push notifications to the operator on every task state transition, including `input-required` and `auth-required`
- [ ] Respect `AGENT_NAME`, `AGENT_NAMESPACE`, `AGENT_UUID`, `AGENT_MODE`, `AGENT_CLUSTER_NAME`, `AGENT_CLUSTER_UUID`, `AGENT_OPERATOR_WEBHOOK_URL`, `AGENT_OPERATOR_WEBHOOK_TOKEN` environment variables
