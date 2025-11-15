# Image configuration
IMAGE_NAME := ghcr.io/language-operator/base
IMAGE_TAG := latest
IMAGE_FULL := $(IMAGE_NAME):$(IMAGE_TAG)

# Include shared targets
include ../../Makefile.common

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

# Push the Docker image to the registry
.PHONY: push
push:
	docker push $(IMAGE_FULL)

# Override test to include base-specific tests
.PHONY: test
test:
	@echo "Testing base image..."
	docker run --rm $(IMAGE_FULL) ruby --version
	docker run --rm $(IMAGE_FULL) bundle --version
	@echo "All tests passed!"

# Run the container
.PHONY: run
run:
	docker run --rm -it $(IMAGE_FULL)

# Display sorted list of environment variables in the image
.PHONY: env
env:
	docker run --rm $(IMAGE_FULL) /bin/sh -c 'env | sort'
