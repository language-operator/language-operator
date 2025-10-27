# Language Operator for Kubernetes

A production-grade Kubernetes operator for managing AI infrastructure including language models, tools, agents, personas, and client interfaces.

## What We've Built

### 1. Custom Resource Definitions (CRDs)

Five comprehensive CRDs have been created at [`language-operator/api/v1alpha1/`](language-operator/api/v1alpha1/):

- **[LanguageTool](language-operator/api/v1alpha1/languagetool_types.go)** - Deploy MCP (Model Context Protocol) tool services with full Kubernetes deployment controls
- **[LanguageModel](language-operator/api/v1alpha1/languagemodel_types.go)** - Configure LLM providers with multi-region support, load balancing, fallbacks, and cost tracking
- **[LanguageAgent](language-operator/api/v1alpha1/languageagent_types.go)** - Deploy autonomous goal-driven agents with execution modes, safety guardrails, and memory management
- **[LanguagePersona](language-operator/api/v1alpha1/languagepersona_types.go)** - Define role-specific behaviors with conditional rules, tool preferences, and knowledge sources
- **[LanguageClient](language-operator/api/v1alpha1/languageclient_types.go)** - Deploy user-facing chat interfaces with authentication, rate limiting, and session management

### 2. Helm Chart

A complete Helm chart at [`charts/language-operator/`](charts/language-operator/) with:

- Production-ready deployment configuration
- High availability (2 replicas, pod anti-affinity)
- Security hardening (non-root, read-only filesystem, dropped capabilities)
- RBAC with minimal required permissions
- Optional metrics (Prometheus ServiceMonitor)
- Optional autoscaling (HPA)
- Pod Disruption Budgets
- Leader election for HA
- Comprehensive documentation

### 3. Generated CRD Manifests

CRD YAML files have been generated and are available at:
- [`language-operator/config/crd/bases/`](language-operator/config/crd/bases/)
- [`charts/language-operator/crds/`](charts/language-operator/crds/)

## Project Structure

```
kubernetes/
├── language-operator/           # Operator source code
│   ├── api/v1alpha1/           # CRD type definitions
│   │   ├── groupversion_info.go
│   │   ├── languagetool_types.go
│   │   ├── languagemodel_types.go
│   │   ├── languageagent_types.go
│   │   ├── languagepersona_types.go
│   │   └── languageclient_types.go
│   ├── config/
│   │   ├── crd/bases/          # Generated CRD YAML manifests
│   │   ├── rbac/               # RBAC configuration
│   │   └── manager/            # Manager deployment
│   ├── examples/               # Example manifests
│   │   ├── basic/              # Simple examples
│   │   ├── production/         # Production-ready examples
│   │   └── complete-stack/     # Full stack deployments
│   ├── go.mod                  # Go dependencies
│   ├── go.sum
│   ├── Makefile                # Build and deployment tasks
│   └── hack/
│       └── boilerplate.go.txt  # License header
└── charts/
    └── language-operator/       # Helm chart
        ├── Chart.yaml
        ├── values.yaml
        ├── README.md
        ├── templates/          # Helm templates
        │   ├── _helpers.tpl
        │   ├── serviceaccount.yaml
        │   ├── clusterrole.yaml
        │   ├── clusterrolebinding.yaml
        │   ├── deployment.yaml
        │   ├── service.yaml
        │   ├── servicemonitor.yaml
        │   ├── poddisruptionbudget.yaml
        │   ├── hpa.yaml
        │   └── NOTES.txt
        └── crds/               # CRD manifests (copied from config/crd/bases/)
            ├── langop.io_languagetools.yaml
            ├── langop.io_languagemodels.yaml
            ├── langop.io_languageagents.yaml
            ├── langop.io_languagepersonas.yaml
            └── langop.io_languageclients.yaml
```

## Quick Start

### Install the Operator

```bash
# Install using Helm
helm install language-operator ./charts/language-operator \
  --namespace language-operator-system \
  --create-namespace
```

### Verify Installation

```bash
# Check operator pods
kubectl get pods -n language-operator-system

# Check CRDs
kubectl get crds | grep langop.io
```

## Development

### Prerequisites

- Go 1.22+
- Kubernetes 1.24+
- kubectl
- Helm 3.8+
- controller-gen (installed automatically via Makefile)

### Build and Generate

```bash
cd language-operator

# Generate CRD manifests
make manifests

# Generate DeepCopy methods
make generate

# Build the operator binary
make build

# Run tests
make test

# Build Docker image
make docker-build IMG=myregistry/language-operator:latest

# Copy CRDs to Helm chart
make helm-crds
```

### Install CRDs Only

```bash
# Install CRDs without the operator
cd language-operator
make install

# Or using kubectl
kubectl apply -f config/crd/bases/
```

## Key Features

### Production-Grade CRDs

All CRDs include:

- ✅ Comprehensive validation using kubebuilder markers
- ✅ Status subresources with conditions and metrics
- ✅ ObservedGeneration pattern for change tracking
- ✅ Multi-region support
- ✅ High availability configurations
- ✅ Cost tracking and observability
- ✅ Security and RBAC integration
- ✅ kubectl printcolumn annotations for useful CLI output
- ✅ Short names for easy kubectl usage

### Helm Chart Features

- ✅ Production-ready defaults
- ✅ Secure by default (non-root, read-only FS, minimal caps)
- ✅ High availability with leader election
- ✅ Prometheus metrics integration
- ✅ Horizontal Pod Autoscaling
- ✅ Pod Disruption Budgets
- ✅ Comprehensive RBAC
- ✅ Configurable via values.yaml (40+ parameters)

## Next Steps

### To Make This Fully Functional

1. **Implement Controllers** - Write reconciliation logic for each CRD in Go
2. **Build Container Images** - Create Dockerfile and build operator image
3. **Create Examples** - Add example manifests in `examples/` directory
4. **Add Tests** - Unit and integration tests for controllers
5. **CI/CD Pipeline** - Automate building, testing, and releasing
6. **Documentation** - API reference docs and usage guides

### Example Use Cases to Implement

1. **Customer Support Bot**
   - LanguageModel (GPT-4)
   - LanguageTool (knowledge base search, ticket system)
   - LanguagePersona (friendly support agent)
   - LanguageAgent (autonomous ticket handler)
   - LanguageClient (web chat interface)

2. **Code Assistant**
   - LanguageModel (Claude or GPT-4)
   - LanguageTool (code search, linter, testing tools)
   - LanguagePersona (senior engineer)
   - LanguageClient (IDE plugin interface)

3. **Multi-Region LLM Gateway**
   - LanguageModel with multi-region endpoints
   - Load balancing across regions
   - Cost tracking and optimization
   - Automatic failover

## Architecture

The Language Operator follows the Kubernetes Operator Pattern:

```
┌─────────────────────────────────────────────────────────────┐
│                     Kubernetes Cluster                       │
│                                                              │
│  ┌────────────────────────────────────────────────────────┐ │
│  │  Language Operator (Deployment)                        │ │
│  │  ┌──────────────┐  ┌──────────────┐  ┌─────────────┐  │ │
│  │  │  Tool        │  │  Model       │  │  Agent      │  │ │
│  │  │  Controller  │  │  Controller  │  │  Controller │  │ │
│  │  └──────────────┘  └──────────────┘  └─────────────┘  │ │
│  │  ┌──────────────┐  ┌──────────────┐                   │ │
│  │  │  Persona     │  │  Client      │                   │ │
│  │  │  Controller  │  │  Controller  │                   │ │
│  │  └──────────────┘  └──────────────┘                   │ │
│  └────────────────────────────────────────────────────────┘ │
│                           ↓ watches/reconciles              │
│  ┌────────────────────────────────────────────────────────┐ │
│  │  Custom Resources                                      │ │
│  │  • LanguageTool  • LanguageModel  • LanguageAgent     │ │
│  │  • LanguagePersona  • LanguageClient                  │ │
│  └────────────────────────────────────────────────────────┘ │
│                           ↓ creates/manages                 │
│  ┌────────────────────────────────────────────────────────┐ │
│  │  Native Kubernetes Resources                           │ │
│  │  • Deployments  • Services  • Ingresses               │ │
│  │  • ConfigMaps  • Secrets  • HPAs  • PDBs              │ │
│  └────────────────────────────────────────────────────────┘ │
└─────────────────────────────────────────────────────────────┘
```

## API Group

All resources are in the API group `langop.io/v1alpha1`.

Example kubectl commands:

```bash
# List all LanguageModels
kubectl get languagemodels
kubectl get lmodel  # short name

# Describe a LanguageTool
kubectl describe languagetool web-search
kubectl describe ltool web-search  # short name

# Get LanguageAgent status
kubectl get languageagent customer-support -o yaml
kubectl get lagent customer-support -o yaml  # short name

# Watch LanguageClient pods
kubectl get languageclient web-chat
kubectl get lclient web-chat  # short name
```

## Contributing

This is a work in progress. To contribute:

1. Implement controllers for the CRDs
2. Add comprehensive tests
3. Create example manifests
4. Improve documentation
5. Add CI/CD pipelines

## License

Apache License 2.0 - See LICENSE file for details.

## Current Status

✅ CRD type definitions completed
✅ Helm chart completed
✅ CRD manifests generated
⏳ Controllers not yet implemented
⏳ Container images not yet built
⏳ Examples not yet created
⏳ Tests not yet written

This provides a solid foundation for building a production-grade Kubernetes operator for managing AI infrastructure.
