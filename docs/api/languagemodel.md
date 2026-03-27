# LanguageModel

The `LanguageModel` CRD configures LLM access through the cluster's shared LiteLLM proxy.

## Overview

A LanguageModel defines:
- Provider (Anthropic, OpenAI, Azure, etc.)
- Model name and version
- API credentials (via Secret reference)
- Rate limits and retry policies
- Cost tracking configuration

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

## Supported Providers

- `anthropic` - Claude models
- `openai` - GPT models
- `azure` - Azure OpenAI Service
- `bedrock` - AWS Bedrock
- `vertex` - Google Vertex AI
- `openai-compatible` - Any OpenAI-compatible API
- `custom` - Custom LiteLLM configuration

## Key Concepts

### Shared Proxy Registration

When you create a LanguageModel:

1. The controller creates a ConfigMap with the model spec
2. The LanguageCluster controller detects the change
3. The shared proxy configuration is regenerated
4. The proxy pod is restarted with new config

All agents immediately have access to the new model via `MODEL_ENDPOINTS`.

### Credential Management

Store API keys in Secrets:

```bash
kubectl create secret generic anthropic-key \
  --from-literal=api-key=sk-ant-...
```

Reference in the model spec:

```yaml
spec:
  apiKeySecretRef:
    name: anthropic-key
    key: api-key
```

The operator injects credentials into the shared proxy, never into agent pods.

### Rate Limiting

Configure per-model rate limits:

```yaml
spec:
  rateLimits:
    requestsPerMinute: 100
    tokensPerMinute: 50000
```

The shared proxy enforces these limits across all agents.

### Cost Tracking

Enable cost metrics:

```yaml
spec:
  costTracking:
    enabled: true
    budgetLimit: "100.00"
```

View costs in the model status:

```bash
kubectl get languagemodel claude-sonnet -o jsonpath='{.status.costMetrics}'
```

## Provider-Specific Examples

### Azure OpenAI

```yaml
spec:
  provider: azure
  modelName: gpt-4
  endpoint: https://my-resource.openai.azure.com
  apiKeySecretRef:
    name: azure-credentials
    key: api-key
  config:
    apiVersion: "2024-02-01"
    deploymentName: gpt-4-deployment
```

### Self-Hosted (Ollama)

```yaml
spec:
  provider: openai-compatible
  modelName: llama3.2
  endpoint: http://ollama.default.svc.cluster.local:11434/v1
```

## Related Resources

- [LanguageAgent](languageagent.md) - Reference models in agents
- [LanguageCluster](languagecluster.md) - Shared proxy architecture

## Examples

See [Examples](../getting-started/examples.md) for multi-model configurations.
