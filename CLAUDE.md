# Standards

## Git
- When committing code, use semantic nouns like "feat: " with a brief one-line summary, not a multiline essay.

# Project Defaults

## Registry
- Private registry: git.theryans.io
- Registry namespace: langop
- Example image: git.theryans.io/langop/tool:latest

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
- **Langop SDK**: 0.1.x (managed in sdk/ruby/langop.gemspec)

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
- **Implementation**: SDK gem (`langop`) pins to the working fork version
- **Impact**: All components use the SDK gem, so they inherit the correct dependency automatically

### Dependency Management Pattern
- **Langop SDK gem** (`sdk/ruby/`) is the single source of truth for all Ruby dependencies
- **Components** depend on the pre-installed `langop` gem from the base image
- **No duplicate Gemfiles**: Components inherit dependencies from the SDK
- **Version control**: Update versions in SDK only, components get updates automatically

## Code Architecture

### DRY Principle: Don't Repeat Yourself
- **Problem Solved**: Eliminated 1,313 lines of duplicate code between SDK and components
- **Pattern**: SDK gem provides all core functionality, components are thin wrappers
- **Namespace Aliasing**: Components can alias SDK classes for backwards compatibility
- **Pre-installed Gem**: `langop` gem is installed in `langop/ruby` base image, available globally

### Component Hierarchy
```
langop/base       - Alpine + Ruby + Bundler
  └─ langop/ruby  - base + langop gem pre-installed
      ├─ langop/client  - ruby + client wrapper
      ├─ langop/agent   - ruby + agent framework
      ├─ langop/tool    - ruby + MCP server framework
      └─ langop/model   - ruby + LiteLLM proxy
```

### Inheritance Pattern
- Agents inherit from `Langop::Client::Base` (not `Based::Client::Base`)
- All core logic lives in SDK (`sdk/ruby/lib/langop/`)
- Components provide deployment-specific wrappers and configurations

## Naming Conventions

### Proof-of-Concept Migration
- **Old name**: "Based" (proof-of-concept name)
- **Current name**: "Langop" (production name)
- **Migration complete**: All code, documentation, and configuration uses "Langop"
- **Historical references**: Only in BACKLOG.md for context

### Module Naming
- Top-level module: `Langop`
- Client code: `Langop::Client`
- Agent code: `Langop::Agent`
- DSL code: `Langop::Dsl`
- Tool code: `Langop::ToolLoader`