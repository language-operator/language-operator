# Controller Architecture

Language Operator implements five controllers, one for each Custom Resource Definition (CRD).

## Controller Design Patterns

All controllers follow a consistent pattern using `controller-runtime`:

### Common Structure

```go
type LanguageAgentReconciler struct {
    client.Client
    Scheme   *runtime.Scheme
    Log      logr.Logger
    Recorder record.EventRecorder
    EventManager *events.EventManager
}
```

### Reconciliation Flow

Each controller follows this pattern:

1. **StartReconcile** - Uses `reconciler.ReconcileHelper` for:
    - Resource fetch
    - OpenTelemetry span setup
    - Deleted resource short-circuit
2. **Finalizer Management** - Added on first reconcile, cleanup in `handleDeletion`
3. **ConfigMap Reconciliation** - Via `CreateOrUpdateConfigMap` / `DeleteConfigMap` helpers
4. **Status Update** - Always updated last using `SetCondition` helper

### Shared Utilities

Located in `src/controllers/utils.go`:

- `GenerateConfigMapName(name, suffix)` - Consistent naming for generated ConfigMaps
- `CreateOrUpdateConfigMap` - Idempotent ConfigMap creation
- `DeleteConfigMap` - Clean ConfigMap removal
- `SetCondition` - Manages status conditions slice
- `FinalizerName` - Generates finalizer names

## Controller Responsibilities

### LanguageAgentController

**File:** `src/controllers/languageagent_controller.go`

**Creates:**

- Deployment for the agent pod
- Service for agent networking
- HTTPRoute for routing (if gateway API available)
- NetworkPolicy for isolation
- Two ConfigMaps:
    - Instructions ConfigMap (`instructions.txt`)
    - Config ConfigMap (`config.yaml` with personas, models, tools)

**Configuration Injection:**

The controller assembles configuration from:

- `spec.instructions` (inline) or `spec.instructionsFrom` (ConfigMap/Secret ref)
- Referenced `LanguagePersona` resources (via `personaRefs`)
- Referenced `LanguageModel` endpoints (via `modelRefs`)
- Referenced `LanguageTool` endpoints (via `toolRefs`)
- Agent metadata (name, namespace, UUID)

**Environment Variables Injected:**

- `AGENT_NAME`, `AGENT_NAMESPACE`, `AGENT_UUID`
- `AGENT_MODE` - from `spec.executionMode`
- `AGENT_CLUSTER_NAME`, `AGENT_CLUSTER_UUID`
- `MODEL_ENDPOINTS` - shared proxy URL (`http://proxy.<namespace>.svc.cluster.local:8000`)
- `LLM_MODEL` - comma-separated list of model names from `modelRefs`
- `TOOL_ENDPOINTS` - resolved MCP tool server URLs

**Volume Mounts:**

- `/etc/agent/instructions.txt` - from instructions ConfigMap
- `/etc/agent/config.yaml` - from config ConfigMap
- `/workspace` - optional persistent storage (if `spec.workspace` configured)

---

### LanguageClusterController

**File:** `src/controllers/languagecluster_controller.go`

**Creates:**

- Managed namespace for the cluster
- Shared LiteLLM proxy Deployment (`proxy`)
- Shared LiteLLM proxy Service (`proxy`)
- Shared LiteLLM proxy ConfigMap (`proxy-config`)
- Optional Ingress or HTTPRoute at `proxy.<spec.domain>`

**Watches:**

The LanguageCluster controller watches `LanguageModel` resources to trigger re-reconciliation when models are added or removed, ensuring the shared proxy configuration stays up to date.

**Proxy Configuration:**

The shared proxy is dynamically configured with all `LanguageModel` resources in the cluster's namespace. When models change, the proxy config is regenerated and the proxy pod is restarted.

**External Access:**

If `spec.domain` is set, the controller creates an Ingress or HTTPRoute (depending on available APIs) to expose the shared proxy externally at `proxy.<domain>`.

---

### LanguageModelController

**File:** `src/controllers/languagemodel_controller.go`

**Creates:**

- ConfigMap containing the model spec (key: `model__<name>.json`)

**No Longer Creates:**

Previously, each LanguageModel had its own Deployment and Service. Now all models share the cluster's LiteLLM proxy. The controller only creates a ConfigMap that the LanguageCluster controller reads when assembling the proxy configuration.

**Triggers:**

When a LanguageModel is created, updated, or deleted, it triggers a reconciliation of the parent LanguageCluster to update the shared proxy.

---

### LanguagePersonaController

**File:** `src/controllers/languagepersona_controller.go`

**Creates:**

- ConfigMap with the persona's JSON spec

**Purpose:**

Personas are behavioral templates that agents can reference. The controller validates the persona spec and stores it in a ConfigMap. The LanguageAgent controller reads these ConfigMaps and merges persona configurations into the agent's `/etc/agent/config.yaml`.

---

### LanguageToolController

**File:** `src/controllers/languagetool_controller.go`

**Creates:**

- Deployment for the MCP tool server
- Service for tool networking

**Validation:**

- Image registry validation via `pkg/validation.ValidateImageRegistry`
- Tool schema validation (if provided)

**Deployment Modes:**

- **Service** (default) - Standalone Deployment shared by multiple agents
- **Sidecar** - Tool container injected into each agent pod (future)

---

## Event Recording

All controllers use `pkg/events.EventManager` for Kubernetes event recording with standardized reasons:

- `ReasonResourceCreated` - Child resource created successfully
- `ReasonResourceUpdated` - Child resource updated
- `ReasonResourceDeleted` - Child resource cleaned up
- `ReasonReconciliationFailed` - Reconciliation error

Events are visible via:

```bash
kubectl get events --sort-by='.lastTimestamp'
```

## Observability

All controllers emit OpenTelemetry traces via `pkg/reconciler.ReconcileHelper`:

- **Spans** for each reconciliation loop
- **Attributes** for resource name, namespace, phase
- **Events** for key reconciliation steps

Traces are exported to the configured OTLP endpoint and queryable via the ClickHouse adapter.

## Testing

### Unit Tests

Use `sigs.k8s.io/controller-runtime/pkg/client/fake`:

```go
scheme := testutil.SetupTestScheme(t)
fakeClient := fake.NewClientBuilder().
    WithScheme(scheme).
    WithObjects(obj).
    WithStatusSubresource(obj).
    Build()

reconciler := &LanguageAgentReconciler{
    Client:  fakeClient,
    Scheme:  scheme,
    Log:     logr.Discard(),
    Recorder: record.NewFakeRecorder(100),
    EventManager: events.NewEventManager(record.NewFakeRecorder(100)),
}
```

**Important:** Two reconcile calls are always needed:
1. First call adds the finalizer
2. Second call creates resources

### Integration Tests

Use `//go:build integration` tag and run against envtest:

```bash
cd src && make integration-test
```

Integration tests run against a real Kubernetes API server (via controller-runtime's envtest) with all CRDs installed.

## RBAC

Each controller declares its required permissions via kubebuilder markers:

```go
//+kubebuilder:rbac:groups=langop.io,resources=languageagents,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=langop.io,resources=languageagents/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=langop.io,resources=languageagents/finalizers,verbs=update
```

Generated RBAC manifests are in `src/config/rbac/` and packaged in the Helm chart.

## Package Organization

```
src/
├── api/v1alpha1/           # CRD type definitions
├── controllers/            # Controller implementations
│   ├── languageagent_controller.go
│   ├── languagecluster_controller.go
│   ├── languagemodel_controller.go
│   ├── languagepersona_controller.go
│   ├── languagetool_controller.go
│   ├── utils.go           # Shared helpers
│   └── testutil/          # Test utilities
├── pkg/
│   ├── events/            # Event recording
│   ├── reconciler/        # ReconcileHelper
│   ├── telemetry/         # OpenTelemetry adapters
│   ├── validation/        # Validation helpers
│   └── network/           # NetworkPolicy helpers
└── internal/
    └── testutil/gen/      # Fluent test fixture builders
```

## Next Steps

- [CRD Reference](../api/overview.md) - Complete API specification
- [Agent Runtime Contract](agents.md) - What gets injected into agent pods
- [Tool Protocol](tools.md) - MCP tool server requirements
