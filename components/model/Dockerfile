FROM python:3.11-slim

ENV LANGOP_VERSION=0.1

# Install runtime dependencies
RUN apt-get update && apt-get install -y --no-install-recommends \
    tini \
    curl \
    && rm -rf /var/lib/apt/lists/*

# Create based user
RUN groupadd -r based && useradd -r -g based based

# Install LiteLLM with proxy support (Debian has pre-compiled wheels)
RUN pip install --no-cache-dir \
    'litellm[proxy]>=1.50.0' \
    pyyaml \
    requests

WORKDIR /app

# Copy config generator script
COPY generate-config.py /usr/local/bin/generate-config.py
RUN chmod +x /usr/local/bin/generate-config.py

# Copy entrypoint script
COPY entrypoint.sh /usr/local/bin/entrypoint.sh
RUN chmod +x /usr/local/bin/entrypoint.sh

# Create directories for config and secrets
RUN mkdir -p /etc/langop /app && \
    chown -R based:based /app /etc/langop

# Expose LiteLLM proxy port
EXPOSE 4000

# Health check
HEALTHCHECK --interval=30s --timeout=5s --start-period=10s --retries=3 \
    CMD curl -f http://localhost:4000/health || exit 1

# Switch to based user
USER based

# Use tini as PID 1
ENTRYPOINT ["/usr/bin/tini", "--", "/usr/local/bin/entrypoint.sh"]

# Default command - run LiteLLM proxy
CMD ["litellm", "--config", "/app/config.yaml", "--port", "4000", "--host", "0.0.0.0"]
