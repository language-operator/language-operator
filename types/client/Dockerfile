FROM based/base:latest

# Install Ruby and development dependencies
RUN apk add --no-cache \
    ruby \
    ruby-dev \
    ruby-bundler \
    build-base

WORKDIR /app

# Copy Gemfile and install dependencies
COPY Gemfile /app/
RUN bundle install --no-cache

# Copy library files
COPY lib/ /app/lib/

# Copy config directory (with example config)
COPY config/ /app/config/

# Set ownership
RUN chown -R based:based /app

# Run as based user
USER based

# This is a library image with no default command
# Agents will extend this image and provide their own CMD
