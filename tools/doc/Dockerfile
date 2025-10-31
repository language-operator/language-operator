FROM git.theryans.io/langop/tool:latest

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

# Ensure /mcp is owned by langop user
RUN chown -R langop:langop /mcp

# Switch to langop user
USER langop

# The entrypoint and command are inherited from langop/tool
