# Operator NetworkPolicy Configuration

## Overview

The language-operator deployment requires network egress access to function properly. If your cluster enforces NetworkPolicy rules (via Cilium, Calico, Weave, or other CNI plugins), you may need to manually create a NetworkPolicy to allow the operator to reach required endpoints.

## Prerequisites

- A CNI plugin that supports NetworkPolicy enforcement (see [CNI Requirements](cni-requirements.md))
- The operator must be able to reach:
  - Kubernetes API server (for reconciliation)
  - DNS services (for name resolution)
  - Synthesis endpoint (for agent code synthesis and self-healing)

## When Is This Needed?

You need NetworkPolicy configurations if:

### Operator Issues
- Your cluster enforces NetworkPolicy rules by default
- You see connection timeout errors in operator logs when synthesis operations occur
- Agent creation with `instructions` field fails with network errors
- Self-healing operations fail to synthesize corrected code

### Dashboard Issues  
- Dashboard fails to load model names ("fetch model names" fails)
- Model creation forms are empty or show connection errors
- API calls to external providers timeout

### Model Issues
- LanguageModel resources show "NetworkPolicyError" in status conditions
- Model pods cannot reach their provider endpoints (OpenAI, Anthropic, etc.)
- Connection timeouts when models try to access external APIs

## Quick Fix

Apply the following NetworkPolicy to allow the operator to function:

```yaml
apiVersion: networking.k8s.io/v1
kind: NetworkPolicy
metadata:
  name: language-operator
  namespace: kube-system  # or your operator namespace
  labels:
    app.kubernetes.io/name: language-operator
    app.kubernetes.io/component: operator
spec:
  podSelector:
    matchLabels:
      app.kubernetes.io/name: language-operator
      app.kubernetes.io/component: operator
  policyTypes:
  - Egress
  egress:
  # Allow internal cluster traffic (Kubernetes API, webhooks, services)
  - to:
    - podSelector: {}

  # Allow DNS resolution
  - to:
    - namespaceSelector:
        matchLabels:
          kubernetes.io/metadata.name: kube-system
      podSelector:
        matchLabels:
          k8s-app: kube-dns
    ports:
    - protocol: UDP
      port: 53
    - protocol: TCP
      port: 53

  # Allow synthesis endpoint (default: http://192.168.68.54:1234/v1)
  # Customize this section based on your SYNTHESIS_ENDPOINT environment variable
  - to:
    - ipBlock:
        cidr: 192.168.68.54/32
    ports:
    - protocol: TCP
      port: 1234
```

### Apply the NetworkPolicy

```bash
# Save the above YAML to a file
kubectl apply -f operator-networkpolicy.yaml

# Verify it was created
kubectl get networkpolicy -n kube-system language-operator
```

## Customization

### Different Synthesis Endpoint

If you configured a different `SYNTHESIS_ENDPOINT` in your Helm values, update the egress rule accordingly:

**Example 1: HTTPS endpoint on standard port**
```yaml
env:
  - name: SYNTHESIS_ENDPOINT
    value: "https://api.openai.com/v1"
```

Requires:
```yaml
  - to:
    - ipBlock:
        cidr: 13.107.42.16/32  # Resolve DNS: nslookup api.openai.com
    ports:
    - protocol: TCP
      port: 443
```

**Example 2: Custom endpoint with non-standard port**
```yaml
env:
  - name: SYNTHESIS_ENDPOINT
    value: "http://my-llm-proxy.example.com:8080/v1"
```

Requires:
```yaml
  - to:
    - ipBlock:
        cidr: 10.0.1.50/32  # Resolve DNS: nslookup my-llm-proxy.example.com
    ports:
    - protocol: TCP
      port: 8080
```

### Resolving DNS Names to IPs

If your synthesis endpoint uses a DNS name, resolve it to IP addresses:

```bash
# Method 1: Using nslookup
nslookup api.openai.com

# Method 2: Using dig
dig +short api.openai.com

# Method 3: Using host
host api.openai.com

# Method 4: Using kubectl
kubectl run dns-lookup --rm -it --image=busybox --restart=Never -- nslookup api.openai.com
```

Add all returned IP addresses as separate `/32` CIDR blocks:

```yaml
  - to:
    - ipBlock:
        cidr: 13.107.42.16/32
    - ipBlock:
        cidr: 13.107.43.16/32
    ports:
    - protocol: TCP
      port: 443
```

### Multiple Synthesis Endpoints

If you need to allow multiple endpoints (for example, different model providers):

```yaml
  # OpenAI API
  - to:
    - ipBlock:
        cidr: 13.107.42.16/32
    ports:
    - protocol: TCP
      port: 443

  # Anthropic API
  - to:
    - ipBlock:
        cidr: 52.85.123.45/32
    ports:
    - protocol: TCP
      port: 443

  # Local LiteLLM proxy
  - to:
    - ipBlock:
        cidr: 192.168.68.54/32
    ports:
    - protocol: TCP
      port: 1234
```

## Verification

### Check NetworkPolicy Status

```bash
# Verify NetworkPolicy exists
kubectl get networkpolicy -n kube-system language-operator

# View NetworkPolicy details
kubectl describe networkpolicy -n kube-system language-operator

# Check if operator pods are selected
kubectl get pods -n kube-system -l app.kubernetes.io/name=language-operator \
  --show-labels
```

### Test Synthesis Operations

Create a test agent with instructions to verify synthesis works:

```yaml
apiVersion: langop.io/v1alpha1
kind: LanguageAgent
metadata:
  name: test-synthesis
  namespace: default
spec:
  instructions: |
    Create a simple agent that greets users.
  modelRefs:
    - name: my-model
      role: primary
```

```bash
# Apply the test agent
kubectl apply -f test-agent.yaml

# Check if synthesis completed successfully
kubectl get languageagent test-synthesis -o yaml | grep -A10 status

# Check operator logs for synthesis activity
kubectl logs -n kube-system -l app.kubernetes.io/name=language-operator | grep -i synthesis
```

### Expected Success Indicators

- Agent status shows `Synthesized: true`
- ConfigMap created: `test-synthesis-code`
- No connection timeout errors in operator logs
- Synthesis metrics show successful operations

## Troubleshooting

### Issue: Operator still can't reach synthesis endpoint

**Symptoms:**
- Connection timeout errors in logs
- Synthesis operations fail
- Agent status remains `Synthesized: false`

**Solutions:**

1. **Verify endpoint IP is correct:**
   ```bash
   # Re-resolve DNS
   nslookup your-endpoint-hostname

   # Update NetworkPolicy with new IPs if changed
   ```

2. **Check if CNI supports NetworkPolicy:**
   ```bash
   # Verify your CNI plugin
   kubectl get pods -n kube-system | grep -E 'cilium|calico|weave'

   # Check CNI documentation for NetworkPolicy support
   ```

3. **Verify pod selector matches:**
   ```bash
   # Get operator pod labels
   kubectl get pods -n kube-system -l app.kubernetes.io/name=language-operator \
     --show-labels

   # Ensure NetworkPolicy podSelector matches exactly
   ```

4. **Check for conflicting NetworkPolicies:**
   ```bash
   # List all NetworkPolicies in operator namespace
   kubectl get networkpolicy -n kube-system

   # Check if a deny-all policy exists
   kubectl get networkpolicy -n kube-system -o yaml | grep -A5 "policyTypes"
   ```

### Issue: DNS resolution fails

**Symptoms:**
- Operator logs show "no such host" errors
- Synthesis endpoint is a hostname (not IP)

**Solutions:**

1. **Verify DNS egress rule allows both UDP and TCP:**
   ```yaml
   ports:
   - protocol: UDP
     port: 53
   - protocol: TCP  # Required for some DNS queries
     port: 53
   ```

2. **Check if kube-dns pods are labeled correctly:**
   ```bash
   kubectl get pods -n kube-system -l k8s-app=kube-dns --show-labels

   # If label is different (e.g., k8s-app: coredns), update NetworkPolicy
   ```

3. **Test DNS resolution from operator pod:**
   ```bash
   kubectl exec -n kube-system deployment/language-operator -- \
     nslookup api.openai.com
   ```

### Issue: Kubernetes API access blocked

**Symptoms:**
- Operator can't reconcile resources
- "connection refused" or "timeout" errors for API server
- Operator repeatedly restarts

**Solutions:**

1. **Ensure internal cluster traffic is allowed:**
   ```yaml
   egress:
   - to:
     - podSelector: {}  # Allow all pods in same namespace
   ```

2. **Add explicit Kubernetes API egress rule:**
   ```yaml
   # Allow Kubernetes API server (adjust CIDR for your cluster)
   - to:
     - ipBlock:
         cidr: 10.0.0.0/8  # Internal cluster network
   - to:
     - ipBlock:
         cidr: 172.16.0.0/12  # Internal cluster network
   - to:
     - ipBlock:
         cidr: 192.168.0.0/16  # Internal cluster network
   ```

3. **Check operator ServiceAccount has correct RBAC:**
   ```bash
   kubectl get clusterrolebinding | grep language-operator
   ```

## Best Practices

1. **Use IP addresses, not DNS names** - DNS records can change, causing connectivity to break
2. **Document your endpoint configuration** - Comment the NetworkPolicy YAML with endpoint details
3. **Monitor DNS changes** - Set up alerts if your synthesis endpoint IPs change
4. **Test after updates** - Verify synthesis works after any NetworkPolicy changes
5. **Use CIDR blocks for internal networks** - Avoid listing individual pod IPs
6. **Prefer /32 for external endpoints** - Maximum specificity for security

## Related Documentation

- [CNI Requirements](cni-requirements.md) - CNI plugin compatibility
- [Operator Installation](../../chart/README.md) - Helm chart configuration
- [Synthesis Architecture](../../requirements/ARCHITECTURE.md) - How code synthesis works

## Dashboard NetworkPolicy (Auto-Configured)

Starting with v0.1.50+, the dashboard Helm chart automatically creates a NetworkPolicy when deployed on clusters with NetworkPolicy enforcement (Cilium, Calico, etc.).

### Dashboard Configuration

The dashboard needs external access to fetch model names from provider APIs. This is automatically configured via:

```yaml
# Configured in chart/charts/dashboard/values.yaml
networkPolicy:
  enabled: true                    # Auto-detects CNI capability
  allowExternalHTTPS: true         # Required for model provider APIs
  allowExternalHTTP: false         # Usually not needed
  allowedCIDRs: []                # Optional: restrict to specific networks
```

### Dashboard NetworkPolicy Template

The auto-generated NetworkPolicy allows:
- ✅ Internal cluster communication (for Kubernetes API, PostgreSQL)
- ✅ DNS resolution (kube-dns in kube-system namespace) 
- ✅ External HTTPS access (for OpenAI, Anthropic, Google, etc.)
- ❌ External HTTP access (disabled by default for security)
- ❌ All other external traffic

### Troubleshooting Dashboard Issues

If model names don't load in the dashboard:

1. **Check NetworkPolicy exists:**
   ```bash
   kubectl get networkpolicy -n language-operator language-operator-dashboard
   ```

2. **Verify dashboard pods are selected:**
   ```bash
   kubectl get pods -n language-operator -l app.kubernetes.io/name=dashboard --show-labels
   ```

3. **Check dashboard logs for network errors:**
   ```bash
   kubectl logs -n language-operator -l app.kubernetes.io/name=dashboard | grep -i "network\|timeout\|connection"
   ```

4. **Test external connectivity from dashboard pod:**
   ```bash
   kubectl exec -n language-operator deploy/language-operator-dashboard -- curl -I https://api.openai.com
   ```

## LanguageModel NetworkPolicy (Auto-Configured)

Starting with v0.1.50+, LanguageModel controllers automatically configure NetworkPolicy egress rules for known providers.

### Supported Providers (Auto-Configured)

The following providers are automatically configured with appropriate egress rules:

| Provider | Default Endpoint | Auto-Configured |
|----------|------------------|-----------------|
| `openai` | `https://api.openai.com` | ✅ |
| `anthropic` | `https://api.anthropic.com` | ✅ |
| `google` | `https://generativelanguage.googleapis.com` | ✅ |
| `groq` | `https://api.groq.com` | ✅ |
| `together` | `https://api.together.xyz` | ✅ |
| `cohere` | `https://api.cohere.ai` | ✅ |
| `mistral` | `https://api.mistral.ai` | ✅ |
| `perplexity` | `https://api.perplexity.ai` | ✅ |
| `fireworks` | `https://api.fireworks.ai` | ✅ |
| `azure` | Customer-specific | Manual config required |
| `bedrock` | Region-specific | Manual config required |
| `vertex` | Region-specific | Manual config required |
| `ollama` | Local | No external egress needed |

### Manual Egress Configuration

For providers requiring custom configuration (Azure, Bedrock, custom endpoints):

```yaml
apiVersion: langop.io/v1alpha1
kind: LanguageModel
metadata:
  name: my-azure-model
spec:
  provider: azure
  endpoint: "https://my-resource.openai.azure.com"
  egress:
  - description: "Azure OpenAI endpoint"
    to:
      cidr: "20.42.65.0/24"  # Resolve: nslookup my-resource.openai.azure.com
    ports:
    - protocol: TCP
      port: 443
```

### Troubleshooting Model Issues

If LanguageModel creation fails with network errors:

1. **Check model status conditions:**
   ```bash
   kubectl get languagemodel my-model -o yaml | grep -A 10 conditions
   ```

2. **Look for NetworkPolicyError:**
   ```bash
   kubectl describe languagemodel my-model | grep -A 5 NetworkPolicy
   ```

3. **Verify NetworkPolicy was created:**
   ```bash
   kubectl get networkpolicy -n my-namespace my-model
   ```

4. **Test connectivity from model pod:**
   ```bash
   kubectl exec -n my-namespace deploy/my-model -- curl -I https://api.openai.com
   ```

## Cilium-Specific Considerations

### Cilium Network Policy Enforcement

Cilium enforces NetworkPolicy rules by default when policies exist. Key considerations:

- **DNS Resolution**: Cilium supports DNS-based policies, but IP-based policies are more reliable
- **Policy Ordering**: Cilium uses a whitelist model - traffic is denied unless explicitly allowed
- **Debugging**: Use `cilium monitor` to trace network policy decisions

### Debugging with Cilium

1. **Check CNI detection:**
   ```bash
   kubectl get pods -n kube-system | grep cilium
   kubectl logs -n kube-system ds/cilium | grep -i networkpolicy
   ```

2. **Monitor policy enforcement:**
   ```bash
   kubectl exec -n kube-system ds/cilium -- cilium monitor --type policy-verdict
   ```

3. **Validate policy syntax:**
   ```bash
   kubectl exec -n kube-system ds/cilium -- cilium policy validate /path/to/policy.yaml
   ```

## Future Enhancement

Dashboard and model NetworkPolicy auto-configuration is available starting with v0.1.50. For older versions, manual NetworkPolicy configuration is required following the examples above.

Until then, this manual configuration provides the necessary network access for operator functionality.
