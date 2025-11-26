.PHONY: build help k8s-install k8s-uninstall k8s-status operator test test-unit test-integration

QA_PROMPT := "/task test"
ITERATE_PROMPT := "/task iterate"
PRIORITIZE_PROMPT := "/task prioritize"

# Use claude to prioritize the backlog
prioritize:
	@claude --dangerously-skip-permissions $(PRIORITIZE_PROMPT)

# Use claude to iterate on the backlog
iterate:
	@claude $(ITERATE_PROMPT)

# Use claude to find bugs
qa:
	@claude --dangerously-skip-permissions $(QA_PROMPT)

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

# Run fast unit tests (no Kubernetes required)
test-unit:
	@echo "Running fast unit tests..."
	@echo ""
	@bundle exec sh -c 'cd test/integration && go test -v -short -timeout 2m ./...'
	@echo ""
	@echo "✓ Unit tests passed!"

# Run integration tests (uses fake Kubernetes client)
test-integration:
	@echo "Running integration tests..."
	@echo ""
	@bundle exec sh -c 'cd test/integration && go test -v -timeout 5m ./...'
	@echo ""
	@echo "✓ Integration tests passed!"

# Show help
help:
	@echo "Hi :-)"
	@echo ""
	@echo "Build & Management:"
	@echo "  build             - Build all Docker images"
	@echo "  operator          - Build and deploy the language operator"
	@echo "  docs              - Generate CRD API reference documentation"
	@echo ""
	@echo "Testing:"
	@echo "  test              - Run all tests"
	@echo "  test-unit         - Run fast unit tests (no K8s required)"
	@echo "  test-integration  - Run integration tests (fake K8s client)"
	@echo ""
	@echo "Kubernetes Operations:"
	@echo "  k8s-install       - Install the language operator to Kubernetes"
	@echo "  k8s-uninstall     - Uninstall the language operator from Kubernetes"
	@echo "  k8s-status        - Check status of all language resources"
	@echo ""
	@echo "Quick Start:"
	@echo "  1. Ensure you have a Kubernetes cluster (kind, minikube, etc.)"
	@echo "  2. Run 'make operator' to build and deploy the operator"
	@echo "  3. Apply LanguageCluster, LanguageAgent, and LanguageTool CRDs"
	@echo "  4. Check status with 'make k8s-status'"

fetch-synthesis-templates:
	@echo "Fetching synthesis templates from language-operator-gem..."
	@curl -fsSL https://raw.githubusercontent.com/language-operator/language-operator-gem/main/lib/language_operator/templates/agent_synthesis.tmpl -o src/pkg/synthesis/agent_synthesis.tmpl
	@curl -fsSL https://raw.githubusercontent.com/language-operator/language-operator-gem/main/lib/language_operator/templates/task_synthesis.tmpl -o src/pkg/synthesis/task_synthesis.tmpl
	@curl -fsSL https://raw.githubusercontent.com/language-operator/language-operator-gem/main/lib/language_operator/templates/persona_distillation.tmpl -o src/pkg/synthesis/persona_distillation.tmpl
	@echo "✓ Synthesis templates updated successfully!"