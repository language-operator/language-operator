# Image configuration
IMAGE_NAME := ghcr.io/language-operator/git
IMAGE_TAG := latest
IMAGE_FULL := $(IMAGE_NAME):$(IMAGE_TAG)

# Build the Docker image
.PHONY: build
build:
	docker build -t $(IMAGE_FULL) .

# Build and import into k3s (development inner loop)
.PHONY: dev
dev: build
	docker save $(IMAGE_FULL) | sudo k3s ctr images import -
	@echo "✅ $(IMAGE_FULL) imported into k3s"

# Push the Docker image to the registry
.PHONY: push
push:
	docker push $(IMAGE_FULL)

# Scan the Docker image with Trivy
.PHONY: scan
scan:
	trivy image $(IMAGE_FULL)

# Run interactive shell in container
.PHONY: shell
shell:
	docker run --rm -it --entrypoint /bin/sh $(IMAGE_FULL)

# Smoke test: uid 1000 resolves (ssh needs it) and git + ssh are present
.PHONY: test
test: build
	docker run --rm --entrypoint /bin/sh $(IMAGE_FULL) -c \
		'set -e; id; getent passwd 1000; git --version; ssh -V; echo "✅ ok"'

# Default target
.PHONY: help
help:
	@echo "Available targets:"
	@echo "  build  - Build the Docker image"
	@echo "  dev    - Build and import into k3s (development inner loop)"
	@echo "  push   - Push the Docker image to the registry"
	@echo "  scan   - Scan the Docker image with Trivy"
	@echo "  shell  - Run the image and exec into it with an interactive shell"
	@echo "  test   - Verify uid 1000 resolves and git/ssh work"
