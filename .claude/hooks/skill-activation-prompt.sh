#!/bin/bash
set -e

# Use CLAUDE_PROJECT_DIR if set, otherwise detect from script location
if [[ -n "$CLAUDE_PROJECT_DIR" ]]; then
    HOOKS_DIR="$CLAUDE_PROJECT_DIR/.claude/hooks"
else
    # Get the directory of this script
    SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
    HOOKS_DIR="$SCRIPT_DIR"
fi

cd "$HOOKS_DIR"
cat | npx tsx skill-activation-prompt.ts