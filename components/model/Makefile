# Image configuration
IMAGE_NAME := git.theryans.io/language-operator/model
IMAGE_TAG := latest
IMAGE_FULL := $(IMAGE_NAME):$(IMAGE_TAG)

# Include shared targets (build, scan, shell from Makefile.common)
include ../../Makefile.common

# Default target
.PHONY: help
help:
	@echo "Available targets:"
	@echo "  build          - Build the Docker image"
	@echo "  push           - Push the Docker image to the registry"
	@echo "  scan           - Scan the Docker image with Trivy"
	@echo "  shell          - Run the image and exec into it with an interactive shell"
	@echo "  run            - Run the container"
	@echo "  env            - Display sorted list of environment variables in the image"
	@echo "  test           - Test the proxy with a sample configuration"
	@echo "  test-local     - Test with local model (no API key needed)"
	@echo "  test-rate-limit - Test with rate limiting"
	@echo "  clean          - Clean up test artifacts"

# Push the Docker image to the registry
.PHONY: push
push:
	docker push $(IMAGE_FULL)

# Run the container
.PHONY: run
run:
	docker run --rm -it -p 4000:4000 $(IMAGE_FULL)

# Display sorted list of environment variables in the image
.PHONY: env
env:
	docker run --rm $(IMAGE_FULL) /bin/sh -c 'env | sort'

# Override test target for model-specific testing
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

# Override clean to remove test artifacts
.PHONY: clean
clean:
	rm -rf test-config
