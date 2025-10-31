# CI/CD Workflows

This directory contains GitHub Actions workflows for automated building, testing, and releasing of the language-operator project.

## Workflows

### 🏗️ build-images.yaml
**Triggers**: Push to main, tags, PRs, manual dispatch

Builds and pushes all Docker images:
- **Operator**: The Kubernetes operator controller
- **Components**: Base images (base, devel, server, client)
- **Tools**: Tool servers (web, email, sms, doc)
- **Agents**: Agent implementations (cli, web, headless)
- **Model**: LiteLLM model proxy

**Image Tagging Strategy**:
- `latest` - Latest build from main branch
- `v1.2.3` - Semantic version tags
- `v1.2` - Major.minor tags
- `v1` - Major version tags
- `main-<sha>` - Branch with git SHA
- `pr-123` - Pull request number

**Registry**: `git.theryans.io/langop/*`

### 🧪 test.yaml
**Triggers**: Push to main, PRs, manual dispatch

Runs comprehensive tests:
- **Lint**: Go formatting, vet, staticcheck
- **Unit Tests**: Go tests with race detection and coverage
- **Manifest Validation**: Ensures CRDs are up-to-date
- **Helm Validation**: Lints and templates Helm charts
- **Build Tests**: Validates Docker builds without pushing

### ✅ pr-checks.yaml
**Triggers**: PR open, sync, reopen

Validates pull requests:
- **PR Title**: Enforces conventional commit format
- **Merge Conflicts**: Detects conflicts with main
- **YAML Lint**: Validates YAML syntax
- **Size Labels**: Auto-labels PR size (XS/S/M/L/XL)
- **Status Comments**: Posts CI status to PR

### 📦 helm-release.yaml
**Triggers**: Push to main (chart changes), releases, manual dispatch

Releases Helm charts:
- Copies CRDs to chart
- Packages Helm chart
- Generates Helm repo index
- Uploads to registry
- Creates GitHub release (for version tags)

## Configuration

### Required Secrets

Set these in Forgejo repository settings:

```bash
REGISTRY_USERNAME  # Username for git.theryans.io
REGISTRY_PASSWORD  # Password/token for git.theryans.io
```

### Optional Secrets

```bash
CODECOV_TOKEN     # For code coverage uploads (optional)
```

## Manual Workflows

### Build and Push Images

Manually trigger image builds:

```bash
# Via GitHub/Forgejo UI:
Actions → Build and Push Images → Run workflow
  - Branch: main
  - Push images: true (default)
```

### Build Without Pushing

Test builds without publishing:

```bash
# Via UI or API:
Actions → Build and Push Images → Run workflow
  - Push images: false
```

## Local Development

### Test Workflows Locally

Use [act](https://github.com/nektos/act) to run workflows locally:

```bash
# Install act
brew install act  # macOS
# or
curl https://raw.githubusercontent.com/nektos/act/master/install.sh | sudo bash

# Run a workflow
act push -W .github/workflows/test.yaml

# Run specific job
act -j lint

# Run with secrets
act -s REGISTRY_USERNAME=user -s REGISTRY_PASSWORD=pass
```

### Validate Workflows

Check workflow syntax:

```bash
# Install actionlint
brew install actionlint  # macOS
# or
go install github.com/rhysd/actionlint/cmd/actionlint@latest

# Lint all workflows
actionlint .github/workflows/*.yaml
```

## Workflow Dependencies

```
┌─────────────────┐
│  build-operator │
└─────────────────┘

┌─────────────────────┐
│  build-components   │
└──────────┬──────────┘
           │
           ├──► ┌──────────────┐
           │    │  build-tools │
           │    └──────────────┘
           │
           └──► ┌───────────────┐
                │  build-agents │
                └───────────────┘
```

Components must build before tools and agents since they may depend on base images.

## Image Build Matrix

### Parallel Builds

Multiple images build in parallel using matrix strategy:

**Components**: base, devel, server, client (4 parallel jobs)
**Tools**: web, email, sms, doc (4 parallel jobs)
**Agents**: cli, web, headless (3 parallel jobs)

Total: ~11 parallel build jobs

### Build Time

Typical build times (with cache):
- Operator: ~2 minutes
- Components: ~1 minute each
- Tools: ~1 minute each
- Agents: ~2 minutes each

**Total workflow time**: ~5 minutes (with parallelization)

## Troubleshooting

### Builds Failing

1. **Check logs**: Click on failed job in Actions tab
2. **Verify Dockerfiles**: Ensure all Dockerfiles exist and are valid
3. **Check dependencies**: Ensure base images are available
4. **Registry access**: Verify credentials are set correctly

### Images Not Pushed

1. **Check push condition**: PRs don't push by default
2. **Verify registry auth**: Check REGISTRY_USERNAME and REGISTRY_PASSWORD secrets
3. **Check permissions**: Ensure service account has push access

### Tests Failing

1. **Run locally**: `cd kubernetes/language-operator && make test`
2. **Check manifests**: `make manifests` and commit changes
3. **Verify Go version**: Ensure using correct Go version from go.mod

## Best Practices

### Commit Messages

Follow [Conventional Commits](https://www.conventionalcommits.org/):

```
feat: add new feature
fix: resolve bug
docs: update documentation
chore: update dependencies
test: add tests
refactor: restructure code
```

### Image Tags

- Use semantic versioning for releases: `v1.2.3`
- Don't push `:latest` from feature branches
- Use branch-specific tags for testing: `feature-xyz-<sha>`

### Caching

Workflows use GitHub Actions cache:
- Go modules cached by go version and go.sum
- Docker layers cached using buildx cache
- Cache persists between workflow runs

### Security

- Never commit secrets or tokens
- Use GitHub secrets for credentials
- Scan images for vulnerabilities (add Trivy/Snyk if needed)
- Keep actions updated to latest versions
