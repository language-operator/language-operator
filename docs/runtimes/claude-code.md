# Deploying Claude Code

Claude Code is Anthropic's agentic coding tool. The `claude-code` runtime is installed by the `language-operator-runtimes` chart. It exposes Claude Code's interactive CLI as a WebSocket terminal (xterm.js in the browser, node-pty on the server) so you can connect directly from any browser.

Claude Code can authenticate interactively (run `/login` in the terminal) or headlessly via an API key or subscription token injected from a Secret — see [Authentication](#authentication) below.

## Prerequisites

- Language Operator [installed](../getting-started/installation.md), including the `language-operator-runtimes` chart (provides the `claude-code` runtime)
- A [`LanguageCluster`](../components/clusters.md) to deploy into, with your `kubectl` context set to its namespace (examples below assume a cluster named `demo-cluster`)
- A Claude account (Pro, Max, Team, or Enterprise)

## Instructions

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
kubectl get lagent -w
```

Wait for `PHASE` to reach `Running`. The `MODE` column shows `service`: this runtime is a
long-lived, addressable agent, which is why it has a Service you can port-forward to. See
[Execution Modes](../guides/execution-modes.md).

### Connect to the Terminal

The agent exposes a WebSocket terminal on port 8080. Port-forward to access it locally:

```bash
kubectl port-forward svc/code-agent 8080:8080
```

Then open `http://localhost:8080` in your browser. You'll see Claude Code's interactive CLI running inside the pod.

To copy text, drag to select (hold **Shift** while dragging if Claude is in a mouse-tracking prompt), then **Ctrl+C** (smart copy: copies when there is a selection, otherwise passes through as `SIGINT`). **Ctrl+Insert** also works. On Mac: **Cmd+C**. Paste with **Ctrl+Shift+V**, **Shift+Insert**, or **Cmd+V** on Mac.

The terminal session persists across page reloads and pod restarts, and is shared across multiple browser tabs to the same agent — close your laptop, reopen, and you're back where you left off. The session is cleared only when Claude exits (e.g. `/exit`).

## Authentication

Claude Code needs credentials to reach Anthropic's API. Pick **one** of the methods below. The non-interactive options inject a Secret value as an environment variable on `spec.deployment.env`, so agents come up authenticated with no per-pod setup — the right choice for headless or multi-agent deployments.

### Interactive login (default)

If you set no credential env vars, open the agent terminal (see [Connect to the Terminal](#connect-to-the-terminal)) and run `/login`, completing the Claude.ai browser flow. Credentials are written to `/workspace/.claude/.credentials.json` and persist on the workspace PVC, so subsequent pod restarts don't re-prompt.

### API key (headless)

Use `ANTHROPIC_API_KEY` for API-key billing — no `/login` needed:

```bash
kubectl create secret generic anthropic-credentials --from-literal=api-key=sk-ant-...
```

```yaml
spec:
  runtime: claude-code
  deployment:
    env:
      - name: ANTHROPIC_API_KEY
        valueFrom:
          secretKeyRef:
            name: anthropic-credentials
            key: api-key
```

### Subscription token (headless)

For subscription billing without a browser, mint a long-lived token with `claude setup-token` on your own machine, then inject it as `CLAUDE_CODE_OAUTH_TOKEN`:

```bash
claude setup-token   # prints a long-lived token
kubectl create secret generic claude-code-oauth --from-literal=token=<token>
```

```yaml
spec:
  runtime: claude-code
  deployment:
    env:
      - name: CLAUDE_CODE_OAUTH_TOKEN
        valueFrom:
          secretKeyRef:
            name: claude-code-oauth
            key: token
```

See [examples/development-team](https://github.com/language-operator/language-operator/tree/main/examples/development-team) for a multi-agent setup that wires these credentials in (with `optional: true` so the same manifest works whether or not you provide them).

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
| WorkflowTemplate | `code-agent` | The agent's pod spec; also what `argo submit --from` targets |
| Workflow | `code-agent` | The long-lived run. Runs the Claude Code WebSocket terminal container |
| Service | `code-agent` | ClusterIP on port 8080 (WebSocket terminal) |
| NetworkPolicy | `code-agent` | Allows inbound from other agents in this namespace. Add `spec.networkPolicies.egress` with `cidr: 0.0.0.0/0` on port 443 to reach `claude.ai`, `api.anthropic.com`, and other public APIs. |
| PVC | `code-agent-workspace` | 10Gi persistent workspace at `/workspace` (also holds Claude config) |
| ConfigMap | `code-agent-agent` | Injected at `/etc/agent/config.yaml` |
