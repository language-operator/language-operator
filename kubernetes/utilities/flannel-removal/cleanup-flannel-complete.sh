#!/usr/bin/env bash
set -euo pipefail

# Complete Flannel cleanup script
# This script removes all Flannel remnants from a node:
# - iptables rules
# - network interfaces (flannel.1)
# - routing table entries
#
# Usage:
#   Run directly on each node as root:
#     sudo ./cleanup-flannel-complete.sh
#
#   OR deploy via kubectl and run on all nodes:
#     See cleanup-flannel-daemonset.yaml

echo "========================================"
echo "   Flannel Complete Cleanup Script"
echo "========================================"
echo "Node: $(hostname)"
echo "Time: $(date)"
echo ""

# Check if running as root
if [ "$EUID" -ne 0 ]; then 
    echo "⚠ Error: This script must be run as root"
    exit 1
fi

# Step 1: Cleanup iptables rules
echo "STEP 1: Cleaning up iptables rules"
echo "-----------------------------------"

# Remove FLANNEL-FWD rules
iptables -D FORWARD -m comment --comment "flanneld forward" -j FLANNEL-FWD 2>/dev/null && echo "✓ Removed FLANNEL-FWD jump rule" || echo "  (rule not found)"
iptables -F FLANNEL-FWD 2>/dev/null && echo "✓ Flushed FLANNEL-FWD chain" || echo "  (chain not found)"
iptables -X FLANNEL-FWD 2>/dev/null && echo "✓ Deleted FLANNEL-FWD chain" || echo "  (chain not found)"

# Remove FLANNEL-POSTRTG rules
iptables -t nat -D POSTROUTING -m comment --comment "flanneld masq" -j FLANNEL-POSTRTG 2>/dev/null && echo "✓ Removed FLANNEL-POSTRTG jump rule" || echo "  (rule not found)"
iptables -t nat -F FLANNEL-POSTRTG 2>/dev/null && echo "✓ Flushed FLANNEL-POSTRTG chain" || echo "  (chain not found)"
iptables -t nat -X FLANNEL-POSTRTG 2>/dev/null && echo "✓ Deleted FLANNEL-POSTRTG chain" || echo "  (chain not found)"

echo ""

# Step 2: Cleanup network interfaces
echo "STEP 2: Cleaning up network interfaces"
echo "---------------------------------------"

if ip link show flannel.1 &>/dev/null; then
    ip link set flannel.1 down 2>/dev/null || true
    ip link delete flannel.1 2>/dev/null && echo "✓ Removed flannel.1 interface" || echo "⚠ Failed to remove flannel.1"
else
    echo "✓ flannel.1 not found (already removed)"
fi

if ip link show cni0 &>/dev/null; then
    echo "⚠ cni0 interface still exists"
    echo "  This may be in use by existing pods. Consider draining the node first."
else
    echo "✓ cni0 not found"
fi

echo ""

# Step 3: Cleanup routes
echo "STEP 3: Cleaning up routing table"
echo "-----------------------------------"

# Remove Flannel routes (routes via physical interfaces for pod CIDR)
DELETED_ROUTES=0
while read -r route; do
    if [ -n "$route" ]; then
        ip route del $route 2>/dev/null && {
            echo "✓ Removed route: $route"
            ((DELETED_ROUTES++))
        } || echo "  Failed to remove: $route"
    fi
done < <(ip route show | grep "^10.42" | grep -v "cilium_host" | grep -v "flannel.1" | grep "via.*proto kernel")

if [ "$DELETED_ROUTES" -eq 0 ]; then
    echo "✓ No Flannel routes found to remove"
else
    echo "✓ Removed $DELETED_ROUTES Flannel routes"
fi

echo ""

# Final verification
echo "========================================"
echo "   Verification"
echo "========================================"

FLANNEL_IPTABLES=$(iptables-save | grep -i flannel | wc -l)
FLANNEL_NAT=$(iptables-save -t nat | grep -i flannel | wc -l)
TOTAL_IPTABLES=$((FLANNEL_IPTABLES + FLANNEL_NAT))

if [ "$TOTAL_IPTABLES" -eq 0 ]; then
    echo "✓ iptables: No Flannel rules remaining"
else
    echo "⚠ iptables: $TOTAL_IPTABLES Flannel rules still present"
fi

if ! ip link show flannel.1 &>/dev/null; then
    echo "✓ Interface: flannel.1 removed"
else
    echo "⚠ Interface: flannel.1 still exists"
fi

FLANNEL_ROUTES=$(ip route show | grep -c "flannel" || true)
if [ "$FLANNEL_ROUTES" -eq 0 ]; then
    echo "✓ Routes: No Flannel routes remaining"
else
    echo "⚠ Routes: $FLANNEL_ROUTES Flannel routes still present"
fi

echo ""
echo "========================================"
echo "   Cleanup Complete"
echo "========================================"
echo ""
echo "Next steps:"
echo "1. Verify Cilium is running: kubectl get pods -n kube-system | grep cilium"
echo "2. Restart pods if needed: kubectl delete pod <pod-name> -n <namespace>"
echo "3. Check Cilium connectivity: cilium connectivity test"
echo ""
