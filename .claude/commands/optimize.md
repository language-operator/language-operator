---
description: Find and propose one high-impact tech debt reduction — dead code, duplication, magic strings
---

## Inputs

- $PERSONA (optional, default: go-engineer) — persona to adopt; definitions in `requirements/personas/`

## Prerequisites

Read:
- `requirements/personas/$PERSONA.md`
- `requirements/SCRATCH.md`

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

Propose ONE high-impact optimization or refactor. Update `requirements/SCRATCH.md` if anything is worth remembering for the future.
