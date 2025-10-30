#!/usr/bin/env bash
set -euo pipefail

# Script to clean up Flannel network interfaces
# Run this on each node to remove leftover Flannel network devices

echo "=== Cleaning up Flannel network interfaces ==="

# Check if flannel.1 exists
if ip link show flannel.1 &>/dev/null; then
    echo "Found flannel.1 interface, removing..."
    ip link set flannel.1 down 2>/dev/null || true
    ip link delete flannel.1 2>/dev/null || true
    echo "✓ flannel.1 interface removed"
else
    echo "✓ flannel.1 interface not found (already removed)"
fi

# Check if cni0 exists
if ip link show cni0 &>/dev/null; then
    echo "Found cni0 interface..."
    echo "⚠ Warning: cni0 may still be in use by existing pods"
    echo "  To safely remove cni0:"
    echo "  1. Drain the node (kubectl drain <node> --ignore-daemonsets --delete-emptydir-data)"
    echo "  2. Delete the interface: ip link set cni0 down && ip link delete cni0"
    echo "  3. Uncordon the node (kubectl uncordon <node>)"
    echo ""
    echo "  Skipping automatic cni0 removal for safety."
else
    echo "✓ cni0 interface not found"
fi

# List any remaining veth interfaces attached to cni0 (informational)
echo ""
echo "=== Checking for veth interfaces ==="
VETH_COUNT=$(ip link show type veth 2>/dev/null | grep -c "master cni0" || true)
if [ "$VETH_COUNT" -gt 0 ]; then
    echo "⚠ Found $VETH_COUNT veth interfaces still attached to cni0"
    echo "  These will be automatically removed when pods are rescheduled"
else
    echo "✓ No veth interfaces attached to cni0"
fi

echo ""
echo "=== Network interface cleanup complete ==="
