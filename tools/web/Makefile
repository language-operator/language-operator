# Image configuration
IMAGE_NAME := based/svc/web
IMAGE_TAG := latest
IMAGE_FULL := $(IMAGE_NAME):$(IMAGE_TAG)

# Default target
.PHONY: help
help:
	@echo "Available targets:"
	@echo "  build     - Build the Docker image"
	@echo "  scan      - Scan the Docker image with Trivy"
	@echo "  shell     - Run the image and exec into it with an interactive shell"
	@echo "  env       - Display sorted list of environment variables in the image"
	@echo "  run       - Run the web MCP server on port 8080"
	@echo "  test      - Run the server and test the web endpoints"
	@echo "  lint      - Run RuboCop linter"
	@echo "  lint-fix  - Auto-fix RuboCop issues"
	@echo "  doc       - Generate YARD documentation"
	@echo "  doc-serve - Generate and serve YARD documentation"
	@echo "  doc-clean - Remove generated documentation"
	@echo "  clean     - Remove all generated files"

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
	docker run --rm -p 8080:80 --name web-server $(IMAGE_FULL)

# Test the server endpoints
.PHONY: test
test:
	@echo "Testing health endpoint..."
	@curl -s http://localhost:8080/health | jq .
	@echo ""
	@echo "Listing available tools..."
	@curl -s http://localhost:8080/tools | jq .
	@echo ""
	@echo "Testing web search for 'ruby programming'..."
	@curl -s -X POST http://localhost:8080/tools/call \
		-H "Content-Type: application/json" \
		-d '{"name":"web_search","arguments":{"query":"ruby programming","max_results":3}}' | jq -r '.content[0].text'
	@echo ""
	@echo "Testing web status for 'https://example.com'..."
	@curl -s -X POST http://localhost:8080/tools/call \
		-H "Content-Type: application/json" \
		-d '{"name":"web_status","arguments":{"url":"https://example.com"}}' | jq -r '.content[0].text'
	@echo ""
	@echo "Testing web headers for 'https://example.com'..."
	@curl -s -X POST http://localhost:8080/tools/call \
		-H "Content-Type: application/json" \
		-d '{"name":"web_headers","arguments":{"url":"https://example.com"}}' | jq -r '.content[0].text' | head -10
	@echo ""
	@echo "Testing web fetch for 'https://example.com'..."
	@curl -s -X POST http://localhost:8080/tools/call \
		-H "Content-Type: application/json" \
		-d '{"name":"web_fetch","arguments":{"url":"https://example.com"}}' | jq -r '.content[0].text' | head -15

# Lint Ruby code
.PHONY: lint
lint:
	@echo "Running RuboCop linter..."
	bundle exec rubocop

# Auto-fix linting issues
.PHONY: lint-fix
lint-fix:
	@echo "Auto-fixing RuboCop issues..."
	bundle exec rubocop -A

# Generate documentation
.PHONY: doc
doc:
	@echo "Generating documentation with YARD..."
	bundle exec yard doc
	@echo "Documentation generated in doc/"

# Serve documentation
.PHONY: doc-serve
doc-serve: doc
	@echo "Serving documentation at http://localhost:8808"
	bundle exec yard server --reload

# Clean documentation
.PHONY: doc-clean
doc-clean:
	@echo "Cleaning documentation..."
	rm -rf doc/ .yardoc

# Clean all generated files
.PHONY: clean
clean: doc-clean
	@echo "Cleaning all generated files..."
	rm -rf vendor/ .bundle/ Gemfile.lock
