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

### 1. Pure Deployment Infrastructure

The operator deploys containers and manages Kubernetes resources. It has no opinion about what the container serves, what protocol it speaks, or how it behaves at runtime. Agent images are treated as opaque workloads.

**Separation of concerns:**
- **Operator**: lifecycle, configuration injection, networking, observability
- **Agent image**: reasoning, tool use, task execution — everything the agent does

This is the openclaw-operator pattern generalised: opinionated about K8s mechanics, completely agnostic about the application runtime.

### 2. Configuration over Code

The operator injects two files into every agent container:

| Path | Content |
|------|---------|
| `/etc/agent/instructions.txt` | Task instructions (plain text) |
| `/etc/agent/config.yaml` | Personas, tools, models, agent identity |

Instructions are what the agent does. The image is how it does it. Changing instructions requires no image rebuild.

### 3. Delegate Specialised Concerns

Memory, knowledge retrieval, and code execution are handled by MCP tool servers (`LanguageTool` CRDs), not by the operator. The operator resolves tool endpoints and injects them into agent config — it does not proxy or inspect tool traffic.

LLM access is handled by `LanguageModel` CRDs, which back a per-model LiteLLM proxy. Agents route all LLM traffic through the proxy rather than connecting to model APIs directly. This allows the operator to manage credentials, token spend, and routing centrally.

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
│  │  · LanguageTool      │    │  Env vars injected:          │   │
│  │  · LanguageModel     │    │  · AGENT_NAME, AGENT_UUID    │   │
│  │  · LanguageCluster   │    │  · MODEL_ENDPOINTS           │   │
│  └──────────────────────┘    │  · TOOL_ENDPOINTS            │   │
│                               └──────────────────────────────┘   │
│  ┌──────────────────────┐    ┌──────────────────────────────┐   │
│  │    MCP Tool Servers  │    │     LiteLLM Proxies          │   │
│  │  (LanguageTool CRDs) │    │   (LanguageModel CRDs)       │   │
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
  name: my-agent
  namespace: language-operator-myapp
spec:
  image: myregistry/agent-runtime:v1.0.0
  port: 18789                               # defaults to 8080

  modelRefs:
    - name: claude-sonnet

  initContainers:
    - name: config-adapter
      image: myregistry/adapter:latest
      # MODEL_ENDPOINTS is injected into all init containers automatically

  livenessProbe:
    httpGet:
      path: /healthz
      port: 18789
    initialDelaySeconds: 15
    periodSeconds: 30

  readinessProbe:
    httpGet:
      path: /readyz
      port: 18789
    initialDelaySeconds: 5
    periodSeconds: 10

  envFrom:
    - secretRef:
        name: my-api-keys

  workspace:
    size: 10Gi
    mountPath: /home/node/.myapp

  executionMode: autonomous     # autonomous|interactive|scheduled|event-driven
  replicas: 1
```

The operator creates: Deployment, Service (on `spec.port`), HTTPRoute, NetworkPolicy, and two ConfigMaps (instructions, config).

If `initContainers` are specified, the operator prepends `MODEL_ENDPOINTS` and `LLM_MODEL` env vars into each init container so config adapters can bridge operator injection to native runtime config formats.

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

An LLM endpoint backed by a LiteLLM proxy. The operator deploys a per-model LiteLLM proxy that holds the real API credential. The proxy URL is injected as `MODEL_ENDPOINTS` into every agent container (main container and all init containers).

```yaml
apiVersion: langop.io/v1alpha1
kind: LanguageModel
metadata:
  name: claude-sonnet
  namespace: language-operator-myapp
spec:
  provider: anthropic
  modelName: claude-sonnet-4-5
  apiKeySecretRef:
    name: anthropic-credentials
    key: api-key
```

Agents route all LLM traffic through the proxy. They never hold real API credentials.

### LanguageCluster

A managed namespace. Each `LanguageCluster` corresponds 1:1 to a Kubernetes namespace with the same name. It is the logical boundary for a group of agents, models, and tools that belong together.

```yaml
apiVersion: langop.io/v1alpha1
kind: LanguageCluster
metadata:
  name: language-operator-myapp
spec:
  domain: agents.example.com
```

The operator creates the namespace, configures shared networking, and sets up default RBAC for agent service accounts within the cluster.

---

## Agent Runtime Contract

The full contract is defined in [`spec/agents.md`](../spec/agents.md). Summary:

**Operator provides:**
- `/etc/agent/instructions.txt` — plain text task instructions (optional)
- `/etc/agent/config.yaml` — structured YAML with agent identity, personas, tools, models (optional)
- Environment variables: `AGENT_NAME`, `AGENT_NAMESPACE`, `AGENT_UUID`, `AGENT_MODE`, `AGENT_CLUSTER_NAME`, `AGENT_CLUSTER_UUID`
- `MODEL_ENDPOINTS` — comma-separated LiteLLM proxy URLs (one per `modelRef`), injected into main container and all init containers
- `LLM_MODEL` — comma-separated model names corresponding to each proxy URL
- `TOOL_ENDPOINTS` — resolved MCP tool server URLs
- ClusterIP Service on `spec.port`
- HTTPRoute for external access
- NetworkPolicy allowing agent-to-agent traffic

**Agent image:** No mandatory endpoints or protocols. The operator is runtime-agnostic. Liveness and readiness probes are defined in `spec.livenessProbe` / `spec.readinessProbe` — if not set, none are configured.

---

## Network Architecture

Each `LanguageAgent` gets:

- **ClusterIP Service** (`<agent-name>.<namespace>.svc.cluster.local:<port>`) — in-cluster access on `spec.port`
- **HTTPRoute** — external access via the cluster Gateway
- **NetworkPolicy** with ingress rules allowing:
  - Agent pods (`langop.io/kind=LanguageAgent`) — agent-to-agent traffic on `spec.port`
  - Operator dashboard
  - Configured trigger pods

Tool servers (`LanguageTool`) get their own Services. The operator reconciles NetworkPolicy to allow agent pods to reach tool pods.

LiteLLM proxies (`LanguageModel`) get their own Services and NetworkPolicy rules that permit outbound HTTPS to the upstream model provider.

---

## Observability

All operator reconciliation loops emit OpenTelemetry spans. Configure an external OTel collector endpoint via `OTEL_EXPORTER_OTLP_ENDPOINT` if you want to collect traces.

**Key trace attributes:**
- `agent.name`, `agent.namespace`, `agent.uuid`
- `persona.name`, `tool.name`, `model.name`

---

## Extensibility

**Custom agent runtimes**: Any container image works. The operator is runtime-agnostic — Python, Node.js, Go, or anything else. Use init containers to bridge operator config injection to native runtime config formats (see `components/agents/openclaw-adapter/` for an example).

**Custom tools**: Any service that implements the tool contract ([`spec/tools.md`](../spec/tools.md)) — MCP protocol on a Kubernetes Service — can be registered as a `LanguageTool`.

**Custom models**: Any LLM endpoint supported by LiteLLM can be registered as a `LanguageModel`.
