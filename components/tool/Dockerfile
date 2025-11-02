FROM git.theryans.io/langop/ruby:latest

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
