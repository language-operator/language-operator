# development-team

Deploys a self-managing AI engineering team: one supervisor agent triages open GitHub issues into priority queues; three worker agents continuously implement changes, run tests, and open pull requests.

```
supervisor (project-manager persona)
  └─ reads open issues → assigns queue/0, queue/1, queue/2 labels

worker-0 (go-engineer persona)  ← queue/0: urgent (bugs, failing CI, security)
worker-1 (go-engineer persona)  ← queue/1: normal (features, improvements)
worker-2 (go-engineer persona)  ← queue/2: backlog (chores, docs, cleanup)
```

## Prerequisites

- Language Operator [installed](https://langop.io/docs/getting-started/installation/)
- `language-operator-runtimes` chart installed (provides the `claude-code` runtime)
- `kubectl` configured for your cluster
- A Claude account (Pro, Max, Team, or Enterprise) — each agent logs in interactively
- A GitHub personal access token with `repo` and `issues` scopes

## Install

### 1. Create the GitHub token secret

```bash
kubectl create secret generic github-credentials \
  --from-literal=token=ghp_... \
  -n development-team
```

### 2. Apply

```bash
kubectl apply -k examples/development-team/
```

### 3. Log in to each agent

Open each agent's terminal and run `/login` inside Claude Code. Sign in with your Claude account through the browser flow. Credentials persist on the agent's workspace PVC across pod restarts.

```bash
kubectl port-forward -n development-team svc/supervisor 8080:8080
# open http://localhost:8080, run /login

kubectl port-forward -n development-team svc/worker-0 8080:8080
# open http://localhost:8080, run /login
# ... repeat for worker-1, worker-2
```

## What's created

- `LanguageCluster/development-team` — namespace, LiteLLM gateway
- `LanguageModel/claude-sonnet` — model name reference (`claude-sonnet-4-6`)
- `LanguagePersona/project-manager` — supervisor behavioral config
- `LanguagePersona/go-engineer` — worker behavioral config
- `LanguageAgent/supervisor` — triages issues; 5Gi workspace
- `LanguageAgent/worker-0` — queue/0 (urgent); 10Gi workspace
- `LanguageAgent/worker-1` — queue/1 (normal); 10Gi workspace
- `LanguageAgent/worker-2` — queue/2 (backlog); 10Gi workspace

Each `LanguageAgent` also produces a `Deployment`, `Service`, `NetworkPolicy`, `PersistentVolumeClaim`, and `ConfigMap`.

## Access

All agents expose an interactive Claude Code terminal on port 8080:

```bash
kubectl port-forward -n development-team svc/supervisor 8080:8080
# open http://localhost:8080
```

Type a prompt in the terminal to trigger a delegation pass. Use the same pattern for `worker-0`, `worker-1`, or `worker-2`.

## Teardown

```bash
kubectl delete -k examples/development-team/
kubectl delete secret github-credentials -n development-team
```
