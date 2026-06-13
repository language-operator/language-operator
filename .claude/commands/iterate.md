# Action: do the next logical piece of work

## Prerequisites

Please read the following context files:

* Project: CLAUDE.md
* Persona: requirements/personas/go-engineer.md
* Memory: .claude/MEMORY.md

## Persona

**CRITICAL**: Adopt the given persona while executing these instructions, please.

## Arguments

`$ARGUMENTS` is either:
- A queue number to work from: `0`, `1`, or `2`
- A specific issue reference: `#706`, `issue #706`, or bare `706`

## Instructions

Follow these directions closely:

1. Determine the issue to work on based on `$ARGUMENTS`:
   - **If it looks like an issue ID** (contains `#` or is a plain integer ≥ 10, e.g. `#706`, `issue #706`, `706`):
     Parse out the number N and fetch it directly: `gh issue view <N> --json number,title,labels,state`
     - If the issue is closed or not found, report and stop.
     - Read its comments as well: `gh issue view <N> --comments`
     - Note: since this issue was not pulled from a queue, skip the queue-label removal in step 3 and pass an empty string for `<queue-number>` to `start-issue.sh`.
   - **Otherwise** treat `$ARGUMENTS` as a queue label:
     `gh issue list --label "queue/$ARGUMENTS" --state open --json number,title,labels --limit 1`
     - If no issue is found, report idle and stop.
     - If found, read its comments as well.
   - **If no argument** pick the next logical issue.
2. Investigate if the issue is valid, or a mis-use of the intended feature.
3. **Label and create worktree** in one step. Determine a short slug (2-4 words) from the issue title, then:
   ```bash
   bash .claude/commands/iterate/start-issue.sh <N> <short-slug> <queue-number>
   ```
   The script labels the issue `in-progress`, removes the queue label, and creates a worktree. It prints `worktree:<path>` — `cd` into that path. All subsequent work happens inside this worktree. Do not `cd` out of it.
5. **CRITICAL:** Switch to plan mode, and propose an implementation plan. Await my feedback.
6. Implement your plan inside the worktree.
7. Run existing tests, and add new ones if necessary. Remember to include CI. Remember the linter.
8. Commit with a semantic, ONE LINE message like `fix: set GatewayReady false on error` and push the branch — run as two separate commands, do not use inline variable assignments:
   ```bash
   bash .claude/commands/iterate/push-branch.sh <branch-name>
   ```
9. Open a pull request: `gh pr create --title "<commit message>" --body "Closes #<N>"`. Use conventional commit style for the PR title.
10. **CRITICAL:** Poll CI on the PR: `gh pr checks <PR-number> --watch`. Fix any failing checks before proceeding.
11. When all checks pass, merge: `gh pr merge <PR-number> --squash --delete-branch`.
12. Clean up the worktree (run from inside it — no arguments needed):
    ```bash
    bash .claude/commands/iterate/remove-worktree.sh
    ```
13. Remove the `in-progress` label, add a comment with resolution details, then close the issue:
    ```bash
    gh issue edit <N> --remove-label "in-progress"
    gh issue comment <N> --body "<resolution details>"
    gh issue close <N>
    ```
14. Consider if you need to update .claude/MEMORY.md for the next run.  It's not a changelog, it's for things that you may forget.
15. If `$ARGUMENTS` was a queue number (not a specific issue ID), check for remaining issues:
    `gh issue list --label "queue/$ARGUMENTS" --state open --json number --limit 1`
    - If issues remain, loop back to step 1 to pick up the next one.
    - If the queue is empty, report idle and stop.
    - If `$ARGUMENTS` was a specific issue ID, stop here.

## Output

A merged PR, test coverage, updated CI, and a closed ticket.
