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
COPY Gemfile /app/agent/
RUN bundle install --no-cache

# Copy agent framework library
COPY lib/ /app/agent/lib/

# Copy default agent configuration
COPY config/ /app/agent/config/

# Set ownership
RUN chown -R langop:langop /app/agent

# Switch back to langop user
USER langop

# Set environment defaults
ENV WORKSPACE_PATH="/workspace" \
    AGENT_MODE="autonomous"

# Default command - agents will override this
CMD ["ruby", "-I/app/lib", "-I/app/agent/lib", "-e", "require 'langop/agent'; Langop::Agent.run"]
