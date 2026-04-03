# Deploying OpenCode

OpenCode is an AI coding assistant with a browser-based HTTP UI. The `opencode` runtime is bundled with Language Operator and installed automatically with the Helm chart.

## Prerequisites

- Language Operator [installed](../getting-started/installation.md)
- An LLM provider API key, or a local model endpoint (e.g. Ollama)
- A StorageClass for the workspace PVC — see [cluster setup](cluster-setup.md#storageclass)

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

### Deploy OpenCode

=== "Anthropic"

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

=== "OpenAI"

    ```bash
    kubectl apply -f - <<EOF
    apiVersion: langop.io/v1alpha1
    kind: LanguageAgent
    metadata:
      name: opencode
    spec:
      runtime: opencode
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
      name: opencode
    spec:
      runtime: opencode
      models:
        - name: llama3
    EOF
    ```

### Verify

```bash
kubectl get languageagents
kubectl get pods -w
```

Wait for the pod to reach `Running` and the LanguageAgent to show `Ready=True`.

### Get Credentials

Retrieve the auto-generated credentials:

```bash
USERNAME=$(kubectl get secret opencode-runtime \
  -o jsonpath='{.data.OPENCODE_SERVER_USERNAME}' | base64 -d)
PASSWORD=$(kubectl get secret opencode-runtime \
  -o jsonpath='{.data.OPENCODE_SERVER_PASSWORD}' | base64 -d)

echo "username: $USERNAME  password: $PASSWORD"
```

### Connect

Log in at https://opencode.demo-cluster.\<your-domain\> with the credentials above, or attach the TUI:

```bash
opencode attach https://opencode.demo-cluster.<your-domain> \
  --username "$USERNAME" --password "$PASSWORD"
```

## What the Operator Created

| Resource | Name | Purpose |
|---|---|---|
| Namespace | `demo-cluster` | Isolated workload namespace |
| Deployment | `opencode` | Runs the OpenCode container |
| Service | `opencode` | ClusterIP on port 3000 |
| Secret | `opencode-runtime` | Auto-generated username and password |
| NetworkPolicy | `opencode` | Allows inbound from other agents in this namespace |
| PVC | `opencode-workspace` | 10Gi persistent workspace |
| ConfigMap | `opencode-agent` | Injected at `/etc/agent/config.yaml` |
