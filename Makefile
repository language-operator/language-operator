.PHONY: help k8s-status test test-unit test-integration setup-hooks dev-up dev-down dev-logs dev-status dev-clean dev-k3s-bridge dev-k3s-bridge-clean build-dashboard-server test-telemetry-integration

QA_PROMPT := "/task test"
ITERATE_PROMPT := "/task iterate"
PRIORITIZE_PROMPT := "/task prioritize"
DISTILL_PROMPT := "/task distill"

# Distill Claude's SCRATCH.md file
distill:
	@claude --dangerously-skip-permissions $(DISTILL_PROMPT)

# Use claude to prioritize the backlog
prioritize:
	@claude --dangerously-skip-permissions $(PRIORITIZE_PROMPT)

# Use claude to iterate on the backlog
iterate:
	@claude --dangerously-skip-permissions $(ITERATE_PROMPT)

# Use claude to find bugs
qa:
	@claude --dangerously-skip-permissions $(QA_PROMPT)

# Set up git hooks for development
setup-hooks:
	@./scripts/setup-hooks

# Check Kubernetes resources status
k8s-status:
	@echo "Language Operator Resources:"
	@kubectl get languageclusters,languageagents,languageclients,languagetools -A
	@echo ""
	@echo "Operator Status:"
	@kubectl get pods -n language-operator


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

# Development Environment Commands

dev-up:
	@echo "Starting development dashboard and database..."
	@docker compose up -d
	@echo ""
	@echo "⏳ Waiting for services to start..."
	@sleep 5
	@echo ""
	@echo "✅ Services running:"
	@echo "   📊 Dashboard:      http://localhost:3000"
	@echo "   🗄️  Database:       postgresql://dev:dev@localhost:5433/language_operator_dev"
	@echo "   🎛️  Prisma Studio:  http://localhost:5555"
	@echo ""
	@echo "📝 Database migrations and seeding are automatically applied during startup"
	@echo "🔑 Development login: james@theryans.io / password123"
	@echo ""
	@echo "📝 Next steps:"
	@echo "   • View logs: make dev-logs"
	@echo "   • Check status: make dev-status"
	@echo "   • Access via kubectl proxy for K8s service discovery"

dev-down:
	@echo "Stopping development environment..."
	@docker compose down

dev-logs:
	@docker compose logs -f

dev-status:
	@echo "📊 Development Environment Status:"
	@echo ""
	@echo "🐳 Docker Services:"
	@docker compose ps
	@echo ""
	@echo "☸️ Kubernetes Cluster:"
	@if kubectl get nodes 2>/dev/null | grep -q "Ready"; then \
		echo "   ✅ K3s cluster accessible"; \
		echo "   Nodes: $$(kubectl get nodes --no-headers | wc -l)"; \
	else \
		echo "   ❌ K3s cluster not accessible"; \
	fi
	@echo ""
	@echo "🌐 Kubectl Proxy:"
	@if docker compose ps kubectl-proxy | grep -q "Up"; then \
		echo "   ✅ kubectl proxy running"; \
	else \
		echo "   ❌ kubectl proxy not running"; \
	fi

dev-clean:
	@echo "🧹 Cleaning up development environment..."
	@docker compose down -v
	@echo "✓ Docker services stopped and volumes cleaned"

# Show help
help:
	@echo "Hi :-)"
	@echo ""
	@echo "Development Environment:"
	@echo "  dev-up            - Start dashboard and database"
	@echo "  dev-down          - Stop development environment"
	@echo "  dev-logs          - Show logs from all services"
	@echo "  dev-status        - Show status of development services"
	@echo "  dev-clean         - Clean up volumes and containers"
	@echo ""
	@echo "Building:"
	@echo "  build-dashboard-server - Build dashboard server with real telemetry integration"
	@echo ""
	@echo "Development Tools:"
	@echo "  docs              - Generate CRD API reference documentation"
	@echo "  setup-hooks       - Install git pre-commit hooks for code quality"
	@echo ""
	@echo "Testing:"
	@echo "  test              - Run all tests"
	@echo "  test-unit         - Run fast unit tests (no K8s required)"
	@echo "  test-integration  - Run integration tests (fake K8s client)"
	@echo "  test-telemetry-integration - Test telemetry integration with real ClickHouse adapter"
	@echo ""
	@echo "Kubernetes Operations:"
	@echo "  k8s-status        - Check status of all language resources"
	@echo ""
	@echo "Quick Start: run 'make dev-up' to start everything!"

fetch-synthesis-templates:
	@echo "Fetching synthesis templates from language-operator-gem..."
	@curl -fsSL https://raw.githubusercontent.com/language-operator/language-operator-gem/main/lib/language_operator/templates/agent_synthesis.tmpl -o src/pkg/synthesis/agent_synthesis.tmpl
	@curl -fsSL https://raw.githubusercontent.com/language-operator/language-operator-gem/main/lib/language_operator/templates/task_synthesis.tmpl -o src/pkg/synthesis/task_synthesis.tmpl
	@curl -fsSL https://raw.githubusercontent.com/language-operator/language-operator-gem/main/lib/language_operator/templates/persona_distillation.tmpl -o src/pkg/synthesis/persona_distillation.tmpl
	@echo "✓ Synthesis templates updated successfully!"

# Build dashboard server with real telemetry integration  
build-dashboard-server:
	@echo "🔨 Building dashboard server with real telemetry integration..."
	cd src && go build -o ../bin/dashboard-server ../cmd/dashboard/main.go
	@echo "✓ Dashboard server built successfully!"

# Test telemetry integration end-to-end
test-telemetry-integration: build-dashboard-server
	@echo "🧪 Testing telemetry integration end-to-end..."
	@echo "🧹 Cleaning up any existing servers on port 8080..."
	@-pkill -f "dashboard-server" || true
	@sleep 1
	@echo "🚀 Starting dashboard server in NoOp mode..."
	@TELEMETRY_ADAPTER_TYPE=noop PORT=8080 ./bin/dashboard-server > /tmp/dashboard-test.log 2>&1 &
	@echo "⏳ Waiting for server to start..."
	@sleep 3
	@echo "🏥 Testing health endpoint..."
	@curl -f http://localhost:8080/api/health | jq '.'
	@echo "📊 Testing agent executions endpoint..."
	@curl -f "http://localhost:8080/api/clusters/test/agents/test-agent/executions?limit=10" | jq '.'
	@echo "📈 Testing trace details endpoint..."
	@curl -f "http://localhost:8080/api/clusters/test/agents/test-agent/executions/exec_123/traces" | jq '.'
	@echo "🛑 Stopping server..."
	@-pkill -f "dashboard-server"
	@echo "✅ All telemetry integration tests passed!"
	@echo "📝 Server logs saved to /tmp/dashboard-test.log"