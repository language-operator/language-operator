IMAGE_NAME := git.theryans.io/language-operator/client
IMAGE_TAG := latest
IMAGE_FULL := $(IMAGE_NAME):$(IMAGE_TAG)

# Include shared targets
include ../../Makefile.common

.PHONY: help
help:
	@echo "Available targets:"
	@echo "  build       - Build the Docker image"
	@echo "  scan        - Scan the Docker image with Trivy"
	@echo "  shell       - Run interactive shell in container"
	@echo "  run         - Run the chat client interactively"
	@echo "  test        - Run tests (linting)"
	@echo "  lint        - Run RuboCop linter"
	@echo "  lint-fix    - Run RuboCop with auto-fix"
	@echo "  doc         - Generate API documentation with YARD"
	@echo "  doc-serve   - Serve documentation on http://localhost:8808"
	@echo "  doc-clean   - Clean generated documentation"
	@echo "  clean       - Remove all generated files"

.PHONY: run
run:
	@echo "Running chat client locally with Docker..."
	docker run -it --rm \
		-v $(PWD)/config:/app/config \
		$(IMAGE_FULL)
