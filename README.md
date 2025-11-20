# Language Operator

Language Operator is a Kubernetes operator that turns natural language into autonomous agents.  

It is under active development and incomplete.

```bash
# Add repository:
helm repo add language-operator https://language-operator.github.io/language-operator
helm repo update

# Install in kube-system:
helm install language-operator language-operator/language-operator
```

## Vision

Modify Kubernetes to promote AI tools, models and agents as first-class resources.

| CRD             | Purpose                                     |
| --------------- | ------------------------------------------- |
| LanguageCluster | A group of related models, tools and agents |
| LanguageModel   | An on-prem model or cloud provider with rate limiting and cost controls |
| LanguageTool    | An MCP tool for agents; runs as sidecar or deployment |
| LanguageAgent   | A goal-directed task that uses models, personas and tools |
| LanguagePersona | A free-form description of a role, job, or preferences |


### LanguageModel

Deploy a [litellm](https://docs.litellm.ai/docs/simple_proxy) proxy with rate limiting and cost control:

```yaml
apiVersion: langop.io/v1alpha1
kind: LanguageModel
metadata:
  name: gpt-4-turbo
spec:
  provider: openai
  modelName: gpt-4-turbo-preview
  apiKeySecretRef:
    name: openai-credentials
    key: api-key
  rateLimits:
    requestsPerMinute: 100
    tokensPerMinute: 100000
  costTracking:
    enabled: true
    inputTokenCost: 0.01
    outputTokenCost: 0.03
```

### LanguageTool

Deploy an MCP tool as a standalone service or sidecar for shared workspaces:

```yaml
apiVersion: langop.io/v1alpha1
kind: LanguageTool
metadata:
  name: web-search
spec:
  image: gchr.io/language-operator/web-tool:latest
  deploymentMode: service  # or 'sidecar' for shared workspace access
  type: mcp
```

### LanguageCluster

A network-isolated <sup>1</sup> environment for agents and tools:

```yaml
apiVersion: langop.io/v1alpha1
kind: LanguageCluster
metadata:
  name: production
spec:
  domain: agents.example.com
```

### LanguagePersona

A free-form description of a person, role, or behavior.

```yaml
apiVersion: langop.io/v1alpha1
kind: LanguagePersona
metadata:
  name: helpful-assistant
spec:
  displayName: "Helpful Assistant"
  description: "A friendly AI assistant for customer support"
  systemPrompt: "You are a helpful, patient customer support agent."
  tone: friendly
  language: en
  capabilities:
    - "Answer questions"
    - "Provide guidance"
    - "Escalate to human when needed"
  limitations:
    - "Cannot access customer payment information"
    - "Cannot make refunds without approval"
```

### LanguageAgent

Self-synthesizing agents <sup>2</sup> from natural language instructions:

```yaml
apiVersion: langop.io/v1alpha1
kind: LanguageAgent
metadata:
  name: customer-support-bot
spec:
  modelRefs:
    - name: gpt-4-turbo
  toolRefs:
    - name: web-search
  personaRefs:
    - name: helpful-assistant
  instructions: |
    You are a customer support agent. Answer customer questions
    using the available tools and escalate complex issues.
  workspace:
    enabled: true
    size: 10Gi
```

<sup>1</sup> requires CNI like Cilium (recommended)
<sup>2</sup> quality of synthesis is model dependent
