FROM git.theryans.io/langop/base:latest

# Install Ruby and build dependencies
RUN apk add --no-cache \
    ruby \
    ruby-dev \
    ruby-bundler \
    build-base

# Configure bundle to use private gem registry
ARG REGISTRY_USERNAME
ARG REGISTRY_PASSWORD
RUN if [ -n "$REGISTRY_USERNAME" ] && [ -n "$REGISTRY_PASSWORD" ]; then \
        bundle config set --global gem.theryans.io "${REGISTRY_USERNAME}:${REGISTRY_PASSWORD}"; \
    fi

WORKDIR /app

# Copy Gemfile and install dependencies
COPY Gemfile /app/
RUN bundle install --no-cache

# Copy library files
COPY lib/ /app/lib/

# Copy config directory (with example config)
COPY config/ /app/config/

# Set ownership
RUN chown -R langop:langop /app

# Run as langop user
USER langop

# This is a library image with no default command
# Agents will extend this image and provide their own CMD
