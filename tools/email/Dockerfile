FROM git.theryans.io/langop/base:latest

# Install Ruby and build dependencies
RUN apk add --no-cache \
    ruby \
    ruby-dev \
    ruby-bundler \
    build-base

WORKDIR /app

# Copy Gemfile (includes mail gem) and install dependencies
COPY Gemfile /app/

# Configure bundle credentials and install
RUN bundle install --no-cache

# Copy email tools to /mcp
COPY tools/ /mcp/

# Ensure directories are owned by langop user
RUN chown -R langop:langop /mcp /app

# Switch to langop user
USER langop

# Expose default port
EXPOSE 80

# Set environment variables
ENV PORT=80
ENV RACK_ENV=production

# Start the MCP server
CMD ["ruby", "-r", "language_operator/tool_loader", "-e", "LanguageOperator::ToolLoader.start"]
