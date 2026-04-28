GIT_SHA   := $(shell git rev-parse --short HEAD)
DEV_IMAGE := language-operator:$(GIT_SHA)

.PHONY: help build test dev setup-hooks install upgrade uninstall wipe k8s-status agent-supervisor

# Build the operator binary
build:
	@cd src && $(MAKE) build

# Set up git hooks for development
setup-hooks:
	@./scripts/setup-hooks

# Run Go tests
test:
	@cd src && $(MAKE) test

# Build, load into k3s, and upgrade the release (development inner loop)
dev:
	@cd src && $(MAKE) build
	docker build -t $(DEV_IMAGE) .
	docker save $(DEV_IMAGE) | sudo k3s ctr images import -
	helm upgrade --install language-operator charts/language-operator \
		--namespace language-operator \
		--create-namespace \
		--values charts/language-operator/values.local.yaml \
		--set image.repository=docker.io/library/language-operator \
		--set-string image.tag=$(GIT_SHA) \
		--set image.pullPolicy=Never \
		--wait --timeout 2m

# Install the Helm chart using charts/language-operator/values.local.yaml
install:
	@cd charts/language-operator && $(MAKE) install

# Upgrade the Helm release using charts/language-operator/values.local.yaml
upgrade:
	@cd charts/language-operator && $(MAKE) upgrade

# Uninstall the Helm release
uninstall:
	@cd charts/language-operator && $(MAKE) uninstall

# Install the runtimes chart (requires language-operator chart with CRDs installed first)
install-runtimes:
	helm upgrade --install language-operator-runtimes charts/language-operator-runtimes \
		--namespace language-operator \
		--create-namespace \
		--wait --timeout 2m

# Upgrade the runtimes release
upgrade-runtimes:
	helm upgrade language-operator-runtimes charts/language-operator-runtimes \
		--namespace language-operator \
		--wait --timeout 2m

# Uninstall the runtimes release
uninstall-runtimes:
	helm uninstall language-operator-runtimes --namespace language-operator

# Wipe everything — delete all CRs, uninstall the chart, delete CRDs and namespace.
# Use this to get back to a clean cluster state.
wipe:
	@echo "Deleting all language operator custom resources..."
	@kubectl delete languageagents --all -A --ignore-not-found --wait=true 2>/dev/null || true
	@kubectl delete languageclusters --all -A --ignore-not-found --wait=true 2>/dev/null || true
	@kubectl delete languagemodels --all -A --ignore-not-found --wait=true 2>/dev/null || true
	@kubectl delete languagetools --all -A --ignore-not-found --wait=true 2>/dev/null || true
	@kubectl delete languagepersonas --all -A --ignore-not-found --wait=true 2>/dev/null || true
	@kubectl delete languageagentruntimes --all -A --ignore-not-found --wait=true 2>/dev/null || true
	@echo "Uninstalling Helm release..."
	@helm uninstall language-operator --namespace language-operator --ignore-not-found 2>/dev/null || true
	@echo "Deleting CRDs..."
	@kubectl get crds -o name 2>/dev/null | grep langop.io | xargs -r kubectl delete --ignore-not-found
	@echo "Deleting namespace..."
	@kubectl delete namespace language-operator --ignore-not-found 2>/dev/null || true
	@echo "Done."

# Check Kubernetes resources status
k8s-status:
	@echo "Language Operator Resources:"
	@kubectl get languageclusters,languageagents,languagetools,languagemodels -A
	@echo ""
	@echo "Operator Status:"
	@kubectl get pods -n language-operator

dev-supervisor:
	claude "/delegate"

dev-worker-%:
	claude "/watch $*"

# Show help
help:
	@echo "Targets:"
	@echo "  build        - Build operator binary"
	@echo "  test         - Run Go test suite"
	@echo "  dev          - Build, load into k3s, and upgrade (inner loop)"
	@echo "  setup-hooks  - Install git pre-commit hooks"
	@echo "  install      - Install Helm chart (charts/language-operator/values.local.yaml)"
	@echo "  upgrade      - Upgrade Helm release"
	@echo "  uninstall    - Uninstall Helm release"
	@echo "  wipe         - Delete all CRs, CRDs, chart, and namespace (start from scratch)"
	@echo "  k8s-status   - Check status of all language resources"
	@echo "  dev-supervisor   - Run the supervisor agent (triage issues into queues)"
	@echo "  dev-worker-N     - Run worker agent for queue N (0, 1, or 2)"
