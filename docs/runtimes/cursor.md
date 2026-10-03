# Deploying Cursor

[Cursor](https://cursor.com/cli)'s CLI agent runs as the `cursor` runtime: its TUI inside tmux behind a browser terminal, installed by the `language-operator-runtimes` chart. The image is built in [cursor-adapter](https://github.com/language-operator/cursor-adapter) on the shared `coding-runtime` base.

Cursor is a **vendor-key runtime**, like [Claude Code](claude-code.md): it authenticates to Cursor with your Cursor key and talks to Cursor directly, **not through the cluster's model gateway**. See [What a vendor-key runtime means](#what-a-vendor-key-runtime-means).

## Prerequisites

- Language Operator [installed](../getting-started/installation.md), including the `language-operator-runtimes` chart (provides the `cursor` runtime)
- A [`LanguageCluster`](../components/clusters.md) to deploy into, with your `kubectl` context set to its namespace (examples below assume a cluster named `demo-cluster`)
- A Cursor account, and for headless or task-mode agents a Cursor API key

## Instructions

### Deploy Cursor

```bash
kubectl apply -f - <<EOF
apiVersion: langop.io/v1alpha1
kind: LanguageAgent
metadata:
  name: cursor
spec:
  runtime: cursor
  instructions: |
    You are an expert software engineer. Help with code review,
    debugging, and implementation tasks.
  networkPolicies:
    egress:
      - to:
          - cidr: "0.0.0.0/0"
        ports:
          - port: 443
            protocol: TCP
EOF
```

The egress rule lets the agent reach Cursor's API; without it, the agent's NetworkPolicy allows only in-cluster traffic.

### Verify

```bash
kubectl get languageagents
kubectl get lagent -w
```

Wait for `PHASE` to reach `Running`. The `MODE` column shows `service`: the TUI is long-lived and addressable. See
[Execution Modes](../guides/execution-modes.md).

### Connect

Access is gated by the cluster's OIDC proxy. If your `LanguageCluster` has a domain and [auth enabled](../components/clusters.md#authentication),
open https://cursor.demo-cluster.\<your-domain\> and sign in through the cluster's OIDC provider.

For local access, port-forward the service (this bypasses the proxy, so no login is required):

```bash
kubectl port-forward svc/cursor 8080:8080
```

Then open `http://localhost:8080` in your browser. With no key configured, the terminal opens on Cursor's sign-in screen (see [Authentication](#authentication)).

!!! warning

    The terminal has no built-in authentication. When the cluster does **not** enable auth, it is
    exposed unauthenticated on its ingress. Enable cluster auth to protect it.

## Authentication

Cursor reads its key from `CURSOR_API_KEY`. Give it one of three ways.

### Interactive login (service mode only)

With no key set, the terminal opens on Cursor's sign-in screen. The login is kept on the workspace volume, so pod restarts do not ask again. Task-mode agents cannot do this — nobody is there to sign in — and need a key.

### Per agent

Store the key in a Secret whose key is literally named `CURSOR_API_KEY`, and reference it from the agent's `spec.credentials`:

```bash
kubectl create secret generic cursor-credentials --from-literal=CURSOR_API_KEY=<key>
```

```yaml
spec:
  runtime: cursor
  credentials:
    - name: CURSOR_API_KEY
      valueFrom:
        name: cursor-credentials
```

Always use `valueFrom` (or `value`): a credential declared by name alone is filled by the operator with a random value.

### For every cursor agent

Set the runtime chart's `credentials.apiKeySecret` and every agent on the `cursor` runtime gets the key, with no per-agent configuration:

```bash
helm upgrade --install language-operator-runtimes \
  language-operator/language-operator-runtimes \
  --namespace language-operator \
  --set cursor.credentials.apiKeySecret=cursor-credentials
```

The Secret must exist **in each agent's namespace**. It is off by default because an agent whose namespace lacks the Secret will not start.

## What a vendor-key runtime means

Cursor has no custom base URL, so its traffic goes straight to Cursor under the key's account, not through the [model gateway](../components/models.md). As a result:

- **`spec.models` does not apply.** Gateway model names are not Cursor model names. Cursor uses its default model unless you set `CURSOR_MODEL`, which is passed as `--model`; an unknown name fails the run:

    ```yaml
    spec:
      deployment:
        env:
          - name: CURSOR_MODEL
            value: <cursor model name>
    ```

- **Gateway features do not cover it**: gateway logging, per-model rate limits and per-agent key attribution see none of Cursor's requests.
- **The agent needs egress** to Cursor's API (port 443), as in the deploy example above.

## Instructions, Persona and Tools

- **`spec.instructions`** is the opening message of a fresh service-mode session, and the prompt of a task run. A resumed session is not sent it again.
- **The persona** becomes an always-applied Cursor rule, `/workspace/.cursor/rules/langop-persona.mdc`. Cursor loads rules from the working directory and its parents, so it applies inside a cloned repository without touching the repository.
- **[`LanguageTool`s](../components/tools.md)** become MCP servers in Cursor's global MCP file, so they need no per-project approval. Header values that reference a secret are written as `${env:NAME}`, which Cursor expands itself, so the secret is never written to disk. A tool whose headers cannot all be resolved is left out with a warning.

## Task Mode

Cursor also runs headless as a [task](../guides/execution-modes.md). Each run executes `agent -p` with `spec.instructions` as the prompt; when the run was started with an [`event`](../guides/execution-modes.md#per-run-inputs), its payload is appended, labelled as data rather than instructions. The exit code is the run's result: `0` is `Succeeded`, anything else `Failed`.

```yaml
apiVersion: langop.io/v1alpha1
kind: LanguageAgent
metadata:
  name: changelog
spec:
  runtime: cursor
  credentials:
    - name: CURSOR_API_KEY
      valueFrom:
        name: cursor-credentials
  repository:
    url: https://github.com/acme/api.git
  instructions: |
    Summarise the commits merged since the last tag into CHANGELOG.md.
  execution:
    mode: task
    schedule: "0 6 * * 1"
  networkPolicies:
    egress:
      - to:
          - cidr: "0.0.0.0/0"
        ports:
          - port: 443
            protocol: TCP
```

!!! warning

    Task runs use `--force`: nobody is there to approve a tool call, so every tool the agent
    chooses to run, runs. Scope what a task agent can reach — its repository, its tokens, its
    tools — accordingly.

## What the Operator Created

| Resource | Name | Purpose |
|---|---|---|
| WorkflowTemplate | `cursor` | The agent's pod spec; also what `argo submit --from` targets |
| Workflow | `cursor` | The long-lived run. Runs the Cursor WebSocket terminal container |
| Service | `cursor` | ClusterIP on port 8080 |
| NetworkPolicy | `cursor` | Allows inbound from other agents in this namespace, plus the egress you declare |
| PVC | `cursor-workspace` | 10Gi persistent workspace |
| ConfigMap | `cursor-agent` | Injected at `/etc/agent/config.yaml` |

A task-mode agent gets a `CronWorkflow` (when it has a schedule) instead of the long-lived `Workflow`, and no `Service`.
