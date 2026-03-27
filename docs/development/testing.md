# Testing

Comprehensive testing guide for Language Operator.

## Test Categories

### Unit Tests

**Location:** `src/controllers/*_test.go`, `src/pkg/**/*_test.go`

**Run:**
```bash
cd src && make test
```

**Characteristics:**

- Use fake Kubernetes client (`controller-runtime/pkg/client/fake`)
- Fast, no external dependencies
- Test controller logic in isolation

**Example:**

```go
func TestLanguageAgentController(t *testing.T) {
    scheme := testutil.SetupTestScheme(t)
    agent := gen.LanguageAgent("test-agent", "default")

    fakeClient := fake.NewClientBuilder().
        WithScheme(scheme).
        WithObjects(agent).
        WithStatusSubresource(agent).
        Build()

    reconciler := &LanguageAgentReconciler{
        Client: fakeClient,
        Scheme: scheme,
        Log:    logr.Discard(),
    }

    // First reconcile adds finalizer
    _, err := reconciler.Reconcile(ctx, req)
    require.NoError(t, err)

    // Second reconcile creates resources
    _, err = reconciler.Reconcile(ctx, req)
    require.NoError(t, err)
}
```

### Integration Tests

**Location:** `src/controllers/*_integration_test.go`

**Run:**
```bash
cd src && make integration-test
```

**Characteristics:**

- Use `//go:build integration` build tag
- Run against real Kubernetes API server (envtest)
- Test full reconciliation loops with CRD validation
- Slower but more realistic

**Setup:**

Integration tests use controller-runtime's envtest which runs a real etcd and kube-apiserver:

```go
//go:build integration

var _ = Describe("LanguageAgent Controller", func() {
    It("Should create deployment", func() {
        agent := &v1alpha1.LanguageAgent{
            ObjectMeta: metav1.ObjectMeta{
                Name:      "test",
                Namespace: "default",
            },
            Spec: v1alpha1.LanguageAgentSpec{
                Image: "test:latest",
            },
        }

        Expect(k8sClient.Create(ctx, agent)).Should(Succeed())

        Eventually(func() error {
            deployment := &appsv1.Deployment{}
            return k8sClient.Get(ctx, types.NamespacedName{
                Name:      agent.Name,
                Namespace: agent.Namespace,
            }, deployment)
        }, timeout, interval).Should(Succeed())
    })
})
```

### End-to-End Tests

**Status:** Not yet implemented

**Planned:**

- Deploy operator to real cluster
- Create CRD instances
- Verify full functionality
- Test upgrades and migrations

## Testing Patterns

### Two-Reconcile Pattern

Controllers always require two reconcile calls in tests:

```go
// First reconcile: adds finalizer
result, err := reconciler.Reconcile(ctx, req)
require.NoError(t, err)
require.False(t, result.Requeue)

// Second reconcile: creates resources
result, err = reconciler.Reconcile(ctx, req)
require.NoError(t, err)
```

### Fluent Fixture Builders

Use builders from `internal/testutil/gen/`:

```go
agent := gen.LanguageAgent("my-agent", "default",
    gen.WithImage("test:latest"),
    gen.WithModelRef("claude-sonnet"),
    gen.WithInstructions("Do something useful"),
)
```

### Event Validation

Verify events are recorded:

```go
recorder := record.NewFakeRecorder(100)
reconciler := &LanguageAgentReconciler{
    Recorder: recorder,
}

// ... reconcile ...

select {
case event := <-recorder.Events:
    assert.Contains(t, event, "ResourceCreated")
default:
    t.Fatal("Expected event not recorded")
}
```

### Status Condition Checks

```go
var agent v1alpha1.LanguageAgent
err := fakeClient.Get(ctx, req.NamespacedName, &agent)
require.NoError(t, err)

condition := meta.FindStatusCondition(agent.Status.Conditions, "Ready")
require.NotNil(t, condition)
assert.Equal(t, metav1.ConditionTrue, condition.Status)
```

## CI Testing

### Test Workflow

**File:** `.github/workflows/test.yaml`

**Jobs:**

1. **lint** - gofmt and go vet
2. **unit-test** - All unit tests with coverage
3. **integration-test** - Integration tests with envtest
4. **validate-manifests** - CRD generation validation

**Runs on:**

- Every push to `main`
- Every pull request
- Manual workflow dispatch

### Coverage

Unit test coverage is reported to CI:

```bash
cd src
go test -race -coverprofile=coverage.out ./...
go tool cover -html=coverage.out
```

## Manual Testing

### Local Cluster Testing

1. **Create k3d cluster:**
   ```bash
   k3d cluster create langop-test
   ```

2. **Install operator from source:**
   ```bash
   cd chart
   helm install language-operator . \
     --namespace language-operator-system \
     --create-namespace
   ```

3. **Create test resources:**
   ```bash
   kubectl apply -f examples/openclaw.yaml
   ```

4. **Watch logs:**
   ```bash
   kubectl logs -n language-operator-system \
     -l app.kubernetes.io/name=language-operator \
     --follow
   ```

5. **Check resources:**
   ```bash
   kubectl get languageagents -A
   kubectl get pods -A
   kubectl describe languageagent openclaw
   ```

### Testing CRD Changes

After modifying CRD types:

1. **Regenerate:**
   ```bash
   cd src
   make generate
   make helm-crds
   ```

2. **Reinstall CRDs:**
   ```bash
   kubectl apply -f chart/crds/
   ```

3. **Test with sample resources**

## Test Data

### No Mock Data Rule

**Critical:** Features must work with real data before commit.

- **No** mock telemetry data
- **No** fake Kubernetes responses (except in unit tests)
- **No** stubbed external services in integration tests

Use real:

- ClickHouse for telemetry queries
- Kubernetes API for integration tests
- Actual model proxies for e2e tests

## Debugging Tests

### Verbose Output

```bash
cd src
go test -v ./controllers/...
```

### Run Single Test

```bash
go test ./controllers/... -run TestLanguageAgentController
```

### Integration Test Debugging

```bash
go test -tags integration -v ./controllers/... -run TestLanguageCluster
```

### Check Test Setup

```bash
# Verify envtest is installed
setup-envtest list

# Use specific Kubernetes version
KUBEBUILDER_ASSETS=$(setup-envtest use 1.29.0 -p path) \
  go test -tags integration ./controllers/...
```

## Best Practices

1. **Test happy path and error cases**
2. **Use table-driven tests** for multiple scenarios
3. **Keep tests focused** - one thing per test
4. **Clean up resources** in AfterEach/teardown
5. **Use meaningful assertions** with clear messages
6. **Avoid sleeps** - use Eventually/Consistently from Gomega
7. **Test status conditions** not just resource existence

## Next Steps

- [Development Setup](setup.md) - Set up your environment
- [Contributing Guide](contributing.md) - PR process
- [Controller Architecture](../architecture/controllers.md) - Understand the code
