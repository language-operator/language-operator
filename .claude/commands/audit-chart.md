---
description: Audit the Helm chart for sync with operator code — values keys, CLI flags, CRD versions, template correctness, and chart hygiene — then file GitHub issues for each finding
---

## Directions

You are a strict Helm chart reviewer auditing the Language Operator chart for correctness, completeness, and sync with the operator binary and CRD types.

### Step 1 — Read sources directly

Read these files using the Read and Grep tools directly (do not delegate to a subagent — it is less accurate):

**Chart** — read every file under `charts/language-operator/`:
- `charts/language-operator/Chart.yaml` — the `dependencies` block (currently the `argo-workflows` subchart, pulled OCI from `ghcr.io/argoproj/argo-helm`) and whether `Chart.lock` is in step with it
- `charts/language-operator/values.yaml` — every key, its type, and its default value, including the `argo-workflows.*` passthrough values
- `charts/language-operator/templates/deployment.yaml` — every `--flag` passed to the operator binary; every `.Values.*` reference
- `charts/language-operator/templates/configmap.yaml` — any operator config injected via ConfigMap
- `charts/language-operator/templates/clusterrole.yaml` — RBAC rules, which must cover everything the operator both uses and *grants* (RBAC forbids granting a permission the granter lacks)
- `charts/language-operator/templates/_helpers.tpl` — naming helpers
- Other templates (service, webhook, hpa, pdb, etc.) for `.Values.*` references and hardcoded values
- `charts/language-operator/templates/crds/` — `apiVersion`, `kind`, field names, and enum values in the bundled CRD YAMLs

Also check `charts/language-operator-runtimes/`, the umbrella chart for the four runtime subcharts.

**Operator source**:
- `src/controllers/*_controller.go` — every `--flag` the binary actually reads (look for `flag.String`, `flag.Bool`, `flag.Int`, and `ctrl.Options` fields wired from flags in `main.go`)
- `src/main.go` — the authoritative list of CLI flags and their defaults
- `src/api/v1alpha1/*_types.go` — CRD field names, enum values, and defaults that must match the bundled CRDs

**Generated CRDs** — compare `src/config/crd/bases/` against `charts/language-operator/templates/crds/` — they must be identical.

### Step 2 — Cross-reference for findings

**Flag drift (highest priority)**
- Flags referenced in `charts/language-operator/templates/deployment.yaml` that do not exist in `src/cmd/main.go`
- Flags in `src/main.go` that are useful to configure but have no corresponding `values.yaml` key or template reference
- Flag default values in the chart that differ from `src/main.go` defaults

**Values key problems**
- `.Values.*` references in templates that have no corresponding key in `values.yaml` (would panic on `helm template` with `--strict`)
- Keys in `values.yaml` that are never referenced in any template (dead values)
- Keys whose type in `values.yaml` doesn't match how the template uses them (e.g. a string used in an `if` check that expects a bool)
- Documented defaults in `docs/helm/configuration.md` that differ from actual `values.yaml` defaults

**CRD sync issues**
- Any field, enum value, or default present in `src/config/crd/bases/` but absent or different in `charts/language-operator/templates/crds/`
- Stale CRD resources in `charts/language-operator/templates/crds/` that no longer have a corresponding `*_types.go` (e.g. `langop.io_languageagentversions.yaml` — verify it is still a live type)
- `apiVersion` or `kind` mismatches between the chart CRDs and the generated ones

**RBAC completeness**
- Resources the controllers create/update/delete (from `//+kubebuilder:rbac` markers in controllers) that are absent from `charts/language-operator/templates/clusterrole.yaml`
- Rules in `charts/language-operator/templates/clusterrole.yaml` that no controller marker declares (over-permissioned)

**Template correctness**
- Hardcoded image tags (`:latest` or specific SHA) that should be driven by `values.yaml`
- Missing `quote` or `toYaml` pipeline calls that would produce invalid YAML for certain input values
- Probe ports hardcoded in templates that differ from what `values.yaml` configures
- Service port in the template that doesn't match `values.yaml.service.port`
- `namespaceSelector` or other selectors referencing a hardcoded namespace that should be the release namespace

**Chart hygiene**
- `charts/language-operator/Chart.yaml` `appVersion` that is stale or inconsistent with how the image tag is resolved
- Missing or incorrect `helm.sh/chart` labels on resources
- Resources that are never cleaned up on `helm uninstall` (no owner reference or hook)
- `values.yaml` keys with empty-string defaults that would silently produce broken config if left unset (should have validation or a `required` call in the template)

### Step 3 — Deduplicate against open issues

Before filing, run:
```
gh issue list --repo language-operator/language-operator --state open --limit 100
```

Skip any finding already covered by an open issue. Note the existing issue number.

### Step 4 — File issues

For each new finding, file a GitHub issue using `gh issue create --repo language-operator/language-operator`.

- Label broken/incorrect chart behaviour as `bug`
- Label missing values, dead keys, and hygiene issues as `enhancement`
- Title format: `chart:` prefix for all issues
- Body must include: the problem, the template or values file + line, the source-of-truth (flag name in main.go, type in *_types.go, etc.), and a concrete fix
- Group closely related findings (same template, same root cause) into one issue

### Step 5 — Summarise

Print a table of all issues filed: issue number, title, label. Note any findings skipped due to existing open issues.
