---
description: Audit documentation against the current codebase for staleness, missing coverage, and broken references — then file GitHub issues for each finding
---

## Directions

You are a strict documentation reviewer auditing the Language Operator docs for accuracy, completeness, and consistency with the current codebase.

### Step 1 — Read sources directly

Read these files using the Read and Grep tools directly (do not delegate to a subagent — it is less accurate):

**Documentation** — read every file under `docs/`. For each, capture:
- Every field name, env var, config key, CLI flag, and resource name mentioned
- Every code example (YAML manifests, shell commands, Go snippets)
- Every claim about operator behaviour (e.g. "the operator injects X", "field Y controls Z")

**CRD types** — read every `src/api/v1alpha1/*_types.go`. Capture every:
- Spec and Status field name, type, and doc comment
- Enum values (`+kubebuilder:validation:Enum`)
- Default values (`+kubebuilder:default`)
- Required vs optional fields (`+kubebuilder:validation:Required` vs `omitempty`)

**Controllers** — skim `src/controllers/*_controller.go` and `src/controllers/utils.go`. Capture:
- Env vars injected into agent pods (name and source)
- ConfigMap keys and mount paths
- Resource names produced by `GenerateConfigMapName` and related helpers

**Helm chart** — read `chart/values.yaml` and `chart/templates/` to capture:
- All configurable values and their defaults
- Operator deployment flags/env vars

**Spec contracts** — read `spec/agents.md` and `spec/tools.md` for the runtime contracts agents must implement.

### Step 2 — Cross-reference for findings

**Stale field references (highest priority)**
- Field names in docs that have been renamed or removed from the CRD types
- Env var names in docs that differ from what the controller actually injects
- ConfigMap keys or mount paths that no longer match the controller

**Missing coverage**
- Spec fields (especially new ones added after initial docs were written) that appear in the types but are not documented in `docs/api/`
- Enum values that are valid in the schema but not mentioned in docs
- Helm chart values present in `values.yaml` but absent from `docs/helm/configuration.md`
- Runtime env vars injected by the operator that are not listed in the agent runtime contract docs

**Broken or misleading code examples**
- YAML examples referencing fields that don't exist or have wrong types
- Shell commands using flags or subcommands that no longer exist
- API version strings in examples that are outdated (e.g. `v1alpha1` vs current)

**Architecture drift**
- Claims in `docs/architecture/` about how components interact that no longer match the controller code
- References to removed resources or concepts (e.g. deleted CRDs, renamed controllers, removed proxy patterns)

**Internal link rot**
- Cross-references between doc pages pointing to sections or pages that have been renamed or removed

### Step 3 — Deduplicate against open issues

Before filing, run:
```
gh issue list --repo language-operator/language-operator --state open --limit 100
```

Skip any finding already covered by an open issue. Note the existing issue number.

### Step 4 — File issues

For each new finding, file a GitHub issue using `gh issue create --repo language-operator/language-operator`.

- Label stale/incorrect docs as `bug`
- Label missing coverage as `enhancement`
- Title format: `docs:` prefix for all issues
- Body must include: the problem, the doc file + line number, the current code truth, and a concrete fix (what the doc should say)
- Group closely related findings (same doc page, same root cause) into one issue

### Step 5 — Summarise

Print a table of all issues filed: issue number, title, label. Note any findings skipped due to existing open issues.
