#!/usr/bin/env bash
set -euo pipefail

# Script to clean up Flannel routing table entries
# Run this on each node to remove leftover Flannel routes

echo "=== Cleaning up Flannel routes ==="

# Get the pod CIDR range (typically 10.42.0.0/16 for k3s with Flannel)
POD_CIDR="10.42.0.0/16"

echo "Looking for Flannel routes (routes via physical interfaces for $POD_CIDR)..."
echo ""

# Find routes that go through physical interfaces (not cilium_host or flannel.1)
# These are the inter-node routes that Flannel created
ROUTES_TO_DELETE=$(ip route show | grep "^10.42" | grep -v "cilium_host" | grep -v "flannel.1" | grep "via.*proto kernel" || true)

if [ -z "$ROUTES_TO_DELETE" ]; then
    echo "✓ No Flannel routes found"
else
    echo "Found Flannel routes to remove:"
    echo "$ROUTES_TO_DELETE"
    echo ""
    
    # Delete each route
    while IFS= read -r route; do
        if [ -n "$route" ]; then
            echo "Removing: $route"
            ip route del $route 2>/dev/null || echo "  Failed to remove (may already be gone)"
        fi
    done <<< "$ROUTES_TO_DELETE"
fi

echo ""
echo "=== Current pod network routes ==="
ip route show | grep "^10.42" || echo "No pod network routes found"

echo ""
echo "=== Route cleanup complete ==="
echo "Note: Cilium will automatically recreate correct routes for pod networking"
