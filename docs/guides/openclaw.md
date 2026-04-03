# Deploying OpenClaw

OpenClaw is an AI coding assistant that connects to your editor via a WebSocket gateway. The `openclaw` runtime is bundled with Language Operator and is installed automatically with the Helm chart.

## Prerequisites

- Language Operator [installed](../getting-started/installation.md)
- An LLM provider API key, or a local model endpoint (e.g. Ollama)
- A StorageClass for the workspace PVC — see [cluster setup](cluster-setup.md#storageclass)
- OpenClaw browser extension or CLI client

## Instructions

### Create a Cluster

```bash
kubectl apply -f - <<EOF
apiVersion: langop.io/v1alpha1
kind: LanguageCluster
metadata:
  name: demo-cluster
spec:
  domain: demo-cluster.<your-domain>
EOF

kubectl wait languagecluster/demo-cluster --for=condition=Ready --timeout=60s
kubectl config set-context --current --namespace=demo-cluster
```

### Configure a Model

=== "Anthropic"

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

=== "OpenAI"

    ```bash
    kubectl create secret generic openai-credentials \
      --from-literal=api-key=sk-your-key-here

    kubectl apply -f - <<EOF
    apiVersion: langop.io/v1alpha1
    kind: LanguageModel
    metadata:
      name: gpt-4o
    spec:
      provider: openai
      modelName: gpt-4o
      apiKeySecretRef:
        name: openai-credentials
        key: api-key
    EOF
    ```

=== "Local Model"

    Assumes Ollama is running in your cluster. No API key required.

    ```bash
    kubectl apply -f - <<EOF
    apiVersion: langop.io/v1alpha1
    kind: LanguageModel
    metadata:
      name: llama3
    spec:
      provider: openai-compatible
      modelName: llama3.2
      endpoint: http://ollama.default.svc.cluster.local:11434/v1
    EOF
    ```

### Deploy OpenClaw

=== "Anthropic"

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

=== "OpenAI"

    ```bash
    kubectl apply -f - <<EOF
    apiVersion: langop.io/v1alpha1
    kind: LanguageAgent
    metadata:
      name: openclaw
    spec:
      runtime: openclaw
      models:
        - name: gpt-4o
    EOF
    ```

=== "Local Model"

    ```bash
    kubectl apply -f - <<EOF
    apiVersion: langop.io/v1alpha1
    kind: LanguageAgent
    metadata:
      name: openclaw
    spec:
      runtime: openclaw
      models:
        - name: llama3
    EOF
    ```

`spec.openclaw: {}` tells the operator to auto-generate a gateway token, written to a Secret named `openclaw-runtime` after creation.

### Verify

```bash
kubectl get languageagents
kubectl get pods -w
```

Wait for the pod to reach `Running` and the LanguageAgent to show `Ready=True`.

### Get Credentials

Retrieve the auto-generated token:

```bash
TOKEN=$(kubectl get secret openclaw-runtime \
  -o jsonpath='{.data.OPENCLAW_GATEWAY_TOKEN}' | base64 -d)

echo "Token: $TOKEN"
```

### Connect

Log in with your token at https://openclaw.demo-cluster.<your-domain>.


## What the Operator Created

| Resource | Name | Purpose |
|---|---|---|
| Namespace | `my-cluster` | Isolated workload namespace |
| Deployment | `openclaw` | Runs the OpenClaw container |
| Service | `openclaw` | ClusterIP on port 18789 |
| Secret | `openclaw-runtime` | Auto-generated gateway token |
| NetworkPolicy | `openclaw` | Allows inbound from other agents in this namespace |
| PVC | `openclaw-workspace` | 10Gi persistent workspace |
| ConfigMap | `openclaw-agent` | Injected at `/etc/agent/config.yaml` |