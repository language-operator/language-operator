# Models

A `LanguageModel` configures LLM access for a `LanguageCluster`. The operator reads all `LanguageModel` resources in the namespace and registers them with the cluster's shared LiteLLM gateway — agents never hold API credentials or connect to model providers directly.

## How It Works

One LiteLLM proxy (`gateway`) runs per `LanguageCluster`. When you add or remove a `LanguageModel`, the gateway restarts with the updated model list — no agent redeploy required.

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
| `MODEL_ENDPOINT` | `http://gateway.<namespace>.svc.cluster.local:8000` |
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
| Anthropic | `provider: anthropic` |
| OpenAI | `provider: openai` |
| Google AI Studio (Gemini API) | `provider: gemini` |
| Azure OpenAI | `provider: azure` (+ `endpoint`, `apiVersion`) |
| AWS Bedrock | `provider: bedrock` (+ `region`) |
| Google Vertex AI | `provider: vertex` (+ `project`, `location`) |
| Any OpenAI chat-completions API (Ollama, vLLM, LM Studio…) | `provider: openai-compatible` (+ `endpoint`) |
| Any other LiteLLM provider (DeepSeek, Qwen, xAI, Mistral, Groq, OpenRouter…) | `litellmProvider: <LiteLLM prefix>` |

`custom` is deprecated and behaves exactly like `openai-compatible`. Providers that need more than one credential (Bedrock access keys, a Vertex service account, an Azure AD app) take a whole Secret through `credentialsSecretRef`, applied to that model only. See [LanguageModel](../api/languagemodel.md#providers) for each provider's fields and examples.

### Self-hosted models (Ollama, vLLM)

```yaml
spec:
  provider: openai-compatible
  modelName: llama3.2
  endpoint: http://ollama.default.svc.cluster.local:11434/v1
```

No `apiKeySecretRef` needed for unauthenticated endpoints.

The backend only needs `/v1/chat/completions`. Clients that speak the Responses API (`/v1/responses`, e.g. Codex) or the Anthropic Messages API (`/v1/messages`) still work: the gateway translates both to chat completions for these models.

### Multiple models

Agents can reference multiple models. Each model is registered with the same gateway; the agent chooses which to call at runtime:

```yaml
# LanguageAgent
spec:
  models:
    - name: claude-sonnet   # primary
    - name: llama3          # fallback / secondary
```

### Load balancing: models that share a `modelName`

Agents call the gateway by `modelName`, not by the LanguageModel's own name. When several LanguageModels in a cluster have the same `modelName`, the gateway serves them as one model and spreads requests across all of them. Use this to run one model on several endpoints:

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

Because the same thing happens by accident, for example two models with the same `modelName` but different providers or keys, creating or updating a LanguageModel whose `modelName` is already taken returns a warning naming the other models, and the gateway logs one on start. Give each model a distinct `modelName` if you want them addressed separately.

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
