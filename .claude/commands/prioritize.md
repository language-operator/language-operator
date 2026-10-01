---
description: Label the highest-priority open GitHub issues as "ready" for /iterate to pick up first
---

## Prerequisites

Read `.claude/MEMORY.md`, if it exists.

## Directions

1. List open issues, excluding those labelled `in-progress` or `question`:
   ```bash
   gh issue list --state open --limit 500 --json number,title,labels,createdAt \
     --jq 'map(select([.labels[].name] | (index("in-progress") or index("question")) | not))'
   ```
2. Pick the **top few (at most 5)** by impact and urgency: broken behaviour and failing CI first, then work that unblocks other issues, then everything else. Read an issue's body when its title isn't enough to judge.
3. Remove `ready` from any open issue that is not in that set:
   ```bash
   gh issue list --label ready --state open --json number --jq '.[].number'
   gh issue edit <N> --remove-label ready
   ```
4. Add `ready` to each issue in the set: `gh issue edit <N> --add-label ready`.

Update `.claude/MEMORY.md` if anything is worth noting for the next run (e.g. the ranking rationale).

## Output

At most five open issues labelled `ready`. `/iterate` takes `ready` issues first, oldest first.
