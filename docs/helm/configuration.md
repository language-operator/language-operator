# Helm Configuration

Language Operator Helm chart configuration reference.

## values.yaml

The complete `values.yaml` is available in the [chart repository](https://github.com/language-operator/language-operator/blob/main/chart/values.yaml).

## Common Configurations

### Operator Image

```yaml
image:
  repository: ghcr.io/language-operator/language-operator
  tag: latest
  pullPolicy: Always
```

### Resource Limits

```yaml
resources:
  limits:
    cpu: 500m
    memory: 512Mi
  requests:
    cpu: 100m
    memory: 128Mi
```

### Network Isolation

```yaml
networkIsolation:
  enabled: true
```

### Gateway Configuration

```yaml
config:
  gateway:
    image: ghcr.io/language-operator/model:latest
    imagePullPolicy: IfNotPresent
```

### Telemetry

```yaml
config:
  telemetry:
    enabled: true
    endpoint: "http://otel-collector:4317"
```

## Full Reference

| Parameter | Description | Default |
|-----------|-------------|---------|
| `image.repository` | Operator container image | `ghcr.io/language-operator/language-operator` |
| `image.tag` | Image tag | `latest` |
| `image.pullPolicy` | Image pull policy | `Always` |
| `resources.limits.cpu` | CPU limit | `500m` |
| `resources.limits.memory` | Memory limit | `512Mi` |
| `networkIsolation.enabled` | Enable NetworkPolicy | `true` |
| `config.gateway.image` | LiteLLM gateway image | `ghcr.io/language-operator/model:latest` |

For the complete list, see the [values.yaml](https://github.com/language-operator/language-operator/blob/main/chart/values.yaml) file.
