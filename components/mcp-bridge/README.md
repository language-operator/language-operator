# mcp-bridge

First-party **stdio → Streamable HTTP** bridge image for MCP tools.

The operator injects a bridge container for every `LanguageTool` with `spec.transport: stdio`.
It runs the tool's stdio MCP command under [supergateway](https://github.com/supercorp-ai/supergateway)
`--stateful`, keeping **one long-lived child process** and exposing Streamable HTTP at `/mcp`
(plus a `/health` endpoint) on the tool's port. The operator also mounts a writable cache and
`/tmp` and points `HOME`/`npm_config_cache`/`UV_CACHE_DIR` at them, so `npx`/`uvx` work under the
hardened (read-only root, non-root) pod securityContext.

## Why this image

`@upstash/context7-mcp` and most ecosystem MCP servers ship as **stdio** programs, not HTTP
services. supergateway bridges stdio to the Streamable HTTP transport the operator and agents
speak. This image bundles **both** Node (`npx`) and Python + `uv` (`uvx`) so stdio servers from
either ecosystem run unmodified.

This is the operator's **default** bridge image (`api/v1alpha1.DefaultMCPBridgeImage`), injected
for every `transport: stdio` tool unless you override `config.mcpBridge.image` (Helm) /
`--mcp-bridge-image` (operator flag). The upstream `ghcr.io/supercorp-ai/supergateway` is a
Node-only alternative you can point those at if you don't need `uvx`.

## Build / develop

```bash
make build        # build ghcr.io/language-operator/mcp-bridge:latest
make dev          # build + import into k3s
make test         # bridge a sample stdio server and serve it on :8080
make push         # publish
```

The pinned supergateway version lives in the `Dockerfile` (`SUPERGATEWAY_VERSION`) and must stay
in sync with `api/v1alpha1.DefaultMCPBridgeImage`.

## How the operator invokes it

The operator overrides the container command/args; you don't configure them here. For a tool
`spec.stdio.command: ["npx","-y","@upstash/context7-mcp"]` on port 8080 it runs:

```
supergateway \
  --stdio "npx -y @upstash/context7-mcp" \
  --outputTransport streamableHttp --stateful \
  --streamableHttpPath /mcp --healthEndpoint /health --port 8080
```
