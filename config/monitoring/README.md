# Language Operator Learning Metrics and Observability

This directory contains monitoring configuration for the Language Operator learning system.

## Overview

The learning system provides comprehensive metrics for tracking the evolution of organic functions from neural to symbolic implementations. These metrics enable monitoring of:

- **Learning Velocity** - How quickly tasks are being optimized
- **Learning Success Rate** - Effectiveness of pattern detection and synthesis
- **Cost Optimization** - Financial impact of neural→symbolic conversions  
- **Re-synthesis Triggers** - Reasons for learning events
- **Pattern Confidence** - Quality of detected execution patterns

## Files

### `learning-dashboard.json`
Grafana dashboard providing visual analytics for learning KPIs including:
- Learning velocity over time
- Success rate gauges by agent
- Cost savings tracking  
- Re-synthesis trigger breakdown
- Pattern confidence distribution
- Error recovery metrics

**Import Instructions:**
1. Open Grafana UI
2. Navigate to **+ → Import**
3. Upload `learning-dashboard.json`
4. Configure Prometheus data source
5. Save dashboard

### `learning-alerts.yaml`
Prometheus alerting rules for learning system health monitoring:

**Alert Categories:**
- **Health Alerts** - Success rates, pattern confidence, learning stagnation
- **Performance Alerts** - Learning velocity, failure bursts, neural task persistence  
- **System Alerts** - Controller health, metrics staleness
- **Cost Alerts** - Negative savings, high activity budget warnings

**Deployment Instructions:**
```bash
kubectl apply -f learning-alerts.yaml
```

## Metrics Reference

### Core Learning Metrics

| Metric | Type | Description |
|--------|------|-------------|
| `learning_tasks_total` | Counter | Total tasks learned by trigger type |
| `learning_success_rate` | Gauge | Learning success rate per agent |
| `learning_cost_savings_usd_total` | Counter | Cost savings from symbolic conversion |
| `resynthesis_trigger_reasons_total` | Counter | Re-synthesis events by reason |
| `pattern_confidence_distribution` | Histogram | Distribution of pattern confidence scores |

### Supporting Metrics

| Metric | Type | Description |
|--------|------|-------------|
| `learning_attempts_total` | Counter | All learning attempts by status |
| `task_symbolic_conversions_total` | Counter | Neural→symbolic conversions |
| `error_triggered_resynthesis_total` | Counter | Error-based re-synthesis events |
| `learning_cooldown_violations_total` | Counter | Cooldown-blocked attempts |

### Labels

**Common Labels:**
- `namespace` - Kubernetes namespace
- `agent` - Agent name
- `task_name` - Specific task being learned

**Trigger Type Labels:**
- `pattern_detection` - Learning from execution traces
- `error_recovery` - Learning from failures
- `manual` - User-initiated learning

## Querying Examples

### Learning Velocity
```promql
# Tasks learned per hour
increase(learning_tasks_total[1h])

# Learning rate per agent  
rate(learning_tasks_total[5m])
```

### Success Rate Analysis
```promql
# Overall success rate
sum(rate(learning_attempts_total{status="success"}[1h])) / 
sum(rate(learning_attempts_total[1h]))

# Success rate by agent
learning_success_rate
```

### Cost Optimization
```promql
# Daily cost savings
increase(learning_cost_savings_usd_total[24h])

# Cost savings velocity
rate(learning_cost_savings_usd_total[5m])
```

### Error Recovery
```promql
# Error re-synthesis rate
rate(error_triggered_resynthesis_total[5m])

# Error types breakdown
sum by (error_type) (error_triggered_resynthesis_total)
```

## Troubleshooting

### Low Learning Success Rate
**Symptoms:** `learning_success_rate < 0.6`
**Causes:**
- Insufficient execution traces (`learning_threshold` too low)
- Pattern confidence threshold too high (`pattern_confidence_min`)
- Synthesis service issues
- Invalid task patterns

**Investigation:**
```promql
# Check pattern confidence
pattern_confidence_distribution

# Check failure reasons
learning_attempts_total{status="failed"}
```

### High Error Re-synthesis Rate  
**Symptoms:** `rate(error_triggered_resynthesis_total[5m]) > 0.01`
**Causes:**
- Unstable task execution environment
- Poor initial synthesis quality
- External service dependency issues

**Investigation:**
```promql
# Error patterns
sum by (error_type) (error_triggered_resynthesis_total)

# Affected tasks
sum by (task_name) (error_triggered_resynthesis_total)
```

### Learning Stagnation
**Symptoms:** No learning progress despite attempts
**Causes:**
- Pattern confidence below threshold
- Cooldown periods too long
- Non-deterministic task behavior

**Investigation:**
```promql
# Cooldown violations
learning_cooldown_violations_total

# Pattern confidence trends
avg_over_time(pattern_confidence_distribution[1h])
```

## Configuration

### Learning Controller Settings
The learning controller can be configured with these parameters:

```go
LearningThreshold:    10,              // Traces needed before learning
LearningInterval:     5 * time.Minute, // Cooldown between attempts  
PatternConfidenceMin: 0.8,             // Minimum confidence for learning
MaxVersions:          5,               // ConfigMap versions to retain
```

### Alert Thresholds
Key alert thresholds can be adjusted in `learning-alerts.yaml`:

```yaml
# Success rate warnings
learning_success_rate < 0.6  # Warning
learning_success_rate < 0.3  # Critical

# Error re-synthesis rate
rate(error_triggered_resynthesis_total[5m]) > 0.01

# Pattern confidence degradation  
avg_over_time(pattern_confidence_distribution[10m]) < 0.5
```

## Integration

### With Existing Monitoring
The learning metrics integrate with existing Language Operator synthesis metrics:

```promql
# Total synthesis cost including learning
sum(synthesis_cost_usd_total) - sum(learning_cost_savings_usd_total)

# Synthesis requests vs learning events
rate(synthesis_requests_total[5m]) / rate(learning_tasks_total[5m])
```

### With OpenTelemetry
Learning events generate distributed traces for detailed analysis:

**Trace Operations:**
- `learning.reconcile` - Main learning loop
- `learning.process_trigger` - Learning trigger processing  
- `learning.estimate_cost_savings` - Cost calculation
- `learning.record_pattern_confidence` - Pattern analysis

### With Cost Tracking
Learning metrics enhance synthesis cost tracking:

```promql
# Learning ROI calculation
sum(learning_cost_savings_usd_total) / sum(synthesis_cost_usd_total)

# Break-even analysis  
learning_cost_savings_usd_total > synthesis_cost_usd_total{namespace=~".*"}
```

## Best Practices

### Metric Collection
- **Retention**: Keep learning metrics for at least 30 days to track trends
- **Sampling**: Pattern confidence histograms provide detailed distribution data
- **Aggregation**: Use recording rules for frequently-queried learning KPIs

### Alerting
- **Success Rate**: Alert on sustained low success rates (>5min duration)
- **Error Recovery**: Immediate alerts on error re-synthesis bursts
- **Cost Optimization**: Info-level alerts on negative cost trends

### Dashboard Usage
- **Time Range**: Use 1-hour default with 6-hour/24-hour views available
- **Templating**: Filter by namespace/agent for targeted analysis  
- **Annotations**: Automatic annotations show learning events in context