# models/openai-compatible

Registers a generic OpenAI-compatible model as a `LanguageModel` — point it at any server that
speaks the OpenAI API (Ollama, vLLM, LM Studio, llama.cpp, Together, Groq, a self-hosted
gateway, …). Endpoint, model id, and credentials are all parameterized.

This is **registration only**. A `LanguageModel` does not run anything by itself — it teaches the
cluster's LiteLLM gateway how to reach a provider. To actually use it, list it under an agent's
`spec.models` (see [agents/opencode](../../agents/opencode/) for a consumer example).

For the `openai-compatible` provider, `spec.endpoint` is **required** — it's the base URL of the
server (typically ending in `/v1`). An API key is **optional**: omit it for unauthenticated
endpoints; provide one and the installer creates a `generic-model-credentials` secret and wires
it in.

## Prerequisites

- Language Operator [installed](https://langop.io/docs/getting-started/installation/)
- `kubectl` configured for your cluster
- `envsubst` (`brew install gettext` on macOS, pre-installed on most Linux distros)
- A `LanguageCluster` already applied in the target namespace (see [clusters/basic](../../clusters/basic/))
- An OpenAI-compatible endpoint reachable from the cluster

## Variables

| Variable | Required | Default | Description |
|----------|----------|---------|-------------|
| `CLUSTER_NAME` | yes | — | Namespace of the target LanguageCluster |
| `ENDPOINT` | yes | `http://my-model.my-namespace.svc.cluster.local:8000/v1` | Base URL of the OpenAI-compatible server |
| `MODEL_ID` | yes | `my-model` | Model identifier the server expects (e.g. `llama3.2`) |
| `API_KEY` | no | — | API key; when set, a `generic-model-credentials` secret is created and referenced |
| `MODEL_NAME` | no | `generic-model` | Name of the LanguageModel CR |

## Install

```bash
CLUSTER_NAME=my-cluster \
  ENDPOINT=http://ollama.default.svc.cluster.local:11434/v1 \
  MODEL_ID=llama3.2 \
  bash examples/models/openai-compatible/install.sh
```

With an API key (creates the secret):
```bash
CLUSTER_NAME=my-cluster \
  ENDPOINT=https://api.groq.com/openai/v1 \
  MODEL_ID=llama-3.3-70b-versatile \
  API_KEY=gsk_... \
  bash examples/models/openai-compatible/install.sh
```

Dry-run (prints rendered YAML, no key needed):
```bash
CLUSTER_NAME=my-cluster ENDPOINT=http://ollama.default.svc.cluster.local:11434/v1 \
  MODEL_ID=llama3.2 bash examples/models/openai-compatible/install.sh --dry-run
```

## What's created

- `LanguageModel/generic-model` — the OpenAI-compatible model
- `Secret/generic-model-credentials` — **only when `API_KEY` is set**, holds the API key

## Using the model from an agent

```yaml
apiVersion: langop.io/v1alpha1
kind: LanguageAgent
metadata:
  name: my-agent
  namespace: my-cluster
spec:
  models:
    - name: generic-model
```

## Teardown

```bash
kubectl delete languagemodel generic-model -n my-cluster
kubectl delete secret generic-model-credentials -n my-cluster --ignore-not-found
```
