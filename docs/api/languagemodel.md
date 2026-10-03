# LanguageModel

The `LanguageModel` CRD configures LLM access through the cluster's shared LiteLLM proxy.

## Overview

A LanguageModel defines:
- Provider (Anthropic, OpenAI, Azure, etc.)
- Model name and version
- API credentials (via Secret references)
- Provider-specific settings (region, project, API version, extra LiteLLM params)
- Rate limits and timeout

## Quick Example

```yaml
apiVersion: langop.io/v1alpha1
kind: LanguageModel
metadata:
  name: claude-sonnet
  namespace: my-cluster
spec:
  provider: anthropic
  modelName: claude-sonnet-4-5
  apiKeySecretRef:
    name: anthropic-credentials
    key: api-key
```

## Complete API Reference

See the [Complete API Reference](reference.md#languagemodel) for full field documentation including:

- **LanguageModel** - Top-level resource
- **LanguageModelSpec** - Specification fields
- **LanguageModelStatus** - Status and endpoint information

## Providers

Set **exactly one** of `provider` or `litellmProvider`.

| `provider` | Reaches | Also needs |
|------------|---------|------------|
| `openai` | OpenAI | API key |
| `anthropic` | Anthropic | API key |
| `gemini` | Google AI Studio (Gemini API) | API key |
| `azure` | Azure OpenAI; `modelName` is the **deployment name** | `endpoint`, `apiVersion`, an API key or Azure AD app credentials |
| `bedrock` | AWS Bedrock | `region`, AWS access keys or a Bedrock bearer token |
| `vertex` | Google Vertex AI | `project`, `location`, a service-account JSON |
| `openai-compatible` | Any OpenAI chat-completions API: Ollama, vLLM, LM Studio… | `endpoint` |
| `custom` | *Deprecated*, identical to `openai-compatible` | |

For anything else LiteLLM supports, set **`litellmProvider`** to its LiteLLM prefix (e.g. `deepseek`, `dashscope`, `xai`, `mistral`, `groq`, `openrouter`, `hosted_vllm`). The gateway calls the model as `<litellmProvider>/<modelName>`, so a new vendor needs no operator change:

```yaml
spec:
  litellmProvider: deepseek
  modelName: deepseek-chat
  apiKeySecretRef:
    name: deepseek-credentials
```

The admission webhook rejects a spec that is missing what its `provider` needs (the "Also needs" column).

### Typed fields and `params`

| Field | LiteLLM param | Used by |
|-------|---------------|---------|
| `region` | `aws_region_name` | `bedrock` |
| `project` | `vertex_project` | `vertex` |
| `location` | `vertex_location` | `vertex` |
| `apiVersion` | `api_version` | `azure` |

`params` passes anything else straight into the model's LiteLLM params, keeping JSON types, and overrides the fields above:

```yaml
spec:
  params:
    aws_bedrock_runtime_endpoint: https://vpce-0123.bedrock-runtime.us-east-1.vpce.amazonaws.com
    extra_headers:
      X-Team: platform
```

Credentials never go in `params`: keys such as `api_key`, `aws_secret_access_key` or `client_secret` are rejected. Use a Secret.

## Key Concepts

### Shared Proxy Registration

When you create a LanguageModel:

1. The LanguageModel controller validates the spec and sets `status.phase: Ready`
2. The LanguageCluster controller (which watches `LanguageModel` resources) detects the new CR
3. The shared `gateway-config` ConfigMap is regenerated with all models in the namespace
4. The gateway Deployment rolls over with the updated configuration

All agents immediately have access to the new model via `MODEL_ENDPOINT`.

### Credential Management

Credentials live in Secrets and are read only by the gateway, never by agent pods.

**One key:** `apiKeySecretRef` names a Secret and a key (default `api-key`):

```bash
kubectl create secret generic anthropic-key --from-literal=api-key=sk-ant-...
```

```yaml
spec:
  apiKeySecretRef:
    name: anthropic-key
    key: api-key
```

**Several values:** `credentialsSecretRef` names a whole Secret. Its keys are recognised by name and applied **to this model only**, so two models can use different credentials for the same provider:

| Secret key | Becomes |
|------------|---------|
| `AWS_ACCESS_KEY_ID`, `AWS_SECRET_ACCESS_KEY`, `AWS_SESSION_TOKEN` | Bedrock SigV4 credentials |
| `AWS_BEARER_TOKEN_BEDROCK` | Bedrock bearer token |
| `VERTEX_CREDENTIALS`, `GOOGLE_APPLICATION_CREDENTIALS`, `service-account.json`, `credentials.json` | Vertex service-account JSON (read from the mounted file) |
| `AZURE_API_KEY`, `GEMINI_API_KEY`, `api-key` | the API key |
| `AZURE_TENANT_ID`, `AZURE_CLIENT_ID`, `AZURE_CLIENT_SECRET`, `AZURE_AD_TOKEN` | Azure AD credentials |

LiteLLM's own param names (`aws_access_key_id`, `api_key`, `tenant_id`…) also work as keys. Other keys are ignored and named in the gateway log. When both references supply an API key, `apiKeySecretRef` wins.

### Rate Limiting

Configure per-model rate limits:

```yaml
spec:
  rateLimits:
    requestsPerMinute: 100
    tokensPerMinute: 50000
```

The shared proxy enforces these limits across all agents.

## Provider-Specific Examples

### AWS Bedrock (access keys)

```bash
kubectl create secret generic aws-bedrock \
  --from-literal=AWS_ACCESS_KEY_ID=AKIA... \
  --from-literal=AWS_SECRET_ACCESS_KEY=...
```

```yaml
spec:
  provider: bedrock
  modelName: anthropic.claude-sonnet-4-5-20250929-v1:0
  region: us-east-1
  credentialsSecretRef:
    name: aws-bedrock
```

### AWS Bedrock (bearer token)

```bash
kubectl create secret generic bedrock-token --from-literal=AWS_BEARER_TOKEN_BEDROCK=...
```

```yaml
spec:
  provider: bedrock
  modelName: amazon.nova-pro-v1:0
  region: us-west-2
  credentialsSecretRef:
    name: bedrock-token
```

### Google Vertex AI

```bash
kubectl create secret generic vertex-sa --from-file=service-account.json=./sa.json
```

```yaml
spec:
  provider: vertex
  modelName: gemini-2.5-pro
  project: my-gcp-project
  location: us-central1
  credentialsSecretRef:
    name: vertex-sa
```

### Gemini (Google AI Studio)

```yaml
spec:
  provider: gemini
  modelName: gemini-2.5-flash
  apiKeySecretRef:
    name: gemini-key
```

### Azure OpenAI

`modelName` is your Azure **deployment** name.

```yaml
spec:
  provider: azure
  modelName: my-gpt4o-deployment
  endpoint: https://my-resource.openai.azure.com
  apiVersion: "2025-01-01-preview"
  apiKeySecretRef:
    name: azure-credentials
```

With an Azure AD app instead of a key, put `AZURE_TENANT_ID`, `AZURE_CLIENT_ID` and `AZURE_CLIENT_SECRET` in a Secret and reference it with `credentialsSecretRef`.

### Self-Hosted (Ollama, vLLM)

```yaml
spec:
  provider: openai-compatible
  modelName: llama3.2
  endpoint: http://ollama.default.svc.cluster.local:11434/v1
```

The backend only needs `/v1/chat/completions`: the gateway also serves `/v1/responses` and `/v1/messages` for these models by translating to it.

## Related Resources

- [LanguageAgent](languageagent.md) - Reference models in agents
- [LanguageCluster](languagecluster.md) - Shared proxy architecture

