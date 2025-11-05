# Language Operator Backlog

When asked to iterate, plan the top item on the backlog.

## Prioritized Requests

* Complete migration from "langop" naming to "language-operator"
* Complete remaining 23 pending SDK tests - Refine LLM/MCP connection mocks, add missing fixtures for edge cases, achieve 100% test pass rate, maintain 85+ examples coverage
* Add comprehensive CLI help text - Include real-world usage examples in each command help, document common workflows (create first agent, monitor execution, troubleshooting), add links to online documentation
* Add debug mode flag - Implement `--debug` global flag for verbose logging, show kubectl API calls, display synthesizer LLM prompts/responses, log timing information for troubleshooting
* Create GETTING_STARTED.md guide - Document installation steps, first cluster setup, first agent creation walkthrough, monitoring and logs, common troubleshooting scenarios
* Add comprehensive controller test coverage - Test LanguageAgent synthesis/reconciliation logic, LanguageModel proxy configuration, LanguageCluster namespace management, achieve 50%+ coverage
* Create integration test suite - Build example manifests for common scenarios, add packaging verification procedures, implement multiple E2E workflows (scheduled agent, autonomous agent, tool integration)
* Add synthesis troubleshooting guide - Document common synthesis errors, validation failure patterns, LLM connection issues, debug procedures, status condition interpretation
* Update sdk/ruby/README.md with DSL examples - Add complete agent DSL examples showing schedule syntax, objectives definition, workflow blocks, constraints usage, persona integration
* Add synthesis architecture documentation - Create flow diagram showing synthesis pipeline, document LLM integration points, explain ConfigMap management, describe hash-based change detection mechanism
* Document synthesis environment variables - Explain SYNTHESIS_MODEL configuration, SYNTHESIS_ENDPOINT usage, default values, override behavior for custom LLMs
* Package Helm chart for distribution - Add comprehensive values.yaml with documentation, create configuration examples, publish to chart repository, write installation guide
* Complete LanguageClient controller - Implement ingress reconciliation for client access, add authentication mechanisms (API keys, OAuth), session management, usage pattern documentation
* Add monitoring and observability - Export Prometheus metrics (request counts, latencies, error rates), implement structured logging standards, add health check endpoints to all components
* Implement LanguageCluster dashboard - Deploy web UI showing cluster resources, agent/tool/model status overview, live event stream, aggregated log viewer
* Implement cost tracking - Add usage metrics to LanguageAgent status, count tokens per request, estimate costs by model pricing, aggregate cluster-wide costs
* Add safety guardrails - Integrate content filtering APIs, implement per-agent rate limiting, enforce blocked topics lists, add configurable safety thresholds
* Add advanced tool features - Support Horizontal Pod Autoscaling configuration, add PodDisruptionBudgets for availability, implement custom health probes, add resource limit recommendations
* Add advanced model features - Implement load balancing across model instances, create fallback model chains, add response caching layer, support multi-region model deployments
* Implement event-driven triggers - Add webhook receiver endpoints, integrate event source platforms, create trigger condition DSL, implement action execution pipeline
* Add memory backend integration - Create Redis adapter for conversation history, Postgres adapter for structured data, S3 adapter for file storage, add backend configuration to LanguageAgent spec

## Completed Requests ✅

### E2E Testing Framework

* ~~E2E test: Full agent lifecycle~~ - Created comprehensive E2E testing framework in sdk/ruby/spec/e2e/ testing aictl CLI workflows: cluster creation → agent creation from natural language → synthesis verification → pod deployment → log access → agent inspection → code retrieval → cleanup. Includes AictlHelper module with utilities for running aictl commands, waiting for conditions, checking resource states. Added rake tasks (rake e2e, rake test), environment variable configuration (E2E_NAMESPACE, E2E_SYNTHESIS_TIMEOUT, E2E_POD_TIMEOUT, E2E_SKIP_CLEANUP), and comprehensive README with setup instructions, usage examples, troubleshooting guide

### Rebrand: langop CLI → aictl

* ~~Rename CLI binary from `langop` to `aictl`~~ - Renamed sdk/ruby/bin/langop → sdk/ruby/bin/aictl, updated shebang and requires
* ~~Rename gem from `langop` to `aictl`~~ - Renamed sdk/ruby/langop.gemspec → sdk/ruby/aictl.gemspec, updated gem name, version, dependencies
* ~~Rename Ruby module from `Langop` to `Aictl`~~ - Find/replaced all `module Langop` → `module Aictl`, `Langop::` → `Aictl::` in sdk/ruby/lib/
* ~~Update all CLI command references~~ - Find/replaced `langop` → `aictl` in BACKLOG.md, VISION.md, documentation, help text, examples
* ~~Update component references to use aictl gem~~ - Updated components/client, components/agent, components/tool Dockerfiles and Gemfiles to require 'aictl' instead of 'langop'
* ~~Update test files~~ - Updated spec files, module references in all tests
* ~~Keep operator and images as langop~~ - Verified kubernetes/, components/ Docker images, CRDs still use langop.io and langop/* image names (no changes needed)

### MVP: Cluster Management

* ~~Implement `aictl cluster create <name>` command~~ - Created LanguageCluster resource creation with namespace, progress display, auto-switch to new cluster context, saves to ~/.aictl/config.yaml
* ~~Implement `aictl cluster list` command~~ - Shows table of all clusters: NAME, NAMESPACE, AGENTS, TOOLS, MODELS, STATUS with cluster health indicators
* ~~Implement `aictl cluster inspect <name>` command~~ - Shows detailed cluster info: namespace, agent count by status, installed tools, configured models, available personas
* ~~Implement `aictl cluster delete <name>` command~~ - Deletes cluster with confirmation prompt, shows resources to be deleted, verifies removal
* ~~Implement `aictl use <cluster>` command~~ - Switches cluster context (like kubectl use-context), saves to config file, displays confirmation
* ~~Implement `aictl cluster current` command~~ - Shows current cluster context with namespace and connection status
* ~~Create config file management~~ - ~/.aictl/config.yaml storing current-cluster and cluster list with namespaces and kubeconfig paths
* ~~Add cluster validation on agent commands~~ - Created ClusterValidator helper with helpful error messages, integrated into all agent commands, suggests `aictl cluster create` or `aictl use` when no cluster selected

### MVP: Agent Lifecycle Management

* ~~Implement `aictl agent create <description>` command~~ - Creates LanguageAgent from natural language description, applies to cluster, watches synthesis status with spinner, shows success confirmation and next steps
* ~~Add `--create-cluster <name>` flag to agent create~~ - Inline cluster creation if user doesn't have one yet, creates cluster and sets context before creating agent
* ~~Add `--cluster <name>` flag to agent create~~ - Override current cluster context for one-time agent creation in different cluster
* ~~Implement `aictl agent list` command~~ - Shows table of agents in current cluster: NAME, MODE, STATUS, NEXT RUN, EXECUTIONS with colored output
* ~~Add `--all-clusters` flag to agent list~~ - Shows agents across all clusters with CLUSTER column
* ~~Implement `aictl agent logs <name> [-f]` command~~ - Streams agent execution logs using kubectl, supports follow mode (-f) and tail options, works with both scheduled and autonomous agents
* ~~Implement `aictl agent inspect <name>` command~~ - Shows detailed agent information: status, configuration, instructions, tools, models, synthesis info, execution stats, conditions with colored status indicators
* ~~Implement `aictl agent delete <name>` command~~ - Deletes agent with confirmation prompt (unless --force), shows agent details before deletion, verifies removal
* ~~Implement `aictl agent code <name>` command~~ - Displays synthesized Ruby DSL code from ConfigMap, shows helpful error if synthesis not complete
* ~~Implement `aictl agent edit <name>` command~~ - Opens $EDITOR to modify agent instructions, updates resource on save, triggers automatic re-synthesis by operator, shows helpful next steps
* ~~Implement `aictl agent pause <name>` command~~ - Pauses scheduled agent execution by setting CronJob suspend=true, validates agent is scheduled mode, provides resume instructions
* ~~Implement `aictl agent resume <name>` command~~ - Resumes paused agent by setting CronJob suspend=false, validates agent is scheduled mode, shows next execution time info

### MVP: System Overview & Status

* ~~Implement `aictl status` command~~ - Overview dashboard showing cluster connection, operator version, current context, agent/tool/model/persona counts by type and status, beautiful formatted output with colored status indicators
* ~~Add multi-cluster view to status~~ - When multiple clusters exist, shows summary table across all clusters with breakdown per cluster, marks current cluster with *
* ~~Implement `aictl version` command~~ - Shows aictl CLI version and operator version from current cluster, checks compatibility status, provides helpful messages if not connected

### MVP: Persona Management

* ~~Create built-in persona library~~ - Ship 5+ personas as ConfigMaps in kubernetes/charts/language-operator/persona-library/: financial-analyst, devops-engineer, general-assistant, executive-assistant, customer-support with complete specs (systemPrompt, tone, capabilities, toolPreferences, responseFormat)
* ~~Implement `aictl persona list` command~~ - Shows table of personas with NAME, TONE, USED BY count, DESCRIPTION; counts agent usage, provides helpful guidance if empty
* ~~Implement `aictl persona show <name>` command~~ - Displays full persona YAML, shows usage example
* ~~Implement `aictl persona create <name>` command~~ - Interactive wizard to create custom persona: prompt for displayName, description, tone, systemPrompt, capabilities, optionally inherit from existing persona
* ~~Add `--from <persona>` flag to persona create~~ - Copy/inherit from existing persona as starting point
* ~~Implement `aictl persona edit <name>` command~~ - Open editor to modify persona YAML, validate on save, trigger agent re-synthesis for agents using this persona
* ~~Implement `aictl persona delete <name>` command~~ - Deletes persona with check for agents using it, requires confirmation if in use, shows list of affected agents

### MVP: Tool Discovery & Management

* ~~Implement `aictl tool list` command~~ - Show table of tools in current cluster: NAME, TYPE, STATUS, AGENTS USING with connection health indicators
* ~~Implement `aictl tool install <name>` command~~ - Create LanguageTool resource from template, apply to current cluster, show installation progress, prompt for authentication if needed
* ~~Implement `aictl tool auth <name>` command~~ - Interactive authentication flow for tools requiring credentials (OAuth2, API keys), store in Kubernetes Secret, verify connection
* ~~Implement `aictl tool test <name>` command~~ - Validate tool connectivity and basic functionality, show success/failure with detailed error messages
* ~~Create tool registry mapping~~ - Add sdk/ruby/config/tool_patterns.yaml mapping common keywords to MCP tools: spreadsheet→mcp-gdrive, email→mcp-gmail, slack→mcp-slack, web→mcp-fetch, file→mcp-filesystem
* ~~Create tool installation templates~~ - Add tool YAML templates to sdk/ruby/lib/langop/cli/templates/tools/ for common MCP servers
* ~~Implement `aictl tool delete <name>` command~~ - Delete tool with check for agents using it, require confirmation if in use

### MVP: CLI Enhancements

* ~~Add `--dry-run` flag to agent create~~ - Preview what would be created: show generated YAML, detected tools, selected persona, schedule extraction without applying to cluster
* ~~Integrate tty-* gems for beautiful CLI output~~ - Added TableFormatter.all_agents() for multi-cluster agent views, TableFormatter.status_dashboard() for status command, improved persona show with formatted output using Pastel, all commands now use consistent formatters (tty-spinner, tty-table, tty-prompt, pastel)
* ~~Add kubeconfig detection with validation~~ - Created KubeconfigValidator helper with detection (KUBECONFIG env → ~/.kube/config), cluster connectivity validation, operator deployment check, helpful error messages with installation guide URLs; integrated into ClusterValidator.get_cluster_config() for automatic validation on all cluster commands
* ~~Generate shell completions for all commands~~ - Created bash/zsh/fish completion scripts with dynamic resource name completion (clusters, agents, personas, tools), added `aictl completion` command for easy installation, supports --stdout flag for manual integration

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
