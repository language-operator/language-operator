REGISTRY  := ghcr.io/language-operator
IMAGE     := $(REGISTRY)/openclaw-adapter
GIT_SHA   := $(shell git rev-parse --short HEAD)
TAG       ?= $(GIT_SHA)

.PHONY: build publish test

build:
	docker build -t $(IMAGE):$(TAG) -t $(IMAGE):latest .

publish: build
	docker push $(IMAGE):$(TAG)
	docker push $(IMAGE):latest

test: build
	docker run --rm --entrypoint sh $(IMAGE):$(TAG) /app/test.sh
