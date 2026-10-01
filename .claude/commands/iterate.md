---
description: Do the next logical piece of work — one issue, from pick to merged PR to closed
argument-hint: "[#issue] [--auto]"
allowed-tools: Bash(gh:*), Bash(git:*), Bash(bash .claude/commands/iterate/*), Bash(printenv:*), Bash(make:*), Bash(go:*), Bash(helm:*), Bash(uv:*), Bash(pytest:*), Read, Edit, Write, Glob, Grep
---
<!-- Canonical /iterate (language-operator#932). Only the frontmatter `allowed-tools`
     build-tool entries and the `## Testing` section vary per repo; everything else,
     and the scripts in .claude/commands/iterate/, is copied verbatim. -->

# Iterate: do the next logical piece of work

One run handles **one issue**, from selection to a merged PR and a closed issue, then stops.
For continuous work, use `/loop /iterate` or a scheduled agent.

## Context

Read:
- `CLAUDE.md`
- `README.md`
- `.claude/MEMORY.md`, if it exists

## Arguments

`$ARGUMENTS` may contain:
- *(nothing)*: pick the next issue (see below).
- `#N` or `N`: work issue N.
- `--auto`: unattended mode (see step 4).

## Picking the next issue

```bash
gh issue list --state open --limit 500 --json number,title,labels,createdAt --jq '
  map(select([.labels[].name] | (index("in-progress") or index("question")) | not))
  | map(. + {rank: ([.labels[].name] as $l |
      if $l | index("ready") then 0
      elif $l | index("bug") then 1
      elif $l | index("enhancement") then 2
      elif ($l | index("tech-debt")) or ($l | index("documentation")) then 3
      else 4 end)})
  | sort_by(.rank, .createdAt) | first // empty'
```

This skips issues labelled `in-progress` or `question`, then takes the first match in order: `ready` (set by `/prioritize`), `bug`, `enhancement`, `tech-debt`/`documentation`, everything else; oldest first within a group. If the output is empty, report idle and stop.

## Steps

1. **Select** the issue, either as above or from `#N`. If it's closed or not found, report and stop. Read the body and comments: `gh issue view <N> --comments`.
2. **Validate.** If the issue is invalid, a duplicate or out of date, comment why, close it, and go back to step 1. (With `#N`, stop instead.)
3. **Claim** the issue. Pick a short slug (2–4 words) from the title, then:
   ```bash
   bash .claude/commands/iterate/start-issue.sh <N> <short-slug>
   ```
   - The script adds `in-progress` (creating the label if the repo lacks it) before creating the worktree.
   - If it exits non-zero because the issue is already `in-progress`, go back to step 1 (with `#N`, stop).
   - It prints `worktree:<path>`. `cd` into that path and stay there for the rest of the run.
4. **Plan.** The run is unattended if `printenv AGENT_NAME` prints a value (the operator injects it into every agent pod) or `$ARGUMENTS` contains `--auto`.
   - Interactive: enter plan mode, propose the plan, and wait for approval.
   - Unattended: post the plan as a comment (`gh issue comment <N> --body "<plan>"`) and continue.
5. **Implement** the plan inside the worktree.
6. **Test**, following `## Testing` below. Add tests as needed.
7. **Commit** with a one-line conventional message (e.g. `fix: set GatewayReady false on error`), then push. Run these as separate commands, without inline variable assignments:
   ```bash
   bash .claude/commands/iterate/push-branch.sh <branch-name>
   ```
8. **Open a PR**: `gh pr create --title "<commit message>" --body "Closes #<N>"`.
9. **Watch CI**: `gh pr checks <PR> --watch`. Fix failures until all checks are green.
10. **Merge**: `gh pr merge <PR> --squash --delete-branch`.
11. **Clean up** the worktree (run from inside it; no arguments needed):
    ```bash
    bash .claude/commands/iterate/remove-worktree.sh
    ```
12. **Close the issue**:
    ```bash
    gh issue comment <N> --body "<resolution details>"
    gh issue edit <N> --remove-label "in-progress"
    gh issue close <N>
    ```
13. **Update `.claude/MEMORY.md`** if it exists and something is worth remembering for the next run (it's not a changelog). Then **stop**.

<!-- per-repo: Testing -->
## Testing

Mirror the PR CI jobs (`.github/workflows/test.yaml`, `pr-checks.yaml`):

- Lint and unit tests: `cd src && make test` (go fmt, go vet, all tests)
- Integration tests: `cd src && make integration-test`
- Model gateway touched: `pytest components/model-gateway/test_generate_config.py`
- `src/api/v1alpha1/` touched: `cd src && make generate && make helm-crds`, and stage the generated output (`validate-manifests` fails on drift)
- A chart touched: `helm dependency build charts/<chart> && helm lint charts/<chart>`
- Docs touched: `make docs-build`
- The PR title must be a conventional commit (`feat:`, `fix:`, `chore:`, `docs:`, `test:`); `clean:` is rejected
<!-- /per-repo -->
