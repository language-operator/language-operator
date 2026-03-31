#!/bin/bash
# Usage: start-issue.sh <issue-number> <short-slug> <queue-number>
# Labels the issue in-progress, removes it from its queue, and creates a worktree.
set -e

ISSUE="$1"
SLUG="$2"
QUEUE="$3"

gh issue edit "$ISSUE" --add-label "in-progress" --remove-label "queue/$QUEUE"

BRANCH="issue-${ISSUE}-${SLUG}"
WORKTREE=".claude/worktrees/${BRANCH}"
MAIN=$(git worktree list --porcelain | grep '^worktree' | head -1 | awk '{print $2}')

if [ "$(pwd)" != "$MAIN" ]; then
  echo "Already in a worktree, proceeding."
  echo "worktree:$(pwd)"
else
  git fetch origin main
  git worktree add -b "$BRANCH" "$WORKTREE" FETCH_HEAD
  echo "worktree:${WORKTREE}"
fi
