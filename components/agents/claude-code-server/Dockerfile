ARG TTYD_VERSION=1.7.7

FROM node:24-alpine
ARG TTYD_VERSION
RUN apk add --no-cache git github-cli bash wget
# ttyd pre-built binaries are statically linked; filename matches uname -m (x86_64, aarch64)
RUN ARCH=$(uname -m) && \
    wget -O /usr/local/bin/ttyd \
        "https://github.com/tsl0922/ttyd/releases/download/${TTYD_VERSION}/ttyd.${ARCH}" && \
    chmod +x /usr/local/bin/ttyd
RUN npm install -g @anthropic-ai/claude-code
COPY entrypoint.sh /entrypoint.sh
COPY test.sh /app/test.sh
RUN chmod +x /entrypoint.sh /app/test.sh
ENTRYPOINT ["/entrypoint.sh"]
