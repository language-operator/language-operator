# Development Setup

## Prerequisites

- **Go 1.25+** — [download](https://go.dev/dl/)
- **Docker** — building images
- **kubectl** and **Helm 3.8+**
- **make**
- **k3s** — `make dev` deploys into a local k3s cluster (it imports images via `k3s ctr`). [kind](https://kind.sigs.k8s.io/)/[k3d](https://k3d.io/) work for manual testing but aren't wired into `make dev`.

## Clone and Install Hooks

```bash
git clone https://github.com/language-operator/language-operator
cd language-operator
make setup-hooks   # pre-commit hooks: conventional commits + generated-files-staged check
```

## Common Commands

All Go work runs from `src/`:

```bash
cd src
make build              # compile the operator binary
make test               # fmt + vet + all unit tests
make fmt                # go fmt
make vet                # go vet
make integration-test   # envtest integration tests (requires setup-envtest)

go test ./controllers/... -run TestLanguageAgentController -v   # single test
go test -tags integration -v ./controllers/...                 # integration only
```

See [Testing](testing.md) for the test layout and patterns.

## Modifying CRD Types

After changing any type in `src/api/v1alpha1/`, regenerate and **stage the output** — the
pre-commit hook (and CI) fail if generated files drift from their sources:

```bash
cd src
make generate   # zz_generated.deepcopy.go
make helm-crds  # CRD YAMLs → charts/language-operator/templates/crds/

git add src/api/v1alpha1/zz_generated.deepcopy.go \
        src/config/crd/bases/ \
        charts/language-operator/templates/crds/
```

## Helm Chart Validation

```bash
helm lint charts/language-operator        && helm template charts/language-operator --debug
helm lint charts/language-operator-runtimes && helm template charts/language-operator-runtimes --debug
```

## Local Cluster (`make dev`)

`make dev` (project root) is the canonical local deploy: it builds the binary, builds a Docker
image tagged with the current git SHA, imports it into k3s, and `helm upgrade`s both the operator
and runtimes releases.

```bash
make dev
```

It reads per-developer overrides from `charts/language-operator/values.local.yaml` and
`charts/language-operator-runtimes/values.local.yaml`. These are gitignored — create them before
the first run (empty files are fine if you have no overrides).

Because the image tag is the git SHA, **commit your changes before `make dev`** so the tag changes
and Docker cache is busted.

For the model gateway image, `cd components/model-gateway && make dev` builds and imports it into k3s.

Watch operator logs:

```bash
kubectl logs -n language-operator -l app.kubernetes.io/name=language-operator --follow
```

## Documentation

```bash
cd src && make docs   # regenerate the CRD API reference (CI publishes docs/api/reference.md)
make docs-serve       # preview the site at http://localhost:8000 (uv provisions deps)
```

## Troubleshooting

- **CI fails but local passes** — you almost certainly skipped `make generate && make helm-crds`, or
  didn't commit the generated files. Re-run both and stage the output.
- **Integration tests fail to start** — install the envtest binaries:
  `go install sigs.k8s.io/controller-runtime/tools/setup-envtest@latest`.

For branching, commit conventions, and the PR process, see [Contributing](contributing.md).
