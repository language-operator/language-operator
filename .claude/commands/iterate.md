# Action: do the next logical piece of work

## Prerequisites

Please read the following context files:

* Project: CLAUDE.md
* Persona: requirements/personas/go-engineer.md
* Memory: .claude/MEMORY.md

## Persona

**CRITICAL**: Adopt the given persona while executing these instructions, please.

## Arguments

`$ARGUMENTS` is the queue number to work from: `0`, `1`, or `2`.

## Instructions

Follow these directions closely:

1. Find the next issue for this queue: `gh issue list --label "queue/$ARGUMENTS" --state open --json number,title,labels --limit 1`
   - If no issue is found, report idle and stop.
2. Investigate if the issue is valid, or a mis-use of the intended feature.
3. Label the issue `in-progress` and remove the `queue/$ARGUMENTS` label:
   ```bash
   gh issue edit <N> --add-label "in-progress" --remove-label "queue/$ARGUMENTS"
   ```
4. **Create a worktree** for this issue. Determine a short slug (2-4 words) from the issue title. Then run:
   ```bash
   ISSUE=<N>
   SLUG=<short-slug>
   BRANCH="issue-${ISSUE}-${SLUG}"
   WORKTREE=".claude/worktrees/${BRANCH}"
   MAIN=$(git worktree list --porcelain | grep '^worktree' | head -1 | awk '{print $2}')
   if [ "$(pwd)" != "$MAIN" ]; then
     echo "Already in a worktree, proceeding."
   else
     git fetch origin main && git worktree add -b "$BRANCH" "$WORKTREE" FETCH_HEAD
     cd "$WORKTREE"
   fi
   ```
   All subsequent work happens inside this worktree. Do not `cd` out of it.
5. **CRITICAL:** Switch to plan mode, and propose an implementation plan. Await my feedback.
6. Implement your plan inside the worktree.
7. Run existing tests, and add new ones if necessary. Remember to include CI. Remember the linter.
8. Commit with a semantic, ONE LINE message like `fix: set GatewayReady false on error` and push the branch to origin using an explicit refspec to avoid pushing to main: `git push origin HEAD:"$BRANCH" && git push -u origin "$BRANCH"`
9. Open a pull request: `gh pr create --title "<commit message>" --body "Closes #<N>"`. Use conventional commit style for the PR title.
10. **CRITICAL:** Poll CI on the PR: `gh pr checks <PR-number> --watch`. Fix any failing checks before proceeding.
11. When all checks pass, merge: `gh pr merge <PR-number> --squash --delete-branch`.
12. Clean up the worktree: `git worktree remove "$WORKTREE"`.
13. Remove the `in-progress` label, add a comment with resolution details, then close the issue:
    ```bash
    gh issue edit <N> --remove-label "in-progress"
    gh issue comment <N> --body "<resolution details>"
    gh issue close <N>
    ```
14. Consider if you need to update .claude/MEMORY.md for the next run.  It's not a changelog, it's for things that you may forget.
15. Check if there are remaining issues in this queue: `gh issue list --label "queue/$ARGUMENTS" --state open --json number --limit 1`
    - If issues remain, loop back to step 1 to pick up the next one.
    - If the queue is empty, report idle and stop.

## Output

A merged PR, test coverage, updated CI, and a closed ticket.
