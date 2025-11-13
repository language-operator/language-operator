# Image configuration
IMAGE_NAME := ghcr.io/language-operator/agent
IMAGE_TAG := latest
IMAGE_FULL := $(IMAGE_NAME):$(IMAGE_TAG)

# Include shared targets
include ../../Makefile.common

# Default target
.PHONY: help
help:
	@echo "Available targets:"
	@echo "  build       - Build the Docker image"
	@echo "  build-dev   - Build the Docker image with local gem"
	@echo "  push-dev    - Build and push dev image to all k8s nodes"
	@echo "  build-all   - Build all inherited images in order (base→agent)"
	@echo "  scan        - Scan the Docker image with Trivy"
	@echo "  shell       - Run interactive shell in container"
	@echo "  run         - Run the agent framework"
	@echo "  test        - Run tests (linting)"
	@echo "  lint        - Run RuboCop linter"
	@echo "  doc         - Generate YARD documentation"
	@echo "  clean       - Clean generated files"

# Build development image with local gem
.PHONY: build-dev
build-dev:
	@echo "Building local gem..."
	cd ../../../language-operator-gem && gem build language-operator.gemspec
	@echo "Copying gem to build context..."
	cp ../../../language-operator-gem/*.gem .
	@echo "Building Docker image..."
	docker build -f Dockerfile.dev -t $(IMAGE_NAME):dev -t $(IMAGE_NAME):$(IMAGE_TAG)-dev .
	@echo "Cleaning up gem file..."
	rm language-operator-*.gem

# Push development image to all k8s cluster nodes (containerd/k3s)
.PHONY: push-dev
push-dev: build-dev
	@echo "Pushing development image to all k8s nodes..."
	@for node in $$(kubectl get nodes -o jsonpath='{.items[*].metadata.name}'); do \
		echo "==> Pushing to node: $$node"; \
		docker save $(IMAGE_NAME):dev | ssh $$node 'sudo k3s ctr images import -' || echo "Failed to push to $$node"; \
	done
	@echo "✓ Development image pushed to all nodes!"

# Build all inherited images in dependency order
.PHONY: build-all
build-all:
	@echo "Building inherited image chain: base → agent"
	@echo ""
	@echo "==> Building base image..."
	cd ../base && $(MAKE) build
	@echo ""
	@echo "==> Building agent image..."
	$(MAKE) build
	@echo ""
	@echo "✓ All images built successfully!"

# Run the agent framework
.PHONY: run
run:
	docker run --rm -it \
		-v $(PWD)/config:/app/agent/config \
		-e WORKSPACE_PATH=/workspace \
		-e AGENT_MODE=autonomous \
		$(IMAGE_FULL)
