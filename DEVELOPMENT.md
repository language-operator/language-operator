# Development Environment

This document describes how to set up the Language Operator Kubernetes operator for development.

## Quick Start

```bash
# 1. Set up local Kubernetes cluster with dependencies
make dev-setup

# 2. Build and deploy operator to local cluster  
make dev-deploy

# 3. Run tests
make test

# 4. View operator logs
kubectl logs -n kube-system deployment/language-operator
```

## Architecture

The development environment focuses on the Kubernetes operator functionality:

### Core Components
- **Language Operator**: Go-based Kubernetes operator managing Language* CRDs
- **ClickHouse**: High-performance telemetry storage
- **OpenTelemetry Collector**: Telemetry collection and forwarding
- **Kind/K3s Cluster**: Local Kubernetes development cluster

### Note on Dashboard
The web dashboard has been moved to a separate repository and is deployed as a container image. For dashboard development, see the [dashboard repository](https://github.com/language-operator/dashboard).

## Development Workflow

### 1. Set up Local Environment
```bash
# Set up local Kubernetes cluster with all dependencies
make dev-setup
```

This will:
- Create a local Kind cluster
- Deploy ClickHouse for telemetry storage
- Deploy OpenTelemetry Collector
- Install necessary CRDs and RBAC

### 2. Develop
- **Operator changes**: Build and deploy to your local cluster using `make dev-deploy`
- **CRD changes**: Run `make manifests` to regenerate Kubernetes manifests
- **Test changes**: Use `make test` for unit tests, `make test-integration` for integration tests

### 3. Test Kubernetes Resources
```bash
# Build and deploy operator
make dev-deploy

# Apply test resources to your cluster
kubectl apply -f examples/

# Watch operator logs
kubectl logs -f -n kube-system deployment/language-operator
```

### 4. Access Services
- **Operator Logs**: `kubectl logs -n kube-system deployment/language-operator`
- **ClickHouse**: Port-forward to access telemetry database
- **Kubernetes API**: Your local cluster endpoint

## Troubleshooting

### Operator Won't Start
```bash
# Check operator logs
kubectl logs -n kube-system deployment/language-operator

# Check if CRDs are installed
kubectl get crd | grep langop.io

# Verify RBAC permissions
kubectl describe clusterrole language-operator-manager-role
```

### Build Issues
```bash
# Clean and rebuild
make clean
make build

# Run tests to verify
make test
```

### ClickHouse/Telemetry Issues
```bash
# Check ClickHouse status
kubectl get pods -l app=clickhouse

# Check OpenTelemetry Collector
kubectl get pods -l app=otel-collector

# View collector logs
kubectl logs -l app=otel-collector
```

### Clean Reset
```bash
# Remove local cluster and start fresh
kind delete cluster --name language-operator
make dev-setup
```

## Configuration

### Environment Variables
- `KUBECONFIG`: Path to your cluster configuration
- `TELEMETRY_ADAPTER_TYPE`: Type of telemetry adapter (clickhouse, signoz, noop)
- `TELEMETRY_ADAPTER_ENDPOINT`: ClickHouse or SigNoz endpoint URL
- `TELEMETRY_ADAPTER_API_KEY`: API key for telemetry service (if required)

## Advanced Usage

### Using Different Cluster
To use a different Kubernetes cluster:

1. Set your kubeconfig to point to the cluster:
```bash
export KUBECONFIG=/path/to/your/kubeconfig
```

2. Deploy the operator:
```bash
make dev-deploy
```

### Custom Configuration
Modify the Helm values in `chart/values.yaml` to customize the operator deployment.

## Next Steps

Once your development environment is running:

1. Create your first LanguageCluster resource
2. Deploy some example agents from the `examples/` directory
3. Monitor the operator logs to see it working
4. Access telemetry data through ClickHouse queries

Happy developing! 🚀