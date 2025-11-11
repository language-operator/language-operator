# OpenTelemetry Integration Guide

This guide covers how to enable, configure, and use OpenTelemetry distributed tracing with the language-operator.

## Table of Contents

- [Overview](#overview)
- [Prerequisites](#prerequisites)
- [Quick Start](#quick-start)
- [Configuration Reference](#configuration-reference)
- [Backend Options](#backend-options)
- [Example Trace Queries](#example-trace-queries)
- [Troubleshooting](#troubleshooting)
- [Performance Considerations](#performance-considerations)

## Overview

The language-operator includes comprehensive OpenTelemetry instrumentation for distributed tracing. This provides visibility into:

- **Agent Lifecycle**: Track agent creation, synthesis, deployment, and updates
- **Code Synthesis**: Observe LLM interactions, token usage, and costs
- **Self-Healing**: Monitor failure detection and automatic recovery
- **Performance**: Identify slow operations and bottlenecks
- **Cost Analysis**: Track synthesis costs per agent and namespace

Key features:
- Zero overhead when disabled (opt-in via configuration)
- Automatic trace context propagation
- Cost tracking with per-operation USD amounts
- Rich semantic attributes for filtering and analysis
- Support for all major observability backends

## Prerequisites

### 1. OpenTelemetry Collector

You need an OpenTelemetry Collector or compatible backend that accepts OTLP (OpenTelemetry Protocol) over gRPC.

Options:
- **OpenTelemetry Collector** (recommended): Vendor-agnostic collection and forwarding
- **Jaeger**: Direct OTLP ingestion (v1.35+)
- **Grafana Tempo**: Direct OTLP ingestion
- **Elastic APM**: Via OTLP receiver
- **Datadog Agent**: Via OTLP receiver

### 2. Observability Backend

Choose a backend for trace visualization and analysis:

| Backend | Setup Difficulty | Cost | Best For |
|---------|-----------------|------|----------|
| Jaeger | Easy | Free | Development, small deployments |
| Grafana Tempo | Medium | Free | Production, Grafana users |
| Elastic APM | Medium | Free/Paid | Elastic stack users |
| Datadog | Easy | Paid | Enterprise, full observability |
| Honeycomb | Easy | Paid | Advanced analysis, high cardinality |

## Quick Start

This quick start deploys Jaeger and configures the operator to send traces.

### Step 1: Deploy Jaeger

Deploy Jaeger all-in-one for development/testing:

```bash
kubectl create namespace observability

kubectl apply -f - <<EOF
apiVersion: apps/v1
kind: Deployment
metadata:
  name: jaeger
  namespace: observability
spec:
  replicas: 1
  selector:
    matchLabels:
      app: jaeger
  template:
    metadata:
      labels:
        app: jaeger
    spec:
      containers:
      - name: jaeger
        image: jaegertracing/all-in-one:1.51
        ports:
        - containerPort: 4317
          name: otlp-grpc
        - containerPort: 16686
          name: ui
        env:
        - name: COLLECTOR_OTLP_ENABLED
          value: "true"
---
apiVersion: v1
kind: Service
metadata:
  name: jaeger
  namespace: observability
spec:
  selector:
    app: jaeger
  ports:
  - name: otlp-grpc
    port: 4317
    targetPort: 4317
  - name: ui
    port: 16686
    targetPort: 16686
EOF
```

### Step 2: Deploy OpenTelemetry Collector (Optional but Recommended)

For production, use a collector to decouple the operator from the backend:

```bash
kubectl apply -f examples/otel-collector.yaml
```

This deploys a collector that:
- Receives traces from the operator via OTLP/gRPC
- Batches spans for efficiency
- Forwards to Jaeger (or other backends)
- Provides memory limits to prevent OOM

### Step 3: Configure Language Operator

Update your Helm values to enable OpenTelemetry:

```yaml
# values.yaml
opentelemetry:
  enabled: true
  endpoint: "otel-collector.kube-system.svc.cluster.local:4317"

  sampling:
    rate: 1.0  # 100% sampling for development

  resourceAttributes:
    environment: "production"
    cluster: "main"
```

Alternatively, if using Jaeger directly without a collector:

```yaml
opentelemetry:
  enabled: true
  endpoint: "jaeger.observability.svc.cluster.local:4317"
  sampling:
    rate: 1.0
```

### Step 4: Upgrade the Operator

Apply the configuration:

```bash
helm upgrade language-operator ./chart \
  --namespace kube-system \
  -f values.yaml
```

### Step 5: Access Traces

Port-forward to the Jaeger UI:

```bash
kubectl port-forward -n observability svc/jaeger 16686:16686
```

Open [http://localhost:16686](http://localhost:16686) in your browser.

Search for:
- **Service**: `language-operator`
- **Operation**: `agent.reconcile`

## Configuration Reference

### Helm Values

#### `opentelemetry.enabled`
**Type**: `bool`
**Default**: `false`
**Description**: Enable OpenTelemetry tracing. When `false`, tracing has zero overhead.

```yaml
opentelemetry:
  enabled: true
```

#### `opentelemetry.endpoint`
**Type**: `string`
**Default**: `""`
**Description**: OTLP gRPC endpoint for trace export. Must be in the format `host:port`.

Examples:
```yaml
# In-cluster collector (same namespace)
endpoint: "otel-collector:4317"

# In-cluster collector (different namespace)
endpoint: "otel-collector.observability.svc.cluster.local:4317"

# External collector
endpoint: "otel-collector.example.com:4317"
```

#### `opentelemetry.sampling.rate`
**Type**: `float`
**Default**: `1.0`
**Range**: `0.0` to `1.0`
**Description**: Trace sampling rate.

Recommendations:
- **Development**: `1.0` (100% - capture everything)
- **Production (low volume)**: `1.0` (100%)
- **Production (medium volume)**: `0.1` (10%)
- **Production (high volume)**: `0.01` (1%)

```yaml
opentelemetry:
  sampling:
    rate: 0.1  # Sample 10% of traces
```

#### `opentelemetry.resourceAttributes`
**Type**: `object`
**Description**: Resource attributes attached to all traces. Useful for filtering and grouping.

```yaml
opentelemetry:
  resourceAttributes:
    environment: "production"  # Environment name
    cluster: "us-east-1"       # Cluster identifier
    custom:
      team: "platform"
      region: "us-east-1"
```

#### `agentTelemetry.endpoint`
**Type**: `string`
**Default**: `""` (inherits from `opentelemetry.endpoint`)
**Description**: Separate OTLP endpoint for agent traces. Useful for routing agent traces to a different backend.

```yaml
agentTelemetry:
  endpoint: "otel-collector-agents.observability.svc.cluster.local:4317"
```

#### `agentTelemetry.samplingRate`
**Type**: `float`
**Default**: `null` (inherits from `opentelemetry.sampling.rate`)
**Description**: Separate sampling rate for agent traces.

```yaml
agentTelemetry:
  samplingRate: 0.1  # Sample 10% of agent traces, but 100% of operator traces
```

### Environment Variables

The operator also respects standard OpenTelemetry environment variables:

#### `OTEL_EXPORTER_OTLP_ENDPOINT`
Set by Helm when `opentelemetry.enabled=true`. Can be overridden manually.

```yaml
env:
  - name: OTEL_EXPORTER_OTLP_ENDPOINT
    value: "otel-collector:4317"
```

#### `OTEL_RESOURCE_ATTRIBUTES`
Additional resource attributes (comma-separated key=value pairs).

```yaml
env:
  - name: OTEL_RESOURCE_ATTRIBUTES
    value: "k8s.cluster.name=production,deployment.environment=prod"
```

#### `POD_NAMESPACE`
Automatically set by Helm. Used to add `k8s.namespace.name` to traces.

## Backend Options

### Jaeger

**Best for**: Development, testing, small deployments

**Pros**:
- Easy to deploy
- No external dependencies
- Free and open-source
- Good UI for trace visualization

**Cons**:
- Limited scalability
- Basic query capabilities

**Deployment**:
```bash
kubectl apply -f - <<EOF
apiVersion: apps/v1
kind: Deployment
metadata:
  name: jaeger
  namespace: observability
spec:
  replicas: 1
  selector:
    matchLabels:
      app: jaeger
  template:
    metadata:
      labels:
        app: jaeger
    spec:
      containers:
      - name: jaeger
        image: jaegertracing/all-in-one:1.51
        env:
        - name: COLLECTOR_OTLP_ENABLED
          value: "true"
        ports:
        - containerPort: 4317
        - containerPort: 16686
---
apiVersion: v1
kind: Service
metadata:
  name: jaeger
  namespace: observability
spec:
  selector:
    app: jaeger
  ports:
  - name: otlp-grpc
    port: 4317
  - name: ui
    port: 16686
EOF
```

**Configuration**:
```yaml
opentelemetry:
  enabled: true
  endpoint: "jaeger.observability.svc.cluster.local:4317"
```

### Grafana Tempo

**Best for**: Production deployments, Grafana users

**Pros**:
- Highly scalable
- Integrates with Grafana dashboards
- Cost-effective storage
- Advanced query capabilities

**Cons**:
- More complex setup
- Requires object storage (S3, GCS, etc.)

**Deployment**:
```bash
# Install via Helm
helm repo add grafana https://grafana.github.io/helm-charts
helm install tempo grafana/tempo \
  --namespace observability \
  --create-namespace
```

**Configuration**:
```yaml
opentelemetry:
  enabled: true
  endpoint: "tempo.observability.svc.cluster.local:4317"
```

### Elastic APM

**Best for**: Elastic stack users

**Pros**:
- Integrates with Elastic ecosystem
- Powerful querying via Kibana
- Combines logs, metrics, traces

**Cons**:
- Resource intensive
- Complex setup

**Configuration**:
```yaml
opentelemetry:
  enabled: true
  endpoint: "apm-server.observability.svc.cluster.local:8200"
```

### Datadog

**Best for**: Enterprise deployments with existing Datadog usage

**Pros**:
- Managed service (no infrastructure)
- Excellent UI and analysis
- Combines all observability signals

**Cons**:
- Paid service
- Vendor lock-in

**Configuration**:
```yaml
opentelemetry:
  enabled: true
  endpoint: "datadog-agent.datadog.svc.cluster.local:4317"
```

## Example Trace Queries

This section provides example queries for common observability scenarios. Query syntax varies by backend.

### Jaeger Query Syntax

Find traces using the Jaeger UI search:

#### 1. Find All Agent Reconciliations

**Service**: `language-operator`
**Operation**: `agent.reconcile`
**Tags**: (none)

#### 2. Find Synthesis for a Specific Agent

**Service**: `language-operator`
**Operation**: `agent.synthesize`
**Tags**: `agent.name=email-responder`

#### 3. Find Expensive Synthesis Operations (>$0.10)

**Service**: `language-operator`
**Operation**: `synthesis.agent.generate`
**Tags**: `synthesis.cost_usd>0.10`

#### 4. Find Self-Healing Attempts

**Service**: `language-operator`
**Operation**: `agent.self_healing.synthesize`

#### 5. Find Failed Operations

**Service**: `language-operator`
**Operation**: (any)
**Tags**: `error=true`

#### 6. Find Slow Operations (>10s)

**Service**: `language-operator`
**Min Duration**: `10s`

### Grafana Tempo Query Syntax (TraceQL)

Use TraceQL in Grafana Explore:

#### 1. Complete Agent Lifecycle Trace

```traceql
{ .agent.name = "email-responder" }
```

#### 2. Expensive Synthesis Operations

```traceql
{
  span.name = "synthesis.agent.generate" &&
  .synthesis.cost_usd > 0.10
}
```

#### 3. Self-Healing Activity

```traceql
{
  span.name =~ "agent.self_healing.*"
}
```

#### 4. Failed Operations with Error Details

```traceql
{
  status = error
}
```

#### 5. Synthesis in Specific Namespace

```traceql
{
  span.name = "agent.synthesize" &&
  .synthesis.namespace = "production"
}
```

#### 6. High Token Usage (>5000 tokens)

```traceql
{
  .synthesis.input_tokens > 5000 ||
  .synthesis.output_tokens > 5000
}
```

### Example Analysis Scenarios

#### Scenario: Debug Failed Agent

**Goal**: Find why agent `email-responder` keeps failing

**Steps**:
1. Search for agent: `{ .agent.name = "email-responder" }`
2. Filter for errors: Add `&& status = error`
3. Examine span attributes for error messages
4. Check `agent.self_healing.detect` spans for failure patterns
5. Review `self_healing.error_context` attribute

#### Scenario: Cost Optimization

**Goal**: Identify agents with high synthesis costs

**Steps**:
1. Query expensive operations: `{ .synthesis.cost_usd > 0.05 }`
2. Group by agent name: Check `synthesis.agent_name` attribute
3. Look for patterns:
   - High token counts? Simplify instructions
   - Many retries? Fix validation errors
   - Large tool lists? Reduce unnecessary tools

#### Scenario: Performance Analysis

**Goal**: Find slow reconciliation loops

**Steps**:
1. Search for reconciliations: `{ span.name = "agent.reconcile" }`
2. Filter by duration: Add `&& duration > 30s`
3. Drill into child spans to find bottleneck:
   - Slow synthesis? Check LLM response times
   - Slow deployment? Check Kubernetes API calls

## Troubleshooting

### No Traces Appearing

**Symptom**: Traces don't appear in the backend

**Checklist**:
1. ✓ Is OpenTelemetry enabled?
   ```bash
   kubectl get deployment language-operator -n kube-system -o yaml | grep OTEL_EXPORTER_OTLP_ENDPOINT
   ```

2. ✓ Is the endpoint reachable?
   ```bash
   kubectl run curl --image=curlimages/curl -it --rm --restart=Never -- \
     curl -v telnet://otel-collector.kube-system:4317
   ```

3. ✓ Are traces being generated?
   ```bash
   # Create a test agent to trigger reconciliation
   kubectl apply -f examples/email-responder.yaml
   ```

4. ✓ Check operator logs for errors:
   ```bash
   kubectl logs -n kube-system deployment/language-operator | grep -i otel
   ```

5. ✓ Check collector logs (if using collector):
   ```bash
   kubectl logs -n kube-system deployment/otel-collector
   ```

### Collector Unreachable

**Symptom**: Operator logs show connection errors

**Error Example**:
```
failed to export spans: context deadline exceeded
```

**Solutions**:

1. **Check endpoint configuration**:
   ```bash
   kubectl get deployment language-operator -n kube-system -o yaml | grep OTEL_EXPORTER
   ```

2. **Verify collector is running**:
   ```bash
   kubectl get pods -n kube-system -l app=otel-collector
   ```

3. **Test connectivity**:
   ```bash
   kubectl run test-curl --image=curlimages/curl -it --rm -- \
     curl -v telnet://otel-collector.kube-system:4317
   ```

4. **Check NetworkPolicies**:
   ```bash
   kubectl get networkpolicies -n kube-system
   ```

### High Overhead

**Symptom**: Operator performance degraded after enabling tracing

**Solutions**:

1. **Reduce sampling rate**:
   ```yaml
   opentelemetry:
     sampling:
       rate: 0.1  # Sample only 10%
   ```

2. **Use batch processing** (collector config):
   ```yaml
   processors:
     batch:
       timeout: 10s
       send_batch_size: 1024
   ```

3. **Monitor collector resource usage**:
   ```bash
   kubectl top pod -n kube-system -l app=otel-collector
   ```

4. **Increase collector resources**:
   ```yaml
   resources:
     limits:
       memory: "1Gi"
       cpu: "1000m"
   ```

### Missing Attributes

**Symptom**: Expected attributes don't appear on spans

**Common Causes**:

1. **Attribute not set for this span type**: Check [semantic conventions](semantic-conventions.md)
2. **Operation skipped**: E.g., no synthesis if code already exists
3. **Backend limits**: Some backends drop attributes with high cardinality

**Debug**:
```bash
# Check operator logs for span creation
kubectl logs -n kube-system deployment/language-operator | grep "span"
```

### Trace Context Lost

**Symptom**: Operations appear as separate traces instead of connected

**Cause**: Trace context not propagated between operations

**This should not happen** - the operator automatically propagates context. If you see this:

1. Report a bug with trace IDs
2. Check for operator restarts during operation
3. Verify all operations are in the same pod

## Performance Considerations

### Overhead

OpenTelemetry tracing has minimal overhead when properly configured:

- **Disabled** (`enabled: false`): Zero overhead - not initialized
- **Enabled, no collector**: ~1-2ms per operation (span creation)
- **Enabled, with collector**: ~1-2ms per operation (async export)

### Sampling Strategies

Choose sampling based on volume and budget:

| Environment | Request Volume | Recommended Rate | Rationale |
|-------------|---------------|------------------|-----------|
| Development | Low | 1.0 (100%) | Capture everything |
| Staging | Medium | 1.0 (100%) | Full visibility for testing |
| Production (small) | <100 req/min | 1.0 (100%) | Low volume, capture all |
| Production (medium) | 100-1000 req/min | 0.1 (10%) | Balance visibility and cost |
| Production (large) | >1000 req/min | 0.01 (1%) | Statistical sampling |

### Resource Recommendations

#### Operator

No changes needed - tracing overhead is negligible.

#### OpenTelemetry Collector

| Deployment Size | CPU Request | Memory Request | CPU Limit | Memory Limit |
|----------------|-------------|----------------|-----------|--------------|
| Small (<100 spans/s) | 100m | 128Mi | 500m | 512Mi |
| Medium (100-1000 spans/s) | 200m | 256Mi | 1000m | 1Gi |
| Large (>1000 spans/s) | 500m | 512Mi | 2000m | 2Gi |

### Storage Considerations

Trace storage requirements depend on:
- Request volume
- Sampling rate
- Retention period
- Span size (attributes, events)

**Example calculation**:
- 1000 agent reconciliations/day
- Average 5 spans per trace
- Average 2KB per span
- 100% sampling
- 30-day retention

```
1000 reconciliations/day × 5 spans × 2KB × 30 days = 300MB
```

For production deployments, plan for:
- **Jaeger**: 1-10GB storage (limited retention)
- **Tempo**: 10-100GB object storage (long retention)
- **Elastic**: 10-100GB disk (depends on retention)

## Next Steps

- Review [Semantic Conventions](semantic-conventions.md) for detailed attribute reference
- Deploy collector: See `examples/otel-collector.yaml`
- Explore example queries in your backend
- Set up dashboards for cost and performance monitoring
- Configure alerts for failed synthesis operations

## Additional Resources

- [OpenTelemetry Documentation](https://opentelemetry.io/docs/)
- [OTLP Specification](https://opentelemetry.io/docs/specs/otlp/)
- [Jaeger Documentation](https://www.jaegertracing.io/docs/)
- [Grafana Tempo Documentation](https://grafana.com/docs/tempo/)
- [TraceQL Query Language](https://grafana.com/docs/tempo/latest/traceql/)
