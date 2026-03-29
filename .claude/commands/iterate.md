# Action: do the next logical piece of work

## Prerequisites

Please read the following context files:

* Project: CLAUDE.md
* Persona: requirements/personas/go-engineer.md
* Scratch: requirements/SCRATCH.md

## Persona

**CRITICAL**: Adopt the given persona while executing these instructions, please.

## Instructions

Follow these directions closely:

1. Use the `gh` tool to find an issue for this repository (language-operator/language-operator) labelled "ready". Pick the one that makes the most logical sense to work on next.
2. Investigate if it's valid, or a mis-use of the intended feature.
3. Label the issue "in-progress" so another agent doesn't pick it up.
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
     git worktree add -b "$BRANCH" "$WORKTREE" main
     cd "$WORKTREE"
   fi
   ```
   All subsequent work happens inside this worktree. Do not `cd` out of it.
5. **CRITICAL:** Switch to plan mode, and propose an implementation plan. Await my feedback.
6. Implement your plan inside the worktree.
7. Run existing tests, and add new ones if necessary. Remember to include CI. Remember the linter.
8. Commit with a semantic, ONE LINE message like `fix: set GatewayReady false on error` and push the branch to origin.
9. Open a pull request: `gh pr create --title "<commit message>" --body "Closes #<N>"`. Use conventional commit style for the PR title.
10. **CRITICAL:** Poll CI on the PR: `gh pr checks <PR-number> --watch`. Fix any failing checks before proceeding.
11. When all checks pass, merge: `gh pr merge <PR-number> --squash --delete-branch`.
12. Clean up the worktree: `git worktree remove "$WORKTREE"`.
13. Add a comment to the GitHub issue with resolution details, then close it.
14. Consider if you need to update requirements/SCRATCH.md for the next run.

## Output

A merged PR, test coverage, updated CI, and a closed ticket.