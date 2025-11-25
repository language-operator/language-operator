# Telemetry Adapter Configuration

The Language Operator supports telemetry adapters for querying historical traces and metrics from observability backends. This enables the learning system to analyze agent execution patterns and optimize performance through automatic code synthesis.

## Overview

Telemetry adapters allow the operator to:
- Query historical execution traces from telemetry backends
- Analyze agent task execution patterns
- Trigger learning-based re-synthesis based on observed behavior
- Monitor adapter connectivity and health

## Supported Backends

| Backend | Type | Support Level | Notes |
|---------|------|---------------|-------|
| SigNoz | `signoz` | ✅ Full | ClickHouse queries, PromQL metrics, 86% test coverage |
| Jaeger | `jaeger` | 🚧 Planned | GRPC query API support |
| Tempo | `tempo` | 🚧 Planned | HTTP query API support |
| No-Op | `noop` | ✅ Full | Disables telemetry queries (default) |

## Configuration

### Basic Configuration

Add the telemetry adapter configuration to your Helm values:

```yaml
# values.yaml
telemetry:
  adapter:
    enabled: true
    type: "signoz"
    endpoint: "https://signoz.example.com"
    auth:
      apiKey: "your-api-key"
```

### Secure Configuration (Recommended)

For production deployments, use Kubernetes Secrets for sensitive credentials:

```yaml
# Create secret
apiVersion: v1
kind: Secret
metadata:
  name: signoz-credentials
  namespace: language-operator-system
type: Opaque
stringData:
  api-key: "your-signoz-api-key"

---
# values.yaml
telemetry:
  adapter:
    enabled: true
    type: "signoz"
    endpoint: "https://signoz.example.com"
    auth:
      apiKeySecret:
        name: "signoz-credentials"
        key: "api-key"
```

### Advanced Configuration

```yaml
telemetry:
  adapter:
    enabled: true
    type: "signoz"
    endpoint: "https://signoz.example.com"
    
    # Authentication
    auth:
      apiKeySecret:
        name: "signoz-credentials"
        key: "api-key"
      # Additional headers for custom auth
      headers:
        X-Custom-Auth: "value"
        Authorization: "Bearer additional-token"
    
    # Connection settings
    timeout: "30s"
    retryAttempts: 3
    retryBackoff: "1s"
    
    # Query configuration
    query:
      maxTraces: 100
      lookbackPeriod: "24h"
      timeout: "10s"
    
    # Health monitoring
    healthCheck:
      enabled: true
      interval: "5m"
      timeout: "10s"
```

## Backend-Specific Examples

### SigNoz

SigNoz is the primary supported backend with full ClickHouse query support:

```yaml
telemetry:
  adapter:
    enabled: true
    type: "signoz"
    endpoint: "https://signoz.example.com"
    auth:
      apiKey: "SIGNOZ-API-KEY"  # Get from SigNoz settings
    timeout: "30s"
    query:
      maxTraces: 100
      lookbackPeriod: "24h"
```

**SigNoz Setup:**
1. Deploy SigNoz: `helm install signoz signoz/signoz`
2. Create API key in SigNoz UI → Settings → API Keys
3. Configure operator with endpoint and API key

### Jaeger (Planned)

```yaml
telemetry:
  adapter:
    enabled: true
    type: "jaeger"
    endpoint: "https://jaeger.example.com"
    timeout: "30s"
```

### Tempo (Planned)

```yaml
telemetry:
  adapter:
    enabled: true
    type: "tempo"
    endpoint: "https://tempo.example.com"
    auth:
      headers:
        Authorization: "Bearer grafana-token"
```

### Disable Telemetry Adapter

```yaml
telemetry:
  adapter:
    enabled: false
    # OR
    type: "noop"
```

## Environment Variables

The Helm chart configures these environment variables automatically:

| Variable | Description | Example |
|----------|-------------|---------|
| `TELEMETRY_ADAPTER_TYPE` | Adapter type | `signoz` |
| `TELEMETRY_ADAPTER_ENDPOINT` | Backend URL | `https://signoz.example.com` |
| `TELEMETRY_ADAPTER_API_KEY` | API key (from secret) | `xxx-api-key` |
| `TELEMETRY_ADAPTER_TIMEOUT` | Connection timeout | `30s` |
| `TELEMETRY_ADAPTER_RETRY_ATTEMPTS` | Retry attempts | `3` |
| `TELEMETRY_ADAPTER_RETRY_BACKOFF` | Retry backoff | `1s` |
| `TELEMETRY_ADAPTER_MAX_TRACES` | Max traces per query | `100` |
| `TELEMETRY_ADAPTER_LOOKBACK_PERIOD` | Query time range | `24h` |
| `TELEMETRY_ADAPTER_QUERY_TIMEOUT` | Query timeout | `10s` |
| `TELEMETRY_ADAPTER_HEALTH_CHECK_ENABLED` | Health checks | `true` |
| `TELEMETRY_ADAPTER_HEALTH_CHECK_INTERVAL` | Health check frequency | `5m` |
| `TELEMETRY_ADAPTER_HEALTH_CHECK_TIMEOUT` | Health check timeout | `10s` |

## Deployment Examples

### Development (Local SigNoz)

```yaml
telemetry:
  adapter:
    enabled: true
    type: "signoz"
    endpoint: "http://signoz-frontend.signoz.svc.cluster.local:3301"
    auth:
      apiKey: "dev-api-key"
    query:
      maxTraces: 50
      lookbackPeriod: "1h"
```

### Production (External SigNoz)

```bash
# Create secret
kubectl create secret generic signoz-prod-credentials \
  --from-literal=api-key="YOUR_PRODUCTION_API_KEY" \
  -n language-operator-system

# Deploy with secure configuration
helm upgrade language-operator ./chart \
  --set telemetry.adapter.enabled=true \
  --set telemetry.adapter.type=signoz \
  --set telemetry.adapter.endpoint=https://signoz-prod.example.com \
  --set telemetry.adapter.auth.apiKeySecret.name=signoz-prod-credentials \
  --set telemetry.adapter.auth.apiKeySecret.key=api-key \
  --set telemetry.adapter.timeout=60s \
  --set telemetry.adapter.query.maxTraces=200
```

### Multi-Environment (Staging)

```yaml
telemetry:
  adapter:
    enabled: true
    type: "signoz"
    endpoint: "https://signoz-staging.example.com"
    auth:
      apiKeySecret:
        name: "signoz-staging-credentials"
        key: "api-key"
    query:
      maxTraces: 75
      lookbackPeriod: "12h"
    healthCheck:
      interval: "2m"
```

## Troubleshooting

### Adapter Not Connecting

1. **Check Configuration:**
   ```bash
   kubectl logs -n language-operator-system deployment/language-operator | grep telemetry
   ```

2. **Verify Environment Variables:**
   ```bash
   kubectl get pod -n language-operator-system -l app.kubernetes.io/name=language-operator -o yaml | grep -A 20 env:
   ```

3. **Test Connectivity:**
   ```bash
   kubectl exec -n language-operator-system deployment/language-operator -- curl -v https://signoz.example.com/api/v1/version
   ```

### Common Issues

**❌ "SigNoz adapter requires TELEMETRY_ADAPTER_ENDPOINT"**
- Check Helm values: `telemetry.adapter.endpoint` is set
- Verify deployment: `kubectl describe pod -n language-operator-system`

**❌ "SigNoz adapter requires TELEMETRY_ADAPTER_API_KEY"** 
- Check secret exists: `kubectl get secret signoz-credentials -n language-operator-system`
- Verify secret key: `kubectl get secret signoz-credentials -o yaml`

**❌ "Failed to create SigNoz telemetry adapter"**
- Check endpoint accessibility from cluster
- Verify API key is valid in SigNoz UI
- Check network policies allowing outbound connections

**❌ "Telemetry adapter health check failed"**
- Verify backend is operational
- Check query timeout settings
- Review backend-specific authentication

### Health Checks

The operator performs periodic health checks when enabled:

```bash
# View health check logs
kubectl logs -n language-operator-system deployment/language-operator | grep "telemetry.*health"

# Check adapter metrics
kubectl port-forward -n language-operator-system service/language-operator 8443:8443
curl localhost:8443/metrics | grep telemetry
```

### Debug Mode

Enable debug logging for telemetry operations:

```yaml
config:
  logging:
    level: debug  # Enables detailed telemetry logging
```

## Integration with Learning System

When properly configured, the telemetry adapter enables:

1. **Pattern Detection:** Analyzes historical traces to identify deterministic patterns
2. **Learning Triggers:** Automatically triggers re-synthesis based on execution data
3. **Performance Monitoring:** Tracks adapter query performance and learning effectiveness
4. **Cost Optimization:** Reduces LLM costs by converting neural tasks to symbolic code

The learning controller uses the adapter to query traces with these criteria:
- Agent namespace and name attributes
- Task execution spans
- Time range based on learning threshold
- Success/failure patterns for error-triggered re-synthesis

## Security Considerations

1. **Use Secrets:** Always use Kubernetes Secrets for API keys in production
2. **Network Policies:** Restrict outbound connections to telemetry backends
3. **RBAC:** Limit secret access to operator service account only
4. **Audit Logging:** Enable audit logs for secret access tracking
5. **Rotation:** Regularly rotate API keys and update secrets

## Performance Tuning

### High-Volume Environments

```yaml
telemetry:
  adapter:
    query:
      maxTraces: 500          # Increase for more pattern data
      lookbackPeriod: "72h"   # Longer period for stable patterns
    timeout: "60s"            # Higher timeout for complex queries
    retryAttempts: 5          # More retries for reliability
```

### Low-Latency Requirements

```yaml
telemetry:
  adapter:
    query:
      maxTraces: 25           # Smaller queries
      lookbackPeriod: "6h"    # Shorter time range
      timeout: "5s"           # Lower timeout
    healthCheck:
      interval: "1m"          # Faster failure detection
```

## Monitoring

Monitor telemetry adapter performance with these metrics:
- `telemetry_adapter_queries_total` - Total queries executed
- `telemetry_adapter_query_duration_seconds` - Query latency
- `telemetry_adapter_health_check_success` - Health check success rate
- `telemetry_adapter_connection_errors_total` - Connection failures

Set up alerts for:
- Query failure rate > 5%
- Query latency > 30s  
- Health check failures
- Connection timeouts

## Next Steps

1. **Deploy Telemetry Backend:** Set up SigNoz, Jaeger, or Tempo
2. **Configure Adapter:** Add telemetry configuration to Helm values
3. **Enable Learning:** Verify learning controller integration
4. **Monitor Performance:** Set up dashboards and alerts
5. **Tune Settings:** Adjust query parameters based on usage patterns