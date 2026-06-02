# clusters/oidc-github

Deploys a `LanguageCluster` with Dex OIDC and a GitHub OAuth connector — agents are protected by a login wall that authenticates users through their GitHub account.

## Prerequisites

- Language Operator [installed](https://langop.io/docs/getting-started/installation/)
- `kubectl` configured for your cluster
- `envsubst` (`brew install gettext` on macOS, pre-installed on most Linux distros)
- A domain with wildcard DNS pointing at your ingress controller (`*.<domain>`)
- cert-manager (recommended) or manually provisioned TLS certificates
- A GitHub OAuth App — create one at <https://github.com/settings/developers>
  - Set the Authorization callback URL to `https://auth.<your-domain>/callback`

## Variables

| Variable | Required | Default | Description |
|----------|----------|---------|-------------|
| `DOMAIN` | yes | — | Domain for agent ingress and Dex (`auth.<domain>`) |
| `GITHUB_CLIENT_ID` | yes | — | GitHub OAuth App client ID |
| `GITHUB_CLIENT_SECRET` | yes | — | GitHub OAuth App client secret |
| `CLUSTER_NAME` | no | `my-cluster` | Name of the LanguageCluster (also the namespace) |

## Install

```bash
DOMAIN=example.com \
  GITHUB_CLIENT_ID=Iv1.abc123 \
  GITHUB_CLIENT_SECRET=your-secret \
  bash examples/clusters/oidc-github/install.sh
```

Dry-run (prints rendered YAML):
```bash
DOMAIN=example.com GITHUB_CLIENT_ID=abc GITHUB_CLIENT_SECRET=xyz \
  bash examples/clusters/oidc-github/install.sh --dry-run
```

## What's created

- `LanguageCluster/my-cluster` — creates the namespace, deploys the LiteLLM gateway and Dex with the GitHub connector

## Teardown

```bash
kubectl delete languagecluster my-cluster
```
