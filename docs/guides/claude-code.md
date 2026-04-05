# Deploying Claude Code

Claude Code is Anthropic's agentic coding tool. The `claude-code` runtime is bundled with Language Operator and installed automatically with the Helm chart. It exposes the [A2A (Agent-to-Agent) protocol](https://google.github.io/A2A/), so any A2A-compatible orchestrator or agent can call it directly.

## Prerequisites

- Language Operator [installed](../getting-started/installation.md)
- An Anthropic API key ([get one here](https://console.anthropic.com/settings/keys))

## Instructions

### Get an API Key

1. Go to [console.anthropic.com/settings/keys](https://console.anthropic.com/settings/keys)
2. Click **Create Key**, give it a name, and copy the key (`sk-ant-...`)

Claude Code makes direct API calls to Anthropic, so you need an API key with access to the model you want to use (e.g. `claude-sonnet-4-5`).

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

### Store Your API Key

```bash
kubectl create secret generic anthropic-credentials \
  --from-literal=ANTHROPIC_API_KEY=sk-ant-your-key-here
```

### Deploy Claude Code

```bash
kubectl apply -f - <<EOF
apiVersion: langop.io/v1alpha1
kind: LanguageAgent
metadata:
  name: code-agent
spec:
  runtime: claude-code
  instructions: |
    You are an expert software engineer. Help with code review,
    debugging, and implementation tasks.
  claudeCode:
    apiKeyRef:
      name: anthropic-credentials
EOF
```

The `apiKeyRef` tells the operator to inject `ANTHROPIC_API_KEY` from your secret directly into the pod — traffic goes straight to Anthropic, bypassing the shared gateway.

### Verify

```bash
kubectl get languageagents
kubectl get pods -w
```

Wait for the pod to reach `Running` and the LanguageAgent to show `Ready=True`.

### Interact via A2A

The agent exposes the A2A protocol at its service endpoint. Submit a task:

```bash
# Get the in-cluster service address
SVC=$(kubectl get svc code-agent -o jsonpath='{.spec.clusterIP}')

# Check the agent card
curl http://$SVC:8080/.well-known/agent-card | jq .

# Submit a task
curl -X POST http://$SVC:8080/ \
  -H 'Content-Type: application/json' \
  -d '{
    "jsonrpc": "2.0",
    "method": "tasks/send",
    "params": {
      "message": {
        "role": "user",
        "parts": [{"type": "text", "text": "List the files in /workspace"}]
      }
    }
  }'
```

Streaming task responses use Server-Sent Events (`tasks/sendSubscribe`). Use `tasks/get` to poll task status by ID.

## Using the Shared Gateway Instead

If you'd rather route Claude Code traffic through the shared LiteLLM gateway (e.g. to centralise billing or use a different provider), configure a `LanguageModel` instead of an `apiKeyRef`:

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
---
apiVersion: langop.io/v1alpha1
kind: LanguageAgent
metadata:
  name: code-agent
spec:
  runtime: claude-code
  instructions: |
    You are an expert software engineer.
  models:
    - name: claude-sonnet
EOF
```

The adapter will set `ANTHROPIC_BASE_URL` to the shared gateway and the operator injects a placeholder key. No `claudeCode.apiKeyRef` is needed.

## Configuration Reference

| Field | Description |
|-------|-------------|
| `spec.claudeCode.apiKeyRef.name` | Secret containing `ANTHROPIC_API_KEY`. Recommended for direct Anthropic access. |
| `spec.claudeCode.apiKey` | Inline API key (stored in a managed Secret). Use `apiKeyRef` instead if the key is already in a Secret. |
| `spec.claudeCode.maxTurns` | Max agentic turns per request. Sets `CLAUDE_CODE_MAX_TURNS`. |
| `spec.claudeCode.enabled` | Set automatically by the `claude-code` runtime. No need to set this manually. |

## What the Operator Created

| Resource | Name | Purpose |
|----------|------|---------|
| Deployment | `code-agent` | Runs the Claude Code server container |
| Service | `code-agent` | ClusterIP on port 8080 |
| NetworkPolicy | `code-agent` | Allows inbound from other agents in this namespace |
| PVC | `code-agent-workspace` | 10Gi persistent workspace; task store at `/workspace/.a2a/` |
| ConfigMap | `code-agent-agent` | Injected at `/etc/agent/config.yaml` |
