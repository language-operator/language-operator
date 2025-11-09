# Security Documentation

Language Operator implements defense-in-depth security to protect against malicious or compromised agents while enabling autonomous execution.

## Security Model

Language Operator follows a **zero-trust** security model:

- **Default Deny**: All dangerous operations blocked by default
- **Explicit Allow**: Network egress requires explicit allowlist
- **Defense in Depth**: Multiple validation layers (synthesis + runtime)
- **Least Privilege**: Non-root execution, read-only filesystems, resource limits
- **Network Isolation**: Kubernetes NetworkPolicy enforces pod-level isolation

## Quick Reference

### CVE Mitigations

Language Operator addresses three primary attack vectors:

| CVE | Attack Vector | Mitigation | Documentation |
|-----|---------------|------------|---------------|
| **CVE-001** | Arbitrary code execution via malicious instructions | AST-based Ruby validation | [CVE Mitigations](cve-mitigations.md#cve-001) |
| **CVE-002** | Network policy bypass via incompatible CNI | CNI detection and enforcement validation | [CVE Mitigations](cve-mitigations.md#cve-002) |
| **CVE-003** | Arbitrary container image execution | Registry whitelist validation | [CVE Mitigations](cve-mitigations.md#cve-003) |

See [CVE Mitigations](cve-mitigations.md) for detailed attack scenarios, impacts, and mitigation details.

### Security Features

#### 1. AST-Based Code Validation

**What it does:** Analyzes synthesized Ruby code using Abstract Syntax Tree parsing to detect dangerous operations before execution.

**Blocks:**
- System command execution (`system`, `exec`, backticks)
- Code evaluation (`eval`, `instance_eval`, `send`)
- File system access (`File`, `Dir`, `open`)
- Network operations (`Socket`, `TCPSocket`)
- Process manipulation (`fork`, `spawn`, `exit`)

**Architecture:**
- Validation at synthesis time (operator pod)
- Validation at runtime (agent pod)
- Single source of truth (Ruby gem validator)

**Learn more:** [ADR 001: AST-Based Ruby Validation](../adr/001-ast-based-ruby-validation.md)

#### 2. Network Isolation

**What it does:** Creates NetworkPolicy resources that isolate agent and tool pods by default, requiring explicit egress allowlists.

**Default behavior:**
- ✅ Pods can communicate within the same LanguageCluster
- ✅ Pods can reach Kubernetes DNS
- ❌ Pods CANNOT reach external networks
- ❌ Pods CANNOT reach other namespaces

**Egress allowlists:**
```yaml
apiVersion: langop.io/v1alpha1
kind: LanguageTool
metadata:
  name: web-search
spec:
  image: git.theryans.io/language-operator/web-tool:latest
  egress:
  - description: Allow DuckDuckGo search
    to:
      dns:
      - "*.duckduckgo.com"
    ports:
    - port: 443
      protocol: TCP
```

**IMPORTANT:** NetworkPolicy enforcement requires a compatible CNI plugin. See [CNI Requirements](cni-requirements.md).

#### 3. CNI Detection and Validation

**What it does:** Detects the cluster's CNI plugin and warns if NetworkPolicy enforcement is not available.

**Supported CNIs (with NetworkPolicy enforcement):**
- Cilium (recommended)
- Calico
- Weave Net
- Antrea

**Non-enforcing CNIs (detected, warning issued):**
- Flannel (default k3s CNI)

**Agent status indicator:**
```yaml
status:
  conditions:
  - type: NetworkPolicyEnforced
    status: "False"
    reason: IncompatibleCNI
    message: "Cluster CNI 'flannel' does not enforce NetworkPolicy. Install Cilium, Calico, Weave, or Antrea."
```

**Learn more:** [CNI Requirements](cni-requirements.md)

#### 4. Registry Whitelist

**What it does:** Validates that all LanguageAgent and LanguageTool images come from approved container registries.

**Default allowed registries:**
- docker.io
- gcr.io, *.gcr.io
- quay.io
- ghcr.io
- registry.k8s.io
- *.amazonaws.com
- *.azurecr.io

**Customization:**
```bash
kubectl edit configmap operator-config -n kube-system
```

**Learn more:** [Registry Whitelist](registry-whitelist.md)

#### 5. Resource Limits and Security Contexts

**What it does:** Enforces resource quotas and secure execution contexts for all agent and tool pods.

**Default security context:**
```yaml
securityContext:
  runAsNonRoot: true
  runAsUser: 65532  # langop user
  readOnlyRootFilesystem: true
  allowPrivilegeEscalation: false
  capabilities:
    drop:
    - ALL
```

**Default resource limits:**
```yaml
resources:
  requests:
    cpu: "100m"
    memory: "128Mi"
  limits:
    cpu: "1"
    memory: "512Mi"
```

#### 6. Workspace Isolation

**What it does:** Provides persistent storage for agent state while maintaining isolation between agents.

**Security properties:**
- Each agent gets a dedicated PersistentVolumeClaim
- Workspaces are NOT shared between agents
- Tools in the same cluster can access the agent's workspace
- Read-write access limited to workspace directory only
- Root filesystem remains read-only

## Security Best Practices

### For Operators

1. **Use a NetworkPolicy-enforcing CNI**
   - Install Cilium, Calico, Weave, or Antrea
   - Verify enforcement: `kubectl get networkpolicies -A`
   - Check agent status for `NetworkPolicyEnforced` condition

2. **Restrict registry whitelist**
   - Remove public registries if using only private registry
   - Use exact domains instead of wildcards when possible
   - Audit registry list regularly

3. **Monitor agent synthesis failures**
   - Review rejected code in agent status
   - Watch for patterns indicating malicious prompts
   - Alert on repeated validation failures

4. **Set resource quotas per namespace**
   - Limit total CPU/memory for agent namespaces
   - Prevent resource exhaustion attacks
   - Use `ResourceQuota` and `LimitRange`

5. **Use RBAC to restrict agent creation**
   - Limit who can create LanguageAgent resources
   - Separate prod and dev namespaces
   - Audit agent creation events

### For Agent Authors

1. **Request minimal egress permissions**
   - Only allow necessary domains
   - Use specific ports (443, 80) not wildcards
   - Document why each egress rule is needed

2. **Avoid sensitive data in instructions**
   - Don't hardcode API keys or credentials
   - Use Kubernetes Secrets for sensitive data
   - Reference secrets via environment variables

3. **Test agents in isolation first**
   - Deploy to test namespace before production
   - Verify behavior matches expectations
   - Check logs for unexpected network attempts

4. **Use personas for consistency**
   - Encode security practices in persona prompts
   - Reuse tested personas across agents
   - Version control persona definitions

## Security Updates

### v0.2.0 (Current)

- ✅ AST-based Ruby validation (CVE-001)
- ✅ CNI detection and NetworkPolicy enforcement (CVE-002)
- ✅ Registry whitelist (CVE-003)
- ✅ Non-root execution
- ✅ Read-only root filesystem
- ✅ Resource limits
- ✅ Workspace isolation

### Planned Security Features

- 🔄 Secret injection API (avoid hardcoded credentials)
- 🔄 Audit logging for agent actions
- 🔄 Rate limiting for synthesis requests
- 🔄 Image signature verification
- 🔄 Runtime behavior monitoring

## Reporting Security Issues

**DO NOT open public issues for security vulnerabilities.**

Email security findings to: `security@theryans.io`

Include:
- Description of the vulnerability
- Steps to reproduce
- Potential impact
- Suggested mitigation (if any)

We will respond within 48 hours.

## Additional Resources

- [CVE Mitigations](cve-mitigations.md) - Detailed attack scenarios and mitigations
- [CNI Requirements](cni-requirements.md) - CNI installation and troubleshooting
- [Registry Whitelist](registry-whitelist.md) - Registry configuration guide
- [ADR 001: AST-Based Validation](../adr/001-ast-based-ruby-validation.md) - Technical deep dive
- [Main README](../../README.md) - Project overview
