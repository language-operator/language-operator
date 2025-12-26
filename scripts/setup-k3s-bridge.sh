#!/bin/bash

# Setup Docker network bridge to K3s cluster for service discovery
set -e

K3S_HOST="dl4"
K3S_DNS_IP=""
BRIDGE_NAME="k3s-bridge"
NETWORK_NAME="langop-k3s"

echo "🔧 Setting up Docker network bridge to K3s cluster..."

# Get K3s cluster DNS IP (usually kube-dns service)
echo "🔍 Discovering K3s DNS server..."
K3S_DNS_IP=$(kubectl get svc -n kube-system kube-dns -o jsonpath='{.spec.clusterIP}' 2>/dev/null || \
             kubectl get svc -n kube-system coredns -o jsonpath='{.spec.clusterIP}' 2>/dev/null || \
             echo "")

if [ -z "$K3S_DNS_IP" ]; then
    echo "❌ Could not discover K3s DNS service IP"
    echo "   Make sure kubectl is configured and K3s is running"
    exit 1
fi

echo "✅ Found K3s DNS at $K3S_DNS_IP"

# Get K3s node IP for routing
K3S_NODE_IP=$(kubectl get nodes -o jsonpath='{.items[0].status.addresses[?(@.type=="InternalIP")].address}' 2>/dev/null || echo "")

if [ -z "$K3S_NODE_IP" ]; then
    echo "❌ Could not discover K3s node IP"
    exit 1
fi

echo "✅ Found K3s node at $K3S_NODE_IP"

# Check if network already exists
if docker network inspect $NETWORK_NAME >/dev/null 2>&1; then
    echo "✅ Docker network '$NETWORK_NAME' already exists"
else
    echo "🌐 Creating Docker network with K3s DNS..."
    
    # Create Docker network (DNS will be configured per container)
    docker network create \
        --driver bridge \
        --opt com.docker.network.bridge.name=$BRIDGE_NAME \
        --subnet="172.20.0.0/16" \
        --gateway="172.20.0.1" \
        $NETWORK_NAME
    
    echo "✅ Created network '$NETWORK_NAME' with DNS $K3S_DNS_IP"
fi

# Add route to K3s service CIDR (typically 10.43.0.0/16 for K3s)
K3S_SERVICE_CIDR=$(kubectl cluster-info dump | grep -oE 'service-cluster-ip-range=[0-9./]+' | cut -d= -f2 | head -1 || echo "10.43.0.0/16")

echo "🛣️  Adding route to K3s services ($K3S_SERVICE_CIDR via $K3S_NODE_IP)..."

# Add route via the K3s node (may need sudo)
if ! ip route show | grep -q "$K3S_SERVICE_CIDR"; then
    echo "   Adding route: $K3S_SERVICE_CIDR via $K3S_NODE_IP"
    sudo ip route add $K3S_SERVICE_CIDR via $K3S_NODE_IP 2>/dev/null || true
fi

# Add route for pod CIDR as well
K3S_POD_CIDR=$(kubectl cluster-info dump | grep -oE 'cluster-cidr=[0-9./]+' | cut -d= -f2 | head -1 || echo "10.42.0.0/16")

if ! ip route show | grep -q "$K3S_POD_CIDR"; then
    echo "   Adding route: $K3S_POD_CIDR via $K3S_NODE_IP"
    sudo ip route add $K3S_POD_CIDR via $K3S_NODE_IP 2>/dev/null || true
fi

echo ""
echo "✅ K3s bridge setup complete!"
echo ""
echo "📋 Network Information:"
echo "   Docker Network: $NETWORK_NAME"
echo "   DNS Server: $K3S_DNS_IP"
echo "   K3s Node: $K3S_NODE_IP"
echo "   Service CIDR: $K3S_SERVICE_CIDR"
echo "   Pod CIDR: $K3S_POD_CIDR"
echo ""
echo "🧪 Test DNS resolution:"
echo "   docker run --rm --network=$NETWORK_NAME alpine nslookup kubernetes.default.svc.cluster.local"
echo ""
echo "🧹 Cleanup:"
echo "   docker network rm $NETWORK_NAME"
echo "   sudo ip route del $K3S_SERVICE_CIDR"
echo "   sudo ip route del $K3S_POD_CIDR"