---
description: Label the highest-priority GitHub issues as "ready" using project-manager persona
---

## Prerequisites

Read:
- `requirements/personas/project-manager.md`
- `.claude/MEMORY.md`

Adopt the project-manager persona.

## Directions

1. Use `gh` to view all open issues (excluding those already labelled `in-progress`)
2. Clear any existing `queue/0`, `queue/1`, `queue/2` labels from all open issues:
   ```bash
   gh issue list --label "queue/0" --state open --json number | jq -r '.[].number' | xargs -I{} gh issue edit {} --remove-label "queue/0"
   gh issue list --label "queue/1" --state open --json number | jq -r '.[].number' | xargs -I{} gh issue edit {} --remove-label "queue/1"
   gh issue list --label "queue/2" --state open --json number | jq -r '.[].number' | xargs -I{} gh issue edit {} --remove-label "queue/2"
   ```
3. Analyze the open issues for **conflict groups** — issues that likely touch the same files, controllers, CRDs, or areas of the codebase should be in the same group (they must serialize). Issues touching unrelated areas can run in parallel across queues.
4. Assign each conflict group to a queue. Label **all** issues in a group with the same queue label (the agent will work through them in priority order):
   - All issues in group 1 → `queue/0`
   - All issues in group 2 → `queue/1`
   - All issues in group 3 → `queue/2`
   - If there are fewer than 3 independent groups, only use as many queues as there are distinct groups.
5. Apply the queue labels: `gh issue edit <N> --add-label "queue/0"` (etc.)

Update `.claude/MEMORY.md` if anything is worth noting for the next run (e.g. the grouping rationale).

## Output

Up to three queues, each containing all issues from one conflict group, labelled `queue/0`, `queue/1`, or `queue/2`. Each queue is the full serialized workload for one agent.
