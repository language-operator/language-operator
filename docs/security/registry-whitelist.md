# Container Registry Whitelist

Language Operator validates all container images against a configurable registry whitelist to prevent arbitrary image execution (CVE-003 mitigation).

## Overview

**Problem:** An attacker with access to create LanguageAgent or LanguageTool resources could specify malicious container images from untrusted registries.

**Solution:** All images are validated against an allowlist of approved registries. Images from non-whitelisted registries are blocked.

**Implementation:**
- ConfigMap-based configuration (`operator-config`)
- Supports exact domain matches and wildcard patterns
- Default allowlist includes major public registries
- Customizable per-cluster

## How It Works

### Validation Flow

```
User creates LanguageAgent or LanguageTool
              ↓
Extract registry from image reference
  git.theryans.io/language-operator/tool:latest
              ↓
            Registry: git.theryans.io
              ↓
Load allowed-registries from ConfigMap
              ↓
Check if registry matches any allowed pattern
              ↓
        Match found?
       /           \
     Yes            No
      ↓             ↓
  Create pod    Block creation
                Set status error
```

### Pattern Matching

**Exact domain match:**
```yaml
allowed-registries: |
  docker.io
  quay.io
  git.theryans.io
```

Matches:
- ✅ `docker.io/library/alpine:latest`
- ✅ `quay.io/cilium/cilium:v1.18.0`
- ✅ `git.theryans.io/language-operator/tool:latest`

Does NOT match:
- ❌ `gcr.io/project/image:tag`
- ❌ `subdomain.docker.io/image:tag` (subdomain not allowed)

**Wildcard subdomain match:**
```yaml
allowed-registries: |
  *.gcr.io
  *.amazonaws.com
```

Matches:
- ✅ `gcr.io/project/image:tag`
- ✅ `us.gcr.io/project/image:tag`
- ✅ `eu.gcr.io/project/image:tag`
- ✅ `123456789.dkr.ecr.us-west-2.amazonaws.com/repo:tag`

Does NOT match:
- ❌ `gcr.io.evil.com/image:tag` (not a subdomain)
- ❌ `docker.io/amazonaws.com/image:tag` (different registry)

**Important:** Wildcards only work for **subdomains**, not arbitrary string matching.

## Default Configuration

### ConfigMap Location

**File:** `src/config/manager/registry-whitelist.yaml`

**Deployed to:** `kube-system/operator-config`

### Default Allowlist

```yaml
apiVersion: v1
kind: ConfigMap
metadata:
  name: operator-config
  namespace: kube-system
data:
  allowed-registries: |
    docker.io
    gcr.io
    *.gcr.io
    quay.io
    ghcr.io
    registry.k8s.io
    codeberg.org
    gitlab.com
    *.amazonaws.com
    *.azurecr.io
```

**Included by default:**

| Registry | Pattern | Notes |
|----------|---------|-------|
| Docker Hub | `docker.io` | Public images, official libraries |
| Google GCR | `gcr.io`, `*.gcr.io` | Google Container Registry (regional) |
| Red Hat Quay | `quay.io` | Red Hat's public registry |
| GitHub GHCR | `ghcr.io` | GitHub Container Registry |
| Kubernetes | `registry.k8s.io` | Official Kubernetes images |
| Codeberg | `codeberg.org` | FOSS-friendly registry |
| GitLab | `gitlab.com` | GitLab Container Registry |
| AWS ECR | `*.amazonaws.com` | Amazon Elastic Container Registry |
| Azure ACR | `*.azurecr.io` | Azure Container Registry |

**Rationale:** The default list includes major public registries to support common use cases. Operators should **customize** this list based on their security requirements.

## Customization Guide

### Adding a Registry

**Scenario:** You want to use images from your private registry `registry.example.com`

1. **Edit the ConfigMap:**
   ```bash
   kubectl edit configmap operator-config -n kube-system
   ```

2. **Add your registry to the list:**
   ```yaml
   data:
     allowed-registries: |
       docker.io
       gcr.io
       *.gcr.io
       quay.io
       ghcr.io
       registry.k8s.io
       registry.example.com  # <-- Add this line
   ```

3. **Save and exit** (`:wq` in vim)

4. **Restart the operator to reload:**
   ```bash
   kubectl rollout restart deployment language-operator -n kube-system
   ```

5. **Verify:**
   ```bash
   kubectl logs -n kube-system deployment/language-operator | grep "Loaded allowed registries"
   ```

   Expected output:
   ```
   Loaded allowed registries: [docker.io gcr.io *.gcr.io quay.io ghcr.io registry.k8s.io registry.example.com]
   ```

### Removing a Registry

**Scenario:** You want to block images from Docker Hub for security

1. **Edit the ConfigMap:**
   ```bash
   kubectl edit configmap operator-config -n kube-system
   ```

2. **Remove `docker.io` from the list:**
   ```yaml
   data:
     allowed-registries: |
       gcr.io
       *.gcr.io
       quay.io
       ghcr.io
       registry.k8s.io
       # docker.io removed
   ```

3. **Save, exit, and restart operator** (see above)

4. **Verify:**
   ```bash
   # Try to create agent with Docker Hub image
   cat <<EOF | kubectl apply -f -
   apiVersion: langop.io/v1alpha1
   kind: LanguageAgent
   metadata:
     name: test-dockerhub
   spec:
     instructions: "Test agent"
     image: docker.io/library/alpine:latest
   EOF

   # Check status (should be blocked)
   kubectl get languageagent test-dockerhub -o jsonpath='{.status.conditions[?(@.type=="ImageValidated")]}'
   ```

   Expected:
   ```json
   {
     "type": "ImageValidated",
     "status": "False",
     "reason": "RegistryNotAllowed",
     "message": "Image registry 'docker.io' is not in whitelist"
   }
   ```

### Private Registry Only

**Scenario:** Maximum security—only allow your private registry

1. **Edit the ConfigMap:**
   ```bash
   kubectl edit configmap operator-config -n kube-system
   ```

2. **Replace entire list with your registry:**
   ```yaml
   data:
     allowed-registries: |
       git.theryans.io
   ```

3. **Save, exit, and restart operator**

**Benefits:**
- ✅ Complete control over all images
- ✅ No supply chain risk from public registries
- ✅ Audit all images in one place

**Requirements:**
- All agent and tool images must be mirrored to private registry
- All base images (Alpine, etc.) must be available
- Registry must be highly available

### Using Wildcards

**Scenario:** Allow all regional Google Container Registry endpoints

```yaml
allowed-registries: |
  gcr.io
  *.gcr.io
```

**This allows:**
- `gcr.io/project/image`
- `us.gcr.io/project/image`
- `eu.gcr.io/project/image`
- `asia.gcr.io/project/image`
- `us-central1.gcr.io/project/image`

**This does NOT allow:**
- `docker.io/gcr.io/image` (wrong registry)
- `gcr.io.evil.com/image` (not a subdomain)

**Security consideration:** Wildcards increase attack surface. Use specific domains when possible.

## Configuration Patterns

### Pattern 1: Public + Private (Default)

**Use case:** Development, testing, mixed workloads

```yaml
allowed-registries: |
  docker.io
  gcr.io
  *.gcr.io
  quay.io
  ghcr.io
  git.theryans.io  # Your private registry
```

**Pros:** Flexible, easy to get started
**Cons:** Larger attack surface, supply chain risk

### Pattern 2: Private Only

**Use case:** Production, high-security environments

```yaml
allowed-registries: |
  git.theryans.io
```

**Pros:** Maximum security, full control
**Cons:** Requires mirroring all images

### Pattern 3: Specific Public Registries

**Use case:** Production with vetted public images

```yaml
allowed-registries: |
  registry.k8s.io  # Official Kubernetes images
  quay.io          # Red Hat images
  git.theryans.io  # Your private registry
```

**Pros:** Balance of security and convenience
**Cons:** Must audit each allowed registry

### Pattern 4: Cloud Provider Only

**Use case:** AWS/Azure/GCP-specific deployments

**AWS ECR:**
```yaml
allowed-registries: |
  *.amazonaws.com
```

**Azure ACR:**
```yaml
allowed-registries: |
  *.azurecr.io
```

**Google GCR:**
```yaml
allowed-registries: |
  gcr.io
  *.gcr.io
```

**Pros:** Consistent with cloud IAM policies
**Cons:** Vendor lock-in

## Status Reporting

### Allowed Image

**Agent status:**
```yaml
status:
  phase: Running
  conditions:
  - type: ImageValidated
    status: "True"
    reason: RegistryAllowed
    message: "Image registry 'git.theryans.io' is in whitelist"
```

### Blocked Image

**Agent status:**
```yaml
status:
  phase: Pending
  conditions:
  - type: ImageValidated
    status: "False"
    reason: RegistryNotAllowed
    message: "Image registry 'evil-registry.com' is not in whitelist. Allowed: docker.io, gcr.io, *.gcr.io, quay.io, ghcr.io, registry.k8s.io, git.theryans.io"
```

**No pod created:** The operator will NOT create a pod for agents/tools with blocked images.

### CLI Visibility

```bash
$ aictl agent status my-agent

Agent: my-agent
Phase: Pending
❌ ERROR: Image validation failed
   Registry 'untrusted-registry.com' is not in whitelist.
   Allowed registries: docker.io, gcr.io, quay.io, git.theryans.io

Update the registry whitelist:
  kubectl edit configmap operator-config -n kube-system
```

## Troubleshooting

### Image Blocked But Should Be Allowed

**Symptom:**
```yaml
status:
  conditions:
  - type: ImageValidated
    status: "False"
    reason: RegistryNotAllowed
    message: "Image registry 'my-registry.com' is not in whitelist"
```

**Diagnosis:**
```bash
# Check current whitelist
kubectl get configmap operator-config -n kube-system -o jsonpath='{.data.allowed-registries}'

# Check operator logs
kubectl logs -n kube-system deployment/language-operator | grep "allowed registries"
```

**Resolution:**

1. Add registry to ConfigMap (see "Adding a Registry" above)
2. Restart operator
3. Verify registry appears in logs
4. Delete and recreate agent/tool

### Wildcard Not Matching

**Symptom:** `*.gcr.io` doesn't match `us.gcr.io`

**Diagnosis:**
```bash
# Check exact pattern in ConfigMap
kubectl get configmap operator-config -n kube-system -o yaml
```

**Resolution:**

1. Ensure wildcard syntax is exactly `*.domain.com`
2. No spaces before/after wildcard
3. Wildcard only works for subdomains, not paths

### ConfigMap Changes Not Applied

**Symptom:** Added registry to ConfigMap but still blocked

**Diagnosis:**
```bash
# Check ConfigMap was saved
kubectl get configmap operator-config -n kube-system -o yaml

# Check operator logs (may not have reloaded)
kubectl logs -n kube-system deployment/language-operator
```

**Resolution:**

1. Verify ConfigMap changes saved
2. Restart operator: `kubectl rollout restart deployment language-operator -n kube-system`
3. Wait for operator to be ready
4. Recreate agent/tool

## Security Best Practices

### For Operators

1. **Minimize the whitelist**
   - Only include registries you actively use
   - Remove unused public registries
   - Review quarterly

2. **Avoid wildcards when possible**
   - Use specific domains instead of `*.example.com`
   - Wildcards increase attack surface
   - Document why each wildcard is needed

3. **Audit registry contents**
   - Scan images for vulnerabilities
   - Use admission controllers (e.g., OPA, Kyverno)
   - Monitor image pulls

4. **Use private registry for production**
   - Mirror all images to private registry
   - Control exactly what runs in cluster
   - Implement image signing

5. **Document the whitelist**
   - Explain why each registry is allowed
   - Track who requested each addition
   - Audit changes via Git

6. **Set up alerts**
   - Alert on `ImageValidated: False` conditions
   - Monitor ConfigMap changes
   - Track blocked image attempts

### For Agent Authors

1. **Use whitelisted registries**
   - Check operator documentation for allowed registries
   - Test in dev environment first
   - Request registry additions via proper channels

2. **Specify full image references**
   - Always include registry: `git.theryans.io/repo/image:tag`
   - Don't rely on default registry
   - Pin specific versions, not `latest`

3. **Document image sources**
   - Explain why specific images are needed
   - Link to image documentation
   - Specify minimum/maximum versions

## Migration Guide

### From No Whitelist to Whitelist

**Scenario:** Upgrading from v0.1.0 (no whitelist) to v0.2.0 (whitelist enabled)

1. **Before upgrade, audit existing agents/tools:**
   ```bash
   kubectl get languageagents,languagetools -A -o jsonpath='{range .items[*]}{.metadata.name}{"\t"}{.spec.image}{"\n"}{end}'
   ```

2. **Extract unique registries:**
   ```bash
   kubectl get languageagents,languagetools -A -o json | \
     jq -r '.items[].spec.image | split("/")[0]' | \
     sort -u
   ```

3. **Add all to whitelist:**
   ```yaml
   allowed-registries: |
     docker.io
     gcr.io
     git.theryans.io
     custom-registry.example.com
     # ... all unique registries
   ```

4. **Upgrade operator**

5. **Verify all agents/tools still work:**
   ```bash
   kubectl get languageagents,languagetools -A
   # Check all show ImageValidated: True
   ```

## Verification Checklist

After configuring the registry whitelist:

- [ ] ConfigMap `operator-config` exists in `kube-system`
- [ ] Operator logs show "Loaded allowed registries: [...]"
- [ ] Test agent with allowed registry → `ImageValidated: True`
- [ ] Test agent with blocked registry → `ImageValidated: False`
- [ ] Existing agents still running
- [ ] Documentation updated with allowed registries
- [ ] Team notified of whitelist policy

## References

- [CVE-003 Mitigation Details](cve-mitigations.md#cve-003)
- [Security Overview](README.md)
- [Kubernetes Image Pull Policy](https://kubernetes.io/docs/concepts/containers/images/)
- [Container Registry Security Best Practices](https://cloud.google.com/architecture/best-practices-for-operating-containers#security)
