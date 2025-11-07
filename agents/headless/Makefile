# Include shared targets from Makefile.common
include ../../Makefile.common

# Agent-specific configuration
IMAGE_NAME := git.theryans.io/language-operator/agent
IMAGE_TAG := latest-headless

# Override run target for headless agent
.PHONY: run
run:
	@echo "Running headless agent locally with Docker..."
	docker run --rm -it \
		-v $(PWD)/config:/app/headless/config \
		-e CONFIG_PATH=/app/headless/config/config.yaml \
		-e GOALS_PATH=/app/headless/config/goals.yaml \
		$(IMAGE_FULL)
