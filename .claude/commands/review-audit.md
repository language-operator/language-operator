---
description: Review recently closed audit issues for implementation quality — reopen with comments if work is incomplete or sloppy
---

## Directions

You are a strict QA reviewer. Your job is to verify that recently closed issues were resolved **completely and correctly**, not just superficially. Sloppy or partial fixes get reopened.

### Step 1 — Identify the issues from this session's audit

**Do not query all recently closed issues.** This command reviews only the specific issues filed by the most recent audit command (`/audit-docs`, `/audit-crds`, `/audit-chart`, `/audit-operator`, or `/audit-tests`) run in this conversation.

Look back at the conversation history and extract the issue numbers that were filed. They appear in the audit's Step 5 summary table, or as `https://github.com/language-operator/language-operator/issues/<NUMBER>` URLs in the output.

For each issue number, fetch the full body:
```
gh issue view <NUMBER> --repo language-operator/language-operator
```

Only review those issues. If none have been closed yet, say so and exit.

### Step 2 — For each closed issue, verify the fix

For each issue:

1. **Parse the issue body** — extract:
   - The problem statement
   - Affected files and line numbers (if specified)
   - The "Fix" section describing what should change

2. **Find how it was fixed** — run:
   ```
   gh issue view <NUMBER> --repo language-operator/language-operator --comments
   ```
   Look for a closing PR reference in the comments or timeline. If found, inspect the diff:
   ```
   gh pr diff <PR> --repo language-operator/language-operator
   ```
   If no PR is linked, scan `git log --oneline -20` for a commit message matching the issue title.

3. **Read the current state of affected files** — use the Read and Grep tools to verify:
   - Every file mentioned in the issue was actually updated
   - The specific problem described no longer exists
   - The fix matches what the issue prescribed (not just "something was changed")

4. **Apply these quality criteria:**

   **Completeness** — did the fix address the *entire* issue, or only part of it?
   - If the issue said "update files A, B, and C", were all three updated?
   - If the issue listed multiple enum values or fields to add, were all of them added?
   - If the issue said "add a cross-reference pointing to X", does the cross-reference actually resolve?

   **Correctness** — is the fix factually accurate?
   - Field names, env var names, and type names must exactly match the code
   - YAML examples must be valid and reference real fields
   - Code truth claims (e.g. "the controller injects X") must still be true in the current code

   **Quality** — is the fix well-written?
   - No copy-paste errors or leftover placeholder text
   - Tables are properly formatted markdown
   - New content is consistent in style and terminology with the surrounding docs
   - No regressions: the fix did not accidentally break or remove other correct content

   **Depth** — for enhancement issues (missing coverage), does the fix actually teach the reader something useful?
   - A one-sentence addition where the issue asked for a field reference table is insufficient
   - Examples should be runnable, not contrived

### Step 3 — Score each issue

For each issue, make a clear judgement:

- **PASS** — fix is complete, correct, and of acceptable quality
- **PARTIAL** — fix addressed the issue but is incomplete or shallow; needs more work
- **FAIL** — fix is wrong, missing, or introduces new problems

### Step 4 — Reopen failing and partial issues

For each PARTIAL or FAIL issue:

```
gh issue reopen <NUMBER> --repo language-operator/language-operator
gh issue comment <NUMBER> --repo language-operator/language-operator --body "..."
```

The comment must be specific and actionable — do not write vague feedback like "this needs improvement". Instead:
- Quote the exact line or section that is wrong or missing
- State precisely what needs to be added, changed, or removed
- If a field was documented incorrectly, give the correct value
- If a section was skipped, explain what it should contain

### Step 5 — Summarise

Print a table:

| Issue | Title | Verdict | Reason (if PARTIAL/FAIL) |
|-------|-------|---------|--------------------------|
| #123  | ...   | PASS    |                          |
| #124  | ...   | PARTIAL | Missing `dns` field example; only `group` was added |
| #125  | ...   | FAIL    | File `docs/api/reference.md` was not updated |

End with a count: X passed, Y reopened.
