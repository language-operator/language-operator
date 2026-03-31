#!/bin/bash
# Usage: remove-worktree.sh [worktree-path]
# Removes a worktree. Pass the path explicitly, or run from inside the worktree.
set -e

MAIN=$(git worktree list --porcelain | grep '^worktree' | head -1 | awk '{print $2}')
WORKTREE="${1:-$(pwd)}"

if [ "$WORKTREE" = "$MAIN" ]; then
  echo "Not in a worktree, nothing to remove."
  exit 0
fi

cd "$MAIN"
git worktree remove "$WORKTREE" --force 2>&1 || echo "Worktree removal attempted"
