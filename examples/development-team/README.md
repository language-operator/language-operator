# development-team

Deploys a self-managing maintainer agent into an existing LanguageCluster, pointed at any git repository: on each scheduled run it picks the next open issue, implements it, runs the tests, opens a pull request, merges it once CI is green, and closes the issue.

The agent declares the project repo via `spec.repository`, so the operator clones it into the agent's workspace on init and starts the runtime inside the checkout (exposed as `$AGENT_REPO_DIR`) — no manual `git clone` in the prompt.

```
maintainer (engineer persona)  → ${PROJECT_REPOSITORY}
  └─ each run: pick next issue → validate → implement → test → PR → merge → close → stop
```

The maintainer picks work in a fixed order. It skips issues labelled `in-progress` or
`question`, then takes the first match among `ready` → `bug` → `enhancement` →
`tech-debt`/`documentation` → everything else, oldest first within each group. Label an issue
`ready` to jump it to the front. It handles **one issue per run** and the schedule supplies the
next; a single agent per repo serializes naturally, so no triage queues are needed.

The maintainer references the [`context7`](../tools/context7/) `LanguageTool`, giving it
up-to-date, version-specific library documentation over MCP so it stops coding against stale or
hallucinated APIs.

## Prerequisites

- Language Operator [installed](https://langop.io/docs/getting-started/installation/)
- `language-operator-runtimes` chart installed (provides the `claude-code` runtime)
- `kubectl` configured for your cluster
- the [`argo` CLI](https://github.com/argoproj/argo-workflows/releases) — the maintainer is a scheduled task, so `argo list`/`logs`/`submit` is how you watch and drive it
- `envsubst` (`brew install gettext` on macOS, pre-installed on most Linux distros)
- A `LanguageCluster` already applied in the target namespace (see [clusters/basic](../clusters/basic/))
- A Claude account (Pro, Max, Team, or Enterprise)
- A GitHub personal access token with `repo` and `issues` scopes — used both to authenticate the clone (`spec.repository.secretRef`) and for the agent's `gh` operations

## Variables

| Variable | Required | Default | Description |
|----------|----------|---------|-------------|
| `CLUSTER_NAME` | yes | — | Namespace of the target LanguageCluster |
| `GITHUB_TOKEN` | yes | — | GitHub PAT (`repo`, `issues` scopes) — written to the `github-credentials/token` secret. The operator uses it for the clone, for `git push`, and exports it as `GH_TOKEN` for `gh` |
| `PROJECT_REPOSITORY` | yes | — | Clone URL of the project the maintainer will work on (e.g. `https://github.com/language-operator/language-operator.git`). Cloned into the agent's workspace via `spec.repository`. |
| `PROJECT_NAME` | no | repo basename (minus `.git`) | Human-readable project name used in agent prompts (e.g. `Language Operator`) |
| `ANTHROPIC_API_KEY` | no | — | If set, written to `anthropic-credentials/api-key` and injected as `ANTHROPIC_API_KEY` on the agent (API-key billing). |
| `CLAUDE_CODE_OAUTH_TOKEN` | no | — | Long-lived subscription token from `claude setup-token`. Written to `claude-code-oauth/token` and injected as `CLAUDE_CODE_OAUTH_TOKEN` (subscription billing, headless — no `/login` needed). |
| `CONTEXT7_API_KEY` | no | — | [Context7 API key](https://context7.com/dashboard) for higher rate limits. Written to `context7-mcp-credentials/api-key`. The `context7` tool works without it at a lower rate limit. |

One of `ANTHROPIC_API_KEY` or `CLAUDE_CODE_OAUTH_TOKEN` is required: the maintainer runs as a scheduled task with no interactive terminal, so there is no `/login` to fall back on.

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

The maintainer runs as a scheduled task with no interactive terminal, so it **must** be able to
authenticate non-interactively. Set either `ANTHROPIC_API_KEY` or `CLAUDE_CODE_OAUTH_TOKEN`
before installing — there is no `/login` prompt to fall back on.

To mint a long-lived OAuth token (subscription billing, no browser needed): run
`claude setup-token` on your laptop and pass the result as `CLAUDE_CODE_OAUTH_TOKEN`.


## Scheduling

The maintainer runs as a **scheduled task** (`spec.execution.mode: task`), not an always-on
pod. Its instructions are written around "on each invocation" — do a pass of work, then stop —
and a `CronWorkflow` supplies the invocation.

| Agent | Schedule |
|-------|----------|
| `maintainer` | `5-59/15 * * * *` |

It carries `concurrencyPolicy: Forbid` so two runs of the same agent never touch the repo at
once, and `activeDeadlineSeconds: 3600` so a stuck run is killed before the next tick.

Watch and drive it with the Argo CLI:

```bash
argo list -n my-cluster                       # runs, newest first
argo logs @latest -n my-cluster               # what the last run did
argo submit --from workflowtemplate/maintainer -n my-cluster   # run one now
kubectl get lagent -n my-cluster              # MODE / PHASE / SCHEDULE / LAST RUN
```

To pause the maintainer without deleting it, set `spec.execution.suspend: true`. Its schedule stops
firing but the `WorkflowTemplate` stays, so you can still trigger runs by hand.

Because task agents are not addressable, they have no Service and no Ingress — there is no
long-running pod to connect to between runs.

## What's created

- `Secret/github-credentials` — GitHub PAT
- `Secret/anthropic-credentials` — Anthropic API key (only if `ANTHROPIC_API_KEY` was set)
- `Secret/claude-code-oauth` — Claude Code OAuth token (only if `CLAUDE_CODE_OAUTH_TOKEN` was set)
- `Secret/context7-mcp-credentials` — Context7 API key (only if `CONTEXT7_API_KEY` was set)
- `LanguagePersona/engineer` — maintainer behavioral config
- `LanguageTool/context7` — Context7 MCP tool referenced by the maintainer; the operator generates its `Deployment` and `Service` (tools are long-running servers, not agents)
- `LanguageAgent/maintainer` — triages and implements the repo's issues, one per run; 10Gi workspace; uses `context7`

The `LanguageAgent` also produces a `WorkflowTemplate`, `CronWorkflow`, `NetworkPolicy`, `PersistentVolumeClaim`, `ConfigMap`, and a `ServiceAccount`/`Role`/`RoleBinding`. Because the agent sets `spec.repository`, the operator injects a `repository` init container that clones the repo into the workspace (authenticated with `github-credentials`) and points the runtime's working directory at the checkout via `$AGENT_REPO_DIR`.

## Watching runs

The maintainer is a scheduled task, so there is nothing to connect to between runs — no Service,
no Ingress, no terminal. Follow it through Argo instead:

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
kubectl delete -k examples/development-team/ -n my-cluster
kubectl delete secret github-credentials anthropic-credentials claude-code-oauth context7-mcp-credentials -n my-cluster --ignore-not-found
```
