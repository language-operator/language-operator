---
name: kubernetes-operator-patterns
description: Go controller patterns, CRDs, RBAC, and Kubernetes operator development best practices
---

# Kubernetes Operator Patterns

## Purpose

Provides consistent patterns and best practices for developing Kubernetes operators in Go using the controller-runtime framework. Covers CRD design, controller implementation, RBAC configuration, and integration testing patterns specific to the Language Operator project.

## When to Use This Skill

Automatically activates when:
- Working with Kubernetes controller code in `src/controllers/`
- Creating or modifying Custom Resource Definitions (CRDs)
- Implementing RBAC configurations
- Writing integration tests for Kubernetes resources
- Working with Kubernetes API objects and client-go

## Quick Start

### New Controller Checklist

- [ ] Define CRD schema in `src/api/v1alpha1/`
- [ ] Implement controller logic in `src/controllers/`
- [ ] Add RBAC permissions in `src/config/rbac/`
- [ ] Generate manifests with `controller-gen`
- [ ] Write integration tests with Ginkgo/Gomega
- [ ] Update Helm chart with new CRD

## Core Principles

### 1. Controller Pattern with controller-runtime

```go
// Controller structure following Language Operator patterns
type LanguageAgentReconciler struct {
    client.Client
    Scheme *runtime.Scheme
    Log    logr.Logger
}

//+kubebuilder:rbac:groups=language-operator.io,resources=languageagents,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=language-operator.io,resources=languageagents/status,verbs=get;update;patch

func (r *LanguageAgentReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
    log := r.Log.WithValues("languageagent", req.NamespacedName)

    // Fetch the LanguageAgent instance
    var agent languageoperatorv1alpha1.LanguageAgent
    if err := r.Get(ctx, req.NamespacedName, &agent); err != nil {
        if errors.IsNotFound(err) {
            return ctrl.Result{}, nil
        }
        return ctrl.Result{}, err
    }

    // Core reconciliation logic
    return r.reconcileAgent(ctx, &agent)
}
```

### 2. CRD Schema Design with Validation

```go
// LanguageAgentSpec follows operator patterns
type LanguageAgentSpec struct {
    // Model configuration with validation
    //+kubebuilder:validation:Required
    ModelName string `json:"modelName"`
    
    //+kubebuilder:validation:Enum=claude-3-sonnet;claude-3-haiku;claude-3-opus
    ModelVersion string `json:"modelVersion,omitempty"`
    
    // Execution configuration
    //+kubebuilder:validation:Enum=scheduled;realtime;manual
    //+kubebuilder:default=scheduled
    ExecutionMode string `json:"executionMode,omitempty"`
    
    // Schedule for scheduled execution
    //+kubebuilder:validation:Pattern=`^(@yearly|@monthly|@weekly|@daily|@hourly|([0-9]{1,2}\s){4}[0-9]{1,2})$`
    Schedule *string `json:"schedule,omitempty"`
    
    // Resource requirements
    Resources *corev1.ResourceRequirements `json:"resources,omitempty"`
}

// Status follows Kubernetes conventions
type LanguageAgentStatus struct {
    //+kubebuilder:validation:Enum=Pending;Running;Succeeded;Failed;Unknown
    Phase string `json:"phase,omitempty"`
    
    Conditions []metav1.Condition `json:"conditions,omitempty"`
    
    LastExecutionTime *metav1.Time `json:"lastExecutionTime,omitempty"`
    
    ExecutionHistory []ExecutionRecord `json:"executionHistory,omitempty"`
}
```

### 3. RBAC Configuration

```go
// Use kubebuilder RBAC markers for automatic generation
//+kubebuilder:rbac:groups=language-operator.io,resources=languageagents,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=language-operator.io,resources=languageagents/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=language-operator.io,resources=languageagents/finalizers,verbs=update
//+kubebuilder:rbac:groups=batch,resources=jobs,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=apps,resources=deployments,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups="",resources=pods,verbs=get;list;watch
//+kubebuilder:rbac:groups="",resources=configmaps,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups="",resources=secrets,verbs=get;list;watch
```

### 4. Error Handling and Status Updates

```go
func (r *LanguageAgentReconciler) reconcileAgent(ctx context.Context, agent *languageoperatorv1alpha1.LanguageAgent) (ctrl.Result, error) {
    // Update status at the end
    defer r.updateStatus(ctx, agent)
    
    // Validate spec
    if err := r.validateSpec(agent); err != nil {
        agent.Status.Phase = "Failed"
        r.setCondition(agent, "Validated", metav1.ConditionFalse, "ValidationFailed", err.Error())
        return ctrl.Result{}, nil
    }
    
    // Create or update resources
    if err := r.ensureJob(ctx, agent); err != nil {
        agent.Status.Phase = "Failed"
        r.setCondition(agent, "JobReady", metav1.ConditionFalse, "JobCreationFailed", err.Error())
        return ctrl.Result{RequeueAfter: time.Minute * 5}, err
    }
    
    agent.Status.Phase = "Running"
    r.setCondition(agent, "JobReady", metav1.ConditionTrue, "JobCreated", "Job created successfully")
    
    return ctrl.Result{RequeueAfter: time.Minute * 10}, nil
}
```

## Common Patterns

### Pattern 1: Resource Ownership and Cleanup

```go
// Set owner reference for proper garbage collection
func (r *LanguageAgentReconciler) ensureJob(ctx context.Context, agent *languageoperatorv1alpha1.LanguageAgent) error {
    job := &batchv1.Job{
        ObjectMeta: metav1.ObjectMeta{
            Name:      fmt.Sprintf("%s-execution", agent.Name),
            Namespace: agent.Namespace,
        },
        Spec: r.buildJobSpec(agent),
    }
    
    // Set owner reference for automatic cleanup
    if err := ctrl.SetControllerReference(agent, job, r.Scheme); err != nil {
        return err
    }
    
    return r.createOrUpdate(ctx, job)
}
```

### Pattern 2: Condition Management

```go
func (r *LanguageAgentReconciler) setCondition(agent *languageoperatorv1alpha1.LanguageAgent, 
    conditionType string, status metav1.ConditionStatus, reason, message string) {
    
    condition := metav1.Condition{
        Type:               conditionType,
        Status:             status,
        Reason:             reason,
        Message:            message,
        LastTransitionTime: metav1.NewTime(time.Now()),
    }
    
    meta.SetStatusCondition(&agent.Status.Conditions, condition)
}
```

### Pattern 3: Integration Testing with Ginkgo

```go
var _ = Describe("LanguageAgent Controller", func() {
    Context("When creating a LanguageAgent", func() {
        It("Should create the associated Job", func() {
            By("Creating a new LanguageAgent")
            agent := &languageoperatorv1alpha1.LanguageAgent{
                ObjectMeta: metav1.ObjectMeta{
                    Name:      "test-agent",
                    Namespace: "default",
                },
                Spec: languageoperatorv1alpha1.LanguageAgentSpec{
                    ModelName:     "claude-3-sonnet",
                    ExecutionMode: "scheduled",
                },
            }
            
            Expect(k8sClient.Create(ctx, agent)).Should(Succeed())
            
            By("Checking that the Job was created")
            Eventually(func() error {
                job := &batchv1.Job{}
                return k8sClient.Get(ctx, types.NamespacedName{
                    Name:      fmt.Sprintf("%s-execution", agent.Name),
                    Namespace: agent.Namespace,
                }, job)
            }, timeout, interval).Should(Succeed())
        })
    })
})
```

## Resource Files

For detailed information, see:
- [CRD Design Patterns](resources/crd-patterns.md) - Advanced CRD schema patterns and validation
- [Controller Testing](resources/testing-patterns.md) - Comprehensive testing strategies
- [RBAC Best Practices](resources/rbac-patterns.md) - Security and permission patterns

## Anti-Patterns to Avoid

❌ **Direct API calls without controller-runtime** - Use the controller-runtime client
❌ **Missing RBAC markers** - Always use kubebuilder RBAC annotations  
❌ **Blocking reconcile loops** - Use RequeueAfter for long-running operations
❌ **Missing owner references** - Always set controller references for cleanup
❌ **Ignoring status updates** - Always update resource status appropriately
❌ **Hardcoded namespaces** - Use the resource's namespace
❌ **Missing validation** - Use kubebuilder validation markers

## Quick Reference

| Need to... | Use this |
|-----------|----------|
| Create new CRD | Add to `src/api/v1alpha1/`, run `make manifests` |
| Add controller | Implement in `src/controllers/`, register in `main.go` |
| Update RBAC | Add kubebuilder markers, run `make manifests` |
| Test controllers | Use Ginkgo/Gomega in `src/controllers/*_test.go` |
| Generate manifests | Run `make manifests` (uses controller-gen) |
| Update status | Use `r.Status().Update(ctx, resource)` |
| Handle errors | Set conditions and appropriate phase in status |

## Code Generation Commands

```bash
# Generate manifests, CRDs, and RBAC
make manifests

# Generate deep copy methods
make generate

# Run tests
make test

# Run integration tests with real cluster
make integration-test
```

## Integration with Language Operator Architecture

This skill integrates with:
- **Telemetry system**: Controllers emit OpenTelemetry traces for monitoring
- **Dashboard API**: Controllers expose status via Kubernetes API for dashboard consumption  
- **Helm charts**: Generated manifests are packaged in `chart/` directory
- **CI/CD**: `make` targets integrate with GitHub Actions workflows