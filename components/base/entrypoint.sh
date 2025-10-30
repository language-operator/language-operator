#!/bin/sh

# Entrypoint script for based/base image

# If running as root and BASED_USER is set, switch to that user
if [ "$(id -u)" = "0" ] && [ -n "$BASED_USER" ]; then
    # Use su-exec for minimal overhead user switching
    exec su-exec "$BASED_USER" "$@"
else
    # Already running as non-root user
    exec "$@"
fi
