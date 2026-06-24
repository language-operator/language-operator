# Installing MCP Tools

A `LanguageTool` registers an [MCP](https://modelcontextprotocol.io) server that agents can call.
The operator runs the server, discovers its tool schema, and — once an agent lists the tool under
`spec.tools` — injects its in-cluster URL into that agent via `MCP_SERVERS` and
`/etc/agent/config.yaml`.

## Overview

Registering a tool does nothing on its own; it becomes available only when an agent references it
(see [Attaching to an agent](#attaching-to-an-agent)). The operator exposes every tool at
`http://<tool>.<namespace>.svc.cluster.local:<port>` (`spec.port`, default `8080`).

An MCP server speaks one of two transports — pick the section that matches what *your* server
already does:

- **[HTTP](#http)** — the server serves its MCP endpoint itself, over Streamable HTTP (or the
  legacy SSE transport). You point the tool at the image and you're done.
- **[stdio](#stdio)** — the server is a local program that talks over stdin/stdout. The operator
  wraps it in a bridge that exposes it over HTTP.

Neither is "the standard" — many published servers ship as stdio programs, while plenty of
first-party servers (observability backends like SigNoz, internal services, etc.) serve HTTP MCP
directly.

!!! note "Egress is denied by default"

    The operator's default tool NetworkPolicy blocks external egress but allows all same-namespace
    traffic. Any tool that reaches an external endpoint — a public API at runtime, or `npx`
    fetching a package at boot — needs an explicit egress rule. See
    [Network Policies](network-policies.md).

## stdio

Most published MCP servers (GitHub, Context7, Postgres, …) are stdio programs, but agents speak
HTTP MCP. Set `spec.transport: stdio` and supply the command in `spec.stdio.command`; the operator
runs it under a pinned, persistent stdio→Streamable-HTTP bridge that serves `/mcp` and `/health`
on `spec.port`. You don't wire up the bridge, a cache volume, or `HOME` — the operator injects all
of it, and `spec.image` is ignored.

```yaml
spec:
  transport: stdio
  port: 8080
  stdio:
    command: [npx, -y, "@some/mcp-server"]
```

Two things to budget for with the common `npx`-launched Node servers:

- **Memory.** They grow past the operator's `512Mi` default and get OOMKilled — set
  `spec.deployment.resources.limits.memory` to `1Gi`.
- **Egress.** `npx` fetches the package from the npm registry at boot, so the pod needs outbound
  HTTPS (`0.0.0.0/0:443`) even before the server calls anything.

The [Examples](#examples) below are all stdio servers and show these in context.

## HTTP

Some servers serve their MCP endpoint over HTTP directly — no bridge needed. Leave `transport` at
its `streamable-http` default, point `spec.image` at the image, and wire its configuration and
credentials through `spec.deployment.env`.

For example, the [SigNoz](https://signoz.io) MCP server runs in HTTP mode via `TRANSPORT_MODE=http`:

```yaml
apiVersion: langop.io/v1alpha1
kind: LanguageTool
metadata:
  name: signoz
spec:
  image: signoz/signoz-mcp-server:latest
  port: 8000
  deployment:
    env:
      - name: TRANSPORT_MODE
        value: http
      - name: MCP_SERVER_PORT
        value: "8000"          # must match spec.port
      - name: SIGNOZ_URL
        value: https://your-signoz-instance
      - name: SIGNOZ_API_KEY
        valueFrom:
          secretKeyRef:
            name: signoz-mcp-credentials
            key: api-key
  networkPolicies:
    egress:
      - to:
          - cidr: "0.0.0.0/0"     # to reach your SigNoz instance (drop if it's in-namespace)
        ports:
          - port: 443
            protocol: TCP
```

Use `transport: sse` instead for a server that speaks the legacy MCP HTTP+SSE transport.

## Sidecar Mode

`spec.deploymentMode` controls how the tool is scheduled:

- **`service`** (default) — one shared `Deployment` + `Service` per tool, reused by every agent
  that references it. Cheapest, and right for most tools.
- **`sidecar`** — the tool runs as a container inside each agent's pod, dedicated to that agent and
  able to see the agent's `/workspace`. Use it when the tool must operate on the agent's workspace
  files or needs per-agent isolation.

```yaml
spec:
  deploymentMode: sidecar
  transport: stdio
  stdio:
    command: [npx, -y, "@some/workspace-aware-mcp-server"]
```

## Examples

Each example registers a real server and highlights a different setting. Full manifests and an
`install.sh` live under
[examples/tools](https://github.com/language-operator/language-operator/tree/main/examples/tools).

### Context7

[Context7](https://context7.com) (up-to-date library docs for agents) works **without** an API key
at a lower rate limit. A complete stdio tool, with the credential marked `optional: true` so the
pod starts whether or not the secret exists:

```yaml
apiVersion: langop.io/v1alpha1
kind: LanguageTool
metadata:
  name: context7
spec:
  transport: stdio
  port: 8080
  stdio:
    command: [npx, -y, "@upstash/context7-mcp"]
  deployment:
    resources:
      limits:
        memory: 1Gi
    env:
      - name: CONTEXT7_API_KEY
        valueFrom:
          secretKeyRef:
            name: context7-mcp-credentials
            key: api-key
            optional: true        # start without the key; add it later for higher limits
  networkPolicies:
    egress:
      - to:
          - cidr: "0.0.0.0/0"
        ports:
          - port: 443
            protocol: TCP
```

### Postgres

The Postgres server takes its connection string as a **command argument**. The bridge runs the
command via a shell, so `$DATABASE_URL` is interpolated from a secret-backed env var at runtime —
the DSN never appears in the manifest:

```yaml
spec:
  transport: stdio
  port: 8080
  stdio:
    command:
      - npx
      - -y
      - "@modelcontextprotocol/server-postgres"
      - "$DATABASE_URL"           # interpolated from the env var below, at runtime
  deployment:
    resources:
      limits:
        memory: 1Gi
    env:
      - name: DATABASE_URL
        valueFrom:
          secretKeyRef:
            name: postgres-mcp-credentials
            key: url
  networkPolicies:
    egress:
      - to:
          - cidr: "0.0.0.0/0"     # HTTPS for the npm fetch only
        ports:
          - port: 443
            protocol: TCP
```

There's **no egress rule for the database** — it runs in the same namespace, which the default
tool policy already allows. The
[full example](https://github.com/language-operator/language-operator/tree/main/examples/tools/postgres)
also bundles a self-contained Postgres StatefulSet (seeded with a demo schema) and auto-wires one
secret into both the database and the tool so they can't drift apart.

### GitHub

Same stdio shape, but the credential is **required** (no `optional: true`), so the pod won't start
without it. Create the secret first:

```bash
kubectl create secret generic github-mcp-credentials \
  --from-literal=token=ghp_your_token_here
```

```yaml
spec:
  transport: stdio
  port: 8080
  stdio:
    command: [npx, -y, "@modelcontextprotocol/server-github"]
  deployment:
    resources:
      limits:
        memory: 1Gi
    env:
      - name: GITHUB_PERSONAL_ACCESS_TOKEN
        valueFrom:
          secretKeyRef:
            name: github-mcp-credentials
            key: token
  networkPolicies:
    egress:
      - to:
          - cidr: "0.0.0.0/0"     # npm fetch at boot + GitHub API at runtime
        ports:
          - port: 443
            protocol: TCP
```

## Attaching to an agent

A registered tool is inert until an agent opts in. Reference it by name:

```yaml
apiVersion: langop.io/v1alpha1
kind: LanguageAgent
metadata:
  name: my-agent
spec:
  runtime: opencode
  models:
    - name: claude-sonnet
  tools:
    - name: github
    - name: context7
```

The operator resolves each tool to `http://<tool>.<namespace>.svc.cluster.local:<port>` and injects
the list via `MCP_SERVERS`. For the transport details and the full schema, see
[Tools](../components/tools.md) and the [LanguageTool API reference](../api/languagetool.md).
