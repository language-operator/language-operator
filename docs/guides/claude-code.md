# Deploying Claude Code

Claude Code is Anthropic's agentic coding tool. The `claude-code` runtime is bundled with Language Operator and installed automatically with the Helm chart. It exposes a WebSocket terminal via [ttyd](https://github.com/tsl0922/ttyd), so you can connect directly to Claude Code's interactive CLI from any browser or WebSocket client.

Authentication is interactive. After deploying, open the agent terminal and run `/login` inside Claude Code. Credentials are written to `/workspace/.claude/.credentials.json` and persist on the workspace PVC, so subsequent pod restarts don't re-prompt.

## Prerequisites

- Language Operator [installed](../getting-started/installation.md)
- A Claude account (Pro, Max, Team, or Enterprise)

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
EOF
```

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

### First-time Login

Inside the terminal, run `/login` and complete the Claude.ai browser flow. The credentials are saved to `/workspace/.claude/.credentials.json` and survive pod restarts.

## Configuration Reference

| Field | Description |
|-------|-------------|
| `spec.claudeCode.maxTurns` | Cap on the number of agentic turns per request (sets `CLAUDE_CODE_MAX_TURNS`). |

## What the Operator Created

| Resource | Name | Purpose |
|----------|------|---------|
| Deployment | `code-agent` | Runs the Claude Code ttyd terminal container |
| Service | `code-agent` | ClusterIP on port 8080 (ttyd WebSocket terminal) |
| NetworkPolicy | `code-agent` | Allows inbound from other agents in this namespace. Add `spec.networkPolicies.egress` with `cidr: 0.0.0.0/0` on port 443 to reach `claude.ai`, `api.anthropic.com`, and other public APIs. |
| PVC | `code-agent-workspace` | 10Gi persistent workspace at `/workspace` (also holds Claude config) |
| ConfigMap | `code-agent-agent` | Injected at `/etc/agent/config.yaml` |
