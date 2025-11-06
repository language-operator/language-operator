# Requirement: Docker Build Targets in Makefile

**Status**: REQUIRED
**Applies to**: All Makefiles in directories containing a Dockerfile
**RFC 2119**: MUST
**Check**: Verify presence of `build`, `scan`, `shell`, `run` targets

## Description

Any Makefile in a directory containing a Dockerfile MUST include standardized Docker-related targets. This ensures consistent container build and management operations across all components.

## Required Targets

All Dockerized projects MUST include:

1. **build** - Build the Docker image
2. **scan** - Security scan the built image
3. **shell** - Open interactive shell in container
4. **run** - Run the container

## Optional Targets

Projects MAY include:

- **publish** - Push image to registry (typically handled by CI/CD)

## Example Implementation

```makefile
IMAGE_NAME := git.theryans.io/language-operator/component-name
IMAGE_TAG := latest
IMAGE_FULL := $(IMAGE_NAME):$(IMAGE_TAG)

.PHONY: help
help:
	@echo "Docker Targets:"
	@echo "  build      - Build the Docker image"
	@echo "  scan       - Security scan the image with trivy"
	@echo "  shell      - Open interactive shell in container"
	@echo "  run        - Run the container"

.PHONY: build
build:
	docker build -t $(IMAGE_FULL) .

.PHONY: scan
scan:
	docker run --rm -v /var/run/docker.sock:/var/run/docker.sock \
		aquasec/trivy image $(IMAGE_FULL)

.PHONY: shell
shell:
	docker run --rm -it $(IMAGE_FULL) /bin/sh

.PHONY: run
run:
	docker run --rm -it $(IMAGE_FULL)

```

## Variable Naming Convention

Makefiles SHOULD define these standard variables:

- **IMAGE_NAME** - Full image path including registry and namespace
- **IMAGE_TAG** - Version tag (default: `latest`)
- **IMAGE_FULL** - Combined `$(IMAGE_NAME):$(IMAGE_TAG)`

## Compliance

To check compliance:

```bash
# Find directories with Dockerfiles
find . -name Dockerfile -exec dirname {} \;

# For each directory, check Makefile has required targets
for dir in $(find . -name Dockerfile -exec dirname {} \;); do
  if [ -f "$dir/Makefile" ]; then
    echo "Checking $dir/Makefile"
    grep -q "^build:" "$dir/Makefile" || echo "  Missing: build"
    grep -q "^scan:" "$dir/Makefile" || echo "  Missing: scan"
    grep -q "^shell:" "$dir/Makefile" || echo "  Missing: shell"
    grep -q "^run:" "$dir/Makefile" || echo "  Missing: run"
  else
    echo "WARNING: $dir has Dockerfile but no Makefile"
  fi
done
```

## Target Behaviors

### build
- MUST build the Docker image with appropriate tags
- SHOULD use build arguments if needed for customization
- MAY use `--pull` to ensure base image is current

### scan
- MUST run security scanning tool (trivy, grype, etc.)
- SHOULD fail on HIGH/CRITICAL vulnerabilities
- MAY use other scanning tools if trivy unavailable

### shell
- MUST open interactive shell in container
- SHOULD use `/bin/sh` for Alpine-based images, `/bin/bash` for others
- MUST use `--rm` flag to clean up container after exit
- MUST use `-it` for interactive terminal

### run
- MUST start the container with default command
- SHOULD expose necessary ports
- MAY include volume mounts or environment variables as needed
- MUST use `--rm` flag for cleanup


## Rationale

- **Consistency**: Same commands work across all components
- **Discoverability**: Developers know what targets are available
- **Security**: `scan` target encourages vulnerability checks
- **Debugging**: `shell` target simplifies troubleshooting
- **CI/CD**: Standardized targets integrate easily into pipelines

## Advanced Example with Multi-Architecture

```makefile
PLATFORMS := linux/amd64,linux/arm64

.PHONY: build-multiarch
build-multiarch:
	docker buildx build --platform $(PLATFORMS) -t $(IMAGE_FULL) .

.PHONY: publish-multiarch
publish-multiarch:
	docker buildx build --platform $(PLATFORMS) -t $(IMAGE_FULL) --push .
```

## Exceptions

- Makefiles in directories with multi-stage Dockerfiles MAY include additional intermediate build targets
- Projects with multiple Dockerfiles MAY namespace targets (e.g., `build-agent`, `build-tool`)

---

**Note**: When implementing this requirement, commit changes with a one-line summary.
