# Installation

This guide covers installing the Language Operator on your Kubernetes cluster.

## Requirements

### Hardware

Minimum recommended node capacity:

- **2 CPU cores** (4+ for running agents)
- **4Gi RAM** (8Gi+ for running agents) — includes the Argo workflow controller
- **15Gi persistent storage** (10Gi workspace per agent)

### Cluster

- **Kubernetes 1.26+**
- **kubectl** and **Helm 3.8+**
- **cert-manager v1.12+** — required for webhook TLS
- **NetworkPolicy-capable CNI** — Cilium, Calico, Weave, or Antrea
- **Persistent storage** — for agent workspace PVCs
- **Argo Workflows** — agents run as Argo Workflows. The operator chart bundles it as a
  subchart and installs it by default, so there is nothing extra to do. See
  [Bring your own Argo](#bring-your-own-argo) if you already run it.

Optionally, the [`argo` CLI](https://github.com/argoproj/argo-workflows/releases) — useful
for inspecting runs and invoking task-mode agents by hand, though everything it does is also
reachable through `kubectl`.

See the [Kubernetes guide](../guides/cluster-setup.md) for instructions on installing and verifying these prerequisites.

## Install via Helm

Language Operator ships as two separate Helm charts:

| Chart | Purpose |
|-------|---------|
| `language-operator/language-operator` | Operator, CRDs, RBAC, webhooks |
| `language-operator/language-operator-runtimes` | Bundled `LanguageAgentRuntime` presets (openclaw, opencode, claude-code, deepagents) |

Install the operator chart first, then the runtimes chart.

The operator chart depends on the upstream `argo-workflows` chart, pulled as OCI from
`ghcr.io/argoproj/argo-helm`. Installing from the Helm repository handles this for you.
Installing from a source checkout needs the dependency resolved first:

```bash
helm dependency build charts/language-operator
```

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
  --set config.agents.storageClassName=local-path \
  --set config.gateway.ingressClassName=traefik \
  --set config.tls.certificateIssuerName=letsencrypt-production \
  --set config.tls.certificateIssuerKind=ClusterIssuer
```

!!! note "Values vary by cluster"
    Replace `traefik` with your ingress class (e.g. `nginx`, `alb`), `local-path` with your StorageClass, and the TLS issuer with the name of your cert-manager `ClusterIssuer` or `Issuer`. All of these can also be set in a values file with `helm install -f values.yaml`.

See the [complete `values.yaml`](https://github.com/language-operator/language-operator/blob/main/charts/language-operator/values.yaml) for all available configuration options.

### 3. Install the Runtimes

The `language-operator-runtimes` chart installs the bundled `LanguageAgentRuntime` presets (openclaw, opencode, claude-code, and deepagents). These are cluster-scoped resources — install once and reference from any namespace.

```bash
helm install language-operator-runtimes \
  language-operator/language-operator-runtimes \
  --namespace language-operator
```

!!! note "Operator must be installed first"
    The runtimes chart requires the CRDs installed by the operator chart. Always install `language-operator` before `language-operator-runtimes`.

You can selectively disable runtimes you don't need:

```bash
helm install language-operator-runtimes \
  language-operator/language-operator-runtimes \
  --namespace language-operator \
  --set claude-code.enabled=false
```

### 4. Verify Installation

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
languageagentselfconfigs.langop.io
languageclusters.langop.io
languagemodels.langop.io
languagepersonas.langop.io
languagetools.langop.io
```

Check the Argo workflow controller is running — agents cannot start without it:

```bash
kubectl get pods -n language-operator -l app.kubernetes.io/name=argo-workflows-workflow-controller
kubectl get crds | grep argoproj.io
```

Expect four CRDs: `workflows`, `workflowtemplates`, `cronworkflows`, and
`workflowtaskresults`. If they are missing the operator will not start — see
[Troubleshooting](../guides/troubleshooting.md).

Check runtimes are installed:

```bash
kubectl get languageagentruntimes
```

Expected output:

```
NAME          AGE
claude-code   30s
openclaw      30s
opencode      30s
```

## Bring your own Argo

If you already run Argo Workflows, disable the bundled subchart:

```bash
helm install language-operator language-operator/language-operator \
  --namespace language-operator \
  --create-namespace \
  --set argo-workflows.enabled=false
```

Your installation must satisfy two requirements:

- The `argoproj.io` CRDs must be served — `workflows`, `workflowtemplates`, `cronworkflows`,
  and `workflowtaskresults`. The operator checks for them at startup and **exits** if they
  are missing.
- The workflow controller must watch the namespaces your LanguageClusters live in. The
  bundled subchart watches all namespaces (`controller.workflowNamespaces: []`); a
  namespace-scoped Argo installation will silently never run your agents.

## Upgrade

Upgrade the operator to the latest version:

```bash
helm repo update
helm upgrade language-operator language-operator/language-operator \
  --namespace language-operator
```

Upgrade runtimes separately:

```bash
helm upgrade language-operator-runtimes language-operator/language-operator-runtimes \
  --namespace language-operator
```

!!! warning "Breaking change: agents no longer run as Deployments"
    Agents now run as Argo Workflows. Two fields are **rejected at admission**, so any
    existing manifest that sets them will fail to apply after the upgrade:

    - `spec.deployment.replicas` — an Argo Workflow has no replica count
    - `spec.deployment.autoscaling` — there is no scale subresource to target

    Remove both. Agent HorizontalPodAutoscalers and PodDisruptionBudgets are no longer
    created; delete any left over from a previous install. The status fields
    `activeReplicas` and `readyReplicas` are replaced by `lastRunPhase` and the other
    `lastRun*` fields — see [Execution Modes](../guides/execution-modes.md#status).

    If you set `spec.deployment.serviceAccountName` to a ServiceAccount of your own, it must
    now grant `create` and `patch` on `argoproj.io/workflowtaskresults`, or every run will
    fail at completion.

!!! note "CRD schema changes"
    Helm does not update CRDs automatically on `helm upgrade`. When upgrading to a version that includes CRD changes, apply the updated CRDs first:

    ```bash
    kubectl apply -f https://raw.githubusercontent.com/language-operator/language-operator/main/charts/language-operator/templates/crds/langop.io_languageagents.yaml
    kubectl apply -f https://raw.githubusercontent.com/language-operator/language-operator/main/charts/language-operator/templates/crds/langop.io_languageagentruntimes.yaml
    kubectl apply -f https://raw.githubusercontent.com/language-operator/language-operator/main/charts/language-operator/templates/crds/langop.io_languageagentselfconfigs.yaml
    kubectl apply -f https://raw.githubusercontent.com/language-operator/language-operator/main/charts/language-operator/templates/crds/langop.io_languageclusters.yaml
    kubectl apply -f https://raw.githubusercontent.com/language-operator/language-operator/main/charts/language-operator/templates/crds/langop.io_languagemodels.yaml
    kubectl apply -f https://raw.githubusercontent.com/language-operator/language-operator/main/charts/language-operator/templates/crds/langop.io_languagepersonas.yaml
    kubectl apply -f https://raw.githubusercontent.com/language-operator/language-operator/main/charts/language-operator/templates/crds/langop.io_languagetools.yaml
    ```

    Check the release notes before upgrading to see if CRD changes are included.

## Uninstall

Remove the runtimes chart first, then the operator:

```bash
helm uninstall language-operator-runtimes --namespace language-operator
helm uninstall language-operator --namespace language-operator
```

!!! warning "Data Loss"
    Uninstalling will delete all LanguageAgent, LanguageModel, and related resources.
    Back up any important configurations before uninstalling.
