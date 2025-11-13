# Language Operator Helm Chart

A Kubernetes operator that turns natural language into autonomous agents.

## Overview

The Language Operator provides Custom Resources for deploying and managing AI infrastructure:

- **LanguageAgent** - Deploy autonomous goal-driven agents with safety guardrails
- **LanguageModel** - Configure LLM providers with load balancing and failover
- **LanguageTool** - Deploy MCP (Model Context Protocol) tool services
- **LanguagePersona** - Define role-specific behaviors and knowledge sources
- **LanguageCluster** - Orchestrate multi-agent systems with networking and RBAC

## Prerequisites

- Kubernetes 1.24+
- Helm 3.8+

## Installation

### Quick Start

```bash
helm install language-operator ./chart \
  --namespace language-operator-system \
  --create-namespace
```

### Install with custom values

```bash
helm install language-operator ./chart \
  --namespace language-operator-system \
  --create-namespace \
  --values my-values.yaml
```

## Configuration

### Core Settings

| Parameter | Description | Default |
|-----------|-------------|---------|
| `replicaCount` | Number of operator replicas | `2` |
| `image.repository` | Operator image repository | `ghcr.io/language-operator/language-operator` |
| `image.pullPolicy` | Image pull policy | `IfNotPresent` |
| `image.tag` | Operator image tag | `""` (uses appVersion) |

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
| `autoscaling.enabled` | Enable HPA | `false` |
| `autoscaling.minReplicas` | Minimum replicas | `2` |
| `autoscaling.maxReplicas` | Maximum replicas | `10` |

### Leader Election

| Parameter | Description | Default |
|-----------|-------------|---------|
| `config.leaderElection.enabled` | Enable leader election | `true` |
| `config.leaderElection.leaseDuration` | Lease duration | `15s` |
| `config.leaderElection.renewDeadline` | Renew deadline | `10s` |
| `config.leaderElection.retryPeriod` | Retry period | `2s` |

### Controller Configuration

| Parameter | Description | Default |
|-----------|-------------|---------|
| `config.controller.concurrency` | Concurrent reconcilers | `5` |
| `config.controller.syncPeriod` | Sync period | `10m` |

### Self-Healing Synthesis

| Parameter | Description | Default |
|-----------|-------------|---------|
| `config.selfHealing.enabled` | Enable self-healing for failed agents | `true` |
| `config.selfHealing.maxAttempts` | Maximum self-healing attempts | `5` |
| `config.selfHealing.minBackoff` | Minimum backoff between attempts | `1m` |
| `config.selfHealing.maxBackoff` | Maximum backoff between attempts | `16m` |
| `config.selfHealing.failureThreshold` | Failures required to trigger healing | `2` |
| `config.selfHealing.sanitizeLogs` | Redact secrets from logs | `true` |
| `config.selfHealing.maxLogLines` | Max log lines to extract | `100` |

### Agent Security

| Parameter | Description | Default |
|-----------|-------------|---------|
| `config.agentSecurity.podSecurityContext.runAsUser` | Agent pod user ID | `1000` |
| `config.agentSecurity.podSecurityContext.fsGroup` | Agent pod group ID | `101` |
| `config.agentSecurity.containerSecurityContext.readOnlyRootFilesystem` | Read-only root filesystem | `true` |
| `config.agentSecurity.tmpfsVolumes.enabled` | Enable tmpfs volumes for writable paths | `true` |

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

### OpenTelemetry

| Parameter | Description | Default |
|-----------|-------------|---------|
| `opentelemetry.enabled` | Enable distributed tracing | `false` |
| `opentelemetry.endpoint` | OTLP gRPC endpoint | `""` |
| `opentelemetry.sampling.rate` | Trace sampling rate (0.0-1.0) | `1.0` |
| `opentelemetry.resourceAttributes.environment` | Environment name | `production` |
| `opentelemetry.resourceAttributes.cluster` | Cluster name | `main` |

### Agent Telemetry

| Parameter | Description | Default |
|-----------|-------------|---------|
| `agentTelemetry.endpoint` | OTLP endpoint override for agents | `""` (inherits from operator) |
| `agentTelemetry.samplingRate` | Sampling rate override for agents | `null` (inherits from operator) |

### Persona Library

| Parameter | Description | Default |
|-----------|-------------|---------|
| `personaLibrary.enabled` | Install built-in persona ConfigMaps | `true` |
| `personaLibrary.personas` | List of personas to install | See values.yaml |

### Webhook

| Parameter | Description | Default |
|-----------|-------------|---------|
| `config.webhook.enabled` | Enable validation/mutation webhooks | `true` |
| `config.webhook.port` | Webhook server port | `9443` |

### RBAC

| Parameter | Description | Default |
|-----------|-------------|---------|
| `rbac.create` | Create RBAC resources | `true` |
| `serviceAccount.create` | Create service account | `true` |
| `serviceAccount.name` | Service account name | `""` (generated) |

### CRD Management

| Parameter | Description | Default |
|-----------|-------------|---------|
| `crds.install` | Install CRDs with chart | `true` |
| `crds.keep` | Keep CRDs on uninstall | `true` |

## Usage Examples

### Deploy a Language Agent

```yaml
apiVersion: langop.io/v1alpha1
kind: LanguageAgent
metadata:
  name: example-agent
  namespace: default
spec:
  image: ghcr.io/language-operator/language-agent:latest
  goal: "Answer questions about Kubernetes"
  modelRefs:
  - name: claude-sonnet
    role: primary
  toolRefs:
  - name: kubectl-tool
  executionMode: interactive
```

### Deploy a Language Model

```yaml
apiVersion: langop.io/v1alpha1
kind: LanguageModel
metadata:
  name: claude-sonnet
spec:
  provider: anthropic
  modelName: claude-sonnet-4-5-20250929
  apiKeySecretRef:
    name: anthropic-credentials
    key: api-key
  rateLimits:
    requestsPerMinute: 50
```

### Deploy an MCP Tool

```yaml
apiVersion: langop.io/v1alpha1
kind: LanguageTool
metadata:
  name: kubectl-tool
spec:
  image: ghcr.io/language-operator/mcp-kubectl:latest
  type: mcp
  port: 8080
  replicas: 2
```

### Deploy a Language Cluster

```yaml
apiVersion: langop.io/v1alpha1
kind: LanguageCluster
metadata:
  name: dev-cluster
  namespace: default
spec:
  domain: example.com
  ingressConfig:
    tls:
      enabled: true
      issuerRef:
        name: letsencrypt-prod
        kind: ClusterIssuer
```

## Upgrading

```bash
helm upgrade language-operator ./chart \
  --namespace language-operator-system \
  --values my-values.yaml
```

Check the [CHANGELOG](../CHANGELOG.md) for breaking changes between versions.

## Uninstalling

```bash
helm uninstall language-operator --namespace language-operator-system
```

**Note:** CRDs are kept by default. To remove them:

```bash
kubectl delete crd languageagents.langop.io
kubectl delete crd languagemodels.langop.io
kubectl delete crd languagetools.langop.io
kubectl delete crd languagepersonas.langop.io
kubectl delete crd languageclusters.langop.io
```

## Development

### Local Installation

```bash
# Build and load image locally
cd src
make docker-build IMG=localhost:5000/language-operator:dev

# Install chart with local image
helm install language-operator ./chart \
  --namespace language-operator-system \
  --create-namespace \
  --set image.repository=localhost:5000/language-operator \
  --set image.tag=dev \
  --set image.pullPolicy=Always
```

### Debugging

```bash
# View operator logs
kubectl logs -n language-operator-system -l app.kubernetes.io/name=language-operator -f

# Check operator status
kubectl get pods -n language-operator-system

# Verify CRDs are installed
kubectl get crds | grep langop.io
```

## Architecture

The operator manages five Custom Resource Definitions:

- **LanguageAgent** - Creates Deployments for autonomous agents
- **LanguageModel** - Manages LLM provider configurations
- **LanguageTool** - Creates Services and Deployments for MCP tools
- **LanguagePersona** - Stores persona definitions in ConfigMaps
- **LanguageCluster** - Orchestrates multiple agents with isolation

Each controller reconciles its resources independently with leader election for high availability.

## Security

### Operator Security

- Runs as non-root user (65532)
- Read-only root filesystem
- Drops all capabilities
- Seccomp profile enabled

### Agent Security

- Runs as non-root user (1000)
- Read-only root filesystem with tmpfs mounts
- Network policies for cluster isolation
- RBAC with least privilege
- Secret redaction in logs

## Troubleshooting

### Operator not starting

Check events and logs:
```bash
kubectl describe pod -n language-operator-system -l app.kubernetes.io/name=language-operator
kubectl logs -n language-operator-system -l app.kubernetes.io/name=language-operator
```

### CRDs not installing

Ensure `crds.install: true` in values and check for conflicts:
```bash
kubectl get crds | grep langop.io
```

### Webhook certificate issues

The operator generates self-signed certificates. Check webhook configuration:
```bash
kubectl get validatingwebhookconfigurations
kubectl get mutatingwebhookconfigurations
```

## Contributing

See [CONTRIBUTING.md](../CONTRIBUTING.md) for development guidelines.

## License

See [LICENSE](../LICENSE) for details.
