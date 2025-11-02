# Requirement: Makefile Test Target

**Status**: REQUIRED
**Applies to**: All Makefiles in repository
**RFC 2119**: MUST
**Check**: Run `make test` in each directory with a Makefile

## Description

For any Makefile in this repo, a `test` target MUST be provided that runs all relevant tests for that component. The test target MUST execute without requiring manual intervention and MUST exit with a non-zero status code if any tests fail.

## Example Implementation

### Go Projects (Operator)
```makefile
test: ## Run unit tests
	go test ./... -coverprofile=coverage.out
	go vet ./...
```

### Ruby Projects (Components/Agents)
```makefile
test: ## Run RSpec tests
	bundle exec rspec
```

### Projects with Multiple Test Types
```makefile
test: test-unit test-integration ## Run all tests

test-unit: ## Run unit tests only
	go test -short ./...

test-integration: ## Run integration tests
	go test -run Integration ./...
```

## Requirements

1. **MUST** run all applicable tests (unit, integration, linting)
2. **MUST** exit with non-zero code on test failure
3. **MUST** be runnable without arguments (`make test` should just work)
4. **SHOULD** include coverage reporting where applicable
5. **SHOULD** document sub-targets (e.g., `test-unit`, `test-integration`) in help output

## Compliance

To check compliance:
```bash
# Find all Makefiles
find . -name Makefile -o -name "*.mk"

# Test each one
make -C <directory> test
echo $?  # Should be 0 if tests pass, non-zero if tests fail
```

## Rationale

- Provides consistent testing interface across all components
- Enables CI/CD pipelines to run tests uniformly
- Makes it easy for developers to validate changes locally
- Supports Test-Driven Development (TDD) workflows
- Ensures code quality through automated testing

---

**Note**: When implementing this requirement, commit changes with a one-line summary.
