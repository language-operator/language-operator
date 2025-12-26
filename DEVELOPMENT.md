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

### Dashboard
- **URL**: `http://localhost:3000`
- **Hot Reload**: Changes to `components/dashboard/` are automatically reflected
- **Database**: Connects to `postgres-dev:5432`
- **Kubernetes**: Connects to your K3s cluster via the network bridge

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
- Set up Docker network bridge to your K3s cluster
- Start the dashboard with hot reload
- Start Prisma Studio

### 2. Develop
- **Dashboard changes**: Edit files in `components/dashboard/` - changes are hot-reloaded
- **Operator changes**: Deploy to your K3s cluster using Helm
- **Database changes**: Use Prisma Studio at `http://localhost:5555`

### 3. Test Kubernetes Resources
```bash
# Check dashboard status
make dev-status

# Apply test resources to your K3s cluster
kubectl apply -f examples/

# Watch operator logs (in K3s cluster)
kubectl logs -f -n language-operator deployment/language-operator
```

### 4. Access Services
- **Dashboard**: `http://localhost:3000`
- **Prisma Studio**: `http://localhost:5555`
- **Kubernetes API**: Your K3s cluster endpoint

## Troubleshooting

### Services Won't Start
```bash
# Check service status
make dev-status

# View logs for specific service
docker compose logs dashboard-dev
docker compose logs postgres-dev
```

### K3s Network Bridge Issues
```bash
# Recreate the bridge
make dev-k3s-bridge

# Test DNS resolution
docker run --rm --network=langop-k3s alpine nslookup kubernetes.default.svc.cluster.local
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
- `NEXTAUTH_SECRET`: Dashboard authentication secret
- `KUBECONFIG`: Path to your K3s cluster configuration

### Volumes
- `postgres_dev_data`: Database storage

## Advanced Usage

### Using Different K3s Cluster
To use a different K3s cluster:

1. Update the K3s host in the bridge setup script:
```bash
# Edit scripts/setup-k3s-bridge.sh
K3S_HOST="your-k3s-host"
```

2. Set your kubeconfig to point to the cluster:
```bash
export KUBECONFIG=/path/to/your/kubeconfig
```

3. Set up the bridge and start development:
```bash
make dev-k3s-bridge
make dev-up
```

### Custom Configuration
Create a `docker-compose.override.yml` file to customize the setup without modifying the main configuration.

## Next Steps

Once your development environment is running:

1. Visit the dashboard at `http://localhost:3000`
2. Create your first LanguageCluster resource
3. Deploy some example agents from the `examples/` directory
4. Monitor the operator logs to see it working

Happy developing! 🚀