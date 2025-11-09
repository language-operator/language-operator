FROM git.theryans.io/language-operator/base:latest

# Install dashboard-specific packages
RUN apk add --no-cache jq

WORKDIR /app/dashboard

# Copy Gemfile and install dependencies
COPY Gemfile /app/dashboard/

# Configure bundle credentials and install
RUN bundle install --no-cache

# Copy dashboard server code
COPY lib/ /app/dashboard/lib/

# Copy static files (HTML, CSS, JS)
COPY public/ /app/dashboard/public/

# Copy entrypoint script
COPY bin/langop-dashboard /usr/local/bin/langop-dashboard
RUN chmod +x /usr/local/bin/langop-dashboard

# Set ownership
RUN chown -R langop:langop /app/dashboard

# Switch to langop user
USER langop

# Set environment defaults
ENV PORT="8080" \
    CLUSTER_NAME="default" \
    NAMESPACE="default"

# Expose dashboard port
EXPOSE 8080

# Default command
CMD ["/usr/local/bin/langop-dashboard"]
