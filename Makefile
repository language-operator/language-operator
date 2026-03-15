.PHONY: help build test setup-hooks install upgrade uninstall k8s-status

# Build the operator binary
build:
	@cd src && $(MAKE) build

# Set up git hooks for development
setup-hooks:
	@./scripts/setup-hooks

# Run Go tests
test:
	@cd src && $(MAKE) test

# Install the Helm chart using chart/values.local.yaml
install:
	@cd chart && $(MAKE) install

# Upgrade the Helm release using chart/values.local.yaml
upgrade:
	@cd chart && $(MAKE) upgrade

# Uninstall the Helm release
uninstall:
	@cd chart && $(MAKE) uninstall

# Check Kubernetes resources status
k8s-status:
	@echo "Language Operator Resources:"
	@kubectl get languageclusters,languageagents,languageagenttasks,languagetools,languagemodels -A
	@echo ""
	@echo "Operator Status:"
	@kubectl get pods -n language-operator

# Show help
help:
	@echo "Targets:"
	@echo "  build        - Build operator binary"
	@echo "  test         - Run Go test suite"
	@echo "  setup-hooks  - Install git pre-commit hooks"
	@echo "  install      - Install Helm chart (chart/values.local.yaml)"
	@echo "  upgrade      - Upgrade Helm release"
	@echo "  uninstall    - Uninstall Helm release"
	@echo "  k8s-status   - Check status of all language resources"
