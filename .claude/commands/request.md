---
description: Plan a feature with the user and file GitHub issues
---

## Inputs

- `$ARGUMENTS` — description of the feature to plan (required)

## Prerequisites

Read:
- `CLAUDE.md`
- `.claude/MEMORY.md`

## Directions

You are a product-minded engineer helping the user turn a feature idea into well-scoped GitHub issues.

### Step 1 — Understand the request

Parse `$ARGUMENTS` as the feature description. If it is missing or too vague to act on, ask the user to clarify before proceeding.

### Step 2 — Explore the codebase

Read the parts of the codebase most likely affected. Depending on the feature, this may include controller files in `src/controllers/`, CRD types in `src/api/v1alpha1/`, Helm chart values/templates, or dashboard components. Identify:
- What already exists that the feature can build on
- What is missing or needs to change
- Technical constraints or risks worth surfacing

Also check for documentation that will need updating:
- `docs/` pages referencing affected APIs, paths, or workflows
- `README.md` if it documents the affected area
- `CLAUDE.md` if it has stale development instructions

If docs changes are non-trivial, include a `docs:` issue in the plan.

### Step 3 — Enter plan mode and present the feature plan

Enter plan mode. Present a structured proposal:

**Feature:** one-line summary of what the feature does and why it matters  
**Motivation:** 1–3 sentences on the problem it solves  
**Proposed approach:** concise implementation strategy referencing specific files and APIs  
**Issue breakdown:** a numbered list of proposed GitHub issues, each with:
  - Proposed title (conventional commit style: `feat:`, `chore:`, etc.)
  - Scope: what this issue covers and what it explicitly does NOT cover
  - Dependencies: which other issues (if any) must land first

Split concerns across issues when the work spans different layers (e.g. CRD type change, controller logic, Helm chart, dashboard). Each issue should be independently workable and reviewable.

Ask the user:
- Does the motivation and approach match their intent?
- Should any issues be merged, split, or dropped?
- Are there acceptance criteria to add?

Await confirmation. Revise the plan if needed.

### Step 4 — File the issues

Once the user confirms, exit plan mode. First deduplicate:

```bash
gh issue list --repo language-operator/language-operator --state open --limit 100
```

Skip any issue already covered by an open issue; note the existing issue number in your working notes.

Then file each confirmed issue:

```bash
gh issue create \
  --repo language-operator/language-operator \
  --title "<title>" \
  --label "enhancement,ready" \
  --body "<body>"
```

Use this body template for each issue:

```
**Context:**
Why this issue exists — the problem or user need it addresses.

**Proposal:**
What to build. Be concrete about files, APIs, or behaviors affected.

**Acceptance Criteria:**
- [ ] ...
- [ ] ...
```

### Step 5 — Summarise

Print a table of all issues filed: issue number and title. Note any findings skipped because they matched existing open issues, with the existing issue number.
