# Models

A `LanguageModel` configures LLM access for a `LanguageCluster`. The operator reads all `LanguageModel` resources in the namespace and registers them with the cluster's shared LiteLLM gateway. Agents on gateway-backed runtimes never hold provider credentials or connect to providers directly: they call the gateway with their own gateway key.

The exception is **vendor-key runtimes** — [Claude Code](../runtimes/claude-code.md) and [Cursor](../runtimes/cursor.md) — which talk to their vendor directly with the vendor's own key and do not use the gateway or `LanguageModel`s at all.

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

## Gateway Authentication

The gateway only serves requests carrying a valid key. Every agent gets its own:

- The operator injects **`MODEL_API_KEY`** into each agent's containers, next to `MODEL_ENDPOINT`. Agents send it as the bearer key (`Authorization: Bearer $MODEL_API_KEY`); the bundled gateway-backed runtimes do this for you.
- A key is `sk-langop-<agent UUID>.<signature>`, where the signature is an HMAC of the agent UUID under the cluster's secret. The gateway checks it without a key store, and records the agent UUID as the request's LiteLLM `user_id`, so usage can be attributed per agent.
- A request without a valid key gets **401**. Custom agent images must send `MODEL_API_KEY`; one that sends a fixed placeholder is rejected.

The HMAC secret lives in the Secret `gateway-auth` (key `hmac-secret`) in the cluster's namespace. The operator generates it on first reconcile; create it yourself beforehand to choose the value. **Rotating** it (edit the value, or delete the Secret to have a new one generated) restarts the gateway and replaces every agent's key, and running agents are restarted with their new key.

For access outside an agent — an admin, a script, a client using the gateway's external ingress — set a master key on the gateway, which the gateway also accepts:

```yaml
apiVersion: langop.io/v1alpha1
kind: LanguageCluster
spec:
  gateway:
    deployment:
      env:
        - name: LITELLM_MASTER_KEY
          valueFrom:
            secretKeyRef:
              name: gateway-master-key
              key: key
```

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

### Other providers

Anything LiteLLM supports is reachable with `litellmProvider`, its LiteLLM provider prefix, plus the provider's API key. LiteLLM knows each one's base URL, so `endpoint` is only needed to override it:

| Provider | `litellmProvider` |
|----------|-------------------|
| DeepSeek | `deepseek` |
| Z.ai (GLM) | `zai` |
| Moonshot (Kimi) | `moonshot` |
| MiniMax | `minimax` |
| xAI (Grok) | `xai` |
| Mistral | `mistral` |
| Groq | `groq` |
| Together AI | `together_ai` |
| Fireworks AI | `fireworks_ai` |
| OpenRouter | `openrouter` |
| Alibaba Qwen (DashScope) | `dashscope` |
| Cerebras | `cerebras` |
| Perplexity | `perplexity` |

```yaml
spec:
  litellmProvider: deepseek
  modelName: deepseek-chat
  apiKeySecretRef:
    name: deepseek-credentials
```

See [LiteLLM's provider list](https://docs.litellm.ai/docs/providers) for the rest. Every [example](https://github.com/language-operator/language-operator/tree/main/components/model-gateway/examples) in the repository is a complete, CI-validated spec.

### Self-hosted models (Ollama, vLLM)

```yaml
spec:
  provider: openai-compatible
  modelName: llama3.2
  endpoint: http://ollama.default.svc.cluster.local:11434/v1
```

No `apiKeySecretRef` needed for unauthenticated endpoints. The same shape works for any server with an OpenAI-compatible API; their usual defaults:

| Server | `endpoint` (default port) |
|--------|---------------------------|
| Ollama | `http://<host>:11434/v1` |
| vLLM | `http://<host>:8000/v1` |
| llama.cpp (`llama-server`) | `http://<host>:8080/v1` |
| SGLang | `http://<host>:30000/v1` |
| LM Studio | `http://<host>:1234/v1` |

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

### Wildcard models: a whole vendor catalogue

A LanguageModel with `modelName: "*"` stands for every model the provider offers, so one resource covers a vendor like OpenRouter:

```yaml
apiVersion: langop.io/v1alpha1
kind: LanguageModel
metadata:
  name: openrouter
spec:
  provider: openai-compatible
  modelName: "*"
  endpoint: https://openrouter.ai/api/v1
  apiKeySecretRef:
    name: openrouter-credentials
```

The gateway registers it as `openrouter/*`. An agent picks a model by the vendor's own name with `model`, and calls the gateway with `<LanguageModel name>/<model>`; the gateway strips the prefix before calling the provider:

```yaml
# LanguageAgent
spec:
  models:
    - name: openrouter
      model: anthropic/claude-sonnet-4.5   # the agent's LLM_MODEL is openrouter/anthropic/claude-sonnet-4.5
```

Each wildcard is addressed by its own resource name, so several vendors can coexist, and a name no LanguageModel serves is still rejected. Rate limits and the timeout apply to all of the vendor's models together. `*` is only valid on its own: partial patterns such as `gpt-*` are rejected.

!!! note "Listing models"

    For a wildcard, the gateway's `/v1/models` does not return the vendor's catalogue. It lists
    LiteLLM's built-in table of known models for the provider type — for an OpenAI-compatible
    vendor like OpenRouter, that is OpenAI's model names. To discover what a vendor offers, use
    the vendor's own model list API.

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

## Advanced: Extra Gateway Configuration

For LiteLLM settings with no `LanguageModel` field (retries, fallbacks, caching, spend callbacks, or a hand-written model entry) set `LANGOP_GATEWAY_EXTRA_CONFIG` on the gateway. It is a YAML mapping deep-merged into the generated LiteLLM config: mappings merge recursively, lists and scalars replace.

```yaml
apiVersion: langop.io/v1alpha1
kind: LanguageCluster
spec:
  gateway:
    deployment:
      env:
        - name: LANGOP_GATEWAY_EXTRA_CONFIG
          value: |
            litellm_settings:
              num_retries: 2
              request_timeout: 600
```

A list such as `model_list` replaces the generated one entirely, so prefer `LanguageModel`s for models. A value that is not a mapping stops the gateway from starting.

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
