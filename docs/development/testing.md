# Testing

## Test Types

| Type | Location | Run |
|------|----------|-----|
| Unit | `src/**/*_test.go` | `cd src && make test` |
| Integration | `src/controllers/` and `src/api/v1alpha1/`, behind the `//go:build integration` tag (harness in `suite_test.go`) | `cd src && make integration-test` |

Unit tests use the controller-runtime fake client and have no external dependencies.
Integration tests run against a real kube-apiserver + etcd via [envtest](https://book.kubebuilder.io/reference/envtest.html).

## Unit Test Pattern

Controllers reconcile twice in tests: the first call adds the finalizer, the second creates resources.

```go
scheme := testutil.SetupTestScheme(t)
agent := gen.LanguageAgent("test-agent", "default", gen.SetAgentImage("test:latest"))

fakeClient := fake.NewClientBuilder().
    WithScheme(scheme).
    WithObjects(agent).
    WithStatusSubresource(agent).
    Build()

recorder := record.NewFakeRecorder(100)
reconciler := &LanguageAgentReconciler{
    Client:                  fakeClient,
    Scheme:                  scheme,
    Log:                     logr.Discard(),
    Recorder:                recorder,
    EventManager:            events.NewEventManager(recorder),
    RegistryManager:         &mockRegistryManager{}, // defined in languageagent_controller_test.go
    NetworkIsolationEnabled: false,
}

_, err := reconciler.Reconcile(ctx, req) // adds finalizer
require.NoError(t, err)
_, err = reconciler.Reconcile(ctx, req)  // creates resources
require.NoError(t, err)
```

## Fixture Builders

Build CRs with the fluent helpers in `src/internal/testutil/gen/`. Modifiers are named `SetAgent*`, `SetModel*`, etc.:

```go
agent := gen.LanguageAgent("my-agent", "default",
    gen.SetAgentImage("test:latest"),
    gen.SetAgentModel("claude-sonnet"),
    gen.SetAgentInstructions("Do something useful"),
)
```

## Common Assertions

```go
// Event recorded
select {
case event := <-recorder.Events:
    assert.Contains(t, event, "ResourceCreated")
default:
    t.Fatal("expected event not recorded")
}

// Status condition
var got v1alpha1.LanguageAgent
require.NoError(t, fakeClient.Get(ctx, req.NamespacedName, &got))
cond := meta.FindStatusCondition(got.Status.Conditions, "Ready")
require.NotNil(t, cond)
assert.Equal(t, metav1.ConditionTrue, cond.Status)
```

## Debugging

```bash
cd src
go test -v ./controllers/...                                   # verbose
go test ./controllers/... -run TestLanguageAgentController     # single unit test
go test -tags integration -v ./controllers/... -run TestLanguageCluster  # single integration test

setup-envtest list                                            # available envtest binaries
KUBEBUILDER_ASSETS=$(setup-envtest use 1.29.0 -p path) \
  go test -tags integration ./controllers/...                 # pin a Kubernetes version
```

## CI

`.github/workflows/test.yaml` runs on every push to `main` and every PR: `lint` (gofmt + go vet),
`unit-test`, `integration-test`, `python-test`, and `validate-manifests` (fails if `make generate` /
`make helm-crds` output is not committed — see [Development Setup](setup.md)).

After changing any type in `src/api/v1alpha1/`, regenerate and stage the output:

```bash
cd src && make generate && make helm-crds
```

Features must work against real Kubernetes APIs before commit — no mock data outside unit tests.
