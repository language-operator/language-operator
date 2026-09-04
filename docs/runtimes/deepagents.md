# Deploying deepagents

[deepagents](https://github.com/langchain-ai/deepagents) is LangChain's framework for building a single, capable agent — planning, sub-agents, a virtual filesystem, and MCP tools. The `deepagents` runtime is installed by the `language-operator-runtimes` chart.

Unlike the coding-CLI runtimes ([Claude Code](claude-code.md), [OpenCode](opencode.md), [OpenClaw](openclaw.md)), deepagents is not an interactive terminal you drive. It is an **autonomous executor**: on startup it reads the agent's `spec.instructions`, runs that task once — streaming every step to **stdout** (so `argo logs` is the primary UI) and a live web view — then finishes. The task *is* the `instructions` field.

That shape makes deepagents the natural fit for `spec.execution.mode: task`: give it a schedule and each fire is a run that starts, does the work, and exits. Run it in the default `service` mode and it does the work once and then sits idle until you change its spec. See [Execution Modes](../guides/execution-modes.md).

## Prerequisites

- Language Operator [installed](../getting-started/installation.md), including the `language-operator-runtimes` chart (provides the `deepagents` runtime)
- A [`LanguageCluster`](../components/clusters.md) to deploy into, with your `kubectl` context set to its namespace (examples below assume a cluster named `demo-cluster`)
- An LLM provider API key, or a local model endpoint (e.g. Ollama)

## Instructions

### Configure a Model

deepagents reaches the model through the in-cluster LiteLLM gateway — it never holds a real provider key. Register a `LanguageModel` for the gateway to route to:

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

### Deploy a deepagents Agent

A deepagents agent needs two things to do work: a `models` reference (the primary model) and `instructions` (the task it runs autonomously).

=== "Anthropic"

    ```bash
    kubectl apply -f - <<EOF
    apiVersion: langop.io/v1alpha1
    kind: LanguageAgent
    metadata:
      name: researcher
    spec:
      runtime: deepagents
      models:
        - name: claude-sonnet
      instructions: |
        Research the public API of the "deepagents" Python library, then write
        a concise, well-organized summary to /workspace/summary.md covering what
        the library is and its main entry points. When the file is written, stop.
    EOF
    ```

=== "OpenAI"

    ```bash
    kubectl apply -f - <<EOF
    apiVersion: langop.io/v1alpha1
    kind: LanguageAgent
    metadata:
      name: researcher
    spec:
      runtime: deepagents
      models:
        - name: gpt-4o
      instructions: |
        Research the public API of the "deepagents" Python library, then write
        a concise, well-organized summary to /workspace/summary.md covering what
        the library is and its main entry points. When the file is written, stop.
    EOF
    ```

=== "Local Model"

    ```bash
    kubectl apply -f - <<EOF
    apiVersion: langop.io/v1alpha1
    kind: LanguageAgent
    metadata:
      name: researcher
    spec:
      runtime: deepagents
      models:
        - name: llama3
      instructions: |
        Research the public API of the "deepagents" Python library, then write
        a concise, well-organized summary to /workspace/summary.md covering what
        the library is and its main entry points. When the file is written, stop.
    EOF
    ```

!!! note

    Without `instructions` the agent comes up healthy but has no task to run. Without a `models` reference it cannot resolve a model. Both are reported in the pod logs on startup.

### Verify

```bash
kubectl get languageagents
kubectl get lagent -w
```

Wait for `PHASE` to reach `Running`. The `MODE` column shows `service`: this runtime is a
long-lived, addressable agent, which is why it has a Service you can port-forward to. See
[Execution Modes](../guides/execution-modes.md).

### Watch it run

The agent starts working as soon as the pod is `Running` — there is nothing to log into. The run streams to stdout:

```bash
argo logs @latest -n <namespace> -f
```

Or, without the `argo` CLI:

```bash
kubectl logs -f -n <namespace> -l app.kubernetes.io/name=researcher -c main
```

You'll see the agent plan, act, and (for this task) write `/workspace/summary.md`.

### Connect to the live view

For a browser view of the same stream plus human-in-the-loop controls, port-forward the service:

```bash
kubectl port-forward svc/researcher 8080:8080
# open http://localhost:8080
```

The thin server exposes `GET /` (live UI), `GET /events` (SSE), `GET /state` (status + any pending interrupt), `POST /resume`, and `POST /restart`. `GET /health` backs the pod's probes.

## Human-in-the-loop

By default the runtime pauses before **side-effecting** tools — the built-in `write_file`/`edit_file` plus every MCP tool — and waits for approval. Read-only operations (`ls`/`read_file`) never pause. While paused the agent reports `interrupted`; approve or reject from the live view at `/`, or with `POST /resume`.

Tune the policy with the `HITL_TOOLS` environment variable:

| `HITL_TOOLS` | Behavior |
|---|---|
| _unset_ (default) | Pause before `write_file`/`edit_file` and all MCP tools |
| `none` or `""` | Never pause — fully autonomous |
| `*` | Pause before every tool (built-in writers + all MCP tools) |
| `write_file,edit_file,…` | Pause only before the named tools |

```yaml
spec:
  runtime: deepagents
  deployment:
    env:
      - name: HITL_TOOLS
        value: "none"   # hands-off; the task runs end to end
```

## Adding tools

Reference any [`LanguageTool`](../components/tools.md) and the operator resolves its MCP endpoint into the agent; deepagents loads it as a tool automatically:

```yaml
spec:
  runtime: deepagents
  models:
    - name: claude-sonnet
  tools:
    - name: context7
  instructions: |
    Use the context7 tool to look up the current API, then …
```

## What the Operator Created

| Resource | Name | Purpose |
|---|---|---|
| WorkflowTemplate | `researcher` | The agent's pod spec; also what `argo submit --from` targets |
| Workflow | `researcher` | The long-lived run. Runs the deepagents container |
| Service | `researcher` | ClusterIP on port 8080 |
| NetworkPolicy | `researcher` | Allows inbound from other agents in this namespace |
| PVC | `researcher-workspace` | 10Gi persistent workspace (agent files + checkpoint DB) |
| ConfigMap | `researcher-agent` | Injected at `/etc/agent/config.yaml` |
```
