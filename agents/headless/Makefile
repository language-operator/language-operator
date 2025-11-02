# Image configuration
IMAGE_NAME := git.theryans.io/langop/headless
IMAGE_TAG := latest
IMAGE_FULL := $(IMAGE_NAME):$(IMAGE_TAG)

# Default target
.PHONY: help
help:
	@echo "Available targets:"
	@echo "  build   - Build the Docker image"
	@echo "  scan    - Scan the Docker image with Trivy"
	@echo "  shell   - Run interactive shell in container"
	@echo "  run     - Run the headless agent"
	@echo "  test    - Run tests (linting)"
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

# Run the headless agent
.PHONY: run
run:
	docker run --rm -it \
		-v $(PWD)/config:/app/headless/config \
		-e CONFIG_PATH=/app/headless/config/config.yaml \
		-e GOALS_PATH=/app/headless/config/goals.yaml \
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
