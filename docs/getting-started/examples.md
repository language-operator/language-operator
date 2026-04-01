# Examples

Common deployment patterns for Language Operator resources.

## Example Configurations

### Multi-Model Agent

Agent that can choose between multiple models:

```yaml
apiVersion: langop.io/v1alpha1
kind: LanguageAgent
metadata:
  name: multi-model-agent
spec:
  image: your-agent:latest
  models:
    - name: claude-sonnet  # Fast, balanced
    - name: claude-opus    # Highest quality
    - name: claude-haiku   # Fastest, cheapest
  instructions: |
    You have access to multiple models.
    Use haiku for simple tasks, sonnet for most work, opus for complex reasoning.
```

The operator sets `LLM_MODEL=claude-sonnet,claude-opus,claude-haiku` and `MODEL_ENDPOINTS=http://gateway.<namespace>.svc.cluster.local:8000` so your agent can choose.

### Agent with Persona

Reusable behavioral templates:

```yaml
apiVersion: langop.io/v1alpha1
kind: LanguagePersona
metadata:
  name: helpful-assistant
spec:
  tone: "professional and friendly"
  personality: "curious, patient, always cites sources and asks clarifying questions when ambiguous"
  expertise: "general-purpose assistant with strong research and writing skills"
---
apiVersion: langop.io/v1alpha1
kind: LanguageAgent
metadata:
  name: assistant-agent
spec:
  image: your-agent:latest
  models:
    - name: claude-sonnet
  persona: helpful-assistant
```

The persona is injected into `/etc/agent/config.yaml` in the agent pod.

### Agent with MCP Tools

Add web search capability:

```yaml
apiVersion: langop.io/v1alpha1
kind: LanguageTool
metadata:
  name: web-search
spec:
  image: mcp/brave-search:latest
  deployment:
    env:
      - name: BRAVE_API_KEY
        valueFrom:
          secretKeyRef:
            name: brave-api-key
            key: api-key
---
apiVersion: langop.io/v1alpha1
kind: LanguageAgent
metadata:
  name: research-agent
spec:
  image: your-agent:latest
  models:
    - name: claude-sonnet
  tools:
    - name: web-search
```

The operator injects `MCP_SERVERS=http://web-search:8080` into the agent.

### Custom Model Provider

Use any OpenAI-compatible endpoint:

```yaml
apiVersion: langop.io/v1alpha1
kind: LanguageModel
metadata:
  name: ollama-llama
spec:
  provider: openai-compatible
  modelName: llama3.2
  endpoint: http://ollama.default.svc.cluster.local:11434/v1
  apiKeySecretRef:
    name: ollama-key  # Can be empty for local models
    key: api-key
```

### Azure OpenAI

```yaml
apiVersion: langop.io/v1alpha1
kind: LanguageModel
metadata:
  name: azure-gpt4
spec:
  provider: azure
  modelName: gpt-4
  endpoint: https://your-resource.openai.azure.com
  apiKeySecretRef:
    name: azure-credentials
    key: api-key
```

## Next Steps

- [CRD Reference](../api/overview.md) - Full API specification
- [Architecture](../architecture/overview.md) - How it all works
- [Development Guide](../development/setup.md) - Build your own agents
