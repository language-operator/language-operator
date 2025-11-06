.PHONY: help
help:
	@echo "Langop CLI Agent - Make targets:"
	@echo "  build  - Build Docker image"
	@echo "  scan   - Scan Docker image with Trivy"
	@echo "  shell  - Run interactive shell in container"
	@echo "  run    - Run CLI agent"
	@echo "  test   - Run tests (linting)"
	@echo "  lint   - Run RuboCop linter"
	@echo "  doc    - Generate YARD documentation"
	@echo "  clean  - Clean generated files"

.PHONY: build
build:
	@echo "Building CLI agent image..."
	docker build \
		-t git.theryans.io/langop/cli:latest .

.PHONY: scan
scan:
	@echo "Scanning CLI agent image with Trivy..."
	trivy image git.theryans.io/langop/cli:latest

.PHONY: shell
shell:
	@echo "Running interactive shell in CLI agent container..."
	docker run --rm -it git.theryans.io/langop/cli:latest /bin/sh

.PHONY: run
run:
	@echo "Running CLI agent locally with Docker..."
	docker run -it --rm \
		-v $(PWD)/config:/app/agent/config \
		-e CONFIG_PATH=/app/agent/config/config.yaml \
		git.theryans.io/langop/cli:latest

.PHONY: test
test: lint ## Run tests (linting)
	@echo "All tests passed!"

.PHONY: lint
lint:
	@echo "Running RuboCop..."
	bundle exec rubocop

.PHONY: doc
doc:
	@echo "Generating YARD documentation..."
	bundle exec yard doc

.PHONY: clean
clean:
	@echo "Cleaning generated files..."
	rm -rf .yardoc doc/
