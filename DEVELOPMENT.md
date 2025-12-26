# Development Environment

This document describes how to set up and use the Language Operator development environment.

## Quick Start

```bash
# 1. Set up Docker network bridge to K3s (one-time setup)
make dev-k3s-bridge

# 2. Start dashboard and database
make dev-up

# 3. View logs from all services
make dev-logs

# 4. Check status
make dev-status

# 5. Stop everything
make dev-down
```

## Architecture

The development environment separates concerns for better reliability:

### Docker Compose Services
- **postgres-dev**: PostgreSQL database for the dashboard  
- **dashboard-dev**: Next.js dashboard with hot reload at `http://localhost:3000`
- **prisma-studio**: Database management UI at `http://localhost:5555`

### K3s Cluster Integration
- **Network Bridge**: Docker containers can resolve Kubernetes service DNS
- **Direct Communication**: Dashboard talks directly to `my-service.default.svc.cluster.local`
- **Existing K3s Cluster**: Uses your existing K3s cluster at `dl4:6443`

This approach provides:
- **Real service discovery**: No port-forwarding or proxies needed
- **Production-like networking**: Same DNS resolution as in-cluster workloads  
- **Minimal complexity**: Leverages your existing K3s infrastructure

## Service Details

### Kind Cluster
- **URL**: `https://kind-cluster:6443` (internal) or `https://localhost:6443` (external)
- **Config**: Automatically sets up CRDs and RBAC
- **Ports**: 6443 (API), 30080 (HTTP ingress), 30443 (HTTPS ingress)

### Dashboard
- **URL**: `http://localhost:3000`
- **Hot Reload**: Changes to `components/dashboard/` are automatically reflected
- **Database**: Connects to `postgres-dev:5432`
- **Kubernetes**: Connects to `kind-cluster:6443`

### Database
- **URL**: `postgresql://dev:dev@localhost:5433/language_operator_dev`
- **Admin**: Use Prisma Studio at `http://localhost:5555`

## Development Workflow

### 1. Start Development Environment
```bash
make dev-up
```

This will:
- Start PostgreSQL database
- Start Kind Kubernetes cluster
- Install Language Operator CRDs and RBAC
- Start the Language Operator controller
- Start the dashboard with hot reload
- Start Prisma Studio

### 2. Develop
- **Dashboard changes**: Edit files in `components/dashboard/` - changes are hot-reloaded
- **Operator changes**: Edit files in `src/` - rebuild with `docker compose build language-operator && docker compose up -d language-operator`
- **Database changes**: Use Prisma Studio at `http://localhost:5555`

### 3. Test Kubernetes Resources
```bash
# Check if Kind cluster is ready
make dev-status

# Apply test resources (from another terminal)
kubectl apply -f examples/ --kubeconfig components/dashboard/.kube/config

# Watch operator logs
docker compose logs -f language-operator
```

### 4. Access Services
- **Dashboard**: `http://localhost:3000`
- **Prisma Studio**: `http://localhost:5555`
- **Kubernetes API**: `https://localhost:6443` (insecure-skip-tls-verify)

## Troubleshooting

### Services Won't Start
```bash
# Check service status
make dev-status

# View logs for specific service
docker compose logs kind-cluster
docker compose logs language-operator
docker compose logs dashboard-dev
```

### Kind Cluster Issues
```bash
# Restart just the cluster
docker compose restart kind-cluster

# Check cluster status
docker compose exec kind-cluster kubectl get nodes
```

### Dashboard Issues
```bash
# Rebuild dashboard
docker compose build dashboard-dev
docker compose up -d dashboard-dev

# Check database connection
docker compose exec postgres-dev psql -U dev -d language_operator_dev -c "SELECT NOW();"
```

### Clean Reset
```bash
# Remove all containers, volumes, and data
make dev-clean
```

## Configuration

### Environment Variables
- `DATABASE_URL`: PostgreSQL connection string  
- `KUBERNETES_SERVER_URL`: Kind cluster API endpoint
- `NEXTAUTH_SECRET`: Dashboard authentication secret

### Volumes
- `postgres_dev_data`: Database storage
- `kind_data`: Kind cluster storage
- `dashboard_kube`: Dashboard kubeconfig
- `operator_kube`: Operator kubeconfig

## Advanced Usage

### Using External Cluster
To use a different Kubernetes cluster instead of Kind:

1. Set environment variables before starting:
```bash
export KUBERNETES_SERVER_URL=https://your-cluster.example.com
export KUBECONFIG=/path/to/your/kubeconfig
```

2. Start the development environment:
```bash
make dev-up
```

The dashboard will automatically connect to your specified cluster.

### Custom Configuration
Create a `docker-compose.override.yml` file to customize the setup without modifying the main configuration.

## Next Steps

Once your development environment is running:

1. Visit the dashboard at `http://localhost:3000`
2. Create your first LanguageCluster resource
3. Deploy some example agents from the `examples/` directory
4. Monitor the operator logs to see it working

Happy developing! 🚀