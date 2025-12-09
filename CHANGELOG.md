# Releases

This document tracks releases of the Language Operator project.

---

## Unreleased

### Learning System Simplification

**2025-12-09:**
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
  - Added `FetchDSLSchema()` function to retrieve JSON Schema from language-operator gem via `aictl system schema`
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
  - Skips gracefully when `aictl` unavailable (CI-friendly)
- All tests pass with >80% coverage on schema code

**Dependencies:**

- Requires `language-operator` gem v0.1.31 or compatible version
- Uses `bundle exec aictl` for schema access
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

- **Natural Language Interface**: Describe tasks in plain English via `aictl` CLI
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
