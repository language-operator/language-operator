.PHONY: help build run lint doc clean

help:
	@echo "Based CLI Agent - Make targets:"
	@echo "  build  - Build Docker image"
	@echo "  run    - Run CLI agent"
	@echo "  lint   - Run RuboCop linter"
	@echo "  doc    - Generate YARD documentation"
	@echo "  clean  - Clean generated files"

build:
	@echo "Building CLI agent image..."
	docker build -t based/agent-cli:latest .

run:
	@echo "Running CLI agent..."
	docker compose -f ../../docker-compose.yml run --rm agent-cli

lint:
	@echo "Running RuboCop..."
	bundle exec rubocop

doc:
	@echo "Generating YARD documentation..."
	bundle exec yard doc

clean:
	@echo "Cleaning generated files..."
	rm -rf .yardoc doc/
