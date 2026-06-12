# Image configuration
IMAGE_NAME := ghcr.io/language-operator/mcp-bridge
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

# Smoke test: bridge a trivial stdio MCP server and probe /health
.PHONY: test
test: build
	@echo "🧪 Starting bridged context7 stdio server on :8080 (Ctrl+C to stop)..."
	docker run --rm -it -p 8080:8080 $(IMAGE_FULL) \
		--stdio "npx -y @upstash/context7-mcp" \
		--outputTransport streamableHttp --stateful \
		--streamableHttpPath /mcp --healthEndpoint /health --port 8080

# Default target
.PHONY: help
help:
	@echo "Available targets:"
	@echo "  build  - Build the Docker image"
	@echo "  dev    - Build and import into k3s (development inner loop)"
	@echo "  push   - Push the Docker image to the registry"
	@echo "  scan   - Scan the Docker image with Trivy"
	@echo "  shell  - Run the image and exec into it with an interactive shell"
	@echo "  test   - Bridge a sample stdio MCP server and serve it on :8080"
