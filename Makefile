.PHONY: build help up down logs ps restart clean-volumes run cli web headless

# Build all Docker images using the build script
build:
	@./scripts/build

# Start all services with docker compose
up:
	docker compose up -d

# Stop all services
down:
	docker compose down

# Run the CLI agent interactively (default)
run: cli

# Run CLI agent interactively
cli:
	docker compose run --rm agent-cli

# Start web agent (accessible at http://localhost:3000)
web:
	docker compose up agent-web

# Run headless agent (autonomous goal execution)
headless:
	docker compose run --rm agent-headless

# View logs from all services
logs:
	docker compose logs -f

# List running containers
ps:
	docker compose ps

# Restart all services
restart:
	docker compose restart

# Stop and remove all volumes (WARNING: removes data)
clean-volumes:
	docker compose down -v

# Show help
help:
	@echo "Based MCP Project - Available Targets:"
	@echo ""
	@echo "Build & Management:"
	@echo "  build         - Build all Docker images"
	@echo "  up            - Start all MCP tool services"
	@echo "  down          - Stop all services"
	@echo "  logs          - View logs from all services"
	@echo "  ps            - List running containers"
	@echo "  restart       - Restart all services"
	@echo "  clean-volumes - Stop and remove all volumes"
	@echo ""
	@echo "Agents:"
	@echo "  run / cli     - Run CLI agent (interactive terminal)"
	@echo "  web           - Start web agent (http://localhost:3000)"
	@echo "  headless      - Run headless agent (autonomous execution)"
	@echo ""
	@echo "Service Ports:"
	@echo "  web-agent - http://localhost:3000 (Web UI & API)"
	@echo "  server    - http://localhost:8080 (MCP Tools Server)"
	@echo "  doc       - http://localhost:8081 (Documentation Tool)"
	@echo "  email     - http://localhost:8082 (Email Tool)"
	@echo "  sms       - http://localhost:8083 (SMS Tool)"
	@echo "  web       - http://localhost:8084 (Web Tool)"
	@echo ""
	@echo "Quick Start:"
	@echo "  1. Configure agents/*/config/config.yaml with your LLM endpoint"
	@echo "  2. Run 'make build' to build all images"
	@echo "  3. Run 'make up' to start MCP tool services"
	@echo "  4. Run 'make cli' for terminal chat OR 'make web' for web UI"
