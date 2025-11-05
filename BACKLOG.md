# Language Operator Backlog

When asked to iterate, plan the top item on the backlog.

## Prioritized Requests

### Rebrand: langop CLI → aictl (Pre-MVP - Do First)

* Rename CLI binary from `langop` to `aictl` - Rename sdk/ruby/bin/langop → sdk/ruby/bin/aictl, update shebang and requires
* Rename gem from `langop` to `aictl` - Rename sdk/ruby/langop.gemspec → sdk/ruby/aictl.gemspec, update gem name, version, dependencies
* Rename Ruby module from `Langop` to `Aictl` - Find/replace all `module Langop` → `module Aictl`, `Langop::` → `Aictl::` in sdk/ruby/lib/
* Update all CLI command references - Find/replace `langop` → `aictl` in BACKLOG.md, VISION.md, documentation, help text, examples
* Update config directory path - Change `~/.aictl/` → `~/.aictl/` in config management code
* Update component references to use aictl gem - Update components/client, components/agent, components/tool Dockerfiles and Gemfiles to require 'aictl' instead of 'langop'
* Update test files - Rename spec files, update module references in all tests
* Keep operator and images as langop - Verify kubernetes/, components/ Docker images, CRDs still use langop.io and langop/* image names (no changes needed)
* Update CI/CD - Update .github/workflows to build and publish aictl gem instead of langop gem
* Create migration guide - Document the rebrand, explain language-operator (platform) vs aictl (CLI) separation

### Vision: Natural Language Agent Creation (Top Priority - User Stories)

**Goal**: Enable users to create autonomous agents by describing tasks in natural language, inspired by how professionals describe their repetitive work.

**User Story**: "As an accountant, I want to create an agent by saying 'review my spreadsheet at 4pm daily and email me any errors' without writing YAML or code."

#### MVP: Cluster Management (Foundation - Required First)

* Implement `aictl cluster create <name>` command - Create LanguageCluster resource with namespace, display creation progress, auto-switch to new cluster context, save to ~/.aictl/config.yaml
* Implement `aictl cluster list` command - Show table of all clusters: NAME, NAMESPACE, AGENTS, TOOLS, MODELS, STATUS with cluster health indicators
* Implement `aictl cluster inspect <name>` command - Show detailed cluster info: namespace, agent count by status, installed tools, configured models, available personas, recent activity, cost tracking (if available)
* Implement `aictl cluster delete <name>` command - Delete cluster with confirmation prompt, show resources to be deleted, verify removal
* Implement `aictl use <cluster>` command - Switch cluster context (like kubectl use-context), save to config file, display confirmation
* Implement `aictl cluster current` command - Show current cluster context with namespace and connection status
* Create config file management - ~/.aictl/config.yaml storing current-cluster and cluster list with namespaces and kubeconfig paths
* Add cluster validation on agent commands - Error with helpful message if no cluster selected, suggest `aictl cluster create` or `aictl use`

#### MVP: Beautiful CLI for Agent Lifecycle Management

* Implement `aictl agent create <description>` command - Parse natural language description, create LanguageAgent YAML with spec.instructions, apply to current cluster, watch synthesis status, show beautiful progress output with spinners and success confirmation
* Add `--create-cluster <name>` flag to agent create - Inline cluster creation if user doesn't have one yet, creates cluster and sets context before creating agent
* Add `--cluster <name>` flag to agent create - Override current cluster context for one-time agent creation in different cluster
* Implement `aictl agent list` command - Show table of agents in current cluster: NAME, MODE, STATUS, NEXT RUN, EXECUTIONS with colored output and formatted timestamps
* Add `--all-clusters` flag to agent list - Show agents across all clusters with CLUSTER column
* Implement `aictl agent logs <name> [-f]` command - Stream agent execution logs in real-time with timestamps, tool calls highlighted, errors in red, follow mode support
* Implement `aictl agent inspect <name>` command - Show detailed agent information: cluster, status, schedule, next run, execution history, tools, model, persona, synthesized code metadata, formatted as structured output
* Implement `aictl agent delete <name>` command - Delete agent with confirmation prompt, show cleanup progress, verify removal from cluster
* Implement `aictl agent code <name>` command - Display synthesized Ruby DSL code with syntax highlighting
* Implement `aictl agent edit <name>` command - Open editor to modify agent instructions, trigger re-synthesis on save, show synthesis progress
* Implement `aictl agent pause <name>` command - Pause scheduled agent execution, update status to Paused
* Implement `aictl agent resume <name>` command - Resume paused agent, update status back to Running/Ready

#### MVP: System Overview & Status

* Implement `aictl status` command - Overview dashboard showing: cluster connection, operator version, current cluster context, agent counts by status, tool availability, persona library, model configurations, recent activity log (Cilium-inspired beautiful output)
* Add multi-cluster view to status - When multiple clusters exist, show summary across all clusters with breakdown per cluster
* Implement `aictl version` command - Show langop CLI version, operator version in cluster, compatibility status

#### MVP: Persona Management

* Create built-in persona library - Ship 5+ personas as ConfigMaps in kubernetes/charts/language-operator/persona-library/: financial-analyst, devops-engineer, general-assistant, executive-assistant, customer-support with complete specs (systemPrompt, tone, capabilities, toolPreferences, responseFormat)
* Implement `aictl persona list` command - Show table of personas in current cluster: NAME, TONE, USED BY (count), DESCRIPTION
* Implement `aictl persona show <name>` command - Display full persona YAML with syntax highlighting
* Implement `aictl persona create <name>` command - Interactive wizard to create custom persona: prompt for displayName, description, tone, systemPrompt, capabilities, optionally inherit from existing persona
* Add `--from <persona>` flag to persona create - Copy/inherit from existing persona as starting point
* Implement `aictl persona edit <name>` command - Open editor to modify persona YAML, validate on save, trigger agent re-synthesis for agents using this persona
* Implement `aictl persona delete <name>` command - Delete persona with check for agents using it, require confirmation if in use
* Document persona system - Add to README: how personas work, when auto-selected, how to create custom personas, inheritance from parent personas, examples

#### MVP: Tool Discovery & Management

* Implement `aictl tool list` command - Show table of tools in current cluster: NAME, TYPE, STATUS, AGENTS USING with connection health indicators
* Implement `aictl tool install <name>` command - Create LanguageTool resource from template, apply to current cluster, show installation progress, prompt for authentication if needed
* Implement `aictl tool auth <name>` command - Interactive authentication flow for tools requiring credentials (OAuth2, API keys), store in Kubernetes Secret, verify connection
* Implement `aictl tool test <name>` command - Validate tool connectivity and basic functionality, show success/failure with detailed error messages
* Create tool registry mapping - Add sdk/ruby/config/tool_patterns.yaml mapping common keywords to MCP tools: spreadsheet→mcp-gdrive, email→mcp-gmail, slack→mcp-slack, web→mcp-fetch, file→mcp-filesystem
* Create tool installation templates - Add tool YAML templates to sdk/ruby/lib/langop/cli/templates/tools/ for common MCP servers
* Implement `aictl tool delete <name>` command - Delete tool with check for agents using it, require confirmation if in use

#### MVP: CLI Foundation & Polish

* Add beautiful CLI output formatting - Integrate tty-spinner for progress, tty-table for tables, tty-prompt for interactive questions, pastel/colorize for highlighting, proper error messages with next steps
* Add `--dry-run` flag to agent create - Preview what would be created: show generated YAML, detected tools, selected persona, schedule extraction without applying to cluster
* Add kubeconfig detection and cluster validation - Check KUBECONFIG env var or ~/.kube/config, validate cluster connectivity, check operator installed, show helpful error with installation steps if missing
* Add shell completions - Generate bash/zsh/fish completions for all commands, subcommands, and flags, package in gem
* Create comprehensive CLI help text - Rich examples for each command with real-world scenarios, link to documentation, common workflows documented in --help output
* Add debug mode - `--debug` flag for verbose logging, show API calls, synthesizer prompts/responses, useful for troubleshooting
* Create getting started guide - Add GETTING_STARTED.md with: installation, first cluster creation, first agent creation, monitoring agents, common workflows

### Foundation & Testing (Functional Dependencies)

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
