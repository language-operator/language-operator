# Image configuration
IMAGE_NAME := based/svc/doc
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
	@echo "  run        - Run the MCP man server on port 8080"
	@echo "  test       - Run the server and test the man page endpoints"
	@echo "  lint       - Run RuboCop linter"
	@echo "  lint-fix   - Run RuboCop with auto-fix"
	@echo "  doc        - Generate API documentation with YARD"
	@echo "  doc-serve  - Serve documentation on http://localhost:8808"
	@echo "  doc-clean  - Remove generated documentation"
	@echo "  clean      - Clean build artifacts"

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
	docker run --rm -p 8080:80 --name doc-server $(IMAGE_FULL)

# Test the server endpoints
.PHONY: test
test:
	@echo "Testing health endpoint..."
	@curl -s http://localhost:8080/health | jq .
	@echo ""
	@echo "Listing available tools..."
	@curl -s http://localhost:8080/tools | jq .
	@echo ""
	@echo "Getting man page for 'ls'..."
	@curl -s -X POST http://localhost:8080/tools/call \
		-H "Content-Type: application/json" \
		-d '{"name":"man","arguments":{"page":"ls"}}' | jq -r '.content[0].text' | head -20
	@echo ""
	@echo "Getting man page for 'printf' (section 1)..."
	@curl -s -X POST http://localhost:8080/tools/call \
		-H "Content-Type: application/json" \
		-d '{"name":"man","arguments":{"page":"printf","section":1}}' | jq -r '.content[0].text' | head -15
	@echo ""
	@echo "Searching for 'network' in man pages..."
	@curl -s -X POST http://localhost:8080/tools/call \
		-H "Content-Type: application/json" \
		-d '{"name":"man_search","arguments":{"keyword":"network"}}' | jq -r '.content[0].text' | head -10
	@echo ""
	@echo "Getting whatis for 'grep'..."
	@curl -s -X POST http://localhost:8080/tools/call \
		-H "Content-Type: application/json" \
		-d '{"name":"whatis","arguments":{"command":"grep"}}' | jq -r '.content[0].text'
	@echo ""
	@echo "Listing man sections..."
	@curl -s -X POST http://localhost:8080/tools/call \
		-H "Content-Type: application/json" \
		-d '{"name":"man_sections","arguments":{}}' | jq -r '.content[0].text'

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

# Clean all build artifacts
.PHONY: clean
clean: doc-clean
	rm -rf vendor/bundle .bundle
	find . -name "*.gem" -delete
	find . -name "*.rbc" -delete
