# Image configuration
IMAGE_NAME := git.theryans.io/langop/agent
IMAGE_TAG := latest
IMAGE_FULL := $(IMAGE_NAME):$(IMAGE_TAG)

# Default target
.PHONY: help
help:
	@echo "Available targets:"
	@echo "  build   - Build the Docker image"
	@echo "  scan    - Scan the Docker image with Trivy"
	@echo "  shell   - Run interactive shell in container"
	@echo "  run     - Run the agent framework"
	@echo "  lint    - Run RuboCop linter"
	@echo "  doc     - Generate YARD documentation"
	@echo "  clean   - Clean generated files"

# Build the Docker image
.PHONY: build
build:
	docker build -t $(IMAGE_FULL) .

# Scan the Docker image with Trivy
.PHONY: scan
scan:
	trivy image $(IMAGE_FULL)

# Run interactive shell in container
.PHONY: shell
shell:
	docker run --rm -it $(IMAGE_FULL) /bin/sh

# Run the agent framework
.PHONY: run
run:
	docker run --rm -it \
		-v $(PWD)/config:/app/agent/config \
		-e WORKSPACE_PATH=/workspace \
		-e AGENT_MODE=autonomous \
		$(IMAGE_FULL)

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
