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

## In Progress 🚧

(none)

## Next Up (Working Demo) 📋

### Critical Path to Working Demo

* Implement Persona integration in LanguageAgent controller
  * Fetch LanguagePersona resource by personaRef
  * Merge persona.systemPrompt with agent.instructions
  * Apply persona rules, examples, constraints to ConfigMap
  * Test with example manifests

* Fix Ruby SDK ruby_llm dependency issues
  * Investigate ruby_llm gem availability/status
  * Fix require statements in client/base.rb
  * Test SDK functionality locally
  * Update component Gemfiles if needed

* Deploy operator to cluster with latest fixes
  * Build operator image with persona integration
  * Update Helm chart if needed
  * Deploy to kube-system namespace

* Run end-to-end demo
  * Deploy LanguageCluster
  * Deploy LanguageModel with persona
  * Deploy LanguageTool (sidecar mode)
  * Deploy LanguageAgent with personaRef
  * Verify agent pod runs with sidecar + workspace
  * Verify agent can execute tasks

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

* Re-enable automated testing in CI
  * Uncomment .github/workflows/test.yaml
  * Fix broken tests
  * Add coverage reporting

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

* Testing completely disabled in CI (deferred for production readiness phase)
* LanguageClient controller incomplete (ingress/auth/session management)
* DNS resolution is snapshot-based (refreshes on reconciliation, not continuous)
* Wildcard DNS (*.example.com) only resolves base domain

## Notes

* Focus: Get working demo running end-to-end
* Priority: Features > Infrastructure > Testing > Polish
* Target: Agent executing tasks with tools, models, and personas
