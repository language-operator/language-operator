# Language Operator

A Kubernetes operator for running AI agents as native workloads.

## What It Does

Language Operator deploys AI agents as standard Kubernetes Deployments. You bring the container image; the operator handles everything else: configuration injection, networking, observability, and task state visibility.

```yaml
apiVersion: langop.io/v1alpha1
kind: LanguageAgent
metadata:
  name: data-analyst
spec:
  image: myregistry/agent-runtime:v1.0.0
  instructions: |
    You are a data analyst. Analyze CSV files and generate insights.
    Focus on trends, anomalies, and actionable recommendations.
  modelRefs:
    - name: claude-sonnet
  toolRefs:
    - name: python-executor
  executionMode: autonomous
```

Apply it and the agent is running, reachable over A2A, and visible in `kubectl`.

## How It Works

**The operator manages infrastructure. Your container manages reasoning.**

For every `LanguageAgent`, the operator:

- Deploys a Kubernetes Deployment with your image
- Mounts `/etc/agent/instructions.txt` — task instructions (plain text)
- Mounts `/etc/agent/config.yaml` — personas, tools, models, agent identity
- Creates a Service on port 8080 for agent-to-agent communication
- Creates a NetworkPolicy so agents can call each other directly
- Watches for task state changes via push notifications and surfaces blocked tasks as `LanguageAgentTask` resources

## A2A Protocol

Every agent speaks [Google's A2A protocol](https://a2a-protocol.org/latest/specification/). Agents discover each other, delegate tasks, and stream results without any orchestration from the operator.

```bash
# Discover what an agent can do
curl http://data-analyst.default.svc.cluster.local:8080/.well-known/agent.json

# Send it a task
curl -X POST http://data-analyst.default.svc.cluster.local:8080/messages \
  -H "Content-Type: application/json" \
  -d '{"message": {"role": "user", "parts": [{"text": "Analyze sales_q1.csv"}]}}'
```

## Task Observability

When an agent pauses waiting for input or credentials, the operator surfaces that state as a Kubernetes resource — not a failure:

```bash
kubectl get latask -A
# NAME                    AGENT          STATE            AGE
# data-analyst-abc123     data-analyst   input-required   2m

kubectl describe latask data-analyst-abc123
# ...
# Status:
#   State: input-required
#   Input Required:
#     Prompt: Which date range should I analyze?
#     Since:  2025-03-14T10:00:00Z
```

## CRDs

| Resource | Purpose |
|----------|---------|
| `LanguageAgent` | Agent deployment — image, instructions, personas, tools, models |
| `LanguageAgentTask` | In-flight task state — surfaced from A2A push notifications |
| `LanguagePersona` | Behavioral config — system prompt, tone, constraints |
| `LanguageTool` | MCP tool server — endpoint resolved and injected into agents |
| `LanguageModel` | LLM endpoint — provider, credentials, injected into agents |
| `LanguageCluster` | Multi-cluster agent grouping |

## Installation

### Requirements

- Kubernetes 1.26+
- NetworkPolicy-capable CNI (Cilium, Calico, Weave, Antrea)

### Install the Operator

```bash
helm repo add language-operator \
  https://language-operator.github.io/language-operator
helm install language-operator language-operator/language-operator
```

## Getting Started

### Create an Agent Cluster

A `LanguageCluster` is a logical grouping for agents, models, and tools in a namespace.

```bash
kubectl apply -f - <<EOF
apiVersion: langop.io/v1alpha1
kind: LanguageCluster
metadata:
  name: my-cluster
spec:
  domain: agents.example.com
EOF
```

### Register a Model

```bash
kubectl create secret generic anthropic-credentials \
  --from-literal=api-key=sk-ant-...

kubectl apply -f - <<EOF
apiVersion: langop.io/v1alpha1
kind: LanguageModel
metadata:
  name: claude-sonnet
spec:
  provider: anthropic
  modelName: claude-sonnet-4-5
  apiKeySecretRef:
    name: anthropic-credentials
    key: api-key
EOF
```

### Deploy an Agent

See [spec/agents.md](spec/agents.md) for the full agent runtime contract — what the operator injects and what your image must implement.

```bash
kubectl apply -f - <<EOF
apiVersion: langop.io/v1alpha1
kind: LanguageAgent
metadata:
  name: data-analyst
spec:
  image: myregistry/agent-runtime:v1.0.0
  instructions: |
    You are a data analyst. Analyze CSV files and generate insights.
  modelRefs:
    - name: claude-sonnet
  executionMode: autonomous
EOF

kubectl get languageagents
```

### Deploy a Tool

Tools are MCP servers. They can run as a shared service or as a sidecar in each agent pod. See [spec/tools.md](spec/tools.md) for the tool implementation contract.

**Standalone service** (shared across agents):

```bash
kubectl apply -f - <<EOF
apiVersion: langop.io/v1alpha1
kind: LanguageTool
metadata:
  name: python-executor
spec:
  image: myregistry/python-executor:v1.0.0
  deploymentMode: service
EOF
```

**Sidecar** (one instance per agent pod, has access to the agent's workspace):

```bash
kubectl apply -f - <<EOF
apiVersion: langop.io/v1alpha1
kind: LanguageTool
metadata:
  name: file-tools
spec:
  image: myregistry/file-tools:v1.0.0
  deploymentMode: sidecar
EOF
```

### Deploy a Tool-enabled Agent

```bash
kubectl apply -f - <<EOF
apiVersion: langop.io/v1alpha1
kind: LanguageAgent
metadata:
  name: data-analyst
spec:
  image: myregistry/agent-runtime:v1.0.0
  instructions: |
    You are a data analyst. Use the python-executor tool to analyze CSV files.
  modelRefs:
    - name: claude-sonnet
  toolRefs:
    - name: python-executor
  executionMode: autonomous
EOF
```

## Development

```bash
# Install git hooks
./scripts/setup-hooks

# Build
cd src && make build

# Test
cd src && make test

# Regenerate CRDs and deepcopy after type changes
cd src && make generate && make helm-crds
```

## Further Reading

- [Architecture](requirements/ARCHITECTURE.md) — system design and component interaction
- [Agent Runtime Contract](spec/agents.md) — what the operator provides and what agent images must implement
- [Tool Contract](spec/tools.md) — how to implement a compatible MCP tool server

## Status

**Pre-alpha** — core functionality works, actively developing toward stable A2A runtime.

## License

[MIT](LICENSE)
