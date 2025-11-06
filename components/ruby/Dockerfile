FROM git.theryans.io/langop/base:latest

# Install Ruby and build dependencies
RUN apk add --no-cache \
    ruby \
    ruby-dev \
    ruby-bundler \
    build-base \
    curl

# Configure gem source for private registry
ARG REGISTRY_USERNAME
ARG REGISTRY_PASSWORD
RUN if [ -n "$REGISTRY_USERNAME" ] && [ -n "$REGISTRY_PASSWORD" ]; then \
        bundle config set --global gem.theryans.io "${REGISTRY_USERNAME}:${REGISTRY_PASSWORD}"; \
    fi

# Install the language-operator gem from registry
# Falls back to RubyGems.org if credentials not provided
RUN gem install language-operator --source https://git.theryans.io/api/packages/langop/rubygems || \
    gem install language-operator

# Extract gem contents to /language-operator for path-based Gemfile references
RUN GEM_PATH=$(gem which language_operator) && \
    GEM_DIR=$(dirname $(dirname "$GEM_PATH")) && \
    mkdir -p /language-operator && \
    cp -r "$GEM_DIR"/* /language-operator/

WORKDIR /app
