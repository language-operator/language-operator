# Development Setup

Set up your local environment for Language Operator development.

## Prerequisites

- **Go 1.21+** - [Download](https://go.dev/dl/)
- **Docker** - For building images
- **kubectl** - Kubernetes CLI
- **Helm 3.8+** - Package manager
- **k3d or kind** (optional) - Local Kubernetes cluster
- **make** - Build automation

## Clone the Repository

```bash
git clone https://github.com/language-operator/language-operator
cd language-operator
```

## Install Git Hooks

Install pre-commit hooks for validation:

```bash
./scripts/setup-hooks
```

This installs hooks that:

- Enforce conventional commit messages
- Validate generated files are staged
- Run basic linting

## Development Commands

All Go work runs from `src/`:

### Build

```bash
cd src
make build       # compile operator binary
```

### Test

```bash
cd src
make test        # fmt + vet + all tests
make fmt         # go fmt only
make vet         # go vet only
```

### Integration Tests

```bash
cd src
make integration-test  # requires setup-envtest
```

Integration tests run against a real Kubernetes API server via controller-runtime's envtest.

### Run Single Test

```bash
cd src
go test ./controllers/... -run TestLanguageAgentController -v
```

### Integration Tests Only

```bash
cd src
go test -tags integration -v ./controllers/...
```

## Modifying CRD Types

After changing types in `src/api/v1alpha1/`:

```bash
cd src
make generate   # regenerate zz_generated.deepcopy.go
make helm-crds  # regenerate CRD YAMLs and copy to chart/crds/
```

**Important:** Always stage generated files together with type changes:

```bash
git add src/api/v1alpha1/zz_generated.deepcopy.go
git add src/config/crd/bases/
git add chart/crds/
```

The pre-commit hook enforces this.

## Helm Chart Validation

```bash
cd chart
helm lint .
helm template language-operator . --debug
```

## Documentation

### Generate CRD Documentation

```bash
cd src
make docs   # generates src/docs/api-reference.md (local inspection only; CI generates docs/api/reference.md)
```

### Preview Documentation Site

```bash
# Install MkDocs (one time)
pip install mkdocs-material mkdocs-awesome-pages-plugin

# Serve locally
mkdocs serve
```

Open [http://localhost:8000](http://localhost:8000)

## Local Kubernetes Testing

### Create a k3d Cluster

```bash
k3d cluster create langop-dev \
  --agents 2 \
  --k3s-arg "--disable=traefik@server:0"
```

### Install the Operator

From source:

```bash
cd chart
helm install language-operator . \
  --create-namespace \
  --namespace language-operator \
  --set image.tag=dev
```

### Build and Load Images

For local development:

```bash
# Build operator image
docker build -t language-operator:dev .

# Import into k3d
k3d image import language-operator:dev -c langop-dev
```

For the model proxy:

```bash
cd components/model
make dev  # builds and imports into k3s
```

### Watch Logs

```bash
kubectl logs -n language-operator \
  -l app.kubernetes.io/name=language-operator \
  --follow
```

## Development Workflow

1. **Create a feature branch**
   ```bash
   git checkout -b feat/your-feature
   ```

2. **Make changes and test**
   ```bash
   cd src && make test
   ```

3. **Commit with conventional commits**
   ```bash
   git commit -m "feat: add new capability"
   ```

4. **Push and create PR**

## Troubleshooting

### Tests Fail on CI but Pass Locally

- Ensure you've run `make generate && make helm-crds`
- Check that all generated files are committed
- Verify you're using the same Go version as CI

### Integration Tests Fail

```bash
# Install setup-envtest
go install sigs.k8s.io/controller-runtime/tools/setup-envtest@latest

# Run with verbose output
cd src && go test -tags integration -v ./controllers/...
```

### Pre-commit Hook Fails

The hook checks:

- Commit message format
- Generated files are staged with their sources
- No obvious issues in Go code

Read the error message carefully—it will tell you what needs fixing.

## Next Steps

- [Next: Testing Guide](testing.md)
