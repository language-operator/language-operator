# Deploying OpenCode

OpenCode is an AI coding assistant with a browser-based HTTP UI. The `opencode` runtime is bundled with Language Operator and installed automatically with the Helm chart.

## Prerequisites

- Language Operator [installed](../getting-started/installation.md)
- An LLM provider API key, or a local model endpoint (e.g. Ollama)
- A StorageClass for the workspace PVC — see [cluster setup](cluster-setup.md#storageclass)

## Step 1: Create a LanguageCluster

```bash
kubectl apply -f - <<EOF
apiVersion: langop.io/v1alpha1
kind: LanguageCluster
metadata:
  name: opencode
spec:
  domain: opencode.agents.example.com
EOF

kubectl wait languagecluster/opencode --for=condition=Ready --timeout=60s
kubectl config set-context --current --namespace=opencode
```

## Step 2: Configure a LanguageModel

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

## Step 3: Verify the Runtime

```bash
kubectl get languageagentruntimes
# NAME       AGE
# openclaw   5m
# opencode   5m
```

## Step 4: Deploy the Agent

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

## Step 5: Verify

```bash
kubectl get languageagents
kubectl get pods -w
```

Wait for the pod to reach `Running` and the LanguageAgent to show `Ready=True`.

## Step 6: Access the UI

OpenCode serves a browser UI on port **3000**. Retrieve the auto-generated credentials and forward the port:

```bash
USERNAME=$(kubectl get secret opencode-runtime \
  -o jsonpath='{.data.OPENCODE_SERVER_USERNAME}' | base64 -d)
PASSWORD=$(kubectl get secret opencode-runtime \
  -o jsonpath='{.data.OPENCODE_SERVER_PASSWORD}' | base64 -d)

echo "username: $USERNAME  password: $PASSWORD"

kubectl port-forward svc/opencode 3000:3000
```

Open [http://localhost:3000](http://localhost:3000) and sign in with the credentials above.

!!! tip "TUI access"
    To attach the OpenCode TUI (v1.0.10+) instead of the browser UI:
    ```bash
    opencode attach http://localhost:3000 --username "$USERNAME" --password "$PASSWORD"
    ```

!!! tip "External access"
    If your `LanguageCluster` has `spec.domain` set, the operator creates an Ingress at `opencode.<cluster-domain>` for external access.

## What the Operator Created

| Resource | Name | Purpose |
|---|---|---|
| Namespace | `opencode` | Isolated workload namespace |
| Deployment | `opencode` | Runs the OpenCode container |
| Service | `opencode` | ClusterIP on port 3000 |
| Secret | `opencode-runtime` | Auto-generated username and password |
| NetworkPolicy | `opencode` | Allows inbound from other agents in this namespace |
| PVC | `opencode-workspace` | 10Gi persistent workspace |
| ConfigMap | `opencode-agent` | Injected at `/etc/agent/config.yaml` |

## Troubleshooting

**Pod stuck in `Pending`:**
```bash
kubectl describe pod -l app=opencode
kubectl get pvc
```

**UI loads but no model available:**
```bash
kubectl get languagemodels
kubectl logs deployment/gateway
```

**UI errors after port-forward:**
```bash
kubectl logs deployment/opencode
```
