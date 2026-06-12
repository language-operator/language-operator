# tools/github

Registers a GitHub MCP server as a `LanguageTool` — gives agents structured access to
repositories, issues, and pull requests over MCP.

The popular [`@modelcontextprotocol/server-github`](https://www.npmjs.com/package/@modelcontextprotocol/server-github)
is a **stdio** server, but the operator and agents speak **HTTP** MCP. This example sets
`spec.transport: stdio` and lets the **operator** run the server under a pinned, persistent
stdio→Streamable-HTTP bridge — one long-lived process that serves the `/mcp` endpoint the
operator discovers. You don't wire up a bridge, cache volume, or `HOME` yourself.

This is **registration only**. A `LanguageTool` does nothing until an agent references it: add
`- name: github` under the agent's `spec.tools`, and the operator injects the tool's URL into the
agent via `MCP_SERVERS` and `/etc/agent/config.yaml`.

## Prerequisites

- Language Operator [installed](https://langop.io/docs/getting-started/installation/)
- `kubectl` configured for your cluster
- `envsubst` (`brew install gettext` on macOS, pre-installed on most Linux distros)
- A `LanguageCluster` already applied in the target namespace (see [clusters/basic](../../clusters/basic/))
- A GitHub personal access token with the scopes your agents need (e.g. `repo`, `read:org`)
- An agent to attach it to (see [agents/opencode](../../agents/opencode/))

## Variables

| Variable | Required | Default | Description |
|----------|----------|---------|-------------|
| `CLUSTER_NAME` | yes | — | Namespace of the target LanguageCluster |
| `GITHUB_TOKEN` | yes | — | GitHub PAT (used to create the `github-mcp-credentials` secret) |
| `TOOL_NAME` | no | `github` | Name of the LanguageTool CR (the name agents reference) |

> **Egress:** the manifest grants outbound HTTPS (`0.0.0.0/0:443`) because `npx` fetches the
> server package from the npm registry at boot and the server calls the GitHub API at runtime.
> The operator's default tool policy denies external egress, so this rule is required.

## Install

```bash
CLUSTER_NAME=my-cluster GITHUB_TOKEN=ghp_... bash examples/tools/github/install.sh
```

Dry-run (prints rendered YAML, no token needed):
```bash
CLUSTER_NAME=my-cluster bash examples/tools/github/install.sh --dry-run
```

## What's created

- `Secret/github-mcp-credentials` — holds the GitHub token
- `LanguageTool/github` — the MCP tool CR (`transport: stdio`)
- `Deployment/github` — the operator-injected bridge running the GitHub stdio server on port 8080
- `Service/github` — exposes the `/mcp` endpoint in-cluster

## Attaching it to an agent

```yaml
apiVersion: langop.io/v1alpha1
kind: LanguageAgent
metadata:
  name: my-agent
  namespace: my-cluster
spec:
  tools:
    - name: github
```

The operator resolves the endpoint to
`http://github.my-cluster.svc.cluster.local:8080` and injects it via `MCP_SERVERS`.

## Swapping the server

The stdio command is `spec.stdio.command` in [languagetool.github.yaml](languagetool.github.yaml). Swap it for any other stdio
MCP server — a different GitHub build, or another tool entirely — and the operator bridges it the
same way.

## Teardown

```bash
kubectl delete languagetool github -n my-cluster
kubectl delete secret github-mcp-credentials -n my-cluster
```
