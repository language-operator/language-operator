# Image configuration
IMAGE_NAME := ghcr.io/language-operator/tool
IMAGE_TAG := latest
IMAGE_FULL := $(IMAGE_NAME):$(IMAGE_TAG)

# Include shared targets
include ../../Makefile.common

# Default target
.PHONY: help
help:
	@echo "Available targets:"
	@echo "  build      - Build the Docker image"
	@echo "  build-dev  - Build the Docker image with local gem"
	@echo "  push-dev   - Build and push dev image to all k8s nodes"
	@echo "  scan       - Scan the Docker image with Trivy"
	@echo "  shell      - Run the image and exec into it with an interactive shell"
	@echo "  env        - Display sorted list of environment variables in the image"
	@echo "  run        - Run the MCP server on port 8080"
	@echo "  run-dev    - Run with custom tools directory"
	@echo "  test       - Run the server and test the endpoints"
	@echo "  lint       - Run RuboCop linter"
	@echo "  lint-fix   - Run RuboCop with auto-fix"
	@echo "  doc        - Generate API documentation with YARD"
	@echo "  doc-serve  - Serve documentation on http://localhost:8808"
	@echo "  doc-clean  - Remove generated documentation"

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

# Display sorted list of environment variables in the image
.PHONY: env
env:
	docker run --rm $(IMAGE_FULL) /bin/sh -c 'env | sort'

# Run the MCP server
.PHONY: run
run:
	docker run --rm -p 8080:80 --name mcp-server $(IMAGE_FULL)

# Run with custom tools directory
.PHONY: run-dev
run-dev:
	docker run --rm -p 8080:80 -v $(PWD)/examples:/mcp --name mcp-server $(IMAGE_FULL)

# Override test to test server endpoints
.PHONY: test
test:
	@echo "Testing health endpoint..."
	@curl -s http://localhost:8080/health | jq .
	@echo ""
	@echo "Listing tools..."
	@curl -s http://localhost:8080/tools | jq .
	@echo ""
	@echo "Calling calculator tool..."
	@curl -s -X POST http://localhost:8080/tools/call \
		-H "Content-Type: application/json" \
		-d '{"name":"calculator","arguments":{"operation":"add","a":5,"b":3}}' | jq .
