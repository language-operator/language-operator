# DNS Configuration for Language Operator

This guide explains how to configure DNS for Language Operator agent webhooks.

## Overview

When you set a `domain` on a LanguageCluster, agent webhooks become accessible at:

```
<agent-uuid>.<domain>
```

For example, with `domain: "ai.theryans.io"`, agents get URLs like:
```
f7c3b2a1-9d8e-4c6f-a5b2-c1d4e7f9a3b6.ai.theryans.io
```

The domain serves dual purposes:
- **Base domain**: Future cluster dashboard/UI at `ai.theryans.io`  
- **Agent subdomains**: Individual agents at `<uuid>.ai.theryans.io`

This requires **wildcard DNS** configuration: `*.<domain>` → your cluster ingress.

## Quick Setup

### Local Development with nip.io

For local testing, use nip.io which provides wildcard DNS for any IP:

```yaml
apiVersion: langop.io/v1alpha1
kind: LanguageCluster
metadata:
  name: local-cluster
spec:
  domain: "<YOUR_IP>.nip.io"
```

Example with local IP:
```yaml
spec:
  domain: "192.168.1.100.nip.io"
```

Agent webhooks become: `<uuid>.192.168.1.100.nip.io`

### Production Cloud Setup

Configure wildcard DNS with your cloud provider:

| Provider | DNS Record Type | Name | Value |
|----------|----------------|------|-------|
| **AWS Route53** | CNAME | `*` | `<ingress-hostname>` |
| **Google Cloud DNS** | CNAME | `*` | `<ingress-hostname>` |
| **Azure DNS** | CNAME | `*` | `<ingress-hostname>` |
| **Cloudflare** | CNAME | `*` | `<ingress-hostname>` |

## Detailed Setup Examples

### AWS Route53

1. **Get your ingress hostname:**
   ```bash
   kubectl get ingress -n language-operator-system
   # Or for Gateway API:
   kubectl get gateway -n language-operator-system
   ```

2. **Create wildcard CNAME record:**
   ```bash
   aws route53 change-resource-record-sets \
     --hosted-zone-id Z1234567890 \
     --change-batch '{
       "Changes": [{
         "Action": "CREATE",
         "ResourceRecordSet": {
           "Name": "*.example.com",
           "Type": "CNAME",
           "TTL": 300,
           "ResourceRecords": [{"Value": "k8s-ingress-123.us-west-2.elb.amazonaws.com"}]
         }
       }]
     }'
   ```

### Google Cloud DNS

1. **Create wildcard CNAME record:**
   ```bash
   gcloud dns record-sets create "*.example.com." \
     --zone="example-zone" \
     --type="CNAME" \
     --ttl="300" \
     --rrdatas="ingress.gcp.example.com."
   ```

### Azure DNS

1. **Create wildcard CNAME record:**
   ```bash
   az network dns record-set cname set-record \
     --resource-group myResourceGroup \
     --zone-name example.com \
     --record-set-name "*" \
     --cname "ingress.azure.example.com"
   ```

### Manual DNS Configuration

For self-hosted DNS servers, add:

```zone
; Wildcard CNAME for agent webhooks  
*.example.com.  IN  CNAME  ingress.example.com.
```

## Verification

### Test Wildcard DNS Resolution

```bash
# Test that wildcard DNS resolves
dig test-agent.example.com

# Should return your ingress IP
nslookup random-uuid.example.com
```

### Test Agent Webhook Access

After deploying an agent:

```bash
# Get agent webhook URL
kubectl get languageagent my-agent -o jsonpath='{.status.webhookURL}'

# Test webhook accessibility  
curl -I https://<agent-id>.example.com/webhook
# Should return HTTP 200 or 405 (method not allowed)
```

## TLS Configuration

### Automatic TLS with cert-manager

Language Operator can automatically provision TLS certificates:

```yaml
apiVersion: langop.io/v1alpha1
kind: LanguageCluster
metadata:
  name: secure-cluster
spec:
  domain: "example.com"
  ingressConfig:
    tls:
      enabled: true
      issuerRef:
        name: letsencrypt-prod
        kind: ClusterIssuer
```

This creates certificates for `*.example.com`.

### Manual TLS Certificate

Provide your own wildcard certificate:

```bash
kubectl create secret tls agents-tls \
  --cert=wildcard-cert.pem \
  --key=wildcard-key.pem \
  -n language-operator-system
```

```yaml
spec:
  ingressConfig:
    tls:
      enabled: true
      secretName: agents-tls
```

## Troubleshooting

### Common Issues

**1. DNS Not Propagating**
```bash
# Check DNS propagation
dig @8.8.8.8 test.example.com
dig @1.1.1.1 test.example.com

# Clear local DNS cache
sudo systemctl flush-dns  # Linux
sudo dscacheutil -flushcache  # macOS
```

**2. Ingress Not Receiving Traffic**
```bash
# Verify ingress configuration
kubectl describe ingress -n language-operator-system

# Check ingress logs
kubectl logs -n ingress-nginx deployment/ingress-nginx-controller
```

**3. TLS Certificate Issues**
```bash
# Check certificate status
kubectl describe certificate agents-tls -n language-operator-system

# Verify certificate covers wildcard
openssl x509 -in cert.pem -text -noout | grep DNS
```

### DNS Validation

Language Operator includes optional DNS validation:

```bash
# Check LanguageCluster status
kubectl describe languagecluster my-cluster

# Look for DNS validation conditions:
# - DNSConfigured: True/False
# - WildcardResolvable: True/False
```

If validation fails, check:
1. Wildcard DNS record exists: `*.<domain>`
2. DNS record points to correct ingress
3. TTL has expired (wait for propagation)

## Environment-Specific Notes

### Air-Gapped Environments

DNS validation can be disabled for air-gapped clusters:

```yaml
# In operator deployment
env:
- name: DISABLE_DNS_VALIDATION
  value: "true"
```

### Multiple Clusters

For multiple clusters sharing a domain, use subdomains:

```yaml
# Production cluster
spec:
  domain: "prod.example.com"  # *.prod.example.com

# Staging cluster  
spec:
  domain: "staging.example.com"  # *.staging.example.com
```

### Load Balancer DNS

When using LoadBalancer services, point DNS to the service:

```bash
# Get LoadBalancer IP
kubectl get svc language-operator-gateway -o jsonpath='{.status.loadBalancer.ingress[0].ip}'

# Create A record for wildcard
*.example.com.  IN  A  203.0.113.10
```

## Advanced Configuration

### Future: Custom Agent Subdomain

In future versions, you may be able to customize the agent subdomain pattern:

```yaml
# Future enhancement - not yet implemented
spec:
  domain: "example.com"
  ingressConfig:
    agentSubdomain: "webhooks"  # Would create *.webhooks.example.com
```

Currently, agents use the domain directly: `<uuid>.<domain>`

### Regional DNS

For multi-region deployments:

```yaml
# US cluster
spec:
  domain: "us.example.com"

# EU cluster
spec:
  domain: "eu.example.com"  
```

## Migration

### Changing Domain

To change the domain of an existing cluster:

1. **Update DNS records** for new domain
2. **Update LanguageCluster** spec
3. **Wait for reconciliation** (agents get new URLs)
4. **Verify new URLs** work
5. **Remove old DNS records** after migration

```bash
# Update cluster domain
kubectl patch languagecluster my-cluster --type='merge' -p='{"spec":{"domain":"new-example.com"}}'

# Check agent status for new URLs
kubectl get languageagents -o custom-columns=NAME:.metadata.name,URL:.status.webhookURL
```

### From IP to Domain

Migrating from direct IP access to domain-based:

```yaml
# Before (IP-based)
spec:
  # No domain configured

# After (domain-based)
spec:
  domain: "example.com"
```

Agents automatically get new domain-based URLs during reconciliation.