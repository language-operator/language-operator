# tools/postgres

Deploys a **self-contained Postgres** database (a StatefulSet with a persistent volume) and
registers the [`@modelcontextprotocol/server-postgres`](https://www.npmjs.com/package/@modelcontextprotocol/server-postgres)
MCP server against it as a `LanguageTool` — letting agents inspect schemas and run SQL with zero
external database setup.

The popular `@modelcontextprotocol/server-postgres` is a **stdio** server, but the operator and
agents speak **HTTP** MCP. This example sets `spec.transport: stdio` and lets the **operator** run
the server under a pinned, persistent stdio→Streamable-HTTP bridge — one long-lived process that
serves the `/mcp` endpoint the operator discovers.

`install.sh` generates a database password, writes it (with the derived connection string) into a
single `postgres-mcp-credentials` secret, and **auto-wires** that one secret into both the
database (which reads `username`/`password`/`dbname`) and the tool (whose `$DATABASE_URL` reads
`url`) — so the two can never drift out of sync. The rendered `LanguageTool` only ever contains
the literal `$DATABASE_URL`, never the real connection string.

The bundled database is seeded once with a tiny **demo schema** (`customers` and `orders` with a
few sample rows) so an attached agent has something to query immediately.

This is **registration only** for the agent side: a `LanguageTool` does nothing until an agent
references it. Add `- name: postgres` under the agent's `spec.tools`, and the operator injects the
tool's URL into the agent via `MCP_SERVERS` and `/etc/agent/config.yaml`.

## Prerequisites

- Language Operator [installed](https://langop.io/docs/getting-started/installation/)
- `kubectl` configured for your cluster
- `envsubst` (`brew install gettext` on macOS, pre-installed on most Linux distros)
- `openssl` (for password generation — pre-installed on macOS and most Linux distros)
- A `LanguageCluster` already applied in the target namespace (see [clusters/basic](../../clusters/basic/))
- A default StorageClass in the cluster (the StatefulSet requests a 1Gi PVC)
- An agent to attach it to (see [agents/opencode](../../agents/opencode/))

## Variables

| Variable | Required | Default | Description |
|----------|----------|---------|-------------|
| `CLUSTER_NAME` | yes | — | Namespace of the target LanguageCluster |
| `TOOL_NAME` | no | `postgres` | Name of the LanguageTool CR (the name agents reference); also prefixes the bundled `${TOOL_NAME}-db` resources |
| `POSTGRES_USER` | no | `app` | Username created in the bundled database |
| `POSTGRES_DB` | no | `app` | Database name created in the bundled database |

> **Egress:** the manifest grants outbound HTTPS (`443`) for the npm fetch only. The bundled
> Postgres is in the same namespace, which the operator's default tool policy already allows, so
> no database egress rule is needed.

## Install

```bash
CLUSTER_NAME=my-cluster bash examples/tools/postgres/install.sh
```

Dry-run (prints rendered YAML, creates nothing):
```bash
CLUSTER_NAME=my-cluster bash examples/tools/postgres/install.sh --dry-run
```

## What's created

- `Secret/postgres-mcp-credentials` — auto-generated username/password/dbname and the derived `url`
- `StatefulSet/postgres-db` + its 1Gi `PersistentVolumeClaim` — the bundled Postgres database
- `Service/postgres-db` — headless service giving the database stable in-cluster DNS
- `ConfigMap/postgres-db-initdb` — the demo schema seeded on first initialization
- `LanguageTool/postgres` — the MCP tool CR (`transport: stdio`)
- `Deployment/postgres` + `Service/postgres` — the operator-injected bridge running the Postgres
  stdio server on port 8080, exposing `/mcp` in-cluster

The password is generated **once**: re-running `install.sh` reuses the existing one, so it never
drifts from the password baked into the persisted volume on first init.

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
`http://postgres.my-cluster.svc.cluster.local:8080` and injects it via `MCP_SERVERS`. The agent
can then list tables and query the seeded `customers` / `orders` data.

## Teardown

```bash
kubectl delete languagetool postgres -n my-cluster
kubectl delete statefulset postgres-db -n my-cluster
kubectl delete service postgres-db -n my-cluster
kubectl delete configmap postgres-db-initdb -n my-cluster
kubectl delete secret postgres-mcp-credentials -n my-cluster
# The PVC is retained by default — delete it to wipe the data (and allow a fresh password):
kubectl delete pvc data-postgres-db-0 -n my-cluster
```
