# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project

Language Operator is a Kubernetes operator that runs AI agents as native workloads. Agents are container images — the operator handles lifecycle, configuration injection, networking, and task observability. There is no code generation or synthesis.

See `requirements/ARCHITECTURE.md` for full design. See `spec/agents.md` and `spec/tools.md` for the runtime contracts agents and tools must implement.

## Development Commands

All Go work runs from `src/`:

```bash
cd src
make build       # compile operator binary
make test        # fmt + vet + all tests
make fmt         # go fmt
make vet         # go vet
```

Run a single test package:
```bash
cd src && go test ./controllers/... -run TestReconcile -v
```

After modifying any type in `src/api/v1alpha1/`:
```bash
cd src && make generate   # regenerate zz_generated.deepcopy.go
cd src && make helm-crds  # regenerate CRD YAMLs and copy to chart/crds/
```

**The pre-commit hook will fail if generated files are modified but not staged.** Always stage `zz_generated.deepcopy.go`, `src/config/crd/bases/`, and `chart/crds/` together with type changes.

Helm chart validation:
```bash
cd chart && helm lint .
cd chart && helm template . --debug
```

## Architecture

### Controllers (`src/controllers/`)

One controller per CRD. Each follows the same pattern:
- `StartReconcile` via `reconciler.ReconcileHelper` (handles fetch, span setup, deleted-resource short-circuit)
- Finalizer added on first reconcile; cleanup logic in `handleDeletion`
- ConfigMap reconciled via `CreateOrUpdateConfigMap` / `DeleteConfigMap` helpers in `utils.go`
- Status updated last; `SetCondition` helper manages the conditions slice

Key controllers:
- `languageagent_controller.go` — main agent reconciler; creates Deployment, Service, HTTPRoute, NetworkPolicy, two ConfigMaps (instructions + config)
- `languagepersona_controller.go` — creates a ConfigMap with the persona's JSON spec for agent mounting
- `languagetool_controller.go` — validates tool image registry, reconciles tool Deployment/Service
- `languageagenttask_controller.go` — *(to be implemented)* watches LanguageAgentTask; sends `POST /messages` to unblock paused tasks

Shared utilities in `utils.go`: `GenerateConfigMapName(name, suffix)`, `CreateOrUpdateConfigMap`, `DeleteConfigMap`, `SetCondition`, `FinalizerName`.

### CRDs (`src/api/v1alpha1/`)

- `LanguageAgent` — agent deployment spec (image, instructions, instructionsFrom, personaRefs, modelRefs, toolRefs, executionMode)
- `LanguageAgentTask` — in-flight A2A task state (agentRef, taskId, contextId; status mirrors A2A states: submitted/working/input-required/auth-required/completed/failed/canceled/rejected)
- `LanguagePersona` — behavioral config (systemPrompt, tone, instructions, capabilities, constraints)
- `LanguageTool` — MCP tool server (serviceRef, port)
- `LanguageModel` — LLM endpoint config
- `LanguageCluster` — multi-cluster grouping

Webhooks live in `*_webhook.go` alongside the types. `zz_generated.deepcopy.go` is auto-generated — never edit by hand.

### Agent Configuration Injection

The operator mounts two files into every agent pod:
- `/etc/agent/instructions.txt` — from `spec.instructions` (inline) or `spec.instructionsFrom` (ConfigMap/Secret ref)
- `/etc/agent/config.yaml` — assembled from referenced personas, resolved tool endpoints, model configs, and agent identity

Env vars injected: `AGENT_NAME`, `AGENT_NAMESPACE`, `AGENT_UUID`, `AGENT_MODE`, `AGENT_CLUSTER_NAME`, `AGENT_CLUSTER_UUID`, `AGENT_OPERATOR_WEBHOOK_URL`, `AGENT_OPERATOR_WEBHOOK_TOKEN`.

### A2A Protocol

Agents implement Google's A2A protocol on port 8080. The operator registers itself as a push notification subscriber per agent on startup (`AGENT_OPERATOR_WEBHOOK_URL`). When an agent reports `input-required` or `auth-required` state, the operator creates/updates a `LanguageAgentTask` CR — these are pauses, not failures.

NetworkPolicy allows any pod with label `langop.io/kind=LanguageAgent` to reach any other agent on port 8080.

### Telemetry (`src/pkg/telemetry/`)

All reconciliation loops emit OpenTelemetry spans via `reconciler.ReconcileHelper`. The ClickHouse adapter (`adapters/clickhouse.go` and `adapters/signoz.go`) queries `otel_traces` and `otel_metrics` tables. No mock data — features must work with real ClickHouse data.

### Package Layout

- `pkg/events/` — Kubernetes event recording (lifecycle events: created, ready, failed, deleted)
- `pkg/reconciler/` — shared `ReconcileHelper` used by all controllers
- `pkg/telemetry/` — OTel adapter interface + ClickHouse/SigNoz implementations
- `pkg/validation/` — image registry validation (`ValidateImageRegistry`)
- `pkg/network/` — NetworkPolicy helpers

## Critical Rules

**No mock data.** Features must work with real ClickHouse telemetry and real Kubernetes APIs before commit.

**Generated files must be staged.** Any change to `src/api/v1alpha1/` requires running `make generate && make helm-crds` and staging the output before committing.

**Conventional commits.** Use `feat:`, `fix:`, `clean:`, `docs:` prefixes. Use `WIP:` for partial implementations.
