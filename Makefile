.PHONY: build help k8s-install k8s-uninstall k8s-status operator test-e2e

# Build all Docker images using the build script
build:
	@./scripts/build

# Install the language operator to Kubernetes
k8s-install:
	@cd src && $(MAKE) deploy

# Uninstall the language operator from Kubernetes
k8s-uninstall:
	@cd src && $(MAKE) undeploy

# Check Kubernetes resources status
k8s-status:
	@echo "Language Operator Resources:"
	@kubectl get languageclusters,languageagents,languageclients,languagetools -A
	@echo ""
	@echo "Operator Status:"
	@kubectl get pods -n language-operator-system

# Build and install the operator
operator:
	@cd src && $(MAKE) docker-build docker-push deploy

# Generate CRD API documentation
docs:
	@cd src && $(MAKE) docs

# Run tests
test:
	@echo "Running full test suite..."
	@echo ""
	@echo "==> Testing Language Operator (Go)"
	@cd src && $(MAKE) test
	@echo ""
	@echo "Note: Ruby SDK tests now run in separate repository (language-operator-gem)"
	@echo ""
	@echo "✓ All tests passed!"

# Run end-to-end integration tests
test-e2e:
	@echo "Running end-to-end integration tests..."
	@echo ""
	@cd test/e2e && go test -v -timeout 10m ./...
	@echo ""
	@echo "✓ E2E tests passed!"

# Show help
help:
	@echo "Hi :-)"
	@echo ""
	@echo "Build & Management:"
	@echo "  build          - Build all Docker images"
	@echo "  operator       - Build and deploy the language operator"
	@echo "  docs           - Generate CRD API reference documentation"
	@echo "  test           - Run all tests"
	@echo "  test-e2e       - Run end-to-end integration tests"
	@echo ""
	@echo "Kubernetes Operations:"
	@echo "  k8s-install    - Install the language operator to Kubernetes"
	@echo "  k8s-uninstall  - Uninstall the language operator from Kubernetes"
	@echo "  k8s-status     - Check status of all language resources"
	@echo ""
	@echo "Quick Start:"
	@echo "  1. Ensure you have a Kubernetes cluster (kind, minikube, etc.)"
	@echo "  2. Run 'make operator' to build and deploy the operator"
	@echo "  3. Apply LanguageCluster, LanguageAgent, and LanguageTool CRDs"
	@echo "  4. Check status with 'make k8s-status'"
