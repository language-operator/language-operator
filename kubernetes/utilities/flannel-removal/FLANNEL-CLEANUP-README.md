# Flannel Cleanup Scripts

These scripts remove all Flannel CNI remnants from Kubernetes nodes after migrating to Cilium.

## Problem

After switching from Flannel to Cilium, the following Flannel artifacts remain and can interfere with Cilium:

- **iptables rules**: `FLANNEL-FWD` and `FLANNEL-POSTRTG` chains
- **Network interfaces**: `flannel.1` VXLAN interface and `cni0` bridge
- **Routing table**: Inter-node routes via physical interfaces for pod CIDR

These remnants cause:
- Pods unable to communicate across nodes
- Connection refused errors
- Mixed routing (some traffic via Flannel routes, some via Cilium)

## Files

- `cleanup-flannel-iptables.sh` - Remove Flannel iptables rules only
- `cleanup-flannel-interfaces.sh` - Remove Flannel network interfaces only
- `cleanup-flannel-routes.sh` - Remove Flannel routing table entries only
- `cleanup-flannel-complete.sh` - Complete cleanup (all of the above)
- `cleanup-flannel-daemonset.yaml` - Kubernetes DaemonSet for automated cleanup across all nodes

## Usage

### Option 1: Automated cleanup via DaemonSet (Recommended)

Deploy the DaemonSet to clean up all nodes automatically:

```bash
# Deploy the cleanup DaemonSet
kubectl apply -f cleanup-flannel-daemonset.yaml

# Watch the cleanup progress
kubectl get pods -n kube-system -l app=flannel-cleanup -w

# Check logs from each node
kubectl logs -n kube-system -l app=flannel-cleanup --all-containers=true

# Once cleanup is complete (check logs), delete the DaemonSet
kubectl delete -f cleanup-flannel-daemonset.yaml
```

### Option 2: Manual cleanup per node

SSH into each node and run the complete cleanup script:

```bash
# Copy script to node
scp cleanup-flannel-complete.sh user@node:/tmp/

# SSH to node and run as root
ssh user@node
sudo /tmp/cleanup-flannel-complete.sh
```

### Option 3: Individual cleanup steps

If you want to clean up specific components only:

```bash
# Remove iptables rules
sudo ./cleanup-flannel-iptables.sh

# Remove network interfaces
sudo ./cleanup-flannel-interfaces.sh

# Remove routing entries
sudo ./cleanup-flannel-routes.sh
```

## Post-Cleanup Steps

After running the cleanup:

1. **Verify Cilium is healthy**:
   ```bash
   kubectl get pods -n kube-system | grep cilium
   cilium status
   ```

2. **Restart affected pods** (if they're still having connectivity issues):
   ```bash
   # Restart specific pod
   kubectl delete pod <pod-name> -n <namespace>
   
   # Or drain and uncordon the node to reschedule all pods
   kubectl drain <node> --ignore-daemonsets --delete-emptydir-data
   kubectl uncordon <node>
   ```

3. **Test connectivity**:
   ```bash
   # Run Cilium connectivity test
   cilium connectivity test
   
   # Or test specific pod-to-pod communication
   kubectl exec -it <pod1> -- ping <pod2-ip>
   ```

4. **Verify no Flannel remnants remain**:
   ```bash
   # Check iptables
   kubectl exec -n kube-system <cilium-pod> -- iptables-save | grep -i flannel
   
   # Check interfaces
   kubectl exec -n kube-system <cilium-pod> -- ip link show | grep flannel
   
   # Check routes
   kubectl exec -n kube-system <cilium-pod> -- ip route show | grep flannel
   ```

## What Gets Cleaned Up

### iptables Rules
- `FLANNEL-FWD` chain and all rules
- `FLANNEL-POSTRTG` chain and all rules
- Jump rules from `FORWARD` and `POSTROUTING` chains

### Network Interfaces
- `flannel.1` VXLAN interface (always removed)
- `cni0` bridge (requires node drain for safe removal)
- Associated veth interfaces (removed automatically when pods are rescheduled)

### Routing Table
- Inter-node routes for pod CIDR (10.42.0.0/16) via physical interfaces
- Cilium will automatically create correct routes

## Safety Notes

- **Backup**: The iptables cleanup script automatically backs up rules to `/tmp/iptables-backup-<timestamp>.txt`
- **cni0 interface**: Not automatically removed if pods are still using it. Drain the node first if needed.
- **Non-destructive**: Scripts check for existence before attempting removal
- **Cilium must be running**: Ensure Cilium is healthy before cleanup, as it will take over networking

## Troubleshooting

**Q: Cleanup script fails with "permission denied"**
A: The scripts must be run as root. Use `sudo` or run the DaemonSet which has privileged access.

**Q: Pods still can't communicate after cleanup**
A: Try restarting the affected pods or draining/uncordoning the node to force pod rescheduling with Cilium networking.

**Q: iptables rules keep coming back**
A: Check if any Flannel pods or controllers are still running: `kubectl get pods -A | grep flannel`

**Q: Some routes still point to old interfaces**
A: Manually delete them: `sudo ip route del <route>`

## Verification

Your cluster is clean when:

- No Flannel iptables rules: `iptables-save | grep -i flannel` returns nothing
- No flannel.1 interface: `ip link show flannel.1` fails with "does not exist"
- All pod routes via Cilium: `ip route show | grep 10.42` shows routes via `cilium_host` only
- Pods can communicate: Cross-node pod connectivity works

