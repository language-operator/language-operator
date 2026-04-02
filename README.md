# Language Operator

A Kubernetes operator for running AI agent clusters as native workloads.

## What It Does

Language Operator provides a purpose-built set of CRDs for deploying and managing agents in Kubernetes:

| Resource | Purpose |
|----------|---------|
| `LanguageCluster` | Managed namespace for agents |
| `LanguageAgent` | Free-form agents like OpenClaw or OpenCode |
| `LanguageAgentRuntime` | Agent runtime presets |
| `LanguageModel` | An LLM configuration (proxied through LiteLLM) |
| `LanguageTool` | A MCP-compatible server |
| `LanguagePersona` | Define tone, personality and expertise |


## Installation

### Requirements

- Kubernetes 1.26+
- NetworkPolicy-capable CNI (Cilium, Calico, Weave, Antrea)
- cert-manager v1.12+ (for webhook TLS — [install guide](docs/getting-started/installation.md#requirements))

### Recommended

These components are not required but enable the full feature set:

| Component | Purpose |
|-----------|---------|
| [cert-manager](https://cert-manager.io) with a [Let's Encrypt ClusterIssuer](https://cert-manager.io/docs/configuration/acme/) | Automatic TLS certificates for agent ingresses |
| [external-dns](https://github.com/kubernetes-sigs/external-dns) | Automatic DNS records for agent hostnames |

With both in place, deploying an agent automatically provisions a DNS record and a trusted TLS certificate at `<agent-name>.<cluster-domain>`.

### Install the Operator

```bash
helm repo add language-operator \
  https://language-operator.github.io/language-operator

helm install language-operator language-operator/language-operator \
  --create-namespace \
  --namespace language-operator
```

## Getting Started

These examples deploy [openclaw](https://github.com/openclaw/openclaw) or [opencode](https://github.com/sst/opencode) to demonstrate the operator's deployment mechanics. LLM traffic routes through an operator-managed LiteLLM proxy rather than connecting to model APIs directly.

### 1. Create a cluster

A `LanguageCluster` is a managed namespace for logically grouped agents, models, and tools.

```bash
kubectl apply -f - <<EOF
apiVersion: langop.io/v1alpha1
kind: LanguageCluster
metadata:
  name: my-cluster
spec:
  domain: demo.langop.io
  ingress:
    tls:
      enabled: true
      issuerRef:
        name: letsencrypt-production
        kind: ClusterIssuer
EOF

kubectl wait languagecluster/my-cluster \
  --for=condition=Ready --timeout=60s

kubectl config set-context --current --namespace=my-cluster
```

### 2. Configure an LLM

`LanguageModel` makes a model available to agents inside the cluster.

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

```bash
kubectl apply -f - <<EOF
apiVersion: langop.io/v1alpha1
kind: LanguageAgent
metadata:
  name: openclaw
spec:
  runtime: openclaw
  models:
    - name: claude-sonnet
EOF
```

The runtime auto-generates a gateway token and stores it in a secret.

**Connect:**

```bash
TOKEN=$(kubectl get secret openclaw-runtime -o jsonpath='{.data.OPENCLAW_GATEWAY_TOKEN}' | base64 -d)
```

Open [https://openclaw.demo.langop.io](https://openclaw.demo.langop.io) in your browser and enter `$TOKEN` when prompted.

You can also connect the openclaw CLI client (see [github.com/openclaw/openclaw](https://github.com/openclaw/openclaw)) directly to `wss://openclaw.demo.langop.io`.

Alternatively, port-forward for local access:

```bash
kubectl port-forward svc/openclaw 18789:18789
# then open http://localhost:18789 or connect to ws://localhost:18789
```

</details>

<details>
<summary><strong>opencode</strong></summary>

```bash
kubectl apply -f - <<EOF
apiVersion: langop.io/v1alpha1
kind: LanguageAgent
metadata:
  name: opencode
spec:
  runtime: opencode
  models:
    - name: claude-sonnet
EOF
```

The runtime auto-generates a login password and stores it in a secret.

**Connect:**

```bash
USERNAME=$(kubectl get secret opencode-runtime -o jsonpath='{.data.OPENCODE_SERVER_USERNAME}' | base64 -d)
PASSWORD=$(kubectl get secret opencode-runtime -o jsonpath='{.data.OPENCODE_SERVER_PASSWORD}' | base64 -d)
echo "username: $USERNAME  password: $PASSWORD"
```

Open [https://opencode.demo.langop.io](https://opencode.demo.langop.io) in your browser and sign in with `$USERNAME` / `$PASSWORD`.

To attach the TUI (opencode v1.0.10+):

```bash
opencode attach https://opencode.demo.langop.io --username "$USERNAME" --password "$PASSWORD"
```

Alternatively, port-forward for local access:

```bash
kubectl port-forward svc/opencode 3000:3000
# then open http://localhost:3000 or: opencode attach http://localhost:3000 --username "$USERNAME" --password "$PASSWORD"
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
