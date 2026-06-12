# tools/context7

Registers [Context7](https://context7.com) as a `LanguageTool` — gives agents up-to-date,
version-specific documentation and code examples for thousands of libraries over MCP, so they
stop hallucinating APIs and writing against stale versions.

Context7 ships as a **stdio** MCP server ([`@upstash/context7-mcp`](https://www.npmjs.com/package/@upstash/context7-mcp)).
This example sets `spec.transport: stdio` and lets the **operator** run it under a pinned,
persistent stdio→Streamable-HTTP bridge — one long-lived process that serves the `/mcp` endpoint
the operator discovers and agents call. You don't wire up a bridge, a cache volume, or `HOME`
yourself; the operator injects all of that.

This is **registration only**. A `LanguageTool` does nothing until an agent references it: add
`- name: context7` under the agent's `spec.tools`, and the operator injects the tool's URL into
the agent via `MCP_SERVERS` and `/etc/agent/config.yaml`.

## API key is optional

Context7 works **without** an API key at a lower rate limit, which is fine for kicking the tires.
Provide a [Context7 API key](https://context7.com/dashboard) via `CONTEXT7_API_KEY` for higher
limits. The tool references the credential with `optional: true`, so the pod starts whether or not
the secret exists.

## Prerequisites

- Language Operator [installed](https://langop.io/docs/getting-started/installation/)
- `kubectl` configured for your cluster
- `envsubst` (`brew install gettext` on macOS, pre-installed on most Linux distros)
- A `LanguageCluster` already applied in the target namespace (see [clusters/basic](../../clusters/basic/))
- An agent to attach it to (see [agents/opencode](../../agents/opencode/))
- _Optional:_ a Context7 API key for higher rate limits

## Variables

| Variable | Required | Default | Description |
|----------|----------|---------|-------------|
| `CLUSTER_NAME` | yes | — | Namespace of the target LanguageCluster |
| `CONTEXT7_API_KEY` | no | — | Context7 API key (when set, creates the `context7-mcp-credentials` secret for higher rate limits) |
| `TOOL_NAME` | no | `context7` | Name of the LanguageTool CR (the name agents reference) |

> **Egress:** the manifest grants outbound HTTPS (`0.0.0.0/0:443`) because `npx` fetches the
> server package from the npm registry at boot and the server calls Context7's API at runtime.
> The operator's default tool policy denies external egress, so this rule is required. Scope it
> tighter with `dns:` peers if your setup resolves them reliably.

## Install

Without an API key (lower rate limit):
```bash
CLUSTER_NAME=my-cluster bash examples/tools/context7/install.sh
```

With an API key (higher rate limit):
```bash
CLUSTER_NAME=my-cluster CONTEXT7_API_KEY=ctx7_... bash examples/tools/context7/install.sh
```

Dry-run (prints rendered YAML, applies nothing):
```bash
CLUSTER_NAME=my-cluster bash examples/tools/context7/install.sh --dry-run
```

## What's created

- `Secret/context7-mcp-credentials` — holds the API key (**only when `CONTEXT7_API_KEY` is set**)
- `LanguageTool/context7` — the MCP tool CR (`transport: stdio`)
- `Deployment/context7` — the operator-injected MCP bridge running `@upstash/context7-mcp` over
  Streamable HTTP on port 8080
- `Service/context7` — exposes the `/mcp` endpoint in-cluster

## Attaching it to an agent

```yaml
apiVersion: langop.io/v1alpha1
kind: LanguageAgent
metadata:
  name: my-agent
  namespace: my-cluster
spec:
  tools:
    - name: context7
```

The operator resolves the endpoint to
`http://context7.my-cluster.svc.cluster.local:8080` and injects it via `MCP_SERVERS`.

## Teardown

```bash
kubectl delete languagetool context7 -n my-cluster
kubectl delete secret context7-mcp-credentials -n my-cluster --ignore-not-found
```
