---
description: Audit the Go code and operator design for convention violations, hygiene issues, and Kubernetes best-practice gaps — then file GitHub issues for each finding
---

## Directions

You are a strict Go and Kubernetes operator reviewer auditing the Language Operator codebase for correctness, convention, and design quality. You are not auditing CRD schemas (audit-crds), Helm chart sync (audit-chart), test coverage (audit-tests), or documentation (audit-docs) — those are handled by other commands. Your scope is the **Go implementation**: controller patterns, error handling, logging, resource lifecycle, observability, and code hygiene.

### Step 1 — Read sources directly

Read these files using the Read and Grep tools directly (do not delegate to a subagent — it is less accurate):

**Controllers** — read every `src/controllers/*_controller.go` and `src/controllers/utils.go`. For each, capture:
- How errors are returned and wrapped (bare `err` vs `fmt.Errorf("...: %w", err)`)
- Every `return ctrl.Result{}, nil` after a non-terminal error (silent drop vs proper requeue)
- Every `return ctrl.Result{Requeue: true}` or `RequeueAfter` — are the intervals hard-coded magic numbers?
- Logging calls: key-value pairs, log levels, whether `err` is always included in error logs
- Every Kubernetes resource created without an owner reference (orphaned resources survive parent deletion)
- Finalizer registration and removal — is it symmetric? Is cleanup guarded against partial failure?
- Every place `status.Phase` is written — is it written last, after all conditions are set?
- Event recording — are events fired for every meaningful state transition?
- `deleteAndVerifyResource` or any polling loop inside a reconcile path (blocks the worker goroutine)

**Shared packages** — read:
- `src/pkg/reconciler/helper.go` and `manager.go` — understand what `ReconcileHelper` provides
- `src/pkg/events/` — what event reasons/types are defined; are raw strings used anywhere in controllers instead of these constants?
- `src/pkg/network/policy.go` — NetworkPolicy construction patterns
- `src/pkg/validation/image_validator.go` — how registry validation errors are surfaced

**Main entrypoint** — read `src/main.go` (or `src/cmd/` if present):
- Leader election enabled?
- Metrics endpoint configured?
- Health/readiness probes registered?
- Graceful shutdown handled?

**Go module** — read `src/go.mod`:
- Note the Go version and key dependency versions (controller-runtime, client-go)

### Step 2 — Cross-reference for findings

Work through each area below. For each finding, note the file path and line number.

**Error handling (highest priority)**
- Errors returned without wrapping context: `return ctrl.Result{}, err` with no `fmt.Errorf` — makes stack traces useless in logs
- Errors silently dropped: `if err != nil { return ctrl.Result{}, nil }` where the error should cause a requeue
- `err` variable shadowing inside blocks (`:=` in inner scope masks outer `err`)
- `errors.Is` / `errors.As` used correctly for sentinel and typed errors (not string comparison)
- `apierrors.IsNotFound` used to distinguish missing resource from API error (not a bare `err != nil`)

**Logging conventions**
- Log calls missing structured key-value pairs (positional strings instead of `"key", value` pairs)
- Error logs missing `"error", err` key — raw `err.Error()` string interpolated into the message instead
- Debug-level logs that belong at info level, or vice versa (e.g. logging every reconcile entry at Info creates log spam)
- Missing log entry at reconcile start or end (hard to correlate logs for a single reconcile)
- Inconsistent field names for the same concept across controllers (e.g. `"agent"` vs `"name"` vs `"resource"`)

**Resource lifecycle**
- Resources created without `ctrl.SetControllerReference` — these will not be garbage-collected when the parent CR is deleted
- Owner references set on cross-namespace resources (invalid in Kubernetes — owner and owned must be in the same namespace)
- Finalizer added but never removed on the deletion path (resource stuck in terminating forever)
- Finalizer removed before cleanup is complete (resource deleted before side effects are undone)
- `CreateOrUpdate` used where a pure `Create` would be safer (avoids accidental overwrites of fields managed elsewhere)

**Requeue and reconcile loop design**
- `ctrl.Result{Requeue: true}` without `RequeueAfter` — busy-loops at maximum rate; should always have a back-off duration for non-error retries
- Magic number durations (`time.Second * 30`) — should be named constants or configurable flags
- Long-running work (HTTP calls, polling) done synchronously inside `Reconcile` — should be async or rate-limited
- Missing rate limiting on external API calls (e.g. LiteLLM, image registry) that could overwhelm downstream

**Status and conditions**
- Status written before all sub-resources are reconciled (partial status visible to users and other controllers)
- Condition `LastTransitionTime` not updated when `Status` changes (condition appears stale)
- `ObservedGeneration` not set on status (controllers watching this CR can't tell if status is current)
- Phase set to a terminal value (e.g. `"Ready"`) without a corresponding `Ready=True` condition — phase and conditions should agree
- Condition `Reason` fields using spaces or non-CamelCase (violates Kubernetes API conventions)

**Event recording**
- State transitions (resource created, updated, deleted, error) with no corresponding event — reduces operator debuggability
- Events fired using raw string literals instead of `pkg/events` constants
- Events recorded with `Warning` type for non-error conditions, or `Normal` type for errors

**Kubernetes operator best practices**
- Watches not set up for owned resources — if a Deployment is deleted out-of-band, the controller won't reconcile
- `Owns()` used for resources the controller doesn't set an owner reference on (watch won't fire correctly)
- Controller not watching referenced objects (e.g. LanguageAgent references LanguageModel but doesn't watch it — changes to LanguageModel won't trigger LanguageAgent reconcile)
- RBAC markers (`//+kubebuilder:rbac`) missing for resources the controller creates, updates, or deletes
- Over-broad RBAC (`verbs: ["*"]`) where specific verbs would suffice

**Go code hygiene**
- Exported functions, types, and methods without a doc comment (`//` line immediately above the declaration)
- `TODO` / `FIXME` / `HACK` comments in production code without an associated issue reference
- Constants that should be `iota` or typed string constants defined as bare `var` or repeated literals
- Receiver naming inconsistency within a type (e.g. `r` in one method, `rec` in another)
- Functions longer than ~80 lines that are candidates for extraction
- Identical error-handling blocks copy-pasted across controllers that should be in a shared helper

**Concurrency safety**
- Shared mutable state (maps, slices) accessed from `Reconcile` without synchronisation — `Reconcile` can be called concurrently for different objects
- `sync.Mutex` or `sync.Map` used where `controller-runtime`'s `Manager` cache would be the correct pattern

### Step 3 — Deduplicate against open issues

Before filing, run:
```
gh issue list --repo language-operator/language-operator --state open --limit 100
```

Skip any finding already covered by an open issue. Note the existing issue number in your working notes.

### Step 4 — File issues

For each new finding, file a GitHub issue using `gh issue create --repo language-operator/language-operator`.

- Label incorrect behaviour (silent error drops, broken finalizer lifecycle, invalid owner refs) as `bug`
- Label convention violations, hygiene, and non-critical design issues as `enhancement`
- Title format: `fix:` for bugs, `clean:` for hygiene/convention, `refactor:` for design improvements
- Body must include: the problem, affected file + line numbers, why it matters (e.g. "resource leaks on deletion"), and a concrete fix or set of options
- Group tightly related findings (same pattern repeated across multiple controllers) into one issue

### Step 5 — Summarise

Print a table of all issues filed: issue number, title, label. Note any findings skipped because they matched existing open issues.
