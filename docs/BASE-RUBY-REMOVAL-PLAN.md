# Base/Ruby Image Removal Plan

## Summary

Simplifying the build process by removing the intermediate `langop/ruby` and component layer images (`langop/client`, `langop/agent`, `langop/tool`). Each component/tool now installs dependencies directly via Gemfile from the private gem registry.

## Changes Completed

### ✅ 1. Updated All Gemfiles

All Gemfiles now include:
- `source 'https://git.theryans.io/api/packages/language-operator/rubygems'`
- `gem 'language-operator', '~> 0.1.0'`

**Files updated:**
- [components/tool/Gemfile](components/tool/Gemfile)
- [components/agent/Gemfile](components/agent/Gemfile)
- [components/client/Gemfile](components/client/Gemfile)
- [tools/web/Gemfile](tools/web/Gemfile)
- [tools/email/Gemfile](tools/email/Gemfile) - also added `mail` gem
- [agents/cli/Gemfile](agents/cli/Gemfile)
- [agents/headless/Gemfile](agents/headless/Gemfile)
- [agents/web/Gemfile](agents/web/Gemfile)

### ✅ 2. Updated All Dockerfiles

All Docker files now:
- Start from `git.theryans.io/langop/base:latest`
- Install Ruby + bundler + build-base via apk
- Configure bundle with registry credentials via build args
- Run `bundle install` to get `language-operator` gem from registry

**Files updated:**
- [components/tool/Dockerfile](components/tool/Dockerfile)
- [components/agent/Dockerfile](components/agent/Dockerfile)
- [components/client/Dockerfile](components/client/Dockerfile)
- [tools/web/Dockerfile](tools/web/Dockerfile)
- [tools/email/Dockerfile](tools/email/Dockerfile)
- [agents/cli/Dockerfile](agents/cli/Dockerfile)
- [agents/headless/Dockerfile](agents/headless/Dockerfile)
- [agents/web/Dockerfile](agents/web/Dockerfile)

## Changes Needed

### 3. Update CI Workflow

**File**: [.github/workflows/build-images.yaml](.github/workflows/build-images.yaml)

#### Remove These Jobs:
- `build-ruby` (lines 117-164) - no longer needed
- `build-ruby-components` (lines 166-213) - components build directly
- `build-agent` (lines 215-259) - agents build directly

#### Update Job Dependencies:

**Before**:
```
build-base
  └─ build-ruby
      ├─ build-ruby-components (tool, client)
      │   └─ build-tools (web-tool, email-tool)
      └─ build-agent
          └─ build-agents (cli, web, headless)
```

**After**:
```
build-base
  ├─ build-tool
  ├─ build-client
  ├─ build-agent
  ├─ build-tools (web-tool, email-tool)
  └─ build-agents (cli, web, headless)
```

#### Add Build Args to All Builds:

Every component/tool/agent build needs:
```yaml
build-args: |
  REGISTRY_USERNAME=${{ secrets.REGISTRY_USERNAME }}
  REGISTRY_PASSWORD=${{ secrets.REGISTRY_PASSWORD }}
```

#### Specific Changes:

1. **Create `build-tool` job** (replaces part of `build-ruby-components`):
```yaml
build-tool:
  runs-on: builder
  needs: build-base
  steps:
    # ... standard checkout, buildx, login, metadata ...
    - name: Build and push tool image
      uses: docker/build-push-action@v5
      with:
        context: ./components/tool
        file: ./components/tool/Dockerfile
        build-args: |
          REGISTRY_USERNAME=${{ secrets.REGISTRY_USERNAME }}
          REGISTRY_PASSWORD=${{ secrets.REGISTRY_PASSWORD }}
        push: ${{ github.event_name != 'pull_request' && (github.event.inputs.push != 'false') }}
        tags: ${{ steps.meta.outputs.tags }}
        labels: ${{ steps.meta.outputs.labels }}
        cache-from: type=gha
        cache-to: type=gha,mode=max
```

2. **Create `build-client` job** (replaces part of `build-ruby-components`):
```yaml
build-client:
  runs-on: builder
  needs: build-base
  steps:
    # ... similar to build-tool ...
    - name: Build and push client image
      uses: docker/build-push-action@v5
      with:
        context: ./components/client
        file: ./components/client/Dockerfile
        build-args: |
          REGISTRY_USERNAME=${{ secrets.REGISTRY_USERNAME }}
          REGISTRY_PASSWORD=${{ secrets.REGISTRY_PASSWORD }}
        # ... rest same as build-tool ...
```

3. **Create `build-agent` job** (replaces old build-agent):
```yaml
build-agent:
  runs-on: builder
  needs: build-base
  steps:
    # ... similar structure ...
```

4. **Update `build-tools` job**:
```yaml
build-tools:
  runs-on: builder
  needs: build-base  # Changed from: needs: build-ruby-components
  strategy:
    matrix:
      tool: [web, email]
  steps:
    # ... checkout, buildx, login, metadata ...
    - name: Build and push ${{ matrix.tool }}-tool image
      uses: docker/build-push-action@v5
      with:
        context: ./tools/${{ matrix.tool }}
        file: ./tools/${{ matrix.tool }}/Dockerfile
        build-args: |
          REGISTRY_USERNAME=${{ secrets.REGISTRY_USERNAME }}
          REGISTRY_PASSWORD=${{ secrets.REGISTRY_PASSWORD }}
        # ... rest ...
```

5. **Update `build-agents` job**:
```yaml
build-agents:
  runs-on: builder
  needs: build-base  # Changed from: needs: build-agent
  strategy:
    matrix:
      agent: [cli, web, headless]
  steps:
    # ... checkout, buildx, login, metadata ...
    - name: Build and push agent-${{ matrix.agent }} image
      uses: docker/build-push-action@v5
      with:
        context: ./agents/${{ matrix.agent }}
        file: ./agents/${{ matrix.agent }}/Dockerfile
        build-args: |
          REGISTRY_USERNAME=${{ secrets.REGISTRY_USERNAME }}
          REGISTRY_PASSWORD=${{ secrets.REGISTRY_PASSWORD }}
        # ... rest ...
```

### 4. Update Component Makefiles

Components that had `build-gem` targets need to be updated.

**Check these files:**
- [components/ruby/Makefile](components/ruby/Makefile) - can be deleted entirely
- [components/tool/Makefile](components/tool/Makefile) - remove build-gem if present
- [components/agent/Makefile](components/agent/Makefile) - remove build-gem if present
- [components/client/Makefile](components/client/Makefile) - remove build-gem if present

All should have a simple `build` target that passes registry credentials:
```makefile
build:
\tdocker build \
\t\t--build-arg REGISTRY_USERNAME="${REGISTRY_USERNAME}" \
\t\t--build-arg REGISTRY_PASSWORD="${REGISTRY_PASSWORD}" \
\t\t-t $(IMAGE_FULL) .
```

### 5. Remove Old Component Directories

After CI is updated and working:
```bash
git rm -rf components/ruby
```

## Testing Plan

### Phase 1: Local Testing

Test one component locally:
```bash
export REGISTRY_USERNAME="your-username"
export REGISTRY_PASSWORD="your-password"

cd components/tool
docker build \
  --build-arg REGISTRY_USERNAME="${REGISTRY_USERNAME}" \
  --build-arg REGISTRY_PASSWORD="${REGISTRY_PASSWORD}" \
  -t test-tool:latest .

# Verify gem is installed
docker run --rm test-tool:latest ruby -e "require 'language_operator'; puts LanguageOperator::VERSION"
```

### Phase 2: CI Testing

1. Create feature branch
2. Apply all CI changes
3. Push and monitor builds
4. Verify all images build successfully

### Phase 3: Deployment Testing

1. Deploy updated images to test cluster
2. Verify all components function correctly
3. Check for any runtime issues

## Benefits

1. **Simpler Build Process**: No intermediate layer images to debug
2. **Faster Iteration**: Components build independently, no waiting for base layers
3. **Easier Debugging**: Each Dockerfile is self-contained
4. **Clearer Dependencies**: Gemfile explicitly shows what's needed
5. **Consistent Pattern**: All components follow same pattern

## Rollback

If issues arise:
1. Revert Dockerfile changes
2. Revert Gemfile changes
3. Revert CI workflow changes
4. Previous `langop/ruby` image should still be in registry

## Image Size Comparison

**Before** (layered approach):
- base: ~10 MB
- ruby (base + gem): ~60 MB
- tool (ruby + deps): ~65 MB
- Total layers: 3

**After** (direct approach):
- base: ~10 MB
- tool (base + ruby + gem + deps): ~65 MB
- Total layers: 2

Size is similar, but build process is much simpler.

## Next Steps

1. Review this plan
2. Update CI workflow ([.github/workflows/build-images.yaml](.github/workflows/build-images.yaml))
3. Test locally with one component
4. Create feature branch and test CI
5. Merge if successful
6. Remove `components/ruby` directory
