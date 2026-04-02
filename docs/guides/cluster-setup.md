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

## CNI — NetworkPolicy Support

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
      --set config.networkIsolationEnabled=false
    ```

## StorageClass

Language Operator requires a **default StorageClass** for:

- Dashboard PostgreSQL (10Gi, can be disabled)
- Agent workspace PVCs created by bundled runtimes (openclaw, opencode)

Verify a default StorageClass is configured:

```bash
kubectl get storageclass
```

The StorageClass with `(default)` in its name is used automatically. If none is marked default, either mark one:

```bash
kubectl patch storageclass <name> \
  -p '{"metadata": {"annotations":{"storageclass.kubernetes.io/is-default-class":"true"}}}'
```

Or disable persistence at install time:

```bash
helm install language-operator language-operator/language-operator \
  --set dashboard.postgresql.persistence.enabled=false
```

Individual agents can also opt out of workspace storage:

```yaml
spec:
  workspace:
    enabled: false
```

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

!!! note "Without cert-manager"
    If you manage TLS certificates yourself, disable cert-manager integration:
    ```bash
    --set config.webhook.certManager.enabled=false
    ```
    You are responsible for populating the webhook TLS secret and setting the `caBundle` on webhook configurations.

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
  name: letsencrypt-prod
spec:
  acme:
    server: https://acme-v02.api.letsencrypt.org/directory
    email: your-email@example.com
    privateKeySecretRef:
      name: letsencrypt-prod
    solvers:
      - http01:
          ingress:
            ingressClassName: traefik
EOF
```

Verify the issuer is ready:

```bash
kubectl get clusterissuer
# NAME                  READY   AGE
# letsencrypt-staging   True    30s
# letsencrypt-prod      True    30s
```

### Using the issuer with Language Operator

Set the cert-manager issuer annotation when installing Language Operator:

```bash
helm install language-operator language-operator/language-operator \
  --set config.ingressAnnotations."cert-manager\.io/cluster-issuer"=letsencrypt-prod
```

When a `LanguageCluster` has `spec.domain` configured, the operator creates an Ingress for `gateway.<domain>` — cert-manager automatically provisions and renews the TLS certificate.

!!! tip "DNS must resolve before HTTP-01 challenge"
    cert-manager proves domain ownership by serving a token over HTTP. Ensure your DNS records point to the Traefik IP before applying the `LanguageCluster` with a domain.

## Verifying Cluster Readiness

Run through this checklist before installing:

```bash
# Kubernetes version
kubectl version --short

# NetworkPolicy-capable CNI pods running
kubectl get pods -n kube-system | grep -E 'cilium|calico|weave|antrea'

# Default StorageClass present
kubectl get storageclass | grep '(default)'

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

- [Install Language Operator](../getting-started/installation.md)
- [Deploy OpenClaw](openclaw.md)
- [Deploy OpenCode](opencode.md)
