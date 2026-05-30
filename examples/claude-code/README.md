# claude-code

Deploys a single Claude Code agent as a native Kubernetes workload, protected by a
username/password login wall via Dex and oauth2-proxy.

```
Browser → https://my-agent.<domain>
  → Ingress → oauth2-proxy (validates Dex-issued OIDC token)
    → ttyd (Claude Code terminal)

Browser → https://auth.<domain>
  → Dex OIDC provider (built-in password store)
```

## Prerequisites

- Language Operator [installed](https://langop.io/docs/getting-started/installation/)
- `language-operator-runtimes` chart installed (provides the `claude-code` runtime)
- `kubectl` configured for your cluster
- A domain with wildcard DNS pointing at your ingress controller (`*.<your-domain>`)
- cert-manager (recommended) or manually provisioned TLS certificates

---

## Setup

Start by setting your configuration. Everything below uses these variables:

```bash
CLUSTER_DOMAIN="demo.langop.io"
CLUSTER_ADMIN_EMAIL="user@example.com"

# Default password is "password" — replace the hash for any shared deployment.
# To generate a new hash: python3 -c "import bcrypt; print(bcrypt.hashpw(b'yourpassword', bcrypt.gensalt(10)).decode())"
CLUSTER_ADMIN_HASH='$2a$10$2b2cU8CPhOTaGrs1HRQuAueS7JTT5ZHsHSzYiFPm1leZck7Mc8T4W'
```

### 1. Cluster

Creates the `claude-code` namespace, deploys the shared LiteLLM gateway, and stands up a
Dex OIDC provider at `https://auth.${CLUSTER_DOMAIN}`:

```bash
kubectl apply -f - <<EOF
apiVersion: langop.io/v1alpha1
kind: LanguageCluster
metadata:
  name: claude-code
spec:
  domain: ${CLUSTER_DOMAIN}
  auth:
    enabled: true
    oidc:
      dex:
        enablePasswordDB: true
        staticPasswords:
          - email: ${CLUSTER_ADMIN_EMAIL}
            hash: "${CLUSTER_ADMIN_HASH}"
            username: user
            userID: "08a8684b-db88-4b73-90a9-3cd1661f5466"
EOF
```

### 2. Claude Code

#### 2A. Authentication

No secrets needed. Claude Code will prompt you to run `claude login` the first time you open
the terminal — it handles Anthropic authentication itself.

The Dex login credentials for the web UI are `${CLUSTER_ADMIN_EMAIL}` / `password`.

#### 2B. Agent

Deploys the Claude Code agent behind an oauth2-proxy. The terminal is only reachable at
`https://my-agent.${CLUSTER_DOMAIN}` after signing in through Dex:

```bash
kubectl apply -f - <<EOF
apiVersion: langop.io/v1alpha1
kind: LanguageAgent
metadata:
  name: my-agent
  namespace: claude-code
spec:
  runtime: claude-code
  instructions: |
    You are a general-purpose AI assistant running as a Claude Code agent.
    You have access to a persistent /workspace volume — use it for any files
    you create or clone. You can run shell commands, write and execute code,
    and browse the web via curl.
  networkPolicies:
    egress:
      - to:
          - cidr: "0.0.0.0/0"
        ports:
          - port: 443
            protocol: TCP
          - port: 80
            protocol: TCP
  workspace:
    size: 10Gi
EOF
```

Watch it come up:

```bash
kubectl get languageagents -n claude-code -w
```

---

## Accessing the agent

Open `https://my-agent.${CLUSTER_DOMAIN}` in a browser. Sign in with your Dex credentials
and you land in the Claude Code terminal. Run `claude login` to authenticate with Anthropic.

### Local access (no domain / no ingress)

```bash
kubectl port-forward -n claude-code svc/my-agent 8080:8080
# open http://localhost:8080
```

Port-forwarding bypasses oauth2-proxy and hits ttyd directly.

---

## Teardown

```bash
kubectl delete languageagent my-agent -n claude-code
kubectl delete languagecluster claude-code
```

---

## Going further

### Adding more users

Generate a hash for each user's password, then add entries to `staticPasswords` in the cluster manifest:

```bash
python3 -c "import bcrypt; print(bcrypt.hashpw(b'yourpassword', bcrypt.gensalt(10)).decode())"
```

### Connecting an external identity provider

Replace the password store with a connector — for example, GitHub:

```yaml
spec:
  auth:
    oidc:
      dex:
        enablePasswordDB: false
        connectors:
          - type: github
            id: github
            name: GitHub
            config:
              clientID: "<your-github-client-id>"
              clientSecret: "<your-github-client-secret>"
```

See [dexidp.io/docs/connectors](https://dexidp.io/docs/connectors/) for all supported types.
