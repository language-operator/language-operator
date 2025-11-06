# Image configuration
IMAGE_NAME := git.theryans.io/language-operator/web-tool
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
	@echo "  spec      - Run RSpec unit tests"
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
	docker build \
		-t $(IMAGE_FULL) .

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

# Run RSpec unit tests
.PHONY: spec
spec:
	@echo "Running RSpec unit tests..."
	bundle exec rspec --format documentation

# Test the server endpoints using MCP JSON-RPC protocol
.PHONY: test
test:
	@echo "Testing health endpoint..."
	@curl -s http://localhost:8080/health | jq .
	@echo ""
	@echo "Initializing MCP session..."
	@curl -s -X POST http://localhost:8080/mcp \
		-H "Content-Type: application/json" \
		-d '{"jsonrpc":"2.0","id":1,"method":"initialize","params":{"protocolVersion":"2024-11-05","capabilities":{},"clientInfo":{"name":"test","version":"1.0"}}}' | jq '.result.serverInfo'
	@echo ""
	@echo "Listing available tools via MCP..."
	@curl -s -X POST http://localhost:8080/mcp \
		-H "Content-Type: application/json" \
		-d '{"jsonrpc":"2.0","id":2,"method":"tools/list","params":{}}' | jq '.result.tools[] | {name, description}'
	@echo ""
	@echo "Testing web_status for 'https://example.com'..."
	@curl -s -X POST http://localhost:8080/mcp \
		-H "Content-Type: application/json" \
		-d '{"jsonrpc":"2.0","id":3,"method":"tools/call","params":{"name":"web_status","arguments":{"url":"https://example.com"}}}' | jq -r '.result.content[0].text'
	@echo ""
	@echo "Testing web_headers for 'https://example.com'..."
	@curl -s -X POST http://localhost:8080/mcp \
		-H "Content-Type: application/json" \
		-d '{"jsonrpc":"2.0","id":4,"method":"tools/call","params":{"name":"web_headers","arguments":{"url":"https://example.com"}}}' | jq -r '.result.content[0].text' | head -10
	@echo ""
	@echo "Testing web_fetch for 'https://example.com'..."
	@curl -s -X POST http://localhost:8080/mcp \
		-H "Content-Type: application/json" \
		-d '{"jsonrpc":"2.0","id":5,"method":"tools/call","params":{"name":"web_fetch","arguments":{"url":"https://example.com","html":false}}}' | jq -r '.result.content[0].text' | head -15
	@echo ""
	@echo "Testing web_search for 'ruby programming'..."
	@curl -s -X POST http://localhost:8080/mcp \
		-H "Content-Type: application/json" \
		-d '{"jsonrpc":"2.0","id":6,"method":"tools/call","params":{"name":"web_search","arguments":{"query":"ruby programming","max_results":3}}}' | jq -r '.result.content[0].text' | head -20

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
