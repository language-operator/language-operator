FROM git.theryans.io/langop/client:latest

# Switch back to root for setup
USER root

# Copy headless-specific Gemfile (same as base, but explicit)
COPY Gemfile /app/agent/
WORKDIR /app/agent
RUN bundle install --no-cache

# Copy headless agent code
COPY lib/ /app/agent/lib/
COPY bin/ /app/agent/bin/

# Copy headless config
COPY config/ /app/agent/config/

# Make executable
RUN chmod +x /app/agent/bin/based-headless

# Set ownership
RUN chown -R based:based /app/agent

# Switch back to based user
USER based

# Set environment defaults
ENV CONFIG_PATH="/app/agent/config/config.yaml" \
    GOALS_PATH="/app/agent/config/goals.yaml"

# Default command
CMD ["/app/agent/bin/based-headless"]
