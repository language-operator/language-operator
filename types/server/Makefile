# Image configuration
IMAGE_NAME := based/server
IMAGE_TAG := latest
IMAGE_FULL := $(IMAGE_NAME):$(IMAGE_TAG)

# Default target
.PHONY: help
help:
	@echo "Available targets:"
	@echo "  build      - Build the Docker image"
	@echo "  scan       - Scan the Docker image with Trivy"
	@echo "  shell      - Run the image and exec into it with an interactive shell"
	@echo "  env        - Display sorted list of environment variables in the image"
	@echo "  run        - Run the MCP server on port 8080"
	@echo "  test       - Run the server and test the endpoints"
	@echo "  lint       - Run RuboCop linter"
	@echo "  lint-fix   - Run RuboCop with auto-fix"
	@echo "  doc        - Generate API documentation with YARD"
	@echo "  doc-serve  - Serve documentation on http://localhost:8808"
	@echo "  doc-clean  - Remove generated documentation"

# Build the Docker image
.PHONY: build
build:
	docker build -t $(IMAGE_FULL) .

# Scan the Docker image with Trivy
.PHONY: scan
scan:
	trivy image $(IMAGE_FULL)

# Run the image and exec into it with an interactive shell
.PHONY: shell
shell:
	docker run --rm -it $(IMAGE_FULL) /bin/sh

# Display sorted list of environment variables in the image
.PHONY: env
env:
	docker run --rm $(IMAGE_FULL) /bin/sh -c 'env | sort'

# Run the MCP server
.PHONY: run
run:
	docker run --rm -p 8080:80 --name mcp-server $(IMAGE_FULL)

# Run with custom tools directory
.PHONY: run-dev
run-dev:
	docker run --rm -p 8080:80 -v $(PWD)/examples:/mcp --name mcp-server $(IMAGE_FULL)

# Test the server endpoints
.PHONY: test
test:
	@echo "Testing health endpoint..."
	@curl -s http://localhost:8080/health | jq .
	@echo ""
	@echo "Listing tools..."
	@curl -s http://localhost:8080/tools | jq .
	@echo ""
	@echo "Calling calculator tool..."
	@curl -s -X POST http://localhost:8080/tools/call \
		-H "Content-Type: application/json" \
		-d '{"name":"calculator","arguments":{"operation":"add","a":5,"b":3}}' | jq .

# Run RuboCop linter
.PHONY: lint
lint:
	bundle exec rubocop

# Run RuboCop with auto-fix
.PHONY: lint-fix
lint-fix:
	bundle exec rubocop -A

# Generate documentation with YARD
.PHONY: doc
doc:
	@echo "Generating documentation with YARD..."
	bundle exec yard doc
	@echo ""
	@echo "Documentation generated in doc/"
	@echo "Open doc/index.html in a browser to view"

# Serve documentation on localhost:8808
.PHONY: doc-serve
doc-serve: doc
	@echo "Starting documentation server on http://localhost:8808"
	bundle exec yard server --reload

# Clean generated documentation
.PHONY: doc-clean
doc-clean:
	rm -rf doc/ .yardoc
	@echo "Documentation cleaned"
