---
name: Network Isolation Enhancement for Cilium Clusters
about: Automatic NetworkPolicy configuration for dashboard and models on Cilium clusters
title: 'feat: Auto-configure network isolation for Cilium clusters with NetworkPolicy enforcement'
labels: ['enhancement', 'security', 'network', 'cilium', 'priority/high']
assignees: ''

---

## Problem Statement

When deploying Language Operator on Cilium clusters with NetworkPolicy enforcement activated, core functionality is broken due to missing egress access:

### 1. Dashboard Issues
- **"Fetch model names" fails** - Dashboard cannot reach external provider APIs
- **Model creation UI broken** - Forms are empty or show connection errors  
- **User experience completely broken** - Core functionality doesn't work

### 2. LanguageModel Issues  
- **Model creation fails** - Pods cannot reach provider endpoints (api.openai.com, api.anthropic.com, etc.)
- **NetworkPolicyError conditions** - Resources show network-related failures
- **Manual configuration required** - Users must specify egress rules for every model

## Root Cause Analysis

### Dashboard Problem
- **Missing NetworkPolicy**: Dashboard chart doesn't include NetworkPolicy template
- **No external egress**: Cilium blocks all external traffic by default
- **Impact**: Dashboard cannot fetch model names from any provider API

### Model Problem  
- **Incomplete auto-configuration**: `BuildEgressNetworkPolicy()` has infrastructure but limited provider mappings
- **Manual effort required**: Users must research and configure IP ranges for each provider
- **Poor UX**: Models should "just work" with known providers

## Current State

**✅ Existing Infrastructure:**
- Comprehensive NetworkPolicy framework in operator controllers
- CNI detection correctly identifies Cilium support  
- `BuildEgressNetworkPolicy()` function with DNS resolution capabilities
- CRDs have `egress` fields for fine-grained control
- Auto-configuration for OpenAI and Anthropic (basic implementation)

**❌ Critical Gaps:**
1. Dashboard has no NetworkPolicy template in Helm chart
2. Limited provider auto-configuration (only 2 of 10+ popular providers)
3. No documentation for Cilium-specific network policy setup

## Proposed Solution

### Phase 1: Dashboard Network Access (Critical - Unblocks UI)

**Create NetworkPolicy template for dashboard:**

```yaml
# chart/charts/dashboard/templates/networkpolicy.yaml
apiVersion: networking.k8s.io/v1
kind: NetworkPolicy
metadata:
  name: {{ include "dashboard.fullname" . }}
spec:
  podSelector:
    matchLabels: {{ include "dashboard.selectorLabels" . | nindent 6 }}
  policyTypes:
  - Egress
  egress:
  # Allow internal cluster communication
  - to: []
  # Allow DNS
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
  # Allow external HTTPS (configurable)
  {{- if .Values.networkPolicy.allowExternalHTTPS }}
  - to: []
    ports:
    - protocol: TCP
      port: 443
  {{- end }}
```

**Configuration Options:**
- `networkPolicy.enabled` - Enable/disable (default: true)
- `networkPolicy.allowExternalHTTPS` - Allow HTTPS egress (default: true)  
- `networkPolicy.allowedCIDRs` - Restrict to specific networks (optional)

### Phase 2: Enhanced Model Auto-Egress (UX Improvement)

**Extend provider mappings in `BuildEgressNetworkPolicy()`:**

```go
var providerDefaultEndpoints = map[string][]string{
    "openai":     {"https://api.openai.com"},
    "anthropic":  {"https://api.anthropic.com"},
    "google":     {"https://generativelanguage.googleapis.com"},
    "groq":       {"https://api.groq.com"},
    "together":   {"https://api.together.xyz"},
    "cohere":     {"https://api.cohere.ai"},
    "mistral":    {"https://api.mistral.ai"},
    "perplexity": {"https://api.perplexity.ai"},
    "fireworks":  {"https://api.fireworks.ai"},
    // Azure, Bedrock, Vertex: require manual config (region-specific)
    "ollama": {}, // Local, no external egress needed
}
```

### Phase 3: Documentation & Tooling

**Enhanced Security Documentation:**
- Cilium-specific troubleshooting guide
- NetworkPolicy debugging commands
- Provider endpoint configuration examples
- Migration guide for existing deployments

## Implementation Plan

### Priority 1: Dashboard NetworkPolicy (Immediate - Critical Path)
- [ ] Create `chart/charts/dashboard/templates/networkpolicy.yaml`
- [ ] Add configuration options to `values.yaml`
- [ ] Test dashboard model discovery on Cilium cluster
- [ ] **Result**: Unblocks dashboard functionality completely

### Priority 2: Model Auto-Egress (UX Enhancement)  
- [ ] Extend `providerDefaultEndpoints` with popular providers
- [ ] Add validation for unknown providers
- [ ] Update CRD documentation with auto-configuration details
- [ ] **Result**: Models work without manual configuration

### Priority 3: Documentation & Polish
- [ ] Update `docs/security/operator-networkpolicy.md` with Cilium examples
- [ ] Add troubleshooting section for common network issues
- [ ] Create provider endpoint reference table
- [ ] **Result**: Production-ready network isolation

## Acceptance Criteria

### Dashboard Functionality
- [ ] Dashboard loads successfully on fresh Cilium cluster with NetworkPolicies
- [ ] Model names populate correctly in dashboard UI ("fetch model names" succeeds)
- [ ] Model creation forms work without network errors
- [ ] Configuration is optional and can be disabled if needed

### Model Functionality  
- [ ] LanguageModel resources for popular providers (OpenAI, Anthropic, Google, etc.) work without manual egress configuration
- [ ] Existing manual egress rules continue to work (backward compatibility)
- [ ] Custom endpoints and Azure/Bedrock still support manual configuration
- [ ] NetworkPolicy errors include helpful troubleshooting information

### Documentation
- [ ] Comprehensive Cilium setup guide with examples
- [ ] Troubleshooting section with common network debugging commands  
- [ ] Provider compatibility table with auto-configuration status
- [ ] Migration guide for existing deployments

## Testing Plan

### Environments
- [ ] **Cilium cluster** with NetworkPolicy enforcement enabled
- [ ] **Kind cluster** with Cilium CNI for CI testing
- [ ] **Existing deployment** to verify backward compatibility

### Test Cases
- [ ] **Dashboard**: Fresh install → model names load successfully
- [ ] **Models**: Create OpenAI model → auto-configures and works
- [ ] **Models**: Create Anthropic model → auto-configures and works  
- [ ] **Custom**: Azure model with manual egress → still works
- [ ] **Local**: Ollama model → no external egress configured
- [ ] **Disabled**: NetworkPolicy disabled → no policies created

## Benefits

### User Experience
- **Zero-Config Cilium Support**: Works immediately on secured clusters
- **Intuitive Behavior**: Models "just work" with popular providers
- **Clear Error Messages**: Helpful troubleshooting when issues occur

### Security  
- **Explicit Allow-Lists**: Only permits necessary external endpoints
- **Principle of Least Privilege**: No blanket internet access
- **Configurable Restrictions**: Can be locked down for air-gapped environments

### Operations
- **Backward Compatible**: Existing manual configurations continue to work
- **Extensible**: Easy to add new providers to auto-configuration
- **Production Ready**: Comprehensive documentation for security teams

## Risk Assessment

### Low Risk Implementation
- **Additive Changes**: New functionality doesn't break existing deployments
- **Optional Features**: NetworkPolicy creation can be disabled if problematic
- **Graceful Fallbacks**: Manual egress configuration remains available

### Mitigation Strategies
- **Gradual Rollout**: Dashboard fix can be deployed independently
- **Escape Hatch**: `networkPolicy.enabled: false` disables all auto-configuration
- **Comprehensive Testing**: Validate against multiple CNI configurations

## Related Issues

- Network isolation advertised as a feature but doesn't work out-of-box on Cilium
- Dashboard model discovery fails on secured clusters  
- Manual NetworkPolicy configuration required for every model

## Environment

- **CNI**: Cilium with NetworkPolicy enforcement
- **Kubernetes**: 1.26+
- **Language Operator**: v0.1.x
- **Affected Components**: Dashboard UI, LanguageModel controller, documentation