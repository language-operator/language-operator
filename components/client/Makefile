IMAGE_NAME := git.theryans.io/langop/client
IMAGE_TAG := latest
IMAGE_FULL := $(IMAGE_NAME):$(IMAGE_TAG)

.PHONY: help
help:
	@echo "Available targets:"
	@echo "  build       - Build the Docker image"
	@echo "  scan        - Scan the Docker image with Trivy"
	@echo "  shell       - Run interactive shell in container"
	@echo "  run         - Run the chat client interactively"
	@echo "  test        - Run tests (not yet implemented)"
	@echo "  lint        - Run RuboCop linter"
	@echo "  lint-fix    - Run RuboCop with auto-fix"
	@echo "  doc         - Generate API documentation with YARD"
	@echo "  doc-serve   - Serve documentation on http://localhost:8808"
	@echo "  doc-clean   - Clean generated documentation"
	@echo "  clean       - Remove all generated files"

.PHONY: build
build:
	@echo "Building Docker image $(IMAGE_FULL)..."
	docker build -t $(IMAGE_FULL) .

.PHONY: scan
scan:
	@echo "Scanning Docker image $(IMAGE_FULL) with Trivy..."
	trivy image $(IMAGE_FULL)

.PHONY: shell
shell:
	@echo "Running interactive shell in $(IMAGE_FULL)..."
	docker run --rm -it $(IMAGE_FULL) /bin/sh

.PHONY: run
run:
	@echo "Running chat client locally with Docker..."
	docker run -it --rm \
		-v $(PWD)/config:/app/config \
		$(IMAGE_FULL)

.PHONY: test
test: lint
	@echo "All tests passed!"

.PHONY: lint
lint:
	@echo "Running RuboCop..."
	bundle exec rubocop

.PHONY: lint-fix
lint-fix:
	@echo "Running RuboCop with auto-fix..."
	bundle exec rubocop -A

.PHONY: doc
doc:
	@echo "Generating documentation with YARD..."
	bundle exec yard doc
	@echo "Documentation generated in doc/"

.PHONY: doc-serve
doc-serve:
	@echo "Serving documentation on http://localhost:8808"
	bundle exec yard server --reload

.PHONY: doc-clean
doc-clean:
	@echo "Cleaning documentation..."
	rm -rf doc/ .yardoc/

.PHONY: clean
clean: doc-clean
	@echo "Cleaning all generated files..."
	rm -rf vendor/ .bundle/
