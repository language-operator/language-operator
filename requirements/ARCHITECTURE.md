# Language Operator Architecture

**Language Operator** is a Kubernetes operator that runs AI agents as native Kubernetes workloads. It handles agent lifecycle, configuration injection, networking, observability, and task state visibility — so agent runtime containers can focus entirely on doing work.

---

## Table of Contents

1. [Architectural Principles](#architectural-principles)
2. [System Overview](#system-overview)
3. [Custom Resource Definitions](#custom-resource-definitions)
4. [Agent Runtime Contract](#agent-runtime-contract)
5. [A2A Protocol](#a2a-protocol)
6. [Task Lifecycle](#task-lifecycle)
7. [Network Architecture](#network-architecture)
8. [Observability](#observability)
9. [Extensibility](#extensibility)

---

## Architectural Principles

### 1. Container-Native Agents

Agents are container images. The operator deploys them as standard Kubernetes Deployments (or CronJobs for scheduled agents). There is no code generation, no synthesis, and no language-specific runtime baked into the operator.

**Separation of concerns:**
- **Operator**: lifecycle, configuration injection, networking, task observability
- **Agent image**: reasoning, tool use, A2A protocol, task execution

### 2. Configuration over Code

The operator injects two files into every agent container:

| Path | Content |
|------|---------|
| `/etc/agent/instructions.txt` | Task instructions (plain text) |
| `/etc/agent/config.yaml` | Personas, tools, models, agent identity |

Instructions are what the agent does. The image is how it does it. Changing instructions requires no image rebuild.

### 3. A2A by Design

Every agent speaks [Google's Agent-to-Agent (A2A) protocol](https://a2a-protocol.org/latest/specification/) on port 8080. A2A is how agents discover each other, delegate work, and stream results — without the operator acting as an intermediary or orchestrator.

The operator adds a NetworkPolicy ingress rule allowing any agent pod to reach any other agent on port 8080. Agent-to-agent communication flows directly.

### 4. Kubernetes-Native Task Observability

The operator surfaces agent task state as Kubernetes resources. Blocked tasks (`input-required`, `auth-required`) appear as `LanguageAgentTask` CRs so operators, humans, and other controllers can observe and resolve them with standard `kubectl` commands.

### 5. Delegate Specialised Concerns

Memory, knowledge retrieval, and code execution are handled by MCP tool servers (`LanguageTool` CRDs), not by the operator. The operator resolves tool endpoints and injects them into agent config — it does not proxy or inspect tool traffic.

---

## System Overview

```
┌─────────────────────────────────────────────────────────────────┐
│                        Kubernetes Cluster                        │
│                                                                  │
│  ┌──────────────────────┐    ┌──────────────────────────────┐   │
│  │   Language Operator   │    │        Agent Pods            │   │
│  │                      │    │                              │   │
│  │  Reconciles:         │    │  /etc/agent/instructions.txt │   │
│  │  · LanguageAgent     │───▶│  /etc/agent/config.yaml      │   │
│  │  · LanguageAgentTask │    │                              │   │
│  │  · LanguagePersona   │    │  Exposes (port 8080):        │   │
│  │  · LanguageTool      │    │  · A2A endpoints             │   │
│  │  · LanguageModel     │    │  · GET /health               │   │
│  │  · LanguageCluster   │    │                              │   │
│  │                      │◀───│  Push notifications →        │   │
│  │  Webhook handler:    │    │  operator task hook          │   │
│  │  · Task state changes│    └──────────────────────────────┘   │
│  └──────────────────────┘                                        │
│                                                                  │
│  ┌──────────────────────┐    ┌──────────────────────────────┐   │
│  │    MCP Tool Servers  │    │       ClickHouse             │   │
│  │  (LanguageTool CRDs) │    │   (OTel traces & metrics)    │   │
│  └──────────────────────┘    └──────────────────────────────┘   │
└─────────────────────────────────────────────────────────────────┘
```

---

## Custom Resource Definitions

### LanguageAgent

The primary resource. Describes an agent to deploy.

```yaml
apiVersion: langop.io/v1alpha1
kind: LanguageAgent
metadata:
  name: data-analyst
spec:
  image: myregistry/agent-runtime:v1.0.0   # required
  instructions: |                           # or instructionsFrom
    You are a data analyst. Analyze CSV
    files and generate insights.
  personaRefs:
    - name: professional-tone
  modelRefs:
    - name: claude-sonnet
  toolRefs:
    - name: python-executor
  executionMode: autonomous                 # autonomous|interactive|scheduled|event-driven
  replicas: 1
```

The operator creates: Deployment, Service (port 8080), HTTPRoute, NetworkPolicy, and two ConfigMaps (instructions, config).

### LanguageAgentTask

Created by the operator when an agent reports a task state change via push notification. Read-only for most consumers — the operator manages the lifecycle.

```yaml
apiVersion: langop.io/v1alpha1
kind: LanguageAgentTask
metadata:
  name: data-analyst-abc123
spec:
  agentRef: data-analyst
  taskId: "abc123"
  contextId: "ctx-456"
status:
  state: input-required                    # submitted|working|input-required|auth-required|completed|failed|canceled|rejected
  inputRequired:
    prompt: "Which date range should I analyze?"
    since: "2025-03-14T10:00:00Z"
```

When a task enters `input-required` or `auth-required`, the operator emits a Kubernetes Event on the parent `LanguageAgent` and creates/updates the `LanguageAgentTask` CR. External actors patch a resolution onto the CR; the controller sends `POST /messages` to unblock the agent.

### LanguagePersona

Behavioral configuration for agents. Defines system prompt, tone, instructions, capabilities, and constraints. The operator assembles referenced personas into `/etc/agent/config.yaml` under the `personas:` key.

```yaml
apiVersion: langop.io/v1alpha1
kind: LanguagePersona
metadata:
  name: professional-tone
spec:
  systemPrompt: "You are a professional assistant. Be concise and precise."
  tone: professional
  instructions:
    - Always cite sources
    - Use structured output
```

### LanguageTool

An MCP-compatible tool server available to agents. The operator resolves the service endpoint and injects it into `/etc/agent/config.yaml` under the `tools:` key.

```yaml
apiVersion: langop.io/v1alpha1
kind: LanguageTool
metadata:
  name: python-executor
spec:
  displayName: Python Executor
  serviceRef:
    name: python-executor
    namespace: tools
  port: 8080
```

Agents connect to tools directly over MCP. The operator does not proxy tool traffic. The full tool contract is defined in [`spec/tools.md`](../spec/tools.md).

### LanguageModel

An LLM endpoint configuration. Injected into `/etc/agent/config.yaml` under the `models:` key, including provider, endpoint, and a reference to a Secret for credentials.

### LanguageCluster

Groups agents into a named cluster with shared networking and deployment configuration. Multi-cluster agent deployments target a `LanguageCluster`.

---

## Agent Runtime Contract

The full contract is defined in [`spec/agents.md`](../spec/agents.md). Summary:

**Operator provides:**
- `/etc/agent/instructions.txt` — plain text task instructions
- `/etc/agent/config.yaml` — structured YAML with agent identity, personas, tools, models
- Environment variables: `AGENT_NAME`, `AGENT_NAMESPACE`, `AGENT_UUID`, `AGENT_MODE`, `AGENT_CLUSTER_NAME`, `AGENT_CLUSTER_UUID`, `AGENT_OPERATOR_WEBHOOK_URL`, `AGENT_OPERATOR_WEBHOOK_TOKEN`
- ClusterIP Service on port 8080
- HTTPRoute for external access
- NetworkPolicy allowing agent-to-agent traffic on port 8080

**Agent image must:**
- Implement the A2A protocol on port 8080
- Expose `GET /health` for readiness probes
- Register the operator as a push notification subscriber on startup
- Send push notifications on every task state transition

---

## A2A Protocol

Agents implement [A2A](https://a2a-protocol.org/latest/specification/) — Google's open standard for agent interoperability. All paths are on port 8080 with no version prefix.

| Endpoint | Purpose |
|----------|---------|
| `GET /.well-known/agent.json` | Public Agent Card (discovery) |
| `GET /agentCard` | Extended Agent Card (post-auth) |
| `POST /messages` | Send message, synchronous response |
| `POST /messages:stream` | Send message, SSE stream response |
| `GET /tasks` | List tasks |
| `GET /tasks/{id}` | Get task status and artifacts |
| `POST /tasks/{id}:cancel` | Cancel a task |
| `GET /tasks/{id}:subscribe` | Subscribe to task updates (SSE) |
| `POST /tasks/{id}/pushNotificationConfigs` | Register push notification webhook |

The **Agent Card** (`/.well-known/agent.json`) is the discovery entry point. It declares the agent's name, description, skills, capabilities (streaming, push notifications), and security requirements. Other agents and external clients use this to understand what the agent can do before sending tasks.

Agent-to-agent communication is direct (agent → service → agent). The operator does not intermediate. Orchestration patterns (fan-out, pipelines, loops) are the agent's responsibility, not the operator's.

---

## Task Lifecycle

```
Client/Agent
    │
    ▼
POST /messages
    │
    ▼
Agent processes → push notification → Operator webhook
                                          │
                                          ▼
                                   LanguageAgentTask CR
                                   (state: working)
    │
    ├─ Normal completion ──────────────────────────────────────────▶ state: completed
    │
    ├─ input-required ─────────────────────────────────────────────▶ state: input-required
    │       │                                                         K8s Event emitted
    │       │   External actor patches resolution
    │       └─▶ Operator sends POST /messages ──────────────────────▶ state: working → completed
    │
    └─ auth-required ──────────────────────────────────────────────▶ state: auth-required
            │                                                         K8s Event emitted
            │   Operator resolves credential from Secret
            └─▶ Operator sends POST /messages ──────────────────────▶ state: working → completed
```

`input-required` and `auth-required` are **not errors**. They are pauses. The agent holds the task open; the operator provides the path to resume it.

---

## Network Architecture

Each `LanguageAgent` gets:

- **ClusterIP Service** (`<agent-name>.<namespace>.svc.cluster.local:8080`) — in-cluster A2A access
- **HTTPRoute** — external access via the cluster Gateway
- **NetworkPolicy** with ingress rules allowing:
  - Agent pods (`langop.io/kind=LanguageAgent`) on port 8080 — A2A traffic
  - Operator dashboard
  - Configured trigger pods

Tool servers (`LanguageTool`) get their own Services. The operator reconciles NetworkPolicy to allow agent pods to reach tool pods.

---

## Observability

All operator reconciliation loops emit OpenTelemetry spans. Traces are collected by the OTel Collector and stored in ClickHouse.

**Key trace attributes:**
- `agent.name`, `agent.namespace`, `agent.uuid`
- `task.id`, `task.context_id`, `task.state`
- `persona.name`, `tool.name`, `model.name`

**LanguageAgentTask** status provides real-time task state visibility via `kubectl`:

```bash
kubectl get latask -A
kubectl describe latask data-analyst-abc123
```

---

## Extensibility

**Custom agent runtimes**: Any container image that implements the agent contract ([`spec/agents.md`](../spec/agents.md)) works. The operator is runtime-agnostic — Python, Node.js, Go, or anything else.

**Custom tools**: Any service that implements the tool contract ([`spec/tools.md`](../spec/tools.md)) — MCP protocol on a Kubernetes Service plus `GET /health` — can be registered as a `LanguageTool`.

**Custom models**: Any LLM endpoint can be registered as a `LanguageModel` and injected into agent config.

**Protocol extensions**: The A2A spec defines a formal extension system. Extensions are declared in the Agent Card with a URI and optional `required` flag. The operator passes through extension declarations without interpreting them.
