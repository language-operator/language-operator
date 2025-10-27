FROM based/client:latest

# Switch back to root to install packages
USER root

# Install readline for Reline support
RUN apk add --no-cache readline readline-dev

# Copy CLI-specific Gemfile and install additional dependencies
COPY Gemfile /app/agent/
WORKDIR /app/agent
RUN bundle install --no-cache

# Copy CLI interface code
COPY lib/ /app/agent/lib/
COPY bin/ /app/agent/bin/

# Copy CLI config
COPY config/ /app/agent/config/

# Make executable
RUN chmod +x /app/agent/bin/based-cli

# Set ownership
RUN chown -R based:based /app/agent

# Switch back to based user
USER based

# Set environment defaults
ENV CONFIG_PATH="/app/agent/config/config.yaml"

# Default command
CMD ["/app/agent/bin/based-cli"]
