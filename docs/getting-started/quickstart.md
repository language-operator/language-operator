# Quick Start

Deploy a Claude Code, DeepAgents, OpenClaw, or OpenCode agent in under 5 minutes.

## Prerequisites

A cluster with [Language Operator](installation.md) installed and access to a large language model.

## Step 1: Create a Cluster

A `LanguageCluster` is a managed namespace for logically grouped agents, models, and tools.

```bash
kubectl apply -f - <<EOF
apiVersion: langop.io/v1alpha1
kind: LanguageCluster
metadata:
  name: language-operator-demo
spec:
  domain: agents.example.com
EOF
```

Wait for it to be ready:

```bash
kubectl wait languagecluster/language-operator-demo \
  --for=condition=Ready --timeout=60s
```

Switch into its namespace:

```bash
kubectl config set-context --current --namespace=language-operator-demo
```

## Step 2: Configure an LLM

Store your API key in a secret:

```bash
kubectl create secret generic anthropic-credentials \
  --from-literal=api-key=sk-ant-your-key-here
```

Create a `LanguageModel` pointing to it:

```bash
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

## Step 3: Deploy an Agent

Choose one of the bundled runtimes:

=== "Claude Code"

    Claude Code authenticates to Anthropic directly (not through the gateway), so it doesn't use the `LanguageModel` from Step 2 — and it needs egress to reach `api.anthropic.com`:

    ```bash
    kubectl apply -f - <<EOF
    apiVersion: langop.io/v1alpha1
    kind: LanguageAgent
    metadata:
      name: code-agent
    spec:
      runtime: claude-code
      networkPolicies:
        egress:
          - to:
              - cidr: "0.0.0.0/0"
            ports:
              - port: 443
                protocol: TCP
    EOF
    ```

    You'll log in interactively in Step 5 (or set an API key — see the [Claude Code guide](../runtimes/claude-code.md#authentication)).

=== "DeepAgents"

    DeepAgents runs a task autonomously, so it needs `instructions` alongside the model:

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
        a concise summary to /workspace/summary.md. When the file is written, stop.
    EOF
    ```

=== "OpenClaw"

    ```bash
    kubectl apply -f - <<EOF
    apiVersion: langop.io/v1alpha1
    kind: LanguageAgent
    metadata:
      name: openclaw
    spec:
      runtime: openclaw
      models:
        - name: claude-sonnet
    EOF
    ```

=== "OpenCode"

    ```bash
    kubectl apply -f - <<EOF
    apiVersion: langop.io/v1alpha1
    kind: LanguageAgent
    metadata:
      name: opencode
    spec:
      runtime: opencode
      models:
        - name: claude-sonnet
    EOF
    ```

## Step 4: Check Status

```bash
kubectl get languageagents
```

```
NAME         MODE      PHASE     SCHEDULE   LAST RUN   AGE
researcher   service   Running                         45s
```

Agents run as [Argo Workflows](https://argo-workflows.readthedocs.io/). The operator renders
a `WorkflowTemplate` holding the agent's pod spec, and — because `MODE` is `service`, the
default — a long-lived `Workflow` that runs it:

```bash
kubectl get workflowtemplate,workflow
```

Watch it come up:

```bash
kubectl get lagent -w
```

Wait for `PHASE` to reach `Running`.

## Step 5: Access the Agent

=== "Claude Code"

    Port-forward the WebSocket terminal:

    ```bash
    kubectl port-forward svc/code-agent 8080:8080
    ```

    Open `http://localhost:8080`, then run `/login` inside Claude Code and complete the browser flow to authenticate with your Claude account. Credentials persist on the workspace PVC. See the [Claude Code guide](../runtimes/claude-code.md#authentication) for headless API-key / token auth.

=== "DeepAgents"

    DeepAgents starts working as soon as its pod is `Running` — there's nothing to log into. The run streams to stdout:

    ```bash
    argo logs @latest -n <namespace> -f
    ```

    Or, without the `argo` CLI:

    ```bash
    kubectl logs -f -n <namespace> -l app.kubernetes.io/name=researcher -c main
    ```

    For a browser view of the same stream plus human-in-the-loop controls, port-forward the service:

    ```bash
    kubectl port-forward svc/researcher 8080:8080
    # open http://localhost:8080
    ```

=== "OpenClaw"

    Retrieve the auto-generated gateway token:

    ```bash
    TOKEN=$(kubectl get secret openclaw-runtime -o jsonpath='{.data.OPENCLAW_GATEWAY_TOKEN}' | base64 -d)
    echo "Token: $TOKEN"
    ```

    If you have a domain configured on your `LanguageCluster`, open `https://openclaw.<cluster-domain>` and enter the token when prompted, or connect the OpenClaw CLI directly to `wss://openclaw.<cluster-domain>`.

    Otherwise, port-forward for local access:

    ```bash
    kubectl port-forward svc/openclaw 18789:18789
    # connect to ws://localhost:18789 with the token above
    ```

    OpenClaw uses a WebSocket gateway on port 18789. Connect using the OpenClaw browser extension or CLI client (see [github.com/openclaw/openclaw](https://github.com/openclaw/openclaw)).

=== "OpenCode"

    Access is gated by the cluster OIDC proxy — there is no separate opencode password.

    If you have a domain and [auth enabled](../components/clusters.md#authentication) on your `LanguageCluster`, open `https://opencode.<cluster-domain>` and sign in through the cluster's OIDC provider.

    Otherwise, port-forward for local access (this bypasses the proxy, so no login is required):

    ```bash
    kubectl port-forward svc/opencode 3000:3000
    ```

    Then open `http://localhost:3000` or attach the TUI (OpenCode v1.0.10+):

    ```bash
    opencode attach http://localhost:3000
    ```

    Note: `opencode attach` against the `https://opencode.<cluster-domain>` URL won't work — the OIDC proxy needs a browser session. Attach via the port-forwarded `http://localhost:3000` instead. opencode has no built-in auth, so enable cluster auth to protect it.

## Step 6: Run an Agent on a Schedule

The agents above are **service** agents: always on, and addressable through a Service. The
other mode is **task** — the agent runs, does its work, exits, and is invoked again on a
schedule. That fits any agent whose instructions read "do X, then stop".

```bash
kubectl apply -f - <<EOF
apiVersion: langop.io/v1alpha1
kind: LanguageAgent
metadata:
  name: nightly-summary
spec:
  runtime: deepagents
  models:
    - name: claude-sonnet
  instructions: |
    Summarise yesterday's activity to /workspace/summary.md, then stop.
  execution:
    mode: task
    schedule: "0 3 * * *"
EOF
```

```bash
kubectl get lagent nightly-summary
```

```
NAME              MODE   PHASE     SCHEDULE     LAST RUN    AGE
nightly-summary   task   Pending   0 3 * * *                10s
```

The operator creates a `CronWorkflow` that fires on that schedule. To run it now rather than
waiting:

```bash
argo submit --from workflowtemplate/nightly-summary
argo list
argo logs @latest
```

A task agent has no Service and no Ingress — its pods only exist while a run is in flight —
so `spec.ports` is rejected for it.

See [Execution Modes](../guides/execution-modes.md) for suspending agents, concurrency
policies, run deadlines, and the status fields that report run history.
