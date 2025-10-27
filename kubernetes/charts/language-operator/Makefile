# Makefile for Language Operator Helm Chart

# Chart configuration
CHART_NAME := language-operator
RELEASE_NAME := language-operator
NAMESPACE := language-operator

# Helm values
VALUES_FILE := values.yaml
VALUES_OVERRIDE :=

# Image configuration (can be overridden)
IMAGE_REPOSITORY ?= ghcr.io/based/language-operator
IMAGE_TAG ?= latest

.PHONY: help
help: ## Display this help message
	@echo "Language Operator Helm Chart - Makefile Commands"
	@echo ""
	@awk 'BEGIN {FS = ":.*##"; printf "Usage:\n  make \033[36m<target>\033[0m\n\nTargets:\n"} /^[a-zA-Z_-]+:.*?##/ { printf "  \033[36m%-20s\033[0m %s\n", $$1, $$2 }' $(MAKEFILE_LIST)

##@ Prerequisites

.PHONY: check-helm
check-helm: ## Check if helm is installed
	@which helm > /dev/null || (echo "❌ Helm is not installed. Please install helm first." && exit 1)
	@echo "✓ Helm is installed"

.PHONY: check-kubectl
check-kubectl: ## Check if kubectl is installed
	@which kubectl > /dev/null || (echo "❌ kubectl is not installed. Please install kubectl first." && exit 1)
	@echo "✓ kubectl is installed"

.PHONY: check-cluster
check-cluster: check-kubectl ## Check if kubectl can connect to cluster
	@kubectl cluster-info > /dev/null || (echo "❌ Cannot connect to Kubernetes cluster" && exit 1)
	@echo "✓ Connected to Kubernetes cluster"

.PHONY: prereqs
prereqs: check-helm check-kubectl check-cluster ## Check all prerequisites

##@ Chart Operations

.PHONY: lint
lint: check-helm ## Lint the Helm chart
	@echo "Linting Helm chart..."
	helm lint .

.PHONY: template
template: check-helm ## Generate Kubernetes manifests from the chart
	@echo "Generating Kubernetes manifests..."
	helm template $(RELEASE_NAME) . \
		--namespace $(NAMESPACE) \
		$(if $(VALUES_OVERRIDE),--values $(VALUES_OVERRIDE),) \
		--set image.repository=$(IMAGE_REPOSITORY) \
		--set image.tag=$(IMAGE_TAG)

.PHONY: template-debug
template-debug: check-helm ## Generate manifests with debug output
	@echo "Generating Kubernetes manifests with debug..."
	helm template $(RELEASE_NAME) . \
		--namespace $(NAMESPACE) \
		--debug \
		$(if $(VALUES_OVERRIDE),--values $(VALUES_OVERRIDE),) \
		--set image.repository=$(IMAGE_REPOSITORY) \
		--set image.tag=$(IMAGE_TAG)

.PHONY: dry-run
dry-run: prereqs ## Perform a dry-run installation
	@echo "Performing dry-run installation..."
	helm install $(RELEASE_NAME) . \
		--namespace $(NAMESPACE) \
		--create-namespace \
		--dry-run \
		--debug \
		$(if $(VALUES_OVERRIDE),--values $(VALUES_OVERRIDE),) \
		--set image.repository=$(IMAGE_REPOSITORY) \
		--set image.tag=$(IMAGE_TAG)

##@ Installation

.PHONY: install-crds
install-crds: check-kubectl ## Install only the CRDs
	@echo "Installing CRDs..."
	kubectl apply -f crds/

.PHONY: install
install: prereqs lint ## Install the Helm chart
	@echo "Installing $(CHART_NAME) chart as $(RELEASE_NAME) in namespace $(NAMESPACE)..."
	helm install $(RELEASE_NAME) . \
		--namespace $(NAMESPACE) \
		--create-namespace \
		$(if $(VALUES_OVERRIDE),--values $(VALUES_OVERRIDE),) \
		--set image.repository=$(IMAGE_REPOSITORY) \
		--set image.tag=$(IMAGE_TAG) \
		--wait \
		--timeout 5m
	@echo ""
	@echo "✓ Installation complete!"
	@echo ""
	@echo "Check the status with:"
	@echo "  make status"

.PHONY: install-dev
install-dev: prereqs ## Install with development settings
	@echo "Installing $(CHART_NAME) with development settings..."
	helm install $(RELEASE_NAME) . \
		--namespace $(NAMESPACE) \
		--create-namespace \
		--set image.repository=$(IMAGE_REPOSITORY) \
		--set image.tag=$(IMAGE_TAG) \
		--set image.pullPolicy=Always \
		--set replicaCount=1 \
		--set config.logging.level=debug \
		--set config.logging.development=true \
		--set podDisruptionBudget.enabled=false \
		--wait \
		--timeout 5m
	@echo "✓ Development installation complete!"

##@ Upgrade

.PHONY: upgrade
upgrade: prereqs lint ## Upgrade the Helm release
	@echo "Upgrading $(RELEASE_NAME)..."
	helm upgrade $(RELEASE_NAME) . \
		--namespace $(NAMESPACE) \
		$(if $(VALUES_OVERRIDE),--values $(VALUES_OVERRIDE),) \
		--set image.repository=$(IMAGE_REPOSITORY) \
		--set image.tag=$(IMAGE_TAG) \
		--wait \
		--timeout 5m
	@echo "✓ Upgrade complete!"

.PHONY: upgrade-force
upgrade-force: prereqs ## Force upgrade (recreate resources)
	@echo "Force upgrading $(RELEASE_NAME)..."
	helm upgrade $(RELEASE_NAME) . \
		--namespace $(NAMESPACE) \
		$(if $(VALUES_OVERRIDE),--values $(VALUES_OVERRIDE),) \
		--set image.repository=$(IMAGE_REPOSITORY) \
		--set image.tag=$(IMAGE_TAG) \
		--force \
		--wait \
		--timeout 5m
	@echo "✓ Force upgrade complete!"

##@ Uninstallation

.PHONY: uninstall
uninstall: check-helm check-kubectl ## Uninstall the Helm release
	@echo "Uninstalling $(RELEASE_NAME)..."
	helm uninstall $(RELEASE_NAME) --namespace $(NAMESPACE) || true
	@echo "✓ Uninstall complete!"
	@echo ""
	@echo "Note: CRDs are not automatically removed."
	@echo "To remove CRDs, run: make uninstall-crds"

.PHONY: uninstall-crds
uninstall-crds: check-kubectl ## Uninstall the CRDs (WARNING: removes all custom resources)
	@echo "⚠️  WARNING: This will delete all CRDs and ALL custom resources!"
	@echo "Are you sure? [y/N] " && read ans && [ $${ans:-N} = y ]
	@echo "Uninstalling CRDs..."
	kubectl delete -f crds/ --ignore-not-found=true
	@echo "✓ CRDs uninstalled"

.PHONY: uninstall-all
uninstall-all: uninstall uninstall-crds ## Uninstall everything including CRDs
	@echo "✓ Complete uninstallation finished"

.PHONY: clean-namespace
clean-namespace: check-kubectl ## Delete the namespace (WARNING: removes everything in namespace)
	@echo "⚠️  WARNING: This will delete the entire $(NAMESPACE) namespace!"
	@echo "Are you sure? [y/N] " && read ans && [ $${ans:-N} = y ]
	@echo "Deleting namespace $(NAMESPACE)..."
	kubectl delete namespace $(NAMESPACE) --ignore-not-found=true
	@echo "✓ Namespace deleted"

##@ Status and Info

.PHONY: status
status: check-helm ## Show Helm release status
	@helm status $(RELEASE_NAME) --namespace $(NAMESPACE)

.PHONY: list
list: check-helm ## List Helm releases in namespace
	@helm list --namespace $(NAMESPACE)

.PHONY: history
history: check-helm ## Show release history
	@helm history $(RELEASE_NAME) --namespace $(NAMESPACE)

.PHONY: get-values
get-values: check-helm ## Get the values used in the current release
	@helm get values $(RELEASE_NAME) --namespace $(NAMESPACE)

.PHONY: get-manifest
get-manifest: check-helm ## Get the manifest from the current release
	@helm get manifest $(RELEASE_NAME) --namespace $(NAMESPACE)

##@ Kubernetes Resources

.PHONY: pods
pods: check-kubectl ## List operator pods
	@kubectl get pods -n $(NAMESPACE) -l app.kubernetes.io/name=$(CHART_NAME)

.PHONY: logs
logs: check-kubectl ## Show operator logs
	@kubectl logs -n $(NAMESPACE) -l app.kubernetes.io/name=$(CHART_NAME) --tail=100 -f

.PHONY: logs-previous
logs-previous: check-kubectl ## Show previous operator logs
	@kubectl logs -n $(NAMESPACE) -l app.kubernetes.io/name=$(CHART_NAME) --previous --tail=100

.PHONY: describe
describe: check-kubectl ## Describe operator pods
	@kubectl describe pods -n $(NAMESPACE) -l app.kubernetes.io/name=$(CHART_NAME)

.PHONY: events
events: check-kubectl ## Show events in namespace
	@kubectl get events -n $(NAMESPACE) --sort-by='.lastTimestamp'

##@ Custom Resources

.PHONY: crds
crds: check-kubectl ## List installed CRDs
	@kubectl get crds | grep langop.io || echo "No Language Operator CRDs found"

.PHONY: get-all-resources
get-all-resources: check-kubectl ## Get all custom resources
	@echo "=== LanguageTools ==="
	@kubectl get languagetools --all-namespaces 2>/dev/null || echo "None found"
	@echo ""
	@echo "=== LanguageModels ==="
	@kubectl get languagemodels --all-namespaces 2>/dev/null || echo "None found"
	@echo ""
	@echo "=== LanguageAgents ==="
	@kubectl get languageagents --all-namespaces 2>/dev/null || echo "None found"
	@echo ""
	@echo "=== LanguagePersonas ==="
	@kubectl get languagepersonas --all-namespaces 2>/dev/null || echo "None found"
	@echo ""
	@echo "=== LanguageClients ==="
	@kubectl get languageclients --all-namespaces 2>/dev/null || echo "None found"

##@ Development

.PHONY: watch-pods
watch-pods: check-kubectl ## Watch operator pods
	@kubectl get pods -n $(NAMESPACE) -l app.kubernetes.io/name=$(CHART_NAME) -w

.PHONY: shell
shell: check-kubectl ## Get a shell in the operator pod
	@kubectl exec -it -n $(NAMESPACE) $$(kubectl get pods -n $(NAMESPACE) -l app.kubernetes.io/name=$(CHART_NAME) -o jsonpath='{.items[0].metadata.name}') -- /bin/sh

.PHONY: port-forward-metrics
port-forward-metrics: check-kubectl ## Port-forward to metrics endpoint (8443)
	@echo "Forwarding metrics port 8443 to localhost:8443..."
	@echo "Access metrics at: https://localhost:8443/metrics"
	@kubectl port-forward -n $(NAMESPACE) svc/$(RELEASE_NAME)-metrics 8443:8443

##@ Package

.PHONY: package
package: lint ## Package the Helm chart
	@echo "Packaging Helm chart..."
	helm package .
	@echo "✓ Chart packaged successfully"

.PHONY: package-deps
package-deps: check-helm ## Update chart dependencies
	@echo "Updating chart dependencies..."
	helm dependency update .

##@ Troubleshooting

.PHONY: debug
debug: ## Show debug information
	@echo "=== Helm Release Status ==="
	@helm status $(RELEASE_NAME) --namespace $(NAMESPACE) 2>&1 || echo "Release not found"
	@echo ""
	@echo "=== Operator Pods ==="
	@kubectl get pods -n $(NAMESPACE) -l app.kubernetes.io/name=$(CHART_NAME) 2>&1 || echo "Namespace not found"
	@echo ""
	@echo "=== Recent Events ==="
	@kubectl get events -n $(NAMESPACE) --sort-by='.lastTimestamp' 2>&1 | tail -10 || echo "No events"
	@echo ""
	@echo "=== CRDs ==="
	@kubectl get crds | grep langop.io 2>&1 || echo "No CRDs found"

.PHONY: reset
reset: uninstall-all ## Complete reset (uninstall all + clean namespace)
	@echo "Waiting for resources to terminate..."
	@sleep 5
	@echo "✓ Complete reset finished"

##@ Quick Commands

.PHONY: up
up: install ## Alias for install

.PHONY: down
down: uninstall ## Alias for uninstall

.PHONY: restart
restart: uninstall install ## Restart (uninstall + install)

.PHONY: reload
reload: upgrade ## Alias for upgrade

# Default target
.DEFAULT_GOAL := help
