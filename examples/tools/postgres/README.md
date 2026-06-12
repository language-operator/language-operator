# tools/postgres

Registers a Postgres MCP server as a `LanguageTool` — lets agents inspect schemas and run
read-oriented SQL against a database over MCP.

The popular [`@modelcontextprotocol/server-postgres`](https://www.npmjs.com/package/@modelcontextprotocol/server-postgres)
is a **stdio** server, but the operator and agents speak **HTTP** MCP. This example sets
`spec.transport: stdio` and lets the **operator** run the server under a pinned, persistent
stdio→Streamable-HTTP bridge — one long-lived process that serves the `/mcp` endpoint the
operator discovers.

The Postgres server takes its connection string as a command argument. The bridge runs the stdio
command through a shell, so [languagetool.postgres.yaml](languagetool.postgres.yaml) passes the literal `$DATABASE_URL` and it's
interpolated from a secret-backed env var at runtime — the rendered `LanguageTool` only ever
contains `$DATABASE_URL`, never the real URL.

This is **registration only**. A `LanguageTool` does nothing until an agent references it: add
`- name: postgres` under the agent's `spec.tools`, and the operator injects the tool's URL into
the agent via `MCP_SERVERS` and `/etc/agent/config.yaml`.

## Prerequisites

- Language Operator [installed](https://langop.io/docs/getting-started/installation/)
- `kubectl` configured for your cluster
- `envsubst` (`brew install gettext` on macOS, pre-installed on most Linux distros)
- A `LanguageCluster` already applied in the target namespace (see [clusters/basic](../../clusters/basic/))
- A Postgres connection string reachable from the cluster (ideally a read-only role)
- An agent to attach it to (see [agents/opencode](../../agents/opencode/))

## Variables

| Variable | Required | Default | Description |
|----------|----------|---------|-------------|
| `CLUSTER_NAME` | yes | — | Namespace of the target LanguageCluster |
| `DATABASE_URL` | yes | — | Postgres connection string (used to create the `postgres-mcp-credentials` secret) |
| `TOOL_NAME` | no | `postgres` | Name of the LanguageTool CR (the name agents reference) |

> **Egress:** the manifest grants outbound HTTPS (`443`, for the npm fetch) and Postgres
> (`5432`). The operator's default tool policy denies external egress, so these are required.
> Adjust the database port if yours differs.

## Install

```bash
CLUSTER_NAME=my-cluster \
  DATABASE_URL=postgresql://reader:pass@db.example.com:5432/app \
  bash examples/tools/postgres/install.sh
```

Dry-run (prints rendered YAML, no connection string needed):
```bash
CLUSTER_NAME=my-cluster bash examples/tools/postgres/install.sh --dry-run
```

## What's created

- `Secret/postgres-mcp-credentials` — holds the connection string
- `LanguageTool/postgres` — the MCP tool CR (`transport: stdio`)
- `Deployment/postgres` — the operator-injected bridge running the Postgres stdio server on port 8080
- `Service/postgres` — exposes the `/mcp` endpoint in-cluster

## Attaching it to an agent

```yaml
apiVersion: langop.io/v1alpha1
kind: LanguageAgent
metadata:
  name: my-agent
  namespace: my-cluster
spec:
  tools:
    - name: postgres
```

The operator resolves the endpoint to
`http://postgres.my-cluster.svc.cluster.local:8080` and injects it via `MCP_SERVERS`.

## Teardown

```bash
kubectl delete languagetool postgres -n my-cluster
kubectl delete secret postgres-mcp-credentials -n my-cluster
```
