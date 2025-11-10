# Image configuration
IMAGE_NAME := git.theryans.io/language-operator/dashboard
IMAGE_TAG := latest
IMAGE_FULL := $(IMAGE_NAME):$(IMAGE_TAG)

# Include shared targets
include ../../Makefile.common

# Default target
.PHONY: help
help:
	@echo "Available targets:"
	@echo "  build       - Build the Docker image"
	@echo "  build-all   - Build all inherited images in order (base→dashboard)"
	@echo "  scan        - Scan the Docker image with Trivy"
	@echo "  shell       - Run interactive shell in container"
	@echo "  run         - Run the dashboard server"
	@echo "  test        - Run tests (linting)"
	@echo "  lint        - Run RuboCop linter"
	@echo "  doc         - Generate YARD documentation"
	@echo "  clean       - Clean generated files"

# Build all inherited images in dependency order
.PHONY: build-all
build-all:
	@echo "Building inherited image chain: base → dashboard"
	@echo ""
	@echo "==> Building base image..."
	cd ../base && $(MAKE) build
	@echo ""
	@echo "==> Building dashboard image..."
	$(MAKE) build
	@echo ""
	@echo "✓ All images built successfully!"

# Run the dashboard server
.PHONY: run
run:
	docker run --rm -it \
		-p 8080:8080 \
		-e CLUSTER_NAME=test-cluster \
		-e NAMESPACE=default \
		-e PORT=8080 \
		$(IMAGE_FULL)
