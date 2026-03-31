#!/bin/bash
# Usage: push-branch.sh <branch-name>
# Pushes the named local branch to origin and sets upstream tracking.
set -e

BRANCH="$1"
git push -u origin "$BRANCH"
