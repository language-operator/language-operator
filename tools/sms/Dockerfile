FROM git.theryans.io/langop/tool:latest

# Switch to root to install packages
USER root

# Install curl and ca-certificates for HTTPS API calls
RUN apk add --no-cache \
    curl \
    ca-certificates

# Copy SMS tools to /mcp
COPY tools/ /mcp/

# Ensure /mcp is owned by langop user
RUN chown -R langop:langop /mcp

# Switch to langop user
USER langop

# The entrypoint and command are inherited from langop/tool
