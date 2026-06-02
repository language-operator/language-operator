# clusters/basic

Deploys a `LanguageCluster` with Dex OIDC and a built-in username/password store — the simplest way to put a login wall in front of your agents with no external identity provider.

## Prerequisites

- Language Operator [installed](https://langop.io/docs/getting-started/installation/)
- `kubectl` configured for your cluster
- `envsubst` (`brew install gettext` on macOS, pre-installed on most Linux distros)
- `python3` with the `bcrypt` package (`pip install bcrypt`) to hash `ADMIN_PASSWORD`
- A domain with wildcard DNS pointing at your ingress controller (`*.<domain>`)
- cert-manager (recommended) or manually provisioned TLS certificates

## Variables

| Variable | Required | Default | Description |
|----------|----------|---------|-------------|
| `DOMAIN` | yes | — | Domain for agent ingress and Dex (`auth.<domain>`) |
| `CLUSTER_NAME` | no | `my-cluster` | Name of the LanguageCluster (also the namespace) |
| `ADMIN_EMAIL` | no | `admin@example.com` | Login email for the admin user |
| `ADMIN_USERNAME` | no | `admin` | Display name for the admin user |
| `ADMIN_PASSWORD` | no | `password` | Plain-text password — bcrypt-hashed by install.sh |

**Change `ADMIN_PASSWORD` before deploying to any shared or production cluster.**

> **Escape hatch:** If `python3`/`bcrypt` is unavailable (e.g. in CI), set `ADMIN_PASSWORD_HASH`
> directly with a pre-computed bcrypt hash to skip the hashing step.

## Install

```bash
DOMAIN=example.com ADMIN_EMAIL=me@example.com ADMIN_PASSWORD=mypassword \
  bash examples/clusters/basic/install.sh
```

Dry-run (prints rendered YAML):
```bash
DOMAIN=example.com bash examples/clusters/basic/install.sh --dry-run
```

## What's created

- `LanguageCluster/my-cluster` — creates the namespace, deploys the LiteLLM gateway and Dex

## Teardown

```bash
kubectl delete languagecluster my-cluster
```
