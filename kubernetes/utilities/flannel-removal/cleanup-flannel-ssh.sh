#!/usr/bin/env bash
set -euo pipefail

# Script to remove Flannel state files and network remnants via SSH
# Usage: ./cleanup-flannel-ssh.sh <hostname>

if [ $# -eq 0 ]; then
    echo "Usage: $0 <hostname>"
    echo "Example: $0 dl1"
    exit 1
fi

HOST="$1"
echo "========================================"
echo "   Flannel State Cleanup via SSH"
echo "========================================"
echo "Target: $HOST"
echo "Time: $(date)"
echo ""

# Function to run commands on remote host
remote_exec() {
    ssh -t james@"$HOST" sudo bash -c "$1"
}

echo "Step 1: Checking for Flannel state files..."
echo "-------------------------------------------"
remote_exec '
if [ -d /var/lib/rancher/k3s/agent/etc/flannel ]; then
    echo "Found: /var/lib/rancher/k3s/agent/etc/flannel"
    ls -la /var/lib/rancher/k3s/agent/etc/flannel/ || true
else
    echo "Not found: /var/lib/rancher/k3s/agent/etc/flannel"
fi

if [ -f /run/flannel/subnet.env ]; then
    echo "Found: /run/flannel/subnet.env"
    cat /run/flannel/subnet.env || true
else
    echo "Not found: /run/flannel/subnet.env"
fi
'

echo ""
echo "Step 2: Stopping k3s service..."
echo "--------------------------------"
remote_exec 'systemctl stop k3s'
echo "✓ k3s stopped"

echo ""
echo "Step 3: Removing Flannel state files..."
echo "----------------------------------------"
remote_exec '
REMOVED=0

# Remove Flannel CNI config
if [ -f /var/lib/rancher/k3s/agent/etc/cni/net.d/10-flannel.conflist ]; then
    rm -f /var/lib/rancher/k3s/agent/etc/cni/net.d/10-flannel.conflist
    echo "✓ Removed /var/lib/rancher/k3s/agent/etc/cni/net.d/10-flannel.conflist"
    ((REMOVED++))
fi

# Remove Flannel configuration directory
if [ -d /var/lib/rancher/k3s/agent/etc/flannel ]; then
    rm -rf /var/lib/rancher/k3s/agent/etc/flannel
    echo "✓ Removed /var/lib/rancher/k3s/agent/etc/flannel"
    ((REMOVED++))
fi

# Remove runtime Flannel directory
if [ -d /run/flannel ]; then
    rm -rf /run/flannel
    echo "✓ Removed /run/flannel"
    ((REMOVED++))
fi

# Remove Flannel env file
if [ -f /var/lib/rancher/k3s/agent/etc/flannel.env ]; then
    rm -f /var/lib/rancher/k3s/agent/etc/flannel.env
    echo "✓ Removed /var/lib/rancher/k3s/agent/etc/flannel.env"
    ((REMOVED++))
fi

# Remove Flannel data directory
if [ -d /var/lib/rancher/k3s/data/cni/flannel ]; then
    rm -rf /var/lib/rancher/k3s/data/cni/flannel
    echo "✓ Removed /var/lib/rancher/k3s/data/cni/flannel"
    ((REMOVED++))
fi

# Note: We do NOT remove the flannel binary as it may be part of k3s data structure
# K3s will ignore it without the CNI config

if [ "$REMOVED" -eq 0 ]; then
    echo "✓ No Flannel state files found to remove"
fi
'

echo ""
echo "Step 4: Removing Flannel network artifacts..."
echo "----------------------------------------------"
remote_exec '
# Remove flannel.1 interface
if ip link show flannel.1 &>/dev/null; then
    ip link set flannel.1 down 2>/dev/null || true
    ip link delete flannel.1 2>/dev/null && echo "✓ Removed flannel.1 interface" || echo "⚠ Failed to remove flannel.1"
else
    echo "✓ flannel.1 interface not found"
fi

# Remove Flannel iptables rules
iptables -D FORWARD -m comment --comment "flanneld forward" -j FLANNEL-FWD 2>/dev/null && echo "✓ Removed FLANNEL-FWD jump rule" || true
iptables -F FLANNEL-FWD 2>/dev/null && echo "✓ Flushed FLANNEL-FWD chain" || true
iptables -X FLANNEL-FWD 2>/dev/null && echo "✓ Deleted FLANNEL-FWD chain" || true

iptables -t nat -D POSTROUTING -m comment --comment "flanneld masq" -j FLANNEL-POSTRTG 2>/dev/null && echo "✓ Removed FLANNEL-POSTRTG jump rule" || true
iptables -t nat -F FLANNEL-POSTRTG 2>/dev/null && echo "✓ Flushed FLANNEL-POSTRTG chain" || true
iptables -t nat -X FLANNEL-POSTRTG 2>/dev/null && echo "✓ Deleted FLANNEL-POSTRTG chain" || true

# Remove Flannel routes
DELETED=0
for route in $(ip route show | grep "^10.42" | grep -v "cilium_host" | grep -v "flannel.1" | grep "via.*proto kernel"); do
    network=$(echo "$route" | awk "{print \$1}")
    gateway=$(echo "$route" | awk "{print \$3}")
    if [ -n "$network" ] && [ -n "$gateway" ]; then
        ip route del "$network" via "$gateway" 2>/dev/null && {
            echo "✓ Removed route: $network via $gateway"
            ((DELETED++))
        } || true
    fi
done

if [ "$DELETED" -eq 0 ]; then
    echo "✓ No Flannel routes to remove"
fi
'

echo ""
echo "Step 5: Starting k3s service..."
echo "--------------------------------"
remote_exec 'systemctl start k3s'
echo "✓ k3s started"

echo ""
echo "Step 6: Waiting for k3s to stabilize..."
echo "----------------------------------------"
sleep 10

echo ""
echo "Step 7: Verification..."
echo "-----------------------"
remote_exec '
# Check iptables
FLANNEL_RULES=$(iptables-save | grep -i flannel | wc -l)
if [ "$FLANNEL_RULES" -eq 0 ]; then
    echo "✓ No Flannel iptables rules"
else
    echo "⚠ Warning: $FLANNEL_RULES Flannel iptables rules still present"
fi

# Check interfaces
if ! ip link show flannel.1 &>/dev/null; then
    echo "✓ flannel.1 interface removed"
else
    echo "⚠ Warning: flannel.1 interface still exists"
fi

# Check state files
if [ ! -d /var/lib/rancher/k3s/agent/etc/flannel ]; then
    echo "✓ Flannel state directory removed"
else
    echo "⚠ Warning: Flannel state directory still exists"
fi
'

echo ""
echo "========================================"
echo "   Cleanup Complete for $HOST"
echo "========================================"
