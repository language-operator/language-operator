# Include shared targets from Makefile.common
include ../../Makefile.common

# Agent-specific configuration
IMAGE_NAME := git.theryans.io/language-operator/agent
IMAGE_TAG := latest-cli

# Override run target for CLI agent
.PHONY: run
run:
	@echo "Running CLI agent locally with Docker..."
	docker run -it --rm \
		-v $(PWD)/config:/app/agent/config \
		-e CONFIG_PATH=/app/agent/config/config.yaml \
		$(IMAGE_FULL)
