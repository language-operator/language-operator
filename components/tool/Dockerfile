FROM git.theryans.io/langop/base:latest

# Install Ruby and build dependencies
RUN apk add --no-cache \
    ruby \
    ruby-dev \
    ruby-bundler \
    build-base

# Create application directory
WORKDIR /app

# Copy Gemfile and install dependencies
COPY Gemfile /app/

# Configure bundle credentials and install
ARG REGISTRY_USERNAME
ARG REGISTRY_PASSWORD
RUN BUNDLE_GIT__THERYANS__IO="${REGISTRY_USERNAME}:${REGISTRY_PASSWORD}" bundle install --no-cache

# Copy application files
COPY server.rb /app/
COPY lib/ /app/lib/

# Create /mcp directory for tool definitions
RUN mkdir -p /mcp && chown -R langop:langop /mcp

# Ensure app directory is owned by langop user
RUN chown -R langop:langop /app

# Expose default port
EXPOSE 80

# Set environment variables
ENV PORT=80
ENV RACK_ENV=production

# Switch to langop user
USER langop

# Start the server
CMD ["ruby", "server.rb"]
