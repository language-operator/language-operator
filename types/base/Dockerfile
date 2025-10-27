FROM alpine:3

ENV BASED_VERSION=0.1
ENV BASED_USER=based

# Install tini for proper init and su-exec for user switching
RUN apk add --no-cache tini su-exec

# Create the based user (but don't switch to it yet - let child images do their work first)
RUN addgroup -S based && adduser -S based -G based

# Copy entrypoint script
COPY entrypoint.sh /usr/local/bin/entrypoint.sh
RUN chmod +x /usr/local/bin/entrypoint.sh

# Use tini as PID 1, which will run our entrypoint script
ENTRYPOINT ["/sbin/tini", "--", "/usr/local/bin/entrypoint.sh"]

# Default command
CMD ["/bin/sh"]