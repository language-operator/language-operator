# Language Operator

A Kubernetes operator for running AI agent clusters as native workloads.

## What It Does

Language Operator provides a purpose-built set of CRDs for deploying and managing scalable AI agent clusters on Kubernetes:

| Resource | Purpose |
|----------|---------|
| `LanguageCluster` | Managed namespace for AI clusters |
| `LanguageAgent` | Autonomous, scheduled, and reactive agents |
| `LanguageAgentRuntime` | Reusable agent preset (image, port, init containers, probes) |
| `LanguageModel` | LLM (proxied through LiteLLM) |
| `LanguageTool` | MCP server |
| `LanguagePersona` | Behavior, tone, constraints |


## Installation

### Requirements

- Kubernetes 1.26+
- NetworkPolicy-capable CNI (Cilium, Calico, Weave, Antrea)
- cert-manager v1.12+ (for webhook TLS — [install guide](docs/getting-started/installation.md#requirements))

### Install the Operator

```bash
helm repo add language-operator \
  https://language-operator.github.io/language-operator
helm install language-operator language-operator/language-operator
```

## Getting Started

These examples deploy [openclaw](https://github.com/openclaw/openclaw) or [opencode](https://github.com/sst/opencode) — self-hosted AI coding assistants — to demonstrate the operator's deployment mechanics. LLM traffic routes through an operator-managed LiteLLM proxy rather than connecting to model APIs directly.

### 1. Create a cluster

A `LanguageCluster` is a managed namespace for logically grouped agents, models, and tools.

```bash
kubectl apply -f - <<EOF
apiVersion: langop.io/v1alpha1
kind: LanguageCluster
metadata:
  name: my-cluster
spec:
  domain: agents.example.com
EOF

kubectl config set-context --current --namespace=my-cluster
```

### 2. Configure an LLM

The `LanguageModel` holds the real API credential and exposes a LiteLLM proxy inside the cluster.

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

### 3. Deploy an agent

Choose one of the following agents:

<details open>
<summary><strong>openclaw</strong></summary>

The `openclaw` runtime preset handles the image, port, init container, and env vars. Reference it with `runtime: openclaw` and the operator fills in the rest.

```bash
kubectl apply -f - <<EOF
apiVersion: langop.io/v1alpha1
kind: LanguageAgent
metadata:
  name: openclaw
spec:
  runtime: openclaw
  openclaw:
    token: changeme
  models:
    - name: claude-sonnet
EOF
```


</details>

<details>
<summary><strong>opencode</strong></summary>

The `opencode` runtime preset handles the image, port, args, init container, volumes, env vars, and health probes.

```bash
kubectl apply -f - <<EOF
apiVersion: langop.io/v1alpha1
kind: LanguageAgent
metadata:
  name: opencode
spec:
  runtime: opencode
  opencode:
    username: demo
    password: changeme
  models:
    - name: claude-sonnet
EOF
```


**Connect:**

```bash
# Port-forward the service
kubectl port-forward svc/opencode 3000:3000

# Open in browser — use Basic Auth: username "demo", password from spec.opencode.password
open http://localhost:3000

# Or attach the TUI (opencode v1.0.10+)
opencode attach http://localhost:3000 --password changeme
```

</details>

### 4. Check status

```bash
kubectl get languageagentruntimes
kubectl get languageagents
kubectl get pods
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
- [Agent Contract](spec/agents.md) — what the operator injects into agent pods
- [Tool Contract](spec/tools.md) — how to implement a compatible MCP tool server

## Status

**Pre-release** — not ready for production.

## License

[MIT](LICENSE)
