#!/bin/bash
# Usage: start-issue.sh <issue-number> <short-slug>
# Claims the issue (labels it in-progress) and creates a worktree for it.
# Exits 3 without creating a worktree if the issue is already in-progress.
set -euo pipefail

ISSUE="$1"
SLUG="$2"

# Create the label if the repo lacks it, without touching an existing one.
if ! gh label list --search in-progress --json name --jq '.[].name' | grep -qx in-progress; then
  gh label create in-progress --description "Being worked on" --color fbca04
fi

if gh issue view "$ISSUE" --json labels --jq '.labels[].name' | grep -qx in-progress; then
  echo "Issue #${ISSUE} is already in-progress." >&2
  exit 3
fi

gh issue edit "$ISSUE" --add-label "in-progress"

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
