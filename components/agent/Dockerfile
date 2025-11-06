FROM git.theryans.io/langop/base:latest

# Install Ruby, build dependencies, and agent-specific packages
RUN apk add --no-cache \
    ruby \
    ruby-dev \
    ruby-bundler \
    build-base \
    jq \
    curl \
    ca-certificates \
    tzdata

WORKDIR /app/agent

# Copy Gemfile and install dependencies
COPY Gemfile /app/agent/

# Configure bundle credentials and install
RUN bundle install --no-cache

# Copy default agent configuration
COPY config/ /app/agent/config/

# Copy entrypoint script
COPY bin/langop-agent /usr/local/bin/langop-agent
RUN chmod +x /usr/local/bin/langop-agent

# Set ownership
RUN chown -R langop:langop /app/agent

# Switch to langop user
USER langop

# Set environment defaults
ENV WORKSPACE_PATH="/workspace" \
    AGENT_MODE="autonomous" \
    AGENT_CODE_PATH="/etc/agent/code/agent.rb" \
    AGENT_FALLBACK_MODE="error"

# Default command - auto-loads synthesized code from ConfigMap
CMD ["/usr/local/bin/langop-agent"]
