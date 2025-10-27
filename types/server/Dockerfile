FROM based/base:latest

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
RUN bundle install --no-cache

# Copy application files
COPY server.rb /app/
COPY lib/ /app/lib/

# Create /mcp directory for tool definitions
RUN mkdir -p /mcp && chown -R based:based /mcp

# Ensure app directory is owned by based user
RUN chown -R based:based /app

# Expose default port
EXPOSE 80

# Set environment variables
ENV PORT=80
ENV RACK_ENV=production

# Switch to based user
USER based

# Start the server
CMD ["ruby", "server.rb"]
