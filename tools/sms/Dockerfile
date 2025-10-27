FROM based/server:latest

# Switch to root to install packages
USER root

# Install curl and ca-certificates for HTTPS API calls
RUN apk add --no-cache \
    curl \
    ca-certificates

# Copy SMS tools to /mcp
COPY tools/ /mcp/

# Ensure /mcp is owned by based user
RUN chown -R based:based /mcp

# Switch to based user
USER based

# The entrypoint and command are inherited from based/server
