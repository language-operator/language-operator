---
description: Check for unassigned GitHub issues and auto-prioritize them into queues
---

# Delegate: triage unassigned work into queues

## Prerequisites

Read:
- `requirements/personas/project-manager.md`
- `.claude/MEMORY.md`

Adopt the project-manager persona.

## Instructions

1. Count open issues with no queue label and not `in-progress`:
   ```bash
   gh issue list --state open --json number,title,labels \
     | jq '[.[] | select(
         (.labels | map(.name) | map(startswith("queue/") or . == "in-progress") | any) | not
       )]'
   ```

2. If **no unassigned issues** are found, report idle and stop.

3. If unassigned issues **are** found:
   - Analyze them for conflict groups (issues touching the same files, controllers, CRDs, or subsystem should serialize)
   - Assign each group to whichever queue (`queue/0`, `queue/1`, `queue/2`) currently has the fewest open issues — preserve existing queue assignments, only label the new issues
   - Apply labels: `gh issue edit <N> --add-label "queue/X"`

4. Report a summary: how many issues were assigned, and to which queues.

## Loop

After completing step 4, wait 1 minute and repeat from step 1. Run indefinitely until stopped.
