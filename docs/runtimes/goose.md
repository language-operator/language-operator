# Deploying Goose

[Goose](https://github.com/aaif-goose/goose) is an open-source, general-purpose AI agent: it writes code, but it also installs, runs, edits and tests whatever a task needs, through any LLM. The `goose` runtime runs the Goose CLI inside tmux behind a browser terminal, and is installed by the `language-operator-runtimes` chart. No Goose account or login is needed: every model call goes through the cluster's [model gateway](../components/models.md) over the OpenAI chat-completions API, so any [LanguageModel](../api/languagemodel.md) works.

The runtime image is built in [goose-adapter](https://github.com/language-operator/goose-adapter) on the shared `coding-runtime` base, like the [Claude Code](claude-code.md), [OpenCode](opencode.md), [Kilo](kilo.md), [Qwen Code](qwen-code.md) and [Cursor](cursor.md) runtimes.

## Prerequisites

- Language Operator [installed](../getting-started/installation.md), including the `language-operator-runtimes` chart (provides the `goose` runtime)
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

### Deploy Goose

```bash
kubectl apply -f - <<EOF
apiVersion: langop.io/v1alpha1
kind: LanguageAgent
metadata:
  name: goose
spec:
  runtime: goose
  models:
    - name: claude-sonnet   # or gpt-4o, llama3
EOF
```

On start, the runtime writes Goose's configuration from the operator's `/etc/agent/config.yaml`: the gateway as Goose's `openai` provider, the first referenced model, and one extension per [`LanguageTool`](../components/tools.md) the agent references. Header tokens for those tools are referenced from the container's environment, so they never land on disk. The persona, and in service mode `spec.instructions`, become Goose's standing hints for every session. Goose's telemetry is turned off.

### Verify

```bash
kubectl get languageagents
kubectl get lagent -w
```

Wait for `PHASE` to reach `Running`. The `MODE` column shows `service`: the session is long-lived and addressable. See
[Execution Modes](../guides/execution-modes.md).

### Connect

Access is gated by the cluster's OIDC proxy — there is no separate Goose password.

If your `LanguageCluster` has a domain and [auth enabled](../components/clusters.md#authentication),
open https://goose.demo-cluster.\<your-domain\> and sign in through the cluster's OIDC provider.

For local access, port-forward the service (this bypasses the proxy, so no login is required):

```bash
kubectl port-forward svc/goose 8080:8080
```

Then open `http://localhost:8080` in your browser. The Goose session lives in tmux, so it survives page reloads, and after a pod restart Goose resumes the previous session from the workspace volume. If Goose exits with an error — an unknown model, an unreachable gateway — the terminal drops to a shell instead of closing, so the message stays on screen.

!!! warning

    The terminal has no built-in authentication. When the cluster does **not** enable auth, it is
    exposed unauthenticated on its ingress. Enable cluster auth to protect it.

## Task Mode

Goose also runs headless as a [task](../guides/execution-modes.md): each run does the work in `spec.instructions` and exits.

```yaml
apiVersion: langop.io/v1alpha1
kind: LanguageAgent
metadata:
  name: weekly-report
spec:
  runtime: goose
  models:
    - name: claude-sonnet
  instructions: |
    Collect last week's merged pull requests and write a summary report.
  execution:
    mode: task
    schedule: "0 6 * * 1"   # or omit it and run on demand with argo submit
```

Each run is one `goose run` of the instructions, and its result is the run's result: `Succeeded` or `Failed`. An agent with no `instructions` fails without calling the model.

!!! note "How failures are detected"

    Goose itself exits `0` on most gateway errors — an unknown model, a 4xx/5xx from the gateway,
    an unreachable gateway — reporting them as an assistant message instead. The runtime checks
    the run's final message and fails the run when it did not come from the model, so a broken
    model configuration shows up as `Failed` rather than a quiet success.

## Gateway Credentials

The operator gives every agent its own gateway key as `MODEL_API_KEY`, and Goose sends it on every model call, so there is nothing to configure. The gateway rejects calls without a valid key and attributes each one to the agent. See [Gateway authentication](../components/models.md#gateway-authentication).

## What the Operator Created

| Resource | Name | Purpose |
|---|---|---|
| WorkflowTemplate | `goose` | The agent's pod spec; also what `argo submit --from` targets |
| Workflow | `goose` | The long-lived run. Runs the Goose WebSocket terminal container |
| Service | `goose` | ClusterIP on port 8080 |
| NetworkPolicy | `goose` | Allows inbound from other agents in this namespace |
| PVC | `goose-workspace` | 10Gi persistent workspace |
| ConfigMap | `goose-agent` | Injected at `/etc/agent/config.yaml` |

A task-mode agent gets a `CronWorkflow` (when it has a schedule) instead of the long-lived `Workflow`, and no `Service`.
