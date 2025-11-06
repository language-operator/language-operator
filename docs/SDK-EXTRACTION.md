# SDK Extraction Guide

The Ruby SDK has been extracted from the monorepo to simplify the build process.

## Overview

**Before**: SDK was built as part of the monorepo CI, creating complex dependencies between jobs.

**After**: SDK is maintained in a separate repository with its own CI pipeline.

## Repository Locations

- **Monorepo**: `/home/james/workspace/language-operator/language-operator`
- **SDK Repository**: `/home/james/workspace/language-operator-gem`

## Changes Made

### 1. New SDK Repository Structure

```
language-operator-gem/
├── .github/workflows/publish.yaml  # CI for testing and publishing
├── .gitignore                       # Standard Ruby gem gitignore
├── CLAUDE.md                        # Project standards
├── README.md                        # Usage instructions
├── Gemfile                          # Development dependencies
├── Rakefile                         # Build tasks
├── language-operator.gemspec        # Gem specification
├── bin/aictl                        # CLI executable
└── lib/                             # Ruby source code
    └── language_operator/
```

### 2. CI/CD Pipeline

The new repository has a GitHub Actions workflow that:

1. **Tests** - Runs RSpec tests and RuboCop linting
2. **Builds** - Creates the gem package
3. **Publishes** - Uploads to `git.theryans.io/api/packages/langop/rubygems`

### 3. Monorepo Changes

#### Removed from CI ([.github/workflows/build-images.yaml](.github/workflows/build-images.yaml))
- `build-gem` job - No longer builds gem locally
- `publish-ruby-gem` job - Publishing happens in SDK repo
- Gem artifact upload/download steps

#### Updated Components

**[components/ruby/Dockerfile](components/ruby/Dockerfile)**:
- Installs gem from registry instead of copying local file
- Accepts `REGISTRY_USERNAME` and `REGISTRY_PASSWORD` build args
- Falls back to RubyGems.org if registry unavailable

**[components/ruby/Makefile](components/ruby/Makefile)**:
- Removed `build-gem` target
- Updated `build` target to pass registry credentials
- Simplified build process

## Workflow

### SDK Development

1. Make changes in `language-operator-gem` repository
2. Commit and push to trigger CI
3. CI runs tests, builds gem, and publishes to registry

### Component Development

1. Components automatically pull latest gem from registry during build
2. No need to rebuild gem locally
3. Faster iteration on components without SDK changes

## Benefits

1. **Simpler CI**: No complex job dependencies in monorepo
2. **Faster Builds**: Components don't wait for gem build
3. **Independent Versioning**: SDK can be versioned separately
4. **Clearer Separation**: SDK development isolated from infrastructure
5. **Easier Testing**: SDK tests run independently

## Publishing the Gem

### Automatic Publishing

The gem is automatically published on:
- Push to `main` branch
- Creating a version tag (e.g., `v0.1.1`)

### Manual Publishing

```bash
cd language-operator-gem
gem build language-operator.gemspec
curl -u "${REGISTRY_USERNAME}:${REGISTRY_PASSWORD}" \
     --upload-file "language-operator-0.1.0.gem" \
     "https://git.theryans.io/api/packages/langop/rubygems"
```

## Installing the Gem

### In Dockerfile

```dockerfile
ARG REGISTRY_USERNAME
ARG REGISTRY_PASSWORD
RUN if [ -n "$REGISTRY_USERNAME" ] && [ -n "$REGISTRY_PASSWORD" ]; then \
        bundle config set --global gem.theryans.io "${REGISTRY_USERNAME}:${REGISTRY_PASSWORD}"; \
    fi
RUN gem install language-operator --source https://git.theryans.io/api/packages/langop/rubygems
```

### For Local Development

```bash
bundle config set --global gem.theryans.io ${REGISTRY_USERNAME}:${REGISTRY_PASSWORD}
gem install language-operator --source https://git.theryans.io/api/packages/langop/rubygems
```

## Migration Checklist

- [x] Extract SDK to separate repository
- [x] Create CI/CD workflow for SDK
- [x] Update monorepo Dockerfile to pull from registry
- [x] Update monorepo CI to remove gem building
- [x] Add registry credentials to build args
- [x] Update documentation
- [ ] Initialize git repository in `language-operator-gem`
- [ ] Push SDK repository to git.theryans.io
- [ ] Configure GitHub Actions secrets for SDK repo
- [ ] Test SDK CI pipeline
- [ ] Test monorepo builds with registry gem
- [ ] Remove `sdk/ruby` from monorepo

## Rollback Plan

If issues arise, the monorepo can temporarily build the gem locally by:

1. Reverting changes to [.github/workflows/build-images.yaml](.github/workflows/build-images.yaml)
2. Reverting changes to [components/ruby/Dockerfile](components/ruby/Dockerfile)
3. Reverting changes to [components/ruby/Makefile](components/ruby/Makefile)

## Next Steps

1. Initialize git repository in SDK directory
2. Push to git.theryans.io
3. Configure CI secrets (REGISTRY_USERNAME, REGISTRY_PASSWORD)
4. Trigger first build to publish gem
5. Test monorepo build with published gem
6. Remove sdk/ruby from monorepo once verified
