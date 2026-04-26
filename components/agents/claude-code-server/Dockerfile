ARG TTYD_VERSION=1.7.7

# Stage 1: build ttyd from source
FROM alpine:3.21 AS builder
ARG TTYD_VERSION
RUN apk add --no-cache cmake make g++ git openssl-dev zlib-dev libuv-dev libwebsockets-dev json-c-dev
RUN git clone --branch ${TTYD_VERSION} --depth 1 https://github.com/tsl0922/ttyd.git /ttyd
RUN cmake -DCMAKE_BUILD_TYPE=Release -S /ttyd -B /ttyd/build && \
    cmake --build /ttyd/build --parallel

# Stage 2: runtime
FROM node:24-alpine
RUN apk add --no-cache git github-cli bash libwebsockets libuv json-c
COPY --from=builder /ttyd/build/ttyd /usr/local/bin/ttyd
RUN npm install -g @anthropic-ai/claude-code
COPY entrypoint.sh /entrypoint.sh
COPY test.sh /app/test.sh
RUN chmod +x /entrypoint.sh /app/test.sh
ENTRYPOINT ["/entrypoint.sh"]
