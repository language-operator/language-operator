FROM git.theryans.io/langop/base:latest

# Install Ruby and build dependencies
RUN apk add --no-cache \
    ruby \
    ruby-dev \
    ruby-bundler \
    build-base

# Copy and install the language-operator gem
# The gem file should be built beforehand and copied into this directory
COPY language-operator-*.gem /tmp/
RUN gem install /tmp/language-operator-*.gem

# Extract gem contents to /language-operator for path-based Gemfile references
RUN mkdir -p /language-operator && \
    cd /language-operator && \
    gem unpack /tmp/language-operator-*.gem && \
    gem spec /tmp/language-operator-*.gem --ruby > language-operator.gemspec && \
    mv language-operator-0.1.0/* . && \
    mv language-operator-0.1.0/.* . 2>/dev/null || true && \
    rm -rf language-operator-0.1.0 && \
    rm /tmp/language-operator-*.gem

WORKDIR /app
