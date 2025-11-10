# Testing Guide

This document describes the testing strategy for the Language Operator project.

## Test Organization

The project uses a layered testing approach for speed and reliability:

```
test/
└── integration/          # Fast integration tests
    ├── synthesis_test.go # Synthesis logic tests (no K8s)
    ├── cni_test.go       # CNI detection tests (fake K8s)
    ├── mock_llm.go       # Mock LLM service
    └── fixtures.go       # Shared test data
```

## Prerequisites

Before running tests, install Ruby dependencies:

```bash
# Install language-operator gem and dependencies
bundle install
```

This installs the `language-operator` gem from the private registry, which includes the Ruby AST validator required for synthesis tests.

## Running Tests

### Quick Start (Recommended)

```bash
# Install dependencies first
bundle install

# Run fast unit tests (< 1 minute)
make test-unit

# Run all integration tests (< 5 minutes)
make test-integration
```

### All Test Commands

```bash
# Fast unit tests (no Kubernetes required)
make test-unit

# Integration tests with fake Kubernetes client
make test-integration

# Operator unit tests
cd src && make test
```

## Test Categories

### 1. Unit Tests (Fast - No Kubernetes)

**Location:** `test/integration/*_test.go`
**Duration:** < 1 second per test
**Purpose:** Test individual functions and logic

**What to test:**
- Synthesis logic with mock LLM
- Code validation
- Error handling
- Data parsing and extraction

**Example:**
```go
func TestSynthesisQuality(t *testing.T) {
    mockLLM := NewMockLLMService(t)
    defer mockLLM.Close()

    // Test synthesis without K8s cluster
    synthesizer := synthesis.NewSynthesizer(mockChatModel, logr.Discard())
    resp, err := synthesizer.SynthesizeAgent(ctx, req)
    // ...
}
```

### 2. Integration Tests (Moderate - Fake Kubernetes)

**Location:** `test/integration/*_test.go`
**Duration:** < 5 minutes total
**Purpose:** Test component interactions with fake K8s

**What to test:**
- CNI detection logic
- K8s resource parsing
- Client interactions
- Multi-component workflows

**Example:**
```go
func TestCNIDetection(t *testing.T) {
    // Use fake Kubernetes client
    clientset := fake.NewSimpleClientset(objects...)
    caps, err := cni.DetectNetworkPolicySupport(ctx, clientset)
    // ...
}
```

## Test Patterns

### Table-Driven Tests

Use table-driven tests to reduce duplication:

```go
func TestFeature(t *testing.T) {
    tests := []struct {
        name     string
        input    string
        expected string
    }{
        {"case1", "input1", "output1"},
        {"case2", "input2", "output2"},
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            result := feature(tt.input)
            assert.Equal(t, tt.expected, result)
        })
    }
}
```

### Shared Fixtures

Use shared test data from `fixtures.go`:

```go
// Reuse common test scenarios
for _, scenario := range TestScenarios {
    t.Run(scenario.Name, func(t *testing.T) {
        // Test using predefined scenario
    })
}
```

### Mock Services

Use the mock LLM service for deterministic tests:

```go
mockLLM := NewMockLLMService(t)
defer mockLLM.Close()

mockChatModel := NewMockChatModel(mockLLM)
synthesizer := synthesis.NewSynthesizer(mockChatModel, logr.Discard())
```

## Writing New Tests

### 1. Decide Test Type

- **Unit test** - Tests single function, no external dependencies → `test/integration/`
- **Integration test** - Tests multiple components, fake K8s → `test/integration/`

### 2. Use Existing Patterns

- Start with table-driven test structure
- Reuse fixtures from `fixtures.go`
- Use mock LLM service for synthesis tests
- Keep tests focused and fast

### 3. Test Naming

```go
// Good names describe what is being tested
func TestSynthesisWithValidInstructions(t *testing.T) { }
func TestCNIDetectionCilium(t *testing.T) { }

// Table-driven tests use subtests
t.Run("valid input", func(t *testing.T) { })
```

### 4. Assertions

Use testify for clear assertions:

```go
assert.Equal(t, expected, actual, "descriptive message")
assert.NoError(t, err, "should not error")
assert.Contains(t, haystack, needle, "should contain")
require.NoError(t, err, "stops test on failure")
```

## Test Coverage

Run tests with coverage:

```bash
cd test/integration
go test -cover ./...
```

Target: ≥80% coverage for new code

## CI/CD Integration

The integration tests run in CI:

```yaml
# .github/workflows/test.yml
- name: Run integration tests
  run: make test-integration
```

## Troubleshooting

### Tests are slow

- Use `test-unit` for quick feedback
- Check if you're using real K8s instead of fake client
- Consider breaking large tests into smaller ones

### Mock LLM not working

- Ensure mock service is started before use
- Check that `defer mockLLM.Close()` is called
- Verify mock generates expected output format

### Import errors

- Run `go mod tidy` in test directory
- Check that replace directive points to correct path
- Ensure all dependencies are in go.mod

### Fake K8s client issues

- Use `fake.NewSimpleClientset()` for unit tests
- Avoid envtest unless absolutely necessary
- Mock external K8s interactions

### Gem installation failures

If `bundle install` fails with authentication errors:

```bash
# Check if private registry credentials are configured
bundle config get https://git.theryans.io/api/packages/language-operator/rubygems

# If not set, you need read access to the private gem registry
# Contact the repository maintainer for credentials
```

If synthesis tests fail with "validator error":

```bash
# Ensure bundle install was run successfully
bundle install

# Verify language-operator gem is installed
bundle list | grep language-operator

# Should show: language-operator (0.1.27)
```

## Best Practices

1. **Keep tests fast** - Target < 1 second per test
2. **Test one thing** - Each test should validate one behavior
3. **Use mocks** - Avoid external dependencies
4. **DRY** - Share fixtures and helpers
5. **Clear names** - Test names should describe what is tested
6. **Table-driven** - Use when testing multiple similar cases
7. **Deterministic** - Tests should always produce same result
8. **No flaky tests** - Fix or skip unstable tests

## Future Improvements

- [ ] Add performance benchmarks
- [ ] Add mutation testing
- [ ] Add property-based testing
- [ ] Improve test documentation
- [ ] Add visual test reports
