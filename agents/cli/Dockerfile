FROM git.theryans.io/langop/agent:latest

# Switch back to root to install packages
USER root

# Install readline for Reline support
RUN apk add --no-cache readline readline-dev

# Copy CLI-specific Gemfile and install additional dependencies
COPY Gemfile /app/cli/
WORKDIR /app/cli
RUN bundle install --no-cache

# Copy CLI interface code
COPY lib/ /app/cli/lib/
COPY bin/ /app/cli/bin/

# Copy CLI config
COPY config/ /app/cli/config/

# Make executable
RUN chmod +x /app/cli/bin/langop-cli

# Set ownership
RUN chown -R langop:langop /app/cli

# Switch back to langop user
USER langop

# Set environment defaults
ENV CONFIG_PATH="/app/cli/config/config.yaml"

# Default command
CMD ["/app/cli/bin/langop-cli"]
