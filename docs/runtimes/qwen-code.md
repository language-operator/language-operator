# Deploying Qwen Code

[Qwen Code](https://github.com/QwenLM/qwen-code) is an open-source AI coding agent for the terminal. The `qwen-code` runtime runs its TUI inside tmux behind a browser terminal, and is installed by the `language-operator-runtimes` chart. No Qwen account or login is needed: every model call goes through the cluster's [model gateway](../components/models.md) over the OpenAI chat-completions API, so any [LanguageModel](../api/languagemodel.md) works, not only Qwen models.

The runtime image is built in [qwen-code-adapter](https://github.com/language-operator/qwen-code-adapter) on the shared `coding-runtime` base, like the [Claude Code](claude-code.md), [OpenCode](opencode.md) and [Kilo](kilo.md) runtimes.

## Prerequisites

- Language Operator [installed](../getting-started/installation.md), including the `language-operator-runtimes` chart (provides the `qwen-code` runtime)
- A [`LanguageCluster`](../components/clusters.md) to deploy into, with your `kubectl` context set to its namespace (examples below assume a cluster named `demo-cluster`)
- An LLM provider API key, or a local model endpoint (e.g. Ollama)

## Instructions

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

### Deploy Qwen Code

```bash
kubectl apply -f - <<EOF
apiVersion: langop.io/v1alpha1
kind: LanguageAgent
metadata:
  name: qwen-code
spec:
  runtime: qwen-code
  models:
    - name: claude-sonnet   # or gpt-4o, llama3
EOF
```

On start, the runtime writes Qwen Code's `settings.json` from the operator's `/etc/agent/config.yaml`: the gateway as an OpenAI-compatible provider, the first referenced model, and every [`LanguageTool`](../components/tools.md) the agent references as an MCP server. Header tokens for those tools are written as `${NAME}` references to the container's environment, so they never land on disk. The persona and `spec.instructions`, when set, are loaded as standing context (`QWEN.md`) for every session.

### Verify

```bash
kubectl get languageagents
kubectl get lagent -w
```

Wait for `PHASE` to reach `Running`. The `MODE` column shows `service`: the TUI is long-lived and addressable. See
[Execution Modes](../guides/execution-modes.md).

### Connect

Access is gated by the cluster's OIDC proxy — there is no separate Qwen Code password.

If your `LanguageCluster` has a domain and [auth enabled](../components/clusters.md#authentication),
open https://qwen-code.demo-cluster.\<your-domain\> and sign in through the cluster's OIDC provider.

For local access, port-forward the service (this bypasses the proxy, so no login is required):

```bash
kubectl port-forward svc/qwen-code 8080:8080
```

Then open `http://localhost:8080` in your browser. The session lives in tmux, so it survives page reloads, and after a pod restart Qwen Code resumes the previous conversation from the workspace volume.

In the terminal, Qwen Code asks before it edits a file or runs a shell command; **Shift+Tab** cycles the approval mode. Qwen's own default, where a model decides each approval, is not used, because every decision would be an extra model call through the gateway.

!!! warning

    The terminal has no built-in authentication. When the cluster does **not** enable auth, it is
    exposed unauthenticated on its ingress. Enable cluster auth to protect it.

## Task Mode

Qwen Code also runs headless as a [task](../guides/execution-modes.md): each run does the work in `spec.instructions` and exits.

```yaml
apiVersion: langop.io/v1alpha1
kind: LanguageAgent
metadata:
  name: changelog
spec:
  runtime: qwen-code
  models:
    - name: claude-sonnet
  repository:
    url: https://github.com/acme/api.git
  instructions: |
    Summarise the commits merged since the last tag into CHANGELOG.md.
  execution:
    mode: task
    schedule: "0 6 * * 1"   # or omit it and run on demand with argo submit
```

Each run executes `qwen "<instructions>" --approval-mode yolo --output-format text`. `yolo` approves every tool call, since nobody is there to answer; the pod's own restrictions (read-only root filesystem, non-root user, writes confined to the workspace) are the boundary. The exit code is the run's result: `0` is `Succeeded`, anything else `Failed`. An agent with no `instructions` fails before Qwen Code starts.

## Gateway Credentials

By default Qwen Code sends a placeholder key to the gateway, which is all a gateway without per-agent keys needs. If your gateway issues per-agent keys, give the agent its key as `MODEL_API_KEY`; the runtime references it from the environment rather than writing it to disk:

```yaml
spec:
  deployment:
    env:
      - name: MODEL_API_KEY
        valueFrom:
          secretKeyRef:
            name: qwen-code-gateway-key
            key: api-key
```

## What the Operator Created

| Resource | Name | Purpose |
|---|---|---|
| WorkflowTemplate | `qwen-code` | The agent's pod spec; also what `argo submit --from` targets |
| Workflow | `qwen-code` | The long-lived run. Runs the Qwen Code WebSocket terminal container |
| Service | `qwen-code` | ClusterIP on port 8080 |
| NetworkPolicy | `qwen-code` | Allows inbound from other agents in this namespace |
| PVC | `qwen-code-workspace` | 10Gi persistent workspace |
| ConfigMap | `qwen-code-agent` | Injected at `/etc/agent/config.yaml` |

A task-mode agent gets a `CronWorkflow` (when it has a schedule) instead of the long-lived `Workflow`, and no `Service`.
