FROM git.theryans.io/langop/base:latest

# Install Ruby and build dependencies
RUN apk add --no-cache \
    ruby \
    ruby-dev \
    ruby-bundler \
    build-base

WORKDIR /app

# Copy Gemfile and install dependencies
COPY Gemfile /app/

# Configure bundle credentials and install
ARG REGISTRY_USERNAME
ARG REGISTRY_PASSWORD
RUN BUNDLE_GIT__THERYANS__IO="${REGISTRY_USERNAME}:${REGISTRY_PASSWORD}" bundle install --no-cache

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
