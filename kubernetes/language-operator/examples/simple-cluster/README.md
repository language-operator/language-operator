# Simple Cluster Example

This example demonstrates how to use **LanguageCluster** to create network-isolated environments for AI workloads with group-based security policies.

## Overview

This example creates:
- **LanguageCluster** - An isolated network environment with 4 security groups (agents, tools, clients, ungrouped)
- **LanguageTool** - A GitHub integration tool assigned to the "tools" group
- **LanguageAgent** - A code assistant agent assigned to the "agents" group

The cluster uses Cilium for advanced network policies including DNS-based egress filtering.

## Architecture

```
┌─────────────────────────────────────────────────────────┐
│  demo-cluster (LanguageCluster)                         │
│  Namespace: demo-cluster-ns                             │
│                                                          │
│  ┌─────────────┐      ┌─────────────┐                  │
│  │   Agents    │      │    Tools    │                  │
│  │  Security   │◄────►│  Security   │                  │
│  │   Group     │      │   Group     │                  │
│  └─────────────┘      └─────────────┘                  │
│        │                     │                          │
│        │                     │                          │
│        ▼                     ▼                          │
│  • OpenAI API          • GitHub API                     │
│  • Anthropic API       • Google APIs                    │
│                                                          │
│  ┌─────────────┐      ┌─────────────┐                  │
│  │  Clients    │      │   Default   │                  │
│  │  Security   │      │  Security   │                  │
│  │   Group     │      │   Group     │                  │
│  └─────────────┘      └─────────────┘                  │
│        │                     │                          │
│        ▼                     ▼                          │
│  • Agents + Tools       • DNS only                      │
│  • External access                                      │
└─────────────────────────────────────────────────────────┘
```

## Security Groups

### Agents Group
- **Purpose**: Autonomous AI agents that interact with tools and external APIs
- **Ingress**: Accept requests from tools
- **Egress**:
  - Can call tools
  - Access to OpenAI API (`api.openai.com`)
  - Access to Anthropic API (`api.anthropic.com`)
  - DNS resolution

### Tools Group
- **Purpose**: MCP tools/functions that agents can invoke
- **Ingress**: Accept requests from agents and clients
- **Egress**:
  - Access to GitHub API (`api.github.com`)
  - Access to Google APIs (`*.googleapis.com` - wildcard)
  - DNS resolution

### Clients Group
- **Purpose**: User-facing chat interfaces and applications
- **Ingress**: Accept external requests from anywhere
- **Egress**: Can call agents and tools, DNS resolution

### Ungrouped Resources
- **Purpose**: Minimal permissions for resources without explicit group assignment
- **Note**: The "default" group name is reserved by the operator
- **Egress**: DNS resolution only

## Prerequisites

1. **Cilium CNI** installed in your cluster (REQUIRED):

   **Option 1: Using cilium CLI**
   ```bash
   cilium install --version 1.15.0
   ```

   **Option 2: Using Helm**
   ```bash
   helm repo add cilium https://helm.cilium.io/
   helm install cilium cilium/cilium --version 1.15.0 --namespace kube-system
   ```

   **Option 3: Using manifests**
   ```bash
   kubectl apply -f https://raw.githubusercontent.com/cilium/cilium/v1.15.0/install/kubernetes/quick-install.yaml
   ```

   Verify Cilium is running:
   ```bash
   kubectl get pods -n kube-system -l k8s-app=cilium
   ```

2. **Language Operator** installed in your cluster:
   ```bash
   helm install language-operator ./charts/language-operator --namespace kube-system
   ```

3. **cert-manager** installed (for webhooks):
   ```bash
   kubectl apply -f https://github.com/cert-manager/cert-manager/releases/download/v1.13.0/cert-manager.yaml
   ```

4. **GitHub credentials** (for the tool example):
   ```bash
   kubectl create secret generic github-credentials \
     --from-literal=token=your_github_token
   ```

## Quick Start

### 1. Create the Cluster

This creates the isolated network environment with security groups:

```bash
kubectl apply -f cluster.yaml
```

Wait for the cluster to be ready:

```bash
kubectl get languagecluster demo-cluster -w
```

You should see:
```
NAME           PHASE   NAMESPACE         AGE
demo-cluster   Ready   demo-cluster-ns   30s
```

### 2. Deploy the GitHub Tool

This deploys a tool in the "tools" security group:

```bash
kubectl apply -f tool.yaml
```

Verify the tool is running in the cluster namespace:

```bash
kubectl get pods -n demo-cluster-ns -l app.kubernetes.io/name=github-tool
```

### 3. Deploy the Code Assistant Agent

This deploys an agent in the "agents" security group:

```bash
kubectl apply -f agent.yaml
```

Verify the agent is running:

```bash
kubectl get pods -n demo-cluster-ns -l app.kubernetes.io/name=code-assistant
```

## Verification

### Check Cluster Status

```bash
kubectl get languagecluster demo-cluster -o yaml
```

Look for the status section showing:
- Phase: `Ready`
- Cilium installation status
- Group membership counts
- Network policy status

### Check Network Policies

View the generated NetworkPolicies:

```bash
kubectl get networkpolicies -n demo-cluster-ns
```

View the generated CiliumNetworkPolicies:

```bash
kubectl get ciliumnetworkpolicies -n demo-cluster-ns
```

### Test Network Isolation

Try to access a service from a pod:

```bash
# Get a shell in the agent pod
kubectl exec -it -n demo-cluster-ns deployment/code-assistant -- sh

# This should work (agent can access OpenAI)
curl -I https://api.openai.com

# This should work (agent can access tools)
curl http://github-tool:8080/health

# This should fail (agent cannot access arbitrary endpoints)
curl -I https://example.com
```

## Viewing Logs

### Cluster Controller Logs

```bash
kubectl logs -n language-operator -l app.kubernetes.io/name=language-operator -f | grep LanguageCluster
```

### Tool Logs

```bash
kubectl logs -n demo-cluster-ns -l app.kubernetes.io/name=github-tool -f
```

### Agent Logs

```bash
kubectl logs -n demo-cluster-ns -l app.kubernetes.io/name=code-assistant -f
```

## Modifying the Example

### Add More Tools

Create additional LanguageTool resources with `clusterRef: demo-cluster` and `group: tools`:

```yaml
apiVersion: langop.io/v1alpha1
kind: LanguageTool
metadata:
  name: slack-tool
spec:
  clusterRef: demo-cluster
  group: tools
  # ... rest of spec
```

### Update Security Rules

Edit the cluster to add new egress rules:

```bash
kubectl edit languagecluster demo-cluster
```

Add a new DNS endpoint to the agents group:

```yaml
- description: Access Slack API
  to:
    dns:
      - "slack.com"
      - "*.slack.com"
  ports:
    - protocol: TCP
      port: 443
```

### Add Custom Security Group

Add a new group for databases or other services:

```yaml
- name: databases
  description: Database services for agents
  ingress:
    - description: Accept from agents
      from:
        group: agents
      ports:
        - protocol: TCP
          port: 5432  # PostgreSQL
```

## Cleanup

Delete all resources in reverse order:

```bash
# Delete agent and tool (must remove cluster members first)
kubectl delete -f agent.yaml
kubectl delete -f tool.yaml

# Wait for resources to be deleted
kubectl wait --for=delete languageagent/code-assistant --timeout=60s
kubectl wait --for=delete languagetool/github-tool --timeout=60s

# Now delete the cluster
kubectl delete -f cluster.yaml
```

The operator will:
1. Remove all network policies
2. Delete the cluster namespace
3. Clean up Cilium policies

## Troubleshooting

### Cluster Stuck in Pending

Check if Cilium installation is progressing:

```bash
kubectl get pods -n kube-system | grep cilium
```

Check cluster status for details:

```bash
kubectl describe languagecluster demo-cluster
```

### Pod Cannot Access External API

1. Check if DNS is working:
   ```bash
   kubectl exec -it -n demo-cluster-ns deployment/code-assistant -- nslookup api.openai.com
   ```

2. Check CiliumNetworkPolicies:
   ```bash
   kubectl get ciliumnetworkpolicies -n demo-cluster-ns -o yaml
   ```

3. Check agent group has the correct egress rule for the API

### Cannot Delete Cluster

The cluster has a finalizer that prevents deletion if members still exist:

```bash
# Check for remaining members
kubectl get languagetools,languageagents,languageclients --all-namespaces | grep demo-cluster

# Delete any remaining members
kubectl delete languagetool github-tool
kubectl delete languageagent code-assistant
```

## Next Steps

- Explore the [complete LanguageCluster documentation](../../LANGUAGE_CLUSTER.md)
- See advanced examples in the `examples/` directory
- Learn about [DNS egress filtering](../../LANGUAGE_CLUSTER.md#dns-egress-filtering)
- Set up [cross-namespace service references](../../LANGUAGE_CLUSTER.md#service-references)

## References

- [LanguageCluster CRD Specification](../../api/v1alpha1/languagecluster_types.go)
- [Cilium Network Policies](https://docs.cilium.io/en/stable/network/kubernetes/policy/)
- [Kubernetes NetworkPolicy](https://kubernetes.io/docs/concepts/services-networking/network-policies/)
