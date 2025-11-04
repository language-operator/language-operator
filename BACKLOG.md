# Language Operator Backlog

When asked to iterate, plan the top item on the backlog.

## Prioritized Requests

### Foundation & Testing (Top Priority - Functional Dependencies)

* ~~Add comprehensive test coverage for web tool~~ - Tests exist (65 examples across 4 tools) but have infrastructure issues preventing execution. Fixed curl→HTTP client mocks, added fixtures, added SimpleCov. **Next step**: Run tests in production Docker image where langop gem is pre-installed, or extract tool logic to standalone testable modules
* E2E test: Create LanguageAgent with natural language instructions → verify synthesis succeeds → verify agent deploys and runs → update instructions → verify automatic re-synthesis and redeployment
* Complete remaining 23 pending SDK tests (refine mocks for LLM/MCP calls to achieve 100% test pass rate)
* Add comprehensive controller test coverage: LanguageAgent (synthesis, reconciliation), LanguageModel (proxy config), LanguageCluster (namespace management)
* Create integration test suite with example manifests (packaging, verification procedures, multiple E2E scenarios)

### Documentation & Production Readiness

* Add synthesis troubleshooting guide: common errors, validation failures, LLM connection issues, debug procedures, status interpretation
* Update sdk/ruby/README.md with agent DSL examples: schedule, objectives, workflows, constraints, personas
* Add synthesis architecture documentation: flow diagram, LLM integration, ConfigMap management, hash-based change detection
* Document synthesis environment variables: SYNTHESIS_MODEL, SYNTHESIS_ENDPOINT, defaults, override behavior
* Document DNS resolution behavior in README.md network isolation section: snapshot-based resolution (refreshes on reconciliation), wildcard DNS limitations (*.example.com resolves base domain only), CIDR alternative recommendations
* Package Helm chart for easy installation: configuration examples, value documentation, publish to chart repository, installation guide

### Production Features

* Complete LanguageClient controller implementation: ingress reconciliation, authentication mechanisms, session management, usage pattern documentation
* Add monitoring and observability: Prometheus metrics (request counts, latencies, errors), structured logging best practices, health check endpoints
* Implement LanguageCluster dashboard deployment: web UI showing cluster resources (tools, models, personas, agents), status overview, event stream, log aggregation

### Advanced Features - Safety & Cost Management

* Implement cost tracking: usage metrics in LanguageAgent status, token counting per request, cost estimation by model, aggregated cluster costs
* Add safety guardrails: content filtering integration, per-agent rate limiting, blocked topics enforcement, configurable thresholds

### Advanced Features - Scalability & Reliability

* Add advanced tool features: Horizontal Pod Autoscaling (HPA) configuration, PodDisruptionBudget (PDB) for availability, custom health probes, resource limits tuning
* Add advanced model features: load balancing across multiple model instances, fallback model chains, response caching layer, multi-region model support
* Implement event-driven triggers: webhook receivers, event source integrations, trigger condition DSL, action execution

### Advanced Features - Persistence & Integration

* Memory backend integration: Redis adapter for conversation history, Postgres adapter for structured data, S3 adapter for file storage, backend configuration in LanguageAgent spec

## Completed Requests ✅

### Core Infrastructure & Operators

* ~~Create Ruby SDK gem and build pipeline~~
* ~~Build component image hierarchy (base → ruby → client/tool/agent)~~
* ~~Implement LanguageCluster controller (namespace management)~~
* ~~Implement LanguageAgent controller (deployments, cronjobs, workspace, networking)~~
* ~~Implement LanguageTool controller (service + sidecar modes)~~
* ~~Implement LanguageModel controller (LiteLLM proxy)~~
* ~~Add DNS-based egress rules with automatic IP resolution~~
* ~~Create working E2E verification script (examples/simple-agent/verify.sh)~~
* ~~Fix status phase values (Running vs Ready)~~
* ~~Fix agent deployment creation for autonomous mode~~

### CI/CD & Registry

* ~~Set up CI/CD for automated image builds~~
* ~~Publish Ruby gem to private registry~~
* ~~Build and push all component images to registry~~
* ~~Re-enable automated testing in CI~~
* ~~Fix CI build order (agent depends on ruby, tools depend on tool component)~~

### SDK & Code Quality

* ~~Implement Persona integration in LanguageAgent controller~~
* ~~Fix Ruby SDK ruby_llm dependency issues~~
* ~~Fix sidecar tool injection bug~~
* ~~Add environment variable config support (MCP_SERVERS, MODEL_ENDPOINTS)~~
* ~~Add TCP readiness probes to sidecar containers~~
* ~~Add basic controller unit tests~~
* ~~Standardize all Makefiles with Docker targets (build, scan, shell, run)~~
* ~~Update .gitignore for Go build artifacts~~
* ~~Add test targets to all Makefiles for compliance (100% compliance achieved)~~

### Retry Logic & Error Handling

* ~~Add retry logic to agent connection code~~
  * ~~Handle startup race conditions gracefully~~
  * ~~Retry MCP server connections on failure~~
  * ~~Add exponential backoff~~

### DRY Refactoring (1,313 lines eliminated)

* ~~DRY Phase 1: Fix agent inheritance (change components/agent to inherit from Langop::Client::Base)~~
* ~~DRY Phase 2: Consolidate client code (363 lines removed)~~
* ~~DRY Phase 3: Consolidate DSL code (950 lines removed)~~
* ~~DRY Phase 4: Move reusable code to SDK (Context, ExecutionContext, ToolLoader)~~
* ~~Complete migration from "Based" to "Langop" nomenclature~~

### Code Quality Optimizations

* ~~Add Loggable mixin to eliminate duplicate logger initialization (50 lines saved)~~
* ~~Add Retryable mixin with exponential backoff (60-80 lines saved)~~
* ~~Unify HTTP client usage - replace curl with Langop::Dsl::HTTP (120 lines saved, eliminated command injection risk)~~

### Agent Synthesis Pipeline

* ~~Phase 1: Create Agent DSL in Ruby SDK~~
  * ~~Create agent_definition.rb, workflow_definition.rb, agent_context.rb~~
  * ~~Update dsl.rb to include agent DSL methods~~
  * ~~Update langop/agent entrypoint to auto-load /etc/agent/code/agent.rb~~
  * ~~Add comprehensive tests (96 examples, 0 failures, 23 pending)~~
* ~~Phase 2: Implement Operator Synthesis Logic~~
  * ~~Create synthesis package with gollm integration~~
  * ~~Add SynthesizeAgent() and DistillPersona() methods~~
  * ~~Add reconcileCodeConfigMap() to controller~~
  * ~~Implement SHA256 hash-based change detection~~
  * ~~Mount code ConfigMap to Deployment and CronJob~~
  * ~~Initialize synthesizer in operator main.go~~
  * ~~Add CreateOrUpdateConfigMapWithAnnotations() utility~~

### Synthesis Features & Testing

* ~~Define agent DSL syntax: `agent "name" do ... end` with schedule, objectives, workflow, constraints~~
* ~~Add synthesis controller methods (synthesizeAgentCode, distillPersona, validateSynthesizedCode)~~
* ~~Add synthesis LLM configuration (env var or ConfigMap)~~
* ~~Support local/remote LLM for synthesis~~
* ~~Create structured prompt templates for synthesis~~
* ~~ConfigMap management with owner references for cleanup~~
* ~~Hash annotation on Deployment to trigger restart on code change~~
* ~~Add status conditions: Synthesized, Validated, CodeUpdated~~
* ~~Event recording: SynthesisStarted, SynthesisSucceeded, SynthesisFailed, ValidationFailed~~
* ~~Add SynthesisInfo status type with detailed metrics~~
* ~~Failure modes implemented (invalid syntax, LLM failures)~~
* ~~Change detection: instructions/tools/models/persona with multi-hash tracking~~
* ~~Operator tests with mock LLM synthesis (coverage: 7.9% → 27.0%)~~

### Deployment & Testing

* ~~Deploy operator to cluster with all fixes~~
* ~~Run end-to-end demo (LanguageCluster + Model + Tool + Agent)~~
* ~~Verify agent pod runs with sidecar + workspace~~
* ~~End-to-End Testing & Deployment (move agent code to SDK, fix images, E2E verification)~~
* ~~Ruby SDK Testing & Versioning (85 tests: 62 passing, 23 pending, 0 failures) - ✅ Production ready~~

### Documentation Updates

* ~~Update STATUS.md (Ruby SDK, CI/CD, persona integration, Makefile standardization)~~
* ~~Update README.md (DNS resolution timing, wildcard DNS behavior)~~
* ~~Update CLAUDE.md (ruby_llm dependency findings, project conventions, DRY principles)~~

### Known Limitations

* **DNS resolution is snapshot-based** - IPs cached until next reconciliation; for frequently changing IPs, use CIDR ranges or accept refresh delays
* **Wildcard DNS (*.example.com) resolves base domain only** - Does not resolve all subdomains individually
* **Agent startup race condition (cosmetic)** - Agent logs one connection error on first startup before sidecar is ready; agent continues running normally after initial error

### Incomplete Features (Spec Exists, Implementation Pending)

* **LanguageClient controller** - Ingress, auth, session management not implemented
* **Memory backends** - Redis, Postgres, S3 adapters (spec exists in CRD)
* **Event-driven triggers** - Webhook/event source support (spec exists in CRD)
* **Cost tracking** - Status fields exist but not populated
* **Safety guardrails** - Content filtering, rate limits (spec exists in CRD)
* **Advanced tool features** - HPA, PDB, custom probes (spec exists in CRD)
* **Advanced model features** - Load balancing, fallback, caching, multi-region (spec exists in CRD)
