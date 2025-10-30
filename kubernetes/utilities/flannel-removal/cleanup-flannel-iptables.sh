#!/usr/bin/env bash
set -euo pipefail

# Script to clean up Flannel iptables rules
# Run this on each node to remove leftover Flannel firewall rules

echo "=== Cleaning up Flannel iptables rules ==="

# Save current rules for backup
echo "Backing up current iptables rules..."
iptables-save > /tmp/iptables-backup-$(date +%Y%m%d-%H%M%S).txt
echo "Backup saved to /tmp/iptables-backup-$(date +%Y%m%d-%H%M%S).txt"

# Remove FLANNEL-FWD rules from FORWARD chain
echo "Removing FLANNEL-FWD jump rule from FORWARD chain..."
iptables -D FORWARD -m comment --comment "flanneld forward" -j FLANNEL-FWD 2>/dev/null || echo "Rule not found (may already be removed)"

# Flush and delete FLANNEL-FWD chain
echo "Flushing FLANNEL-FWD chain..."
iptables -F FLANNEL-FWD 2>/dev/null || echo "Chain not found (may already be removed)"
echo "Deleting FLANNEL-FWD chain..."
iptables -X FLANNEL-FWD 2>/dev/null || echo "Chain not found (may already be removed)"

# Remove FLANNEL-POSTRTG rules from POSTROUTING chain
echo "Removing FLANNEL-POSTRTG jump rule from POSTROUTING chain..."
iptables -t nat -D POSTROUTING -m comment --comment "flanneld masq" -j FLANNEL-POSTRTG 2>/dev/null || echo "Rule not found (may already be removed)"

# Flush and delete FLANNEL-POSTRTG chain
echo "Flushing FLANNEL-POSTRTG chain..."
iptables -t nat -F FLANNEL-POSTRTG 2>/dev/null || echo "Chain not found (may already be removed)"
echo "Deleting FLANNEL-POSTRTG chain..."
iptables -t nat -X FLANNEL-POSTRTG 2>/dev/null || echo "Chain not found (may already be removed)"

# Verify cleanup
echo ""
echo "=== Verification ==="
FLANNEL_RULES=$(iptables-save | grep -i flannel | wc -l)
if [ "$FLANNEL_RULES" -eq 0 ]; then
    echo "✓ All Flannel iptables rules have been removed"
else
    echo "⚠ Warning: $FLANNEL_RULES Flannel rules still found:"
    iptables-save | grep -i flannel
fi

echo ""
echo "=== Cleanup complete ==="
