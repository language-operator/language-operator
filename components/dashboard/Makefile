# Image configuration
IMAGE_NAME := git.theryans.io/language-operator/dashboard
IMAGE_TAG := latest
IMAGE_FULL := $(IMAGE_NAME):$(IMAGE_TAG)

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

# Build the Docker image
.PHONY: build
build:
	docker build \
		-t $(IMAGE_FULL) .

# Scan the Docker image with Trivy
.PHONY: scan
scan:
	trivy image $(IMAGE_FULL)

# Run interactive shell in container
.PHONY: shell
shell:
	docker run --rm -it $(IMAGE_FULL) /bin/sh

# Run the dashboard server
.PHONY: run
run:
	docker run --rm -it \
		-p 8080:8080 \
		-e CLUSTER_NAME=test-cluster \
		-e NAMESPACE=default \
		-e PORT=8080 \
		$(IMAGE_FULL)

# Run tests
.PHONY: test
test: lint ## Run tests (linting)
	@echo "All tests passed!"

# Run RuboCop linter
.PHONY: lint
lint:
	bundle exec rubocop

# Generate YARD documentation
.PHONY: doc
doc:
	bundle exec yard doc

# Clean generated files
.PHONY: clean
clean:
	rm -rf .yardoc doc/
