---
description: Audit CRDs and controllers for inconsistencies, dead fields, and unimplemented features — then file GitHub issues for each finding
---

## Directions

You are a strict API reviewer auditing the Language Operator CRDs against their controllers.

### Step 1 — Read sources directly

Read these files using the Read and Grep tools directly (do not delegate to a subagent — it is less accurate):

**Types** — read every `src/api/v1alpha1/*_types.go` file. For each, capture:
- Every field in every Spec, Status, and nested struct
- Enum constraints (`+kubebuilder:validation:Enum`)
- Default values (`+kubebuilder:default`)
- Printcolumn annotations (`+kubebuilder:printcolumn`)
- `bool` fields that carry `omitempty` — flag any that also have `+kubebuilder:default=true` (these cannot be set to false; they need `*bool`)

**Controllers** — read every `src/controllers/*_controller.go`. For each, capture:
- Every `status.` field written
- Every `spec.` field read (including reads inside helper methods, not just `Reconcile`)
- Phase string literals written to `status.Phase`
- TODO / FIXME comments

**Runtime merge logic** — read `src/api/v1alpha1/languageagentruntime_merge.go`. This is where `LanguageAgentRuntimeSpec` fields are consumed via `ApplyRuntimeDefaults`. A field in `LanguageAgentRuntimeSpec.Deployment` is only dead if it appears in neither this file nor any controller. Pay special attention to fields in `DeploymentSpec` that `ApplyRuntimeDefaults` does **not** handle (e.g. `ServiceAccountAnnotations`, `RoleRules`) — these are potential dead fields when used via a runtime.

**Phase constants** — read `src/pkg/events/manager.go`. Phase values are defined as constants here (`PhaseStatusPending`, `PhaseStatusRunning`, `PhaseStatusReady`, `PhaseStatusFailed`, `PhaseStatusUpdating`, `PhaseStatusDegraded`). Controllers reference these constants, not raw string literals. When cross-referencing phase values written vs. enum constraints on the type, look up these constants rather than searching for inline strings.

**Component scripts** — also read `components/model-gateway/generate-config.py`. Some spec fields are consumed here rather than in Go controllers (e.g. `LanguageModel.Spec` fields flow through to LiteLLM config). A field is not dead just because no controller reads it if a component script does.

**Shared types** — `DeploymentSpec` is used by LanguageAgent, LanguageTool, and LanguageCluster controllers. A field is only dead if *no* controller reads it. Check all three before marking a field dead.

**Known intentional patterns** (do not flag these as bugs):
- `LanguageAgentRuntime` has no `+kubebuilder:subresource:status` and no `Status` field. Its controller only manages a finalizer. This is by design — it is a cluster-scoped preset, not a reconciled workload.
- `SelfConfigPhase` is a named Go type with an enum constraint on the type declaration rather than on the struct field. Controllers write all four values (Pending, Applied, Failed, Denied).

### Step 2 — Cross-reference for findings

**Enum violations (highest priority)**
- Phase values written by the controller that are not in the type's `+kubebuilder:validation:Enum`
- Enum values declared in the type that no controller ever writes (dead enum values inflate the schema and mislead users)

**Dead spec fields**
- Fields defined in Spec (including nested structs) that no controller, merge function, or component script reads
- `bool` fields with `omitempty` and `+kubebuilder:default=true` anywhere in the type hierarchy — these must be `*bool` or the user can never set them to false
- `DeploymentSpec` fields that appear in `LanguageAgentRuntimeSpec.Deployment` but are not merged by `ApplyRuntimeDefaults` — silently ignored at runtime despite being valid in `LanguageAgentSpec.Deployment`

**Dead status fields and printcolumns**
- Status fields that no controller ever writes
- `+kubebuilder:printcolumn` entries pointing at status fields that are never written (they always show `<none>`)

**Vestigial types**
- Structs defined in `*_types.go` that are not referenced by any current field, controller, or component script

**Unimplemented features**
- Enum values with no corresponding controller code path
- Fields whose doc comment describes behaviour the controller does not perform
- TODO comments in controllers

**Logical inconsistencies**
- Fields that belong to the wrong layer (e.g. `BackoffLimit` is a Job field, not a Deployment field)
- Deprecated fields with no migration path
- Inconsistent optionality within a struct (mix of pointer/non-pointer without clear reason)
- Terminology drift — field names, status fields, ConfigMap names, and condition strings should use consistent domain vocabulary throughout

### Step 3 — Deduplicate against open issues

Before filing, run:
```
gh issue list --repo language-operator/language-operator --state open --limit 100
```

Skip any finding that is already covered by an open issue. Note the existing issue number in your working notes.

### Step 4 — File issues

For each new finding, file a GitHub issue using `gh issue create --repo language-operator/language-operator`.

- Label bugs (enum violations, type bugs, dead printcolumns) as `bug`
- Label dead code and naming issues as `enhancement`
- Label unimplemented features as `enhancement`
- Title format: `fix:` for bugs, `chore:` for dead code/naming, `feat:` for missing implementations
- Body must include: the problem, affected file + line numbers, and a concrete fix or set of options
- Group tightly related findings (same root cause, same struct) into one issue

### Step 5 — Summarise

Print a table of all issues filed: issue number, title, label. Also note any findings skipped because they matched existing open issues.
