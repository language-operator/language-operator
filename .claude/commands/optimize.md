---
description: Find and propose one high-impact tech debt reduction — dead code, duplication, magic strings
---

## Inputs

- $PERSONA (optional, default: go-engineer) — persona to adopt; definitions in `requirements/personas/`

## Prerequisites

Read:
- `requirements/personas/$PERSONA.md`
- `.claude/MEMORY.md`

## Directions

Adopt the $PERSONA persona.

You are a detective of tech debt. Find:
- Opportunities to reduce lines of code
- DRY violations
- Dead code paths
- Duplicate utility implementations
- Magic strings
- Other tech debt

This code has been written by different agents with different contexts, unaware of overall patterns. These cross-cutting optimizations are high priority.

## Output

Enter plan mode. Present ONE finding concisely — what to change, why it matters,
and which files are affected. Ask the user if they want a GitHub issue filed.

Do NOT implement. Do NOT file an issue until the user confirms.

Update `.claude/MEMORY.md` if anything is worth remembering for the future.
