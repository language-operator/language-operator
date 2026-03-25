# Language Operator Architecture

**Language Operator** is a Kubernetes operator that runs AI agents as native Kubernetes workloads. It handles agent lifecycle, configuration injection, networking, and observability — so agent runtime containers can focus entirely on doing work.

---

## Table of Contents

1. [Architectural Principles](#architectural-principles)
2. [System Overview](#system-overview)
3. [Custom Resource Definitions](#custom-resource-definitions)
4. [Agent Runtime Contract](#agent-runtime-contract)
5. [Network Architecture](#network-architecture)
6. [Observability](#observability)
7. [Extensibility](#extensibility)

---

## Architectural Principles

### 1. Container-Native Agents

Agents are container images. The operator deploys them as standard Kubernetes Deployments (or CronJobs for scheduled agents). There is no code generation, no synthesis, and no language-specific runtime baked into the operator.

**Separation of concerns:**
- **Operator**: lifecycle, configuration injection, networking, observability
- **Agent image**: reasoning, tool use, task execution, inter-agent communication

### 2. Configuration over Code

The operator injects two files into every agent container:

| Path | Content |
|------|---------|
| `/etc/agent/instructions.txt` | Task instructions (plain text) |
| `/etc/agent/config.yaml` | Personas, tools, models, agent identity |

Instructions are what the agent does. The image is how it does it. Changing instructions requires no image rebuild.

### 3. Delegate Specialised Concerns

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
│  │  · LanguagePersona   │    │                              │   │
│  │  · LanguageTool      │    │  Exposes (port 8080):        │   │
│  │  · LanguageModel     │    │  · GET /health               │   │
│  │  · LanguageCluster   │    │  · whatever the agent serves │   │
│  └──────────────────────┘    └──────────────────────────────┘   │
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
- Environment variables: `AGENT_NAME`, `AGENT_NAMESPACE`, `AGENT_UUID`, `AGENT_MODE`, `AGENT_CLUSTER_NAME`, `AGENT_CLUSTER_UUID`
- ClusterIP Service on port 8080
- HTTPRoute for external access
- NetworkPolicy allowing agent-to-agent traffic on port 8080

**Agent image must:**
- Listen on port 8080
- Expose `GET /health` for readiness/liveness probes

---

## Network Architecture

Each `LanguageAgent` gets:

- **ClusterIP Service** (`<agent-name>.<namespace>.svc.cluster.local:8080`) — in-cluster access
- **HTTPRoute** — external access via the cluster Gateway
- **NetworkPolicy** with ingress rules allowing:
  - Agent pods (`langop.io/kind=LanguageAgent`) on port 8080 — agent-to-agent traffic
  - Operator dashboard
  - Configured trigger pods

Tool servers (`LanguageTool`) get their own Services. The operator reconciles NetworkPolicy to allow agent pods to reach tool pods.

---

## Observability

All operator reconciliation loops emit OpenTelemetry spans. Traces are collected by the OTel Collector and stored in ClickHouse.

**Key trace attributes:**
- `agent.name`, `agent.namespace`, `agent.uuid`
- `persona.name`, `tool.name`, `model.name`

---

## Extensibility

**Custom agent runtimes**: Any container image that implements the agent contract ([`spec/agents.md`](../spec/agents.md)) works. The operator is runtime-agnostic — Python, Node.js, Go, or anything else.

**Custom tools**: Any service that implements the tool contract ([`spec/tools.md`](../spec/tools.md)) — MCP protocol on a Kubernetes Service plus `GET /health` — can be registered as a `LanguageTool`.

**Custom models**: Any LLM endpoint can be registered as a `LanguageModel` and injected into agent config.

