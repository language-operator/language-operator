# language-operator-team

Deploys the full Language Operator engineering team into an existing LanguageCluster: the
[development-team](../development-team/) maintainer for the core repo **plus** one maintainer
agent for each of the seven sister adapter repositories.

Each agent declares its repo via `spec.repository`, so the operator clones it into the agent's
workspace on init and starts the runtime inside the checkout (exposed as `$AGENT_REPO_DIR`) — no
manual `git clone` in the prompt. Every agent uses the `claude-code` runtime.

```
# one maintainer per repo — each triages AND implements its own repo
maintainer          (engineer persona)  → ${PROJECT_REPOSITORY} (the core repo)
adapter-claude-code (engineer persona)  → language-operator/claude-code-adapter
adapter-openclaw    (engineer persona)  → language-operator/openclaw-adapter
adapter-opencode    (engineer persona)  → language-operator/opencode-adapter
adapter-deepagents  (engineer persona)  → language-operator/deepagents-adapter
adapter-kilo        (engineer persona)  → language-operator/kilo-adapter
adapter-qwen-code   (engineer persona)  → language-operator/qwen-code-adapter
adapter-cursor      (engineer persona)  → language-operator/cursor-adapter
```

Every agent owns a single repo end to end and picks work the same way. It skips issues labelled
`in-progress` or `question`, then takes the first match among `ready` → `bug` → `enhancement` →
`tech-debt`/`documentation` → everything else, oldest first within each group. It then
validates → implements → tests → opens a PR → merges → closes the issue, and stops: **one issue per
run**, with the schedule supplying the next. A single agent per repo serializes naturally, so no
triage queues are needed.

All eight agents reference the [`context7`](../tools/context7/) `LanguageTool`, giving them
up-to-date, version-specific library documentation over MCP so they stop coding against stale or
hallucinated APIs.

> The seven `adapter-*` agents hardcode their `spec.repository.url` to the
> `language-operator/<name>-adapter` repos. Edit those URLs in
> `languageagent.adapter-*.yaml` if you work against forks.

## Prerequisites

- Language Operator [installed](https://langop.io/docs/getting-started/installation/)
- `language-operator-runtimes` chart installed (provides the `claude-code` runtime)
- `kubectl` configured for your cluster
- the [`argo` CLI](https://github.com/argoproj/argo-workflows/releases) — these agents are scheduled tasks, so `argo list`/`logs`/`submit` is how you watch and drive them
- `envsubst` (`brew install gettext` on macOS, pre-installed on most Linux distros)
- A `LanguageCluster` already applied in the target namespace (see [clusters/basic](../clusters/basic/))
- A Claude account (Pro, Max, Team, or Enterprise)
- A GitHub personal access token with `repo` and `issues` scopes — used both to authenticate the clones (`spec.repository.secretRef`) and for the agents' `gh` operations. The same token must have access to the core repo **and** the four adapter repos.

## Variables

| Variable | Required | Default | Description |
|----------|----------|---------|-------------|
| `CLUSTER_NAME` | yes | — | Namespace of the target LanguageCluster |
| `GITHUB_TOKEN` | yes | — | GitHub PAT (`repo`, `issues` scopes) — written to the `github-credentials/token` secret. The operator uses it for the clones, for `git push`, and exports it as `GH_TOKEN` for `gh`. Needs access to the core repo and all four adapter repos. |
| `PROJECT_REPOSITORY` | yes | — | Clone URL of the **core** repo the maintainer works on (e.g. `https://github.com/language-operator/language-operator.git`). Cloned into the maintainer's workspace via `spec.repository`. The adapter agents hardcode their own repo URLs. |
| `PROJECT_NAME` | no | repo basename (minus `.git`) | Human-readable project name used in the maintainer's prompt (e.g. `Language Operator`) |
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
  bash examples/language-operator-team/install.sh
```

Dry-run (prints rendered YAML, skips secret creation):
```bash
CLUSTER_NAME=my-cluster \
  GITHUB_TOKEN=ghp_... \
  PROJECT_REPOSITORY=https://github.com/language-operator/language-operator.git \
  bash examples/language-operator-team/install.sh --dry-run
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

The core maintainer runs every 15 minutes; the adapter agents run every 30, staggered so they do
not all start at the same instant:

| Agent | Schedule |
|-------|----------|
| `maintainer` | `5-59/15 * * * *` |
| `adapter-claude-code` | `*/30 * * * *` |
| `adapter-cursor` | `2-59/30 * * * *` |
| `adapter-qwen-code` | `7-59/30 * * * *` |
| `adapter-deepagents` | `10-59/30 * * * *` |
| `adapter-kilo` | `15-59/30 * * * *` |
| `adapter-openclaw` | `20-59/30 * * * *` |
| `adapter-opencode` | `25-59/30 * * * *` |

Each carries `concurrencyPolicy: Forbid` so two runs of the same agent never touch the repo at
once, and `activeDeadlineSeconds: 3600` so a stuck run is killed before the next tick.

Watch and drive them with the Argo CLI:

```bash
argo list -n my-cluster                       # runs, newest first
argo logs @latest -n my-cluster               # what the last run did
argo submit --from workflowtemplate/maintainer -n my-cluster   # run one now
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
- `LanguagePersona/engineer` — behavioral config shared by every agent
- `LanguageTool/context7` — Context7 MCP tool referenced by every agent; the operator generates its `Deployment` and `Service` (tools are long-running servers, not agents)
- `LanguageAgent/maintainer` — owns the core repo; 10Gi workspace; uses `context7`
- `LanguageAgent/adapter-claude-code` — owns `claude-code-adapter`; 10Gi workspace; uses `context7`
- `LanguageAgent/adapter-openclaw` — owns `openclaw-adapter`; 10Gi workspace; uses `context7`
- `LanguageAgent/adapter-opencode` — owns `opencode-adapter`; 10Gi workspace; uses `context7`
- `LanguageAgent/adapter-deepagents` — owns `deepagents-adapter`; 10Gi workspace; uses `context7`
- `LanguageAgent/adapter-kilo` — owns `kilo-adapter`; 10Gi workspace; uses `context7`
- `LanguageAgent/adapter-qwen-code` — owns `qwen-code-adapter`; 10Gi workspace; uses `context7`
- `LanguageAgent/adapter-cursor` — owns `cursor-adapter`; 10Gi workspace; uses `context7`

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
argo submit --from workflowtemplate/maintainer -n my-cluster
```

## Teardown

```bash
kubectl delete -k examples/language-operator-team/ -n my-cluster
kubectl delete secret github-credentials anthropic-credentials claude-code-oauth context7-mcp-credentials -n my-cluster --ignore-not-found
```
