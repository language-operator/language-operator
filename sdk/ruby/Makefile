.PHONY: help build test install console docs clean version-bump lint

.DEFAULT_GOAL := help

help: ## Show this help message
	@echo 'Usage: make [target]'
	@echo ''
	@echo 'Available targets:'
	@awk 'BEGIN {FS = ":.*?## "} /^[a-zA-Z_-]+:.*?## / {printf "  %-20s %s\n", $$1, $$2}' $(MAKEFILE_LIST)

build: ## Build the gem
	@echo "Building language-operator gem..."
	@gem build language-operator.gemspec
	@echo "✅ Gem built successfully"

test: ## Run the test suite
	@echo "Running tests..."
	@bundle exec rspec
	@echo "✅ All tests passed"

install: build ## Build and install the gem locally
	@echo "Installing gem..."
	@gem install language-operator-*.gem
	@echo "✅ Gem installed successfully"

console: ## Open an IRB console with the gem loaded
	@bundle exec rake console

docs: ## Generate YARD documentation
	@echo "Generating documentation..."
	@bundle exec yard doc
	@echo "✅ Documentation generated in doc/"

lint: ## Run RuboCop linter
	@echo "Running RuboCop..."
	@bundle exec rubocop
	@echo "✅ No linting issues found"

lint-fix: ## Auto-fix RuboCop issues
	@echo "Auto-fixing RuboCop issues..."
	@bundle exec rubocop --autocorrect-all

clean: ## Clean build artifacts
	@echo "Cleaning build artifacts..."
	@rm -f language-operator-*.gem
	@rm -rf doc/
	@rm -rf .yardoc/
	@echo "✅ Cleaned"

version-bump: ## Bump version (usage: make version-bump TYPE=patch|minor|major)
	@if [ -z "$(TYPE)" ]; then \
		echo "❌ Error: TYPE not specified"; \
		echo "Usage: make version-bump TYPE=patch|minor|major"; \
		exit 1; \
	fi
	@./bin/bump-version $(TYPE)

version-bump-patch: ## Bump patch version (0.1.0 -> 0.1.1)
	@./bin/bump-version patch

version-bump-minor: ## Bump minor version (0.1.0 -> 0.2.0)
	@./bin/bump-version minor

version-bump-major: ## Bump major version (0.1.0 -> 1.0.0)
	@./bin/bump-version major

# CI targets
ci-test: test lint ## Run CI test suite (tests + linting)

# Development workflow
dev-setup: ## Install development dependencies
	@echo "Installing dependencies..."
	@bundle install
	@echo "✅ Development environment ready"

dev-watch: ## Run tests in watch mode
	@bundle exec guard
