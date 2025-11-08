# End-to-End (E2E) Tests

This directory contains end-to-end integration tests for the Language Operator.

## Overview

The e2e tests verify the complete flow:
```
LanguageAgent CRD → Operator → Synthesis → ConfigMap → Deployment → Pod → Execution
```

## Test Categories

### Synthesis Quality Tests (`synthesis_test.go`)
- Validates synthesis output quality
- Tests schedule extraction
- Tests tool detection
- Tests persona selection
- Leverages existing test fixtures from `test/instructions/`

### Agent Execution Tests (`agent_execution_test.go`)
- Full lifecycle testing of LanguageAgent resources
- Verifies ConfigMap creation
- Verifies Deployment creation
- Verifies Pod startup
- Tests different execution modes (scheduled, autonomous)
- Tests workspace configuration

### Error Handling Tests (`error_handling_test.go`)
- Synthesis failures
- Invalid instructions
- Missing tool references
- Reconciliation retry logic
- Concurrent agent creation

## Prerequisites

### Required Tools
- Go 1.21+
- kubectl
- kubebuilder (for envtest)

### Environment Variables
- `SYNTHESIS_ENDPOINT` - LLM endpoint for synthesis (defaults to mock in tests)
- `SYNTHESIS_MODEL` - Model to use for synthesis
- `KUBEBUILDER_ASSETS` - Path to kubebuilder test binaries (auto-downloaded)

## Running Tests

### Run all e2e tests
```bash
make test-e2e
```

### Run specific test file
```bash
cd test/e2e
go test -v -run TestSynthesisQuality
```

### Run with short mode (skips actual cluster tests)
```bash
go test -short ./test/e2e/...
```

### Run specific test case
```bash
go test -v -run TestAgentExecution/scheduled
```

## Test Infrastructure

### Test Environment (`helpers.go`)
- Uses `controller-runtime/envtest` for lightweight Kubernetes API server
- Automatically starts/stops test environment
- Provides helper functions for creating resources and waiting for conditions

### Mock LLM Service (`mock_llm.go`)
- Simulates LLM API for predictable synthesis output
- Extracts schedules and tools from instructions
- Generates valid Ruby DSL code
- Eliminates API costs during testing
- Enables deterministic test results

## Writing New Tests

### Basic Test Structure
```go
func TestMyFeature(t *testing.T) {
    if testing.Short() {
        t.Skip("Skipping e2e test in short mode")
    }

    // Setup test environment
    env := SetupTestEnvironment(t)
    defer env.Teardown(t)

    // Start mock LLM (optional)
    mockLLM := NewMockLLMService(t)
    defer mockLLM.Close()
    os.Setenv("SYNTHESIS_ENDPOINT", mockLLM.URL())
    defer os.Unsetenv("SYNTHESIS_ENDPOINT")

    // Create test namespace
    namespace := "test-my-feature"
    env.CreateNamespace(t, namespace)
    defer env.DeleteNamespace(t, namespace)

    // Create resources and test...
}
```

### Helper Functions
- `SetupTestEnvironment(t)` - Creates test k8s environment
- `env.CreateNamespace(t, name)` - Creates namespace
- `env.CreateLanguageAgent(t, agent)` - Creates LanguageAgent
- `env.WaitForCondition(t, ns, name, type, status)` - Waits for condition
- `env.WaitForConfigMap(t, ns, name)` - Waits for ConfigMap
- `env.WaitForDeployment(t, ns, name)` - Waits for Deployment
- `env.WaitForPod(t, ns, label)` - Waits for Pod
- `env.GetPodLogs(t, ns, name)` - Retrieves pod logs

## CI/CD Integration

### GitHub Actions / Forgejo CI
```yaml
name: E2E Tests
on: [push, pull_request]
jobs:
  e2e:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v3
      - uses: actions/setup-go@v4
        with:
          go-version: '1.21'
      - name: Run E2E tests
        run: make test-e2e
```

## Troubleshooting

### Tests hang waiting for conditions
- Check operator logs for reconciliation errors
- Verify CRDs are installed correctly
- Check that mock LLM service is responding

### envtest fails to start
- Ensure kubebuilder assets are downloaded: `make envtest`
- Check `KUBEBUILDER_ASSETS` environment variable
- Try manually running: `setup-envtest use`

### Synthesis fails in tests
- Check `SYNTHESIS_ENDPOINT` is set correctly
- Verify mock LLM service is running
- Check synthesis request format

## Future Improvements

- [ ] Add tool integration tests (requires test tool implementation)
- [ ] Add persona system tests (requires persona CRD implementation)
- [ ] Add network policy tests (requires CNI with NetworkPolicy support)
- [ ] Add multi-agent cluster tests
- [ ] Add metrics and monitoring tests
- [ ] Add upgrade/downgrade tests
- [ ] Add performance/load tests

## References

- [Controller Runtime EnvTest](https://book.kubebuilder.io/reference/envtest.html)
- [Kubernetes E2E Testing](https://github.com/kubernetes/community/blob/master/contributors/devel/sig-testing/e2e-tests.md)
- Issue #40: Missing end-to-end integration tests
