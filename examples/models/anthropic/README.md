# models/anthropic

Registers the two state-of-the-art Anthropic models — Claude Opus and Claude Sonnet — as
`LanguageModel` resources sharing a single `anthropic-credentials` secret. A good default model
catalog for a cluster: agents pick whichever they need.

These are **registration only**. A `LanguageModel` does not run anything by itself — it teaches
the cluster's LiteLLM gateway how to reach a provider. To actually use one, list it under an
agent's `spec.models` (see [agents/opencode](../../agents/opencode/) for a consumer example).

## Prerequisites

- Language Operator [installed](https://langop.io/docs/getting-started/installation/)
- `kubectl` configured for your cluster
- `envsubst` (`brew install gettext` on macOS, pre-installed on most Linux distros)
- A `LanguageCluster` already applied in the target namespace (see [clusters/basic](../../clusters/basic/))
- An Anthropic API key

## Variables

| Variable | Required | Default | Description |
|----------|----------|---------|-------------|
| `CLUSTER_NAME` | yes | — | Namespace of the target LanguageCluster |
| `ANTHROPIC_API_KEY` | yes | — | Anthropic API key (used to create the `anthropic-credentials` secret) |
| `OPUS_NAME` | no | `claude-opus` | Name of the Opus LanguageModel CR |
| `OPUS_MODEL_ID` | no | `claude-opus-4-8` | Provider model identifier for Opus |
| `SONNET_NAME` | no | `claude-sonnet` | Name of the Sonnet LanguageModel CR |
| `SONNET_MODEL_ID` | no | `claude-sonnet-4-6` | Provider model identifier for Sonnet |

## Install

```bash
CLUSTER_NAME=my-cluster ANTHROPIC_API_KEY=sk-ant-... bash examples/models/anthropic/install.sh
```

Dry-run (prints rendered YAML, no API key needed):
```bash
CLUSTER_NAME=my-cluster bash examples/models/anthropic/install.sh --dry-run
```

## What's created

- `Secret/anthropic-credentials` — holds the Anthropic API key, shared by both models
- `LanguageModel/claude-opus` — Claude Opus (`claude-opus-4-8`)
- `LanguageModel/claude-sonnet` — Claude Sonnet (`claude-sonnet-4-6`)

## Using a model from an agent

```yaml
apiVersion: langop.io/v1alpha1
kind: LanguageAgent
metadata:
  name: my-agent
  namespace: my-cluster
spec:
  models:
    - name: claude-opus
```

## Teardown

```bash
kubectl delete languagemodel claude-opus claude-sonnet -n my-cluster
kubectl delete secret anthropic-credentials -n my-cluster
```
