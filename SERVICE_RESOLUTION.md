# Service Resolution Strategy

This document explains how the dashboard resolves Kubernetes service URLs in different deployment environments.

## The Problem

When the dashboard runs in Docker Compose, it **cannot** directly resolve Kubernetes service hostnames like:
```
http://qwen3-coder-30b.org-2h6mnkxh.svc.cluster.local:8000
```

This fails with "hostname not found" because the Docker container doesn't have access to Kubernetes DNS.

## The Solution

We implemented environment-aware service resolution that automatically adapts based on where the dashboard is running:

### In Kubernetes (Production)
```typescript
// Direct service DNS - works natively
http://qwen3-coder-30b.org-2h6mnkxh.svc.cluster.local:8000/v1/chat/completions
```

### In Docker Compose (Development) 
```typescript  
// kubectl proxy route - works via proxy
http://kubectl-proxy:8001/api/v1/namespaces/org-2h6mnkxh/services/qwen3-coder-30b:8000/proxy/v1/chat/completions
```

## Implementation

### 1. Service Resolver (`src/lib/service-resolver.ts`)

The `ServiceResolver` class automatically detects the environment and returns the correct URL:

```typescript
import { serviceResolver } from '@/lib/service-resolver'

// This automatically resolves to the correct URL format
const url = serviceResolver.resolveAgentChatUrl('my-agent', 'my-namespace', 80)
```

### 2. Environment Detection

The resolver detects the environment using these criteria:

**Kubernetes Environment:**
- `KUBERNETES_SERVICE_HOST` is set (automatic in K8s pods)
- `KUBECONFIG` contains `/var/run/secrets/kubernetes.io`
- `NODE_ENV=production`

**Docker Compose Environment:**
- `KUBERNETES_SERVER_URL` contains `kubectl-proxy`
- Not running in Kubernetes

### 3. Updated API Routes

Modified these endpoints to use the resolver:

- **Agent Chat**: `src/app/api/clusters/[name]/agents/[agentName]/chat/route.ts`
- **Persona Generation**: `src/app/api/personas/generate/route.ts`

### 4. Testing

Test the resolution with:
```bash
curl "http://localhost:3000/api/test/service-resolution?service=qwen3-coder-30b&namespace=org-2h6mnkxh"
```

## Usage Examples

### Basic Service Resolution
```typescript
const url = serviceResolver.resolveServiceUrl({
  serviceName: 'my-service',
  namespace: 'my-namespace', 
  port: 8080,
  path: '/api/health'
})
```

### Agent Chat URLs
```typescript
const chatUrl = serviceResolver.resolveAgentChatUrl('story-writer', 'org-123', 80)
```

### Model URLs (LiteLLM)
```typescript
const modelUrl = serviceResolver.resolveModelUrl('qwen3-coder-30b', 'org-123', 8000)
```

### Environment Info
```typescript
const info = serviceResolver.getEnvironmentInfo()
console.log(info.environment) // 'kubernetes' | 'docker-compose' | 'unknown'
```

## Environment Variables

The resolver uses these environment variables:

| Variable | Docker Compose | Kubernetes |
|----------|---------------|------------|
| `KUBERNETES_SERVICE_HOST` | unset | `10.43.0.1` (auto-injected) |
| `KUBERNETES_SERVER_URL` | `http://kubectl-proxy:8001` | `https://kubernetes.default.svc` |
| `KUBECONFIG` | `/root/.kube/config` | `/var/run/secrets/...` |
| `NODE_ENV` | `development` | `production` |

## Benefits

1. **Transparent**: Same code works in both environments
2. **Automatic**: No manual configuration required  
3. **Debuggable**: Environment info available for troubleshooting
4. **Maintainable**: Single place to update URL resolution logic

## Migration Guide

### Before (Broken in Docker Compose)
```typescript
const url = `http://${service}.${namespace}.svc.cluster.local:${port}/path`
```

### After (Works Everywhere)
```typescript
import { serviceResolver } from '@/lib/service-resolver'

const url = serviceResolver.resolveServiceUrl({
  serviceName: service,
  namespace,
  port,
  path: '/path'
})
```

This approach ensures the dashboard can communicate with Kubernetes services regardless of whether it's running in Docker Compose for development or deployed in Kubernetes for production.