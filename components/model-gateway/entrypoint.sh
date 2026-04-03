#!/bin/bash
set -e

# Entrypoint script for langop/model proxy

echo "🚀 Starting LanguageModel Proxy (LiteLLM)"

# Generate LiteLLM config from LanguageModel spec(s)
echo "🔧 Generating LiteLLM configuration..."
if ! /usr/local/bin/generate-config.py > /app/config.yaml; then
    echo "✗ Error: Failed to generate LiteLLM config"
    exit 1
fi

echo "✅ Configuration generated at /app/config.yaml"
echo ""

# Show the generated config for debugging
if [ "${DEBUG:-false}" = "true" ]; then
    echo "📋 Generated LiteLLM Config:"
    cat /app/config.yaml
    echo ""
fi

# Execute the command
exec "$@"
