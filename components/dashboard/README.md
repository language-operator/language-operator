# LanguageCluster Dashboard

Web-based dashboard for managing and monitoring LanguageCluster agents.

## Overview

The dashboard provides a minimal-code web UI using:
- **HTMX** (14KB) - Declarative AJAX interactions
- **Alpine.js** (15KB) - Reactive UI components
- **Tailwind CSS** - Utility-first styling
- **Ruby + Rack** - Lightweight backend server

Total frontend code: ~500 lines HTML, zero build step required.

## Features

### Current (MVP)
- Agent list view with auto-refresh
- Agent status and metrics display
- Create agent form (natural language input)
- Cluster-level metrics dashboard
- Responsive design

### Planned
- Real-time log streaming
- Agent execution controls (run/stop)
- Synthesis progress monitoring
- Cost analytics and charts
- Agent detail views

## Architecture

```
┌─────────────────────────────────────┐
│  Frontend (HTMX + Alpine.js)        │
│  - Single HTML file                 │
│  - Auto-refreshing components       │
│  - Modal dialogs                    │
└─────────────────────────────────────┘
                ↓
┌─────────────────────────────────────┐
│  Backend (Ruby + Rack)              │
│  - JSON API endpoints               │
│  - HTML fragment endpoints (HTMX)   │
│  - Kubernetes client integration    │
└─────────────────────────────────────┘
                ↓
┌─────────────────────────────────────┐
│  Kubernetes API                     │
│  - LanguageAgent resources          │
│  - Pod logs and metrics             │
└─────────────────────────────────────┘
```

## Building

```bash
# Build the dashboard image
make build

# Build with base image dependency
make build-all

# Scan for vulnerabilities
make scan
```

## Running Locally

```bash
# Run with Docker
make run

# Access at http://localhost:8080
```

Environment variables:
- `CLUSTER_NAME` - Name of the LanguageCluster (default: "default")
- `NAMESPACE` - Kubernetes namespace (default: "default")
- `PORT` - HTTP port (default: 8080)

## Development

```bash
# Install dependencies
bundle install

# Run linter
make lint

# Auto-fix linting issues
bundle exec rubocop -A

# Run all tests
make test
```

## API Endpoints

### JSON API
- `GET /api/cluster/info` - Cluster information
- `GET /api/cluster/metrics` - Cluster metrics (JSON)
- `GET /api/agents` - List all agents (JSON)
- `GET /api/agents/:name` - Get agent details
- `POST /api/agents` - Create new agent
- `POST /api/agents/:name/run` - Trigger agent execution
- `GET /api/agents/:name/logs` - Get agent logs

### HTML Fragment API (for HTMX)
- `GET /components/agents` - Agent list table HTML
- `GET /components/agent/:name` - Agent detail HTML
- `GET /components/metrics` - Metrics dashboard HTML
- `GET /components/synthesis` - Synthesis monitor HTML

## Deployment

The dashboard is automatically deployed by the LanguageCluster operator when enabled:

```yaml
apiVersion: langop.io/v1alpha1
kind: LanguageCluster
metadata:
  name: my-cluster
spec:
  dashboard:
    enabled: true
    port: 8080
```

Access via:
```bash
# Port-forward to local machine
kubectl port-forward -n default svc/my-cluster-dashboard 8080:80

# Open in browser
open http://localhost:8080
```

## File Structure

```
components/dashboard/
├── Dockerfile              # Container image definition
├── Gemfile                 # Ruby dependencies
├── Makefile                # Build and test targets
├── README.md               # This file
├── bin/
│   └── langop-dashboard    # Entrypoint script
├── lib/
│   └── dashboard_server.rb # Web server implementation
└── public/
    └── index.html          # Frontend UI (HTMX + Alpine.js)
```

## Technology Choices

### Why HTMX?
- **Zero JavaScript required** for most interactions
- **Server-driven architecture** - return HTML fragments, not JSON
- **Progressive enhancement** - works without JS
- **14KB total size**

### Why Alpine.js?
- **Reactive state management** in HTML attributes
- **Perfect complement to HTMX** for client-side interactions
- **No build step required**
- **15KB total size**

### Why Tailwind CSS (CDN)?
- **Zero CSS to write** - use utility classes
- **Rapid prototyping**
- **Consistent design**
- **Responsive out-of-box**

## Design Philosophy

**Minimal Code, Maximum Value**
- Single HTML file (~500 lines)
- Simple Ruby server (~400 lines)
- No build step, no transpilation
- Works with existing infrastructure
- Fast development iteration

This approach delivers 80% of the functionality with 2% of the code compared to React/Vue alternatives.

## Future Enhancements

- WebSocket support for real-time updates
- Authentication and authorization
- Multi-cluster support
- Agent template gallery
- Workflow visualization
- Export/import agents

## License

Same as language-operator project.
