#!/usr/bin/env bash
set -euo pipefail

# Script to remove Flannel from all cluster nodes
# Runs cleanup on each node sequentially

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
CLEANUP_SCRIPT="$SCRIPT_DIR/cleanup-flannel-ssh.sh"

# List of nodes
NODES=("dl1" "dl2" "dl3" "dl4" "dl5")

echo "========================================"
echo " Cluster-Wide Flannel Cleanup"
echo "========================================"
echo "Target nodes: ${NODES[*]}"
echo ""
read -p "Proceed with cleanup on all nodes? (yes/no): " confirm

if [ "$confirm" != "yes" ]; then
    echo "Aborted."
    exit 0
fi

echo ""

SUCCESS=0
FAILED=0

for node in "${NODES[@]}"; do
    echo ""
    echo "###############################################################################"
    echo "# Processing: $node"
    echo "###############################################################################"
    echo ""
    
    if "$CLEANUP_SCRIPT" "$node"; then
        echo "✓ Successfully cleaned up $node"
        ((SUCCESS++))
    else
        echo "✗ Failed to clean up $node"
        ((FAILED++))
    fi
    
    echo ""
    echo "Waiting 5 seconds before next node..."
    sleep 5
done

echo ""
echo "========================================"
echo " Cleanup Summary"
echo "========================================"
echo "Success: $SUCCESS/$((SUCCESS + FAILED)) nodes"
echo "Failed:  $FAILED/$((SUCCESS + FAILED)) nodes"
echo ""

if [ "$FAILED" -eq 0 ]; then
    echo "✓ All nodes cleaned successfully!"
    echo ""
    echo "Next steps:"
    echo "1. Verify Cilium is healthy: kubectl get pods -n kube-system -l k8s-app=cilium"
    echo "2. Test connectivity: cilium connectivity test"
    echo "3. Check for any remaining Flannel artifacts"
else
    echo "⚠ Some nodes failed. Please check the output above."
fi
