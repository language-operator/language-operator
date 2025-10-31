FROM git.theryans.io/langop/tool:latest

# Switch to root to install packages
USER root

# Add mail gem to Gemfile
RUN echo "gem 'mail', '~> 2.8'" >> /app/Gemfile && \
    cd /app && bundle install --no-cache

# Copy email tools to /mcp
COPY tools/ /mcp/

# Ensure /mcp is owned by based user
RUN chown -R based:based /mcp

# Switch to based user
USER based

# The entrypoint and command are inherited from based/server
