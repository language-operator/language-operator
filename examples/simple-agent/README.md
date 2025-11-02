# Simple Agent Example

This example demonstrates a complete working deployment of the language-operator with:
- A LanguageCluster (namespace)
- A LanguageTool (web search tool with sidecar deployment)
- A LanguageModel (LiteLLM proxy for local LLM)
- A LanguageAgent (autonomous agent with workspace)

## Prerequisites

1. Kubernetes cluster (kind, k3s, or any cluster)
2. language-operator installed
3. Local LLM running (e.g., LM Studio) or OpenAI API key

## Automated Verification

The easiest way to test this example is using the automated verification script:

```bash
# Run full end-to-end verification
./verify.sh

# Run with verbose output for debugging
./verify.sh --verbose

# Continue on errors (don't exit on first failure)
./verify.sh --no-fail-fast

# Custom timeout (default: 300 seconds)
./verify.sh --timeout 600

# Clean up after testing
./verify.sh --cleanup-only
```

The verification script will:
1. Check prerequisites (kubectl, operator running)
2. Clean up any existing resources
3. Apply all manifests in the correct order
4. Wait for resources to become ready
5. Verify pod status, sidecar injection, and workspace PVC
6. Check environment variables and configurations
7. Examine logs for errors
8. Print a detailed summary

## Manual Deployment

If you prefer to deploy manually:

```bash
# Deploy resources in order
kubectl apply -f cluster.yaml  # Creates namespace
kubectl apply -f model.yaml    # Creates model proxy
kubectl apply -f tool.yaml     # Creates tool (sidecar mode)
kubectl apply -f agent.yaml    # Creates agent with sidecar and workspace
```

## Check Status

```bash
# Check all custom resources
kubectl get languagecluster,languagemodel,languagetool,languageagent -n demo

# Check pods and services
kubectl get pods,pvc,svc -n demo

# Check agent logs (agent container)
kubectl logs -n demo -l langop.io/agent=demo-agent -c agent -f

# Check tool sidecar logs
kubectl logs -n demo -l langop.io/agent=demo-agent -c tool-web-tool -f

# Check model proxy logs
kubectl logs -n demo -l langop.io/model=magistral-small-2509 -f
```

## Verify Sidecar Injection

The agent should have 2 containers (agent + tool sidecar):

```bash
# Check container count
kubectl get pods -n demo -l langop.io/agent=demo-agent -o jsonpath='{.items[0].spec.containers[*].name}'
# Expected output: agent tool-web-tool

# Verify environment variables
kubectl get pods -n demo -l langop.io/agent=demo-agent -o json | grep -A 5 MODEL_ENDPOINTS
kubectl get pods -n demo -l langop.io/agent=demo-agent -o json | grep -A 5 MCP_SERVERS
```

## Verify Workspace

The agent should have a workspace PVC mounted:

```bash
# Check PVC
kubectl get pvc demo-agent-workspace -n demo

# Check mount in pod
kubectl get pods -n demo -l langop.io/agent=demo-agent -o jsonpath='{.items[0].spec.containers[0].volumeMounts[*].mountPath}'
# Should include /workspace
```

## Clean Up

```bash
# Using the verification script
./verify.sh --cleanup-only

# Or manually
kubectl delete languageagent demo-agent -n demo
kubectl delete languagetool web-tool -n demo
kubectl delete languagemodel magistral-small-2509 -n demo
kubectl delete languagecluster demo-cluster -n demo
```

## Troubleshooting

### ImagePullBackOff Errors

If pods can't pull images from private registry:

```bash
# Create registry credentials secret
kubectl create secret docker-registry registry-credentials \
  --docker-server=git.theryans.io \
  --docker-username=ci \
  --docker-password=YOUR_PASSWORD \
  -n demo

# Patch default ServiceAccount
kubectl patch serviceaccount default -n demo \
  -p '{"imagePullSecrets": [{"name": "registry-credentials"}]}'
```

### Pod Not Starting

Check pod events and logs:

```bash
kubectl describe pod -n demo -l langop.io/agent=demo-agent
kubectl logs -n demo -l langop.io/agent=demo-agent --all-containers --tail=100
```

### Model Not Ready

Check if the LiteLLM proxy can reach your local LLM:

```bash
# Check model pod logs
kubectl logs -n demo -l langop.io/model=magistral-small-2509

# Test connectivity from within cluster
kubectl run -it --rm debug --image=curlimages/curl --restart=Never -- \
  curl http://magistral-small-2509.demo.svc.cluster.local:4000/health
```
