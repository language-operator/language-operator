# CLAUDE.md

## Project Overview

Language Operator is a Kubernetes operator for managing AI agents and language models across clusters. It provides declarative YAML configuration for AI workloads, integrated telemetry collection via OpenTelemetry, and a comprehensive dashboard for monitoring and management.

The project consists of the Kubernetes operator (Go), web dashboard (Next.js), and supporting infrastructure including ClickHouse for telemetry storage.

## Repository Structure

```
language-operator/
├── src/                           # Go Kubernetes operator
│   ├── api/v1alpha1/             # Custom Resource Definitions (CRDs)
│   ├── controllers/              # Kubernetes controllers
│   ├── pkg/telemetry/           # Telemetry collection and adapters
│   └── config/                   # Kubernetes manifests and RBAC
├── components/dashboard/         # Next.js dashboard application
│   ├── src/
│   │   ├── app/                 # Next.js 15 App Router pages
│   │   ├── components/          # React components (shadcn/ui)
│   │   ├── hooks/              # React hooks for data fetching
│   │   └── lib/                # Utilities and API clients
│   └── public/                  # Static assets
├── chart/                        # Helm chart for deployment
├── examples/                     # Example AI agent configurations
├── scripts/                      # Development and deployment scripts
└── docs/                        # Documentation
```

## Architecture

This is a **Kubernetes-native Go operator** with the following components:

### Core Applications
- **`/src/`** - Go Kubernetes operator managing LanguageCluster and Agent CRDs
- **`/components/dashboard/`** - Next.js 15 dashboard for monitoring and management
- **`/chart/`** - Helm chart for production deployment

### Supporting Infrastructure
- **ClickHouse** - High-performance telemetry storage via OpenTelemetry Collector
- **Kubernetes APIs** - Native integration with cluster resources
- **Prometheus/OTel** - Metrics and tracing collection

## Development Commands

### Local Development
```sh
# Start local Kubernetes cluster with all dependencies
make dev-setup

# Build and deploy operator to local cluster  
make dev-deploy

# Start dashboard development server
cd components/dashboard && npm run dev

# Run all tests (operator + dashboard)
make test
```

### Database Management
```sh
# ClickHouse is managed via Helm chart - no manual migrations needed
# Data is ingested automatically via OpenTelemetry Collector

# To reset telemetry data in development:
kubectl delete -n kube-system deployment/otel-collector
helm upgrade language-operator ./chart --reset-values
```

### Infrastructure
```sh
# Deploy full infrastructure to local kind cluster
make dev-kind

# Deploy to existing Kubernetes cluster
helm install language-operator ./chart
```

### Building & Testing
```sh
make build              # Build Go operator binary
make test               # Run all Go tests
make test-integration   # Run integration tests with real Kubernetes
cd components/dashboard && npm test  # Dashboard unit tests
```

## Technology Stack

### Kubernetes Operator (`/src/`)
- **Language**: Go 1.21+
- **Framework**: controller-runtime (Kubernetes operator framework)
- **CRDs**: LanguageCluster, Agent custom resources
- **Telemetry**: OpenTelemetry Go SDK with ClickHouse adapter
- **Testing**: Ginkgo/Gomega for BDD-style tests
- **Build**: Make, Go modules

### Dashboard (`/components/dashboard/`)
- **Framework**: Next.js 15 (App Router)
- **APIs**: REST APIs proxying to Kubernetes API and ClickHouse
- **Authentication**: Kubernetes RBAC (no separate auth system)
- **Database**: ClickHouse (read-only telemetry queries)
- **Validation**: Zod schemas for API validation
- **Styling**: Tailwind CSS with CSS variables
- **Components**: shadcn/ui (Radix UI primitives)
- **State Management**: React Query + React hooks
- **Charts**: Recharts for telemetry visualization

### Infrastructure
- **Telemetry Storage**: ClickHouse (high-volume trace/metric data)
- **Orchestration**: Kubernetes (native integration)
- **Telemetry Collection**: OpenTelemetry Collector
- **Packaging**: Helm charts for deployment

## Development Guidelines

### **CRITICAL: Real-First Development**
- **NO MOCK DATA** unless explicitly requested for demos
- **NO "demo mode", "fallback mode", or similar shortcuts**
- All features must work with **real ClickHouse data** before commit
- All features must integrate with **real Kubernetes APIs** before commit
- Infrastructure dependencies must be working locally before development

### **✅ COMPLIANCE VERIFICATION**
The telemetry visualization feature (GitHub #209) now fully complies with these standards:
- ✅ Uses real ClickHouse adapter (not mock data)
- ✅ Proxies through Go telemetry service (not Next.js mock APIs)
- ✅ Tested with `make test-telemetry-integration`
- ✅ Gracefully handles unavailable telemetry (returns empty data, not errors)

### Definition of Done
A feature is **ONLY complete** when:
1. ✅ Works end-to-end with real ClickHouse telemetry data
2. ✅ Integrates properly with Kubernetes APIs
3. ✅ All tests passing (unit + integration)
4. ✅ Manual verification completed with real data
5. ✅ Linting and type checking passing
6. ✅ No mock data or temporary workarounds

### Frontend Development
- All dashboard features in `/components/dashboard/src/app/[feature]/`
- Use React Server Components where possible (Next.js 15 App Router)
- API routes proxy to either Kubernetes API or ClickHouse
- Follow existing component patterns for consistency
- Use shadcn/ui components from `@/components/ui`

### Kubernetes Integration
- All operator logic in `/src/controllers/`
- Use controller-runtime patterns and best practices
- Implement proper RBAC for all resource access
- Add comprehensive unit tests for all controller logic
- Integration tests must use real Kubernetes cluster (kind)

### Telemetry Integration
- **Required**: Use existing ClickHouse adapter at `/src/pkg/telemetry/adapters/clickhouse.go`
- Query real OpenTelemetry schema tables: `otel_traces`, `otel_metrics`
- Implement proper error handling for ClickHouse connectivity
- All telemetry features must work with real trace data

### Testing Requirements
- **Go**: Ginkgo/Gomega for all controller tests
- **Integration**: Tests against real kind cluster with ClickHouse
- **Dashboard**: Jest for React component tests
- **E2E**: Manual verification with real telemetry data required
- Tests must be independent and run concurrently
- **NO COMMITS without passing tests**

### Code Standards
- **Go**: Follow standard Go conventions, use gofmt
- **TypeScript**: Strict TypeScript throughout dashboard
- **Validation**: Zod for all API input validation
- **Error Handling**: Proper error propagation and logging
- **Documentation**: Inline comments for complex logic

## Environment Setup

### Prerequisites
- **Go**: Version 1.21+ (see `go.mod`)
- **Node.js**: Version 18+ for dashboard
- **Docker**: For building container images
- **kubectl**: For Kubernetes cluster interaction
- **kind**: For local Kubernetes development cluster
- **Helm**: For chart management

### Local Development Setup
```sh
# 1. Setup local Kubernetes with ClickHouse
make dev-setup

# 2. Deploy operator to cluster
make dev-deploy

# 3. Start dashboard development
cd components/dashboard
npm install
npm run dev  # http://localhost:3000

# 4. Verify telemetry data flow
kubectl logs -n kube-system deployment/otel-collector
```

## ClickHouse Schema

The OpenTelemetry Collector writes to these ClickHouse tables:
- **`otel_traces`**: Span data with attributes, timing, status
- **`otel_metrics`**: Metric points with labels and values
- **`otel_logs`**: Structured log data with context

Always query these real tables - no mock data allowed.

## Testing with Real Data

### Generate Test Telemetry
```sh
# Deploy sample agent to generate traces
kubectl apply -f examples/simple-agent/

# Verify traces appear in ClickHouse
kubectl exec -it clickhouse-0 -- clickhouse-client \
  -q "SELECT count() FROM otel_traces WHERE SpanName LIKE '%agent%'"
```

### Dashboard Verification
```sh
# 1. Start dashboard
cd components/dashboard && npm run dev

# 2. Navigate to agent details page
open http://localhost:3000/clusters/docker/agents/[agent-name]/traces

# 3. Verify real trace data displays (no mock data)
# 4. Test all interactions work with real ClickHouse queries
```

## Common Issues

### ClickHouse Connection
- Ensure TELEMETRY_ADAPTER_ENDPOINT points to correct ClickHouse service
- Check ClickHouse logs: `kubectl logs clickhouse-0`
- Verify network connectivity from operator pod

### Missing Telemetry Data  
- Check OpenTelemetry Collector configuration
- Verify agent pods are generating traces
- Check ClickHouse table structure matches expectations

## Commit Standards

### Before Any Commit
1. ✅ Run full test suite: `make test && cd components/dashboard && npm test`
2. ✅ Verify with real data: Manual testing with actual ClickHouse traces
3. ✅ Check linting: `make lint && cd components/dashboard && npm run lint`
4. ✅ Confirm no mock data or shortcuts remain
5. ✅ All features work end-to-end

### Commit Messages
- Use conventional commits format
- **Never claim "complete" unless fully functional**
- Use "WIP:" prefix for partial implementations
- Include verification steps in commit messages

**Examples:**
- ❌ `feat: complete telemetry visualization` (when using mock data)
- ✅ `feat: add telemetry UI components (WIP: needs ClickHouse integration)`
- ✅ `feat: telemetry visualization with real ClickHouse queries`

This document enforces enterprise software standards with no shortcuts, ensuring all development uses real infrastructure and data from day one.