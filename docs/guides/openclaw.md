# Deploying OpenClaw

OpenClaw is an AI coding assistant that connects to your editor via a WebSocket gateway. The `openclaw` runtime is bundled with Language Operator and is installed automatically with the Helm chart.

## Prerequisites

- Language Operator [installed](../getting-started/installation.md)
- An LLM provider API key, or a local model endpoint (e.g. Ollama)
- A StorageClass for the workspace PVC — see [cluster setup](cluster-setup.md#storageclass)
- OpenClaw browser extension or CLI client

## Step 1: Create a LanguageCluster

```bash
kubectl apply -f - <<EOF
apiVersion: langop.io/v1alpha1
kind: LanguageCluster
metadata:
  name: openclaw
spec:
  domain: openclaw.agents.example.com
EOF

kubectl wait languagecluster/openclaw --for=condition=Ready --timeout=60s
kubectl config set-context --current --namespace=openclaw
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
      name: openclaw
    spec:
      runtime: openclaw
      openclaw: {}
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
      openclaw: {}
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
      openclaw: {}
      models:
        - name: llama3
    EOF
    ```

`spec.openclaw: {}` tells the operator to auto-generate a gateway token, written to a Secret named `openclaw-runtime` after creation.

## Step 5: Verify

```bash
kubectl get languageagents
kubectl get pods -w
```

Wait for the pod to reach `Running` and the LanguageAgent to show `Ready=True`.

## Step 6: Connect

Retrieve the auto-generated token and forward the port:

```bash
TOKEN=$(kubectl get secret openclaw-runtime \
  -o jsonpath='{.data.OPENCLAW_GATEWAY_TOKEN}' | base64 -d)

echo "Token: $TOKEN"

kubectl port-forward svc/openclaw 18789:18789
```

OpenClaw listens on a **WebSocket** at port 18789. Connect using the [OpenClaw browser extension](https://github.com/openclaw/openclaw) or CLI client, pointing it to `ws://localhost:18789` with the token above.

!!! tip "External access"
    If your `LanguageCluster` has `spec.domain` set, the operator creates an Ingress at `openclaw.<cluster-domain>` for external access.

## What the Operator Created

| Resource | Name | Purpose |
|---|---|---|
| Namespace | `openclaw` | Isolated workload namespace |
| Deployment | `openclaw` | Runs the OpenClaw container |
| Service | `openclaw` | ClusterIP on port 18789 |
| Secret | `openclaw-runtime` | Auto-generated gateway token |
| NetworkPolicy | `openclaw` | Allows inbound from other agents in this namespace |
| PVC | `openclaw-workspace` | 10Gi persistent workspace |
| ConfigMap | `openclaw-agent` | Injected at `/etc/agent/config.yaml` |

## Troubleshooting

**Pod stuck in `Pending`:**
```bash
kubectl describe pod -l app=openclaw
kubectl get pvc
```

**Pod `CrashLoopBackOff`:**
```bash
kubectl get pods
kubectl logs deployment/gateway
```

**Token not found:**
The secret is created after the first successful reconcile. Check events if the agent is not yet `Ready`:
```bash
kubectl get events --sort-by='.lastTimestamp'
```
