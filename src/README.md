# language-operator

Kubernetes operator for managing language agents, models, tools, and personas.

**API Group:** `langop.io/v1alpha1`

## Development Setup

### Prerequisites

- Go (see the `go` directive in `go.mod`)
- Kubernetes cluster (K3s recommended)
- kubectl configured
- **Argo Workflows installed in the target cluster** — LanguageAgents run as Argo Workflows,
  and `make run` exits immediately if the `argoproj.io` CRDs are not served. `make dev` from
  the repo root installs it as part of the operator chart.

### Quick Start

```bash
# Install CRDs into the current cluster
make helm-crds && kubectl apply -f config/crd/bases/

# Run operator locally against that cluster
make run

# Generate code after API changes
make manifests generate
```

## Building & Deployment

```bash
# Build binary
make build

# Update Helm chart CRDs after changing api/v1alpha1
make helm-crds

# Build the image and deploy to the local k3s cluster (from the repo root)
cd .. && make dev
```

## Available Make Targets

**Development:**
- `make manifests` - Generate CRD manifests
- `make generate` - Generate DeepCopy code
- `make fmt` - Format code
- `make vet` - Lint code
- `make test` - Run tests
- `make docs` - Generate API documentation

- `make integration-test` - Run envtest integration tests
- `make helm-crds` - Regenerate CRDs and copy them into the Helm chart

**Build and run:**
- `make build` - Build manager binary
- `make run` - Run locally against the current cluster

Deploying to a cluster is driven from the repo root, not here: `make dev` builds, imports into
k3s, and upgrades both Helm releases; `make wipe` resets the cluster.

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
