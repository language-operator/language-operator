package controllers

import (
	"context"
	"testing"
	"time"

	"github.com/go-logr/logr"
	langopv1alpha1 "github.com/language-operator/language-operator/api/v1alpha1"
	"github.com/language-operator/language-operator/controllers/testutil"
	"github.com/language-operator/language-operator/internal/testutil/gen"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

func TestLanguageAgentRuntimeController_FinalizerAdded(t *testing.T) {
	scheme := testutil.SetupTestScheme(t)

	rt := gen.LanguageAgentRuntime("test-runtime",
		gen.SetRuntimeImage("ghcr.io/example/agent:latest"),
	)

	fakeClient := fake.NewClientBuilder().
		WithScheme(scheme).
		WithObjects(rt).
		Build()

	reconciler := &LanguageAgentRuntimeReconciler{
		Client: fakeClient,
		Scheme: scheme,
		Log:    logr.Discard(),
	}

	ctx := context.Background()
	req := ctrl.Request{NamespacedName: types.NamespacedName{Name: rt.Name}}

	// First reconcile: should add finalizer and requeue
	result, err := reconciler.Reconcile(ctx, req)
	if err != nil {
		t.Fatalf("First reconcile failed: %v", err)
	}
	if !result.Requeue {
		t.Error("Expected Requeue=true after first reconcile (finalizer added)")
	}

	updated := &langopv1alpha1.LanguageAgentRuntime{}
	if err := fakeClient.Get(ctx, types.NamespacedName{Name: rt.Name}, updated); err != nil {
		t.Fatalf("Failed to fetch runtime after first reconcile: %v", err)
	}
	if !controllerutil.ContainsFinalizer(updated, FinalizerName) {
		t.Error("Expected finalizer to be added after first reconcile")
	}

	// Second reconcile: finalizer already present, should succeed without requeue
	result, err = reconciler.Reconcile(ctx, req)
	if err != nil {
		t.Fatalf("Second reconcile failed: %v", err)
	}
	if result.Requeue {
		t.Error("Expected Requeue=false after second reconcile (idempotent)")
	}
}

func TestLanguageAgentRuntimeController_Deletion(t *testing.T) {
	scheme := testutil.SetupTestScheme(t)

	rt := gen.LanguageAgentRuntime("del-runtime")
	rt.Finalizers = []string{FinalizerName}
	rt.DeletionTimestamp = &metav1.Time{Time: time.Now()}

	fakeClient := fake.NewClientBuilder().
		WithScheme(scheme).
		WithObjects(rt).
		Build()

	reconciler := &LanguageAgentRuntimeReconciler{
		Client: fakeClient,
		Scheme: scheme,
		Log:    logr.Discard(),
	}

	ctx := context.Background()
	req := ctrl.Request{NamespacedName: types.NamespacedName{Name: rt.Name}}

	_, err := reconciler.Reconcile(ctx, req)
	if err != nil {
		t.Fatalf("Reconcile during deletion returned unexpected error: %v", err)
	}

	// Finalizer should be removed
	updated := &langopv1alpha1.LanguageAgentRuntime{}
	if err := fakeClient.Get(ctx, types.NamespacedName{Name: rt.Name}, updated); err == nil {
		if controllerutil.ContainsFinalizer(updated, FinalizerName) {
			t.Error("Expected finalizer to be removed after deletion")
		}
	}
}

func TestLanguageAgentRuntimeController_NotFoundHandling(t *testing.T) {
	scheme := testutil.SetupTestScheme(t)

	fakeClient := fake.NewClientBuilder().
		WithScheme(scheme).
		Build()

	reconciler := &LanguageAgentRuntimeReconciler{
		Client: fakeClient,
		Scheme: scheme,
		Log:    logr.Discard(),
	}

	ctx := context.Background()
	result, err := reconciler.Reconcile(ctx, ctrl.Request{
		NamespacedName: types.NamespacedName{Name: "non-existent-runtime"},
	})

	if err != nil {
		t.Errorf("Expected no error for not-found runtime, got: %v", err)
	}
	if result.Requeue {
		t.Error("Expected no requeue for not-found runtime")
	}
}

// TestLanguageAgentRuntimeController_NoChildResources verifies that the runtime
// controller is intentionally minimal: it creates no Deployments, Services, or
// ConfigMaps. The runtime is a configuration preset read by the agent controller.
func TestLanguageAgentRuntimeController_NoChildResources(t *testing.T) {
	scheme := testutil.SetupTestScheme(t)

	rt := gen.LanguageAgentRuntime("preset-runtime",
		gen.SetRuntimeImage("ghcr.io/example/agent:v1"),
	)

	fakeClient := fake.NewClientBuilder().
		WithScheme(scheme).
		WithObjects(rt).
		Build()

	reconciler := &LanguageAgentRuntimeReconciler{
		Client: fakeClient,
		Scheme: scheme,
		Log:    logr.Discard(),
	}

	ctx := context.Background()
	req := ctrl.Request{NamespacedName: types.NamespacedName{Name: rt.Name}}

	_, _ = reconciler.Reconcile(ctx, req)
	_, _ = reconciler.Reconcile(ctx, req)

	deployList := &appsv1.DeploymentList{}
	if err := fakeClient.List(ctx, deployList, &client.ListOptions{}); err != nil {
		t.Fatalf("Failed to list Deployments: %v", err)
	}
	if len(deployList.Items) > 0 {
		t.Errorf("LanguageAgentRuntime controller should not create Deployments; got %d", len(deployList.Items))
	}

	svcList := &corev1.ServiceList{}
	if err := fakeClient.List(ctx, svcList, &client.ListOptions{}); err != nil {
		t.Fatalf("Failed to list Services: %v", err)
	}
	if len(svcList.Items) > 0 {
		t.Errorf("LanguageAgentRuntime controller should not create Services; got %d", len(svcList.Items))
	}

	cmList := &corev1.ConfigMapList{}
	if err := fakeClient.List(ctx, cmList, &client.ListOptions{}); err != nil {
		t.Fatalf("Failed to list ConfigMaps: %v", err)
	}
	if len(cmList.Items) > 0 {
		t.Errorf("LanguageAgentRuntime controller should not create ConfigMaps; got %d", len(cmList.Items))
	}
}
