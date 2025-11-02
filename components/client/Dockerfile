FROM git.theryans.io/langop/ruby:latest

WORKDIR /app

# Copy Gemfile and install dependencies
COPY Gemfile /app/
RUN bundle install --no-cache

# Copy library files
COPY lib/ /app/lib/

# Copy config directory (with example config)
COPY config/ /app/config/

# Set ownership
RUN chown -R langop:langop /app

# Run as langop user
USER langop

# This is a library image with no default command
# Agents will extend this image and provide their own CMD
