# Releases

This document tracks releases of the Language Operator project.

---

## Unreleased

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

- **Operator Namespace**: `kube-system`
- **Base Images**: Alpine-based with `langop` user for security
- **SDK**: Published `language-operator` gem for Ruby components
- **Infrastructure**: Tested on k3s with Cilium CNI

### Getting Started

See [docs/quickstart.md](docs/quickstart.md) for installation and usage instructions.
