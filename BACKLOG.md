# Language Operator Backlog

Simple chronological checklist of what to do next.

## Completed ✅

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
* ~~Set up CI/CD for automated image builds~~
* ~~Publish Ruby gem to private registry~~
* ~~Build and push all component images to registry~~
* ~~Implement Persona integration in LanguageAgent controller~~
* ~~Fix Ruby SDK ruby_llm dependency issues~~
* ~~Fix sidecar tool injection bug~~
* ~~Add environment variable config support (MCP_SERVERS, MODEL_ENDPOINTS)~~
* ~~Add TCP readiness probes to sidecar containers~~
* ~~Deploy operator to cluster with all fixes~~
* ~~Run end-to-end demo (LanguageCluster + Model + Tool + Agent)~~
* ~~Verify agent pod runs with sidecar + workspace~~
* ~~Re-enable automated testing in CI~~
* ~~Add basic controller unit tests~~
* ~~Standardize all Makefiles with Docker targets (build, scan, shell, run)~~
* ~~Update .gitignore for Go build artifacts~~
* ~~Add retry logic to agent connection code~~
  * ~~Handle startup race conditions gracefully~~
  * ~~Retry MCP server connections on failure~~
  * ~~Add exponential backoff~~
* ~~DRY Phase 1: Fix agent inheritance~~
  * ~~Change components/agent to inherit from Langop::Client::Base~~
  * ~~Update components/agent/Gemfile to depend on langop gem~~
  * ~~Fix require statements in agent code~~
* ~~DRY Phase 2: Consolidate client code (363 lines removed)~~
  * ~~Delete duplicate client files base.rb and config.rb~~
  * ~~Create namespace wrapper Based::Client = Langop::Client~~
  * ~~Remove duplicate gem dependencies~~
* ~~DRY Phase 3: Consolidate DSL code (950 lines removed)~~
  * ~~Delete 8 duplicate DSL files (adapter, config, helpers, http, parameter_definition, registry, shell, tool_definition)~~
  * ~~Create namespace wrapper Based::Dsl = Langop::Dsl~~
  * ~~Update tool server and loader to use gem DSL~~
* ~~DRY Phase 4: Move reusable code to SDK for better developer experience~~
  * ~~Move Context, ExecutionContext, and ToolLoader to SDK~~
  * ~~Update component wrapper to alias SDK classes~~
  * ~~SDK now provides complete tool development experience~~

## In Progress 🚧

(none)

## Next Up 📋

### Immediate Priority: Fix Code Architecture

The DRY consolidation (see below) must be done FIRST before building/testing can proceed. The current broken inheritance and code duplication will cause runtime issues.

### Documentation Updates

* ~~Update STATUS.md~~
  * ~~Add Ruby SDK & CI/CD section~~
  * ~~Mark gem publishing as complete~~
  * ~~Update component image status~~
  * ~~Document persona integration when done~~
  * ~~Update "Last updated" date~~
  * ~~Document Makefile standardization~~

* ~~Update README.md~~
  * ~~Add note about DNS resolution timing~~
  * ~~Add note about wildcard DNS behavior (*.example.com)~~
  * Persona examples already complete ✅

* Update CLAUDE.md
  * Document ruby_llm dependency findings
  * Add any new project conventions discovered

## Future Enhancements 🔮

### Code Quality & DRY (High Priority)

**Problem**: 1,600+ lines of duplicated code between SDK gem and components. Components copy-paste from SDK instead of using it as dependency.

* ~~**Phase 1: Fix Agent Inheritance (Quick Win)**~~ ✅ COMPLETE
  * ~~Change `components/agent` to inherit from `Langop::Client::Base` instead of `Based::Client::Base`~~
  * ~~Update `components/agent/Gemfile` to depend on `langop` gem (uses pre-installed gem from base image)~~
  * ~~Fix require statements in agent code~~
  * ~~Files: `components/agent/lib/langop/agent.rb`, `components/agent/lib/langop/agent/base.rb`~~
  * **Impact**: ✅ Fixed broken inheritance, agent now gets client improvements automatically

* ~~**Phase 2: Consolidate Client Code (363 lines)**~~ ✅ COMPLETE
  * ~~Delete duplicate files: `components/client/lib/based/client/base.rb`, `components/client/lib/based/client/config.rb`~~
  * ~~Update `components/client/Gemfile` to depend on `langop` gem (uses pre-installed gem from base image)~~
  * ~~Create namespace wrapper: `Based::Client = Langop::Client` for backwards compatibility~~
  * ~~Update `components/client/lib/based_client.rb` to require and wrap gem~~
  * **Impact**: ✅ Removed 363 lines of duplicate code, single source of truth for client logic

* ~~**Phase 3: Consolidate DSL Code (950 lines)**~~ ✅ COMPLETE
  * ~~Delete 8 duplicate DSL files from `components/tool/lib/based/dsl/`~~
  * ~~Update `components/tool/Gemfile` to depend on `langop` gem (uses pre-installed gem from base image)~~
  * ~~Create namespace wrapper: `Based::Dsl = Langop::Dsl` for backwards compatibility~~
  * ~~Keep component-specific files: `context.rb`, `execution_context.rb`, `tool_loader.rb`, `server.rb`~~
  * ~~Update tool server and loader to use gem DSL~~
  * **Impact**: ✅ Removed 950 lines of duplicate code, single source of truth for DSL logic

* ~~**Phase 4: Move Component-Specific Code to SDK**~~ ✅ COMPLETE
  * ~~Move `ExecutionContext` to SDK as `Langop::Dsl::ExecutionContext`~~
  * ~~Move `Context` to SDK as `Langop::Dsl::Context`~~
  * ~~Move `ToolLoader` to SDK as `Langop::ToolLoader`~~
  * ~~Keep `server.rb` in component (Sinatra-specific)~~
  * **Impact**: ✅ Better code organization, classes now reusable across all tools and projects

### End-to-End Testing & Deployment (After DRY Consolidation)

**Prerequisites**: Phases 1-3 of DRY consolidation must be complete before this work can begin.

* **Build and push updated component images**
  * Rebuild `langop/client` image with consolidated code
  * Rebuild `langop/agent` image with fixed inheritance
  * Rebuild `langop/tool` image with consolidated DSL
  * Rebuild agent implementations (cli, headless, web)
  * Push images to registry
  * Verify images pull successfully in cluster

* **Deploy test environment**
  * Create test LanguageCluster
  * Deploy LanguageModel with API key
  * Deploy LanguageTool in sidecar mode
  * Deploy LanguageAgent with toolRefs and modelRefs
  * Verify all resources reach "Running" state

* **Test connectivity and integration**
  * Check agent pod logs for successful MCP connection (no errors after retry)
  * Verify sidecar tool container is healthy
  * Verify agent can connect to model proxy
  * Test tool invocation from agent
  * Test LLM API calls through proxy

* **End-to-end task execution test**
  * Deploy agent with simple task/goal
  * Verify agent can call tools via MCP
  * Verify agent can call LLM via model proxy
  * Confirm task completes successfully
  * Check logs for complete execution flow

* **Test persona integration**
  * Deploy LanguagePersona resource
  * Deploy agent with personaRef
  * Verify persona environment variables set correctly
  * Test agent behavior matches persona configuration

### Production Readiness

* Add more comprehensive test coverage
  * Add tests for LanguageAgent controller
  * Add tests for LanguageModel controller
  * Add tests for LanguageCluster controller
  * Add integration tests

* Complete LanguageClient controller
  * Implement ingress reconciliation
  * Add authentication/session management
  * Document usage patterns

* Add monitoring and observability
  * Prometheus metrics
  * Logging best practices
  * Health checks

### Advanced Features

* Implement cost tracking
  * Usage metrics in status
  * Token counting
  * Cost estimation

* Add safety guardrails
  * Content filtering
  * Rate limiting
  * Blocked topics enforcement

* Implement event-driven triggers
  * Webhook support
  * Event sources
  * Trigger conditions

* Add advanced model features
  * Load balancing across models
  * Fallback models
  * Response caching
  * Multi-region support

* Add advanced tool features
  * Horizontal Pod Autoscaling (HPA)
  * PodDisruptionBudget (PDB)
  * Custom health probes

* Memory backend integration
  * Redis support
  * Postgres support
  * S3 support

## Known Issues 🐛

* LanguageClient controller incomplete (ingress/auth/session management)
* DNS resolution is snapshot-based (refreshes on reconciliation, not continuous)
* Wildcard DNS (*.example.com) only resolves base domain
* ~~Agent logs connection error on first startup~~ - FIXED with retry logic
* ~~**Broken inheritance**: Agent inherits from `Based::Client::Base` instead of `Langop::Client::Base`~~ - FIXED (Phase 1)
* ~~**Code duplication**: 1,600+ lines duplicated between SDK gem and components~~ - FIXED (Phases 1-3 complete, removed 1,313 lines)

## Notes

* Focus: Get working demo running end-to-end
* Priority: Features > Infrastructure > Testing > Polish
* Target: Agent executing tasks with tools, models, and personas
