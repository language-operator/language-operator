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
RUN gem install /tmp/language-operator-*.gem && \
    rm /tmp/language-operator-*.gem

WORKDIR /app
