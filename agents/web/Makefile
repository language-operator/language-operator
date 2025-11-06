# Image configuration
IMAGE_NAME := git.theryans.io/langop/web
IMAGE_TAG := latest
IMAGE_FULL := $(IMAGE_NAME):$(IMAGE_TAG)

# Default target
.PHONY: help
help:
	@echo "Available targets:"
	@echo "  build   - Build the Docker image"
	@echo "  scan    - Scan the Docker image with Trivy"
	@echo "  shell   - Run interactive shell in container"
	@echo "  run     - Run the web agent (Rails + Vite)"
	@echo "  test    - Run tests (linting)"
	@echo "  lint    - Run RuboCop linter"
	@echo "  doc     - Generate YARD documentation"
	@echo "  clean   - Clean generated files"

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

# Run the web agent
.PHONY: run
run:
	docker run --rm -it \
		-p 3000:3000 \
		-p 3036:3036 \
		-v $(PWD)/config:/app/web/config \
		-e RAILS_ENV=development \
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
	rm -rf .yardoc doc/ public/vite public/vite-dev public/vite-test
