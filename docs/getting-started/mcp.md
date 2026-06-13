# MCP Tools

Tools extend what agents can do. Every tool the operator manages speaks [MCP](../components/tools.md) — the Model Context Protocol — a JSON-RPC interface that lets agents discover and call tools at runtime.

This guide adds a Kubernetes tool to the agent you deployed in [Quick Start](quickstart.md).

## Prerequisites

A cluster with a running agent from the [Quick Start](quickstart.md).

## Step 1: Create a LanguageTool

A `LanguageTool` describes a tool server. The operator creates and manages its Deployment, Service, and NetworkPolicy.

```bash
kubectl apply -f - <<EOF
apiVersion: langop.io/v1alpha1
kind: LanguageTool
metadata:
  name: k8s
spec:
  image: ghcr.io/language-operator/k8s-tool:latest
  port: 80
EOF
```

Wait for the tool to be ready:

```bash
kubectl wait languagetool/k8s --for=condition=Ready --timeout=60s
```

Once ready, the operator has queried the tool and catalogued its capabilities:

```bash
kubectl get languagetool k8s -o jsonpath='{.status.toolSchemas}' | jq
```

```json
[
  {
    "name": "kubectl_get",
    "description": "Get Kubernetes resources",
    "inputSchema": { ... }
  },
  {
    "name": "kubectl_apply",
    "description": "Apply a Kubernetes manifest",
    "inputSchema": { ... }
  }
]
```

## Step 2: Attach the Tool to Your Agent

Add `spec.tools` to your agent and re-apply:

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
      tools:
        - name: k8s
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
      tools:
        - name: k8s
    EOF
    ```

Watch the agent pod restart with the updated configuration:

```bash
kubectl get pods -w
```

## Step 3: Verify

=== "OpenClaw"

    OpenClaw reads its tool config from `openclaw.json`, which the `openclaw-adapter` init
    container writes on every pod start from the operator-injected `config.yaml`. Confirm
    the k8s server was registered:

    ```bash
    kubectl exec deploy/openclaw -- cat /workspace/.openclaw/openclaw.json | jq '.mcp'
    ```

    ```json
    {
      "servers": {
        "k8s": {
          "url": "http://k8s.language-operator-demo.svc.cluster.local:80"
        }
      }
    }
    ```

=== "OpenCode"

    OpenCode reads its tool config from `/etc/opencode/opencode.jsonc`, which the
    `opencode-adapter` init container writes on every pod start from the
    operator-injected `config.yaml`. Confirm the k8s server was registered:

    ```bash
    kubectl exec deploy/opencode -- cat /etc/opencode/opencode.jsonc | jq '.mcp'
    ```

    ```json
    {
      "k8s": {
        "type": "remote",
        "url": "http://k8s.language-operator-demo.svc.cluster.local:80"
      }
    }
    ```

Your agent now has access to Kubernetes operations. Ask it to list the pods in your namespace.

## What Just Happened?

**When the LanguageTool was created**, the operator:

1. Deployed the tool image as a `Deployment` named `k8s`
2. Created a `Service` named `k8s` on port 80
3. Created a `NetworkPolicy` permitting egress to the Kubernetes API server (auto-detected from the cluster CNI)
4. Queried `POST /mcp` → `tools/list` on the service, and stored the discovered schemas in `status.toolSchemas`

**When the agent was updated**, the operator:

1. Resolved the `k8s` LanguageTool to its in-cluster DNS address
2. Updated `config.yaml` (mounted at `/etc/agent/config.yaml`) with the tool endpoint
3. Rolled the agent pod to pick up the new configuration

On pod start, each runtime's adapter init container reads `config.yaml` and writes the tool endpoints into the runtime's native config before the agent starts — no agent code changes required.

## Using a stdio MCP server

Many MCP servers ship as stdio programs (npm packages, Python scripts) rather than HTTP servers. The operator wraps them with a persistent bridge — no custom image required. Set `transport: stdio` and provide the command argv in `spec.stdio.command`:

```bash
kubectl apply -f - <<EOF
apiVersion: langop.io/v1alpha1
kind: LanguageTool
metadata:
  name: context7
spec:
  transport: stdio
  port: 8080
  stdio:
    command:
      - npx
      - -y
      - "@upstash/context7-mcp"
  networkPolicies:
    egress:
      - to:
          - cidr: "0.0.0.0/0"
        ports:
          - port: 443
            protocol: TCP
EOF
```

The bridge (`ghcr.io/language-operator/mcp-bridge`) bundles Node and Python+uv, so both `npx` and `uvx` commands work. The egress rule is required because `npx` fetches the package from the npm registry on first boot.

Attach it to an agent the same way as any other tool:

```yaml
spec:
  tools:
    - name: context7
```

The endpoint injected into `config.yaml` and `MCP_SERVERS` is the same Streamable HTTP URL (`http://context7.<namespace>.svc.cluster.local:8080/mcp`) — agents don't need to know the underlying transport.

## What's Next?

- [Tools reference](../components/tools.md) — transports, sidecar mode, network policies, and writing your own tool server
- [LanguageTool API reference](../api/languagetool.md) — full spec field documentation
