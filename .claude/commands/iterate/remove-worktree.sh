#!/bin/bash
# Usage: remove-worktree.sh <worktree-path>
# Removes a git worktree, returning to the main repo root first.
set -e

WORKTREE="$1"
MAIN=$(git worktree list --porcelain | grep '^worktree' | head -1 | awk '{print $2}')

cd "$MAIN"
git worktree remove "$WORKTREE" --force 2>&1 || echo "Worktree removal attempted"
