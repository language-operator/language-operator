# Image configuration
IMAGE_NAME := git.theryans.io/langop/ruby
IMAGE_TAG := latest
IMAGE_FULL := $(IMAGE_NAME):$(IMAGE_TAG)

# Default target
.PHONY: help
help:
	@echo "Available targets:"
	@echo "  build      - Build gem from SDK and then build Docker image"
	@echo "  build-gem  - Build langop gem from SDK source"
	@echo "  scan       - Scan the Docker image with Trivy"
	@echo "  test       - Test the Docker image"
	@echo "  shell      - Run interactive shell in container"
	@echo "  run        - Run the container"

# Build the gem from SDK source
.PHONY: build-gem
build-gem:
	@echo "Building langop gem from SDK..."
	cd ../../sdk/ruby && gem build langop.gemspec
	@echo "Copying gem to components/ruby..."
	cp ../../sdk/ruby/langop-*.gem .
	@echo "Gem built successfully"

# Build the Docker image (depends on gem being built)
.PHONY: build
build: build-gem
	docker build -t $(IMAGE_FULL) .

# Scan the Docker image with Trivy
.PHONY: scan
scan:
	trivy image $(IMAGE_FULL)

# Test the Docker image
.PHONY: test
test: ## Test the Docker image
	@echo "Testing ruby image with Langop gem..."
	docker run --rm $(IMAGE_FULL) ruby -e "require 'langop'; puts 'Langop gem loaded successfully'"
	@echo "All tests passed!"

# Run interactive shell in container
.PHONY: shell
shell:
	docker run --rm -it $(IMAGE_FULL) /bin/sh

# Run the container
.PHONY: run
run:
	docker run --rm -it $(IMAGE_FULL)
