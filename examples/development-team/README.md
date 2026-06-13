# development-team

Deploys a self-managing AI engineering team into an existing LanguageCluster, pointed at any git repository: one supervisor agent triages open issues into priority queues; three worker agents continuously implement changes, run tests, and open pull requests.

Each agent declares the project repo via `spec.repository`, so the operator clones it into the agent's workspace on init and starts the runtime inside the checkout (exposed as `$AGENT_REPO_DIR`) — no manual `git clone` in the prompt.

```
supervisor (project-manager persona)
  └─ reads open issues → assigns queue/0, queue/1, queue/2 labels

worker-0 (engineer persona)  ← queue/0: urgent (bugs, failing CI, security)
worker-1 (engineer persona)  ← queue/1: normal (features, improvements)
worker-2 (engineer persona)  ← queue/2: backlog (chores, docs, cleanup)
```

The three workers reference the [`context7`](../tools/context7/) `LanguageTool`, giving them
up-to-date, version-specific library documentation over MCP so they stop coding against stale or
hallucinated APIs.

## Prerequisites

- Language Operator [installed](https://langop.io/docs/getting-started/installation/)
- `language-operator-runtimes` chart installed (provides the `claude-code` runtime)
- `kubectl` configured for your cluster
- `envsubst` (`brew install gettext` on macOS, pre-installed on most Linux distros)
- A `LanguageCluster` already applied in the target namespace (see [clusters/basic](../clusters/basic/))
- A Claude account (Pro, Max, Team, or Enterprise)
- A GitHub personal access token with `repo` and `issues` scopes — used both to authenticate the clone (`spec.repository.secretRef`) and for the agents' `gh` operations

## Variables

| Variable | Required | Default | Description |
|----------|----------|---------|-------------|
| `CLUSTER_NAME` | yes | — | Namespace of the target LanguageCluster |
| `GITHUB_TOKEN` | yes | — | GitHub PAT (`repo`, `issues` scopes) — written to `github-credentials/token` secret; used for the clone and for `gh` |
| `PROJECT_REPOSITORY` | yes | — | Clone URL of the project the team will work on (e.g. `https://github.com/language-operator/language-operator.git`). Cloned into each agent's workspace via `spec.repository`. |
| `PROJECT_NAME` | no | repo basename (minus `.git`) | Human-readable project name used in agent prompts (e.g. `Language Operator`) |
| `ANTHROPIC_API_KEY` | no | — | If set, written to `anthropic-credentials/api-key` and injected as `ANTHROPIC_API_KEY` on every agent (API-key billing). |
| `CLAUDE_CODE_OAUTH_TOKEN` | no | — | Long-lived subscription token from `claude setup-token`. Written to `claude-code-oauth/token` and injected as `CLAUDE_CODE_OAUTH_TOKEN` (subscription billing, headless — no `/login` needed). |
| `CONTEXT7_API_KEY` | no | — | [Context7 API key](https://context7.com/dashboard) for higher rate limits. Written to `context7-mcp-credentials/api-key`. The `context7` tool works without it at a lower rate limit. |

If neither `ANTHROPIC_API_KEY` nor `CLAUDE_CODE_OAUTH_TOKEN` is set, agents authenticate interactively via `/login` in each terminal.

## Install

```bash
CLUSTER_NAME=my-cluster \
  GITHUB_TOKEN=ghp_... \
  PROJECT_REPOSITORY=https://github.com/language-operator/language-operator.git \
  PROJECT_NAME="Language Operator" \
  bash examples/development-team/install.sh
```

Dry-run (prints rendered YAML, skips secret creation):
```bash
CLUSTER_NAME=my-cluster \
  GITHUB_TOKEN=ghp_... \
  PROJECT_REPOSITORY=https://github.com/language-operator/language-operator.git \
  bash examples/development-team/install.sh --dry-run
```

## First-time setup

If you provided `ANTHROPIC_API_KEY` or `CLAUDE_CODE_OAUTH_TOKEN`, no per-agent setup is needed — agents authenticate non-interactively. Otherwise, open each agent's terminal and run `/login` inside Claude Code. Credentials are saved to `/workspace/.claude/.credentials.json` and persist across pod restarts.

To mint a long-lived OAuth token (subscription billing, no browser needed): run `claude setup-token` on your laptop and pass the result as `CLAUDE_CODE_OAUTH_TOKEN`.

```bash
kubectl port-forward -n my-cluster svc/supervisor 8080:8080
# open http://localhost:8080, run /login

kubectl port-forward -n my-cluster svc/worker-0 8080:8080
# open http://localhost:8080, run /login
# ... repeat for worker-1, worker-2
```

## What's created

- `Secret/github-credentials` — GitHub PAT
- `Secret/anthropic-credentials` — Anthropic API key (only if `ANTHROPIC_API_KEY` was set)
- `Secret/claude-code-oauth` — Claude Code OAuth token (only if `CLAUDE_CODE_OAUTH_TOKEN` was set)
- `Secret/context7-mcp-credentials` — Context7 API key (only if `CONTEXT7_API_KEY` was set)
- `LanguagePersona/project-manager` — supervisor behavioral config
- `LanguagePersona/engineer` — worker behavioral config
- `LanguageTool/context7` — Context7 MCP tool referenced by the workers; the operator generates its `Deployment` and `Service`
- `LanguageAgent/supervisor` — triages issues; 5Gi workspace
- `LanguageAgent/worker-0` — queue/0 (urgent); 10Gi workspace; uses `context7`
- `LanguageAgent/worker-1` — queue/1 (normal); 10Gi workspace; uses `context7`
- `LanguageAgent/worker-2` — queue/2 (backlog); 10Gi workspace; uses `context7`

Each `LanguageAgent` also produces a `Deployment`, `Service`, `NetworkPolicy`, `PersistentVolumeClaim`, and `ConfigMap`. Because each agent sets `spec.repository`, the operator injects a `repository` init container that clones `PROJECT_REPOSITORY` into the workspace (authenticated with `github-credentials`) and points the runtime's working directory at the checkout via `$AGENT_REPO_DIR`.

## Access

### With OIDC auth (cluster has a domain)

```
https://supervisor.<your-domain>
https://worker-0.<your-domain>
https://worker-1.<your-domain>
https://worker-2.<your-domain>
```

Sign in through the OIDC provider configured on the cluster.

### Local access (port-forward)

```bash
kubectl port-forward -n my-cluster svc/supervisor 8080:8080
# open http://localhost:8080
```

Type a prompt in the terminal to trigger a delegation pass. Use the same pattern for `worker-0`, `worker-1`, or `worker-2`.

## Teardown

```bash
kubectl delete -k examples/development-team/ -n my-cluster
kubectl delete secret github-credentials anthropic-credentials claude-code-oauth context7-mcp-credentials -n my-cluster --ignore-not-found
```
