# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project

Language Operator is a Kubernetes operator that runs AI agents as native workloads. Agents are container images — the operator handles lifecycle, configuration injection, networking, and observability. There is no code generation or synthesis.

See `requirements/ARCHITECTURE.md` for full design. See `spec/agents.md` and `spec/tools.md` for the runtime contracts agents and tools must implement.

## Development Commands

All Go work runs from `src/`:

```bash
cd src
make build       # compile operator binary
make test        # fmt + vet + all tests
make fmt         # go fmt
make vet         # go vet
make integration-test  # run envtest integration tests (requires setup-envtest)
```

Run a single test or package:
```bash
cd src && go test ./controllers/... -run TestLanguageAgentController -v
cd src && go test -tags integration -v ./controllers/...  # integration tests only
```

After modifying any type in `src/api/v1alpha1/`:
```bash
cd src && make generate   # regenerate zz_generated.deepcopy.go
cd src && make helm-crds  # regenerate CRD YAMLs and copy to charts/language-operator/templates/crds/
```

**The pre-commit hook will fail if generated files are modified but not staged.** Always stage `zz_generated.deepcopy.go`, `src/config/crd/bases/`, and `charts/language-operator/templates/crds/` together with type changes.

Helm chart validation:
```bash
cd charts/language-operator && helm lint .
cd charts/language-operator && helm template . --debug
cd charts/language-operator-runtimes && helm lint .
cd charts/language-operator-runtimes && helm template . --debug
```

Two charts live under `charts/`:
- `charts/language-operator/` — operator workload (CRDs, Deployment, RBAC, webhooks) **plus the `argo-workflows` subchart**, pulled from `oci://ghcr.io/argoproj/argo-helm` and enabled by default (`argo-workflows.enabled`). Agents run as Argo Workflows and the operator refuses to start without the `argoproj.io` CRDs. Run `helm dependency build charts/language-operator` before linting, templating, or installing from a checkout. Install this chart first.
- `charts/language-operator-runtimes/` — umbrella chart that pulls the four runtimes (openclaw, opencode, claude-code, deepagents) as subcharts from `oci://ghcr.io/language-operator/charts`. Requires the operator chart's CRDs to be present. Run `helm dependency build charts/language-operator-runtimes` before packaging/installing (CI and the `make` targets do this). `Chart.lock` is committed; pulled `charts/*.tgz` are gitignored.

Each runtime now lives in its **own repository** (image source **and** self-contained chart), not in this repo:
- `language-operator/claude-code-adapter` — combined terminal image + `claude-code` runtime chart
- `language-operator/openclaw-adapter` — adapter init image + `openclaw` runtime chart
- `language-operator/opencode-adapter` — adapter init image + `opencode` runtime chart

The umbrella's values are keyed by subchart name (e.g. `claude-code.enabled`, `claude-code.image.pullPolicy`), forwarded to each subchart.

Documentation:
```bash
cd src && make docs              # generate API reference markdown
make docs-serve                  # preview docs site at http://localhost:8000 (uv run mkdocs serve)
make docs-build                  # build static site to site/ (uv run mkdocs build --strict)
```

Docs dependencies are managed with [uv](https://docs.astral.sh/uv/) via `pyproject.toml` + `uv.lock`. `uv run` provisions the environment automatically; run `uv sync` to materialize `.venv` explicitly.

## Architecture

### Controllers (`src/controllers/`)

One controller per CRD. Each follows the same pattern:
- `StartReconcile` via `reconciler.ReconcileHelper` (handles fetch, span setup, deleted-resource short-circuit)
- Finalizer added on first reconcile; cleanup logic in `handleDeletion`
- ConfigMap reconciled via `CreateOrUpdateConfigMap` / `DeleteConfigMap` helpers in `utils_resource.go`
- Status updated last; `SetCondition` helper manages the conditions slice

Key controllers:
- `languageagent_controller.go` — main agent reconciler; drives the flow and derives status from Argo. Creates a `WorkflowTemplate` always, plus a long-lived `Workflow` (`spec.execution.mode: service`, the default) or a `CronWorkflow` (`mode: task` with a schedule). Also a NetworkPolicy, one ConfigMap (`config.yaml`), ServiceAccount/Role/RoleBinding (named `language-agent-<agent-name>`, namespace-scoped), optionally a PVC for `spec.workspace`, and — for service-mode agents only — a Service and Ingress. Cleanup logic in `cleanupResources`
- `languageagent_workflow.go` — builds the agent pod spec (`buildAgentPodSpec`) and renders it as Argo objects; also `syncWorkflowStatus`, which maps Workflow/CronWorkflow state onto the agent's status. This is the single podspec assembly point
- `languageagentruntime_controller.go` — reconciles status only; the agent controller merges runtime defaults via `merge.ApplyRuntimeDefaults`
- `languageagentselfconfig_controller.go` — applies agent-submitted self-configuration requests
- `languagecluster_controller.go` — reconciles the shared LiteLLM gateway (Deployment `gateway`, Service `gateway`, ConfigMap `gateway-config`) and optional Ingress/HTTPRoute at `gateway.<cluster.domain>`; watches LanguageModels to trigger re-reconcile when the model list changes
- `languagemodel_controller.go` — reconciles status only; no longer creates any ConfigMap, Deployment, or Service — the cluster controller reads LanguageModel CRs directly when building `gateway-config`
- `languagepersona_controller.go` — reconciles status only; the agent controller reads LanguagePersona CRs directly when building config.yaml
- `languagetool_controller.go` — validates tool image registry, reconciles tool Deployment/Service and NetworkPolicy

Shared utilities are split across `utils_resource.go` (`GenerateConfigMapName`, `GenerateServiceAccountName`, `GeneratePVCName`, `CreateOrUpdateOwned`, `CreateOrUpdateConfigMap`, `DeleteConfigMap`, `GetCommonLabels`), `utils_network.go`, `utils_status.go` (`SetCondition`, `SetPhase`), and `utils_security.go`. `FinalizerName` is in `constants.go`.

### CRDs (`src/api/v1alpha1/`)

- `LanguageAgent` — agent spec (image, instructions, personas, models, tools) plus `spec.execution` (`mode`, `schedule`, `timezone`, `concurrencyPolicy`, `activeDeadlineSeconds`, `ttlSecondsAfterFinished`, `retryLimit`, `suspend`). `spec.deployment.replicas` and `spec.deployment.autoscaling` are rejected by the webhook — an Argo Workflow has neither
- `LanguageAgentRuntime` — reusable agent defaults (image, spec.openclaw, spec.opencode, deployment config); merged into the agent's effective spec at reconcile time via `ApplyRuntimeDefaults`
- `LanguagePersona` — behavioral config (systemPrompt, tone, instructions, capabilities, constraints)
- `LanguageTool` — MCP tool server (serviceRef, port)
- `LanguageModel` — LLM endpoint config
- `LanguageCluster` — managed namespace; owns the shared LiteLLM gateway and optional external ingress at `gateway.<spec.domain>`

Webhooks live in `*_webhook.go` alongside the types. `zz_generated.deepcopy.go` is auto-generated — never edit by hand.

### Agent Configuration Injection

The operator mounts one file into every agent pod:
- `/etc/agent/config.yaml` — assembled from `spec.instructions`, referenced personas, resolved tool endpoints, model configs, and agent identity

Env vars injected: `AGENT_NAME`, `AGENT_NAMESPACE`, `AGENT_UUID`, `AGENT_CLUSTER_NAME`, `AGENT_CLUSTER_UUID`.

`MODEL_ENDPOINT` is the shared gateway URL (`http://gateway.<namespace>.svc.cluster.local:8000`) — one URL regardless of how many models are referenced. `LLM_MODEL` is a comma-separated list of model names from all `models`. Both are injected into the main container and all init containers. `MCP_SERVERS` contains resolved MCP tool server URLs.

NetworkPolicy allows any pod with label `langop.io/kind=LanguageAgent` to reach any other agent on the agent's ports (`spec.ports`, defaulting to one `http` port on 8080). The Service selector and NetworkPolicy podSelector match the operator-managed labels the Workflow stamps onto its pods via `podMetadata`, so user-supplied `spec.deployment.podLabels` cannot detach them.

Agent pods additionally get `create`/`patch` on `argoproj.io/workflowtaskresults` in their Role — the Argo executor reports each node's outcome that way, and a run fails at completion without it.

The shared gateway image (`ghcr.io/language-operator/model-gateway:latest`) is configured via `config.gateway.image` and `config.gateway.imagePullPolicy` in the Helm chart. For local development, `make dev` in `components/model-gateway/` builds and imports the image into k3s.

### Telemetry (`src/pkg/telemetry/`)

All reconciliation loops emit OpenTelemetry spans via `reconciler.ReconcileHelper`. `otel.go` bootstraps the global OTel tracer provider (`InitTracer`/`Shutdown`) called from `cmd/main.go`.

### Package Layout

- `pkg/events/` — Kubernetes event recording via `EventManager`; use its constants (`ReasonResourceCreated`, etc.) rather than raw strings
- `pkg/reconciler/` — shared `ReconcileHelper` used by all controllers
- `pkg/telemetry/` — OTel tracer bootstrap (`InitTracer`, `Shutdown`)
- `pkg/validation/` — image registry validation (`ValidateImageRegistry`)
- `pkg/network/` — NetworkPolicy helpers
- `internal/testutil/gen/` — fluent fixture builders for tests (`gen.LanguageAgent(name, ns, mods...)`)

### Test Infrastructure

Unit tests use `sigs.k8s.io/controller-runtime/pkg/client/fake` with a real scheme. Two reconcile calls are always needed: first adds the finalizer, second creates resources. Setup pattern:

```go
scheme := testutil.SetupTestScheme(t)  // controllers/testutil/scheme.go
fakeClient := fake.NewClientBuilder().WithScheme(scheme).WithObjects(obj).WithStatusSubresource(obj).Build()
reconciler := &LanguageAgentReconciler{Client: fakeClient, Scheme: scheme, Log: logr.Discard(), ...}
```

Integration tests use `//go:build integration` tag and run against a real envtest API server (see `suite_test.go`). The `LanguageAgentReconciler` has additional required fields beyond `Client`/`Scheme`: `Recorder`, `EventManager`, `RegistryManager`, `NetworkIsolationEnabled`, `DefaultIngressClassName`. Use `mockRegistryManager` from the disabled test file as a reference.

## Critical Rules

**No mock data.** Features must work with real Kubernetes APIs before commit.

**Generated files must be staged.** Any change to `src/api/v1alpha1/` requires running `make generate && make helm-crds` and staging the output before committing.

**Conventional commits.** Use `feat:`, `fix:`, `chore:`, `docs:`, `test:` prefixes. Use `WIP:` for partial implementations. PR titles must also follow this convention (enforced by CI). Note: `clean:` is rejected by CI — use `chore:` instead.

**Local development deploys via `make dev`** (project root) — builds the Go binary, builds a Docker image tagged with the current git SHA, imports it into k3s, resolves both charts' dependencies, and helm-upgrades the operator and runtimes releases (5m timeout on the operator, which also brings up Argo Workflows), then restarts and waits on the operator rollout. Commit changes before running `make dev` so the git SHA changes and Docker cache is busted.

**`make wipe` resets the cluster** — removes all CRs (stripping finalizers if the operator is already gone), both Helm releases, the `langop.io` and `argoproj.io` CRDs, the namespace, orphaned cluster-scoped resources from a previously broken uninstall, and the dev images in k3s. It deletes the admission webhooks first: they have `failurePolicy: Fail` and point at a Service in the operator namespace, so once the operator is gone they reject every write to a `langop.io` resource — including the patch that removes a finalizer.
