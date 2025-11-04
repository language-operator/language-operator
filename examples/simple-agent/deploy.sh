#!/bin/bash
# Deploy simple-agent example with proper resource ordering
set -e

NAMESPACE="demo"
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

echo "🚀 Deploying simple-agent to namespace: $NAMESPACE"
echo ""

# Step 1: Create namespace if it doesn't exist
echo "📦 Step 1/6: Ensuring namespace exists..."
kubectl create namespace $NAMESPACE --dry-run=client -o yaml | kubectl apply -f -
echo "✅ Namespace ready"
echo ""

# Step 2: Deploy LanguageCluster (creates default configurations)
echo "🔧 Step 2/6: Deploying LanguageCluster..."
kubectl apply -f "$SCRIPT_DIR/cluster.yaml"
echo "✅ LanguageCluster deployed"
echo ""

# Step 3: Deploy Persona
echo "👤 Step 3/6: Deploying Persona..."
kubectl apply -f "$SCRIPT_DIR/persona.yaml"
echo "✅ Persona deployed"
echo ""

# Step 4: Deploy Model and wait for it to be ready
echo "🤖 Step 4/6: Deploying LanguageModel..."
kubectl apply -f "$SCRIPT_DIR/model.yaml"

echo "⏳ Waiting for model deployment to be ready..."
if kubectl wait --for=jsonpath='{.status.phase}'=Ready \
    languagemodel/magistral-small-2509 \
    -n $NAMESPACE \
    --timeout=60s 2>/dev/null; then
    echo "✅ Model is Ready"
else
    echo "⚠️  Model not ready after 60s, checking status..."
    kubectl get languagemodel magistral-small-2509 -n $NAMESPACE -o jsonpath='{.status.phase}'
    echo ""
fi

echo "⏳ Waiting for model pod to be running..."
kubectl wait --for=condition=ready pod \
    -l app=magistral-small-2509 \
    -n $NAMESPACE \
    --timeout=120s || echo "⚠️  Model pod not ready (may still be starting)"
echo ""

# Step 5: Deploy Tool
echo "🔨 Step 5/6: Deploying LanguageTool..."
kubectl apply -f "$SCRIPT_DIR/tool.yaml"
echo "✅ Tool deployed"
echo ""

# Step 6: Deploy Agent
echo "🤖 Step 6/6: Deploying LanguageAgent..."
kubectl apply -f "$SCRIPT_DIR/agent.yaml"

echo "⏳ Waiting for agent to synthesize..."
sleep 5

# Check synthesis status
if kubectl wait --for=jsonpath='{.status.conditions[?(@.type=="Synthesized")].status}'=True \
    languageagent/demo-agent \
    -n $NAMESPACE \
    --timeout=30s 2>/dev/null; then
    echo "✅ Agent code synthesized"
else
    echo "⚠️  Synthesis status unknown, checking conditions..."
    kubectl get languageagent demo-agent -n $NAMESPACE -o jsonpath='{.status.conditions}'
    echo ""
fi

echo ""
echo "✨ Deployment complete!"
echo ""
echo "📊 Current status:"
kubectl get languageagent,languagemodel,languagetool,languagepersona,languagecluster -n $NAMESPACE
echo ""
echo "🔍 To verify deployment, run: ./verify.sh"
echo "📋 To view agent logs, run: kubectl logs -f -n $NAMESPACE -l app=demo-agent -c agent"
