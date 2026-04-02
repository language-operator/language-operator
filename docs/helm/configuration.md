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

OpenTelemetry tracing is configured via environment variables on the operator pod, not Helm values. Set `OTEL_EXPORTER_OTLP_ENDPOINT` in the operator deployment environment to enable tracing — the operator will propagate it to all agent pods automatically.

```yaml
env:
  - name: OTEL_EXPORTER_OTLP_ENDPOINT
    value: "http://otel-collector:4317"
```

## Full Reference

### Operator Deployment

| Parameter | Description | Default |
|-----------|-------------|---------|
| `replicaCount` | Number of operator replicas | `1` |
| `image.repository` | Operator container image | `ghcr.io/language-operator/language-operator` |
| `image.tag` | Image tag (defaults to chart appVersion) | `""` |
| `image.pullPolicy` | Image pull policy | `Always` |
| `imagePullSecrets` | List of image pull secrets | `[]` |
| `nameOverride` | Override the chart name | `""` |
| `fullnameOverride` | Override the full release name | `""` |
| `priorityClassName` | Pod priority class name | `""` |

### Service Account

| Parameter | Description | Default |
|-----------|-------------|---------|
| `serviceAccount.create` | Create a service account | `true` |
| `serviceAccount.annotations` | Annotations for the service account | `{}` |
| `serviceAccount.name` | Service account name (auto-generated if empty) | `""` |

### Pod Configuration

| Parameter | Description | Default |
|-----------|-------------|---------|
| `podAnnotations` | Annotations for operator pods | `{}` |
| `podLabels` | Extra labels for operator pods | `{}` |
| `podSecurityContext.runAsNonRoot` | Run as non-root user | `true` |
| `podSecurityContext.runAsUser` | UID to run as | `65532` |
| `podSecurityContext.fsGroup` | fsGroup for volumes | `65532` |
| `podSecurityContext.seccompProfile.type` | Seccomp profile | `RuntimeDefault` |
| `securityContext.allowPrivilegeEscalation` | Allow privilege escalation | `false` |
| `securityContext.capabilities.drop` | Capabilities to drop | `["ALL"]` |
| `securityContext.readOnlyRootFilesystem` | Read-only root filesystem | `true` |
| `securityContext.runAsNonRoot` | Run as non-root | `true` |
| `securityContext.runAsUser` | UID for container | `65532` |

### Service

| Parameter | Description | Default |
|-----------|-------------|---------|
| `service.type` | Service type | `ClusterIP` |
| `service.metricsPort` | Metrics port | `8443` |
| `service.annotations` | Service annotations | `{}` |
| `serviceLabels` | Extra labels for the service | `{}` |

### Resources

| Parameter | Description | Default |
|-----------|-------------|---------|
| `resources.limits.cpu` | CPU limit | `500m` |
| `resources.limits.memory` | Memory limit | `512Mi` |
| `resources.requests.cpu` | CPU request | `100m` |
| `resources.requests.memory` | Memory request | `128Mi` |

### Health Probes

| Parameter | Description | Default |
|-----------|-------------|---------|
| `livenessProbe.httpGet.path` | Liveness probe path | `/healthz` |
| `livenessProbe.initialDelaySeconds` | Initial delay | `15` |
| `livenessProbe.periodSeconds` | Period | `20` |
| `readinessProbe.httpGet.path` | Readiness probe path | `/readyz` |
| `readinessProbe.initialDelaySeconds` | Initial delay | `5` |
| `readinessProbe.periodSeconds` | Period | `10` |

> **Note:** Both probes use `config.health.port` (default `8081`) for the probe port. The `livenessProbe.httpGet.port` and `readinessProbe.httpGet.port` fields are not user-settable — the port is always derived from `config.health.port`.

### Scheduling

| Parameter | Description | Default |
|-----------|-------------|---------|
| `nodeSelector` | Node selector | `{}` |
| `tolerations` | Tolerations | `[]` |
| `affinity` | Affinity rules (default: soft anti-affinity on hostname) | see values.yaml |
| `topologySpreadConstraints` | Topology spread constraints | `[]` |

### Availability

| Parameter | Description | Default |
|-----------|-------------|---------|
| `podDisruptionBudget.enabled` | Create a PodDisruptionBudget | `true` |
| `podDisruptionBudget.minAvailable` | Minimum available pods | `1` |
| `autoscaling.enabled` | Enable HorizontalPodAutoscaler | `false` |
| `autoscaling.minReplicas` | Minimum replicas | `2` |
| `autoscaling.maxReplicas` | Maximum replicas | `10` |
| `autoscaling.targetCPUUtilizationPercentage` | Target CPU utilization | `80` |
| `autoscaling.targetMemoryUtilizationPercentage` | Target memory utilization | `80` |

### Extra Workload Configuration

| Parameter | Description | Default |
|-----------|-------------|---------|
| `volumes` | Extra volumes to mount | `[]` |
| `volumeMounts` | Extra volume mounts | `[]` |
| `env` | Extra environment variables | `[]` |
| `envFrom` | Extra env from ConfigMap or Secret | `[]` |

### Network Isolation

| Parameter | Description | Default |
|-----------|-------------|---------|
| `networkIsolation.enabled` | Create NetworkPolicy resources for all operator-managed pods | `true` |

### Operator Configuration (`config.*`)

#### Leader Election

| Parameter | Description | Default |
|-----------|-------------|---------|
| `config.leaderElection.enabled` | Enable leader election | `true` |
| `config.leaderElection.leaseDuration` | Lease duration | `15s` |
| `config.leaderElection.renewDeadline` | Renew deadline | `10s` |
| `config.leaderElection.retryPeriod` | Retry period | `2s` |

#### Health

| Parameter | Description | Default |
|-----------|-------------|---------|
| `config.health.port` | Health probe bind port | `8081` |

#### Webhook

| Parameter | Description | Default |
|-----------|-------------|---------|
| `config.webhook.enabled` | Enable admission webhooks | `true` |
| `config.webhook.port` | Webhook server port | `9443` |
| `config.webhook.certDir` | TLS certificate directory | `/tmp/k8s-webhook-server/serving-certs` |

#### Logging

| Parameter | Description | Default |
|-----------|-------------|---------|
| `config.logging.level` | Log level (`debug`, `info`, `warn`, `error`) | `info` |
| `config.logging.format` | Log format (`json`, `console`) | `json` |
| `config.logging.development` | Enable development mode logging | `false` |

#### Controller

| Parameter | Description | Default |
|-----------|-------------|---------|
| `config.controller.concurrency` | Concurrent reconcilers per controller | `5` |
| `config.controller.syncPeriod` | Full resync period | `10m` |

#### Network Policy (for agent pods)

| Parameter | Description | Default |
|-----------|-------------|---------|
| `config.networkPolicy.apiServerCIDRs` | CIDRs for Kubernetes API server access from agent pods | `["10.0.0.0/8", "172.16.0.0/12", "192.168.0.0/16"]` |
| `config.networkPolicy.customCIDRs` | Additional custom CIDR ranges | `[]` |
| `config.networkPolicy.requireNetworkPolicy` | Fail startup if CNI does not support NetworkPolicy | `false` |
| `config.networkPolicy.timeout` | Timeout for CNI detection and NetworkPolicy operations | `30s` |
| `config.networkPolicy.retries` | Retry attempts for NetworkPolicy operations | `3` |

#### Agent Ingress

| Parameter | Description | Default |
|-----------|-------------|---------|
| `config.agents.ingressClassName` | Default IngressClass for agent Ingress resources | `""` |
| `config.agents.ingressControllerNamespace` | Namespace the ingress controller runs in (used to allow ingress controller NetworkPolicy access to agent ports) | `""` |

#### Gateway

| Parameter | Description | Default |
|-----------|-------------|---------|
| `config.gateway.image` | LiteLLM gateway image (defaults to `ghcr.io/language-operator/model:latest`) | `""` |
| `config.gateway.imagePullPolicy` | Gateway image pull policy | `""` |

#### Watch

| Parameter | Description | Default |
|-----------|-------------|---------|
| `config.watch.namespaces` | Namespaces to watch (empty = all namespaces) | `[]` |

### RBAC

| Parameter | Description | Default |
|-----------|-------------|---------|
| `rbac.create` | Create RBAC resources (ClusterRole, RoleBindings) | `true` |

### Monitoring

| Parameter | Description | Default |
|-----------|-------------|---------|
| `monitoring.serviceMonitor.enabled` | Create a Prometheus ServiceMonitor | `false` |
| `monitoring.serviceMonitor.namespace` | Namespace for the ServiceMonitor | `""` |
| `monitoring.serviceMonitor.labels` | Extra labels for the ServiceMonitor | `{}` |
| `monitoring.serviceMonitor.interval` | Scrape interval | `30s` |
| `monitoring.serviceMonitor.scrapeTimeout` | Scrape timeout | `10s` |

### CRDs

| Parameter | Description | Default |
|-----------|-------------|---------|
| `crds.install` | Install CRDs as part of chart | `true` |
| `crds.keep` | Keep CRDs on chart uninstall | `true` |
| `crds.annotations` | Annotations to add to CRDs | `{}` |

---

## Dashboard (`dashboard.*`)

### Basic

| Parameter | Description | Default |
|-----------|-------------|---------|
| `dashboard.enabled` | Deploy the web dashboard | `true` |
| `dashboard.replicaCount` | Dashboard replica count | `1` |

### Dashboard Image

| Parameter | Description | Default |
|-----------|-------------|---------|
| `dashboard.image.repository` | Dashboard image | `ghcr.io/language-operator/dashboard` |
| `dashboard.image.tag` | Image tag | `latest` |
| `dashboard.image.pullPolicy` | Image pull policy | `Always` |

### Dashboard Service

| Parameter | Description | Default |
|-----------|-------------|---------|
| `dashboard.service.type` | Service type | `ClusterIP` |
| `dashboard.service.port` | Service port | `3000` |

### Dashboard Resources

| Parameter | Description | Default |
|-----------|-------------|---------|
| `dashboard.resources.limits.cpu` | CPU limit | `500m` |
| `dashboard.resources.limits.memory` | Memory limit | `512Mi` |
| `dashboard.resources.requests.cpu` | CPU request | `100m` |
| `dashboard.resources.requests.memory` | Memory request | `256Mi` |

### Dashboard Scaling

| Parameter | Description | Default |
|-----------|-------------|---------|
| `dashboard.autoscaling.enabled` | Enable HPA | `false` |
| `dashboard.autoscaling.minReplicas` | Min replicas | `1` |
| `dashboard.autoscaling.maxReplicas` | Max replicas | `3` |
| `dashboard.autoscaling.targetCPUUtilizationPercentage` | Target CPU | `80` |

### Dashboard Scheduling

| Parameter | Description | Default |
|-----------|-------------|---------|
| `dashboard.nodeSelector` | Node selector | `{}` |
| `dashboard.tolerations` | Tolerations | `[]` |
| `dashboard.affinity` | Affinity | `{}` |

### Dashboard Pod Security

| Parameter | Description | Default |
|-----------|-------------|---------|
| `dashboard.podAnnotations` | Pod annotations | `{}` |
| `dashboard.podSecurityContext.runAsNonRoot` | Run as non-root | `true` |
| `dashboard.podSecurityContext.runAsUser` | UID | `1001` |
| `dashboard.podSecurityContext.fsGroup` | fsGroup | `1001` |
| `dashboard.securityContext.allowPrivilegeEscalation` | Allow privilege escalation | `false` |
| `dashboard.securityContext.readOnlyRootFilesystem` | Read-only root filesystem | `false` |

### Dashboard Service Account

| Parameter | Description | Default |
|-----------|-------------|---------|
| `dashboard.serviceAccount.create` | Create service account | `true` |
| `dashboard.serviceAccount.annotations` | Annotations | `{}` |
| `dashboard.serviceAccount.name` | Name (auto-generated if empty) | `""` |

### Dashboard Ingress (legacy)

| Parameter | Description | Default |
|-----------|-------------|---------|
| `dashboard.ingress.enabled` | Enable legacy Ingress | `false` |
| `dashboard.ingress.className` | IngressClass | `""` |
| `dashboard.ingress.annotations` | Ingress annotations | `{}` |
| `dashboard.ingress.hosts` | Ingress hosts | see values.yaml |
| `dashboard.ingress.tls` | TLS configuration | `[]` |

### Dashboard Gateway API

| Parameter | Description | Default |
|-----------|-------------|---------|
| `dashboard.gateway.enabled` | Enable Gateway API HTTPRoute | `false` |
| `dashboard.gateway.gatewayName` | Name of existing Gateway | `""` |
| `dashboard.gateway.gatewayNamespace` | Gateway namespace | `""` |
| `dashboard.gateway.hostname` | Hostname for the dashboard | `dashboard.example.com` |
| `dashboard.gateway.tls.enabled` | Enable TLS | `false` |
| `dashboard.gateway.tls.certificateRef.name` | TLS certificate secret name | `""` |
| `dashboard.gateway.tls.certificateRef.namespace` | TLS certificate secret namespace | `""` |

### PostgreSQL

| Parameter | Description | Default |
|-----------|-------------|---------|
| `dashboard.postgresql.enabled` | Deploy bundled PostgreSQL | `true` |
| `dashboard.postgresql.auth.username` | Database user | `dashboard` |
| `dashboard.postgresql.auth.database` | Database name | `dashboard` |
| `dashboard.postgresql.auth.password` | Database password (auto-generated if empty) | `""` |
| `dashboard.postgresql.persistence.enabled` | Persist database data | `true` |
| `dashboard.postgresql.persistence.size` | PVC size | `10Gi` |
| `dashboard.postgresql.persistence.storageClass` | Storage class (uses default if empty) | `""` |
| `dashboard.postgresql.resources.limits.cpu` | CPU limit | `500m` |
| `dashboard.postgresql.resources.limits.memory` | Memory limit | `512Mi` |
| `dashboard.postgresql.resources.requests.cpu` | CPU request | `100m` |
| `dashboard.postgresql.resources.requests.memory` | Memory request | `256Mi` |

### External Database

| Parameter | Description | Default |
|-----------|-------------|---------|
| `dashboard.externalDatabase.enabled` | Use an external database instead of bundled PostgreSQL | `false` |
| `dashboard.externalDatabase.host` | Database host | `""` |
| `dashboard.externalDatabase.port` | Database port | `5432` |
| `dashboard.externalDatabase.user` | Database user | `dashboard` |
| `dashboard.externalDatabase.password` | Database password | `""` |
| `dashboard.externalDatabase.database` | Database name | `dashboard` |

### Authentication

| Parameter | Description | Default |
|-----------|-------------|---------|
| `dashboard.nextAuth.secret` | NextAuth secret (auto-generated if empty) | `""` |
| `dashboard.nextAuth.url` | NextAuth URL (auto-configured from ingress if empty) | `""` |
| `dashboard.oauth.google.clientId` | Google OAuth client ID | `""` |
| `dashboard.oauth.google.clientSecret` | Google OAuth client secret | `""` |
| `dashboard.oauth.github.clientId` | GitHub OAuth client ID | `""` |
| `dashboard.oauth.github.clientSecret` | GitHub OAuth client secret | `""` |

### Feature Flags

| Parameter | Description | Default |
|-----------|-------------|---------|
| `dashboard.features.saasMode` | Enable SaaS multi-tenant mode | `false` |
| `dashboard.features.emailAuth` | Enable email/password authentication | `true` |
| `dashboard.features.billing` | Enable billing features | `false` |
| `dashboard.features.invites` | Enable user invitations | `true` |
| `dashboard.features.signupsDisabled` | Require invitation token to sign up | `false` |

### Organization

| Parameter | Description | Default |
|-----------|-------------|---------|
| `dashboard.organization.namespacePrefix` | Prefix for organization namespaces | `language-operator-` |

### Initial Setup

| Parameter | Description | Default |
|-----------|-------------|---------|
| `dashboard.initialSetup.enabled` | Seed first admin user on startup | `false` |
| `dashboard.initialSetup.adminUser.name` | Admin user full name | `""` |
| `dashboard.initialSetup.adminUser.email` | Admin user email | `""` |
| `dashboard.initialSetup.adminUser.passwordHash` | bcrypt hash of admin password | `""` |

### Dashboard Network Policy

| Parameter | Description | Default |
|-----------|-------------|---------|
| `dashboard.networkPolicy.enabled` | Create NetworkPolicy for the dashboard | `true` |
| `dashboard.networkPolicy.allowModelDiscovery` | Allow unrestricted egress for model discovery | `true` |
| `dashboard.networkPolicy.allowExternalHTTPS` | Allow external HTTPS egress | `true` |
| `dashboard.networkPolicy.allowExternalHTTP` | Allow external HTTP egress | `false` |
| `dashboard.networkPolicy.allowedCIDRs` | Restrict egress to specific CIDRs | `[]` |
| `dashboard.networkPolicy.allowedEndpoints` | Allow specific provider endpoints | `[]` |
