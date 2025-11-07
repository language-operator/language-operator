# Standards

## Git
- When committing code, use semantic nouns like "feat: " with a brief one-line summary, not a multiline essay.

# Project Defaults

## Registry
- Private registry: git.theryans.io
- Registry namespace: language-operator
- Example image: git.theryans.io/language-operator/tool:latest

## Infrastructure
- Kubernetes cluster: k3s on 5 nodes (dl1-dl5)
- SSH access: james@dl1, james@dl2, james@dl3, james@dl4, james@dl5
- Master node: dl1 (runs k3s service)
- Agent nodes: dl2-dl5 (run k3s service, not k3s-agent)

## Operator
- Operator namespace: kube-system
- Operator deployment: language-operator
- CRDs: LanguageCluster, LanguageModel, LanguageTool, LanguageAgent

## Testing
- Never commit code that hasn't been tested
- Verify scripts must work end-to-end before committing
- Registry authentication must be verified before testing image pulls
- All Makefiles MUST have a `test` target (see requirements/makefile/MUST-have-test-target.md)
- All Makefiles MUST have a `help` target as default (see requirements/makefile/MUST-have-help-target.md)

## Version Standards
- **Ruby**: 3.4 (components using 3.3 should be upgraded)
- **RuboCop**: ~> 1.60
- **RuboCop Performance**: ~> 1.20
- **YARD**: ~> 0.9.37
- **Language Operator SDK**: 0.1.x (published to git.theryans.io private gem registry)

## DRY Tooling

### Shared Makefile
- **Location**: `Makefile.common` at repository root
- **Usage**: Components include with `include ../../Makefile.common`
- **Purpose**: Eliminates 300-400 lines of duplicated Makefile code
- **Standard targets**: build, scan, shell, test, lint, lint-fix, doc, doc-serve, doc-clean, clean
- **Override**: Components can override any target by redefining it after the include

### Shared RuboCop Config
- **Location**: `.rubocop.yml` at repository root
- **Usage**: Components inherit with `inherit_from: ../../.rubocop.yml`
- **Purpose**: Eliminates 150+ lines of duplicated configuration
- **Standards**: Ruby 3.4, NewCops enabled, sensible metric limits
- **Override**: Components can override specific rules for stricter or looser limits

## Ruby Dependencies

### ruby_llm Gem Resolution
- **Issue**: The `ruby_llm` gem had circular dependency issues and version conflicts
- **Solution**: Use the Anthropic fork at `github.com/anthropics/ruby_llm` instead of the original
- **Implementation**: SDK gem (`language-operator`) pins to the working fork version
- **Impact**: All components use the SDK gem, so they inherit the correct dependency automatically

### Dependency Management Pattern
- **Language Operator SDK gem** is published to the private gem registry at `git.theryans.io`
- **SDK Repository**: Separate repository manages versioning and gem publishing
- **Components** install the `language-operator` gem via Gemfile with private source
- **Authentication**: Base image contains read-only credentials for gem registry access
- **Version control**: Components specify SDK version in Gemfile (e.g., `~> 0.1.0`)

## Code Architecture

### DRY Principle: Don't Repeat Yourself
- **Problem Solved**: Eliminated duplicate code by publishing SDK as gem
- **Pattern**: SDK gem provides all core functionality, components install via Gemfile
- **Gem Source**: Private registry at `https://git.theryans.io/api/packages/language-operator/rubygems`
- **No Layered Images**: Each component installs dependencies directly from base Alpine image

### Component Hierarchy
```
langop/base - Alpine + tini + su-exec + langop user
  ├─ langop/tool      - base + Ruby + language-operator gem + tool dependencies
  ├─ langop/client    - base + Ruby + language-operator gem + client dependencies
  ├─ langop/agent     - base + Ruby + language-operator gem + agent runtime (synthesis-driven)
  ├─ langop/web-tool  - base + Ruby + language-operator gem + web tool code
  ├─ langop/email-tool - base + Ruby + language-operator gem + email tool code
  └─ langop/model     - base + Python + LiteLLM proxy
```

### Build Pattern
- All Ruby components start from `langop/base:latest`
- Install Ruby + bundler + build-base via apk
- Configure bundle with registry credentials (build args)
- Copy Gemfile and run `bundle install` to get `language-operator` gem
- Each component is self-contained and explicit

### Inheritance Pattern
- Agents inherit from `LanguageOperator::Agent::Base`
- All core logic lives in published SDK gem
- Components provide deployment-specific wrappers and configurations

## Naming Conventions

### Proof-of-Concept Migration
- **Old name**: "Based" (proof-of-concept name)
- **Current name**: "Langop" (production name)
- **Migration complete**: All code, documentation, and configuration uses "Langop"
- **Historical references**: Only in BACKLOG.md for context

### Module Naming
- **Gem name**: `language-operator` (installed via `gem install language-operator`)
- **CLI command**: `aictl` (the command-line interface)
- **Top-level module**: `LanguageOperator`
- **Client code**: `LanguageOperator::Client`
- **Agent code**: `LanguageOperator::Agent`
- **DSL code**: `LanguageOperator::Dsl`
- **Tool code**: `LanguageOperator::ToolLoader`