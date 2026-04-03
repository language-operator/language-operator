# Quick Start

Deploy an openclaw agent in under 5 minutes.

## Prerequisites

A cluster with [Language Operator](installation.md) installed and access to a large language model.

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
```

Wait for it to be ready:

```bash
kubectl wait languagecluster/language-operator-openclaw \
  --for=condition=Ready --timeout=60s
```

Switch into its namespace:

```bash
kubectl config set-context --current --namespace=language-operator-openclaw
```

## Step 2: Configure an LLM

Store your API key in a secret:

```bash
kubectl create secret generic anthropic-credentials \
  --from-literal=api-key=sk-ant-your-key-here
```

Create a `LanguageModel` pointing to it:

```bash
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

## Step 3: Deploy an Agent

Choose one of the bundled runtimes:

=== "OpenClaw"

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

=== "OpenCode"

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
```

Watch the agent pod come up:

```bash
kubectl get pods -w
```

## Step 5: Access the Agent

=== "OpenClaw"

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

=== "OpenCode"

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
    ```

    Then open `http://localhost:3000` or attach the TUI:

    ```bash
    opencode attach http://localhost:3000 --username "$USERNAME" --password "$PASSWORD"
    ```

!!! success "You're Running!"
    You now have openclaw running on Kubernetes with AI capabilities provided through the Language Operator.

## What Just Happened?

The operator automatically:

1. Created a dedicated namespace (`language-operator-openclaw`)
2. Deployed a LiteLLM proxy with your model credentials
3. Injected the proxy URL into your agent via `MODEL_ENDPOINTS`
4. Created a `{agent}-runtime` Secret from `spec.openclaw.token` and injected it via `envFrom`
5. Created a Deployment, Service, and secure NetworkPolicy for OpenClaw

## Next Steps

- [CRD Reference](../api/overview.md) - Complete API documentation
- [Architecture](../architecture/overview.md) - How the operator works
