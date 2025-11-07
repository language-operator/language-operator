# Language Operator Architecture

**Language Operator** is a Kubernetes operator that orchestrates autonomous AI agents, LLM proxies, and MCP tool servers as native Kubernetes resources. The system is built on an intentionally decoupled architecture where the operator defines the platform and components are pluggable implementations.

---

## Table of Contents

1. [Architectural Principles](#architectural-principles)
2. [System Overview](#system-overview)
3. [Core Components](#core-components)
4. [Custom Resource Definitions (CRDs)](#custom-resource-definitions-crds)
5. [Agent Synthesis Pipeline](#agent-synthesis-pipeline)
6. [Component Architecture](#component-architecture)
7. [Network Architecture](#network-architecture)
8. [Data Flow](#data-flow)
9. [Extensibility](#extensibility)

---

## Architectural Principles

### 1. **Intentional Decoupling**
The operator and components are deliberately decoupled:
- **Operator**: Defines CRDs, manages Kubernetes resources, handles lifecycle
- **Components**: Pluggable implementations that can be replaced
- **Contract**: Components communicate via environment variables and standard protocols (HTTP, MCP)

**Why?** This allows users to:
- Build custom agents in any language
- Replace components with alternative implementations
- Extend functionality without modifying the operator

### 2. **Kubernetes-Native**
Everything is a Kubernetes resource:
- Agents, models, tools, personas are CRDs
- Standard `kubectl` commands work
- Native RBAC, namespaces, and quotas apply
- GitOps-friendly (declarative YAML)

### 3. **Zero Configuration**
Components auto-configure from environment variables injected by the operator:
- `MCP_SERVERS`: JSON array of tool server URLs
- `MODEL_ENDPOINTS`: JSON array of LLM proxy URLs
- `PERSONA_*`: Persona configuration
- No config files, no manual wiring

### 4. **Synthesis Over Configuration**
Instead of writing code, users write natural language instructions:
- Operator synthesizes executable DSL code via LLM
- Code stored in ConfigMaps, mounted to agent containers
- Auto-loads on boot, no rebuild required

---

## System Overview

```
┌─────────────────────────────────────────────────────────────────────┐
│                         Kubernetes Cluster                          │
│                                                                     │
│  ┌─────────────────────────────────────────────────────────────┐  │
│  │                    Language Operator                         │  │
│  │  ┌──────────────────────────────────────────────────────┐   │  │
│  │  │  Controllers (Go)                                     │   │  │
│  │  │  • LanguageCluster    • LanguageAgent                │   │  │
│  │  │  • LanguageModel      • LanguageTool                 │   │  │
│  │  │  • LanguagePersona    • LanguageClient               │   │  │
│  │  └──────────────────────────────────────────────────────┘   │  │
│  │  ┌──────────────────────────────────────────────────────┐   │  │
│  │  │  Synthesis Engine (pkg/synthesis)                    │   │  │
│  │  │  • Natural Language → Ruby DSL                       │   │  │
│  │  │  • Persona Distillation                              │   │  │
│  │  │  • Smart Change Detection                            │   │  │
│  │  └──────────────────────────────────────────────────────┘   │  │
│  └─────────────────────────────────────────────────────────────┘  │
│                                                                     │
│  ┌─────────────────────────────────────────────────────────────┐  │
│  │                Component Pods (User-Defined)                 │  │
│  │                                                              │  │
│  │  ┌──────────────┐  ┌──────────────┐  ┌──────────────┐     │  │
│  │  │ LLM Proxy    │  │ Tool Server  │  │ Agent Pod    │     │  │
│  │  │ (LiteLLM)    │  │ (MCP)        │  │              │     │  │
│  │  │              │  │              │  │ ┌──────────┐ │     │  │
│  │  │ • OpenAI     │  │ • Email      │  │ │  Agent   │ │     │  │
│  │  │ • Anthropic  │  │ • Web Search │  │ │ Container│ │     │  │
│  │  │ • Ollama     │  │ • Custom     │  │ └──────────┘ │     │  │
│  │  │ • Local      │  │              │  │ ┌──────────┐ │     │  │
│  │  │              │  │              │  │ │  Tool    │ │     │  │
│  │  │              │  │              │  │ │ Sidecar  │ │     │  │
│  │  └──────────────┘  └──────────────┘  │ └──────────┘ │     │  │
│  │                                       └──────────────┘     │  │
│  └─────────────────────────────────────────────────────────────┘  │
└─────────────────────────────────────────────────────────────────────┘
```

---

## Core Components

### 1. **Language Operator** (Go)
**Location**: `src/`

The operator is a Kubernetes controller that:
- Watches CRD resources
- Reconciles desired state with actual state
- Creates/updates Deployments, Services, ConfigMaps, NetworkPolicies
- Synthesizes agent code from natural language
- Emits Kubernetes events for observability

**Key Packages**:
- `controllers/`: Reconciliation logic for each CRD
- `api/v1alpha1/`: CRD type definitions
- `pkg/synthesis/`: LLM-powered code synthesis
- `config/`: Kubernetes manifests (CRDs, RBAC, etc.)

**Not Included**: Business logic for agents, tools, or models. The operator only orchestrates.

### 2. **Langop Ruby SDK** (Optional Reference Implementation)
**Location**: `sdk/ruby/`

The `langop` gem is **one possible implementation** of the operator's contracts. It provides:
- Client library for MCP and LLM communication
- Agent execution framework
- DSL for defining agents and tools
- Tool server implementation

**Important**: You don't have to use the langop gem. You can build components in:
- Python
- JavaScript
- Go
- Any language that can run in a container

The operator doesn't care - it just sets environment variables.

### 3. **Component Images** (Reference Implementations)
**Location**: `components/`, `agents/`, `tools/`

These are **example implementations** built with the langop gem:
- `langop/agent`: Agent container that executes autonomous tasks
- `langop/model`: LiteLLM proxy wrapper
- `langop/tool`: MCP tool server wrapper

Users can replace these with custom implementations.

---

## Custom Resource Definitions (CRDs)

### LanguageCluster
**Purpose**: Multi-tenant namespace management

```yaml
apiVersion: langop.io/v1alpha1
kind: LanguageCluster
metadata:
  name: production
spec:
  namespace: langop-prod  # Creates this namespace
  resourceQuotas:
    requests.cpu: "10"
    requests.memory: "20Gi"
```

**What the operator does**:
- Creates namespace if it doesn't exist
- Sets resource quotas
- Tracks status (Ready/Pending)

### LanguageModel
**Purpose**: LLM proxy deployment

```yaml
apiVersion: langop.io/v1alpha1
kind: LanguageModel
metadata:
  name: claude
spec:
  provider: anthropic
  model: claude-sonnet-4-5
  apiKeySecretRef:
    name: anthropic-key
    key: api-key
```

**What the operator does**:
- Deploys LiteLLM proxy (or your custom proxy)
- Creates Service for HTTP access
- Injects API keys from secrets
- Sets `MODEL_ENDPOINTS` env var in agents that reference this model

**Decoupling**: You can replace the proxy with any HTTP server that accepts OpenAI-compatible requests.

### LanguageTool
**Purpose**: MCP tool server deployment

```yaml
apiVersion: langop.io/v1alpha1
kind: LanguageTool
metadata:
  name: email-tool
spec:
  type: mcp
  image: langop/tool-email:latest
  deploymentMode: sidecar  # or "service"
  port: 8080
```

**What the operator does**:
- **Service mode**: Creates Deployment + Service
- **Sidecar mode**: Injects container into agent pods
- Sets `MCP_SERVERS` env var in agents that reference this tool

**Decoupling**: Any MCP-compatible server works. Language and implementation don't matter.

### LanguagePersona
**Purpose**: Reusable agent personality/behavior templates

```yaml
apiVersion: langop.io/v1alpha1
kind: LanguagePersona
metadata:
  name: helpful-assistant
spec:
  description: A helpful, friendly assistant
  systemPrompt: You are a helpful assistant...
  tone: professional
  language: en
```

**What the operator does**:
- Stores persona configuration
- Distills into concise system message during synthesis
- Injects as `PERSONA_*` env vars

### LanguageAgent
**Purpose**: Autonomous AI agent deployment

```yaml
apiVersion: langop.io/v1alpha1
kind: LanguageAgent
metadata:
  name: monitor-agent
spec:
  instructions: |
    Monitor the workspace directory for changes.
    Alert via email when critical files are modified.

  modelRefs:
    - name: claude

  toolRefs:
    - name: email-tool
    - name: filesystem-tool

  personaRef:
    name: helpful-assistant

  workspace:
    enabled: true
    storage: 10Gi
```

**What the operator does**:
1. **Synthesizes code**: Calls LLM to generate Ruby DSL from `instructions`
2. **Creates ConfigMap**: Stores synthesized code
3. **Creates Deployment/CronJob**: Runs agent container
4. **Injects env vars**: `MCP_SERVERS`, `MODEL_ENDPOINTS`, `PERSONA_*`
5. **Mounts volumes**: Code ConfigMap, workspace PVC
6. **Creates NetworkPolicy**: Restricts egress to only allowed resources
7. **Tracks status**: Synthesis metrics, execution counts, errors

**Decoupling**: The operator doesn't care what language your agent is written in. It just:
- Mounts `/etc/agent/code/agent.rb` (or whatever path you use)
- Sets environment variables
- Your container does the rest

---

## Agent Synthesis Pipeline

The synthesis pipeline is a **key differentiator** - users write natural language instead of code.

### Architecture

```
┌─────────────────────────────────────────────────────────────────┐
│                     Synthesis Pipeline                          │
│                                                                 │
│  User writes:                                                   │
│  ┌─────────────────────────────────────────────────────────┐   │
│  │ instructions: "Monitor workspace, alert on changes"     │   │
│  └─────────────────────────────────────────────────────────┘   │
│                            │                                    │
│                            ▼                                    │
│  ┌─────────────────────────────────────────────────────────┐   │
│  │ Operator reconciles LanguageAgent                       │   │
│  │ • Detects instructions changed (SHA256 hash)            │   │
│  │ • Calls synthesis.SynthesizeAgent()                     │   │
│  └─────────────────────────────────────────────────────────┘   │
│                            │                                    │
│                            ▼                                    │
│  ┌─────────────────────────────────────────────────────────┐   │
│  │ Synthesizer (pkg/synthesis)                             │   │
│  │ • Builds structured prompt with examples                │   │
│  │ • Calls LLM (via gollm)                                 │   │
│  │ • Extracts code from markdown                           │   │
│  │ • Validates syntax (basic checks)                       │   │
│  │ • Returns DSL code + metrics                            │   │
│  └─────────────────────────────────────────────────────────┘   │
│                            │                                    │
│                            ▼                                    │
│  ┌─────────────────────────────────────────────────────────┐   │
│  │ Generated DSL code:                                     │   │
│  │                                                         │   │
│  │ require 'language_operator'                             │   │
│  │                                                         │   │
│  │ agent 'monitor-agent' do                               │   │
│  │   schedule every: '5m'                                  │   │
│  │                                                         │   │
│  │   objectives [                                          │   │
│  │     "Monitor /workspace for file changes",             │   │
│  │     "Send email alerts for critical files"             │   │
│  │   ]                                                     │   │
│  │                                                         │   │
│  │   workflow do                                           │   │
│  │     step "Check workspace" do                          │   │
│  │       use_tool "filesystem", action: "list_files"      │   │
│  │     end                                                 │   │
│  │                                                         │   │
│  │     step "Alert if changed" do                         │   │
│  │       use_tool "email", action: "send"                 │   │
│  │     end                                                 │   │
│  │   end                                                   │   │
│  │ end                                                     │   │
│  └─────────────────────────────────────────────────────────┘   │
│                            │                                    │
│                            ▼                                    │
│  ┌─────────────────────────────────────────────────────────┐   │
│  │ Operator stores code in ConfigMap                       │   │
│  │ • Key: agent.rb                                         │   │
│  │ • Annotations: hashes for change detection              │   │
│  │   - langop.io/instructions-hash: abc123                 │   │
│  │   - langop.io/tools-hash: def456                        │   │
│  │   - langop.io/models-hash: ghi789                       │   │
│  │   - langop.io/persona-hash: jkl012                      │   │
│  └─────────────────────────────────────────────────────────┘   │
│                            │                                    │
│                            ▼                                    │
│  ┌─────────────────────────────────────────────────────────┐   │
│  │ Operator creates Deployment                             │   │
│  │ • Mounts ConfigMap at /etc/agent/code/agent.rb          │   │
│  │ • Injects MCP_SERVERS, MODEL_ENDPOINTS env vars        │   │
│  └─────────────────────────────────────────────────────────┘   │
│                            │                                    │
│                            ▼                                    │
│  ┌─────────────────────────────────────────────────────────┐   │
│  │ Agent container starts                                  │   │
│  │ • Entrypoint: /usr/local/bin/langop-agent               │   │
│  │ • Auto-loads /etc/agent/code/agent.rb                   │   │
│  │ • Executes synthesized workflow                         │   │
│  └─────────────────────────────────────────────────────────┘   │
└─────────────────────────────────────────────────────────────────┘
```

### Smart Change Detection

The operator uses **surgical re-synthesis** to avoid unnecessary LLM calls:

| Change Type | Action | Why? |
|------------|--------|------|
| **Instructions changed** | Full re-synthesis | Code logic needs updating |
| **Persona changed** | Re-distill only | Update context, code stays same |
| **Tools/Models changed** | No synthesis | Env vars auto-update |
| **Nothing changed** | Use cached code | No work needed |

**Implementation**:
- Store SHA256 hashes in ConfigMap annotations
- Compare current vs. previous hashes on reconciliation
- Route to appropriate handler

**Benefits**:
- Reduces LLM costs (only call when truly needed)
- Faster reconciliation for non-code changes
- Preserves code stability

### Synthesis Configuration

The operator's synthesis engine is configured via environment variables:

```yaml
env:
  - name: SYNTHESIS_MODEL
    value: "claude-sonnet-4-5"
  - name: SYNTHESIS_PROVIDER
    value: "anthropic"  # or "openai", "ollama"
  - name: SYNTHESIS_API_KEY
    valueFrom:
      secretKeyRef:
        name: anthropic-key
        key: api-key
  - name: SYNTHESIS_ENDPOINT
    value: "http://ollama:11434"  # Optional, for local LLMs
```

**If `SYNTHESIS_MODEL` is not set**: Synthesis is disabled. Agents require pre-built images.

---

## Component Architecture

Components are intentionally decoupled from the operator. The operator provides the **platform**, components provide the **implementation**.

### Contract: Environment Variables

The operator injects configuration via standard environment variables:

#### For Agents:
```bash
# MCP Tool Servers (JSON array)
MCP_SERVERS='[
  {"name":"email","url":"http://email-tool:8080"},
  {"name":"filesystem","url":"http://filesystem-tool:8080"}
]'

# LLM Model Endpoints (JSON array)
MODEL_ENDPOINTS='[
  {"name":"claude","url":"http://claude-proxy:8000","model":"claude-sonnet-4-5"},
  {"name":"gpt","url":"http://openai-proxy:8000","model":"gpt-4"}
]'

# Persona Configuration
PERSONA_NAME="helpful-assistant"
PERSONA_DESCRIPTION="A helpful, friendly assistant"
PERSONA_SYSTEM_PROMPT="You are a helpful assistant..."
PERSONA_TONE="professional"
PERSONA_LANGUAGE="en"

# Agent Configuration
AGENT_NAME="monitor-agent"
AGENT_NAMESPACE="demo"
INSTRUCTIONS="Monitor workspace for changes..."
```

#### For Models:
```bash
# LiteLLM configuration
LITELLM_PROVIDER="anthropic"
LITELLM_MODEL="claude-sonnet-4-5"
OPENAI_API_KEY="sk-..."  # Or ANTHROPIC_API_KEY, etc.
```

#### For Tools:
```bash
# Tool-specific configuration (user-defined)
SMTP_HOST="smtp.example.com"
SMTP_PORT="587"
# ... etc
```

### Reference Implementation: langop Gem

The `langop` Ruby gem is **one way** to implement these contracts. It provides:

#### 1. **Client Library** (`Langop::Client::Base`)
- Parses `MCP_SERVERS` and `MODEL_ENDPOINTS`
- Connects to MCP servers
- Configures LLM client
- Provides helper methods

```ruby
class MyAgent < Langop::Client::Base
  def initialize
    super
    # MCP servers and models auto-configured from env vars
  end
end
```

#### 2. **Agent Framework** (`Langop::Agent::Executor`)
- Autonomous execution loop
- Iteration limits
- Error handling
- Workspace integration

```ruby
executor = Langop::Agent::Executor.new(
  client: my_agent,
  instructions: ENV['INSTRUCTIONS'],
  max_iterations: 10
)
executor.run
```

#### 3. **DSL** (`Langop::Dsl`)
- Define agents declaratively
- Define tools with parameters
- Workflow steps with dependencies

```ruby
agent 'my-agent' do
  schedule every: '1h'

  objectives [
    "Monitor resources",
    "Alert on issues"
  ]

  workflow do
    step "check" do
      use_tool "monitoring", action: "check_status"
    end

    step "alert", depends_on: "check" do
      use_tool "email", action: "send"
    end
  end
end
```

#### 4. **Tool Server** (`Langop::ToolLoader`)
- Loads tool definitions from Ruby files
- Exposes MCP-compatible HTTP server
- Parameter validation

```ruby
tool "email" do
  description "Send email notifications"

  parameter :to, type: :string, required: true
  parameter :subject, type: :string, required: true
  parameter :body, type: :string, required: true

  execute do |params|
    # Send email
  end
end
```

### Building Custom Components

You **don't need** the langop gem. Build components in any language:

#### Python Agent Example:
```python
import os
import json

# Parse MCP_SERVERS from env
mcp_servers = json.loads(os.environ['MCP_SERVERS'])

# Parse MODEL_ENDPOINTS from env
models = json.loads(os.environ['MODEL_ENDPOINTS'])

# Load code if synthesized
code_path = '/etc/agent/code/agent.py'
if os.path.exists(code_path):
    exec(open(code_path).read())

# Your agent logic here
```

#### Go Tool Server Example:
```go
package main

import (
    "encoding/json"
    "net/http"
    "os"
)

func main() {
    // Implement MCP protocol
    http.HandleFunc("/tools", toolsHandler)
    http.HandleFunc("/call", callHandler)

    port := os.Getenv("PORT")
    http.ListenAndServe(":"+port, nil)
}
```

The operator doesn't care. It just:
1. Sets environment variables
2. Mounts volumes
3. Creates network policies
4. Your code does the rest

---

## Network Architecture

### Namespace Isolation

Each `LanguageCluster` creates an isolated namespace:
- Resource quotas
- Network policies
- RBAC boundaries

### Network Policies

The operator creates **egress-only** NetworkPolicies for agents:
- Allow DNS (kube-system/kube-dns)
- Allow tools (referenced LanguageTool services)
- Allow models (referenced LanguageModel services)
- **Deny everything else**

```yaml
apiVersion: networking.k8s.io/v1
kind: NetworkPolicy
metadata:
  name: monitor-agent
spec:
  podSelector:
    matchLabels:
      langop.io/agent: monitor-agent
  policyTypes:
    - Egress
  egress:
    # DNS
    - to:
      - namespaceSelector:
          matchLabels:
            kubernetes.io/metadata.name: kube-system
        podSelector:
          matchLabels:
            k8s-app: kube-dns
      ports:
        - protocol: UDP
          port: 53

    # Email Tool
    - to:
      - podSelector:
          matchLabels:
            langop.io/tool: email-tool
      ports:
        - protocol: TCP
          port: 8080

    # Claude Model
    - to:
      - podSelector:
          matchLabels:
            langop.io/model: claude
      ports:
        - protocol: TCP
          port: 8000
```

**Security**: Agents can only communicate with explicitly referenced resources.

### Service Discovery

Tools and models are discovered via Kubernetes Services:
- `email-tool.demo.svc.cluster.local:8080`
- `claude-proxy.demo.svc.cluster.local:8000`

The operator constructs URLs and injects them in `MCP_SERVERS` and `MODEL_ENDPOINTS`.

---

## Data Flow

### Agent Execution Flow

```
1. User creates LanguageAgent
   ↓
2. Operator reconciles
   ↓
3. Synthesis (if instructions present)
   ├─ Call LLM
   ├─ Generate DSL code
   └─ Store in ConfigMap
   ↓
4. Create Deployment
   ├─ Mount code ConfigMap
   ├─ Inject MCP_SERVERS env var
   ├─ Inject MODEL_ENDPOINTS env var
   └─ Create NetworkPolicy
   ↓
5. Pod starts
   ├─ Entrypoint: /usr/local/bin/langop-agent
   └─ Auto-load /etc/agent/code/agent.rb
   ↓
6. Agent executes
   ├─ Parse env vars (MCP_SERVERS, MODEL_ENDPOINTS)
   ├─ Connect to MCP servers
   ├─ Configure LLM client
   └─ Run autonomous loop
   ↓
7. Agent calls tools
   ├─ HTTP POST to http://email-tool:8080/call
   └─ MCP protocol
   ↓
8. Agent calls LLM
   ├─ HTTP POST to http://claude-proxy:8000/v1/chat/completions
   └─ OpenAI-compatible API
   ↓
9. Operator updates status
   ├─ Execution count
   ├─ Success/failure counts
   └─ Last execution time
```

### Synthesis Flow

```
1. User updates LanguageAgent.spec.instructions
   ↓
2. Operator detects change (hash comparison)
   ↓
3. Call synthesis.SynthesizeAgent()
   ├─ Build prompt with tools/models context
   ├─ Call LLM (gollm)
   ├─ Parse response
   ├─ Validate syntax
   └─ Return DSL code + metrics
   ↓
4. Update ConfigMap
   ├─ Store new code
   ├─ Update hash annotations
   └─ Update timestamp
   ↓
5. Update agent status
   ├─ SynthesisInfo.LastSynthesisTime
   ├─ SynthesisInfo.SynthesisDuration
   ├─ SynthesisInfo.CodeHash
   └─ SynthesisInfo.SynthesisAttempts++
   ↓
6. Emit Kubernetes events
   ├─ SynthesisStarted
   ├─ SynthesisSucceeded
   └─ (or SynthesisFailed)
   ↓
7. Deployment restarts (ConfigMap changed)
   └─ Agent loads new code
```

---

## Extensibility

### Custom Components

The operator is designed for **maximum extensibility**:

#### 1. **Custom Agents**
Build agents in any language:
- Read `MCP_SERVERS` and `MODEL_ENDPOINTS` from environment
- Load synthesized code from `/etc/agent/code/*` (optional)
- Implement your business logic
- Set `spec.image` to your custom image

#### 2. **Custom Tool Servers**
Implement the MCP protocol in any language:
- `POST /tools` - List available tools
- `POST /call` - Execute tool with parameters
- Set `spec.image` to your custom image

#### 3. **Custom Model Proxies**
Implement OpenAI-compatible API:
- `POST /v1/chat/completions`
- Transform requests/responses as needed
- Set `spec.image` to your custom image

#### 4. **Custom Synthesis**
Replace the synthesis engine:
- Modify `pkg/synthesis/synthesizer.go`
- Implement `AgentSynthesizer` interface
- Use different LLM or prompting strategy

### Extending CRDs

Add custom fields to CRDs:
1. Edit `api/v1alpha1/*_types.go`
2. Run `make manifests` to regenerate CRDs
3. Update controller logic

Example:
```go
// Add to LanguageAgentSpec
type LanguageAgentSpec struct {
    // ... existing fields

    // MyCustomField adds custom configuration
    // +optional
    MyCustomField string `json:"myCustomField,omitempty"`
}
```

---

## Deployment Architecture

### Component Hierarchy

```
langop/base (Alpine + Ruby + Bundler)
  │
  ├─ langop/ruby (base + langop gem pre-installed)
  │   │
  │   ├─ langop/agent (ruby + agent entrypoint)
  │   │
  │   ├─ langop/tool (ruby + tool server)
  │   │
  │   └─ langop/model (ruby + LiteLLM wrapper)
  │
  └─ langop/client (base + client wrapper, deprecated)
```

**Key Insight**: The `langop` gem is pre-installed in `langop/ruby`, so all child images inherit it. This eliminates Gemfile duplication and ensures consistency.

### CI/CD Pipeline

Build order is critical due to dependencies:

```
1. Build langop/base
   ↓
2. Build langop/ruby (depends on base)
   ↓
3. Build langop/agent, langop/tool, langop/model (parallel, depend on ruby)
   ↓
4. Build specific agents (agents/cli, agents/headless, etc.)
   ↓
5. Build specific tools (tools/email, tools/web, etc.)
```

Managed by `.github/workflows/build-images.yaml`.

---

## Key Design Decisions

### 1. **Why Decoupled?**
- **Flexibility**: Users can replace any component
- **Language Agnostic**: Build in Python, Go, Rust, etc.
- **Evolution**: Operator can evolve independently of components

### 2. **Why Environment Variables?**
- **12-Factor App**: Industry standard for container configuration
- **Language Agnostic**: Every language can read env vars
- **Kubernetes Native**: Secrets, ConfigMaps integrate seamlessly

### 3. **Why Synthesis?**
- **Accessibility**: Non-developers can create agents
- **Speed**: Faster than writing code
- **Iteration**: Easy to modify behavior (just change instructions)
- **Cost**: Synthesis LLM call is cheaper than developer time

### 4. **Why MCP?**
- **Standardization**: Model Context Protocol is emerging standard
- **Interoperability**: Tools work across different agent frameworks
- **Security**: HTTP-based, easy to network-isolate

### 5. **Why Kubernetes?**
- **Orchestration**: Deployment, scaling, self-healing built-in
- **Multi-Tenancy**: Namespaces, quotas, RBAC
- **Ecosystem**: Monitoring, logging, service mesh integrate easily
- **GitOps**: Declarative, version-controlled infrastructure

---

## Performance Characteristics

### Synthesis Performance
- **Cold start** (first synthesis): 2-5 seconds (LLM call)
- **Warm start** (cached code): <100ms (hash comparison)
- **Re-synthesis** (instructions change): 2-5 seconds
- **Persona update**: <1 second (re-distill only)
- **Tool/model update**: <100ms (no synthesis)

### Agent Performance
- **Startup time**: 1-3 seconds (Ruby + gem loading)
- **Iteration latency**: 500ms - 2s (depends on LLM)
- **Tool call latency**: 10-100ms (HTTP to service)

### Scalability
- **Agents per cluster**: Thousands (limited by node resources)
- **Tools per agent**: Unlimited (NetworkPolicy has no limit)
- **Models per agent**: Unlimited

---

## Security Model

### Principle of Least Privilege
- Agents can **only** access referenced tools and models
- NetworkPolicies enforce egress restrictions
- RBAC controls who can create/modify resources

### Secret Management
- API keys stored in Kubernetes Secrets
- Mounted as environment variables or files
- Never logged or exposed in status

### Code Isolation
- Synthesized code stored in ConfigMaps (immutable on creation)
- Read-only mount in agent pods
- Code hash tracked in status for audit trail

---

## Observability

### Kubernetes Events
The operator emits events for key lifecycle moments:
- `SynthesisStarted` / `SynthesisSucceeded` / `SynthesisFailed`
- `ValidationFailed`
- `PersonaUpdated`
- `DeploymentCreated` / `DeploymentUpdated`

View with:
```bash
kubectl describe languageagent my-agent
kubectl events -n demo
```

### Status Fields
Each CRD exposes rich status:
- **LanguageAgent**:
  - `phase`: Pending/Running/Succeeded/Failed
  - `synthesisInfo`: Synthesis metrics
  - `executionCount`, `successfulExecutions`, `failedExecutions`
  - `lastExecutionTime`, `lastExecutionResult`
  - `activeReplicas`, `readyReplicas`

- **LanguageModel**:
  - `phase`: Pending/Running/Failed
  - `endpoint`: Service URL
  - `readyReplicas`

- **LanguageTool**:
  - `phase`: Pending/Running/Failed
  - `endpoint`: Service URL (service mode)
  - `deploymentMode`: "service" or "sidecar"

### Metrics
(Future enhancement)
- Prometheus metrics for synthesis latency, success rate
- Agent execution metrics
- Tool call metrics

---

## Future Directions

### Planned Enhancements
1. **Synthesis Caching**: Cache LLM responses for identical instructions
2. **Multi-Language Support**: Generate Python, JS, Go from instructions
3. **Cost Tracking**: Track LLM token usage per agent
4. **Advanced Scheduling**: Cron expressions, event triggers
5. **Agent Collaboration**: Agents can call other agents as tools
6. **Hosted Components**: SaaS-style shared tools and models

### Research Areas
1. **Incremental Synthesis**: Only regenerate changed workflow steps
2. **Code Verification**: Formal verification of synthesized code
3. **Agent Learning**: Agents improve over time from feedback
4. **Distributed Execution**: Workflow steps run on different nodes

---

## Summary

**Language Operator** is a Kubernetes-native platform for autonomous AI agents with these key architectural properties:

1. **Decoupled**: Operator provides the platform, components are pluggable
2. **Synthesis-First**: Natural language → executable code
3. **Kubernetes-Native**: Everything is a CRD, standard tooling works
4. **Zero-Config**: Components auto-configure from environment
5. **Secure by Default**: NetworkPolicies enforce least privilege
6. **Language-Agnostic**: Build components in any language
7. **Observable**: Events, status, and logs provide full visibility

The `langop` gem is **one reference implementation**. The real power is the platform - users can build custom agents, tools, and models in any language, and the operator orchestrates them as native Kubernetes resources.
