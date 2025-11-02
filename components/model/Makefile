# Image configuration
IMAGE_NAME := git.theryans.io/langop/model
IMAGE_TAG := latest
IMAGE_FULL := $(IMAGE_NAME):$(IMAGE_TAG)

# Default target
.PHONY: help
help:
	@echo "Available targets:"
	@echo "  build   - Build the Docker image"
	@echo "  push    - Push the Docker image to the registry"
	@echo "  scan    - Scan the Docker image with Trivy"
	@echo "  shell   - Run the image and exec into it with an interactive shell"
	@echo "  run     - Run the container"
	@echo "  env     - Display sorted list of environment variables in the image"
	@echo "  test    - Test the proxy with a sample configuration"
	@echo "  clean   - Clean up test artifacts"

# Build the Docker image
.PHONY: build
build:
	docker build -t $(IMAGE_FULL) .

# Push the Docker image to the registry
.PHONY: push
push:
	docker push $(IMAGE_FULL)

# Scan the Docker image with Trivy
.PHONY: scan
scan:
	trivy image $(IMAGE_FULL)

# Run the image and exec into it with an interactive shell
.PHONY: shell
shell:
	docker run --rm -it $(IMAGE_FULL) /bin/sh

# Run the container
.PHONY: run
run:
	docker run --rm -it -p 4000:4000 $(IMAGE_FULL)

# Display sorted list of environment variables in the image
.PHONY: env
env:
	docker run --rm $(IMAGE_FULL) /bin/sh -c 'env | sort'

# Test the proxy with a sample configuration
.PHONY: test
test:
	@echo "🧪 Testing LiteLLM proxy with sample config..."
	@mkdir -p test-config
	@echo '{"provider":"openai","modelName":"gpt-4","apiKeySecretRef":{"name":"openai-key","key":"api-key"}}' > test-config/model.json
	@echo "Starting proxy with test config (press Ctrl+C to stop)..."
	@docker run --rm -it \
		-v $(PWD)/test-config/model.json:/etc/langop/model.json:ro \
		-e OPENAI_KEY=test-key \
		-e DEBUG=true \
		-p 4000:4000 \
		$(IMAGE_FULL)

# Test with local model (no API key needed)
.PHONY: test-local
test-local:
	@echo "🧪 Testing LiteLLM proxy with local model config..."
	@mkdir -p test-config
	@echo '{"provider":"openai-compatible","modelName":"llama3.2","endpoint":"http://host.docker.internal:11434/v1"}' > test-config/model.json
	@echo "Starting proxy with local model config (press Ctrl+C to stop)..."
	@docker run --rm -it \
		-v $(PWD)/test-config/model.json:/etc/langop/model.json:ro \
		-e DEBUG=true \
		-p 4000:4000 \
		$(IMAGE_FULL)

# Test with rate limiting
.PHONY: test-rate-limit
test-rate-limit:
	@echo "🧪 Testing LiteLLM proxy with rate limiting..."
	@mkdir -p test-config
	@echo '{"provider":"openai","modelName":"gpt-4","rateLimits":{"requestsPerMinute":10,"tokensPerMinute":10000}}' > test-config/model.json
	@echo "Starting proxy with rate limits (press Ctrl+C to stop)..."
	@docker run --rm -it \
		-v $(PWD)/test-config/model.json:/etc/langop/model.json:ro \
		-e OPENAI_KEY=test-key \
		-e DEBUG=true \
		-p 4000:4000 \
		$(IMAGE_FULL)

# Clean up test artifacts
.PHONY: clean
clean:
	rm -rf test-config
