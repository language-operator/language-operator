# language-operator-team

An example that deploys a self-managing AI engineering team as native Kubernetes workloads using Language Operator.

The team consists of one **supervisor** agent and three **worker** agents. The supervisor triages open GitHub issues into priority queues; each worker continuously drains its assigned queue by implementing changes, running tests, and opening pull requests.

## Architecture

```
supervisor (project-manager persona)
  └─ reads open issues → assigns queue/0, queue/1, queue/2 labels

worker-0 (go-engineer persona)  ← queue/0: urgent (bugs, failing CI, security)
worker-1 (go-engineer persona)  ← queue/1: normal (features, improvements)
worker-2 (go-engineer persona)  ← queue/2: backlog (chores, docs, cleanup)
```

Each agent runs the `claude-code` runtime in a persistent `/workspace` volume, clones the repo on first start, and uses `gh` to interact with GitHub.

## Prerequisites

- Language Operator [installed](https://langop.io/docs/getting-started/installation/)
- `language-operator-runtimes` chart installed (provides the `claude-code` runtime)
- `kubectl` configured for your cluster
- An Anthropic API key **or** Claude OAuth credentials
- A GitHub personal access token with `repo` and `issues` scopes

## Quick Start

### 1. Create secrets

```bash
# Anthropic API key (required unless using OAuth)
kubectl create secret generic anthropic-credentials \
  --from-literal=api-key=sk-ant-... \
  -n language-operator-team

# GitHub token (required)
kubectl create secret generic github-credentials \
  --from-literal=token=ghp_... \
  -n language-operator-team

# Claude OAuth credentials (optional — takes precedence over API key)
kubectl create secret generic claude-oauth-credentials \
  --from-literal=credentials="$(cat ~/.claude/.credentials.json)" \
  -n language-operator-team
```

### 2. Deploy

```bash
kubectl apply -k examples/language-operator-team/
```

### 3. Watch agents start

```bash
kubectl get languageagents -n language-operator-team -w
kubectl get pods -n language-operator-team -w
```

All four agents should reach `Ready=True` within a few minutes.

## Running the Supervisor Loop

The `start` script port-forwards the supervisor's service and sends a delegation pass on a configurable interval:

```bash
# Default: one delegation pass every 120 seconds
./examples/language-operator-team/start

# Custom interval
INTERVAL=300 ./examples/language-operator-team/start
```

The script sends a JSON-RPC `tasks/send` message to the supervisor on each tick and streams the response via SSE. Workers run autonomously — they pick up queue items whenever their `STARTUP_PROMPT` fires.

## Authentication

Two authentication modes are supported. You can provide one or both secrets:

| Mode | Secret | Notes |
|------|--------|-------|
| API key | `anthropic-credentials` | Standard `ANTHROPIC_API_KEY` flow |
| OAuth | `claude-oauth-credentials` | Written to `$HOME/.claude/.credentials.json` at startup; **takes precedence** over the API key when present |

If you only have an API key, skip the `claude-oauth-credentials` secret. If you only have OAuth credentials, you can omit `anthropic-credentials` (though the `LanguageModel` CR still expects the secret — set a placeholder value).

## Agents

| Agent | Persona | Queue | Workspace |
|-------|---------|-------|-----------|
| `supervisor` | `project-manager` | — triages all queues | 5 Gi |
| `worker-0` | `go-engineer` | `queue/0` (urgent) | 10 Gi |
| `worker-1` | `go-engineer` | `queue/1` (normal) | 10 Gi |
| `worker-2` | `go-engineer` | `queue/2` (backlog) | 10 Gi |

## Resources Created

```
LanguageCluster/language-operator-team
LanguageModel/claude-sonnet
LanguagePersona/project-manager
LanguagePersona/go-engineer
LanguageAgent/supervisor
LanguageAgent/worker-0
LanguageAgent/worker-1
LanguageAgent/worker-2
```

Each `LanguageAgent` produces a `Deployment`, `Service`, `NetworkPolicy`, `PersistentVolumeClaim`, and `ConfigMap` in the `language-operator-team` namespace.

## Teardown

```bash
kubectl delete -k examples/language-operator-team/
kubectl delete secret anthropic-credentials github-credentials claude-oauth-credentials \
  -n language-operator-team
```
