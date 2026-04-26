# Deploying Claude Code

Claude Code is Anthropic's agentic coding tool. The `claude-code` runtime is bundled with Language Operator and installed automatically with the Helm chart. It exposes a WebSocket terminal via [ttyd](https://github.com/tsl0922/ttyd), so you can connect directly to Claude Code's interactive CLI from any browser or WebSocket client.

## Prerequisites

- Language Operator [installed](../getting-started/installation.md)
- An Anthropic API key ([get one here](https://console.anthropic.com/settings/keys))

## Instructions

### Get an API Key

1. Go to [console.anthropic.com/settings/keys](https://console.anthropic.com/settings/keys)
2. Click **Create Key**, give it a name, and copy the key (`sk-ant-...`)

Claude Code makes direct API calls to Anthropic, so you need an API key with access to the model you want to use (e.g. `claude-sonnet-4-6`).

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

### Connect to the Terminal

The agent exposes a WebSocket terminal on port 8080. Port-forward to access it locally:

```bash
kubectl port-forward svc/code-agent 8080:8080
```

Then open `http://localhost:8080` in your browser. You'll see Claude Code's interactive CLI running inside the pod.

## Using the Shared Gateway Instead

> **Not currently supported.** Claude Code 2.x sends `Authorization: Bearer` (OAuth format) when `ANTHROPIC_BASE_URL` is set. Anthropic's API rejects OAuth authentication, so routing through the LiteLLM gateway fails at the model-call level. Use `claudeCode.apiKeyRef` (or `spec.deployment.env`) to supply a real API key and let Claude Code call Anthropic directly.

## Configuration Reference

| Field | Description |
|-------|-------------|
| `spec.claudeCode.apiKeyRef.name` | Secret containing `ANTHROPIC_API_KEY`. Recommended for direct Anthropic access. |
| `spec.claudeCode.apiKey` | Inline API key (stored in a managed Secret). Use `apiKeyRef` instead if the key is already in a Secret. |
| `spec.claudeCode.enabled` | Set automatically by the `claude-code` runtime. No need to set this manually. |

## What the Operator Created

| Resource | Name | Purpose |
|----------|------|---------|
| Deployment | `code-agent` | Runs the Claude Code ttyd terminal container |
| Service | `code-agent` | ClusterIP on port 8080 (ttyd WebSocket terminal) |
| NetworkPolicy | `code-agent` | Allows inbound from other agents in this namespace. Add `spec.networkPolicies.egress` with `cidr: 0.0.0.0/0` on port 443 to reach `api.anthropic.com` and other public APIs. |
| PVC | `code-agent-workspace` | 10Gi persistent workspace at `/workspace` |
| ConfigMap | `code-agent-agent` | Injected at `/etc/agent/config.yaml` |
