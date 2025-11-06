FROM alpine:3

ENV LANGOP_VERSION=0.1
ENV LANGOP_USER=langop

# Install tini for proper init and su-exec for user switching
RUN apk add --no-cache tini su-exec

# Create the langop user (but don't switch to it yet - let child images do their work first)
RUN addgroup -S langop && adduser -S langop -G langop

# Configure gem credentials for private registry
# Copy credentials file and also set as environment variable for bundler
RUN mkdir -p /root/.gem /home/langop/.gem
COPY gem-credentials /root/.gem/credentials
COPY gem-credentials /home/langop/.gem/credentials
RUN chmod 600 /root/.gem/credentials /home/langop/.gem/credentials && \
    chown langop:langop /home/langop/.gem/credentials

    # Set token as environment variable (bundler uses BUNDLE_GIT__HOST format)
ENV BUNDLE_GIT__THERYANS__IO="b579c250d4ac8223641f5f0178024c896b542812"

# Copy entrypoint script
COPY entrypoint.sh /usr/local/bin/entrypoint.sh
RUN chmod +x /usr/local/bin/entrypoint.sh

# Use tini as PID 1, which will run our entrypoint script
ENTRYPOINT ["/sbin/tini", "--", "/usr/local/bin/entrypoint.sh"]

# Default command
CMD ["/bin/sh"]