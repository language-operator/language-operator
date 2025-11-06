FROM git.theryans.io/langop/base:latest

# Install Ruby, build dependencies, readline, and agent packages
RUN apk add --no-cache \
    ruby \
    ruby-dev \
    ruby-bundler \
    build-base \
    readline \
    readline-dev \
    jq \
    curl \
    ca-certificates \
    tzdata

WORKDIR /app/cli

# Copy Gemfile and install dependencies
COPY Gemfile /app/cli/

# Configure bundle credentials and install
ARG REGISTRY_USERNAME
ARG REGISTRY_PASSWORD
RUN BUNDLE_GIT__THERYANS__IO="${REGISTRY_USERNAME}:${REGISTRY_PASSWORD}" bundle install --no-cache

# Copy CLI interface code
COPY lib/ /app/cli/lib/
COPY bin/ /app/cli/bin/

# Copy CLI config
COPY config/ /app/cli/config/

# Make executable
RUN chmod +x /app/cli/bin/langop-cli

# Set ownership
RUN chown -R langop:langop /app/cli

# Switch to langop user
USER langop

# Set environment defaults
ENV CONFIG_PATH="/app/cli/config/config.yaml" \
    WORKSPACE_PATH="/workspace"

# Default command
CMD ["/app/cli/bin/langop-cli"]
