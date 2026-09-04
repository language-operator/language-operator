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
- the [`argo` CLI](https://github.com/argoproj/argo-workflows/releases) — these agents are scheduled tasks, so `argo list`/`logs`/`submit` is how you watch and drive them
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

One of `ANTHROPIC_API_KEY` or `CLAUDE_CODE_OAUTH_TOKEN` is required: these agents run as scheduled tasks with no interactive terminal, so there is no `/login` to fall back on.

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

## Credentials

These agents run as scheduled tasks with no interactive terminal, so they **must** be able to
authenticate non-interactively. Set either `ANTHROPIC_API_KEY` or `CLAUDE_CODE_OAUTH_TOKEN`
before installing — there is no `/login` prompt to fall back on.

To mint a long-lived OAuth token (subscription billing, no browser needed): run
`claude setup-token` on your laptop and pass the result as `CLAUDE_CODE_OAUTH_TOKEN`.


## Scheduling

Every agent here runs as a **scheduled task** (`spec.execution.mode: task`), not an always-on
pod. Their instructions are written around "on each invocation" — do a pass of work, then stop —
and a `CronWorkflow` supplies the invocation.

The schedules are staggered so the supervisor triages before the workers look for queued work,
and so no two workers start at the same instant:

| Agent | Schedule |
|-------|----------|
| `supervisor` | `*/15 * * * *` |
| `worker-0` | `5-59/15 * * * *` |
| `worker-1` | `7-59/15 * * * *` |
| `worker-2` | `9-59/15 * * * *` |

Each carries `concurrencyPolicy: Forbid` so two runs of the same agent never touch the repo at
once, and `activeDeadlineSeconds: 3600` so a stuck run is killed before the next tick.

Watch and drive them with the Argo CLI:

```bash
argo list -n my-cluster                       # runs, newest first
argo logs @latest -n my-cluster               # what the last run did
argo submit --from workflowtemplate/supervisor -n my-cluster   # run one now
kubectl get lagent -n my-cluster              # MODE / PHASE / SCHEDULE / LAST RUN
```

To pause an agent without deleting it, set `spec.execution.suspend: true`. Its schedule stops
firing but the `WorkflowTemplate` stays, so you can still trigger runs by hand.

Because task agents are not addressable, they have no Service and no Ingress — there is no
long-running pod to connect to between runs.

## What's created

- `Secret/github-credentials` — GitHub PAT
- `Secret/anthropic-credentials` — Anthropic API key (only if `ANTHROPIC_API_KEY` was set)
- `Secret/claude-code-oauth` — Claude Code OAuth token (only if `CLAUDE_CODE_OAUTH_TOKEN` was set)
- `Secret/context7-mcp-credentials` — Context7 API key (only if `CONTEXT7_API_KEY` was set)
- `LanguagePersona/project-manager` — supervisor behavioral config
- `LanguagePersona/engineer` — worker behavioral config
- `LanguageTool/context7` — Context7 MCP tool referenced by the workers; the operator generates its `Deployment` and `Service` (tools are long-running servers, not agents)
- `LanguageAgent/supervisor` — triages issues; 5Gi workspace
- `LanguageAgent/worker-0` — queue/0 (urgent); 10Gi workspace; uses `context7`
- `LanguageAgent/worker-1` — queue/1 (normal); 10Gi workspace; uses `context7`
- `LanguageAgent/worker-2` — queue/2 (backlog); 10Gi workspace; uses `context7`

Each `LanguageAgent` also produces a `WorkflowTemplate`, `CronWorkflow`, `NetworkPolicy`, `PersistentVolumeClaim`, `ConfigMap`, and a `ServiceAccount`/`Role`/`RoleBinding`. Because each agent sets `spec.repository`, the operator injects a `repository` init container that clones the repo into the workspace (authenticated with `github-credentials`) and points the runtime's working directory at the checkout via `$AGENT_REPO_DIR`.

## Watching runs

These agents are scheduled tasks, so there is nothing to connect to between runs — no Service,
no Ingress, no terminal. Follow them through Argo instead:

```bash
kubectl get lagent -n my-cluster      # MODE / PHASE / SCHEDULE / LAST RUN
argo list -n my-cluster               # every run, newest first
argo logs @latest -n my-cluster       # what the last run did
argo logs <run-name> -n my-cluster    # a specific run
```

To trigger a pass right now instead of waiting for the schedule:

```bash
argo submit --from workflowtemplate/supervisor -n my-cluster
```

## Teardown

```bash
kubectl delete -k examples/development-team/ -n my-cluster
kubectl delete secret github-credentials anthropic-credentials claude-code-oauth context7-mcp-credentials -n my-cluster --ignore-not-found
```
