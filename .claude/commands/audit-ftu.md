---
description: Audit the first-time user experience of installation and setup — walk through every step a new user would take and identify friction, broken steps, and missing prerequisites
---

## Directions

You are a new user who has just discovered Language Operator. You have a working Kubernetes cluster (k3s/k8s 1.26+), `kubectl`, `helm`, and an Anthropic API key. You have never used this project before. Walk through the entire setup flow and audit every step for correctness, completeness, and UX quality.

### Step 1 — Read the entry points

Read these in order — this is exactly what a new user reads:

1. `README.md` — the full file
2. `docs/getting-started/installation.md`
3. `docs/getting-started/quickstart.md`
4. `docs/getting-started/examples.md`

For each file capture:
- Every shell command a user would run
- Every YAML manifest they would apply
- Every prerequisite assumed but not stated
- Every step that references an external resource (image, secret, URL)

### Step 2 — Validate the Helm chart path

Check that the Helm install path works end-to-end:

```
helm repo add language-operator https://language-operator.github.io/language-operator
helm install language-operator language-operator/language-operator
```

Read `chart/values.yaml` and `chart/templates/` to verify:
- CRDs are installed with the chart (check `chart/templates/crds/`)
- Standard runtimes are bundled and enabled by default (`runtimes.openclaw.enabled`, `runtimes.opencode.enabled`)
- RBAC is correct for creating Secrets (needed for inline credential injection)
- The operator image and gateway image defaults are valid (not empty, not localhost)
- Any `chart/templates/runtimes/` templates render correctly with default values (`helm template chart/`)

### Step 3 — Walk the Getting Started flow step by step

Simulate being a user following the README "Getting Started" section. For each step:

**Step: Create a LanguageCluster**
- Does the YAML apply cleanly? (check the CRD exists in the chart)
- Does the namespace get created? (check cluster controller behaviour)
- Is `kubectl config set-context` included so subsequent commands don't need `-n`?

**Step: Configure an LLM**
- Is the secret creation command correct?
- Is the LanguageModel YAML complete and valid against the CRD (`src/api/v1alpha1/languagemodel_types.go`)?
- Does `provider: anthropic` actually work, or is there an enum constraint to check?

**Step: Deploy an agent**
- Do the inline credential fields (`spec.openclaw.token`, `spec.opencode.username/password`) exist in the CRD? Check `src/api/v1alpha1/languageagent_types.go`
- Is `runtime: openclaw` valid? Does the CRD accept a `runtime` field?
- Are the runtimes guaranteed to exist in-cluster when the user applies the agent? (they're bundled — confirm they install before the agent reconciles)
- Does the `{agent}-runtime` Secret get created correctly? Trace through `reconcileRuntimeSecret` in `src/controllers/languageagent_controller.go`

**Step: Connect**
- Is the port-forward command targeting the right service name and port?
- For opencode: is the browser URL correct? Does opencode actually serve a UI at `/`?
- For openclaw: does port 18789 serve a browser UI, or is it a WebSocket-only port?

### Step 4 — Check for silent failure modes

Look for steps that would fail without a clear error message:

- Missing CNI (NetworkPolicy silently does nothing on vanilla k8s without Cilium/Calico)
- Image pull failures (are any images private or not yet pushed?)
- Gateway not ready when agent reconciles (race condition? does the operator handle it?)
- `OPENCODE_SERVER_PASSWORD` env var — does opencode actually enforce auth, or is it optional?
- Webhook admission errors if CRD validation is strict and examples have missing required fields

Read `src/controllers/languageagent_controller.go` around the runtime resolution block and the `reconcileRuntimeSecret` function to check error handling and status reporting.

### Step 5 — Check for missing prerequisites

Identify anything a new user needs that is not mentioned:

- CNI requirements (NetworkPolicy enforcement)
- Storage class (workspace PVC — does it need a specific StorageClass?)
- Gateway API CRDs (if HTTPRoute is used — does the chart install them?)
- Minimum resource requirements (does the gateway/agent fit on a single-node k3s?)
- Any firewall or egress requirements (e.g. Anthropic API access on port 443)

### Step 6 — Deduplicate against open issues

Before filing, run:
```
gh issue list --repo language-operator/language-operator --state open --limit 100
```

Skip any finding already covered by an open issue. Note the existing issue number.

### Step 7 — File issues

For each new finding, file a GitHub issue using `gh issue create --repo language-operator/language-operator`.

- Broken or wrong steps → label `bug`
- Missing prerequisite or unclear step → label `enhancement`  
- Title format: `ftu:` prefix for all issues (first-time user)
- Body must include: the exact step that fails, what the user would see, and a concrete fix
- Order issues by severity: things that block setup entirely first

### Step 8 — Summarise

Print a table: issue number, title, severity (blocker / friction / polish). Note any findings skipped due to existing open issues.
