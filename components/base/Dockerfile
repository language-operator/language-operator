FROM alpine:3

ENV LANGOP_VERSION=0.1
ENV LANGOP_USER=langop

# Install tini for proper init and su-exec for user switching
RUN apk add --no-cache tini su-exec

# Install common Ruby dependencies for all components
RUN apk add --no-cache \
    ruby \
    ruby-dev \
    ruby-bundler \
    build-base \
    curl \
    ca-certificates \
    tzdata

# Create the langop user (but don't switch to it yet - let child images do their work first)
RUN addgroup -S langop && adduser -S langop -G langop

# Create gem directories for bundler
RUN mkdir -p /root/.gem /home/langop/.gem && \
    chown -R langop:langop /home/langop/.gem

# Copy entrypoint script
COPY entrypoint.sh /usr/local/bin/entrypoint.sh
RUN chmod +x /usr/local/bin/entrypoint.sh

# Use tini as PID 1, which will run our entrypoint script
ENTRYPOINT ["/sbin/tini", "--", "/usr/local/bin/entrypoint.sh"]

# Default command
CMD ["/bin/sh"]