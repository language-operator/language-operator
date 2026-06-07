# GitHub Pages Documentation Setup

This document describes the auto-generated CRD documentation setup for Language Operator.

## Overview

The Language Operator documentation site is built with **MkDocs Material** and automatically deployed to GitHub Pages at:

**https://language-operator.github.io/language-operator/docs**

CRD API reference documentation is automatically generated from Go type definitions using `crd-ref-docs`.

## What Was Set Up

### 1. MkDocs Configuration

**File:** `mkdocs.yml`

- Material theme with dark/light mode
- Navigation structure for all documentation sections
- Search and syntax highlighting
- Responsive design optimized for API documentation

### 2. Documentation Structure

**Directory:** `docs/`

```
docs/
├── index.md                    # Homepage
├── getting-started/
│   ├── installation.md         # Installation guide
│   ├── quickstart.md          # Quick start tutorial
│   └── examples.md            # Common patterns
├── architecture/
│   ├── overview.md            # Architecture doc (from requirements/)
│   ├── controllers.md         # Controller patterns
│   ├── agents.md              # Agent runtime contract (from spec/)
│   └── tools.md               # Tool protocol (from spec/)
├── api/
│   ├── overview.md            # CRD overview
│   ├── languageagent.md       # Auto-generated from Go types
│   ├── languagecluster.md     # Auto-generated
│   ├── languagemodel.md       # Auto-generated
│   ├── languagetool.md        # Auto-generated
│   └── languagepersona.md     # Auto-generated
├── helm/
│   ├── installation.md        # Helm installation
│   ├── configuration.md       # Chart configuration
│   └── repository.md          # Helm repository info
├── development/
│   ├── contributing.md        # Contribution guidelines
│   ├── setup.md               # Development setup
│   └── testing.md             # Testing guide
├── changelog.md               # Project changelog
└── README.md                  # Documentation guide
```

### 3. GitHub Actions Workflow

**File:** `.github/workflows/docs.yaml`

**Triggers:**

- Push to `main` (when docs/ or CRD types change)
- Pull requests (builds preview artifact)
- Manual dispatch

**Steps:**

1. Install `crd-ref-docs` (Go tool)
2. Generate complete API reference from `src/api/v1alpha1/`
3. Split generated docs into individual CRD pages
4. Set up uv (`astral-sh/setup-uv`)
5. Build static site (`uv run mkdocs build --strict`)
6. Deploy to GitHub Pages under `docs/` directory
7. Keep Helm repository files in place (`keep_files: true`)

### 4. Python Requirements

**Files:** `pyproject.toml` + `uv.lock` (managed with [uv](https://docs.astral.sh/uv/))

```toml
[project]
dependencies = [
    "mkdocs>=1.5",
    "mkdocs-material>=9.5",
    "mkdocs-awesome-pages-plugin>=2.9",
]
```

`uv.lock` pins exact versions for reproducible builds. Run `uv lock --upgrade` to refresh.

### 5. Updated .gitignore

Added:
```
# MkDocs
site/
docs/api-generated.md
```

### 6. Updated CLAUDE.md

Added documentation commands:
```bash
cd src && make docs              # generate API reference markdown
make docs-serve                  # preview docs site (uv run mkdocs serve)
make docs-build                  # build static site (uv run mkdocs build --strict)
```

## Local Development

### Install Dependencies

Docs dependencies are managed with [uv](https://docs.astral.sh/uv/); `uv run` provisions them
automatically. To materialize the virtualenv explicitly:

```bash
uv sync
```

### Generate CRD Documentation

```bash
# Install crd-ref-docs (one time)
go install github.com/elastic/crd-ref-docs@latest

# Generate from Go types
cd src && make docs
```

### Preview Locally

```bash
make docs-serve   # or: uv run mkdocs serve
```

Open http://localhost:8000

### Build Static Site

```bash
make docs-build   # or: uv run mkdocs build --strict
```

Output in `site/` directory.

## Automatic Deployment

### First Deployment

The first time the workflow runs:

1. Push any documentation change to trigger the workflow
2. GitHub Actions will build the site
3. Site is deployed to `gh-pages` branch under `docs/` directory

### Ongoing Updates

Documentation is automatically rebuilt and deployed when:

- CRD type definitions change (`src/api/v1alpha1/*.go`)
- Documentation files change (`docs/**/*.md`)
- MkDocs configuration changes (`mkdocs.yml`)

## GitHub Pages Configuration

The repository serves two things from GitHub Pages:

1. **Helm Repository** (root)
   - URL: `https://language-operator.github.io/language-operator/`
   - Files: `index.yaml`, chart `.tgz` files

2. **Documentation Site** (`docs/` subdirectory)
   - URL: `https://language-operator.github.io/language-operator/docs/`
   - Static MkDocs site

Both are managed by separate workflows that use `keep_files: true` to avoid conflicts.

## Integration with Existing Setup

### Existing Workflows

- **`helm-release.yaml`** - Continues to publish Helm charts to GitHub Pages root
- **`docs.yaml`** (new) - Publishes documentation to `docs/` subdirectory

Both workflows coordinate via `keep_files: true` flag.

### CRD Generation

The existing `make docs` command in `src/Makefile` generates API reference to `src/docs/api-reference.md`. The CI workflow:

1. Runs the same generation
2. Copies to `docs/api-generated.md`
3. Splits into individual CRD pages
4. Builds MkDocs site

## Updating Documentation

### User Documentation

Edit markdown files in `docs/` and push. CI handles the rest.

### CRD API Documentation

1. Edit Go type comments in `src/api/v1alpha1/*.go`
2. Run `cd src && make docs` to regenerate locally (optional)
3. Push changes
4. CI automatically regenerates and deploys

**Example:**

```go
// LanguageAgent represents an autonomous AI agent deployment
//
// LanguageAgent runs a container image with LLM access, tool endpoints,
// and persona configuration injected by the operator.
type LanguageAgent struct {
    // Standard Kubernetes metadata
    metav1.TypeMeta   `json:",inline"`
    metav1.ObjectMeta `json:"metadata,omitempty"`

    // Spec defines the desired agent configuration
    Spec LanguageAgentSpec `json:"spec,omitempty"`

    // Status represents the observed state
    Status LanguageAgentStatus `json:"status,omitempty"`
}
```

Comments become documentation on the site.

## CI/CD Integration

### PR Previews

Pull requests build documentation as an artifact (retained for 7 days):

1. Open a PR with doc changes
2. Wait for `Documentation` workflow to complete
3. Download `docs-preview` artifact from workflow run
4. Extract and open `index.html` locally

### Production Deployment

Merging to `main` automatically deploys to:

**https://language-operator.github.io/language-operator/docs/**

## Troubleshooting

### Site not updating after push

1. Check Actions tab for workflow run status
2. Verify workflow triggered (check paths filter)
3. Check for build errors in workflow logs

### CRD docs not appearing

1. Ensure Go type comments exist
2. Run `cd src && make docs` locally to test generation
3. Check workflow logs for `crd-ref-docs` errors

### Helm repository affected

The workflows use `keep_files: true`, so they shouldn't conflict. If the Helm index is missing:

1. Re-run the `helm-release.yaml` workflow manually
2. Check that both workflows use `keep_files: true`

## Next Steps

After merging this setup:

1. **First deployment** will happen automatically on next push to `main`
2. **Update README.md** to link to the new documentation site
3. **Add badge** to README showing docs status:
   ```markdown
   [![Docs](https://img.shields.io/badge/docs-GitHub%20Pages-blue)](https://language-operator.github.io/language-operator/docs/)
   ```

## References

- [MkDocs](https://www.mkdocs.org/)
- [MkDocs Material](https://squidfunk.github.io/mkdocs-material/)
- [crd-ref-docs](https://github.com/elastic/crd-ref-docs)
- [GitHub Pages](https://docs.github.com/en/pages)
