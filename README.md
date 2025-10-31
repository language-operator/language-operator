# language-operator

Kubernetes CRDs for spoken language automations.


## Architecture

The language operator will install the following CRDs:

| CRD               | Purpose                                   |
| ----------------- | ----------------------------------------- |
| `LanguageCluster` | A network-isolated environment for agents and tools |
| `LanguageAgent`   | Perform an arbitrary goal-directed task in perpetuity |
| `LanguageModel`   | A model configuration and access policy |
| `LanguageTool`    | MCP-compatible tool server (web search, etc.) |
| `LanguagePersona` | Reusable personality/behavior templates for agents |
| `LanguageClient`  | Connect and interact with a running agent |


## Example

Create a **personal-assistants** cluster:

```yaml
# cluster.yaml
apiVersion: langop.io/v1alpha1
kind: LanguageCluster
metadata:
  name: personal-assistants
spec:
  namespace: personal-assistants
```

Create a tool for searching the web:

```yaml
# web-tool.yaml
apiVersion: langop.io/v1alpha1
kind: LanguageTool
metadata:
  name: web-tool
spec:
  cluster: personal-assistants
  image: git.theryans.io/langop/web-tool:latest
  deploymentMode: sidecar  # Runs as sidecar, gets workspace access
```

Create a tool for email:

```yaml
# email-tool.yaml
apiVersion: langop.io/v1alpha1
kind: LanguageTool
metadata:
  name: email-tool
spec:
  cluster: personal-assistants
  image: git.theryans.io/langop/email-tool:latest
  deploymentMode: sidecar  # Runs as sidecar, gets workspace access
```

Create a model configuration for the agent:

```yaml
# model.yaml
apiVersion: langop.io/v1alpha1
kind: LanguageModel
metadata:
  name: mistralai-magistral-small-2506
spec:
  cluster: personal-assistants
  model:
    provider: openai-compatible
    model: mistralai/magistral-small-2506
    endpoint: http://my-on-prem-model.com/v1
    api_key: magistral
  proxy:
    image: git.theryans.io/langop/model:latest
```

Finally, define the agent and dependencies:

```yaml
# agent.yaml
apiVersion: langop.io/v1alpha1
kind: LanguageAgent
metadata:
  name: retrieve-daily-headlines
spec:
  cluster: personal-assistants
  image: git.theryans.io/langop/agent:latest
  toolRefs:
  - name: email-tool
  - name: web-tool
  modelRefs:
  - name: mistralai-magistral-small-2506
  instructions: |
    You are a helpful assistant designed to summarize world current events.
    Every morning at 6am US central, send a summary to james@theryans.io.
    Include a paragraph summary no more than 5 sentences.
    Include a bullet list of links with no more than 10 items.
  workspace:
    enabled: true
    size: 10Gi
    mountPath: /workspace
```

## Workspace Storage

Each agent can have a persistent workspace volume that acts as a shared whiteboard between the agent and its tools.

### How It Works

When `workspace.enabled: true`:
1. Operator creates a PersistentVolumeClaim for the agent
2. Agent container mounts the workspace at `/workspace` (or custom `mountPath`)
3. Sidecar tools (with `deploymentMode: sidecar`) also mount the same workspace
4. Agent and tools can read/write files to coordinate and persist state

### Use Cases

**Data persistence:** Agent remembers what it did yesterday
```bash
# Agent writes state
echo "2025-10-30: Sent headlines to user" >> /workspace/history.log

# Next run, agent reads history
cat /workspace/history.log
```

**Tool coordination:** Web tool scrapes, agent summarizes, email tool sends
```bash
# web-tool sidecar writes
curl https://news.ycombinator.com > /workspace/articles.html

# agent processes
# (LLM reads /workspace/articles.html, generates summary)

# agent writes summary for email-tool
echo "Summary: ..." > /workspace/email-body.txt

# email-tool sidecar reads and sends
mail -s "Daily News" user@example.com < /workspace/email-body.txt
```

### Tool Deployment Modes

**Sidecar mode** (workspace access):
```yaml
kind: LanguageTool
spec:
  deploymentMode: sidecar  # Deployed in agent pod, shares workspace
```

**Service mode** (shared, no workspace):
```yaml
kind: LanguageTool
spec:
  deploymentMode: service  # Deployed separately, called via HTTP
  replicas: 3              # Can scale independently
```

## Network Isolation

By default, all resources within a cluster can communicate with each other, but external access is denied. Individual agents, tools, and models can define egress rules to allow specific external endpoints.

### Example: Web Tool with External Access

```yaml
apiVersion: langop.io/v1alpha1
kind: LanguageTool
metadata:
  name: web-tool
spec:
  cluster: personal-assistants
  image: git.theryans.io/langop/web-tool:latest
  deploymentMode: sidecar
  egress:
  # Allow HTTPS to specific news sites
  - description: Allow news websites
    to:
      dns:
      - "news.ycombinator.com"
      - "*.cnn.com"
      - "*.bbc.com"
    ports:
    - port: 443
      protocol: TCP
```

### Example: Email Tool with SMTP Access

```yaml
apiVersion: langop.io/v1alpha1
kind: LanguageTool
metadata:
  name: email-tool
spec:
  cluster: personal-assistants
  image: git.theryans.io/langop/email-tool:latest
  deploymentMode: sidecar
  egress:
  # Allow SMTP to mail server
  - description: Allow SMTP to corporate mail server
    to:
      dns:
      - "smtp.company.com"
    ports:
    - port: 587
      protocol: TCP
```

### Example: Model Proxy with API Access

```yaml
apiVersion: langop.io/v1alpha1
kind: LanguageModel
metadata:
  name: gpt-4
spec:
  cluster: personal-assistants
  provider: openai
  modelName: gpt-4
  apiKeySecretRef:
    name: openai-credentials
  egress:
  # Allow HTTPS to OpenAI API
  - description: Allow OpenAI API access
    to:
      dns:
      - "api.openai.com"
    ports:
    - port: 443
      protocol: TCP
```

### Example: Agent with No External Access

```yaml
apiVersion: langop.io/v1alpha1
kind: LanguageAgent
metadata:
  name: internal-agent
spec:
  cluster: personal-assistants
  image: git.theryans.io/langop/agent:latest
  # No egress defined - can only talk to tools/models within cluster
```

### How It Works

The operator automatically creates Kubernetes NetworkPolicies:

1. **Default policy**: Allow all traffic within the cluster namespace, deny all external egress
2. **Per-resource policies**: For each resource with `egress` defined, create a NetworkPolicy allowing that specific external access
3. **DNS support**: DNS-based rules (like `*.cnn.com`) require a DNS-aware CNI. Otherwise, use CIDR blocks for IP-based restrictions.

## Reusable Personas

LanguagePersona allows you to define reusable personality templates that agents can reference. This promotes consistency across agents and makes it easy to update behavior across multiple agents at once.

### Example: Customer Support Persona

```yaml
apiVersion: langop.io/v1alpha1
kind: LanguagePersona
metadata:
  name: customer-support
spec:
  displayName: "Friendly Customer Support"
  description: "Empathetic and helpful customer service representative"

  systemPrompt: |
    You are a friendly and professional customer support representative.
    Your goal is to help customers solve their problems efficiently while
    maintaining a warm, empathetic tone.

  tone: friendly

  instructions:
  - "Always greet customers warmly"
  - "Listen carefully to understand the customer's issue"
  - "Provide clear, step-by-step solutions"
  - "Follow up to ensure the issue is resolved"
  - "Thank the customer for their patience"

  rules:
  - name: escalate-angry-customer
    condition: "when customer expresses frustration or anger"
    action: "acknowledge their feelings, apologize for the inconvenience, and offer to escalate to a supervisor if needed"
    priority: 10

  - name: verify-account
    condition: "when discussing account-specific information"
    action: "verify customer identity before sharing sensitive information"
    priority: 5

  examples:
  - input: "I've been waiting for my order for 3 weeks!"
    output: "I sincerely apologize for the delay with your order. I understand how frustrating this must be. Let me look into this right away and find out what's happening with your shipment."
    context: "Delayed order complaint"

  - input: "How do I reset my password?"
    output: "I'd be happy to help you reset your password! Here's what you need to do: 1) Go to the login page, 2) Click 'Forgot Password', 3) Enter your email address, 4) Check your email for the reset link. Let me know if you need any help with these steps!"
    context: "Password reset request"

  capabilities:
  - "Order tracking"
  - "Account management"
  - "Product information"
  - "Refund processing"

  limitations:
  - "Cannot modify orders already shipped"
  - "Cannot access payment card details"
  - "Cannot override company policies"

  responseFormat:
    type: markdown
    maxLength: 500
    includeSources: false

  toolPreferences:
    preferredTools:
    - order-lookup-tool
    - customer-db-tool
    strategy: balanced
    explainToolUse: true

  constraints:
    maxResponseTokens: 300
    maxToolCalls: 5
    responseTimeout: 30s
    blockedTopics:
    - "competitor comparisons"
    - "medical advice"
```

### Using a Persona in an Agent

```yaml
apiVersion: langop.io/v1alpha1
kind: LanguageAgent
metadata:
  name: support-agent-1
spec:
  cluster: customer-support-cluster
  image: git.theryans.io/langop/agent:latest

  # Reference the persona
  personaRef:
    name: customer-support

  # Agent-specific instructions (merged with persona)
  instructions: |
    You are assigned to the premium support queue.
    Prioritize response time and white-glove service.

  modelRefs:
  - name: gpt-4

  toolRefs:
  - name: order-lookup-tool
  - name: customer-db-tool
  - name: email-tool
```

### Persona Benefits

✅ **Consistency** - Same behavior across multiple agents
✅ **Reusability** - Define once, use many times
✅ **Centralized updates** - Update persona, all agents inherit changes
✅ **Versioning** - Track persona versions and roll back if needed
✅ **Inheritance** - Personas can inherit from parent personas
✅ **Validation** - Built-in validation checks for persona quality