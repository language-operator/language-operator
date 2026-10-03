# Deploying Kilo

[Kilo](https://kilo.ai) is an open-source AI coding agent; its CLI (`@kilocode/cli`) is a fork of OpenCode. The `kilo` runtime runs the Kilo TUI inside tmux behind a browser terminal, and is installed by the `language-operator-runtimes` chart. No Kilo account is needed: every model call goes through the cluster's [model gateway](../components/models.md).

The runtime image is built in [kilo-adapter](https://github.com/language-operator/kilo-adapter) on the shared `coding-runtime` base, like the [Claude Code](claude-code.md) and [OpenCode](opencode.md) runtimes.

## Prerequisites

- Language Operator **v0.3.14 or later** [installed](../getting-started/installation.md), including the `language-operator-runtimes` chart (provides the `kilo` runtime). Kilo calls the gateway's `/v1/responses` endpoint; earlier gateways served it only for hosted providers, not for `openai-compatible` models such as Ollama.
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

### Deploy Kilo

```bash
kubectl apply -f - <<EOF
apiVersion: langop.io/v1alpha1
kind: LanguageAgent
metadata:
  name: kilo
spec:
  runtime: kilo
  models:
    - name: claude-sonnet   # or gpt-4o, llama3
EOF
```

On start, the runtime writes Kilo's config (`kilo.jsonc`) from the operator's `/etc/agent/config.yaml`: the first referenced model, reached through the gateway, and every [`LanguageTool`](../components/tools.md) the agent references, as a remote MCP server. `spec.instructions`, when set, is loaded as standing context for every session. Kilo's telemetry, session upload, sharing and auto-update are turned off.

### Verify

```bash
kubectl get languageagents
kubectl get lagent -w
```

Wait for `PHASE` to reach `Running`. The `MODE` column shows `service`: the TUI is long-lived and addressable. See
[Execution Modes](../guides/execution-modes.md).

### Connect

Access is gated by the cluster's OIDC proxy — there is no separate Kilo password.

If your `LanguageCluster` has a domain and [auth enabled](../components/clusters.md#authentication),
open https://kilo.demo-cluster.\<your-domain\> and sign in through the cluster's OIDC provider.

For local access, port-forward the service (this bypasses the proxy, so no login is required):

```bash
kubectl port-forward svc/kilo 8080:8080
```

Then open `http://localhost:8080` in your browser. The session lives in tmux, so it survives page reloads, and after a pod restart Kilo resumes the previous conversation from the workspace volume.

!!! warning

    The terminal has no built-in authentication. When the cluster does **not** enable auth, it is
    exposed unauthenticated on its ingress. Enable cluster auth to protect it.

## Task Mode

Kilo also runs headless as a [task](../guides/execution-modes.md): each run does the work in `spec.instructions` and exits.

```yaml
apiVersion: langop.io/v1alpha1
kind: LanguageAgent
metadata:
  name: changelog
spec:
  runtime: kilo
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

Each run executes `kilo run --auto --format json` with the instructions as the prompt. `--auto` approves every permission Kilo would otherwise ask for, since nobody is there to answer. The exit code is the run's result: `0` is `Succeeded`; `1` is `Failed`, for example on an unknown model or an unreachable gateway. An agent with no `instructions` fails before Kilo starts.

## Gateway Credentials

The operator gives every agent its own gateway key as `MODEL_API_KEY`, and Kilo sends it on every model call, so there is nothing to configure. The gateway rejects calls without a valid key and attributes each one to the agent. See [Gateway authentication](../components/models.md#gateway-authentication).

## What the Operator Created

| Resource | Name | Purpose |
|---|---|---|
| WorkflowTemplate | `kilo` | The agent's pod spec; also what `argo submit --from` targets |
| Workflow | `kilo` | The long-lived run. Runs the Kilo WebSocket terminal container |
| Service | `kilo` | ClusterIP on port 8080 |
| NetworkPolicy | `kilo` | Allows inbound from other agents in this namespace |
| PVC | `kilo-workspace` | 10Gi persistent workspace |
| ConfigMap | `kilo-agent` | Injected at `/etc/agent/config.yaml` |

A task-mode agent gets a `CronWorkflow` (when it has a schedule) instead of the long-lived `Workflow`, and no `Service`.
