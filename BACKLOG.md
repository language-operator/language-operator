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

## In Progress 🚧

(none)

## Next Up (Working Demo) 📋

### Critical Path to Working Demo

* Add retry logic to agent connection code
  * Handle startup race conditions gracefully
  * Retry MCP server connections on failure
  * Add exponential backoff

* Verify agent can execute tasks end-to-end
  * Test with actual task execution
  * Verify tool calls work correctly
  * Confirm persona behavior

### Documentation Updates

* Update STATUS.md
  * Add Ruby SDK & CI/CD section
  * Mark gem publishing as complete
  * Update component image status
  * Document persona integration when done
  * Update "Last updated" date

* Update README.md
  * Add note about DNS resolution timing
  * Add note about wildcard DNS behavior (*.example.com)
  * Update persona examples when integration complete

* Update CLAUDE.md
  * Document ruby_llm dependency findings
  * Add any new project conventions discovered

## Future Enhancements 🔮

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
* Agent logs connection error on first startup (cosmetic - both containers start simultaneously, agent retries and succeeds)

## Notes

* Focus: Get working demo running end-to-end
* Priority: Features > Infrastructure > Testing > Polish
* Target: Agent executing tasks with tools, models, and personas
