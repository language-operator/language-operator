package controllers

import (
	"context"
	"testing"
	"time"

	"github.com/go-logr/logr"
	rbacv1 "k8s.io/api/rbac/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	langopv1alpha1 "github.com/language-operator/language-operator/api/v1alpha1"
	"github.com/language-operator/language-operator/controllers/testutil"
	"github.com/language-operator/language-operator/internal/testutil/gen"
)

func newSelfConfigReconciler(t *testing.T, objs ...interface{}) (*LanguageAgentSelfConfigReconciler, *fake.ClientBuilder) {
	t.Helper()
	scheme := testutil.SetupTestScheme(t)
	builder := fake.NewClientBuilder().WithScheme(scheme)
	for _, o := range objs {
		if obj, ok := o.(interface {
			DeepCopyObject() interface{}
		}); ok {
			_ = obj
		}
	}
	return &LanguageAgentSelfConfigReconciler{
		Scheme: scheme,
		Log:    logr.Discard(),
	}, builder
}

func reconcileSelfConfig(t *testing.T, r *LanguageAgentSelfConfigReconciler, name, ns string) ctrl.Result {
	t.Helper()
	ctx := context.Background()
	req := ctrl.Request{NamespacedName: types.NamespacedName{Name: name, Namespace: ns}}
	result, err := r.Reconcile(ctx, req)
	if err != nil {
		t.Fatalf("unexpected reconcile error: %v", err)
	}
	return result
}

// agentWithSelfConfigure builds a ready LanguageAgent that allows the given actions.
func agentWithSelfConfigure(name, ns string, actions ...langopv1alpha1.SelfConfigAction) *langopv1alpha1.LanguageAgent {
	return gen.LanguageAgent(name, ns, func(a *langopv1alpha1.LanguageAgent) {
		a.Spec.SelfConfigure = &langopv1alpha1.SelfConfigureSpec{
			Enabled:        true,
			AllowedActions: actions,
		}
	})
}

func TestSelfConfigController_HappyPath_AddTool(t *testing.T) {
	scheme := testutil.SetupTestScheme(t)
	ns := "default"
	agent := agentWithSelfConfigure("my-agent", ns, langopv1alpha1.SelfConfigActionTools)
	sc := gen.LanguageAgentSelfConfig("req1", ns, "my-agent",
		gen.SetSelfConfigAddTools("web-search"),
	)

	fakeClient := fake.NewClientBuilder().
		WithScheme(scheme).
		WithObjects(agent, sc).
		WithStatusSubresource(sc).
		Build()

	r := &LanguageAgentSelfConfigReconciler{Client: fakeClient, Scheme: scheme, Log: logr.Discard()}
	ctx := context.Background()
	req := ctrl.Request{NamespacedName: types.NamespacedName{Name: "req1", Namespace: ns}}

	// First reconcile: adds finalizer.
	r.Reconcile(ctx, req)

	// Second reconcile: sets owner ref + re-fetches (requeue from owner ref update).
	r.Reconcile(ctx, req)

	// Third reconcile: applies the patch.
	r.Reconcile(ctx, req)

	updated := &langopv1alpha1.LanguageAgentSelfConfig{}
	if err := fakeClient.Get(ctx, req.NamespacedName, updated); err != nil {
		t.Fatalf("get selfconfig: %v", err)
	}
	if updated.Status.Phase != langopv1alpha1.SelfConfigPhaseApplied {
		t.Errorf("expected phase Applied, got %q (msg: %s)", updated.Status.Phase, updated.Status.Message)
	}
	if updated.Status.CompletionTime == nil {
		t.Error("expected CompletionTime to be set")
	}

	// Verify agent spec was patched.
	updatedAgent := &langopv1alpha1.LanguageAgent{}
	if err := fakeClient.Get(ctx, types.NamespacedName{Name: "my-agent", Namespace: ns}, updatedAgent); err != nil {
		t.Fatalf("get agent: %v", err)
	}
	found := false
	for _, tool := range updatedAgent.Spec.Tools {
		if tool.Name == "web-search" {
			found = true
		}
	}
	if !found {
		t.Errorf("expected tool 'web-search' to be added to agent spec, got %v", updatedAgent.Spec.Tools)
	}
}

func TestSelfConfigController_Denied_NotEnabled(t *testing.T) {
	scheme := testutil.SetupTestScheme(t)
	ns := "default"
	agent := gen.LanguageAgent("my-agent", ns) // no selfConfigure
	sc := gen.LanguageAgentSelfConfig("req1", ns, "my-agent",
		gen.SetSelfConfigAddTools("web-search"),
	)

	fakeClient := fake.NewClientBuilder().
		WithScheme(scheme).
		WithObjects(agent, sc).
		WithStatusSubresource(sc).
		Build()

	r := &LanguageAgentSelfConfigReconciler{Client: fakeClient, Scheme: scheme, Log: logr.Discard()}
	ctx := context.Background()
	req := ctrl.Request{NamespacedName: types.NamespacedName{Name: "req1", Namespace: ns}}

	r.Reconcile(ctx, req) // add finalizer
	r.Reconcile(ctx, req) // set owner ref
	r.Reconcile(ctx, req) // deny

	updated := &langopv1alpha1.LanguageAgentSelfConfig{}
	if err := fakeClient.Get(ctx, req.NamespacedName, updated); err != nil {
		t.Fatalf("get selfconfig: %v", err)
	}
	if updated.Status.Phase != langopv1alpha1.SelfConfigPhaseDenied {
		t.Errorf("expected phase Denied, got %q", updated.Status.Phase)
	}
}

func TestSelfConfigController_Denied_ActionNotAllowed(t *testing.T) {
	scheme := testutil.SetupTestScheme(t)
	ns := "default"
	// Agent allows models but not tools.
	agent := agentWithSelfConfigure("my-agent", ns, langopv1alpha1.SelfConfigActionModels)
	sc := gen.LanguageAgentSelfConfig("req1", ns, "my-agent",
		gen.SetSelfConfigAddTools("web-search"),
	)

	fakeClient := fake.NewClientBuilder().
		WithScheme(scheme).
		WithObjects(agent, sc).
		WithStatusSubresource(sc).
		Build()

	r := &LanguageAgentSelfConfigReconciler{Client: fakeClient, Scheme: scheme, Log: logr.Discard()}
	ctx := context.Background()
	req := ctrl.Request{NamespacedName: types.NamespacedName{Name: "req1", Namespace: ns}}

	r.Reconcile(ctx, req)
	r.Reconcile(ctx, req)
	r.Reconcile(ctx, req)

	updated := &langopv1alpha1.LanguageAgentSelfConfig{}
	if err := fakeClient.Get(ctx, req.NamespacedName, updated); err != nil {
		t.Fatalf("get selfconfig: %v", err)
	}
	if updated.Status.Phase != langopv1alpha1.SelfConfigPhaseDenied {
		t.Errorf("expected phase Denied, got %q", updated.Status.Phase)
	}
}

func TestSelfConfigController_Failed_AgentNotFound(t *testing.T) {
	scheme := testutil.SetupTestScheme(t)
	ns := "default"
	sc := gen.LanguageAgentSelfConfig("req1", ns, "nonexistent",
		gen.SetSelfConfigAddTools("web-search"),
	)

	fakeClient := fake.NewClientBuilder().
		WithScheme(scheme).
		WithObjects(sc).
		WithStatusSubresource(sc).
		Build()

	r := &LanguageAgentSelfConfigReconciler{Client: fakeClient, Scheme: scheme, Log: logr.Discard()}
	ctx := context.Background()
	req := ctrl.Request{NamespacedName: types.NamespacedName{Name: "req1", Namespace: ns}}

	r.Reconcile(ctx, req) // add finalizer
	r.Reconcile(ctx, req) // fail: agent not found

	updated := &langopv1alpha1.LanguageAgentSelfConfig{}
	if err := fakeClient.Get(ctx, req.NamespacedName, updated); err != nil {
		t.Fatalf("get selfconfig: %v", err)
	}
	if updated.Status.Phase != langopv1alpha1.SelfConfigPhaseFailed {
		t.Errorf("expected phase Failed, got %q", updated.Status.Phase)
	}
}

func TestSelfConfigController_TTL_DeletesAfterExpiry(t *testing.T) {
	scheme := testutil.SetupTestScheme(t)
	ns := "default"

	expiredTime := metav1.NewTime(time.Now().Add(-2 * time.Hour))
	sc := gen.LanguageAgentSelfConfig("req1", ns, "my-agent")
	sc.Status.Phase = langopv1alpha1.SelfConfigPhaseApplied
	sc.Status.CompletionTime = &expiredTime

	fakeClient := fake.NewClientBuilder().
		WithScheme(scheme).
		WithObjects(sc).
		WithStatusSubresource(sc).
		Build()

	// Add finalizer manually so we can test the TTL path directly.
	sc.Finalizers = []string{FinalizerName}
	fakeClient.Update(context.Background(), sc)

	r := &LanguageAgentSelfConfigReconciler{Client: fakeClient, Scheme: scheme, Log: logr.Discard()}
	ctx := context.Background()
	req := ctrl.Request{NamespacedName: types.NamespacedName{Name: "req1", Namespace: ns}}

	result, err := r.Reconcile(ctx, req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.RequeueAfter != 0 {
		t.Errorf("expected no requeue after deletion, got %v", result.RequeueAfter)
	}

	// Fake client sets DeletionTimestamp (not removes) when finalizer is present.
	deleted := &langopv1alpha1.LanguageAgentSelfConfig{}
	if err := fakeClient.Get(ctx, req.NamespacedName, deleted); err != nil {
		t.Fatalf("get: %v", err)
	}
	if deleted.DeletionTimestamp == nil {
		t.Error("expected object to be marked for deletion (DeletionTimestamp set)")
	}
}

func TestSelfConfigController_HappyPath_AddRoleRules(t *testing.T) {
	scheme := testutil.SetupTestScheme(t)
	ns := "default"
	agent := agentWithSelfConfigure("my-agent", ns, langopv1alpha1.SelfConfigActionRoleRules)
	sc := gen.LanguageAgentSelfConfig("req1", ns, "my-agent",
		gen.SetSelfConfigAddRoleRules(rbacv1.PolicyRule{
			APIGroups: []string{""},
			Resources: []string{"secrets"},
			Verbs:     []string{"get"},
		}),
	)

	fakeClient := fake.NewClientBuilder().
		WithScheme(scheme).
		WithObjects(agent, sc).
		WithStatusSubresource(sc).
		Build()

	r := &LanguageAgentSelfConfigReconciler{Client: fakeClient, Scheme: scheme, Log: logr.Discard()}
	ctx := context.Background()
	req := ctrl.Request{NamespacedName: types.NamespacedName{Name: "req1", Namespace: ns}}

	r.Reconcile(ctx, req)
	r.Reconcile(ctx, req)
	r.Reconcile(ctx, req)

	updated := &langopv1alpha1.LanguageAgentSelfConfig{}
	if err := fakeClient.Get(ctx, req.NamespacedName, updated); err != nil {
		t.Fatalf("get selfconfig: %v", err)
	}
	if updated.Status.Phase != langopv1alpha1.SelfConfigPhaseApplied {
		t.Errorf("expected Applied, got %q: %s", updated.Status.Phase, updated.Status.Message)
	}

	updatedAgent := &langopv1alpha1.LanguageAgent{}
	if err := fakeClient.Get(ctx, types.NamespacedName{Name: "my-agent", Namespace: ns}, updatedAgent); err != nil {
		t.Fatalf("get agent: %v", err)
	}
	found := false
	for _, rule := range updatedAgent.Spec.Deployment.RoleRules {
		if len(rule.Resources) > 0 && rule.Resources[0] == "secrets" {
			found = true
		}
	}
	if !found {
		t.Errorf("expected secrets rule in roleRules, got %v", updatedAgent.Spec.Deployment.RoleRules)
	}
}
