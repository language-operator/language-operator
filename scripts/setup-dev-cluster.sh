#!/bin/bash

# Setup script for development Kind cluster
set -e

echo "Setting up Language Operator development cluster..."

# Wait for Kind cluster to be ready
echo "Waiting for Kind cluster to be ready..."
timeout 300 sh -c 'until kubectl get nodes --kubeconfig /etc/kubernetes/admin.conf; do sleep 5; done'

# Create language-operator namespace
kubectl create namespace language-operator --kubeconfig /etc/kubernetes/admin.conf --dry-run=client -o yaml | kubectl apply -f - --kubeconfig /etc/kubernetes/admin.conf

# Apply CRDs
echo "Installing Language Operator CRDs..."
kubectl apply -f /operator-config/crd/bases/ --kubeconfig /etc/kubernetes/admin.conf

# Create RBAC resources
echo "Setting up RBAC..."
kubectl apply -f /operator-config/rbac/ --kubeconfig /etc/kubernetes/admin.conf

echo "✓ Kind cluster configured for Language Operator development"
echo "✓ CRDs installed"
echo "✓ RBAC configured"
echo ""
echo "Cluster is ready for development!"