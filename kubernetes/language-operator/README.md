# Language Operator

A production-grade Kubernetes operator for managing AI infrastructure including language models, tools, agents, personas, and client interfaces.

**API Group:** `langop.io/v1alpha1`

## Overview

The Language Operator provides Kubernetes-native management of AI infrastructure with five custom resource types:

| Resource | Description | Short Name |
|----------|-------------|------------|
| **LanguageTool** | Deploy MCP (Model Context Protocol) tool services | `ltool` |
| **LanguageModel** | Configure LLM providers with multi-region, load balancing, and failover | `lmodel` |
| **LanguageAgent** | Deploy autonomous goal-driven agents with safety guardrails | `lagent` |
| **LanguagePersona** | Define role-specific behaviors and knowledge sources | `lpersona` |
| **LanguageClient** | Deploy user-facing chat interfaces with auth and rate limiting | `lclient` |

## Project Structure

```
.
├── api/v1alpha1/              # CRD type definitions
│   ├── groupversion_info.go   # API group configuration (langop.io/v1alpha1)
│   ├── languagetool_types.go  # LanguageTool CRD
│   ├── languagemodel_types.go # LanguageModel CRD
│   ├── languageagent_types.go # LanguageAgent CRD
│   ├── languagepersona_types.go # LanguagePersona CRD
│   └── languageclient_types.go # LanguageClient CRD
├── cmd/                       # Main entry point (TODO)
│   └── main.go               # Operator bootstrap
├── controllers/               # Reconciliation logic (TODO)
│   ├── languagetool_controller.go
│   ├── languagemodel_controller.go
│   ├── languageagent_controller.go
│   ├── languagepersona_controller.go
│   └── languageclient_controller.go
├── config/                    # Kubernetes manifests
│   ├── crd/bases/            # Generated CRD YAML files
│   ├── rbac/                 # RBAC configuration
│   └── manager/              # Operator deployment
├── examples/                  # Example custom resources
│   ├── basic/                # Simple examples
│   ├── production/           # Production-ready configs
│   └── complete-stack/       # Full stack deployments
├── go.mod                     # Go module dependencies
├── go.sum
├── Makefile                   # Build, test, and deployment targets
└── hack/
    └── boilerplate.go.txt    # License header template
```

## Prerequisites

- **Go** 1.22 or later
- **Kubernetes** 1.24 or later
- **kubectl** configured to access your cluster
- **controller-gen** (installed automatically via Makefile)

## Quick Start

### 1. Install CRDs

```bash
# Generate and install CRDs
make manifests
make install
```

### 2. Run Locally (for development)

```bash
# Run the operator locally against your kubeconfig cluster
make run
```

### 3. Build and Deploy

```bash
# Build the operator binary
make build

# Build Docker image
make docker-build IMG=myregistry/language-operator:latest

# Push image
make docker-push IMG=myregistry/language-operator:latest

# Deploy to cluster
make deploy IMG=myregistry/language-operator:latest
```

### 4. Using Helm (Recommended)

See [../charts/language-operator/](../charts/language-operator/) for Helm chart installation.

```bash
cd ../charts/language-operator
make install
```

## Development

### Generate Code

The operator uses code generation for CRDs and DeepCopy methods:

```bash
# Generate CRD manifests
make manifests

# Generate DeepCopy methods
make generate

# Do both
make manifests generate
```

### Testing

```bash
# Run tests
make test

# Run with coverage
make test
cat cover.out
```

### Building

```bash
# Format code
make fmt

# Lint code
make vet

# Build binary
make build
./bin/manager --help
```

### Docker

```bash
# Build image
make docker-build IMG=myregistry/language-operator:v1.0.0

# Push image
make docker-push IMG=myregistry/language-operator:v1.0.0
```

## API Reference

### LanguageTool

Deploy MCP tool services as Kubernetes Deployments with full production controls.

**Features:**
- Standard Kubernetes deployment controls (replicas, resources, affinity)
- Health probes (liveness, readiness, startup)
- Service exposure (ClusterIP, NodePort, LoadBalancer)
- Pod Disruption Budgets
- Horizontal Pod Autoscaling
- Multi-region support

**Example:**

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

### LanguageModel

Configure LLM providers with enterprise features.

**Features:**
- Multi-provider support (OpenAI, Anthropic, Azure, Bedrock, Vertex AI)
- Load balancing (round-robin, latency-based, weighted)
- Automatic failover with fallback models
- Retry policies with exponential backoff
- Multi-region endpoints
- Response caching (memory, Redis, Memcached)
- Rate limiting and cost tracking
- Comprehensive metrics and observability

**Example:**

```yaml
apiVersion: langop.io/v1alpha1
kind: LanguageModel
metadata:
  name: gpt4-prod
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
  loadBalancing:
    strategy: latency
  costTracking:
    enabled: true
    inputTokenCost: 10
    outputTokenCost: 30
```

### LanguageAgent

Deploy autonomous goal-driven agents.

**Features:**
- Multiple execution modes (autonomous, interactive, scheduled, event-driven)
- Safety guardrails (tool call limits, approval requirements, content filtering)
- Memory management (ephemeral, Redis, Postgres, S3)
- Event-driven triggers (webhooks, Kubernetes events, message queues)
- Cron-based scheduling
- Comprehensive metrics and cost tracking

**Example:**

```yaml
apiVersion: langop.io/v1alpha1
kind: LanguageAgent
metadata:
  name: support-agent
spec:
  image: ghcr.io/based/language-agent:latest
  modelRefs:
  - name: gpt4-prod
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
    costLimitPerExecution: 100
```

### LanguagePersona

Define role-specific behaviors and knowledge.

**Features:**
- System prompts and instructions
- Conditional behavior rules with priorities
- Few-shot learning examples
- Tool usage preferences
- Knowledge sources (URLs, documents, databases, vector stores)
- Response formatting
- Usage analytics

**Example:**

```yaml
apiVersion: langop.io/v1alpha1
kind: LanguagePersona
metadata:
  name: senior-engineer
spec:
  displayName: "Senior Software Engineer"
  description: "Experienced engineer specializing in systems design"
  systemPrompt: "You are a senior software engineer with expertise in distributed systems, cloud architecture, and best practices."
  instructions:
  - "Always consider scalability and maintainability"
  - "Suggest testing strategies"
  - "Reference relevant documentation"
  tone: professional
  capabilities:
  - "code-review"
  - "architecture-design"
  - "debugging"
  toolPreferences:
    preferredTools:
    - code-search
    - documentation-lookup
```

### LanguageClient

Deploy user-facing chat interfaces.

**Features:**
- Multiple client types (web, API, CLI, Slack, Discord, Teams)
- Authentication (OAuth2, OIDC, SAML, LDAP, API keys)
- Role-based access control (RBAC)
- Session management (memory, Redis, Postgres, DynamoDB)
- Per-user rate limiting
- Content moderation
- UI customization
- Ingress and TLS configuration
- Comprehensive metrics

**Example:**

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
  - name: gpt4-prod
    role: primary
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

## Architecture

### Controller Pattern

Each CRD has a corresponding controller that:

1. **Watches** for changes to custom resources
2. **Reconciles** the desired state with actual state
3. **Updates** status with observed state
4. **Manages** owned Kubernetes resources (Deployments, Services, etc.)

```
┌─────────────────────────────────────────────────────────────┐
│                     Kubernetes API Server                    │
│                                                              │
│  ┌────────────────────────────────────────────────────────┐ │
│  │  Custom Resources (langop.io/v1alpha1)                 │ │
│  │  • LanguageTool  • LanguageModel  • LanguageAgent     │ │
│  │  • LanguagePersona  • LanguageClient                  │ │
│  └────────────────────────────────────────────────────────┘ │
│                           ↕                                  │
│  ┌────────────────────────────────────────────────────────┐ │
│  │  Language Operator Controllers                         │ │
│  │  ┌──────────┐ ┌──────────┐ ┌──────────┐ ┌──────────┐ │ │
│  │  │  Tool    │ │  Model   │ │  Agent   │ │  Client  │ │ │
│  │  │Controller│ │Controller│ │Controller│ │Controller│ │ │
│  │  └──────────┘ └──────────┘ └──────────┘ └──────────┘ │ │
│  └────────────────────────────────────────────────────────┘ │
│                           ↕                                  │
│  ┌────────────────────────────────────────────────────────┐ │
│  │  Native Kubernetes Resources                           │ │
│  │  • Deployments  • Services  • ConfigMaps  • Secrets   │ │
│  │  • Ingresses  • HPAs  • PDBs  • Jobs  • CronJobs     │ │
│  └────────────────────────────────────────────────────────┘ │
└─────────────────────────────────────────────────────────────┘
```

### Reconciliation Loop

```go
func (r *LanguageModelReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
    // 1. Fetch the resource
    model := &langopv1alpha1.LanguageModel{}
    if err := r.Get(ctx, req.NamespacedName, model); err != nil {
        return ctrl.Result{}, client.IgnoreNotFound(err)
    }

    // 2. Validate the resource
    if err := r.validateModel(model); err != nil {
        return ctrl.Result{}, err
    }

    // 3. Reconcile owned resources
    if err := r.reconcileConfigMap(ctx, model); err != nil {
        return ctrl.Result{}, err
    }

    // 4. Update status
    model.Status.Phase = "Ready"
    model.Status.ObservedGeneration = model.Generation
    if err := r.Status().Update(ctx, model); err != nil {
        return ctrl.Result{}, err
    }

    return ctrl.Result{}, nil
}
```

## Status Management

All CRDs follow the standard Kubernetes status pattern:

- **ObservedGeneration**: Tracks which generation was last processed
- **Conditions**: Standard condition types (Ready, Available, Progressing)
- **Phase**: High-level state (Pending, Running, Failed, Unknown)
- **Metrics**: Resource-specific metrics and statistics

## RBAC

The operator requires the following permissions:

```yaml
# Core resources
- apiGroups: [""]
  resources: [configmaps, secrets, services, pods]
  verbs: [get, list, watch, create, update, patch, delete]

# Apps resources
- apiGroups: [apps]
  resources: [deployments, statefulsets]
  verbs: [get, list, watch, create, update, patch, delete]

# Custom resources
- apiGroups: [langop.io]
  resources: [languagetools, languagemodels, languageagents, languagepersonas, languageclients]
  verbs: [get, list, watch, create, update, patch, delete]

# Custom resource status
- apiGroups: [langop.io]
  resources: [languagetools/status, languagemodels/status, ...]
  verbs: [get, update, patch]
```

## Makefile Targets

### Development

- `make manifests` - Generate CRD manifests
- `make generate` - Generate DeepCopy methods
- `make fmt` - Format code
- `make vet` - Lint code
- `make test` - Run tests

### Build

- `make build` - Build operator binary
- `make docker-build IMG=<image>` - Build Docker image
- `make docker-push IMG=<image>` - Push Docker image

### Deployment

- `make install` - Install CRDs to cluster
- `make uninstall` - Remove CRDs from cluster
- `make deploy` - Deploy operator to cluster
- `make undeploy` - Remove operator from cluster
- `make helm-crds` - Copy CRDs to Helm chart

### Utilities

- `make help` - Show available targets
- `make run` - Run locally against cluster

## Configuration

The operator can be configured via command-line flags:

```bash
./bin/manager \
  --leader-elect \
  --leader-elect-lease-duration=15s \
  --metrics-bind-address=:8443 \
  --health-probe-bind-address=:8081 \
  --zap-log-level=info \
  --zap-encoder=json
```

## Monitoring

### Metrics

The operator exposes Prometheus metrics at `:8443/metrics`:

- Controller metrics (reconciliation rate, errors, duration)
- Workqueue metrics (depth, latency, retries)
- Custom resource counts and status

### Health Checks

- **Liveness**: `:8081/healthz` - Operator is alive
- **Readiness**: `:8081/readyz` - Operator is ready to serve

## Current Status

### ✅ Completed

- API type definitions for all 5 CRDs
- CRD manifest generation
- Go module configuration
- Makefile with build/deploy targets
- Helm chart integration

### 🚧 TODO

- [ ] Implement controller reconciliation logic
- [ ] Add cmd/main.go bootstrap code
- [ ] Create comprehensive unit tests
- [ ] Add integration tests
- [ ] Create example manifests
- [ ] Build and publish Docker images
- [ ] Add CI/CD pipeline
- [ ] Write API documentation

## Contributing

### Adding a New Controller

1. Create controller file in `controllers/`
2. Implement `Reconcile()` method
3. Set up watches for owned resources
4. Update status subresource
5. Add RBAC markers
6. Update tests

### Testing

```bash
# Unit tests
go test ./api/v1alpha1/... -v

# Controller tests
go test ./controllers/... -v

# Integration tests (requires cluster)
make test
```

## Resources

- [Kubebuilder Book](https://book.kubebuilder.io/)
- [controller-runtime](https://github.com/kubernetes-sigs/controller-runtime)
- [Operator Pattern](https://kubernetes.io/docs/concepts/extend-kubernetes/operator/)
- [Helm Chart](../charts/language-operator/)

## License

Apache License 2.0 - See LICENSE file for details.
