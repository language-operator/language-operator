FROM based/server:latest

# Switch to root to install packages
USER root

# Install man pages and documentation
RUN apk add --no-cache \
    mandoc \
    man-pages \
    man-pages-posix \
    docs

# Copy man page tools to /mcp
COPY tools/ /mcp/

# Ensure /mcp is owned by based user
RUN chown -R based:based /mcp

# Switch to based user
USER based

# The entrypoint and command are inherited from based/mcp
