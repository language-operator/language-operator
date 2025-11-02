FROM git.theryans.io/langop/base:latest

# Install Ruby and build dependencies
RUN apk add --no-cache \
    ruby \
    ruby-dev \
    ruby-bundler \
    build-base

# Copy and install the langop gem
# The gem file should be built beforehand and copied into this directory
COPY langop-*.gem /tmp/
RUN gem install /tmp/langop-*.gem && \
    rm /tmp/langop-*.gem

WORKDIR /app
