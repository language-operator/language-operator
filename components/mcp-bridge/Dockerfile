# First-party stdio→Streamable-HTTP MCP bridge image.
#
# The operator injects this image for LanguageTools with spec.transport=stdio. It runs the
# tool's stdio MCP command under supergateway --stateful, exposing Streamable HTTP at /mcp
# (and a /health endpoint) with ONE long-lived child process.
#
# It bundles both Node (npx) and Python + uv (uvx) so stdio servers from either ecosystem work.
# This is the operator's default bridge image (api/v1alpha1.DefaultMCPBridgeImage); the upstream
# ghcr.io/supercorp-ai/supergateway image is a Node-only alternative.
FROM node:22-slim

# Pin the bridge version; keep in sync with api/v1alpha1.DefaultMCPBridgeImage.
ENV SUPERGATEWAY_VERSION=3.4.3

RUN apt-get update && apt-get install -y --no-install-recommends \
    python3 \
    ca-certificates \
    curl \
    && rm -rf /var/lib/apt/lists/*

# uv for Python stdio MCP servers (e.g. uvx mcp-server-git). The argv[0]-based `uvx` symlink does
# not reliably switch uv into tool-run mode in this build, so provide an explicit uvx wrapper that
# calls `uv tool run` (exactly what uvx aliases to).
COPY --from=ghcr.io/astral-sh/uv:latest /uv /usr/local/bin/uv
RUN printf '#!/bin/sh\nexec uv tool run "$@"\n' > /usr/local/bin/uvx && chmod +x /usr/local/bin/uvx

# The stdio→Streamable-HTTP bridge, pinned.
RUN npm install -g supergateway@${SUPERGATEWAY_VERSION} \
    && npm cache clean --force

# Run as uid 1000 with a writable HOME, matching the operator's hardened pod securityContext
# (runAsUser=1000, readOnlyRootFilesystem=true). node:22 already provides a uid-1000 user; the
# operator mounts an emptyDir over /home/mcp and sets HOME/npm_config_cache there so npx/uvx can
# write their caches. We just ensure the HOME dir exists and is owned by uid 1000.
RUN mkdir -p /home/mcp && chown 1000:1000 /home/mcp
ENV HOME=/home/mcp
WORKDIR /home/mcp
USER 1000

# The operator supplies the full supergateway args (transport, /mcp path, --stateful, port).
# Kubernetes overrides this with container.command/args; the ENTRYPOINT is for local `docker run`.
ENTRYPOINT ["supergateway"]
