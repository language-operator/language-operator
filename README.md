# Language Operator

A Kubernetes operator for running AI agent clusters as native workloads.

## What It Does

Language Operator provides a purpose-built set of CRDs for deploying and managing scalable AI agent clusters on Kubernetes:

| Resource | Purpose |
|----------|---------|
| `LanguageCluster` | Managed namespace for AI clusters |
| `LanguageAgent` | Autonomous, scheduled, and reactive agents |
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

This example deploys [openclaw](https://github.com/openclaw/openclaw) — a self-hosted AI assistant — to demonstrate the operator's deployment mechanics. LLM traffic routes through an operator-managed LiteLLM proxy rather than connecting to model APIs directly.

### 1. Create a cluster

A `LanguageCluster` is a managed namespace for logically grouped agents, models, and tools.

```bash
kubectl apply -f - <<EOF
apiVersion: langop.io/v1alpha1
kind: LanguageCluster
metadata:
  name: language-operator-openclaw
spec:
  domain: openclaw.langop.io
EOF
```

### 2. Configure an LLM

The `LanguageModel` holds the real API credential and exposes a LiteLLM proxy inside the cluster.

```bash
kubectl create secret generic anthropic-credentials \
  -n language-operator-openclaw \
  --from-literal=api-key=sk-ant-...

kubectl apply -n language-operator-openclaw -f - <<EOF
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

### 3. Create an openclaw gateway token

```bash
kubectl create secret generic openclaw-gateway \
  -n language-operator-openclaw \
  --from-literal=OPENCLAW_GATEWAY_TOKEN=$(openssl rand -hex 32)
```

### 4. Deploy openclaw

The `openclaw-adapter` init container receives the resolved LiteLLM proxy URL via `MODEL_ENDPOINTS` (injected by the operator) and seeds `openclaw.json` so openclaw routes through the proxy on first run.

```bash
kubectl apply -n language-operator-openclaw -f - <<EOF
apiVersion: langop.io/v1alpha1
kind: LanguageAgent
metadata:
  name: openclaw
spec:
  image: ghcr.io/openclaw/openclaw:latest
  port: 18789
  models:
    - name: claude-sonnet
  initContainers:
    - name: openclaw-adapter
      image: ghcr.io/language-operator/openclaw-adapter:latest
      env:
        - name: OPENCLAW_STATE_DIR
          value: /workspace/.openclaw
      volumeMounts:
        - name: workspace
          mountPath: /workspace
  env:
    - name: OPENCLAW_HOME
      value: /workspace
  envFrom:
    - secretRef:
        name: openclaw-gateway
  workspace:
    size: 10Gi
EOF
```

### 5. Check status

```bash
kubectl get languageagents -n language-operator-openclaw
kubectl get pods -n language-operator-openclaw
```

See [examples/openclaw.yaml](examples/openclaw.yaml) for the full annotated example.

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
