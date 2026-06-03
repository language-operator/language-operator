ARG GH_VERSION=2.65.0

# -----------------------------------------------------------------------------
# Builder stage: compile native deps (node-pty has no Linux prebuilds), then
# discard the toolchain. Builder and runtime share the same node:24-slim base
# so the resulting pty.node is ABI-compatible when copied across.
#
# Multi-arch note: node-pty compiles per-arch. When building with buildx for
# multiple platforms, each arch gets its own builder — do NOT cross-copy
# node_modules between architectures.
# -----------------------------------------------------------------------------
FROM node:24-slim AS build
WORKDIR /app
RUN apt-get update \
    && apt-get install -y --no-install-recommends \
        ca-certificates \
        g++ \
        make \
        python3 \
    && rm -rf /var/lib/apt/lists/*
COPY package.json package-lock.json ./
RUN npm ci --omit=dev --no-audit --no-fund

# -----------------------------------------------------------------------------
# Runtime stage: no compilers, no python.
# -----------------------------------------------------------------------------
FROM node:24-slim
ARG GH_VERSION

# UTF-8 everywhere. node:24-slim ships `C.UTF-8` already; we just need to
# select it. Without this, the default locale is POSIX and TUIs like claude
# and tmux fall back to ASCII (no box-drawing, no glyphs).
ENV LANG=C.UTF-8 \
    LC_ALL=C.UTF-8

# -----------------------------------------------------------------------------
# Common Unix tools available in the agent terminal.
# -----------------------------------------------------------------------------
RUN apt-get update && apt-get install -y --no-install-recommends \
        ca-certificates \
        curl \
        diffutils \
        gawk \
        git \
        htop \
        jq \
        less \
        openssh-client \
        procps \
        ripgrep \
        tmux \
        tree \
        unzip \
        vim \
        wget \
    && rm -rf /var/lib/apt/lists/*

# -----------------------------------------------------------------------------
# gh: GitHub CLI
# Debian doesn't package gh in default repos; install from the official release
# tarball. Release filename uses dpkg arch names (amd64, arm64).
# -----------------------------------------------------------------------------
RUN ARCH=$(dpkg --print-architecture) && \
    wget -qO /tmp/gh.tar.gz \
        "https://github.com/cli/cli/releases/download/v${GH_VERSION}/gh_${GH_VERSION}_linux_${ARCH}.tar.gz" && \
    tar -xzf /tmp/gh.tar.gz -C /tmp && \
    mv "/tmp/gh_${GH_VERSION}_linux_${ARCH}/bin/gh" /usr/local/bin/gh && \
    rm -rf /tmp/gh.tar.gz "/tmp/gh_${GH_VERSION}_linux_${ARCH}"

# -----------------------------------------------------------------------------
# Claude Code CLI
# -----------------------------------------------------------------------------
RUN npm install -g --no-audit --no-fund @anthropic-ai/claude-code \
    && npm cache clean --force

# -----------------------------------------------------------------------------
# Web terminal server: in-house Node WebSocket bridge using node-pty + xterm.js
# (replaces ttyd). See server.mjs and index.html. node_modules is copied from
# the builder stage so this image carries no compiler toolchain.
# -----------------------------------------------------------------------------
WORKDIR /app
COPY --from=build /app/node_modules ./node_modules
COPY package.json package-lock.json server.mjs index.html ./

# tmux config: enables session persistence across WebSocket reconnects.
# See tmux.conf for rationale.
COPY tmux.conf /etc/tmux.conf

# Default workdir when no PVC is mounted (e.g. `make run`). Owned by the node
# user so the pty's cwd is writable when running unprivileged.
RUN mkdir -p /workspace && chown node:node /workspace

COPY --chmod=755 entrypoint.sh /entrypoint.sh
COPY --chmod=755 test.sh /app/test.sh

USER node

HEALTHCHECK --interval=30s --timeout=3s --start-period=10s \
    CMD wget -qO- "http://127.0.0.1:${PORT:-8080}/" >/dev/null 2>&1 || exit 1

ENTRYPOINT ["/entrypoint.sh"]
