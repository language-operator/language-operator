#!/bin/bash
# Usage: create-worktree.sh <issue-number> <short-slug>
# Creates a git worktree for isolated issue work and prints the worktree path.
set -e

ISSUE="$1"
SLUG="$2"
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
