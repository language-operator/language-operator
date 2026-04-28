---
description: Verify that a feature implementation satisfies the original request — reopen issues if incomplete or incorrect
---

## Inputs

- `$ARGUMENTS` — one or more of:
  - Issue numbers filed by `/request`: `#100`, `#101 #102`, or bare `100 101`
  - A PR number: `pr 42` or `#42`
  - Empty: look back at conversation history for the most recent `/request` run

## Directions

You are a strict QA reviewer. Your job is to verify that a feature was implemented **completely and correctly** — not just that code was merged. Partial or incorrect implementations get reopened.

### Step 1 — Identify the issues to review

**If `$ARGUMENTS` contains issue numbers:** fetch each directly.

**If `$ARGUMENTS` looks like a PR number** (e.g. `pr 42`):
```bash
gh pr view <N> --repo language-operator/language-operator --json number,title,body,closingIssuesReferences
```
Extract issue numbers from `closingIssuesReferences`.

**If `$ARGUMENTS` is empty:** look back at conversation history for the most recent `/request` run and extract issue numbers from its Step 5 summary table.

For each issue, fetch its full body and comments:
```bash
gh issue view <N> --repo language-operator/language-operator
gh issue view <N> --repo language-operator/language-operator --comments
```

If an issue is still open (not yet implemented), note it and skip — this command reviews implementations, not plans.

### Step 2 — For each issue, locate the implementation

Find the PR that closed the issue:
- Check the issue's closing comment or timeline for a PR reference
- If no PR is linked, scan recent commits: `git log --oneline -30`

Inspect the diff:
```bash
gh pr diff <PR> --repo language-operator/language-operator
```

Also read the **current state** of all affected files using Read and Grep — the diff shows intent, but the current code state determines correctness.

### Step 3 — Verify against acceptance criteria

For each issue:

1. **Parse the issue body** — extract:
   - The **Proposal**: what was supposed to be built
   - The **Acceptance Criteria**: the checklist of things that must be true

2. **Check each acceptance criterion explicitly.** For each item:
   - Read the relevant code or file
   - Determine: is this criterion actually satisfied in the current state of the repo?
   - Note any that are unmet, partially met, or cannot be verified without running the cluster

3. **Check test coverage** — does the implementation include tests for the new behavior?
   - Controller changes: unit tests in `src/controllers/` and/or integration tests
   - CRD type changes: webhook validation tests if applicable
   - Dashboard changes: component tests, or note that manual testing is required

4. **Check documentation** — if the feature touches public-facing APIs, config fields, or workflows:
   - Was a `docs:` issue part of the plan? If so, is it closed?
   - Are `docs/` pages, Helm chart comments, or API reference accurate and up to date?

5. **Check for regressions** — did the implementation inadvertently break adjacent behavior?
   - Look for removed code that may be used elsewhere
   - Run `cd src && make test` to verify existing tests still pass

### Step 4 — Apply quality criteria

**Completeness** — did the implementation cover the entire scope?
- Every acceptance criterion is met, not just the easy ones
- If the proposal described multiple behaviors, all were implemented
- If a `docs:` issue was part of the plan, it was addressed

**Correctness** — is the implementation factually accurate?
- Field names, env var names, and type names match the code
- No placeholder text, TODO comments, or stubbed logic left behind
- The feature actually does what the issue described, not a superficially passing but semantically wrong implementation

**Quality** — is the code well-written?
- Follows existing patterns in the codebase (see `CLAUDE.md`)
- No dead code introduced
- Error paths handled where the codebase already handles them

### Step 5 — Score each issue

For each issue, make a clear judgement:

- **PASS** — all acceptance criteria met, tests present, no regressions, quality acceptable
- **PARTIAL** — implementation exists but is incomplete or shallow; some criteria unmet
- **FAIL** — implementation is wrong, missing, or introduces regressions

### Step 6 — Reopen failing and partial issues

For each PARTIAL or FAIL issue:

```bash
gh issue reopen <N> --repo language-operator/language-operator
gh issue comment <N> --repo language-operator/language-operator --body "..."
```

The comment must be specific and actionable:
- Quote the exact acceptance criterion that is unmet
- State precisely what needs to be added, changed, or fixed — reference the specific file and line if applicable
- Do not write vague feedback like "tests are missing"; say which behavior is untested and what the test should verify

### Step 7 — Summarise

Print a table:

| Issue | Title | Verdict | Reason (if PARTIAL/FAIL) |
|-------|-------|---------|--------------------------|
| #100  | ...   | PASS    |                          |
| #101  | ...   | PARTIAL | `spec.workspaceSize` validation not tested |
| #102  | ...   | FAIL    | Helm chart `values.yaml` not updated |

End with a count: X passed, Y reopened.
