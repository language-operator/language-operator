# Deploying Claude Code

Claude Code is Anthropic's agentic coding tool. The `claude-code` runtime is bundled with Language Operator and installed automatically with the Helm chart. It exposes Claude Code's interactive CLI as a WebSocket terminal (xterm.js in the browser, node-pty on the server) so you can connect directly from any browser.

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

To copy text, drag to select (hold **Shift** while dragging if Claude is in a mouse-tracking prompt), then **Ctrl+C** (smart copy: copies when there is a selection, otherwise passes through as `SIGINT`). **Ctrl+Insert** also works. On Mac: **Cmd+C**. Paste with **Ctrl+Shift+V**, **Shift+Insert**, or **Cmd+V** on Mac.

The terminal session persists across page reloads and is shared across multiple browser tabs to the same agent — close your laptop, reopen, and you're back where you left off. The session is cleared when Claude exits (e.g. `/exit`) or when the agent pod restarts.

### First-time Login

Inside the terminal, run `/login` and complete the Claude.ai browser flow. The credentials are saved to `/workspace/.claude/.credentials.json` and survive pod restarts.

## Configuration Reference

The `claude-code` runtime is configured through plain environment variables on the agent's `spec.deployment.env`. For example, to cap the number of agentic turns per request, set `CLAUDE_CODE_MAX_TURNS`:

```yaml
spec:
  runtime: claude-code
  deployment:
    env:
      - name: CLAUDE_CODE_MAX_TURNS
        value: "10"
```

## What the Operator Created

| Resource | Name | Purpose |
|----------|------|---------|
| Deployment | `code-agent` | Runs the Claude Code WebSocket terminal container |
| Service | `code-agent` | ClusterIP on port 8080 (WebSocket terminal) |
| NetworkPolicy | `code-agent` | Allows inbound from other agents in this namespace. Add `spec.networkPolicies.egress` with `cidr: 0.0.0.0/0` on port 443 to reach `claude.ai`, `api.anthropic.com`, and other public APIs. |
| PVC | `code-agent-workspace` | 10Gi persistent workspace at `/workspace` (also holds Claude config) |
| ConfigMap | `code-agent-agent` | Injected at `/etc/agent/config.yaml` |
