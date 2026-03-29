---
description: Audit the test suite for gaps, dead tests, and missing coverage against controllers and CRDs — then file GitHub issues for each finding
---

## Directions

You are a strict test reviewer auditing the Language Operator test suite for coverage gaps, dead tests, and quality issues.

### Step 1 — Inventory what exists

Read these files using the Read and Grep tools directly (do not delegate to a subagent):

**Test files** — read every `src/controllers/*_test.go` and `src/api/v1alpha1/*_test.go`. For each test function, capture:
- What behaviour or code path it exercises
- What it asserts (what fields/conditions it checks)
- Whether it calls `Reconcile` the correct number of times (first call adds finalizer, second creates resources)

**Controllers** — skim `src/controllers/*_controller.go` to identify every meaningful code path:
- Each early-exit error branch (registry validation, ConfigMap, PVC, NetworkPolicy, Service, Deployment)
- Each status field written (`phase`, `conditions`, `activeReplicas`, `readyReplicas`, `uuid`, `webhookURLs`, `endpoint`, etc.)
- Each condition type set (`Ready`, `RegistryValidated`, `NetworkPolicyReady`, `GatewayReady`, etc.)
- Each env var injected into pods
- Each Kubernetes resource created (Deployment, Service, ConfigMap, NetworkPolicy, Ingress, PVC, ServiceAccount, ClusterRoleBinding, ResourceQuota)

**Generator helpers** — read `src/internal/testutil/gen/` to understand what fixture builders exist, since missing builders often indicate missing test coverage.

### Step 2 — Cross-reference for findings

Work through each controller and check its test file against the inventory above.

**Missing test cases (highest priority)**
- Error paths that have no test: if a controller early-exits on error X, there should be a test that injects error X and asserts the correct condition/phase is set
- Happy-path gaps: resources the controller creates that are never asserted in any test (e.g. a test reconciles but never checks that the NetworkPolicy was created)
- Status field gaps: status fields written by the controller that no test ever reads back and asserts

**Dead or vacuous tests**
- Tests that call `Reconcile` but assert nothing meaningful (no `Get`, no field checks)
- Tests that always pass regardless of the code under test (e.g. only check `err == nil` but never inspect the created resource)
- Tests whose name describes behaviour X but the body tests behaviour Y

**Reconcile call count errors**
- Tests that only call `Reconcile` once and then assert resources were created — the first call adds the finalizer and returns, resources are created on the second call. Single-call tests for resource creation will always fail or produce false positives.

**Fixture builder gaps**
- Spec fields that exist on the CRD but have no corresponding option in the `gen.*` builders, making it impossible to construct test fixtures for those fields without boilerplate

**Integration vs unit confusion**
- Tests tagged `//go:build integration` that could be unit tests (no envtest dependency)
- Tests that import `fake.NewClientBuilder` but also try to test real Kubernetes behaviour that only works with a real API server

**Assertion quality**
- Tests that create a resource and only assert it exists (`IsNotFound` check) but never assert its spec/content is correct
- Tests that check a condition is set but not its `Status`, `Reason`, or `Message`
- Phase transition tests that only assert the final phase, not that intermediate phases were correct

### Step 3 — Deduplicate against open issues

Before filing, run:
```
gh issue list --repo language-operator/language-operator --state open --limit 100
```

Skip any finding already covered by an open issue.

### Step 4 — File issues

For each new finding, file a GitHub issue using `gh issue create --repo language-operator/language-operator`.

- Label missing coverage as `enhancement`
- Label dead/vacuous tests as `enhancement`
- Label incorrect tests (wrong reconcile count, always-pass) as `bug`
- Title format: `test:` prefix for all issues
- Body must include: the gap, affected test file + function name (or "missing"), the controller behaviour being untested, and a concrete example of what the test should assert
- Group related gaps for the same controller into one issue where they share the same root cause

### Step 5 — Summarise

Print a table of all issues filed: issue number, title, label. Note any findings skipped due to existing open issues.
