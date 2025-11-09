# CNI Requirements for NetworkPolicy Enforcement

Language Operator uses Kubernetes NetworkPolicy to isolate agent and tool pods. However, NetworkPolicy is only a **declarative resource**—it requires a compatible CNI plugin to actually enforce the restrictions.

This document explains CNI requirements, installation procedures, and troubleshooting.

## The Problem: Silent Policy Failure

Kubernetes accepts NetworkPolicy resources even if your CNI doesn't enforce them. **No warnings. No errors. Policies are silently ignored.**

**Example:**
```yaml
apiVersion: networking.k8s.io/v1
kind: NetworkPolicy
metadata:
  name: deny-all-egress
spec:
  podSelector: {}
  policyTypes:
  - Egress
  egress: []  # Deny all egress
```

- **With Cilium/Calico:** Pods cannot reach external networks ✅
- **With Flannel:** Pods can reach ANY network, policy ignored ❌

## CNI Compatibility

### ✅ Compatible CNIs (Enforce NetworkPolicy)

| CNI | NetworkPolicy Support | Recommended | Notes |
|-----|----------------------|-------------|-------|
| **Cilium** | ✅ Full | **Yes** | eBPF-based, best performance, advanced features |
| **Calico** | ✅ Full | Yes | Mature, widely used, good documentation |
| **Weave Net** | ✅ Full | Yes | Simple setup, good for small clusters |
| **Antrea** | ✅ Full | Yes | VMware-backed, good for vSphere environments |

### ❌ Incompatible CNIs (Do NOT Enforce NetworkPolicy)

| CNI | NetworkPolicy Support | Why Not? |
|-----|----------------------|----------|
| **Flannel** | ❌ None | Default k3s/kubeadm CNI, only does basic overlay networking |
| **Host-local** | ❌ None | IP address management only, no policy enforcement |
| **Bridge** | ❌ None | Basic bridge networking, no policy features |

## How Language Operator Detects CNI

At startup, the operator:

1. Lists DaemonSets in `kube-system` namespace
2. Matches against known CNI DaemonSet names:
   - `cilium` → Cilium
   - `calico-node` → Calico
   - `weave-net` → Weave Net
   - `antrea-agent` → Antrea
   - `kube-flannel-ds*` → Flannel

3. Checks ConfigMaps for CNI configuration as fallback

4. Sets agent status condition `NetworkPolicyEnforced`

**Status indicators:**

✅ **Compatible CNI:**
```yaml
status:
  conditions:
  - type: NetworkPolicyEnforced
    status: "True"
    reason: CNISupported
    message: "CNI 'cilium v1.18.0' supports NetworkPolicy enforcement"
```

❌ **Incompatible CNI:**
```yaml
status:
  conditions:
  - type: NetworkPolicyEnforced
    status: "False"
    reason: IncompatibleCNI
    message: "Cluster CNI 'flannel' does not enforce NetworkPolicy. Install Cilium, Calico, Weave, or Antrea."
```

❓ **Unknown CNI:**
```yaml
status:
  conditions:
  - type: NetworkPolicyEnforced
    status: "Unknown"
    reason: CNINotDetected
    message: "Could not detect CNI plugin. NetworkPolicy enforcement status unknown."
```

## Installation Guides

### Option 1: Cilium (Recommended)

**Why Cilium?**
- eBPF-based (fastest NetworkPolicy enforcement)
- Advanced features (DNS-based policies, L7 filtering)
- Excellent observability (Hubble)
- Native on many platforms (GKE, EKS, AKS)

#### Prerequisites

- Kubernetes 1.23+
- Kernel 4.19+ (5.10+ recommended)
- Helm 3

#### Installation Steps

1. **Add Cilium Helm repository:**
   ```bash
   helm repo add cilium https://helm.cilium.io/
   helm repo update
   ```

2. **Install Cilium:**
   ```bash
   helm install cilium cilium/cilium --version 1.18.0 \
     --namespace kube-system \
     --set ipam.mode=cluster-pool \
     --set ipam.operator.clusterPoolIPv4PodCIDRList=10.42.0.0/16
   ```

   **For k3s clusters:**
   ```bash
   # k3s uses a different pod CIDR by default
   helm install cilium cilium/cilium --version 1.18.0 \
     --namespace kube-system \
     --set ipam.mode=kubernetes
   ```

3. **Wait for Cilium to be ready:**
   ```bash
   kubectl -n kube-system rollout status ds/cilium
   ```

4. **Verify installation:**
   ```bash
   cilium status --wait
   ```

   Expected output:
   ```
   /¯¯\
    /¯¯\__/¯¯\    Cilium:         OK
    \__/¯¯\__/    Operator:       OK
    /¯¯\__/¯¯\    Hubble:         OK
    \__/¯¯\__/    ClusterMesh:    disabled
       \__/

   DaemonSet         cilium             Desired: 5, Ready: 5/5, Available: 5/5
   Deployment        cilium-operator    Desired: 1, Ready: 1/1, Available: 1/1
   ```

5. **Test NetworkPolicy enforcement:**
   ```bash
   # Deploy test pod
   kubectl run test-pod --image=alpine --command -- sleep 3600

   # Test egress (should work)
   kubectl exec test-pod -- wget -O- http://example.com

   # Apply deny-all egress policy
   kubectl apply -f - <<EOF
   apiVersion: networking.k8s.io/v1
   kind: NetworkPolicy
   metadata:
     name: deny-all-egress
   spec:
     podSelector:
       matchLabels:
         run: test-pod
     policyTypes:
     - Egress
     egress: []
   EOF

   # Test egress again (should FAIL)
   kubectl exec test-pod -- timeout 5 wget -O- http://example.com
   # Expected: wget: download timed out (policy enforced!)

   # Cleanup
   kubectl delete pod test-pod
   kubectl delete networkpolicy deny-all-egress
   ```

#### Migrating from Flannel to Cilium (k3s)

**⚠️ WARNING:** This requires cluster downtime. Plan maintenance window.

1. **Backup cluster:**
   ```bash
   # On k3s master node
   sudo k3s etcd-snapshot save
   ```

2. **Disable Flannel:**
   ```bash
   # Edit k3s config
   sudo vim /etc/systemd/system/k3s.service

   # Add to ExecStart line:
   --flannel-backend=none

   # Restart k3s
   sudo systemctl daemon-reload
   sudo systemctl restart k3s
   ```

3. **Install Cilium** (see steps above)

4. **Restart all pods:**
   ```bash
   kubectl get pods --all-namespaces -o custom-columns=NAMESPACE:.metadata.namespace,NAME:.metadata.name --no-headers | while read ns pod; do
     kubectl delete pod -n $ns $pod
   done
   ```

5. **Verify networking:**
   ```bash
   kubectl run test --image=alpine --command -- sleep 3600
   kubectl exec test -- ping -c 3 8.8.8.8
   kubectl delete pod test
   ```

### Option 2: Calico

**Why Calico?**
- Mature, battle-tested
- Widely used in production
- Excellent documentation
- Works everywhere

#### Installation Steps

1. **Download Calico manifest:**
   ```bash
   curl https://raw.githubusercontent.com/projectcalico/calico/v3.27.0/manifests/calico.yaml -O
   ```

2. **Apply manifest:**
   ```bash
   kubectl apply -f calico.yaml
   ```

3. **Wait for Calico to be ready:**
   ```bash
   kubectl -n kube-system rollout status ds/calico-node
   ```

4. **Verify installation:**
   ```bash
   kubectl get pods -n kube-system -l k8s-app=calico-node
   ```

5. **Test NetworkPolicy** (same test as Cilium above)

### Option 3: Weave Net

**Why Weave Net?**
- Simple setup
- No configuration needed
- Good for small clusters

#### Installation Steps

1. **Apply Weave Net manifest:**
   ```bash
   kubectl apply -f https://github.com/weaveworks/weave/releases/download/v2.8.1/weave-daemonset-k8s.yaml
   ```

2. **Wait for Weave to be ready:**
   ```bash
   kubectl -n kube-system rollout status ds/weave-net
   ```

3. **Verify installation:**
   ```bash
   kubectl get pods -n kube-system -l name=weave-net
   ```

4. **Test NetworkPolicy** (same test as Cilium above)

### Option 4: Antrea

**Why Antrea?**
- VMware-backed
- Good for vSphere/Tanzu environments
- Modern architecture

#### Installation Steps

1. **Download Antrea manifest:**
   ```bash
   kubectl apply -f https://github.com/antrea-io/antrea/releases/download/v1.15.0/antrea.yml
   ```

2. **Wait for Antrea to be ready:**
   ```bash
   kubectl -n kube-system rollout status deployment/antrea-controller
   kubectl -n kube-system rollout status ds/antrea-agent
   ```

3. **Verify installation:**
   ```bash
   kubectl get pods -n kube-system -l app=antrea
   ```

4. **Test NetworkPolicy** (same test as Cilium above)

## Troubleshooting

### Agent Shows NetworkPolicyEnforced: False

**Symptom:**
```bash
$ kubectl get languageagent my-agent -o jsonpath='{.status.conditions[?(@.type=="NetworkPolicyEnforced")].status}'
False
```

**Diagnosis:**
```bash
# Check what CNI is detected
kubectl logs -n kube-system deployment/language-operator | grep CNI

# Check DaemonSets in kube-system
kubectl get ds -n kube-system

# Check for Flannel
kubectl get ds -n kube-system | grep flannel
```

**Resolution:**

1. If Flannel detected → Install Cilium/Calico/Weave/Antrea
2. If CNI not detected → Check DaemonSet names match expected patterns
3. If CNI detected but status still False → File a bug

### NetworkPolicy Not Working

**Symptom:** Pods can reach external networks despite deny-all egress policy

**Diagnosis:**
```bash
# Verify NetworkPolicy exists
kubectl get networkpolicies -A

# Check agent/tool pod labels match policy selector
kubectl get pod -n <namespace> --show-labels

# Check CNI logs for errors
kubectl logs -n kube-system ds/cilium  # or calico-node, weave-net, etc.
```

**Resolution:**

1. Verify policy selector matches pod labels
2. Check CNI is running on all nodes
3. Verify kernel version supports eBPF (for Cilium)
4. Check firewall rules don't bypass CNI

### CNI Detection Shows "none"

**Symptom:**
```
CNI detected: none (NetworkPolicy not supported)
```

**Diagnosis:**
```bash
# Check if any CNI DaemonSet exists
kubectl get ds -n kube-system

# Check ConfigMaps for CNI config
kubectl get cm -n kube-system | grep -E "cni|calico|cilium|weave|flannel"
```

**Resolution:**

1. If no CNI installed → Install one (see above)
2. If CNI installed but not detected → Check DaemonSet name
3. If custom CNI → File feature request to add detection

### Pods Not Starting After CNI Migration

**Symptom:** Pods stuck in `ContainerCreating` after switching CNI

**Diagnosis:**
```bash
kubectl describe pod <pod-name>
# Look for events: "failed to create pod sandbox" or "CNI plugin not initialized"
```

**Resolution:**

1. Restart all nodes (agent nodes first, then master)
2. Delete stuck pods: `kubectl delete pod <pod-name> --force --grace-period=0`
3. Verify new CNI is running on ALL nodes
4. Check kubelet logs: `journalctl -u kubelet -n 100`

## Verification Checklist

After installing a compatible CNI, verify everything works:

- [ ] CNI DaemonSet running on all nodes
- [ ] Language Operator detects CNI correctly (check logs)
- [ ] Agent status shows `NetworkPolicyEnforced: True`
- [ ] Test NetworkPolicy blocks egress (see test above)
- [ ] Existing agents restart successfully
- [ ] New agents deploy correctly

## Security Implications

### With Compatible CNI (Cilium, Calico, Weave, Antrea)

✅ **Agents are truly isolated:**
- Cannot reach external networks without explicit egress rules
- Cannot reach other namespaces
- Can only communicate within their LanguageCluster
- Zero-trust networking enforced

### With Incompatible CNI (Flannel)

❌ **Agents are NOT isolated:**
- Can reach ANY external network
- Can reach other namespaces
- NetworkPolicy resources created but ignored
- **Data exfiltration possible**
- **Lateral movement possible**

**Recommendation:** Do NOT run Language Operator in production with Flannel or other non-enforcing CNI.

## Production Recommendations

1. **Use Cilium** for best performance and features
2. **Use Calico** if you need mature, widely-adopted solution
3. **Test NetworkPolicy enforcement** after installation
4. **Monitor CNI health** (Prometheus metrics, logs)
5. **Set up alerts** for `NetworkPolicyEnforced: False` condition
6. **Document CNI version** in runbooks
7. **Plan CNI upgrades** (minor versions every 6 months)

## References

- [Kubernetes NetworkPolicy Documentation](https://kubernetes.io/docs/concepts/services-networking/network-policies/)
- [Cilium Documentation](https://docs.cilium.io/)
- [Calico Documentation](https://docs.tigera.io/calico/latest/about/)
- [Weave Net Documentation](https://www.weave.works/docs/net/latest/overview/)
- [Antrea Documentation](https://antrea.io/docs/)
- [CVE-002 Mitigation Details](cve-mitigations.md#cve-002)
