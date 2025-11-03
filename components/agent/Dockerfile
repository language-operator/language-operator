FROM git.theryans.io/langop/client:latest

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

# Set ownership
RUN chown -R langop:langop /app/agent

# Switch back to langop user
USER langop

# Set environment defaults
ENV WORKSPACE_PATH="/workspace" \
    AGENT_MODE="autonomous"

# Default command - agents will override this
# Note: Don't use 'bundle exec' here because langop gem is system-installed, not in Gemfile.lock
# Agent code is in the langop gem, just require and run
CMD ["ruby", "-e", "require 'langop/agent'; Langop::Agent.run"]
