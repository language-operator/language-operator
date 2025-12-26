#!/bin/bash

# Setup Kind cluster for Language Operator development
set -e

CLUSTER_NAME="language-operator-dev"

echo "🔧 Setting up Kind cluster for Language Operator development..."

# Check if kind is installed
if ! command -v kind &> /dev/null; then
    echo "❌ Kind not found. Please install Kind first:"
    echo "   https://kind.sigs.k8s.io/docs/user/quick-start/#installation"
    exit 1
fi

# Check if kubectl is installed
if ! command -v kubectl &> /dev/null; then
    echo "❌ kubectl not found. Please install kubectl first"
    exit 1
fi

# Check if cluster already exists
if kind get clusters | grep -q "^${CLUSTER_NAME}$"; then
    echo "✅ Kind cluster '${CLUSTER_NAME}' already exists"
    echo "   Use 'kind delete cluster --name ${CLUSTER_NAME}' to recreate"
else
    echo "🚀 Creating Kind cluster '${CLUSTER_NAME}'..."
    
    # Create kind cluster with port mappings for ingress
    cat <<EOF | kind create cluster --name=${CLUSTER_NAME} --config=-
kind: Cluster
apiVersion: kind.x-k8s.io/v1alpha4
nodes:
- role: control-plane
  kubeadmConfigPatches:
  - |
    kind: InitConfiguration
    nodeRegistration:
      kubeletExtraArgs:
        node-labels: "ingress-ready=true"
  extraPortMappings:
  - containerPort: 80
    hostPort: 30080
    protocol: TCP
  - containerPort: 443
    hostPort: 30443
    protocol: TCP
EOF
fi

# Switch to the kind cluster context
echo "🔧 Setting kubectl context to kind-${CLUSTER_NAME}..."
kubectl cluster-info --context=kind-${CLUSTER_NAME}

# Create namespace
echo "📦 Creating language-operator namespace..."
kubectl create namespace language-operator --dry-run=client -o yaml | kubectl apply -f -

# Install CRDs if they exist
if [ -d "src/config/crd/bases" ]; then
    echo "📋 Installing Language Operator CRDs..."
    kubectl apply -f src/config/crd/bases/
else
    echo "⚠️  CRDs not found - run 'make -C src manifests' first if needed"
fi

# Install RBAC if it exists  
if [ -d "src/config/rbac" ]; then
    echo "🔐 Installing RBAC configuration..."
    kubectl apply -f src/config/rbac/
else
    echo "⚠️  RBAC config not found - run 'make -C src manifests' first if needed"
fi

# Deploy the operator using Helm (optional)
if [ -d "chart" ] && [ -f "chart/Chart.yaml" ]; then
    echo "🎯 Deploying Language Operator via Helm..."
    helm upgrade --install language-operator ./chart \
        --namespace language-operator \
        --create-namespace \
        --wait \
        --timeout 300s \
        --set image.pullPolicy=Never
else
    echo "ℹ️  Helm chart not found - operator deployment skipped"
    echo "   Deploy manually or use helm to install"
fi

echo ""
echo "✅ Development environment ready!"
echo ""
echo "📋 Cluster Information:"
echo "   Cluster Name: ${CLUSTER_NAME}"
echo "   Context: kind-${CLUSTER_NAME}"
echo "   API Server: https://localhost:6443"
echo ""
echo "🚀 Next Steps:"
echo "   1. Start dashboard: make dev-up"
echo "   2. Deploy test resources: kubectl apply -f examples/"
echo "   3. Watch operator logs: kubectl logs -f -n language-operator deployment/language-operator"
echo ""
echo "🧹 Cleanup:"
echo "   kind delete cluster --name ${CLUSTER_NAME}"