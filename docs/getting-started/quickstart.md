# Quick Start

Deploy an openclaw agent in under 5 minutes.

## Prerequisites

- Language Operator [installed](installation.md)
- Anthropic API key (get one at [console.anthropic.com](https://console.anthropic.com))

## Step 1: Create a Cluster

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

kubectl config set-context --current --namespace=language-operator-openclaw
```

## Step 2: Configure an LLM

Create a secret with your Anthropic API key, then create a `LanguageModel`:

```bash
kubectl create secret generic anthropic-credentials \
  --from-literal=api-key=sk-ant-your-key-here

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

The operator creates a LiteLLM proxy so all agents can reach the model through a single in-cluster endpoint.

## Step 3: Deploy the Agent

The `openclaw` runtime is bundled with the operator. Reference it with `runtime: openclaw` and supply a gateway token inline — the operator creates the credential secret automatically:

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

## Step 4: Check Status

```bash
kubectl get languageagents
kubectl get pods -w
```

## Step 5: Access the Agent

```bash
kubectl port-forward svc/openclaw 18789:18789
```

openclaw uses a WebSocket gateway on port 18789 — it is not a browser-based HTTP UI.
Connect using the [openclaw browser extension](https://openclaw.ai/docs/extension) or CLI client, pointing it to `ws://localhost:18789`.

!!! success "You're Running!"
    You now have openclaw running on Kubernetes with AI capabilities provided through the Language Operator.

## What Just Happened?

The operator automatically:

1. Created a dedicated namespace (`language-operator-openclaw`)
2. Deployed a LiteLLM proxy with your Anthropic credentials
3. Injected the proxy URL into your agent via `MODEL_ENDPOINTS`
4. Created a `{agent}-runtime` Secret from `spec.openclaw.token` and injected it via `envFrom`
5. Created a Deployment, Service, and NetworkPolicy for openclaw

## Next Steps

- [Examples](examples.md) - More deployment patterns
- [CRD Reference](../api/overview.md) - Complete API documentation
- [Architecture](../architecture/overview.md) - How the operator works
