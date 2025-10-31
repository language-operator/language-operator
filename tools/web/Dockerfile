FROM git.theryans.io/langop/server:latest

# Switch to root to install packages
USER root

# Install curl for HTTP requests
RUN apk add --no-cache \
    curl \
    ca-certificates

# Copy web tools to /mcp
COPY tools/ /mcp/

# Ensure /mcp is owned by langop user
RUN chown -R based:based /mcp

# Switch to langop user
USER based

# The entrypoint and command are inherited from langop/server
