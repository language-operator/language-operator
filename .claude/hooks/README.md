# Hooks

Event-driven automation for Claude Code in the Language Operator project.

## Available Hooks

- [skill-activation-prompt](skill-activation-prompt.sh) - Auto-activates relevant skills based on user prompts and file context
- [pre-tool-use-bash-approval](pre-tool-use-bash-approval.sh) - Auto-approves named bash operations from delegate/watch/iterate workflows; patterns configured in [bash-approvals.json](bash-approvals.json)

## Setup

Install dependencies:
```bash
cd .claude/hooks
npm install
chmod +x *.sh
```

## Hook Types

- **UserPromptSubmit**: Runs when you submit a prompt (skill activation)
- **PreToolUse**: Runs before tools execute (validation, permission checks)  
- **PostToolUse**: Runs after tools complete (file tracking, state management)
- **Stop**: Runs when execution is stopped (cleanup, logging)

## Development

Hooks are implemented as TypeScript scripts with shell wrappers for better error handling and type safety.