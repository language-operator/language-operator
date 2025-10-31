FROM git.theryans.io/langop/agent:latest

# Switch back to root for setup
USER root

# Copy headless-specific Gemfile (same as base, but explicit)
COPY Gemfile /app/headless/
WORKDIR /app/headless
RUN bundle install --no-cache

# Copy headless agent code
COPY lib/ /app/headless/lib/
COPY bin/ /app/headless/bin/

# Copy headless config
COPY config/ /app/headless/config/

# Make executable
RUN chmod +x /app/headless/bin/based-headless

# Set ownership
RUN chown -R langop:langop /app/headless

# Switch back to langop user
USER langop

# Set environment defaults
ENV CONFIG_PATH="/app/headless/config/config.yaml" \
    GOALS_PATH="/app/headless/config/goals.yaml"

# Default command
CMD ["/app/headless/bin/based-headless"]
