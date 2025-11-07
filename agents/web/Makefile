# Include shared targets from Makefile.common
include ../../Makefile.common

# Agent-specific configuration
IMAGE_NAME := git.theryans.io/language-operator/agent
IMAGE_TAG := latest-web

# Override run target for web agent (Rails + Vite)
.PHONY: run
run:
	@echo "Running web agent locally with Docker..."
	docker run --rm -it \
		-p 3000:3000 \
		-p 3036:3036 \
		-v $(PWD)/config:/app/web/config \
		-e RAILS_ENV=development \
		$(IMAGE_FULL)

# Override clean target to include Vite artifacts
.PHONY: clean
clean:
	rm -rf .yardoc doc/ public/vite public/vite-dev public/vite-test
