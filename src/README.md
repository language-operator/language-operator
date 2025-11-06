# language-operator

Kubernetes operator for managing language agents, models, tools, and personas.

**API Group:** `langop.io/v1alpha1`

## Development Setup

### Prerequisites

- Go 1.22+
- Kubernetes cluster (kind, minikube, etc.)
- kubectl configured

### Quick Start

```bash
# Install CRDs
make install

# Run operator locally
make run

# Generate code after API changes
make manifests generate
```

## Building & Deployment

```bash
# Build binary
make build

# Build and push Docker image
make docker-build docker-push IMG=your-registry/language-operator:latest

# Deploy to cluster
make deploy IMG=your-registry/language-operator:latest

# Update Helm chart CRDs
make helm-crds
```

## Available Make Targets

**Development:**
- `make manifests` - Generate CRD manifests
- `make generate` - Generate DeepCopy code
- `make fmt` - Format code
- `make vet` - Lint code
- `make test` - Run tests
- `make docs` - Generate API documentation

**Deployment:**
- `make install` - Install CRDs to cluster
- `make uninstall` - Remove CRDs from cluster
- `make deploy` - Deploy operator to cluster
- `make undeploy` - Remove operator from cluster
- `make run` - Run locally against cluster

**Build:**
- `make build` - Build manager binary
- `make docker-build` - Build Docker image
- `make docker-push` - Push Docker image

## Project Structure

```
.
├── api/v1alpha1/          # CRD type definitions
├── controllers/           # Reconciliation logic
├── config/
│   ├── crd/bases/        # Generated CRD manifests
│   ├── rbac/             # RBAC configuration
│   └── manager/          # Deployment manifests
├── docs/                  # API reference documentation
└── Makefile
```

## CRDs

The operator manages these custom resources:

- **LanguageCluster** - Network-isolated environments
- **LanguageAgent** - Autonomous agents
- **LanguageTool** - MCP tool servers
- **LanguageModel** - Model configurations
- **LanguagePersona** - Reusable personalities
- **LanguageClient** - User interfaces

See the [main README](../../README.md) for usage examples.

See [docs/api-reference.md](docs/api-reference.md) for complete API documentation.

## Contributing

After modifying CRD types:

1. Run `make manifests generate` to update generated code
2. Run `make docs` to update API documentation
3. Run `make test` to verify tests pass
4. Update examples if needed

## Resources

- [Kubebuilder Book](https://book.kubebuilder.io/)
- [controller-runtime](https://github.com/kubernetes-sigs/controller-runtime)
