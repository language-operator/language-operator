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

### Install the Operator

```bash
helm repo add language-operator \
  https://language-operator.github.io/language-operator
helm install language-operator language-operator/language-operator
```

## Getting Started

These examples deploy [openclaw](https://github.com/openclaw/openclaw) or [opencode](https://github.com/sst/opencode) — self-hosted AI coding assistants — to demonstrate the operator's deployment mechanics. LLM traffic routes through an operator-managed LiteLLM proxy rather than connecting to model APIs directly.

### 1. Install standard runtimes

`LanguageAgentRuntime` is a cluster-scoped preset that packages the image, port, init containers, probes, and env vars for a specific agent type. Install once, use across any namespace.

```bash
kubectl apply -f runtimes/openclaw.yaml
kubectl apply -f runtimes/opencode.yaml
```

### 2. Create a cluster

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

### 3. Configure an LLM

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

### 4. Deploy an agent

Choose one of the following agents:

<details open>
<summary><strong>openclaw</strong></summary>

The `openclaw` runtime preset handles the image, port, init container, and env vars. Reference it with `runtime: openclaw` and the operator fills in the rest.

```bash
kubectl create secret generic openclaw-gateway \
  --from-literal=OPENCLAW_GATEWAY_TOKEN=$(openssl rand -hex 32)

kubectl apply -f - <<EOF
apiVersion: langop.io/v1alpha1
kind: LanguageAgent
metadata:
  name: openclaw
spec:
  runtime: openclaw
  models:
    - name: claude-sonnet
  deployment:
    envFrom:
      - secretRef:
          name: openclaw-gateway
EOF
```

See [runtimes/openclaw.yaml](runtimes/openclaw.yaml) for the full runtime definition and [examples/openclaw.yaml](examples/openclaw.yaml) for a self-contained example without a runtime.

</details>

<details>
<summary><strong>opencode</strong></summary>

The `opencode` runtime preset handles the image, port, args, init container, volumes, env vars, and health probes.

```bash
kubectl create secret generic opencode-server \
  --from-literal=OPENCODE_SERVER_USERNAME=demo \
  --from-literal=OPENCODE_SERVER_PASSWORD=$(openssl rand -hex 32)

kubectl apply -f - <<EOF
apiVersion: langop.io/v1alpha1
kind: LanguageAgent
metadata:
  name: opencode
spec:
  runtime: opencode
  models:
    - name: claude-sonnet
  deployment:
    envFrom:
      - secretRef:
          name: opencode-server
EOF
```

See [runtimes/opencode.yaml](runtimes/opencode.yaml) for the full runtime definition and [examples/opencode.yaml](examples/opencode.yaml) for a self-contained example without a runtime.

**Connect:**

```bash
# Port-forward the service
kubectl port-forward svc/opencode 3000:3000

# Open in browser — use Basic Auth: username "demo", password from the secret
open http://localhost:3000
kubectl get secret opencode-server \
  -o jsonpath='{.data.OPENCODE_SERVER_PASSWORD}' | base64 -d

# Or attach the TUI (opencode v1.0.10+)
opencode attach http://localhost:3000 \
  --password $(kubectl get secret opencode-server \
    -o jsonpath='{.data.OPENCODE_SERVER_PASSWORD}' | base64 -d)
```

</details>

### 5. Check status

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
