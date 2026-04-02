# Contributing

Thank you for your interest in contributing to Language Operator!

## Getting Started

1. **Fork the repository** on GitHub
2. **Clone your fork** locally
3. **Set up development environment** - see [Development Setup](setup.md)
4. **Create a feature branch** from `main`
5. **Make your changes** following our guidelines below
6. **Test your changes** - see [Testing](testing.md)
7. **Commit with conventional commits** (see below)
8. **Push and create a pull request**

## Commit Message Convention

We use [Conventional Commits](https://www.conventionalcommits.org/):

```
<type>: <description>

[optional body]

[optional footer]
```

**Types:**

- `feat:` - New feature
- `fix:` - Bug fix
- `docs:` - Documentation changes
- `test:` - Test additions or fixes
- `chore:` - Code cleanup, refactoring, dependency updates
- `WIP:` - Work in progress (partial implementation)

**Examples:**

```
feat: add support for Azure OpenAI provider
fix: resolve race condition in model reconciliation
docs: update installation guide with RBAC requirements
test: add integration tests for LanguageCluster controller
```

**The pre-commit hook enforces this convention.** Install hooks with:

```bash
./scripts/setup-hooks
```

## Pull Request Guidelines

### PR Title

Must follow conventional commit format:

```
feat: add LanguageAgentVersion CRD for versioning
fix: handle missing namespace in cross-namespace references
docs: clarify MODEL_ENDPOINTS injection behavior
```

**This is enforced by CI** via `pr-checks.yaml`.

### PR Description

Include:

- **What** this PR changes
- **Why** the change is needed
- **How** it works (for complex changes)
- **Testing** - how you verified it works

### PR Size

- Keep PRs focused and reasonably sized
- CI automatically labels PRs by size (xs/s/m/l/xl)
- Consider breaking large changes into multiple PRs

## Code Guidelines

### Go Code

- **Format:** Run `cd src && make fmt` before committing
- **Lint:** Run `cd src && make vet` to check for issues
- **Tests:** Add tests for new features
- **Comments:** Add godoc comments for exported types and functions

### Generated Files

**After modifying CRD types** (`src/api/v1alpha1/*.go`):

```bash
cd src
make generate    # regenerate zz_generated.deepcopy.go
make helm-crds   # regenerate CRD YAMLs and copy to chart/crds/
```

**Stage all generated files together:**

```bash
git add src/api/v1alpha1/zz_generated.deepcopy.go
git add src/config/crd/bases/
git add chart/crds/
```

**The pre-commit hook will fail if generated files are modified but not staged.**

### Documentation

- **API docs** are auto-generated from Go comments
- **User docs** live in `docs/` (Markdown)
- Update relevant docs when changing behavior
- Use clear, concise language

## Testing

See the [Testing Guide](testing.md) for details on:

- Unit tests
- Integration tests
- Manual testing

**Run tests before submitting:**

```bash
cd src && make test
```

## Development Workflow

### Making Changes

1. **Branch naming:** Use descriptive names
   ```bash
   git checkout -b feat/azure-openai-support
   git checkout -b fix/model-reconciliation-race
   ```

2. **Make incremental commits** with clear messages
3. **Test locally** before pushing
4. **Push and create PR** when ready

### Code Review

- All PRs require review before merging
- Address review feedback promptly
- Keep discussion focused and respectful

### CI Checks

All PRs must pass:

- ✅ Go formatting (`gofmt`)
- ✅ Go vet
- ✅ Unit tests
- ✅ Integration tests
- ✅ CRD manifest validation
- ✅ PR title convention check
- ✅ YAML linting

## Architecture Decisions

For significant architectural changes:

1. **Open an issue** to discuss the approach first
2. **Reference the issue** in your PR
3. **Update architecture docs** if needed

## Questions?

- **Issues:** [GitHub Issues](https://github.com/language-operator/language-operator/issues)
- **Discussions:** [GitHub Discussions](https://github.com/language-operator/language-operator/discussions)

## License

By contributing, you agree that your contributions will be licensed under the MIT License.
