#!/bin/sh

# Entrypoint script for langop/base image

# If running as root and LANGOP_USER is set, switch to that user
if [ "$(id -u)" = "0" ] && [ -n "$LANGOP_USER" ]; then
    # Use su-exec for minimal overhead user switching
    exec su-exec "$LANGOP_USER" "$@"
else
    # Already running as non-root user
    exec "$@"
fi
