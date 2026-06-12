# Releases

This document tracks releases of the Language Operator project.

---

## Unreleased

---

## v0.1.130 — 2026-06-12

**Breaking Changes**
- make gateway ingress opt-in instead of created by default (#856)
- extract runtime adapters into their own repos (#855)
- make runtime CRs dictate credentials and auth gating (#854)

**Features**
- transport-aware MCP tools with operator-injected stdio bridge (#857)
- make gateway ingress opt-in instead of created by default (#856)
- install Go toolchain in claude-code-adapter image
- per-agent window title with bell-driven attention indicator
- show operator chart version in Dex login navbar and inline the wordmark
- brand the Dex login page with embedded Marfa templates
- auto-execute agent instructions on Claude startup, refactor dev-team
- rewrite claude-code runtime with custom xterm.js terminal, tmux persistence, and clipboard support
- add OIDC auth to LanguageAgent ingress via Dex and oauth2-proxy
- fix claudeCode.apiKeyRef to honour api-key secret key convention (#848)
- add /request and /audit-request slash commands
- replace A2A server with ttyd WebSocket terminal for claude-code runtime
- OAuth credentials support for Claude Code subscription billing
- passive supervisor + start script for delegation loop
- run supervisor delegation in a continuous loop via sleep
- wire up autonomous agent startup end-to-end
- add STARTUP_PROMPT to claude-code-server for autonomous agent startup
- LanguageCluster adopts pre-existing member resources on reconcile (#823)
- allow LanguageCluster to adopt existing namespaces (#822)
- add namespace conflict and port range validation to LanguageCluster webhook (#817)
- add image registry and resource quantity validation to LanguageAgentRuntime webhook (#816)
- add admission webhook for LanguageAgentRuntime (#815)
- add admission webhook for LanguageCluster (#812)
- wire gateway command/args and implement managed gateway ServiceAccount (#776)
- add Updating phase to LanguageAgent status during rolling updates (#767)
- HorizontalPodAutoscaler support for LanguageCluster gateway (#757)
- implement Command, Args, and Autoscaling for LanguageTool spec.deployment (#756)
- implement Pending and Failed phase writes in LanguagePersona controller (#745)
- implement Pending and Failed phase writes in LanguageModel controller (#744)
- add Gateway and GatewayReady printcolumns to LanguageCluster (#743)
- claude-code runtime with A2A protocol support (#731)
- per-agent ServiceMonitor and PrometheusRule for Prometheus Operator integration (#730)
- add LanguageAgentSelfConfig CRD for agent self-modification at runtime (#729)
- per-agent ServiceAccount with annotations and custom RBAC rules (#728)
- workspace file seeding with seed-once semantics (#727)
- add Degraded phase to LanguageAgent and LanguageTool status (#726)
- HorizontalPodAutoscaler support for LanguageAgent (#725)
- create PodDisruptionBudget for multi-replica LanguageAgent deployments (#723)
- add status.managedResources inventory to LanguageAgent and LanguageCluster (#722)
- add workspace PVC retain policy to prevent data loss on agent deletion (#720)
- add webhook security warnings for dangerous agent configurations (#719)

**Bug Fixes**
- display agent name on OAuth grant page instead of "Language Operator" (#853)
- use pre-built ttyd binaries instead of building from source
- skip gateway env vars in settings.json when OAuth credentials present
- tell supervisor it's a daemon so it runs the sleep loop
- restore instructions indentation in example agents
- adapter writes placeholder API key when routing through gateway
- translate api-key secret to ANTHROPIC_API_KEY in claude-code-adapter
- pin @anthropic-ai/claude-code to 2.0.23 — last version with SDK query() export
- parenthesise nullish-coalescing/OR chain in claude-code-server
- parenthesise nullish-coalescing/OR chain in claude-code-adapter agent card
- restore crds.keep conditional and crds.annotations in LanguageCluster CRD template (#827)
- remove adoptExistingNamespace field — LanguageCluster auto-adopts unmanaged namespaces (#826)
- grant langop-admin rolebinding CRUD for dashboard Access page
- move LanguageTool image registry validation from controller to webhook (#813)
- wire crds.keep and crds.annotations values into CRD templates (#799)
- set certificateIssuerName default to empty string, matching binary default (#794)
- always pass --zap-devel flag explicitly so chart value is honoured (#792)
- merge ServiceAccountAnnotations and RoleRules from LanguageAgentRuntime (#774)
- use HPA-managed replica count for LanguageTool Updating phase detection (#765)
- inject ANTHROPIC_BASE_URL in claude-code gateway mode (#764)
- include claude-code runtime Secret in status.managedResources (#763)
- remove Namespace field from SecretReference — cross-namespace secrets not supported (#754)
- mount each apiKeySecretRef secret to unique /etc/secrets/<name> subdirectory (#753)
- write Pending phase on first LanguageAgentSelfConfig reconcile (#752)
- expand LanguageModel and LanguagePersona Phase enum to include Pending and Failed (#740)
- respect Enabled field in OpencodeConfig/OpenclawConfig/ClaudeCodeConfig (#739)
- set ConditionCapacityReady True on reconcileCapacity success (#738)
- add missing RBAC rules and skip webhook validation on deletion
- prevent DNS goroutine race in GatewayIngressClassName test
- prevent DNS goroutine race in GatewayIngressClassName test
- update hardcoded controller-gen and envtest versions in CI
- validate immutable workspace fields (storageClassName, accessMode) on update (#721)
- inject config-hash annotation into agent pod template to trigger rolling updates (#718)

**Refactoring**
- extract runtime adapters into their own repos (#855)
- make runtime CRs dictate credentials and auth gating (#854)
- merge claude-code-server into claude-code-adapter (single image)
- rename dev-team manifests and swap to generic engineer persona
- restructure examples and switch claude-code to interactive auth
- run oauth2-proxy as agent sidecar instead of separate Deployment

**Documentation**
- fix start script description in language-operator-team README
- write README for examples/language-operator-team (#850)
- fix stale chart/ path in testing.md (#838)
- update installation and development docs for /charts split
- correct claude-code guide — gateway unsupported, model name, egress note
- add claude-code runtime guide

**Tests**
- add unit tests for LanguageAgentRuntime controller (#811)

**Chores**
- update commands
- update runtime subchart pins
- bump worker resource limits for go build/test
- remove A2A leftovers from language-operator-team example (#851)
- simplify examples to use spec.claudeCode.apiKeyRef (#849)
- add OAuth credentials setup to kustomization.yaml deploy instructions (#845)
- fix secret creation command to use api-key literal in examples (#844)
- install runtimes chart in make dev and strengthen audit-request
- update CI helm-release workflow to publish both Helm charts (#837)
- create /charts/language-operator-runtimes chart for bundled LanguageAgentRuntime CRs (#835)
- move /chart → /charts/language-operator and strip runtime templates (#834)
- migrate claude-code-server to @anthropic-ai/claude-agent-sdk
- consolidate LanguageTool status updates into single deferred write (#819)
- normalize SetCondition + Phase mutation — use SetPhase everywhere (#814)
- add roles
- consolidate RegistryManager interface into pkg/validation and remove dead field (#818)
- docs
- move ApplyRuntimeDefaults to pkg/merge — out of api/v1alpha1 (#803)
- reorganize controllers/utils.go — split by domain, move constants to pkg/ (#802)
- split languageagent_controller_test.go into domain-focused files
- split languageagent_controller.go into domain-focused files (#801)
- remove over-permissioned languageagentruntimes/status ClusterRole rule (#800)
- add CreateOrUpdateOwned helper and eliminate SetControllerReference boilerplate (22 sites) (#793)
- make EventManager methods nil-safe, remove 17 guard blocks from controllers (#791)
- extract buildNetworkPolicyIngressRules helper to deduplicate NetworkPolicy reconcilers (#790)
- extract serviceURL helper to replace repeated svc.cluster.local format strings (#780)
- define GatewayResourceName constant to replace 25 hardcoded "gateway" strings (#784)
- extract SetPhase helper to eliminate Status boilerplate in model and persona controllers (#783)
- replace boolPtr helper with ptr.To — extend #770 cleanup (#779)
- replace duplicate protocolPtr helpers with ptr.To from k8s.io/utils (#775)
- remove redundant ConditionWebhooksReady — superseded by WebhookRouteCreated/Ready (#766)
- add missing condition reason constants for LanguageCluster controller (#747)
- replace plain bool omitempty with *bool for enablement flags in agent types (#746)
- update CLAUDE.md to reference chart/templates/crds/
- move CRDs to chart/templates/ for proper Helm upgrade handling
- update build tools, actions, and fix security vulnerability
- update go and npm dependencies
- add claude-code-adapter and claude-code-server to build pipeline
- tweak iterate command

**Other**
- Update README.md
- Update README.md
- Update README.md
- Update README.md
- revert: back to @anthropic-ai/claude-code 2.0.23
- chart: add missing LanguageAgentSelfConfig ValidatingWebhookConfiguration entry (#798)

**Migration Notes**
- Extract the claude-code, openclaw, and opencode runtimes into their own
  repositories (image source + self-contained chart). `language-operator-runtimes`
  is now an umbrella chart that pulls each runtime as a subchart from
  `oci://ghcr.io/language-operator/charts`. Its values are now keyed by subchart
  name — `runtimes.claudeCode.enabled` → `claude-code.enabled`,
  `runtimes.claudeCode.imagePullPolicy` → `claude-code.image.pullPolicy`,
  `runtimes.openclaw.enabled` → `openclaw.enabled`,
  `runtimes.opencode.enabled` → `opencode.enabled`. Run
  `helm dependency build charts/language-operator-runtimes` before installing or
  packaging the umbrella chart.
- Remove `components/agents/*` adapter source and their build jobs from this repo;
  each runtime now builds and publishes its own image and chart via its own CI.

---

## v0.1.129 — 2026-04-03

**Features**
- auto-detect ingress controller namespace from IngressClass

**Bug Fixes**
- mount agent-config volume into user init containers
- config.yaml generated with PascalCase keys; adapter mcp.servers not updated on restart
- update mcp.servers on every adapter run, not only on first boot
- missing Accept header and double http:// prefix in MCP schema discovery
- add ingressclasses RBAC so ingress controller namespace detection works
- run ingress controller namespace detection before cache start

**Documentation**
- add MCP getting-started guide
- fix README links to use langop.io custom domain

**Chores**
- update docs

---

## v0.1.128 — 2026-04-03

**Bug Fixes**
- enable TLS by default when DefaultTLSIssuerName is configured
- disable device pairing in openclaw-adapter for Kubernetes compatibility

**Chores**
- update docs

---

## v0.1.127 — 2026-04-03

**Bug Fixes**
- move CRDs to chart/crds/ so Helm installs them before post-install hooks

**Documentation**
- add links to Kubernetes operator docs and CRD API reference in README

**Chores**
- readme (×3)
- release command

---

## v0.1.126 — 2026-04-03

### Learning System Simplification

**2025-12-09:**
- **Added workaround for language-operator gem 0.1.66 packaging issues**
  - Gem 0.1.66 has incorrect file permissions on `lib/language_operator/constants.rb` (600 instead of 644)
  - Gem 0.1.66 is missing required file `lib/language_operator/instrumentation/task_tracer.rb`
  - Added chmod command in Dockerfile to fix constants.rb permissions after gem install
  - **Note**: Task schema extraction still fails due to missing task_tracer.rb file
  - Filed issue: https://github.com/language-operator/language-operator-gem/issues/131
  - Learning system falls back to default task schema when extraction fails

- **Fixed read-only filesystem preventing task schema extraction**
  - Added emptyDir volume mount for `/tmp` in operator deployment
  - Task schema extraction script creates temporary files to pass agent code to Ruby parser
  - **Critical fix**: Schema extraction can now write temp files (was failing with "read-only file system")

- **Fixed SigNoz Query Builder v5 response parsing**
  - Updated response structure to match actual Query Builder v5 format: `{"status":"success","data":{"type":"raw","data":{"results":[{"rows":[{"data":{...}}]}]}}}`
  - Changed field selection from `task.input.keys`/`task.output.keys` to `task.inputs`/`task.outputs`
  - Fixed operation name field from `operationName` to `name` in Query Builder v5 responses
  - Fixed duration field from `duration` to `durationNano` in Query Builder v5 responses
  - Added extraction of top-level task and GenAI attributes from query results
  - **Critical fix**: Learning system can now actually parse telemetry data from SigNoz (was getting zero traces due to parsing failure)

- **Auto-inject task I/O capture env vars for learning system**
  - Operator now automatically sets `CAPTURE_TASK_INPUTS=true` and `CAPTURE_TASK_OUTPUTS=true` in agent pods
  - Enables learning system to extract task inputs/outputs from OpenTelemetry spans
  - Required for task schema analysis and pattern detection
  - Removes need for manual configuration in agent specs

- **Fixed span name filter to match Ruby gem's actual telemetry format**
  - Updated `convertSpansToTaskTraces()` to filter for `task_executor.execute_task` instead of `execute_task`
  - This was preventing the learning system from finding any task execution traces
  - Root cause: Code expected different span name than what Ruby gem actually sends
  - Verified actual span format from SigNoz UI: `task_executor.execute_task` with parent `agent_executor`
  - Created issue #130 in language-operator-gem to document expected span naming convention
  - **Critical fix**: Learning system can now properly retrieve and analyze task execution traces

- **Implemented task schema extraction for learning optimization**
  - Created `extract-task-schema.rb` script to parse agent DSL and extract task definitions
  - Implemented `extractTaskSchema()` in learning controller to call Ruby script
  - Now extracts real task inputs, outputs, instructions, and current code from agent
  - Replaced TODOs/placeholders with actual task information from agent code
  - **Critical fix**: LLM now has proper context for optimization (was using empty placeholders)
  - Falls back gracefully to defaults if extraction fails
  - Script installed in Docker image at `/usr/local/bin/extract-task-schema.rb`

- **Completed learning system simplification cleanup (Issue #102)**
  - Removed 4 unused fields from `LearningReconciler` struct:
    - `MetricsCollector` - unused after event processing removal
    - `EventProcessor` - deleted with event processing code
    - `SuccessRateAggregator` - unused per-agent success tracking
    - `MaxVersions` - was for ConfigMap version limits (no longer needed)
  - Removed unused `pkg/learning` import
  - Created ADR-001 documenting learning system simplification rationale and architecture
  - Final result: Clean, simplified learning system with 38% less code
  - Documentation: [docs/architecture/adr-001-learning-system-simplification.md](docs/architecture/adr-001-learning-system-simplification.md)

- **Deleted event processing and ConfigMap management code (Issue #101)**
  - Removed 1190 lines of unused code after migration to direct SigNoz queries (Issue #100)
  - Deleted 23 functions: event processing, job tracking, ConfigMap management, trace summarization
  - Deleted 3 structs: `TaskLearningStatus`, `AgentExecutionSummary`, `TaskExecutionStatus`
  - Removed Event and Job watches from SetupWithManager
  - Result: learning_controller.go reduced from 3118 to 1928 lines (38% reduction)
  - Benefits:
    - Simpler codebase with single learning path
    - No ConfigMap size management complexity
    - No event watching overhead
    - Easier to maintain and debug

- **Simplified task identification to query SigNoz directly (Issue #100)**
  - Rewrote `identifyTasksForOptimization()` to use direct SigNoz queries (Ruby gem approach)
  - Removed ConfigMap-based learning status dependency
  - Now queries traces directly from SigNoz when optimization triggers
  - Calculates pattern confidence inline from trace data
  - Benefits:
    - No ConfigMap persistence layer needed
    - Real-time data from SigNoz (not cached)
    - Simpler data flow: query → analyze → synthesize
    - Matches Ruby gem's proven approach

- **Reverted event-based learning data collection (Issue #99)**
  - Removed `processAgentExecutions()` call from Reconcile loop added in commit efb218d
  - This commit perpetuated the complex event-based approach we're simplifying
  - Preparing codebase for direct SigNoz query approach (Issue #100)
  - Moving to simpler Ruby gem-style pattern: query SigNoz when optimization triggers, analyze patterns, synthesize tasks
  - No more intermediate ConfigMap persistence of learning data

### Learning System Optimization

**Overview:**
Fixed learning system ConfigMap explosion and excessive LLM calls by removing dual learning paths and implementing query-based task identification.

**Fixes:**

- **Issue #98: Learning System ConfigMap Explosion**
  - **Problem**: Learning system created 16+ ConfigMaps and excessive LLM calls when `runsPendingLearning` reached threshold (10 runs)
  - **Root Cause**: Two separate learning systems (threshold-based and event-based) ran simultaneously
  - **Solution**:
    - Deleted event-based learning path entirely (33 functions, ~1560 lines)
    - Removed individual ConfigMap creation per task
    - Unified to single threshold-based path using `LanguageAgentVersion` resources
  - **Impact**:
    - Single `LanguageAgentVersion` created instead of 16+ ConfigMaps
    - Reduced LLM calls from 16+ to 2-3 per learning cycle
    - Counter resets correctly after optimization
    - No more "OptimizationTriggered after 0 runs" events

- **Task Identification Improvements**
  - Replaced hardcoded task list with query-based identification from learning status
  - **Added missing data collection**: Learning controller now calls `processAgentExecutions()` to populate learning status ConfigMap
    - **Root Cause**: After removing event-based path, the simplified Reconcile loop never populated the ConfigMap
    - **Fix**: Added `processAgentExecutions()` call before threshold check to process TaskCompleted events
    - **Result**: Learning status ConfigMap now contains task execution metrics (TraceCount, PatternConfidence, etc.)
  - Tasks now identified based on:
    - Trace count >= threshold
    - Pattern confidence >= minimum
    - Not recently optimized (respects learning interval)
    - Not already symbolic
  - Extracts actual execution traces and tool usage from telemetry
  - Formats trace data for synthesis instead of using placeholder values

**Code Changes:**

- **Deleted Functions** (event-based learning path):
  - `checkLearningTriggers()`, `checkErrorTriggers()` - Event generation
  - `processLearningTrigger()`, `generateLearnedCode()` - ConfigMap creation per task
  - `getTaskFailures()`, `parseTaskFailureFromEvent()` - Error analysis
  - `updateDeployment()`, `updateAlternativeWorkload()` - Direct deployment updates
  - 25+ deployment/ConfigMap patching helper functions

- **Deleted Types**:
  - `LearningEvent` struct
  - Error-related fields from `LearningReconciler` and `TaskLearningStatus`

- **Improved Functions**:
  - `identifyTasksForOptimization()` - Now queries learning status instead of hardcoded tasks
  - Added `formatTracesForSynthesis()` - Converts traces to synthesis-friendly format
  - Added `extractToolsFromTraces()` - Extracts unique tool names from traces

**Files Modified**:
- `src/controllers/learning_controller.go` - Deleted 1560 lines, improved task identification
- `src/controllers/learning_controller_test.go` - Disabled obsolete tests (to be updated)

**Testing**: Main code compiles and builds successfully. Test updates in progress.

### Schema Integration & Validation

**Overview:**
Integration between the Go operator and the Ruby gem's DSL schema, enabling runtime validation of synthesized agent code and version compatibility checking.

**Features:**

- **Schema Fetching** ([#1](https://github.com/language-operator/language-operator/issues/1))
  - Added `FetchDSLSchema()` function to retrieve JSON Schema from language-operator gem via `langop system schema`
  - Added `GetSchemaVersion()` for efficient version-only queries
  - Implemented command execution with timeout handling and context cancellation
  - Schema is fetched from the installed gem, ensuring operator uses the correct DSL specification

- **Schema Validation** ([#2](https://github.com/language-operator/language-operator/issues/2))
  - Integrated schema validation into synthesis pipeline via `ValidateGeneratedCodeAgainstSchema()`
  - Ruby validator script (`scripts/validate-dsl-schema.rb`) validates DSL code using gem's actual parser
  - Returns structured violation reports with line numbers and error messages
  - Gracefully skips validation when Ruby/bundler unavailable (non-blocking in CI)
  - Generated agents are validated before being stored in ConfigMaps

- **Version Compatibility** ([#3](https://github.com/language-operator/language-operator/issues/3))
  - Added semantic version parser supporting major.minor.patch format
  - Implemented `ValidateSchemaCompatibility()` called during operator startup
  - Logs compatibility warnings for version mismatches:
    - **ERROR** for major version mismatch (incompatible, breaking changes)
    - **WARNING** for minor version mismatch (new features, should be compatible)
    - **INFO** for patch version differences (bug fixes, fully compatible)
  - Expected schema version: `0.1.31` (matches gem v0.1.31)

- **Telemetry Attributes** ([#4](https://github.com/language-operator/language-operator/issues/4))
  - Added `schema_version` attribute to synthesis telemetry spans
  - OpenTelemetry traces now include schema version for debugging
  - Version information propagated through synthesis pipeline

**Testing:**

- Comprehensive unit tests for all schema functions (`src/pkg/synthesis/schema_test.go`)
- Template validation integration tests (`src/pkg/synthesis/template_test.go`)
- End-to-end schema integration test ([#7](https://github.com/language-operator/language-operator/issues/7))
  - Tests full Go→Ruby schema flow in `test/integration/schema_integration_test.go`
  - Validates schema fetching, validation, compatibility checking, and error handling
  - Skips gracefully when `langop` unavailable (CI-friendly)
- All tests pass with >80% coverage on schema code

**Dependencies:**

- Requires `language-operator` gem v0.1.31 or compatible version
- Uses `bundle exec langop` for schema access
- Validator script: `scripts/validate-dsl-schema.rb`

**Breaking Changes:**

None. Schema validation is additive and degrades gracefully when dependencies are unavailable.

**Migration Notes:**

- Operators should use gem version matching `ExpectedSchemaVersion` constant in `src/pkg/synthesis/schema.go`
- Check operator logs on startup for schema compatibility warnings
- If major version mismatch logged, update either the operator or gem to matching major version

---

## v0.2.0 - 2025-11-09

**Initial Public Release**

Language Operator is a Kubernetes operator that transforms natural language descriptions into autonomous agents that execute tasks on your behalf.

### Core Capabilities

- **Natural Language Interface**: Describe tasks in plain English via `langop` CLI
- **Autonomous Agent Synthesis**: Automatically generates Ruby code from task descriptions
- **Kubernetes-Native**: Deploys agents as CRDs (LanguageAgent, LanguageCluster, LanguageModel, LanguageTool)
- **Scheduled Execution**: Cron-based scheduling for recurring tasks
- **Tool Integration**: Built-in support for email, spreadsheets, web scraping, and custom tools
- **Multi-LLM Support**: Integration with Anthropic Claude and other language models via LiteLLM
- **Network Isolation**: NetworkPolicy enforcement for secure agent execution (requires compatible CNI like Cilium)
- **Private Registry Support**: Container image whitelist and authentication for air-gapped deployments
- **Observability**: OpenTelemetry instrumentation for all controllers with W3C trace context propagation to agent pods

### Architecture

- **Operator Namespace**: `language-operator`
- **Base Images**: Alpine-based with `langop` user for security
- **SDK**: Published `language-operator` gem for Ruby components
- **Infrastructure**: Tested on k3s with Cilium CNI

### Getting Started

See [docs/quickstart.md](docs/quickstart.md) for installation and usage instructions.
