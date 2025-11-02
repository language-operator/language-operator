# Image configuration
IMAGE_NAME := git.theryans.io/langop/base
IMAGE_TAG := latest
IMAGE_FULL := $(IMAGE_NAME):$(IMAGE_TAG)

# Default target
.PHONY: help
help:
	@echo "Available targets:"
	@echo "  build - Build the Docker image"
	@echo "  push  - Push the Docker image to the registry"
	@echo "  scan  - Scan the Docker image with Trivy"
	@echo "  test  - Test the Docker image"
	@echo "  shell - Run the image and exec into it with an interactive shell"
	@echo "  run   - Run the container"
	@echo "  env   - Display sorted list of environment variables in the image"

# Build the Docker image
.PHONY: build
build:
	docker build -t $(IMAGE_FULL) .

# Push the Docker image to the registry
.PHONY: push
push:
	docker push $(IMAGE_FULL)

# Scan the Docker image with Trivy
.PHONY: scan
scan:
	trivy image $(IMAGE_FULL)

# Test the Docker image
.PHONY: test
test: ## Test the Docker image
	@echo "Testing base image..."
	docker run --rm $(IMAGE_FULL) ruby --version
	docker run --rm $(IMAGE_FULL) bundle --version
	@echo "All tests passed!"

# Run the image and exec into it with an interactive shell
.PHONY: shell
shell:
	docker run --rm -it $(IMAGE_FULL) /bin/sh

# Run the container
.PHONY: run
run:
	docker run --rm -it $(IMAGE_FULL)

# Display sorted list of environment variables in the image
.PHONY: env
env:
	docker run --rm $(IMAGE_FULL) /bin/sh -c 'env | sort'
