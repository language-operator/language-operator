# Ruby Standards for BASED Project

AUDIENCE: AI Agents
PURPOSE: Establish mandatory conventions for all Ruby-based images in the BASED project
ENFORCEMENT: Apply these standards when creating, modifying, or auditing Ruby code

## Required Project Files

MUST HAVE:
- Dockerfile
- Gemfile
- Gemfile.lock (for applications, not libraries)
- Makefile
- README.md
- .yardopts
- .rubocop.yml
- .gitignore

MUST CREATE:
- lib/ directory for library code
- examples/ directory for usage examples

## Gemfile Requirements

TEMPLATE:
```ruby
source 'https://rubygems.org'

# Production dependencies - use pessimistic versioning
gem 'sinatra', '~> 4.0'
gem 'json', '~> 2.7'

group :development do
  gem 'rubocop', '~> 1.60'
  gem 'rubocop-performance', '~> 1.20'
  gem 'yard', '~> 0.9.37'
end
```

RULES:
- ALL gems MUST use pessimistic operator (~>)
- Production gems MUST lock major version
- Development gems MUST include: rubocop, rubocop-performance, yard
- NixOS incompatible gems MUST be documented with comment
- DO NOT include rdoc (fails on NixOS)

## YARD Documentation Requirements

.yardopts TEMPLATE:
```
--markup markdown
--readme README.md
--output-dir doc
--protected
--private
lib/**/*.rb
server.rb
```

DOCUMENTATION REQUIREMENTS:
- ALL public methods MUST have @param, @return, @example tags
- ALL modules/classes MUST have description and @example
- Target coverage: 60% minimum, 100% for public APIs
- ALWAYS include code examples in documentation

METHOD DOCUMENTATION TEMPLATE:
```ruby
# Brief description
#
# @param name [Type] Description
# @return [Type] Description
# @raise [Exception] When raised
# @example
#   result = method_name('value')
def method_name(name)
```

## Makefile Requirements

MANDATORY TARGETS:
- help (MUST be first/default target)
- build
- run
- test
- lint
- lint-fix
- doc
- doc-serve
- doc-clean
- clean

TEMPLATE:
```makefile
IMAGE_NAME := based/project-name
IMAGE_TAG := latest
IMAGE_FULL := $(IMAGE_NAME):$(IMAGE_TAG)

.PHONY: help
help:
	@echo "Available targets:"
	@echo "  build      - Build the Docker image"
	@echo "  doc        - Generate API documentation with YARD"

.PHONY: doc
doc:
	@echo "Generating documentation with YARD..."
	bundle exec yard doc
	@echo "Documentation generated in doc/"

.PHONY: lint
lint:
	bundle exec rubocop

.PHONY: lint-fix
lint-fix:
	bundle exec rubocop -A
```

## Code Style Requirements

RUBOCOP CONFIGURATION (.rubocop.yml):
```yaml
AllCops:
  NewCops: enable
  TargetRubyVersion: 3.4
  Exclude:
    - 'vendor/**/*'
    - 'doc/**/*'

Style/Documentation:
  Enabled: false

Metrics/MethodLength:
  Max: 20
```

NAMING CONVENTIONS:
- Classes/Modules: PascalCase
- Methods/Variables: snake_case
- Constants: SCREAMING_SNAKE_CASE
- Files: snake_case.rb

CODE STANDARDS:
- Indentation: 2 spaces (NEVER tabs)
- Line length: 120 characters max
- String literals: single quotes (unless interpolation)
- Hash syntax: modern (key: value)

## Testing Requirements

FRAMEWORK: RSpec

DIRECTORY STRUCTURE:
```
spec/
  spec_helper.rb
  lib/
    config_spec.rb
    helpers_spec.rb
  integration/
    server_spec.rb
```

COVERAGE TARGET:
- Minimum: 80%
- Critical paths: 100%

## Docker Requirements

DOCKERFILE TEMPLATE:
```dockerfile
FROM based/base:latest

RUN apk add --no-cache ruby ruby-dev ruby-bundler build-base

WORKDIR /app

COPY Gemfile Gemfile.lock* /app/
RUN bundle install --no-cache

COPY . /app/
RUN chown -R based:based /app

USER based

EXPOSE 80

CMD ["ruby", "server.rb"]
```

## .gitignore Requirements

MUST IGNORE:
```
/doc/
/.yardoc/
/vendor/bundle/
/.bundle/
/.rubocop-*
/coverage/
*.gem
*.rbc
tmp/
.idea/
.vscode/
```

## Security Requirements

MUST:
- Use environment variables for configuration
- Validate all user input
- Use Shellwords.escape for shell commands
- NEVER commit secrets

## Error Handling Requirements

PATTERN:
```ruby
def method_name(input)
  validate_input(input)
  
  begin
    result = operation(input)
  rescue SpecificError => e
    logger.error("Failed: #{e.message}")
    return default_value
  end
  
  result
end
```

DO NOT:
- Use bare rescue
- Rescue Exception
- Silent failures with rescue nil

## Audit Checklist

When auditing Ruby code, verify:
- [ ] Gemfile uses ~> for all dependencies
- [ ] .yardopts file exists
- [ ] Makefile has all required targets
- [ ] YARD comments on public APIs
- [ ] RuboCop passes with zero offenses
- [ ] Tests exist and pass
- [ ] Documentation generates without errors
- [ ] Dockerfile follows template
- [ ] .gitignore includes standard exclusions
- [ ] No secrets in code
- [ ] Environment variables documented

## Reference Implementation

LOCATION: types/server/

FILES TO REFERENCE:
- types/server/Gemfile - Dependency management
- types/server/.yardopts - YARD configuration
- types/server/Makefile - Standard targets
- types/server/lib/config.rb - YARD documentation example
- types/server/lib/helpers.rb - Module documentation
- types/server/lib/shell.rb - Method documentation

VERIFY IMPLEMENTATION:
```bash
cd types/server
make doc        # Should generate without errors
make lint       # Should pass with zero offenses
```

## Agent Actions

When creating new Ruby project:
1. Copy structure from types/server/
2. Create all required files from templates
3. Run make lint and fix all issues
4. Run make doc and verify generation
5. Add YARD comments to all public methods
6. Update README.md with configuration table
7. Verify build succeeds

When modifying Ruby code:
1. Add YARD comments if missing
2. Run make lint-fix
3. Run make test
4. Update documentation if API changed
5. Verify make doc still works

When auditing Ruby code:
1. Check all items in audit checklist
2. Generate report of missing items
3. Fix high-priority issues
4. Document any exceptions

## Maintenance

UPDATE THIS DOCUMENT when:
- Ruby version changes
- New standard dependencies added
- Documentation requirements change
- Testing conventions evolve

Last updated: 2025-01-24
