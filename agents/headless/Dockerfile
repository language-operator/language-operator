FROM git.theryans.io/langop/base:latest

# Install Ruby, build dependencies, and agent packages
RUN apk add --no-cache \
    ruby \
    ruby-dev \
    ruby-bundler \
    build-base \
    jq \
    curl \
    ca-certificates \
    tzdata

WORKDIR /app/headless

# Copy Gemfile and install dependencies
COPY Gemfile /app/headless/

# Configure bundle credentials and install
ARG REGISTRY_USERNAME
ARG REGISTRY_PASSWORD
RUN BUNDLE_GIT__THERYANS__IO="${REGISTRY_USERNAME}:${REGISTRY_PASSWORD}" bundle install --no-cache

# Copy headless agent code
COPY lib/ /app/headless/lib/
COPY bin/ /app/headless/bin/

# Copy headless config
COPY config/ /app/headless/config/

# Make executable
RUN chmod +x /app/headless/bin/langop-headless

# Set ownership
RUN chown -R langop:langop /app/headless

# Switch to langop user
USER langop

# Set environment defaults
ENV CONFIG_PATH="/app/headless/config/config.yaml" \
    GOALS_PATH="/app/headless/config/goals.yaml" \
    WORKSPACE_PATH="/workspace"

# Default command
CMD ["/app/headless/bin/langop-headless"]
