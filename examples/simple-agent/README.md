# Simple Agent Example

This example demonstrates a complete working deployment of the language-operator with:
- A LanguageCluster (namespace)
- A LanguageTool (web search)
- A LanguageModel (LiteLLM proxy)
- A LanguageAgent (autonomous agent)

## Prerequisites

1. Kubernetes cluster (kind, k3s, or any cluster)
2. language-operator installed
3. OpenAI API key (or other LLM provider)

## Deploy

```bash
# Create secret with API key
kubectl create secret generic openai-key \
  --from-literal=api-key=YOUR_API_KEY_HERE

# Deploy all resources
kubectl apply -f cluster.yaml
kubectl apply -f model.yaml
kubectl apply -f tool.yaml
kubectl apply -f agent.yaml
```

## Check Status

```bash
# Check resources
kubectl get languagecluster,languagemodel,languagetool,languageagent -n demo

# Check agent logs
kubectl logs -n demo -l langop.io/resource=demo-agent -f
```

## Clean Up

```bash
kubectl delete -f agent.yaml
kubectl delete -f tool.yaml
kubectl delete -f model.yaml
kubectl delete -f cluster.yaml
kubectl delete secret openai-key
```
