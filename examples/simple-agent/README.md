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

## Quick Start

### Deploy the Example

Use the deployment script to deploy all resources in the correct order with proper waiting:

```bash
# Deploy all resources
./deploy.sh
```

The script will:
1. Create the namespace
2. Deploy LanguageCluster
3. Deploy LanguagePersona
4. Deploy and wait for LanguageModel to be ready
5. Deploy LanguageTool
6. Deploy LanguageAgent and wait for synthesis

### Verify the Deployment

After deployment, verify that all resources are working:

```bash
# Run verification checks
./verify.sh

# Run with verbose output for debugging
./verify.sh --verbose
```

The verification script will check:
1. Namespace exists
2. LanguageCluster is deployed
3. LanguagePersona is deployed
4. LanguageModel is Ready and pod is running
5. LanguageTool is deployed
6. LanguageAgent is Synthesized with deployment and pods ready

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

### Deployment Issues

If deployment fails or resources don't become ready:

```bash
# Check operator logs for synthesis errors
kubectl logs -n kube-system -l app.kubernetes.io/name=language-operator

# Check if model is ready before agent deployment
kubectl get languagemodel magistral-small-2509 -n demo -o jsonpath='{.status.phase}'

# Check agent synthesis status
kubectl get languageagent demo-agent -n demo -o jsonpath='{.status.conditions[?(@.type=="Synthesized")]}'
```

**Tip**: Always ensure models are Ready before deploying agents. The deploy.sh script handles this automatically with proper waiting.

### Agent Crash Loops

If the agent pod is restarting frequently, check for permission errors:

```bash
# Check agent logs for errors
kubectl logs -n demo -l app=demo-agent -c agent --tail=50

# Common issue: workspace permission denied
# The agent will now log output instead of crashing
```

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
