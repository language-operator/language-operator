# Cilium CNI Configuration

This directory contains Cilium CNI configuration for the language-operator Kubernetes cluster.

## Quick Start

```bash
# Install Cilium
make install

# Upgrade after editing values.yaml
make upgrade

# Test configuration before applying
make dry-run

# Check health
make check-health
```

## Available Commands

### Installation

| Command | Description |
|---------|-------------|
| `make install` | Install Cilium with current values.yaml |
| `make upgrade` | Upgrade Cilium with current values.yaml |
| `make uninstall` | Uninstall Cilium (prompts for confirmation) |

### Testing & Validation

| Command | Description |
|---------|-------------|
| `make dry-run` | Show what would be applied without installing |
| `make template` | Generate Kubernetes manifests from Helm chart |
| `make validate` | Validate values.yaml syntax |
| `make diff` | Show diff between current and new config (requires helm-diff plugin) |

### Status & Monitoring

| Command | Description |
|---------|-------------|
| `make status` | Show Cilium installation status |
| `make check-health` | Check Cilium health status |
| `make logs` | Follow Cilium agent logs |
| `make operator-logs` | Follow Cilium operator logs |
| `make watch` | Watch Cilium pods status |
| `make version` | Show installed and available versions |

### Maintenance

| Command | Description |
|---------|-------------|
| `make restart` | Restart Cilium pods |
| `make config` | Show current Cilium ConfigMap |
| `make connectivity-test` | Run Cilium connectivity test |
| `make connectivity-cleanup` | Clean up connectivity test |

### Debugging

| Command | Description |
|---------|-------------|
| `make debug-endpoints` | Show Cilium endpoints |
| `make debug-policy` | Show Cilium network policies |
| `make debug-service-list` | Show Cilium service list |
| `make debug-status` | Show detailed Cilium status |
| `make debug-bpf` | Show BPF maps |

### Helpers

| Command | Description |
|---------|-------------|
| `make backup` | Backup current values.yaml |
| `make restore-backup` | Restore values.yaml from latest backup |
| `make list-backups` | List available backups |
| `make clean-backups` | Remove old backups (keep last 5) |
| `make help` | Show all available commands |

## Typical Workflow

### Making Configuration Changes

1. **Backup current configuration**:
   ```bash
   make backup
   ```

2. **Edit values.yaml**:
   ```bash
   vim values.yaml
   # or
   code values.yaml
   ```

3. **Test changes**:
   ```bash
   make dry-run
   # or
   make validate
   ```

4. **Apply changes**:
   ```bash
   make upgrade
   ```

5. **Verify**:
   ```bash
   make status
   make check-health
   ```

### Troubleshooting

1. **Check Cilium status**:
   ```bash
   make status
   make debug-status
   ```

2. **View logs**:
   ```bash
   make logs
   # or for operator
   make operator-logs
   ```

3. **Check connectivity**:
   ```bash
   make connectivity-test
   # Wait for test to complete, then:
   kubectl get pods -n cilium-test
   # Clean up when done:
   make connectivity-cleanup
   ```

4. **Restart if needed**:
   ```bash
   make restart
   ```

## Configuration

### Environment Variables

You can override default values using environment variables:

```bash
# Use a different Cilium version
CILIUM_VERSION=1.16.5 make install

# Use a different values file
VALUES_FILE=values-prod.yaml make dry-run

# Use a different namespace
CILIUM_NAMESPACE=cilium-system make status
```

### Current Configuration

This Cilium installation is configured with:

- **Native routing mode** - Compatible with NixOS kernel
- **Kubernetes IPAM** - Uses pod CIDR `10.42.0.0/16`
- **kube-proxy replacement** - Full eBPF datapath
- **Network policy enforcement** - Default mode
- **DNS-based egress policies** - For language-operator NetworkPolicies

See [values.yaml](./values.yaml) for complete configuration.

## Integration with language-operator

The language-operator uses Cilium's NetworkPolicy features for:

- **DNS-based egress control** - LanguageModel and LanguageTool resources specify allowed domains
- **Automatic DNS resolution** - DNS names are resolved to IPs by Cilium
- **Policy isolation** - Each LanguageCluster gets isolated network policies

### Verifying DNS Policies

```bash
# Check DNS policies are loaded
make debug-policy | grep -i dns

# Check service list includes DNS
make debug-service-list | grep -i dns
```

## Troubleshooting

### Common Issues

**Cilium pods not starting**:
```bash
make logs
# Check for errors in output
```

**Network connectivity issues**:
```bash
make connectivity-test
# Wait and check results
```

**DNS not resolving**:
```bash
make debug-endpoints
# Verify DNS endpoints are present
```

**Configuration not applying**:
```bash
make diff
# See what would change
make validate
# Verify syntax
```

### Rolling Back

If an upgrade causes issues:

```bash
# Restore previous configuration
make restore-backup

# Downgrade to previous version
CILIUM_VERSION=1.15.0 make upgrade
```

## Advanced Usage

### Using helm-diff Plugin

For better visibility into what changes will be applied:

```bash
# Install helm-diff plugin
helm plugin install https://github.com/databus23/helm-diff

# Show detailed diff
make diff
```

### Custom Values

Create environment-specific values files:

```bash
# Create production values
cp values.yaml values-prod.yaml
# Edit as needed

# Use custom values
VALUES_FILE=values-prod.yaml make dry-run
VALUES_FILE=values-prod.yaml make upgrade
```

### Monitoring

For continuous monitoring:

```bash
# Watch pod status
make watch

# In another terminal, follow logs
make logs
```

## References

- [Cilium Documentation](https://docs.cilium.io/)
- [Cilium k3s Guide](https://docs.cilium.io/en/latest/installation/k3s/)
- [Cilium Network Policies](https://docs.cilium.io/en/latest/security/policy/)
- [Cilium DNS-based Policies](https://docs.cilium.io/en/latest/security/policy/language/#dns-based)
