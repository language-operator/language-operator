# Deploying OpenCode

OpenCode is an AI coding assistant with a browser-based HTTP UI. The `opencode` runtime is bundled with Language Operator and installed automatically with the Helm chart.

## Prerequisites

- Language Operator [installed](../getting-started/installation.md)
- An LLM provider API key (Anthropic, OpenAI, or any OpenAI-compatible endpoint)
- A default StorageClass (for the workspace PVC — see [cluster setup](cluster-setup.md#storageclass))

## Step 1: Create a LanguageCluster

```bash
kubectl apply -f - <<EOF
apiVersion: langop.io/v1alpha1
kind: LanguageCluster
metadata:
  name: opencode
spec:
  domain: opencode.langop.io
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

## Step 3: Verify the Runtime

```bash
kubectl get languageagentruntimes
# NAME       AGE
# openclaw   5m
# opencode   5m
```

## Step 4: Deploy the Agent

```bash
kubectl apply -f - <<EOF
apiVersion: langop.io/v1alpha1
kind: LanguageAgent
metadata:
  name: opencode
spec:
  runtime: opencode
  models:
    - name: claude-sonnet   # or gpt-4o if you configured OpenAI
EOF
```

## Step 5: Verify

```bash
kubectl get languageagents
kubectl get pods -w
```

Wait for the pod to reach `Running` and the LanguageAgent to show `Ready=True`.

## Step 6: Access the UI

OpenCode serves a browser UI on port **3000**:

```bash
kubectl port-forward svc/opencode 3000:3000
```

Open [http://localhost:3000](http://localhost:3000) in your browser.

!!! tip "External access"
    If your cluster has a Gateway API controller, the operator creates an `HTTPRoute` at `opencode.<cluster-domain>`. Set `spec.domain` in the `LanguageCluster` to enable this.

## What the Operator Created

| Resource | Name | Purpose |
|---|---|---|
| Namespace | `opencode` | Isolated workload namespace |
| Deployment | `opencode` | Runs the OpenCode container |
| Service | `opencode` | ClusterIP on port 3000 |
| NetworkPolicy | `opencode` | Allows inbound from other agents in this namespace |
| PVC | `opencode-workspace` | 10Gi persistent workspace |
| ConfigMap | `opencode-agent` | Injected at `/etc/agent/config.yaml` |

## Troubleshooting

**Pod stuck in `Pending`:**
Check PVC binding — a default StorageClass is required:
```bash
kubectl describe pod -l app=opencode
kubectl get pvc
```

**UI loads but no model available:**
Verify the gateway and LanguageModel are ready:
```bash
kubectl get languagemodels
kubectl get pods
kubectl logs deployment/gateway
```

**Port-forward works but UI errors:**
Check the OpenCode container logs:
```bash
kubectl logs deployment/opencode
```

## Next Steps

- [Deploying Custom Agents](custom-agents.md)
- [Examples — multi-model and persona patterns](../getting-started/examples.md)
- [LanguageAgentRuntime reference](../api/languageagentruntime.md)
