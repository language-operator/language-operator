# Integration Tests

Simplified integration tests for the Language Operator that focus on core functionality without requiring a full Kubernetes cluster.

## Test Organization

### Unit Tests (Fast - No K8s Required)
- **synthesis_test.go** - Tests synthesis logic with mock LLM
- **cni_test.go** - Tests CNI detection logic with mock K8s client

### Integration Tests (Moderate - Uses envtest)
- **agent_lifecycle_test.go** - Tests LanguageAgent CRD lifecycle

## Running Tests

```bash
# Run all integration tests
make test-integration

# Run only fast unit tests (no envtest)
make test-unit

# Run specific test
cd test/integration
go test -v -run TestSynthesis
```

## Design Principles

1. **DRY** - Shared test fixtures and helpers
2. **Fast** - Unit tests < 1 second, integration tests < 5 minutes
3. **Table-driven** - One test function with multiple scenarios
4. **Focused** - Each test validates one thing clearly
5. **Deterministic** - No flaky tests, predictable mocks

## Writing Tests

Use table-driven patterns:

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
