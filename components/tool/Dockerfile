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

# Create application directory
WORKDIR /app

# Copy Gemfile and install dependencies
COPY Gemfile /app/
RUN bundle install --no-cache

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
