# Models

A `LanguageModel` configures LLM access for a `LanguageCluster`. The operator reads all `LanguageModel` resources in the namespace and registers them with the cluster's shared LiteLLM gateway — agents never hold API credentials or connect to model providers directly.

## How It Works

The model gateway is a single LiteLLM proxy Deployment (`gateway`) deployed per `LanguageCluster`. When you create or update a `LanguageModel`, the `LanguageCluster` controller detects the change, regenerates the `gateway-config` ConfigMap with the full model list, and rolls the gateway Deployment. Existing agents pick up the new model immediately — no agent redeploy required.

```
LanguageModel CRs  →  gateway-config ConfigMap  →  gateway Deployment (LiteLLM)
                                                            ↑
                                          all agents connect here via MODEL_ENDPOINTS
```

The `LanguageModel` controller itself only validates the spec and sets `status.phase`. It creates no Deployment or Service of its own.

## Credential Management

API keys are never injected into agent pods. Store them in a Secret:

```bash
kubectl create secret generic anthropic-credentials \
  --from-literal=api-key=sk-ant-your-key-here
```

Reference the Secret from the model spec:

```yaml
apiVersion: langop.io/v1alpha1
kind: LanguageModel
metadata:
  name: claude-sonnet
spec:
  provider: anthropic
  modelName: claude-sonnet-4-5
  apiKeySecretRef:
    name: anthropic-credentials
    key: api-key
```

The gateway pod mounts the Secret and presents a single OpenAI-compatible endpoint to agents. Rotating a key is a `kubectl create secret` operation — the gateway restarts, agents are unaffected.

## Agent Integration

The operator injects two environment variables into every agent container:

| Variable | Value |
|----------|-------|
| `MODEL_ENDPOINTS` | `http://gateway.<namespace>.svc.cluster.local:8000` |
| `LLM_MODEL` | Comma-separated list of model names from `spec.models[].name` |

Both are also available through `/etc/agent/config.yaml` under the `models:` key:

```yaml
models:
  claude-sonnet:
    role: primary
    provider: anthropic
    model: claude-sonnet-4-5
    endpoint: http://gateway.my-cluster.svc.cluster.local:8000
```

Agents call the gateway with the model name they want. The gateway routes to the correct upstream provider.

## Supported Providers

| Provider | Value |
|----------|-------|
| Anthropic | `anthropic` |
| OpenAI | `openai` |
| Azure OpenAI | `azure` |
| AWS Bedrock | `bedrock` |
| Google Vertex AI | `vertex` |
| Any OpenAI-compatible API | `openai-compatible` |
| Custom LiteLLM config | `custom` |

### Self-hosted models (Ollama, vLLM)

```yaml
spec:
  provider: openai-compatible
  modelName: llama3.2
  endpoint: http://ollama.default.svc.cluster.local:11434/v1
```

No `apiKeySecretRef` needed for unauthenticated endpoints.

### Multiple models

Agents can reference multiple models. Each model is registered with the same gateway; the agent chooses which to call at runtime:

```yaml
# LanguageAgent
spec:
  models:
    - name: claude-sonnet   # primary
    - name: llama3          # fallback / secondary
```

## Rate Limiting

```yaml
spec:
  rateLimits:
    requestsPerMinute: 100
    tokensPerMinute: 50000
```

Limits are enforced by the shared gateway across all agents. Per-agent limits are not currently supported.

## Related

- [LanguageCluster](../api/languagecluster.md) — owns the shared gateway
- [LanguageAgent](../api/languageagent.md) — references models via `spec.models`
- [LanguageModel API Reference](../api/languagemodel.md) — full field documentation
