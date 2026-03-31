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
4. Assign each conflict group to a queue. Label only the **single highest-priority issue** from each group — leave the rest unlabeled for `delegate` to fill in as queues drain:
   - Top issue from group 1 → `queue/0`
   - Top issue from group 2 → `queue/1`
   - Top issue from group 3 → `queue/2`
   - If there are fewer than 3 independent groups, only use as many queues as there are distinct groups.
5. Apply the queue labels: `gh issue edit <N> --add-label "queue/0"` (etc.)

Update `.claude/MEMORY.md` if anything is worth noting for the next run (e.g. the grouping rationale).

## Output

Up to three queues, each containing exactly one issue — the next item for that agent to pick up. Remaining issues stay unassigned until `delegate` refills the queue.
