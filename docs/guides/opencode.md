# Deploying OpenCode

OpenCode is an AI coding assistant similar to Claude Code. The `opencode` runtime is bundled with Language Operator and installed automatically with the Helm chart.

## Prerequisites

- Language Operator [installed](../getting-started/installation.md)
- An LLM provider API key, or a local model endpoint (e.g. Ollama)

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

### Connect

Access is gated by the cluster's OIDC proxy — there is no separate opencode password.

If your `LanguageCluster` has a domain and [auth enabled](../components/clusters.md#authentication),
open https://opencode.demo-cluster.\<your-domain\> and sign in through the cluster's OIDC provider.

For local access, port-forward the service (this bypasses the proxy, so no login is required):

```bash
kubectl port-forward svc/opencode 3000:3000
# open http://localhost:3000, or attach the TUI:
opencode attach http://localhost:3000
```

!!! warning

    opencode has no built-in authentication. When the cluster does **not** enable auth, opencode is
    exposed unauthenticated on its ingress. Enable cluster auth to protect it.

## What the Operator Created

| Resource | Name | Purpose |
|---|---|---|
| Namespace | `demo-cluster` | Isolated workload namespace |
| Deployment | `opencode` | Runs the OpenCode container |
| Service | `opencode` | ClusterIP on port 3000 |
| NetworkPolicy | `opencode` | Allows inbound from other agents in this namespace |
| PVC | `opencode-workspace` | 10Gi persistent workspace |
| ConfigMap | `opencode-agent` | Injected at `/etc/agent/config.yaml` |
