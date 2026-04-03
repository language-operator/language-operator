# Quick Start

Deploy an openclaw agent in under 5 minutes.

## Prerequisites

- Language Operator [installed](installation.md)
- Anthropic API key (get one at [console.anthropic.com](https://console.anthropic.com))
- A **default StorageClass** — verify with `kubectl get storageclass` (see [installation requirements](installation.md#requirements))

## Step 1: Create a Cluster

A `LanguageCluster` is a managed namespace for logically grouped agents, models, and tools.

```bash
kubectl apply -f - <<EOF
apiVersion: langop.io/v1alpha1
kind: LanguageCluster
metadata:
  name: language-operator-openclaw
spec:
  domain: openclaw.langop.io
EOF

kubectl wait languagecluster/language-operator-openclaw \
  --for=condition=Ready --timeout=60s

kubectl config set-context --current --namespace=language-operator-openclaw
```

## Step 2: Configure an LLM

Create a secret with your Anthropic API key, then create a `LanguageModel`:

```bash
kubectl create secret generic anthropic-credentials \
  --from-literal=api-key=sk-ant-your-key-here

kubectl apply -f - <<EOF
apiVersion: langop.io/v1alpha1
kind: LanguageModel
metadata:
  name: claude-sonnet
spec:
  provider: anthropic
  modelName: claude-sonnet-4-5
  apiKeySecretRef:
    name: anthropic-credentials
    key: api-key
EOF
```

The operator creates a LiteLLM proxy so all agents can reach the model through a single in-cluster endpoint.

## Step 3: Deploy an Agent

Choose one of the bundled runtimes:

=== "openclaw"

    ```bash
    kubectl apply -f - <<EOF
    apiVersion: langop.io/v1alpha1
    kind: LanguageAgent
    metadata:
      name: openclaw
    spec:
      runtime: openclaw
      openclaw: {}    # token is auto-generated; retrieve it after creation
      models:
        - name: claude-sonnet
    EOF
    ```

=== "opencode"

    ```bash
    kubectl apply -f - <<EOF
    apiVersion: langop.io/v1alpha1
    kind: LanguageAgent
    metadata:
      name: opencode
    spec:
      runtime: opencode
      models:
        - name: claude-sonnet
    EOF
    ```

## Step 4: Check Status

```bash
kubectl get languageagents
kubectl get pods -w
```

## Step 5: Access the Agent

=== "openclaw"

    Retrieve the auto-generated gateway token:

    ```bash
    TOKEN=$(kubectl get secret openclaw-runtime -o jsonpath='{.data.OPENCLAW_GATEWAY_TOKEN}' | base64 -d)
    ```

    If you have a domain configured on your `LanguageCluster`, open `https://openclaw.<cluster-domain>` and enter the token when prompted, or connect the openclaw CLI directly to `wss://openclaw.<cluster-domain>`.

    Otherwise, port-forward for local access:

    ```bash
    kubectl port-forward svc/openclaw 18789:18789
    # connect to ws://localhost:18789 with the token above
    ```

    openclaw uses a WebSocket gateway on port 18789. Connect using the openclaw browser extension or CLI client (see [github.com/openclaw/openclaw](https://github.com/openclaw/openclaw)).

=== "opencode"

    Retrieve the auto-generated credentials:

    ```bash
    USERNAME=$(kubectl get secret opencode-runtime -o jsonpath='{.data.OPENCODE_SERVER_USERNAME}' | base64 -d)
    PASSWORD=$(kubectl get secret opencode-runtime -o jsonpath='{.data.OPENCODE_SERVER_PASSWORD}' | base64 -d)
    echo "username: $USERNAME  password: $PASSWORD"
    ```

    If you have a domain configured on your `LanguageCluster`, open `https://opencode.<cluster-domain>` and sign in with the credentials above.

    To attach the TUI (opencode v1.0.10+):

    ```bash
    opencode attach https://opencode.<cluster-domain> --username "$USERNAME" --password "$PASSWORD"
    ```

    Otherwise, port-forward for local access:

    ```bash
    kubectl port-forward svc/opencode 3000:3000
    # then open http://localhost:3000 or:
    opencode attach http://localhost:3000 --username "$USERNAME" --password "$PASSWORD"
    ```

!!! success "You're Running!"
    You now have openclaw running on Kubernetes with AI capabilities provided through the Language Operator.

## What Just Happened?

The operator automatically:

1. Created a dedicated namespace (`language-operator-openclaw`)
2. Deployed a LiteLLM proxy with your Anthropic credentials
3. Injected the proxy URL into your agent via `MODEL_ENDPOINTS`
4. Created a `{agent}-runtime` Secret from `spec.openclaw.token` and injected it via `envFrom`
5. Created a Deployment, Service, and NetworkPolicy for openclaw

## Next Steps

- [CRD Reference](../api/overview.md) - Complete API documentation
- [Architecture](../architecture/overview.md) - How the operator works
