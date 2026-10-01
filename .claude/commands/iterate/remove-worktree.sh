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

# git worktree remove can unregister the worktree and still leave the directory
# behind ("Directory not empty"), which leaves a tree git no longer tracks and
# prune will not collect. Remove the directory too, then prune.
#
# Guarded: only ever delete inside the worktrees directory this command creates,
# so a stray argument cannot turn this into an arbitrary recursive delete.
case "$WORKTREE" in
  "$MAIN"/.claude/worktrees/?*) ;;
  *)
    echo "Refusing to remove '$WORKTREE': not under $MAIN/.claude/worktrees/" >&2
    exit 1
    ;;
esac

git worktree remove "$WORKTREE" --force 2>&1 || true
if [ -e "$WORKTREE" ]; then
  rm -rf "$WORKTREE"
fi
git worktree prune

if [ -e "$WORKTREE" ]; then
  echo "Failed to remove $WORKTREE" >&2
  exit 1
fi
