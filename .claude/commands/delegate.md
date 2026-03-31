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
   - Check how many open issues each queue currently has:
     ```bash
     for q in 0 1 2; do
       echo "queue/$q: $(gh issue list --label "queue/$q" --state open --json number | jq 'length')"
     done
     ```
   - For each queue that has **0 open issues**, pick the single highest-priority unassigned issue that fits that queue's conflict group and assign it
   - Do not assign to a queue that already has an open issue — it is not ready to receive new work yet
   - Apply labels: `gh issue edit <N> --add-label "queue/X"`

4. Report a summary: which queues were refilled and with which issue, and which queues were skipped (still have work).

## Loop

After completing step 4, wait 1 minute and repeat from step 1. Run indefinitely until stopped.
