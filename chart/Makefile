# Makefile for Language Operator Helm Chart

# Chart configuration
CHART_NAME := language-operator
RELEASE_NAME := language-operator
NAMESPACE := kube-system
CHART_DIR := .
VALUES_FILE := values.local.yaml

.PHONY: help
help: ## Display this help message
	@echo "Language Operator Helm Chart - Makefile Commands"
	@echo ""
	@awk 'BEGIN {FS = ":.*##"; printf "Usage:\n  make \033[36m<target>\033[0m\n\nTargets:\n"} /^[a-zA-Z_-]+:.*?##/ { printf "  \033[36m%-20s\033[0m %s\n", $$1, $$2 }' $(MAKEFILE_LIST)

.PHONY: test
test: ## Run Helm chart tests (lint)
	@echo "Testing Helm chart..."
	helm lint $(CHART_DIR) --values $(VALUES_FILE)
	@echo "✓ All tests passed!"

.PHONY: install
install: ## Install the Helm chart
	@echo "Installing $(CHART_NAME) as $(RELEASE_NAME) in namespace $(NAMESPACE)..."
	helm install $(RELEASE_NAME) $(CHART_DIR) \
		--namespace $(NAMESPACE) \
		--create-namespace \
		--values $(VALUES_FILE) \
		--wait \
		--timeout 5m
	@echo "✓ Installation complete!"

.PHONY: upgrade
upgrade: ## Upgrade the Helm release
	@echo "Upgrading $(RELEASE_NAME)..."
	helm upgrade $(RELEASE_NAME) $(CHART_DIR) \
		--namespace $(NAMESPACE) \
		--values $(VALUES_FILE) \
		--wait \
		--timeout 5m
	@echo "✓ Upgrade complete!"

.PHONY: uninstall
uninstall: ## Uninstall the Helm release
	@echo "Uninstalling $(RELEASE_NAME)..."
	helm uninstall $(RELEASE_NAME) --namespace $(NAMESPACE)
	@echo "✓ Uninstall complete!"

# Default target
.DEFAULT_GOAL := help
