#!/usr/bin/env bash
set -euo pipefail

# Complete working Flannel cleanup procedure for K3s with Cilium
# Usage: ./cleanup-flannel-complete-working.sh <node>

if [ $# -eq 0 ]; then
    echo "Usage: $0 <hostname>"
    echo "Example: $0 dl1"
    exit 1
fi

HOST="$1"

echo "========================================"
echo "   Complete Flannel Cleanup"
echo "========================================"
echo "Target: $HOST"
echo ""

# Step 1: Stop k3s
echo "Step 1: Stopping k3s..."
ssh james@"$HOST" 'sudo systemctl stop k3s'
sleep 3
echo "✓ k3s stopped"
echo ""

# Step 2: Remove Flannel files
echo "Step 2: Removing Flannel configuration files..."
ssh james@"$HOST" 'sudo rm -f /var/lib/rancher/k3s/agent/etc/cni/net.d/10-flannel.conflist && echo "✓ Removed Flannel CNI config"'
ssh james@"$HOST" 'sudo rm -rf /run/flannel && echo "✓ Removed /run/flannel"'
ssh james@"$HOST" 'sudo rm -rf /var/lib/rancher/k3s/agent/etc/flannel && echo "✓ Removed /var/lib/rancher/k3s/agent/etc/flannel"'
ssh james@"$HOST" 'sudo rm -rf /var/lib/rancher/k3s/data/cni/flannel && echo "✓ Removed /var/lib/rancher/k3s/data/cni/flannel"'
echo ""

# Step 3: Start k3s
echo "Step 3: Starting k3s..."
ssh james@"$HOST" 'sudo systemctl start k3s'
echo "✓ k3s started"
echo ""

# Step 4: Wait for system to stabilize
echo "Step 4: Waiting for k3s to stabilize..."
sleep 15
echo "✓ System stabilized"
echo ""

# Step 5: Remove Flannel iptables rules
echo "Step 5: Removing Flannel iptables rules..."
ssh james@"$HOST" 'sudo iptables -D FORWARD -m comment --comment "flanneld forward" -j FLANNEL-FWD 2>/dev/null && echo "✓ Removed FLANNEL-FWD jump" || echo "  (already removed)"'
ssh james@"$HOST" 'sudo iptables -F FLANNEL-FWD 2>/dev/null && echo "✓ Flushed FLANNEL-FWD" || echo "  (already removed)"'
ssh james@"$HOST" 'sudo iptables -X FLANNEL-FWD 2>/dev/null && echo "✓ Deleted FLANNEL-FWD chain" || echo "  (already removed)"'
ssh james@"$HOST" 'sudo iptables -t nat -D POSTROUTING -m comment --comment "flanneld masq" -j FLANNEL-POSTRTG 2>/dev/null && echo "✓ Removed FLANNEL-POSTRTG jump" || echo "  (already removed)"'
ssh james@"$HOST" 'sudo iptables -t nat -F FLANNEL-POSTRTG 2>/dev/null && echo "✓ Flushed FLANNEL-POSTRTG" || echo "  (already removed)"'
ssh james@"$HOST" 'sudo iptables -t nat -X FLANNEL-POSTRTG 2>/dev/null && echo "✓ Deleted FLANNEL-POSTRTG chain" || echo "  (already removed)"'
echo ""

# Step 6: Remove Flannel routes
echo "Step 6: Removing Flannel routes..."
for cidr in "10.42.0.0/24" "10.42.1.0/24" "10.42.2.0/24" "10.42.4.0/24" "10.42.5.0/24"; do
    # Get the gateway for this CIDR from the routing table
    gateway=$(ssh james@"$HOST" "sudo ip route show $cidr 2>/dev/null | grep -oP 'via \K[\d.]+' || true")
    if [ -n "$gateway" ] && [ "$gateway" != "10.42."* ]; then
        ssh james@"$HOST" "sudo ip route del $cidr via $gateway 2>/dev/null && echo '✓ Removed route: $cidr via $gateway' || true"
    fi
done
echo ""

# Step 7: Verification
echo "========================================"
echo "   Verification"
echo "========================================"
FLANNEL_IPTABLES=$(ssh james@"$HOST" 'sudo iptables-save | grep -i flannel | wc -l')
FLANNEL_ROUTES=$(ssh james@"$HOST" 'sudo ip route show | grep "^10.42" | grep -v "cilium" | grep "via.*proto kernel" | wc -l')
CNI_CONF=$(ssh james@"$HOST" 'sudo ls /var/lib/rancher/k3s/agent/etc/cni/net.d/10-flannel.conflist 2>/dev/null && echo "EXISTS" || echo "REMOVED"')

if [ "$FLANNEL_IPTABLES" -eq 0 ]; then
    echo "✓ No Flannel iptables rules"
else
    echo "⚠ Warning: $FLANNEL_IPTABLES Flannel iptables rules remain"
fi

if [ "$FLANNEL_ROUTES" -eq 0 ]; then
    echo "✓ No Flannel routes"
else
    echo "⚠ Warning: $FLANNEL_ROUTES Flannel routes remain"
fi

if [ "$CNI_CONF" = "REMOVED" ]; then
    echo "✓ Flannel CNI config removed"
else
    echo "⚠ Warning: Flannel CNI config still exists"
fi

echo ""
echo "========================================"
echo "   Cleanup Complete for $HOST"
echo "========================================"
