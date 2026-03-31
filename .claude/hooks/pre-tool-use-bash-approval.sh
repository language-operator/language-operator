#!/bin/bash
set -e

if [[ -n "$CLAUDE_PROJECT_DIR" ]]; then
    HOOKS_DIR="$CLAUDE_PROJECT_DIR/.claude/hooks"
else
    SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
    HOOKS_DIR="$SCRIPT_DIR"
fi

cd "$HOOKS_DIR"
cat | npx tsx pre-tool-use-bash-approval.ts
