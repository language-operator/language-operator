# langop/model

The model gateway for [language-operator](../../README.md): one per LanguageCluster, serving every `LanguageModel` in the cluster's namespace. Powered by [LiteLLM](https://docs.litellm.ai/), it gives agents one OpenAI-compatible endpoint (plus `/v1/responses` and `/v1/messages`) in front of any provider LiteLLM supports, with per-model rate limits and per-agent keys.

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
│   (gateway-config)  │
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

- **Any LiteLLM provider** - first-class `provider` values for OpenAI, Anthropic, Gemini, Azure, Bedrock, Vertex and OpenAI-compatible servers; anything else by `litellmProvider`
- **Per-agent keys** - every agent gets its own key; calls without a valid key are rejected, and usage is attributed per agent
- **Rate limiting** - per-model request and token limits (`spec.rateLimits`)
- **Load balancing** - LanguageModels sharing a `modelName` form one load-balanced group
- **OpenAI, Responses and Anthropic APIs** - `/v1/chat/completions`, `/v1/responses` and `/v1/messages` for every model
- **Anything else LiteLLM offers** (retries, fallbacks, caching, spend callbacks) through `LANGOP_GATEWAY_EXTRA_CONFIG`; none of it is on by default

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
  name: gpt-4o
  namespace: my-cluster      # your LanguageCluster's namespace
spec:
  provider: openai
  modelName: gpt-4o
  apiKeySecretRef:
    name: openai-credentials
    key: api-key
  rateLimits:
    requestsPerMinute: 100
    tokensPerMinute: 100000
```

The language-operator adds the model to the cluster's `gateway-config` ConfigMap (one `model__<name>.json` per LanguageModel, mounted at `/etc/langop/models/`), mounts the Secrets it references, and rolls the shared `gateway` Deployment. One gateway serves every model in the namespace, at `http://gateway.<namespace>.svc.cluster.local:8000`.

### 3. Use from Agents/Clients

Inside an agent, the operator injects `MODEL_ENDPOINT` (the gateway URL) and `MODEL_API_KEY` (the agent's own gateway key). Use them with any OpenAI SDK; ask for the model by its `modelName`:

```python
import os
from openai import OpenAI

client = OpenAI(
    base_url=os.environ["MODEL_ENDPOINT"] + "/v1",
    api_key=os.environ["MODEL_API_KEY"],  # calls without a valid key get 401
)

response = client.chat.completions.create(
    model="gpt-4o",
    messages=[{"role": "user", "content": "Hello!"}]
)
```

## Configuration

The proxy automatically generates LiteLLM configuration from the LanguageModel CRD spec:

| LanguageModel Field | LiteLLM Mapping | Description |
|---------------------|-----------------|-------------|
| `spec.provider` | `litellm_params.model` prefix | `anthropic/`, `gemini/`, `azure/`, `bedrock/`, `vertex_ai/`, `openai/` (openai-compatible); `openai` stays unprefixed |
| `spec.litellmProvider` | `litellm_params.model` prefix | Any LiteLLM provider prefix, instead of `provider` |
| `spec.modelName` | `model_name` | Model identifier (what agents ask for) |
| `spec.endpoint` | `litellm_params.api_base` | Custom endpoint URL |
| `spec.apiKeySecretRef` | `litellm_params.api_key` | One key from a Secret |
| `spec.credentialsSecretRef` | per-key `litellm_params` | A whole Secret: AWS keys or bearer token, Vertex service account (as a file path), Azure AD app, API key |
| `spec.region` / `project` / `location` / `apiVersion` | `aws_region_name` / `vertex_project` / `vertex_location` / `api_version` | Provider settings |
| `spec.params` | `litellm_params.*` | Anything else, typed; overrides the fields above |
| `spec.rateLimits.requestsPerMinute` | `rpm` | Request rate limit |
| `spec.rateLimits.tokensPerMinute` | `tpm` | Token rate limit |
| `spec.timeout` | `litellm_params.timeout` | Request timeout (Go duration, e.g. `5m`) |

LanguageModels that share a `spec.modelName` become one `model_name` group, which LiteLLM load-balances (see [below](#high-availability-with-load-balancing)).

## Operator-level settings (environment)

Two environment variables on the gateway Deployment shape the generated config without a new image.

| Variable | Effect |
|---|---|
| `LANGOP_GATEWAY_EXTRA_CONFIG` | Optional; set through `LanguageCluster.spec.gateway.deployment.env`. A YAML mapping deep-merged into the generated LiteLLM config: mappings merge recursively, lists and scalars replace. Use it for spend callbacks, retries, fallbacks, caching, or any other LiteLLM setting. Anything that is not a mapping fails startup. |
| `LANGOP_GATEWAY_HMAC_SECRET` | **Set by the operator** from the Secret `gateway-auth` in the cluster namespace. Turns on per-agent keys through `custom_auth.py`: a key is `sk-langop-<agent-id>.<signature>`, the signature the first 32 hex characters of HMAC-SHA256(secret, agent-id), and the agent id becomes the request's LiteLLM user id. The operator gives every agent its key as `MODEL_API_KEY`; calls without a valid key get 401. A `LITELLM_MASTER_KEY` you set keeps working alongside, for admin and external access. |

Example: a master key for admin access, plus spend callbacks to a control plane. Per-agent keys need no configuration.

```yaml
spec:
  gateway:
    deployment:
      env:
        - name: LITELLM_MASTER_KEY
          valueFrom: {secretKeyRef: {name: gateway-master-key, key: key}}
        - name: GENERIC_LOGGER_ENDPOINT
          value: https://cloud.example.com/hooks/litellm/<token>
        - name: LANGOP_GATEWAY_EXTRA_CONFIG
          value: |
            litellm_settings:
              success_callback: ["generic_api"]
              failure_callback: ["generic_api"]
```

## Supported Providers

Every provider LiteLLM supports is reachable. These have first-class `provider` values; anything else uses `litellmProvider`:

### Cloud Providers
- **OpenAI** - `provider: openai`
- **Anthropic** - `provider: anthropic`
- **Azure OpenAI** - `provider: azure`
- **AWS Bedrock** - `provider: bedrock`
- **Google Vertex AI** - `provider: vertex`
- **Google AI Studio (Gemini API)** - `provider: gemini`

### Local/Self-Hosted
- **Ollama** - `provider: openai-compatible`, `endpoint: http://ollama:11434/v1`
- **LM Studio** - `provider: openai-compatible`, `endpoint: http://lmstudio:1234/v1`
- **vLLM** - `provider: openai-compatible`, `endpoint: http://vllm:8000/v1`
- **Text Generation WebUI** - `provider: openai-compatible`

### Custom Endpoints
- **OpenAI-Compatible** - `provider: openai-compatible`
- **Any other LiteLLM provider** - `litellmProvider: <prefix>` (e.g. `deepseek`, `dashscope`); `provider: custom` is deprecated (same as `openai-compatible`)

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
  modelName: my-gpt4o-deployment        # the Azure deployment name
  endpoint: https://your-resource.openai.azure.com
  apiVersion: "2025-01-01-preview"
  apiKeySecretRef:
    name: azure-credentials
```

See [examples/azure-openai.yaml](examples/azure-openai.yaml)

### High-Availability with Load Balancing

There are no load-balancing fields on a LanguageModel. Instead, create several LanguageModels with the **same `modelName`**: agents call the gateway by `modelName`, so LiteLLM treats the entries as one model group and spreads requests across them, skipping an endpoint that fails until its cooldown ends.

```yaml
apiVersion: langop.io/v1alpha1
kind: LanguageModel
metadata:
  name: llama-gpu-a
spec:
  provider: openai-compatible
  modelName: llama3.2
  endpoint: http://ollama-a.inference.svc.cluster.local:11434/v1
---
apiVersion: langop.io/v1alpha1
kind: LanguageModel
metadata:
  name: llama-gpu-b
spec:
  provider: openai-compatible
  modelName: llama3.2
  endpoint: http://ollama-b.inference.svc.cluster.local:11434/v1
```

Because this also happens by accident, the operator's admission webhook warns when a `modelName` is already taken, and `generate-config.py` logs a warning on start.

See [examples/multi-endpoint-loadbalanced.yaml](examples/multi-endpoint-loadbalanced.yaml)

### Model Aliases (e.g. for Gemini CLI)

`spec.aliases` gives a model extra names, for clients that ask for fixed model names of their own. `generate-config.py` turns them into LiteLLM's `router_settings.model_group_alias`, mapping each alias onto the model's `modelName`:

```yaml
spec:
  provider: anthropic
  modelName: claude-sonnet-4-5
  aliases: [gemini-3.1-pro-preview, gemini-3.1-flash-lite]
```

```yaml
# generated
router_settings:
  model_group_alias:
    gemini-3.1-pro-preview: claude-sonnet-4-5
    gemini-3.1-flash-lite: claude-sonnet-4-5
```

LiteLLM resolves a real model name before an alias, so an alias equal to any `modelName` is dropped with a warning, as is one already claimed by another model. `LANGOP_GATEWAY_EXTRA_CONFIG` merges over the generated `router_settings`. Gemini-format clients reach the gateway at `/v1beta/models/<model>:generateContent` and send the agent key as `x-goog-api-key`; the full Gemini CLI alias set is in the [Models](../../docs/components/models.md#gemini-cli) docs.

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

- `GET /health/liveliness` - Liveness check (proxy is alive)
- `GET /health/readiness` - Readiness check (proxy is ready)
- `GET /health` - Overall health status (makes LLM API calls - disabled by default)
- `GET /metrics` - Prometheus metrics (if enabled)

Health check configuration in Kubernetes:

```yaml
livenessProbe:
  httpGet:
    path: /health/liveliness
    port: 4000
  initialDelaySeconds: 30
  periodSeconds: 300

readinessProbe:
  httpGet:
    path: /health/readiness
    port: 4000
  initialDelaySeconds: 30
  periodSeconds: 300
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
  git.theryans.io/language-operator/model-gateway:latest
```

## Architecture Details

### Config Generation Flow

1. **Startup** - Entrypoint script runs
2. **Read model specs** - Parse every `/etc/langop/models/model__*.json` (the mounted `gateway-config` ConfigMap); a single `/etc/langop/model.json` is read instead when that directory is empty (local testing)
3. **Load credentials** - Read keys from the Secrets mounted under `/etc/secrets/<name>/`
4. **Generate Config** - Python script creates LiteLLM `config.yaml`
5. **Start Proxy** - Launch LiteLLM with generated config

### File Locations

- `/etc/langop/models/model__<name>.json` - one LanguageModel spec per model (mounted `gateway-config` ConfigMap)
- `/etc/secrets/<secret-name>/<key>` - credentials (mounted Secrets)
- `/app/config.yaml` - Generated LiteLLM configuration
- `/usr/local/bin/generate-config.py` - Config generator script
- `/usr/local/bin/entrypoint.sh` - Container entrypoint

### Port Configuration

- **4000** - LiteLLM proxy HTTP server in the container; the `gateway` Service exposes it on **8000**
- **/v1/chat/completions**, **/v1/responses**, **/v1/messages** - OpenAI chat, OpenAI Responses and Anthropic Messages APIs
- **/health/liveliness**, **/health/readiness** - probes; no key needed

## Integration with language-operator

The LanguageCluster controller runs one gateway per cluster namespace:

1. **Watch LanguageModels** in the namespace
2. **Write `gateway-config`** - one `model__<name>.json` per model, from its spec
3. **Mount the referenced Secrets** under `/etc/secrets/<name>/`
4. **Keep `gateway-auth`** - the HMAC secret per-agent keys are signed with
5. **Run the `gateway` Deployment and Service** (port 8000), restarting it when any of the above changes

Agents reference models by name, and are given the gateway's address and their own key:

```yaml
apiVersion: langop.io/v1alpha1
kind: LanguageAgent
metadata:
  name: my-agent
spec:
  models:
    - name: gpt-4o   # a LanguageModel in the same namespace
```

The agent receives `MODEL_ENDPOINT=http://gateway.<namespace>.svc.cluster.local:8000` and `MODEL_API_KEY`.

## Performance

### Benchmarks

Tested with LiteLLM v1.89.0 on Kubernetes 1.28:

- **Latency Overhead** - ~5-10ms added latency vs direct API calls
- **Throughput** - 1000+ req/s per proxy instance (CPU-bound)
- **Memory** - ~150MB base + ~50MB per 1000 cached responses
- **Startup Time** - ~2-3 seconds from container start to ready

### Scaling

For high-traffic models:

1. **Horizontal Scaling** - Run several gateway replicas (`LanguageCluster.spec.gateway.deployment`)
2. **Load Balancing** - Several LanguageModels sharing a `modelName`, on different endpoints
3. **Caching** - Configure LiteLLM caching through `LANGOP_GATEWAY_EXTRA_CONFIG` (off by default)
4. **Rate Limiting** - Prevent overload and control costs

## Troubleshooting

### Common Issues

**Proxy won't start:**
- Check `kubectl logs` for the config generator's output: each model spec it loaded, and any key it could not find
- Verify API key secret is mounted correctly
- Enable debug mode: `DEBUG=true`

**Rate limits not working:**
- Ensure `rateLimits` is set in LanguageModel spec
- Check LiteLLM logs for rate limit configuration
- Verify multiple proxies aren't sharing limits (use Redis for shared state)

**High latency:**
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