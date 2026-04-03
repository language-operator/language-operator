# Installation

This guide covers installing the Language Operator on your Kubernetes cluster.

## Requirements

### Hardware

Minimum recommended node capacity:

- **2 CPU cores** (4+ for running agents)
- **4Gi RAM** (8Gi+ for running agents)
- **15Gi persistent storage** (10Gi workspace per agent)

### Cluster

- **Kubernetes 1.26+**
- **kubectl** and **Helm 3.8+**
- **cert-manager v1.12+** — required for webhook TLS
- **NetworkPolicy-capable CNI** — Cilium, Calico, Weave, or Antrea
- **Persistent storage** — for agent workspace PVCs

See the [Kubernetes guide](../guides/cluster-setup.md) for instructions on installing and verifying these prerequisites.

## Install via Helm

### 1. Add the Helm Repository

```bash
helm repo add language-operator \
  https://language-operator.github.io/language-operator

helm repo update
```

### 2. Install the Operator

Install into the `language-operator` namespace, substituting the values for your cluster:

```bash
helm install language-operator \
  language-operator/language-operator \
  --create-namespace \
  --namespace language-operator \
  --set config.agents.ingressClassName=traefik \
  --set config.agents.ingressControllerNamespace=traefik \
  --set config.agents.storageClassName=local-path \
  --set config.gateway.ingressClassName=traefik \
  --set config.tls.certificateIssuerName=letsencrypt-production \
  --set config.tls.certificateIssuerKind=ClusterIssuer
```

!!! note "Values vary by cluster"
    Replace `traefik` with your ingress class (e.g. `nginx`, `alb`), `local-path` with your StorageClass, and the TLS issuer with the name of your cert-manager `ClusterIssuer` or `Issuer`. All of these can also be set in a values file with `helm install -f values.yaml`.

### 3. Verify Installation

Check that the operator pod is running:

```bash
kubectl get pods -n language-operator
```

Expected output:

```
NAME                                  READY   STATUS    RESTARTS   AGE
language-operator-5f7b8d9c4d-x8z2q    1/1     Running   0          30s
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
