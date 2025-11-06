# Image configuration
IMAGE_NAME := git.theryans.io/langop/ruby
IMAGE_TAG := latest
IMAGE_FULL := $(IMAGE_NAME):$(IMAGE_TAG)

# Default target
.PHONY: help
help:
	@echo "Available targets:"
	@echo "  build      - Build Docker image (installs gem from registry)"
	@echo "  scan       - Scan the Docker image with Trivy"
	@echo "  test       - Test the Docker image"
	@echo "  shell      - Run interactive shell in container"
	@echo "  run        - Run the container"

# Build the Docker image (installs gem from registry)
.PHONY: build
build:
	docker build \
		--build-arg REGISTRY_USERNAME="${REGISTRY_USERNAME}" \
		--build-arg REGISTRY_PASSWORD="${REGISTRY_PASSWORD}" \
		-t $(IMAGE_FULL) .

# Scan the Docker image with Trivy
.PHONY: scan
scan:
	trivy image $(IMAGE_FULL)

# Test the Docker image
.PHONY: test
test: ## Test the Docker image
	@echo "Testing ruby image with LanguageOperator gem..."
	docker run --rm $(IMAGE_FULL) ruby -e "require 'language_operator'; puts 'LanguageOperator gem loaded successfully'"
	@echo "All tests passed!"

# Run interactive shell in container
.PHONY: shell
shell:
	docker run --rm -it $(IMAGE_FULL) /bin/sh

# Run the container
.PHONY: run
run:
	docker run --rm -it $(IMAGE_FULL)
