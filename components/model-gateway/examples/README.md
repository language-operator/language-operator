# LanguageModel Examples

This directory contains example LanguageModel configurations for different providers and network scenarios.

## Overview

These examples demonstrate how to configure LanguageModel resources with appropriate network egress rules for Cilium NetworkPolicy enforcement. Each example includes:

- Provider-specific configuration
- Rate limiting and retry policies
- Observability settings
- **Network egress rules** for external API access

## Quick Reference

| Provider | File | Egress Pattern | Description |
|----------|------|----------------|-------------|
| OpenAI | `openai-model.yaml` | `api.openai.com:443` | Standard OpenAI API |
| Azure OpenAI | `azure-openai.yaml` | `*.openai.azure.com:443` | Azure-hosted OpenAI |
| Anthropic | `anthropic-model.yaml` | `api.anthropic.com:443` | Claude via Anthropic API |
| AWS Bedrock | `aws-bedrock.yaml` | `bedrock-runtime.*.amazonaws.com:443` | AWS Bedrock service |
| Google Vertex | `google-vertex.yaml` | `generativelanguage.googleapis.com:443` | Google Vertex AI |
| Local Ollama | `ollama-local.yaml` | `192.168.1.0/24:11434` | Local network Ollama |
| LM Studio | `lm-studio.yaml` | `192.168.1.0/24:1234` | Local LM Studio server |
| Corporate Proxy | `corporate-proxy.yaml` | `10.0.0.0/8:8080` | Via corporate proxy |
| Load Balanced | `multi-endpoint-loadbalanced.yaml` | Multiple endpoints | Multi-region setup |

## Network Egress Patterns

### Cloud Providers

**OpenAI:**
```yaml
egress:
- description: "Allow OpenAI API access"
  to:
    dns: ["api.openai.com"]
  ports:
  - port: 443
    protocol: TCP
```

**Azure OpenAI:**
```yaml
egress:
- description: "Allow Azure OpenAI API access"
  to:
    dns: ["*.openai.azure.com"]
  ports:
  - port: 443
    protocol: TCP
```

**AWS Bedrock:**
```yaml
egress:
- description: "Allow AWS Bedrock API access"
  to:
    dns: ["bedrock-runtime.*.amazonaws.com"]
  ports:
  - port: 443
    protocol: TCP
- description: "Allow AWS STS for credential refresh"
  to:
    dns: ["sts.*.amazonaws.com"]
  ports:
  - port: 443
    protocol: TCP
```

**Google Vertex AI:**
```yaml
egress:
- description: "Allow Google Vertex AI API access"
  to:
    dns: ["generativelanguage.googleapis.com", "aiplatform.googleapis.com"]
  ports:
  - port: 443
    protocol: TCP
- description: "Allow Google OAuth for authentication"
  to:
    dns: ["oauth2.googleapis.com", "accounts.google.com"]
  ports:
  - port: 443
    protocol: TCP
```

### Local and Self-Hosted

**Local Network (Ollama/LM Studio):**
```yaml
egress:
- description: "Allow local model server"
  to:
    cidr: "192.168.1.0/24"  # Adjust to your network
  ports:
  - port: 11434  # Ollama default port
    protocol: TCP
```

**Corporate Proxy:**
```yaml
egress:
- description: "Allow corporate proxy access"
  to:
    cidr: "10.0.0.0/8"  # Corporate network range
  ports:
  - port: 8080
    protocol: TCP
```

## Usage Instructions

1. **Choose the appropriate example** for your model provider
2. **Customize the network ranges** in egress rules to match your environment
3. **Create required secrets** for API keys:
   ```bash
   kubectl create secret generic openai-credentials \
     --from-literal=api-key=your-api-key \
     --namespace langop-system
   ```
4. **Apply the LanguageModel**:
   ```bash
   kubectl apply -f openai-model.yaml
   ```
5. **Verify network connectivity**:
   ```bash
   kubectl logs -n langop-system deployment/language-operator
   ```

## NetworkPolicy Requirements

These examples require a CNI that enforces NetworkPolicy:

- ✅ **Cilium** (recommended)
- ✅ **Calico**
- ✅ **Weave Net**
- ✅ **Antrea**
- ❌ **Flannel** (does not enforce NetworkPolicy)

To check if NetworkPolicy is enforced:
```bash
kubectl get networkpolicies -A
```

## Security Best Practices

1. **Use specific DNS names** instead of wildcards when possible
2. **Restrict to HTTPS (port 443)** for external APIs
3. **Use CIDR ranges** for local networks, not `0.0.0.0/0`
4. **Test egress rules** after deployment:
   ```bash
   kubectl logs -f deployment/language-operator -n langop-system
   ```
5. **Monitor for connection failures** in model status

## Troubleshooting

### Model Shows "NetworkPolicy" Error

Check the model status:
```bash
kubectl describe languagemodel your-model-name
```

Common issues:
- Egress rule doesn't match the actual endpoint
- CNI doesn't enforce NetworkPolicy
- Incorrect CIDR range for local networks

### Connection Timeouts

1. Verify egress rules match your network:
   ```bash
   kubectl get networkpolicy -A -o yaml
   ```
2. Check DNS resolution:
   ```bash
   kubectl run debug --rm -it --image=nicolaka/netshoot -- nslookup api.openai.com
   ```
3. Test connectivity:
   ```bash
   kubectl run debug --rm -it --image=nicolaka/netshoot -- curl -v https://api.openai.com
   ```

### Corporate Proxy Issues

1. Ensure proxy settings in model configuration
2. Add egress rules for proxy server
3. Include DNS servers in egress rules
4. Test proxy connectivity from cluster

## Related Documentation

- [Security Documentation](../../../docs/security/README.md)
- [CNI Requirements](../../../docs/security/cni-requirements.md)
- [NetworkPolicy Guide](../../../docs/security/operator-networkpolicy.md)
- [LanguageModel API Reference](../../../src/docs/api-reference.md)