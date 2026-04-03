# Preparing a Kubernetes Cluster

Before installing Language Operator, your cluster must meet a few requirements. This guide walks through the prerequisites and how to verify them.

## Supported Distributions

Language Operator runs on any CNCF-conformant Kubernetes 1.26+ cluster. Tested distributions:

| Distribution | Notes |
|---|---|
| **k3s** | Recommended for local development; includes Traefik and local-path provisioner out of the box |
| **kind** | Good for CI; requires manual CNI and StorageClass setup |
| **EKS** | Use VPC CNI + Calico for NetworkPolicy support |
| **GKE** | Enable "Network Policy" at cluster creation; uses built-in provisioner |
| **AKS** | Enable "Network Policy: Azure" or install Calico |

## Prerequisites

Install these tools locally before proceeding:

- **kubectl** — configured to access your target cluster
- **Helm 3.8+**

Verify:

```bash
kubectl version --client
helm version
```

## CNI Support

Language Operator uses `NetworkPolicy` resources to isolate agents. Your cluster's CNI plugin must support NetworkPolicy enforcement. The standard CNI plugins that ship with most cloud providers do **not** enforce NetworkPolicy on their own.

Supported CNIs:

- [Cilium](https://cilium.io/) (recommended)
- [Calico](https://www.tigera.io/project-calico/)
- [Weave Net](https://github.com/weaveworks/weave)
- [Antrea](https://antrea.io/)

Verify that NetworkPolicy is enforced by checking that your CNI pods are running:

```bash
kubectl get pods -n kube-system | grep -E 'cilium|calico|weave|antrea'
```

!!! note "k3s"
    k3s ships with Flannel by default, which does **not** enforce NetworkPolicy. Install Cilium or Calico before deploying the operator, or disable network isolation in Helm values:
    ```bash
    helm install language-operator language-operator/language-operator \
      --set networkIsolation.enabled=false
    ```

## StorageClass

Language Operator requires a StorageClass for agent workspace PVCs created by bundled runtimes (openclaw, opencode).

List available StorageClasses:

```bash
kubectl get storageclass
```

Set `config.agents.storageClassName` in your Helm values to the StorageClass you want to use. If left empty, the cluster default StorageClass is used.

## cert-manager

Language Operator uses admission webhooks, which require TLS certificates. cert-manager provisions these automatically.

Check if cert-manager is already installed:

```bash
kubectl get pods -n cert-manager
```

If not, install it:

```bash
kubectl apply -f https://github.com/cert-manager/cert-manager/releases/download/v1.16.3/cert-manager.yaml
kubectl wait --for=condition=Available deployment --all -n cert-manager --timeout=60s
```


## Traefik

Traefik is the recommended ingress controller for Language Operator. When a `LanguageCluster` has `spec.domain` set, the operator creates an Ingress resource at `gateway.<domain>` — Traefik handles routing and TLS termination.

**k3s**: Traefik is pre-installed. Verify it is running:

```bash
kubectl get pods -n kube-system | grep traefik
kubectl get svc -n kube-system traefik
```

Note the `EXTERNAL-IP` of the `traefik` service — this is the IP your DNS records should point to.

**Other clusters**: Install Traefik via Helm:

```bash
helm repo add traefik https://traefik.github.io/charts
helm repo update

helm install traefik traefik/traefik \
  --namespace traefik \
  --create-namespace \
  --set ports.web.redirectTo.port=websecure \
  --set ports.websecure.tls.enabled=true

kubectl wait --for=condition=Available deployment/traefik -n traefik --timeout=60s
```

Retrieve the external IP once the LoadBalancer is provisioned:

```bash
kubectl get svc -n traefik traefik
```

Point a wildcard DNS record (`*.<your-domain>`) at this IP, or create individual A records for each agent domain.

## Let's Encrypt

With cert-manager installed, configure a `ClusterIssuer` to automatically provision TLS certificates via Let's Encrypt.

### Staging (test first)

```bash
kubectl apply -f - <<EOF
apiVersion: cert-manager.io/v1
kind: ClusterIssuer
metadata:
  name: letsencrypt-staging
spec:
  acme:
    server: https://acme-staging-v02.api.letsencrypt.org/directory
    email: your-email@example.com
    privateKeySecretRef:
      name: letsencrypt-staging
    solvers:
      - http01:
          ingress:
            ingressClassName: traefik
EOF
```

### Production

Once staging certificates are issued successfully, switch to the production issuer:

```bash
kubectl apply -f - <<EOF
apiVersion: cert-manager.io/v1
kind: ClusterIssuer
metadata:
  name: letsencrypt-production
spec:
  acme:
    server: https://acme-v02.api.letsencrypt.org/directory
    email: your-email@example.com
    privateKeySecretRef:
      name: letsencrypt-production
    solvers:
      - http01:
          ingress:
            ingressClassName: traefik
EOF
```

Verify the issuer is ready:

```bash
kubectl get clusterissuer
# NAME                     READY   AGE
# letsencrypt-staging      True    30s
# letsencrypt-production   True    30s
```

### Using the issuer with Language Operator

Pass the issuer name when installing the operator:

```bash
--set config.tls.certificateIssuerName=letsencrypt-production \
--set config.tls.certificateIssuerKind=ClusterIssuer
```

These are included in the `helm install` command on the [installation page](../getting-started/installation.md). The operator uses them to annotate every gateway and agent Ingress it creates, so cert-manager automatically provisions and renews the TLS certificates.

!!! tip "DNS must resolve before HTTP-01 challenge"
    cert-manager proves domain ownership by serving a token over HTTP. Ensure your DNS records point to the Traefik IP before creating any `LanguageCluster` with a domain.

## Verifying Cluster Readiness

Run through this checklist before installing:

```bash
# Kubernetes version
kubectl version --short

# NetworkPolicy-capable CNI pods running
kubectl get pods -n kube-system | grep -E 'cilium|calico|weave|antrea'

# StorageClass available
kubectl get storageclass

# cert-manager running
kubectl get pods -n cert-manager

# Traefik running and has external IP
kubectl get svc -n kube-system traefik 2>/dev/null || kubectl get svc -n traefik traefik

# ClusterIssuers ready
kubectl get clusterissuer

# Sufficient node resources (operator + gateway + one agent needs ~4Gi RAM)
kubectl top nodes
```

## Next Steps

- [Next: Install Language Operator](../getting-started/installation.md)
