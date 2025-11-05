# E2E Testing for Language Operator

This directory contains end-to-end (E2E) tests for the Language Operator SDK, specifically testing the full user workflow through the `aictl` CLI.

## Overview

The E2E tests verify the complete agent lifecycle:

1. **Cluster creation** - Create a language cluster via `aictl cluster create`
2. **Agent creation** - Create an agent from natural language via `aictl agent create`
3. **Synthesis verification** - Verify the operator synthesizes agent code
4. **Deployment verification** - Verify agent pods are deployed and running
5. **Log access** - Verify agent logs are accessible via `aictl agent logs`
6. **Agent editing** - Edit agent instructions via `aictl agent edit`
7. **Re-synthesis** - Verify the operator re-synthesizes after edits
8. **Cleanup** - Delete agents and clusters

## Prerequisites

### Required

- **Kubernetes cluster** - Must have a working k8s cluster (k3s, minikube, kind, etc.)
- **kubectl** - Must be in PATH and configured
- **Language Operator** - Must be installed in the cluster
- **aictl CLI** - Must be installed and in PATH

### Setup

```bash
# Install the SDK gem (includes aictl)
cd sdk/ruby
bundle install
gem build langop.gemspec
gem install langop-*.gem

# Verify aictl is available
aictl version

# Verify operator is installed
kubectl get deployment -n kube-system language-operator
```

## Running E2E Tests

### Run all E2E tests

```bash
cd sdk/ruby
bundle exec rake e2e
```

### Run specific E2E test file

```bash
bundle exec rspec spec/e2e/agent_lifecycle_spec.rb --tag e2e
```

### Run with verbose output

```bash
E2E=true bundle exec rspec spec/e2e/agent_lifecycle_spec.rb -fd
```

### Run without cleanup (for debugging)

```bash
E2E_SKIP_CLEANUP=true bundle exec rake e2e
```

## Configuration

E2E tests can be configured via environment variables:

| Variable | Default | Description |
|----------|---------|-------------|
| `E2E` | `false` | Set to `true` to enable E2E tests |
| `E2E_NAMESPACE` | `default` | Kubernetes namespace for test agents |
| `E2E_SKIP_CLEANUP` | `false` | Skip cleanup after tests (useful for debugging) |
| `E2E_SYNTHESIS_TIMEOUT` | `300` | Timeout in seconds for agent synthesis |
| `E2E_POD_TIMEOUT` | `120` | Timeout in seconds for pod readiness |

Example:

```bash
E2E=true E2E_NAMESPACE=e2e-tests E2E_SYNTHESIS_TIMEOUT=600 bundle exec rake e2e
```

## Test Structure

### Helper Modules

- **`E2E::AictlHelper`** - Helper methods for running aictl commands
  - `run_aictl(command)` - Run aictl and capture output
  - `wait_for_condition(timeout:, &block)` - Wait for a condition
  - `agent_exists?(name)` - Check if agent exists
  - `agent_synthesized?(name)` - Check synthesis status
  - `agent_running?(name)` - Check pod status
  - `get_agent_logs(name)` - Retrieve logs
  - `cleanup_test_resources(prefix)` - Clean up test resources

- **`E2E::Config`** - Test configuration
  - `test_prefix` - Unique prefix for test resources
  - `test_namespace` - Kubernetes namespace
  - `synthesis_timeout` - Synthesis timeout
  - `pod_ready_timeout` - Pod ready timeout

### Test Files

- **`agent_lifecycle_spec.rb`** - Full agent lifecycle E2E test
  - Creates cluster and agent
  - Verifies synthesis and deployment
  - Tests log access
  - Verifies edit and re-synthesis flow
  - Cleans up resources

## Writing New E2E Tests

To write a new E2E test:

1. Create a new spec file in `spec/e2e/`
2. Require the E2E helper: `require_relative 'e2e_helper'`
3. Tag the test with `type: :e2e`
4. Use helper methods from `E2E::AictlHelper`

Example:

```ruby
# spec/e2e/my_feature_spec.rb
require_relative 'e2e_helper'

RSpec.describe 'My Feature E2E', type: :e2e do
  let(:test_prefix) { E2E::Config.test_prefix }
  let(:cluster_name) { "#{test_prefix}-cluster" }

  before(:all) do
    run_aictl("cluster create #{cluster_name}")
  end

  after(:all) do
    cleanup_test_resources(test_prefix) unless E2E::Config.skip_cleanup?
  end

  it 'tests my feature' do
    result = run_aictl('agent create "my test agent"')
    expect(result[:success]).to be(true)

    # Wait for synthesis
    synthesized = wait_for_condition(timeout: 300) do
      agent_synthesized?('my-test-agent')
    end
    expect(synthesized).to be(true)
  end
end
```

## Debugging

### View test resources

If tests fail, you can inspect the resources:

```bash
# List all clusters
aictl cluster list

# List all agents
aictl agent list --all-clusters

# Inspect specific agent
aictl agent inspect e2e-test-<timestamp>-agent

# View agent logs
aictl agent logs e2e-test-<timestamp>-agent

# View agent code
aictl agent code e2e-test-<timestamp>-agent
```

### Skip cleanup

To keep resources after test failure:

```bash
E2E_SKIP_CLEANUP=true bundle exec rake e2e
```

Then manually inspect and clean up:

```bash
# Delete specific agent
aictl agent delete e2e-test-<timestamp>-agent

# Delete specific cluster
aictl cluster delete e2e-test-<timestamp>-cluster
```

### Increase timeouts

If tests fail due to slow synthesis or deployment:

```bash
E2E_SYNTHESIS_TIMEOUT=600 E2E_POD_TIMEOUT=300 bundle exec rake e2e
```

## Troubleshooting

### "aictl: command not found"

Install the SDK gem:

```bash
cd sdk/ruby
gem build langop.gemspec
gem install langop-*.gem
```

### "Operator not found"

Install the Language Operator in your cluster:

```bash
cd /path/to/language-operator
make install
make deploy
```

### "Synthesis timeout"

- Check operator logs: `kubectl logs -n kube-system -l app=language-operator`
- Verify LLM API credentials are configured
- Increase timeout: `E2E_SYNTHESIS_TIMEOUT=600 bundle exec rake e2e`

### "Pod not ready"

- Check pod status: `kubectl get pods -n default`
- Check pod events: `kubectl describe pod <pod-name>`
- Check pod logs: `kubectl logs <pod-name>`
- Increase timeout: `E2E_POD_TIMEOUT=300 bundle exec rake e2e`

## CI/CD Integration

To run E2E tests in CI/CD:

```yaml
e2e-test:
  stage: test
  script:
    - make deploy  # Deploy operator
    - cd sdk/ruby
    - bundle install
    - E2E=true bundle exec rake e2e
  only:
    - main
    - merge_requests
```

## Best Practices

1. **Use unique prefixes** - Tests auto-generate unique prefixes to avoid collisions
2. **Clean up resources** - Always clean up in `after` hooks unless debugging
3. **Use timeouts** - All waits should have reasonable timeouts
4. **Test real workflows** - E2E tests should mirror actual user workflows
5. **Keep tests independent** - Each test should be runnable independently
6. **Document requirements** - Clearly document prerequisites in test descriptions

## Contributing

When adding new E2E tests:

1. Follow the existing test structure
2. Use the helper methods from `E2E::AictlHelper`
3. Add cleanup in `after` hooks
4. Document new environment variables
5. Update this README with new test coverage
