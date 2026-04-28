# Makefile for Language Operator Runtimes Helm Chart

CHART_NAME := language-operator-runtimes
RELEASE_NAME := language-operator-runtimes
NAMESPACE := language-operator
CHART_DIR := .

.PHONY: help
help: ## Display this help message
	@echo "Language Operator Runtimes Helm Chart - Makefile Commands"
	@echo ""
	@awk 'BEGIN {FS = ":.*##"; printf "Usage:\n  make \033[36m<target>\033[0m\n\nTargets:\n"} /^[a-zA-Z_-]+:.*?##/ { printf "  \033[36m%-20s\033[0m %s\n", $$1, $$2 }' $(MAKEFILE_LIST)

.PHONY: test
test: ## Run Helm chart tests (lint)
	@echo "Testing Helm chart..."
	helm lint $(CHART_DIR)
	@echo "✓ All tests passed!"

.PHONY: install
install: ## Install the Helm chart
	@echo "Installing $(CHART_NAME) as $(RELEASE_NAME) in namespace $(NAMESPACE)..."
	helm install $(RELEASE_NAME) $(CHART_DIR) \
		--namespace $(NAMESPACE) \
		--create-namespace \
		--wait \
		--timeout 5m
	@echo "✓ Installation complete!"

.PHONY: upgrade
upgrade: ## Upgrade the Helm release
	@echo "Upgrading $(RELEASE_NAME)..."
	helm upgrade $(RELEASE_NAME) $(CHART_DIR) \
		--namespace $(NAMESPACE) \
		--wait \
		--timeout 5m
	@echo "✓ Upgrade complete!"

.PHONY: uninstall
uninstall: ## Uninstall the Helm release
	@echo "Uninstalling $(RELEASE_NAME)..."
	helm uninstall $(RELEASE_NAME) --namespace $(NAMESPACE)
	@echo "✓ Uninstall complete!"

.DEFAULT_GOAL := help
