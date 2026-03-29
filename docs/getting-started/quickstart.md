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
```

Verify the cluster is ready:

```bash
kubectl get languagecluster language-operator-openclaw
```

## Step 2: Configure an LLM

Create a secret with your Anthropic API key:

```bash
kubectl create secret generic anthropic-credentials \
  -n language-operator-openclaw \
  --from-literal=api-key=sk-ant-your-key-here
```

Create a `LanguageModel` resource:

```bash
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

The operator will create a LiteLLM proxy that agents can use to access the model.

## Step 3: Create Gateway Token

Generate a secure token for openclaw's web gateway:

```bash
kubectl create secret generic openclaw-gateway \
  -n language-operator-openclaw \
  --from-literal=OPENCLAW_GATEWAY_TOKEN=$(openssl rand -hex 32)
```

## Step 4: Deploy the Agent

Deploy openclaw as a `LanguageAgent`:

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

## Step 5: Check Status

Watch the agent pod come up:

```bash
kubectl get pods -n language-operator-openclaw -w
```

Check the agent status:

```bash
kubectl get languageagent openclaw -n language-operator-openclaw
```

View agent logs:

```bash
kubectl logs -n language-operator-openclaw -l langop.io/agent=openclaw
```

## Step 6: Access the Agent

Forward the openclaw web interface:

```bash
kubectl port-forward -n language-operator-openclaw \
  svc/openclaw 18789:8080
```

Open [http://localhost:18789](http://localhost:18789) in your browser.

!!! success "You're Running!"
    You now have openclaw running on Kubernetes with AI capabilities provided through the Language Operator.

## What Just Happened?

The operator automatically:

1. Created a dedicated namespace (`language-operator-openclaw`)
2. Deployed a LiteLLM proxy with your Anthropic credentials
3. Injected the proxy URL into your agent via `MODEL_ENDPOINTS`
4. Created a Deployment, Service, and HTTPRoute for openclaw
5. Set up NetworkPolicy for secure communication

## Next Steps

- [Examples](examples.md) - More deployment patterns
- [CRD Reference](../api/overview.md) - Complete API documentation
- [Architecture](../architecture/overview.md) - How the operator works
