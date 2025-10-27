# Language Operator Helm Chart

A Kubernetes operator for managing language models, tools, agents, personas, and clients.

## Overview

The Language Operator provides production-grade Kubernetes Custom Resources for deploying and managing AI infrastructure:

- **LanguageTool** - Deploy MCP (Model Context Protocol) tool services
- **LanguageModel** - Configure LLM providers with load balancing, failover, and cost tracking
- **LanguageAgent** - Deploy autonomous goal-driven agents with safety guardrails
- **LanguagePersona** - Define role-specific behaviors and knowledge sources
- **LanguageClient** - Deploy user-facing chat interfaces with authentication and rate limiting

## Prerequisites

- Kubernetes 1.24+
- Helm 3.8+
- (Optional) Prometheus Operator for metrics collection

## Installation

### Add the Helm repository

```bash
helm repo add language-operator https://github.com/based/language-operator
helm repo update
```

### Install the operator

```bash
helm install language-operator language-operator/language-operator \
  --namespace language-operator-system \
  --create-namespace
```

### Install with custom values

```bash
helm install language-operator language-operator/language-operator \
  --namespace language-operator-system \
  --create-namespace \
  --values my-values.yaml
```

## Configuration

The following table lists the configurable parameters of the Language Operator chart and their default values.

### Basic Configuration

| Parameter | Description | Default |
|-----------|-------------|---------|
| `replicaCount` | Number of operator replicas | `2` |
| `image.repository` | Operator image repository | `ghcr.io/based/language-operator` |
| `image.pullPolicy` | Image pull policy | `IfNotPresent` |
| `image.tag` | Operator image tag | `""` (uses appVersion) |

### Service Account

| Parameter | Description | Default |
|-----------|-------------|---------|
| `serviceAccount.create` | Create service account | `true` |
| `serviceAccount.annotations` | Service account annotations | `{}` |
| `serviceAccount.name` | Service account name | `""` (generated) |

### RBAC

| Parameter | Description | Default |
|-----------|-------------|---------|
| `rbac.create` | Create RBAC resources | `true` |

### Resources

| Parameter | Description | Default |
|-----------|-------------|---------|
| `resources.limits.cpu` | CPU limit | `500m` |
| `resources.limits.memory` | Memory limit | `512Mi` |
| `resources.requests.cpu` | CPU request | `100m` |
| `resources.requests.memory` | Memory request | `128Mi` |

### High Availability

| Parameter | Description | Default |
|-----------|-------------|---------|
| `affinity` | Pod affinity rules | Anti-affinity preferred |
| `podDisruptionBudget.enabled` | Enable PDB | `true` |
| `podDisruptionBudget.minAvailable` | Minimum available pods | `1` |

### Autoscaling

| Parameter | Description | Default |
|-----------|-------------|---------|
| `autoscaling.enabled` | Enable HPA | `false` |
| `autoscaling.minReplicas` | Minimum replicas | `2` |
| `autoscaling.maxReplicas` | Maximum replicas | `10` |
| `autoscaling.targetCPUUtilizationPercentage` | Target CPU % | `80` |
| `autoscaling.targetMemoryUtilizationPercentage` | Target memory % | `80` |

### Leader Election

| Parameter | Description | Default |
|-----------|-------------|---------|
| `config.leaderElection.enabled` | Enable leader election | `true` |
| `config.leaderElection.leaseDuration` | Lease duration | `15s` |
| `config.leaderElection.renewDeadline` | Renew deadline | `10s` |
| `config.leaderElection.retryPeriod` | Retry period | `2s` |

### Logging

| Parameter | Description | Default |
|-----------|-------------|---------|
| `config.logging.level` | Log level (debug, info, warn, error) | `info` |
| `config.logging.format` | Log format (json, text) | `json` |
| `config.logging.development` | Development mode | `false` |

### Monitoring

| Parameter | Description | Default |
|-----------|-------------|---------|
| `monitoring.serviceMonitor.enabled` | Create ServiceMonitor | `false` |
| `monitoring.serviceMonitor.interval` | Scrape interval | `30s` |
| `monitoring.serviceMonitor.scrapeTimeout` | Scrape timeout | `10s` |

### Controller

| Parameter | Description | Default |
|-----------|-------------|---------|
| `config.controller.concurrency` | Concurrent reconcilers | `5` |
| `config.controller.syncPeriod` | Sync period | `10m` |

### Webhook (Future)

| Parameter | Description | Default |
|-----------|-------------|---------|
| `config.webhook.enabled` | Enable webhooks | `false` |
| `config.webhook.port` | Webhook port | `9443` |

## Usage Examples

### Deploy a Language Model

```yaml
apiVersion: langop.io/v1alpha1
kind: LanguageModel
metadata:
  name: gpt4
spec:
  provider: openai
  modelName: gpt-4-turbo
  apiKeySecretRef:
    name: openai-credentials
    key: api-key
  rateLimits:
    requestsPerMinute: 100
    tokensPerMinute: 150000
  fallbacks:
  - modelName: gpt-3.5-turbo
    conditions:
    - type: rate-limit
    - type: unavailable
```

### Deploy an MCP Tool

```yaml
apiVersion: langop.io/v1alpha1
kind: LanguageTool
metadata:
  name: web-search
spec:
  image: ghcr.io/based/mcp-web-search:latest
  type: http
  port: 8080
  replicas: 3
  resources:
    requests:
      cpu: 100m
      memory: 128Mi
    limits:
      cpu: 500m
      memory: 512Mi
```

### Deploy an Agent

```yaml
apiVersion: langop.io/v1alpha1
kind: LanguageAgent
metadata:
  name: customer-support
spec:
  image: ghcr.io/based/language-agent:latest
  modelRefs:
  - name: gpt4
    role: primary
  toolRefs:
  - name: web-search
  - name: knowledge-base
  personaRef:
    name: friendly-support
  goal: "Provide helpful customer support"
  executionMode: interactive
  safetyConfig:
    maxToolCallsPerIteration: 10
    requireApproval:
    - email-sender
    - payment-processor
```

### Deploy a Client Interface

```yaml
apiVersion: langop.io/v1alpha1
kind: LanguageClient
metadata:
  name: web-chat
spec:
  image: ghcr.io/based/language-client-web:latest
  type: web
  replicas: 3
  modelRefs:
  - name: gpt4
    role: primary
  toolRefs:
  - name: web-search
  sessionConfig:
    backend: redis
    ttl: 24h
  authentication:
    enabled: true
    providers:
    - name: google
      type: oauth2
  rateLimiting:
    requestsPerMinute: 20
    tokensPerDay: 100000
  ingress:
    enabled: true
    hosts:
    - chat.example.com
```

## Upgrading

### From 0.1.x to 0.2.x

Check the [CHANGELOG](../../CHANGELOG.md) for breaking changes.

```bash
helm upgrade language-operator language-operator/language-operator \
  --namespace language-operator-system \
  --values my-values.yaml
```

## Uninstalling

```bash
helm uninstall language-operator --namespace language-operator-system
```

**Note:** CRDs are not removed by default. To remove them:

```bash
kubectl delete crd languagetools.langop.io
kubectl delete crd languagemodels.langop.io
kubectl delete crd languageagents.langop.io
kubectl delete crd languagepersonas.langop.io
kubectl delete crd languageclients.langop.io
```

## Development

### Building from source

```bash
cd kubernetes/language-operator
make docker-build docker-push IMG=myregistry/language-operator:latest
```

### Installing locally built chart

```bash
helm install language-operator ./kubernetes/charts/language-operator \
  --namespace language-operator-system \
  --create-namespace \
  --set image.repository=myregistry/language-operator \
  --set image.tag=latest
```

## Contributing

See [CONTRIBUTING.md](../../CONTRIBUTING.md) for guidelines.

## License

See [LICENSE](../../LICENSE) for details.
