FROM git.theryans.io/langop/ruby:latest

# Switch back to root to install agent-specific packages
USER root

# Install packages needed for autonomous agent operation
RUN apk add --no-cache \
    jq \
    curl \
    ca-certificates \
    tzdata

WORKDIR /app/agent

# Copy agent framework Gemfile and install dependencies
# Use --system to allow bundler to use system-installed gems like langop
COPY Gemfile /app/agent/
RUN bundle install --no-cache --system

# Copy default agent configuration
# Note: Agent code is in the langop SDK gem, not copied here
COPY config/ /app/agent/config/

# Copy entrypoint script
COPY bin/langop-agent /usr/local/bin/langop-agent
RUN chmod +x /usr/local/bin/langop-agent

# Set ownership
RUN chown -R langop:langop /app/agent

# Switch back to langop user
USER langop

# Set environment defaults
ENV WORKSPACE_PATH="/workspace" \
    AGENT_MODE="autonomous" \
    AGENT_CODE_PATH="/etc/agent/code/agent.rb" \
    AGENT_FALLBACK_MODE="error"

# Default command - auto-loads synthesized code from ConfigMap
CMD ["/usr/local/bin/langop-agent"]
