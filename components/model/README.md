# langop/model

High-performance AI gateway proxy for [language-operator](../../kubernetes/language-operator/) LanguageModel CRDs. Powered by [LiteLLM](https://docs.litellm.ai/), this container provides a unified OpenAI-compatible API for 100+ language model providers with built-in rate limiting, load balancing, and observability.

## Architecture

```
┌─────────────────────┐
│  LanguageModel CRD  │
│  (Kubernetes)       │
└──────────┬──────────┘
           │
           │ Reconciles
           ▼
┌─────────────────────┐
│   ConfigMap         │
│   (model.json)      │
└──────────┬──────────┘
           │
           │ Mounted as volume
           ▼
┌─────────────────────┐       ┌──────────────┐
│  langop/model       │◄──────┤   Secret     │
│  (LiteLLM Proxy)    │       │  (API keys)  │
└──────────┬──────────┘       └──────────────┘
           │
           │ OpenAI-compatible API
           │ :4000/v1/chat/completions
           ▼
┌─────────────────────┐
│  Agents & Clients   │
│  (OpenAI SDK)       │
└─────────────────────┘
```

## Features

- 🚀 **100+ Provider Support** - OpenAI, Anthropic, Azure, Ollama, Bedrock, Vertex AI, and more
- 🔒 **Rate Limiting** - Per-model request and token limits (enforced by LiteLLM)
- ⚖️ **Load Balancing** - Distribute requests across multiple endpoints
- 🔄 **Automatic Retries** - Configurable exponential backoff
- 💾 **Response Caching** - Reduce costs and latency with intelligent caching
- 📊 **Observability** - Prometheus metrics, distributed tracing, structured logging
- 🎯 **Fallback Models** - Automatic failover to backup models
- 💰 **Cost Tracking** - Monitor token usage and costs per model
- 🔌 **OpenAI Compatible** - Works with any OpenAI SDK or client library

## Quick Start

### 1. Build the Image

```bash
make build
```

### 2. Deploy with Kubernetes

Create a LanguageModel resource:

```yaml
apiVersion: langop.io/v1alpha1
kind: LanguageModel
metadata:
  name: gpt-4
  namespace: langop-system
spec:
  provider: openai
  modelName: gpt-4-turbo-preview
  apiKeySecretRef:
    name: openai-credentials
    key: api-key
  rateLimits:
    requestsPerMinute: 100
    tokensPerMinute: 100000
```

The language-operator will:
1. Create a ConfigMap with the model spec at `/etc/langop/model.json`
2. Deploy the `langop/model` proxy container
3. Mount the ConfigMap and Secret
4. Create a Service for accessing the proxy

### 3. Use from Agents/Clients

Connect to the proxy using any OpenAI SDK:

```python
from openai import OpenAI

client = OpenAI(
    base_url="http://gpt-4.langop-system.svc.cluster.local:4000/v1",
    api_key="not-needed"  # Key is already configured in proxy
)

response = client.chat.completions.create(
    model="gpt-4-turbo-preview",
    messages=[{"role": "user", "content": "Hello!"}]
)
```

## Configuration

The proxy automatically generates LiteLLM configuration from the LanguageModel CRD spec:

| LanguageModel Field | LiteLLM Mapping | Description |
|---------------------|-----------------|-------------|
| `spec.provider` | `litellm_params.model` | Provider prefix (e.g., `azure/`, `openai/`) |
| `spec.modelName` | `model_name` | Model identifier |
| `spec.endpoint` | `litellm_params.api_base` | Custom endpoint URL |
| `spec.apiKeySecretRef` | `litellm_params.api_key` | API key from secret |
| `spec.rateLimits.requestsPerMinute` | `rpm` | Request rate limit |
| `spec.rateLimits.tokensPerMinute` | `tpm` | Token rate limit |
| `spec.retryPolicy` | `litellm_settings.num_retries` | Retry configuration |
| `spec.fallbacks` | `litellm_settings.fallbacks` | Fallback models |
| `spec.loadBalancing` | `router_settings.routing_strategy` | Load balancing strategy |
| `spec.caching` | `litellm_settings.cache` | Response caching |

## Supported Providers

The proxy supports 100+ providers through LiteLLM. Most common providers:

### Cloud Providers
- **OpenAI** - `provider: openai`
- **Anthropic** - `provider: anthropic`
- **Azure OpenAI** - `provider: azure`
- **AWS Bedrock** - `provider: bedrock`
- **Google Vertex AI** - `provider: vertex`

### Local/Self-Hosted
- **Ollama** - `provider: openai-compatible`, `endpoint: http://ollama:11434/v1`
- **LM Studio** - `provider: openai-compatible`, `endpoint: http://lmstudio:1234/v1`
- **vLLM** - `provider: openai-compatible`, `endpoint: http://vllm:8000/v1`
- **Text Generation WebUI** - `provider: openai-compatible`

### Custom Endpoints
- **OpenAI-Compatible** - `provider: openai-compatible`
- **Custom** - `provider: custom`

See [examples/](examples/) for provider-specific configurations.

## Examples

### OpenAI with Rate Limiting

```yaml
spec:
  provider: openai
  modelName: gpt-4-turbo-preview
  apiKeySecretRef:
    name: openai-credentials
  rateLimits:
    requestsPerMinute: 100
    tokensPerMinute: 100000
    concurrentRequests: 10
```

See [examples/openai-model.yaml](examples/openai-model.yaml)

### Anthropic Claude

```yaml
spec:
  provider: anthropic
  modelName: claude-3-5-sonnet-20241022
  apiKeySecretRef:
    name: anthropic-credentials
  rateLimits:
    requestsPerMinute: 50
    tokensPerMinute: 80000
```

See [examples/anthropic-model.yaml](examples/anthropic-model.yaml)

### Local Ollama

```yaml
spec:
  provider: openai-compatible
  modelName: llama3.2
  endpoint: http://ollama.default.svc.cluster.local:11434/v1
  rateLimits:
    requestsPerMinute: 200
```

See [examples/ollama-local.yaml](examples/ollama-local.yaml)

### Azure OpenAI

```yaml
spec:
  provider: azure
  modelName: gpt-4-deployment
  endpoint: https://your-resource.openai.azure.com/
  apiKeySecretRef:
    name: azure-credentials
  configuration:
    additionalParameters:
      api_version: "2025-01-01-preview"
```

See [examples/azure-openai.yaml](examples/azure-openai.yaml)

### High-Availability with Load Balancing

```yaml
spec:
  provider: openai
  modelName: gpt-4
  loadBalancing:
    strategy: latency-based
    endpoints:
      - url: https://api.openai.com/v1
        region: us-east-1
        priority: 1
      - url: https://api.openai.com/v1
        region: us-west-2
        priority: 1
    healthCheck:
      enabled: true
      interval: "30s"
  fallbacks:
    - modelRef: gpt-3.5-turbo
```

See [examples/multi-endpoint-loadbalanced.yaml](examples/multi-endpoint-loadbalanced.yaml)

## Rate Limiting Behavior

Rate limits are enforced by LiteLLM at the proxy level:

- **Per-Model Isolation** - Each LanguageModel CRD gets its own proxy instance with isolated rate limits
- **Token Tracking** - Both request count and token usage are tracked
- **Automatic Queuing** - Requests that exceed limits are queued and retried
- **Fair Distribution** - Multiple endpoints share rate limits proportionally

Example rate limit configuration:

```yaml
rateLimits:
  requestsPerMinute: 100    # Max 100 requests/minute
  tokensPerMinute: 100000   # Max 100k tokens/minute
  concurrentRequests: 10    # Max 10 concurrent requests
```

## Health Checks

The proxy exposes health check endpoints:

- `GET /health` - Overall health status
- `GET /metrics` - Prometheus metrics (if enabled)

Health check configuration in Kubernetes:

```yaml
livenessProbe:
  httpGet:
    path: /health
    port: 4000
  initialDelaySeconds: 10
  periodSeconds: 30

readinessProbe:
  httpGet:
    path: /health
    port: 4000
  initialDelaySeconds: 5
  periodSeconds: 10
```

## Observability

### Metrics

Enable Prometheus metrics:

```yaml
spec:
  observability:
    metrics: true
```

Available metrics:
- Request counts and error rates
- Latency percentiles (p50, p95, p99)
- Token usage (input/output)
- Cost tracking
- Rate limit hits

### Logging

Configure logging levels:

```yaml
spec:
  observability:
    logging:
      level: info  # debug, info, warn, error
      logRequests: true
      logResponses: false  # Disable for privacy
```

### Tracing

Enable distributed tracing:

```yaml
spec:
  observability:
    tracing: true
```

## Development

### Local Testing

Test with a sample configuration:

```bash
# Test with OpenAI (requires API key)
make test

# Test with local Ollama (no API key needed)
make test-local

# Test with rate limiting
make test-rate-limit
```

### Build and Push

```bash
# Build the image
make build

# Scan for vulnerabilities
make scan

# Push to registry
make push
```

### Debug Mode

Enable debug output:

```bash
docker run -e DEBUG=true \
  -v $(pwd)/test-config/model.json:/etc/langop/model.json:ro \
  -p 4000:4000 \
  git.theryans.io/language-operator/model:latest
```

## Architecture Details

### Config Generation Flow

1. **Startup** - Entrypoint script runs
2. **Read CRD** - Parse `/etc/langop/model.json` (mounted ConfigMap)
3. **Load Secrets** - Read API keys from `/etc/secrets/`
4. **Generate Config** - Python script creates LiteLLM `config.yaml`
5. **Start Proxy** - Launch LiteLLM with generated config

### File Locations

- `/etc/langop/model.json` - LanguageModel spec (mounted ConfigMap)
- `/etc/secrets/<secret-name>/<key>` - API keys (mounted Secrets)
- `/app/config.yaml` - Generated LiteLLM configuration
- `/usr/local/bin/generate-config.py` - Config generator script
- `/usr/local/bin/entrypoint.sh` - Container entrypoint

### Port Configuration

- **4000** - LiteLLM proxy HTTP server
- **4000/v1/chat/completions** - OpenAI-compatible chat API
- **4000/health** - Health check endpoint
- **4000/metrics** - Prometheus metrics (if enabled)

## Integration with language-operator

The language-operator automates the deployment:

1. **Watch LanguageModel CRDs** - Controller monitors for changes
2. **Create ConfigMap** - Serialize spec to JSON
3. **Create Deployment** - Deploy `langop/model` container
4. **Mount Volumes** - Attach ConfigMap and Secrets
5. **Create Service** - Expose proxy on port 4000
6. **Update Status** - Report health and metrics

Agents and clients reference models by name:

```yaml
apiVersion: langop.io/v1alpha1
kind: LanguageAgent
metadata:
  name: my-agent
spec:
  modelRef:
    name: gpt-4  # References LanguageModel CRD
```

The agent connects to: `http://gpt-4.langop-system.svc.cluster.local:4000`

## Performance

### Benchmarks

Tested with LiteLLM v1.50.0 on Kubernetes 1.28:

- **Latency Overhead** - ~5-10ms added latency vs direct API calls
- **Throughput** - 1000+ req/s per proxy instance (CPU-bound)
- **Memory** - ~150MB base + ~50MB per 1000 cached responses
- **Startup Time** - ~2-3 seconds from container start to ready

### Scaling

For high-traffic models:

1. **Horizontal Scaling** - Deploy multiple replicas of the proxy
2. **Load Balancing** - Use multiple upstream endpoints
3. **Caching** - Enable response caching to reduce API calls
4. **Rate Limiting** - Prevent overload and control costs

## Troubleshooting

### Common Issues

**Proxy won't start:**
- Check if `/etc/langop/model.json` exists and is valid JSON
- Verify API key secret is mounted correctly
- Enable debug mode: `DEBUG=true`

**Rate limits not working:**
- Ensure `rateLimits` is set in LanguageModel spec
- Check LiteLLM logs for rate limit configuration
- Verify multiple proxies aren't sharing limits (use Redis for shared state)

**High latency:**
- Check if caching is enabled
- Verify network connectivity to provider
- Monitor health check intervals (may cause spikes)

**API key errors:**
- Verify secret exists: `kubectl get secret <name>`
- Check secret is mounted at `/etc/secrets/<name>/<key>`
- Ensure secret has correct format (plain text, no quotes)

### Debugging

View generated config:

```bash
kubectl exec -it <pod-name> -- cat /app/config.yaml
```

Check proxy logs:

```bash
kubectl logs <pod-name> -f
```

Test health endpoint:

```bash
kubectl exec -it <pod-name> -- curl http://localhost:4000/health
```

## Security Considerations

- ✅ **Non-root User** - Proxy runs as `based` user (UID 1000)
- ✅ **Secrets Management** - API keys never logged or exposed
- ✅ **Network Isolation** - Deploy in isolated namespace
- ✅ **TLS Support** - Use HTTPS endpoints for providers
- ✅ **Rate Limiting** - Prevent abuse and cost overruns
- ⚠️ **OpenAI API Compatibility** - Anyone with access can use the proxy

### Recommended Security Practices

1. **Use Network Policies** - Restrict which pods can access proxies
2. **Enable RBAC** - Limit who can create LanguageModel CRDs
3. **Rotate API Keys** - Regularly update secrets
4. **Monitor Costs** - Enable cost tracking and alerts
5. **Audit Logs** - Enable request logging for security audits

## License

Apache License 2.0 - See [LICENSE](../../LICENSE) for details.

## Contributing

Contributions welcome! See [CONTRIBUTING.md](../../CONTRIBUTING.md) for guidelines.

## Related Projects

- [language-operator](../../kubernetes/language-operator/) - Kubernetes operator for managing LLM infrastructure
- [LiteLLM](https://github.com/BerriAI/litellm) - Unified LLM API gateway
- [OpenAI Python SDK](https://github.com/openai/openai-python) - Client library for OpenAI API