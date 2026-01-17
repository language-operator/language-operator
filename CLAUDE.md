# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

Language Operator is a Kubernetes operator for managing AI agents and language models across clusters. It provides declarative YAML configuration for AI workloads and integrated telemetry collection via OpenTelemetry.

The project consists of the Kubernetes operator (Go) and supporting infrastructure including ClickHouse for telemetry storage.

## Repository Structure

```
language-operator/
├── src/                           # Go Kubernetes operator
│   ├── api/v1alpha1/             # Custom Resource Definitions (CRDs)
│   ├── controllers/              # Kubernetes controllers
│   ├── pkg/telemetry/           # Telemetry collection and adapters
│   └── config/                   # Kubernetes manifests and RBAC
├── chart/                        # Helm chart for deployment
├── examples/                     # Example AI agent configurations
├── scripts/                      # Development and deployment scripts
└── docs/                        # Documentation
```

## Architecture

This is a **Kubernetes-native Go operator** with the following components:

### Core Applications
- **`/src/`** - Go Kubernetes operator managing LanguageCluster and Agent CRDs
- **`/chart/`** - Helm chart for production deployment

### Supporting Infrastructure
- **ClickHouse** - High-performance telemetry storage via OpenTelemetry Collector
- **Kubernetes APIs** - Native integration with cluster resources
- **Prometheus/OTel** - Metrics and tracing collection

## Development Commands

### Building & Testing (Go Operator)
```sh
cd src                  # Navigate to Go operator directory
make test              # Run all Go tests with linting and formatting
make build             # Build Go operator binary
make run               # Run operator locally (requires K8s cluster)
make fmt               # Run go fmt
make vet               # Run go vet
make generate          # Generate DeepCopy methods
make manifests         # Generate CRDs and RBAC
```

### Repository-Level Testing
```sh
make test              # Run all tests (Go operator + integration)
make test-unit         # Run fast unit tests (no K8s required)
make test-integration  # Run integration tests with fake K8s client
```

### Development Tools
```sh
make docs              # Generate CRD API reference documentation
make setup-hooks       # Install git pre-commit hooks
make k8s-status        # Check status of deployed language resources
```

### Helm Chart Development
```sh
cd chart               # Navigate to Helm chart directory
helm lint .            # Validate Helm chart syntax
helm template . --debug  # Render templates locally for validation
cd ../src && make helm-crds  # Copy generated CRDs to chart
```

## Technology Stack

### Kubernetes Operator (`/src/`)
- **Language**: Go 1.23+ (see `src/go.mod`)
- **Framework**: controller-runtime (Kubernetes operator framework)
- **CRDs**: LanguageCluster, LanguageAgent, LanguageAgentVersion, LanguageModel, LanguageTool custom resources
- **Telemetry**: OpenTelemetry and ClickHouse
- **Testing**: Standard Go testing with testify
- **Build**: Make, Go modules


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


### Definition of Done
A feature is **ONLY complete** when:
1. ✅ Works end-to-end with real ClickHouse telemetry data
2. ✅ Integrates properly with Kubernetes APIs
3. ✅ All tests passing (unit + integration)
4. ✅ Manual verification completed with real data
5. ✅ Linting and type checking passing
6. ✅ No mock data or temporary workarounds


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
- **E2E**: Manual verification with real telemetry data required
- Tests must be independent and run concurrently
- **NO COMMITS without passing tests**

### Code Standards
- **Go**: Follow standard Go conventions, use gofmt
- **Error Handling**: Proper error propagation and logging
- **Documentation**: Inline comments for complex logic

## Environment Setup

### Prerequisites
- **Go**: Version 1.21+ (see `go.mod`)
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

# 3. Verify telemetry data flow
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

### Operator Verification
```sh
# 1. Check operator is running
kubectl logs -n kube-system deployment/language-operator

# 2. Apply test resources
kubectl apply -f examples/

# 3. Verify CRDs are processed correctly
kubectl get languageclusters,languageagents -A

# 4. Check telemetry data is being collected
kubectl exec -it clickhouse-0 -- clickhouse-client -q "SELECT count() FROM otel_traces"
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
1. ✅ Run full test suite: `make test`
2. ✅ Verify with real data: Manual testing with actual ClickHouse traces
3. ✅ Check linting: `cd src && make fmt && make vet`
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