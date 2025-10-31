# Image configuration
IMAGE_NAME := git.theryans.io/langop/sms-tool
IMAGE_TAG := latest
IMAGE_FULL := $(IMAGE_NAME):$(IMAGE_TAG)

# SMS configuration for testing (override these)
SMS_PROVIDER ?= twilio
TWILIO_ACCOUNT_SID ?= ACxxxxxxxxxxxxxxxxxxxxxxxxxxxx
TWILIO_AUTH_TOKEN ?= your_auth_token
TWILIO_FROM ?= +1234567890

# Default target
.PHONY: help
help:
	@echo "Available targets:"
	@echo "  build     - Build the Docker image"
	@echo "  scan      - Scan the Docker image with Trivy"
	@echo "  shell     - Run the image and exec into it with an interactive shell"
	@echo "  env       - Display sorted list of environment variables in the image"
	@echo "  run       - Run the SMS MCP server on port 8080"
	@echo "  test      - Run the server and test the SMS endpoints"
	@echo "  lint      - Run RuboCop linter"
	@echo "  lint-fix  - Auto-fix RuboCop issues"
	@echo "  doc       - Generate YARD documentation"
	@echo "  doc-serve - Generate and serve YARD documentation"
	@echo "  doc-clean - Remove generated documentation"
	@echo "  clean     - Remove all generated files"
	@echo ""
	@echo "SMS Configuration (set via environment):"
	@echo "  For Twilio:"
	@echo "    SMS_PROVIDER=twilio"
	@echo "    TWILIO_ACCOUNT_SID - Your Twilio Account SID"
	@echo "    TWILIO_AUTH_TOKEN  - Your Twilio Auth Token"
	@echo "    TWILIO_FROM        - Your Twilio phone number"
	@echo ""
	@echo "  For Vonage/Nexmo:"
	@echo "    SMS_PROVIDER=vonage"
	@echo "    VONAGE_API_KEY     - Your Vonage API key"
	@echo "    VONAGE_API_SECRET  - Your Vonage API secret"
	@echo "    VONAGE_FROM        - Your Vonage sender ID"

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
	docker run --rm -it \
		-e SMS_PROVIDER=$(SMS_PROVIDER) \
		-e TWILIO_ACCOUNT_SID=$(TWILIO_ACCOUNT_SID) \
		-e TWILIO_AUTH_TOKEN=$(TWILIO_AUTH_TOKEN) \
		-e TWILIO_FROM=$(TWILIO_FROM) \
		$(IMAGE_FULL) /bin/sh

# Display sorted list of environment variables in the image
.PHONY: env
env:
	docker run --rm $(IMAGE_FULL) /bin/sh -c 'env | sort'

# Run the MCP server
.PHONY: run
run:
	docker run --rm -p 8080:80 \
		-e SMS_PROVIDER=$(SMS_PROVIDER) \
		-e TWILIO_ACCOUNT_SID=$(TWILIO_ACCOUNT_SID) \
		-e TWILIO_AUTH_TOKEN=$(TWILIO_AUTH_TOKEN) \
		-e TWILIO_FROM=$(TWILIO_FROM) \
		--name sms-server $(IMAGE_FULL)

# Test the server endpoints
.PHONY: test
test:
	@echo "Testing health endpoint..."
	@curl -s http://localhost:8080/health | jq .
	@echo ""
	@echo "Listing available tools..."
	@curl -s http://localhost:8080/tools | jq .
	@echo ""
	@echo "Checking SMS configuration..."
	@curl -s -X POST http://localhost:8080/tools/call \
		-H "Content-Type: application/json" \
		-d '{"name":"sms_config","arguments":{}}' | jq -r '.content[0].text'
	@echo ""
	@echo "Testing SMS configuration..."
	@curl -s -X POST http://localhost:8080/tools/call \
		-H "Content-Type: application/json" \
		-d '{"name":"test_sms","arguments":{}}' | jq -r '.content[0].text'
	@echo ""
	@echo "Note: To send an actual SMS, configure credentials and use:"
	@echo "  curl -X POST http://localhost:8080/tools/call \\"
	@echo "    -H 'Content-Type: application/json' \\"
	@echo "    -d '{\"name\":\"send_sms\",\"arguments\":{\"to\":\"+1234567890\",\"message\":\"Test message\"}}'"

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
