package controllers

import (
	"context"
	"testing"

	"github.com/go-logr/logr"
	langopv1alpha1 "github.com/language-operator/language-operator/api/v1alpha1"
	"github.com/language-operator/language-operator/controllers/testutil"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

func TestLanguageClusterController_BasicReconciliation(t *testing.T) {
	scheme := testutil.SetupTestScheme(t)

	cluster := &langopv1alpha1.LanguageCluster{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-cluster",
			Namespace: "default",
		},
		Spec: langopv1alpha1.LanguageClusterSpec{},
	}

	fakeClient := fake.NewClientBuilder().
		WithScheme(scheme).
		WithObjects(cluster).
		WithStatusSubresource(cluster).
		Build()

	reconciler := &LanguageClusterReconciler{
		Client: fakeClient,
		Scheme: scheme,
		Log:    logr.Discard(),
	}

	ctx := context.Background()
	result, err := reconciler.Reconcile(ctx, ctrl.Request{
		NamespacedName: types.NamespacedName{
			Name:      cluster.Name,
			Namespace: cluster.Namespace,
		},
	})
	if err != nil {
		t.Fatalf("Reconcile failed: %v", err)
	}

	// Verify no requeue
	if result.Requeue {
		t.Error("Expected no requeue, but got requeue=true")
	}

	// Fetch updated cluster to check status
	updatedCluster := &langopv1alpha1.LanguageCluster{}
	err = fakeClient.Get(ctx, types.NamespacedName{
		Name:      cluster.Name,
		Namespace: cluster.Namespace,
	}, updatedCluster)
	if err != nil {
		t.Fatalf("Failed to fetch updated cluster: %v", err)
	}

	// Verify status phase is Ready
	if updatedCluster.Status.Phase != "Ready" {
		t.Errorf("Expected phase 'Ready', got '%s'", updatedCluster.Status.Phase)
	}
}

func TestLanguageClusterController_ReadyCondition(t *testing.T) {
	scheme := testutil.SetupTestScheme(t)

	cluster := &langopv1alpha1.LanguageCluster{
		ObjectMeta: metav1.ObjectMeta{
			Name:       "test-cluster-condition",
			Namespace:  "default",
			Generation: 5,
		},
		Spec: langopv1alpha1.LanguageClusterSpec{},
	}

	fakeClient := fake.NewClientBuilder().
		WithScheme(scheme).
		WithObjects(cluster).
		WithStatusSubresource(cluster).
		Build()

	reconciler := &LanguageClusterReconciler{
		Client: fakeClient,
		Scheme: scheme,
		Log:    logr.Discard(),
	}

	ctx := context.Background()
	_, err := reconciler.Reconcile(ctx, ctrl.Request{
		NamespacedName: types.NamespacedName{
			Name:      cluster.Name,
			Namespace: cluster.Namespace,
		},
	})
	if err != nil {
		t.Fatalf("Reconcile failed: %v", err)
	}

	// Fetch updated cluster to check conditions
	updatedCluster := &langopv1alpha1.LanguageCluster{}
	err = fakeClient.Get(ctx, types.NamespacedName{
		Name:      cluster.Name,
		Namespace: cluster.Namespace,
	}, updatedCluster)
	if err != nil {
		t.Fatalf("Failed to fetch updated cluster: %v", err)
	}

	// Verify Ready condition exists
	var readyCondition *metav1.Condition
	for i := range updatedCluster.Status.Conditions {
		if updatedCluster.Status.Conditions[i].Type == "Ready" {
			readyCondition = &updatedCluster.Status.Conditions[i]
			break
		}
	}
	if readyCondition == nil {
		t.Fatal("Ready condition not found")
	}

	// Verify condition is True
	if readyCondition.Status != metav1.ConditionTrue {
		t.Errorf("Expected condition status True, got %s", readyCondition.Status)
	}

	// Verify reason
	if readyCondition.Reason != "ReconcileSuccess" {
		t.Errorf("Expected reason 'ReconcileSuccess', got '%s'", readyCondition.Reason)
	}

	// Verify message
	if readyCondition.Message != "LanguageCluster is ready" {
		t.Errorf("Expected message 'LanguageCluster is ready', got '%s'", readyCondition.Message)
	}

	// Verify ObservedGeneration matches
	if readyCondition.ObservedGeneration != cluster.Generation {
		t.Errorf("Expected ObservedGeneration %d, got %d", cluster.Generation, readyCondition.ObservedGeneration)
	}
}

func TestLanguageClusterController_NotFoundHandling(t *testing.T) {
	scheme := testutil.SetupTestScheme(t)

	fakeClient := fake.NewClientBuilder().
		WithScheme(scheme).
		Build()

	reconciler := &LanguageClusterReconciler{
		Client: fakeClient,
		Scheme: scheme,
		Log:    logr.Discard(),
	}

	ctx := context.Background()
	result, err := reconciler.Reconcile(ctx, ctrl.Request{
		NamespacedName: types.NamespacedName{
			Name:      "non-existent-cluster",
			Namespace: "default",
		},
	})

	// Should not return error for not found
	if err != nil {
		t.Errorf("Expected no error for not found cluster, got: %v", err)
	}

	// Should not requeue
	if result.Requeue {
		t.Error("Expected no requeue for not found cluster")
	}
}

func TestLanguageClusterController_MultipleReconciles(t *testing.T) {
	scheme := testutil.SetupTestScheme(t)

	cluster := &langopv1alpha1.LanguageCluster{
		ObjectMeta: metav1.ObjectMeta{
			Name:       "test-cluster-multiple",
			Namespace:  "default",
			Generation: 2,
		},
		Spec: langopv1alpha1.LanguageClusterSpec{},
	}

	fakeClient := fake.NewClientBuilder().
		WithScheme(scheme).
		WithObjects(cluster).
		WithStatusSubresource(cluster).
		Build()

	reconciler := &LanguageClusterReconciler{
		Client: fakeClient,
		Scheme: scheme,
		Log:    logr.Discard(),
	}

	ctx := context.Background()
	req := ctrl.Request{
		NamespacedName: types.NamespacedName{
			Name:      cluster.Name,
			Namespace: cluster.Namespace,
		},
	}

	// First reconcile
	_, err := reconciler.Reconcile(ctx, req)
	if err != nil {
		t.Fatalf("First reconcile failed: %v", err)
	}

	// Second reconcile should also succeed (idempotent)
	_, err = reconciler.Reconcile(ctx, req)
	if err != nil {
		t.Fatalf("Second reconcile failed: %v", err)
	}

	// Fetch updated cluster
	updatedCluster := &langopv1alpha1.LanguageCluster{}
	err = fakeClient.Get(ctx, types.NamespacedName{
		Name:      cluster.Name,
		Namespace: cluster.Namespace,
	}, updatedCluster)
	if err != nil {
		t.Fatalf("Failed to fetch updated cluster: %v", err)
	}

	// Still should be Ready
	if updatedCluster.Status.Phase != "Ready" {
		t.Errorf("Expected phase 'Ready' after multiple reconciles, got '%s'", updatedCluster.Status.Phase)
	}
}
