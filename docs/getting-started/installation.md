# Installation

This guide covers installing the Language Operator on your Kubernetes cluster.

## Requirements

- **Kubernetes 1.26+**
- **NetworkPolicy-capable CNI** - One of:
    - Cilium
    - Calico
    - Weave
    - Antrea
- **kubectl** configured to access your cluster
- **Helm 3.8+**
- **cert-manager v1.12+** — required for webhook TLS certificate provisioning

Install cert-manager if not already present:

```bash
kubectl apply -f https://github.com/cert-manager/cert-manager/releases/latest/download/cert-manager.yaml
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

Install into the `language-operator-system` namespace:

```bash
helm install language-operator \
  language-operator/language-operator \
  --create-namespace \
  --namespace language-operator-system
```

### 3. Verify Installation

Check that the operator pod is running:

```bash
kubectl get pods -n language-operator-system
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
  --set config.networkIsolationEnabled=true
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
  --namespace language-operator-system
```

## Uninstall

Remove the operator (CRDs and custom resources will be deleted):

```bash
helm uninstall language-operator --namespace language-operator-system
```

!!! warning "Data Loss"
    Uninstalling will delete all LanguageAgent, LanguageModel, and related resources.
    Back up any important configurations before uninstalling.

## Next Steps

- [Quick Start Guide](quickstart.md) - Deploy your first agent
- [Examples](examples.md) - Common deployment patterns
- [Helm Configuration](../helm/configuration.md) - Detailed configuration reference
