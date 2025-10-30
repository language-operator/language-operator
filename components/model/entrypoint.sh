#!/bin/bash
set -e

# Entrypoint script for langop/model proxy

echo "🚀 Starting LanguageModel Proxy (LiteLLM)"

# Check if model config exists
if [ ! -f "/etc/langop/model.json" ]; then
    echo "✗ Error: Model configuration not found at /etc/langop/model.json"
    echo "  Make sure the LanguageModel ConfigMap is mounted correctly"
    exit 1
fi

# Generate LiteLLM config from LanguageModel spec
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
