# Deploying OpenClaw

OpenClaw is an AI coding assistant that connects to your editor via a WebSocket gateway. The `openclaw` runtime is bundled with Language Operator and is installed automatically with the Helm chart.

## Prerequisites

- Language Operator [installed](../getting-started/installation.md)
- Anthropic API key (get one at [console.anthropic.com](https://console.anthropic.com))
- A default StorageClass (for the workspace PVC — see [cluster setup](cluster-setup.md#storageclass))
- OpenClaw browser extension or CLI client

## Step 1: Create a LanguageCluster

A `LanguageCluster` is a managed namespace. All agents, models, and tools inside it share a LiteLLM gateway.

```bash
kubectl apply -f - <<EOF
apiVersion: langop.io/v1alpha1
kind: LanguageCluster
metadata:
  name: openclaw
spec:
  domain: openclaw.agents.my-company.com
EOF

kubectl wait languagecluster/openclaw --for=condition=Ready --timeout=60s
kubectl config set-context --current --namespace=openclaw
```

## Step 2: Configure a LanguageModel

Create a secret with your Anthropic API key, then declare a model:

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

The operator deploys a LiteLLM proxy in the cluster namespace so the agent never holds raw API credentials.

## Step 3: Verify the Runtime

The `openclaw` `LanguageAgentRuntime` is installed by the Helm chart. Confirm it is present:

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
  name: openclaw
spec:
  runtime: openclaw
  openclaw: {}
  models:
    - name: claude-sonnet
EOF
```

`spec.openclaw: {}` tells the operator to auto-generate a gateway token. The token is written to a Secret named `<agent>-runtime` after creation.

## Step 5: Verify

```bash
kubectl get languageagents
kubectl get pods -w
```

Wait for the pod to reach `Running` and the LanguageAgent to show `Ready=True`.

## Step 6: Connect

Retrieve the auto-generated gateway token and forward the port:

```bash
TOKEN=$(kubectl get secret openclaw-runtime \
  -o jsonpath='{.data.OPENCLAW_GATEWAY_TOKEN}' | base64 -d)

echo "Token: $TOKEN"

kubectl port-forward svc/openclaw 18789:18789
```

OpenClaw listens on a **WebSocket** at port 18789 — it is not a browser HTTP UI. Connect using the [OpenClaw browser extension](https://github.com/openclaw/openclaw) or CLI client, pointing it to `ws://localhost:18789` with the token above.

!!! tip "External access"
    If your cluster has a Gateway API controller, the operator creates an `HTTPRoute` at `openclaw.<cluster-domain>` for external access. Set `spec.domain` in the `LanguageCluster` to enable this.

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
Check PVC binding — a default StorageClass is required for the workspace PVC:
```bash
kubectl describe pod -l app=openclaw
kubectl get pvc
```

**Pod `CrashLoopBackOff`:**
The agent couldn't reach the LiteLLM gateway. Verify the model and gateway pod are running:
```bash
kubectl get pods
kubectl logs deployment/gateway
```

**Token not found:**
The secret is created after the first successful reconcile. If the agent is not `Ready`, the secret may not exist yet:
```bash
kubectl get events --sort-by='.lastTimestamp'
```

## Next Steps

- [Deploying OpenCode](opencode.md)
- [Deploying Custom Agents](custom-agents.md)
- [LanguageAgentRuntime reference](../api/languageagentruntime.md)
