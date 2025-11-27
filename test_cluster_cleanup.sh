#!/bin/bash

# Test script to validate cluster finalizer cleanup functionality
# This tests the implementation for issue #84

set -e

echo "=== Testing LanguageCluster Finalizer Cleanup ==="
echo

# Create test namespace
echo "Creating test namespace..."
kubectl create namespace test-cluster-cleanup || echo "Namespace already exists"

# Create test cluster
echo "Creating test cluster..."
cat <<EOF | kubectl apply -f -
apiVersion: langop.io/v1alpha1
kind: LanguageCluster
metadata:
  name: test-cluster
  namespace: test-cluster-cleanup
spec:
  domain: test.example.com
EOF

# Wait for cluster to be ready
echo "Waiting for cluster to be ready..."
kubectl wait --for=condition=Ready languagecluster/test-cluster -n test-cluster-cleanup --timeout=30s

# Verify finalizer was added
echo "Verifying finalizer was added..."
FINALIZER=$(kubectl get languagecluster test-cluster -n test-cluster-cleanup -o jsonpath='{.metadata.finalizers[0]}')
if [[ "$FINALIZER" == "langop.io/finalizer" ]]; then
    echo "✅ Finalizer correctly added: $FINALIZER"
else
    echo "❌ Finalizer not found or incorrect: $FINALIZER"
    exit 1
fi

# Create dependent agent (using minimal spec to avoid synthesis requirements)
echo "Creating dependent agent..."
cat <<EOF | kubectl apply -f -
apiVersion: langop.io/v1alpha1
kind: LanguageAgent
metadata:
  name: test-agent
  namespace: test-cluster-cleanup
  annotations:
    langop.io/optimized: "true"  # Skip synthesis
spec:
  clusterRef: test-cluster
  instructions: "test agent for cleanup test"
  executionMode: autonomous
EOF

# Create dependent tool
echo "Creating dependent tool..."
cat <<EOF | kubectl apply -f -
apiVersion: langop.io/v1alpha1
kind: LanguageTool
metadata:
  name: test-tool
  namespace: test-cluster-cleanup
spec:
  clusterRef: test-cluster
  type: shell
  config:
    image: alpine:latest
EOF

# Verify dependent resources exist
echo "Verifying dependent resources exist..."
kubectl get languageagent test-agent -n test-cluster-cleanup > /dev/null
kubectl get languagetool test-tool -n test-cluster-cleanup > /dev/null
echo "✅ Dependent resources created successfully"

# Delete the cluster (this should trigger cleanup)
echo "Deleting cluster (should trigger cleanup of dependent resources)..."
kubectl delete languagecluster test-cluster -n test-cluster-cleanup

# Wait a bit for cleanup to process
echo "Waiting for cleanup to complete..."
sleep 5

# Verify dependent resources were deleted
echo "Verifying dependent resources were cleaned up..."

# Check if agent was deleted
if kubectl get languageagent test-agent -n test-cluster-cleanup 2>/dev/null; then
    AGENT_DELETION=$(kubectl get languageagent test-agent -n test-cluster-cleanup -o jsonpath='{.metadata.deletionTimestamp}')
    if [[ -n "$AGENT_DELETION" ]]; then
        echo "✅ Agent marked for deletion: $AGENT_DELETION"
    else
        echo "❌ Agent still exists without deletion timestamp"
        exit 1
    fi
else
    echo "✅ Agent successfully deleted"
fi

# Check if tool was deleted  
if kubectl get languagetool test-tool -n test-cluster-cleanup 2>/dev/null; then
    TOOL_DELETION=$(kubectl get languagetool test-tool -n test-cluster-cleanup -o jsonpath='{.metadata.deletionTimestamp}')
    if [[ -n "$TOOL_DELETION" ]]; then
        echo "✅ Tool marked for deletion: $TOOL_DELETION"
    else
        echo "❌ Tool still exists without deletion timestamp"
        exit 1
    fi
else
    echo "✅ Tool successfully deleted"
fi

# Check if cluster was deleted
if kubectl get languagecluster test-cluster -n test-cluster-cleanup 2>/dev/null; then
    CLUSTER_DELETION=$(kubectl get languagecluster test-cluster -n test-cluster-cleanup -o jsonpath='{.metadata.deletionTimestamp}')
    if [[ -n "$CLUSTER_DELETION" ]]; then
        echo "✅ Cluster marked for deletion: $CLUSTER_DELETION"
        
        # Check if finalizer was removed
        REMAINING_FINALIZERS=$(kubectl get languagecluster test-cluster -n test-cluster-cleanup -o jsonpath='{.metadata.finalizers}')
        if [[ -z "$REMAINING_FINALIZERS" || "$REMAINING_FINALIZERS" == "null" ]]; then
            echo "✅ Finalizer correctly removed after cleanup"
        else
            echo "❌ Finalizer still present: $REMAINING_FINALIZERS"
            exit 1
        fi
    else
        echo "❌ Cluster exists without deletion timestamp"
        exit 1
    fi
else
    echo "✅ Cluster successfully deleted"
fi

# Cleanup test namespace
echo "Cleaning up test namespace..."
kubectl delete namespace test-cluster-cleanup

echo
echo "=== All tests passed! Cluster finalizer cleanup working correctly ==="
echo "✅ Issue #84 resolved: Deleting clusters now properly cleans up orphaned resources"