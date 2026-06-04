#!/bin/sh
# Launch Claude Code with the agent's role and opening task from operator
# env vars. AGENT_PERSONA → --append-system-prompt (role/tone); AGENT_INSTRUCTIONS
# → initial user message (the task to execute on startup). Either or both may be
# absent — falls back to bare interactive claude.
set -eu

if [ -n "${AGENT_PERSONA:-}" ]; then
    set -- --append-system-prompt "$AGENT_PERSONA"
else
    set --
fi

if [ -n "${AGENT_INSTRUCTIONS:-}" ]; then
    exec claude "$@" "$AGENT_INSTRUCTIONS"
else
    exec claude "$@"
fi
