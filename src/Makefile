# Image URL to use all building/pushing image targets
IMG ?= ghcr.io/language-operator/controller:latest

# Get the currently used golang install path (in GOPATH/bin, unless GOBIN is set)
ifeq (,$(shell go env GOBIN))
GOBIN=$(shell go env GOPATH)/bin
else
GOBIN=$(shell go env GOBIN)
endif

# Setting SHELL to bash allows bash commands to be executed by recipes.
SHELL = /usr/bin/env bash -o pipefail
.SHELLFLAGS = -ec

.PHONY: all
all: build

##@ General

.PHONY: help
help: ## Display this help.
	@awk 'BEGIN {FS = ":.*##"; printf "\nUsage:\n  make \033[36m<target>\033[0m\n"} /^[a-zA-Z_0-9-]+:.*?##/ { printf "  \033[36m%-15s\033[0m %s\n", $$1, $$2 } /^##@/ { printf "\n\033[1m%s\033[0m\n", substr($$0, 5) } ' $(MAKEFILE_LIST)

##@ Development

.PHONY: manifests
manifests: controller-gen ## Generate CustomResourceDefinition objects.
	$(CONTROLLER_GEN) rbac:roleName=manager-role crd:allowDangerousTypes=true webhook paths="./..." output:crd:artifacts:config=config/crd/bases

.PHONY: generate
generate: controller-gen ## Generate code containing DeepCopy, DeepCopyInto, and DeepCopyObject method implementations.
	$(CONTROLLER_GEN) object:headerFile="hack/boilerplate.go.txt" paths="./..."

.PHONY: fmt
fmt: ## Run go fmt against code.
	go fmt ./...

.PHONY: vet
vet: ## Run go vet against code.
	go vet ./...

.PHONY: test
test: manifests generate fmt vet ## Run tests.
	go test $(shell go list ./... | grep -v '/testutil' | grep -v '/gen') -coverprofile cover.out -covermode=atomic

.PHONY: integration-test
integration-test: manifests generate fmt vet envtest ## Run integration tests against a real envtest API server.
	KUBEBUILDER_ASSETS="$(shell $(ENVTEST) use $(ENVTEST_K8S_VERSION) -p path)" \
		go test -tags integration -v -timeout 120s ./...

.PHONY: docs
docs: crd-ref-docs ## Generate CRD API reference documentation.
	$(CRD_REF_DOCS) --source-path=./api/v1alpha1 --config=.crd-ref-docs.yaml --renderer=markdown --output-path=./docs/api-reference.md

##@ Build

.PHONY: build
build: manifests generate fmt vet ## Build manager binary.
	go build -o bin/manager cmd/main.go

.PHONY: run
run: manifests generate fmt vet ## Run a controller from your host.
	go run ./cmd/main.go

.PHONY: scan
scan: ## Scan docker image with Trivy.
	trivy image ${IMG}

.PHONY: shell
shell: ## Run interactive shell in container.
	docker run --rm -it ${IMG} /bin/sh

##@ Helm Chart Support
# NOTE: To deploy the operator, use the Helm chart in ../charts/language-operator/

.PHONY: helm-crds
helm-crds: manifests ## Generate CRDs and copy to charts/language-operator/templates/crds/ (run after changing api/v1alpha1 types)
	mkdir -p ../charts/language-operator/templates/crds
	@for src in config/crd/bases/*.yaml; do \
	  dest=../charts/language-operator/templates/crds/$$(basename $$src); \
	  printf '{{- if .Values.crds.install }}\n' > "$$dest"; \
	  sed 's/    controller-gen.kubebuilder.io/    "helm.sh\/resource-policy": keep\n    controller-gen.kubebuilder.io/' "$$src" >> "$$dest"; \
	  printf '{{- end }}\n' >> "$$dest"; \
	done

##@ Build Dependencies

## Location to install dependencies to
LOCALBIN ?= $(shell pwd)/bin
$(LOCALBIN):
	mkdir -p $(LOCALBIN)

## Tool Binaries
CONTROLLER_GEN ?= $(LOCALBIN)/controller-gen
CRD_REF_DOCS ?= $(LOCALBIN)/crd-ref-docs
ENVTEST ?= $(LOCALBIN)/setup-envtest

## Tool Versions
CONTROLLER_TOOLS_VERSION ?= v0.20.1
CRD_REF_DOCS_VERSION ?= v0.3.0
ENVTEST_K8S_VERSION ?= 1.35.0

.PHONY: controller-gen
controller-gen: $(CONTROLLER_GEN) ## Download controller-gen locally if necessary.
$(CONTROLLER_GEN): $(LOCALBIN)
	test -s $(LOCALBIN)/controller-gen || GOBIN=$(LOCALBIN) go install sigs.k8s.io/controller-tools/cmd/controller-gen@$(CONTROLLER_TOOLS_VERSION)

.PHONY: crd-ref-docs
crd-ref-docs: $(CRD_REF_DOCS) ## Download crd-ref-docs locally if necessary.
$(CRD_REF_DOCS): $(LOCALBIN)
	test -s $(LOCALBIN)/crd-ref-docs || GOBIN=$(LOCALBIN) go install github.com/elastic/crd-ref-docs@$(CRD_REF_DOCS_VERSION)

.PHONY: envtest
envtest: $(ENVTEST) ## Download setup-envtest locally if necessary.
$(ENVTEST): $(LOCALBIN)
	test -s $(LOCALBIN)/setup-envtest || GOBIN=$(LOCALBIN) go install sigs.k8s.io/controller-runtime/tools/setup-envtest@latest
