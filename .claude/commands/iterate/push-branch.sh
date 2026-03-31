#!/bin/bash
# Usage: push-branch.sh <branch-name>
# Pushes the current HEAD to the named branch and sets upstream tracking.
set -e

BRANCH="$1"
git push origin HEAD:"$BRANCH"
git push -u origin "$BRANCH"
