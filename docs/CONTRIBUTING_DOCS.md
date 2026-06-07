# Language Operator Documentation

This directory contains the source for the Language Operator documentation site, which is built with [MkDocs Material](https://squidfunk.github.io/mkdocs-material/).

## Documentation Structure

```
docs/
├── index.md                    # Homepage
├── getting-started/            # Installation and quick start
├── architecture/               # System design and contracts
├── api/                        # CRD reference (auto-generated)
├── helm/                       # Helm chart documentation
└── development/                # Contributing and development guides
```

## Building Locally

### Prerequisites

Docs dependencies are managed with [uv](https://docs.astral.sh/uv/) (`pyproject.toml` + `uv.lock`).
`uv run` provisions the environment automatically — no manual install step is required. To materialize
the virtualenv explicitly:

```bash
uv sync
```

### Generate CRD Documentation

The CRD API reference is auto-generated from Go types:

```bash
# Install crd-ref-docs (one time)
go install github.com/elastic/crd-ref-docs@latest

# Generate API docs
cd src
make docs
```

This creates `src/docs/api-reference.md` for **local inspection only**. In CI, `crd-ref-docs` is run separately with `--output-path=../docs/api-generated.md`, and that file is copied to `docs/api/reference.md` to produce the single reference page on the docs site.

### Preview the Site

Start a local development server:

```bash
make docs-serve   # or: uv run mkdocs serve
```

Open [http://localhost:8000](http://localhost:8000) in your browser. The site auto-reloads when you edit markdown files.

### Build Static Site

Build the complete static site:

```bash
make docs-build   # or: uv run mkdocs build --strict
```

Output is in `site/` (git-ignored).

## Automatic Deployment

Documentation is automatically deployed to GitHub Pages at:

**https://language-operator.github.io/language-operator/docs**

The deployment workflow (`.github/workflows/docs.yaml`) runs on:

- Push to `main` branch (when docs files change)
- Pull requests (builds preview artifact)
- Manual workflow dispatch

## Documentation Guidelines

### Markdown Style

- Use clear, concise language
- Include code examples for complex concepts
- Use admonitions for warnings/notes:
  ```markdown
  !!! warning "Important"
      This is a warning message.
  ```

### Code Blocks

Always specify the language:

````markdown
```yaml
apiVersion: langop.io/v1alpha1
kind: LanguageAgent
```
````

### Internal Links

Use relative links to other docs:

```markdown
See [Installation Guide](getting-started/installation.md)
```

### API Reference Pages

The individual CRD reference pages in `api/` are auto-generated during CI. Don't edit them manually—edit the Go type comments in `src/api/v1alpha1/` instead.

## Configuration

Site configuration is in `mkdocs.yml` at the repository root.

Key settings:

- **theme:** Material theme with dark/light mode
- **nav:** Site navigation structure
- **plugins:** Search and awesome-pages
- **markdown_extensions:** Code highlighting, admonitions, etc.

## Publishing

The documentation site is published to the `gh-pages` branch under the `docs/` directory, alongside the Helm repository which is in the root.

GitHub Pages serves:

- **Helm repo:** `https://language-operator.github.io/language-operator/index.yaml`
- **Documentation:** `https://language-operator.github.io/language-operator/docs/`

## Contributing

When adding new documentation:

1. Create or edit markdown files in `docs/`
2. Update navigation in `mkdocs.yml` if adding new pages
3. Preview locally with `mkdocs serve`
4. Commit and push—CI will handle deployment

For CRD documentation changes:

1. Edit Go type comments in `src/api/v1alpha1/*.go`
2. Regenerate with `cd src && make docs`
3. The CI workflow will regenerate `docs/api/reference.md` from your type comments

## Troubleshooting

### MkDocs build fails

Check for:

- Missing dependencies: `uv sync` (or just use `uv run mkdocs ...`)
- Invalid markdown syntax
- Broken internal links

### CRD docs not updating

Ensure you:

1. Modified Go type comments (not generated markdown)
2. Ran `make docs` to regenerate
3. Pushed changes to trigger CI workflow

### Local preview shows outdated content

- MkDocs caches aggressively
- Restart `mkdocs serve`
- Clear browser cache
