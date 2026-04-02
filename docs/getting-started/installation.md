# Installation

This guide covers installing the Language Operator on your Kubernetes cluster.

## Requirements

### Cluster Resources

Each agent deployment consumes meaningful resources. Plan accordingly before installing.

| Component | CPU Request | Memory Request | Notes |
|-----------|-------------|----------------|-------|
| Operator | 100m | 128Mi | Runs in `language-operator-system` |
| LiteLLM gateway | — | — | One per `LanguageCluster`; no explicit request |
| Agent (openclaw/opencode) | 250m | 512Mi | Per agent; 2Gi memory limit |
| Dashboard PostgreSQL | 100m | 256Mi | Optional; disable with `dashboard.postgresql.enabled=false` |

**Minimum recommended node capacity:**

- **2 CPU cores** (4+ for running agents)
- **4Gi RAM** (8Gi+ for running agents)
- **15Gi persistent storage** (10Gi workspace per agent)

!!! tip "Minimal setup"
    To minimise resource usage, disable the dashboard PostgreSQL:
    ```bash
    helm install language-operator language-operator/language-operator \
      --set dashboard.postgresql.enabled=false
    ```

### Software

- **Kubernetes 1.26+**
- **NetworkPolicy-capable CNI** - One of:
    - Cilium
    - Calico
    - Weave
    - Antrea
- **kubectl** configured to access your cluster
- **Helm 3.8+**
- **cert-manager v1.12+** — required for webhook TLS certificate provisioning
- **Default StorageClass** — required for dashboard PostgreSQL and agent workspace PVCs

Verify a default StorageClass is available:

```bash
kubectl get storageclass
```

The StorageClass marked `(default)` is used for:

- The dashboard's PostgreSQL database (10Gi, `dashboard.postgresql.persistence.enabled: true` by default)
- Workspace PVCs created for agents that use bundled runtimes (openclaw, opencode)

To install without a default StorageClass, disable persistence for each component:

```bash
helm install language-operator language-operator/language-operator \
  --set dashboard.postgresql.persistence.enabled=false
```

For individual agents, disable the workspace PVC with `spec.workspace.enabled: false` in the LanguageAgent spec.

Install cert-manager if not already present:

```bash
kubectl apply -f https://github.com/cert-manager/cert-manager/releases/download/v1.16.3/cert-manager.yaml
kubectl wait --for=condition=Available deployment --all -n cert-manager --timeout=60s
```

!!! note "Installing without cert-manager"
    If you manage webhook TLS certificates yourself, you can disable cert-manager integration:
    ```bash
    helm install language-operator language-operator/language-operator \
      --set config.webhook.certManager.enabled=false
    ```
    You are then responsible for populating the webhook server's TLS secret and setting the `caBundle` field on the webhook configurations.

## Install via Helm

### 1. Add the Helm Repository

```bash
helm repo add language-operator \
  https://language-operator.github.io/language-operator
helm repo update
```

### 2. Install the Operator

Install into the `language-operator` namespace:

```bash
helm install language-operator \
  language-operator/language-operator \
  --create-namespace \
  --namespace language-operator
```

### 3. Verify Installation

Check that the operator pod is running:

```bash
kubectl get pods -n language-operator
```

Expected output:

```
NAME                                  READY   STATUS    RESTARTS   AGE
language-operator-5f7b8d9c4d-x8z2q   1/1     Running   0          30s
```

Check CRDs are installed:

```bash
kubectl get crds | grep langop.io
```

Expected output:

```
languageagentruntimes.langop.io
languageagents.langop.io
languageclusters.langop.io
languagemodels.langop.io
languagepersonas.langop.io
languagetools.langop.io
```

## Configuration Options

The operator can be configured via Helm values. See the [Helm Configuration](../helm/configuration.md) guide for details.

### Common Configurations

**Custom image:**

```bash
helm install language-operator language-operator/language-operator \
  --set image.repository=ghcr.io/your-org/language-operator \
  --set image.tag=v1.0.0
```

**Enable network isolation:**

```bash
helm install language-operator language-operator/language-operator \
  --set networkIsolation.enabled=true
```

**Resource limits:**

```bash
helm install language-operator language-operator/language-operator \
  --set resources.limits.cpu=500m \
  --set resources.limits.memory=512Mi
```

## Upgrade

Upgrade the operator to the latest version:

```bash
helm repo update
helm upgrade language-operator language-operator/language-operator \
  --namespace language-operator
```

## Uninstall

Remove the operator (CRDs and custom resources will be deleted):

```bash
helm uninstall language-operator --namespace language-operator
```

!!! warning "Data Loss"
    Uninstalling will delete all LanguageAgent, LanguageModel, and related resources.
    Back up any important configurations before uninstalling.

## Next Steps

- [Quick Start Guide](quickstart.md) - Deploy your first agent
- [Examples](examples.md) - Common deployment patterns
- [Helm Configuration](../helm/configuration.md) - Detailed configuration reference
