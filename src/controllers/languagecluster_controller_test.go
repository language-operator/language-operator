package controllers

import (
	"context"
	"testing"
	"time"

	"github.com/go-logr/logr"
	langopv1alpha1 "github.com/language-operator/language-operator/api/v1alpha1"
	"github.com/language-operator/language-operator/controllers/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
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
	req := ctrl.Request{
		NamespacedName: types.NamespacedName{
			Name:      cluster.Name,
			Namespace: cluster.Namespace,
		},
	}

	// First reconcile adds finalizer and requeues
	result, err := reconciler.Reconcile(ctx, req)
	if err != nil {
		t.Fatalf("First reconcile failed: %v", err)
	}

	// Should requeue for finalizer
	if !result.Requeue {
		t.Error("Expected requeue after adding finalizer")
	}

	// Second reconcile should set status
	result, err = reconciler.Reconcile(ctx, req)
	if err != nil {
		t.Fatalf("Second reconcile failed: %v", err)
	}

	// Should not requeue after status update
	if result.Requeue {
		t.Error("Expected no requeue after status update")
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

	// Verify finalizer was added
	if !controllerutil.ContainsFinalizer(updatedCluster, FinalizerName) {
		t.Error("Expected finalizer to be added")
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
	req := ctrl.Request{
		NamespacedName: types.NamespacedName{
			Name:      cluster.Name,
			Namespace: cluster.Namespace,
		},
	}

	// First reconcile adds finalizer
	_, err := reconciler.Reconcile(ctx, req)
	if err != nil {
		t.Fatalf("First reconcile failed: %v", err)
	}

	// Second reconcile sets status
	_, err = reconciler.Reconcile(ctx, req)
	if err != nil {
		t.Fatalf("Second reconcile failed: %v", err)
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

func TestLanguageClusterController_Finalizer(t *testing.T) {
	scheme := testutil.SetupTestScheme(t)

	cluster := &langopv1alpha1.LanguageCluster{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-cluster-finalizer",
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
	req := ctrl.Request{
		NamespacedName: types.NamespacedName{
			Name:      cluster.Name,
			Namespace: cluster.Namespace,
		},
	}

	// First reconcile should add finalizer
	result, err := reconciler.Reconcile(ctx, req)
	require.NoError(t, err)

	// Should requeue for finalizer
	if !result.Requeue {
		t.Error("Expected requeue after adding finalizer")
	}

	// Fetch cluster to verify finalizer
	updatedCluster := &langopv1alpha1.LanguageCluster{}
	err = fakeClient.Get(ctx, types.NamespacedName{
		Name:      cluster.Name,
		Namespace: cluster.Namespace,
	}, updatedCluster)
	require.NoError(t, err)

	// Verify finalizer was added
	if !controllerutil.ContainsFinalizer(updatedCluster, FinalizerName) {
		t.Error("Expected finalizer to be added after first reconcile")
	}

	// Second reconcile should be normal (no requeue)
	result, err = reconciler.Reconcile(ctx, req)
	require.NoError(t, err)
	assert.False(t, result.Requeue)
}

func TestLanguageClusterController_DeletionWithoutDependents(t *testing.T) {
	scheme := testutil.SetupTestScheme(t)

	cluster := &langopv1alpha1.LanguageCluster{
		ObjectMeta: metav1.ObjectMeta{
			Name:              "test-cluster-deletion",
			Namespace:         "default",
			Finalizers:        []string{FinalizerName},
			DeletionTimestamp: &metav1.Time{Time: time.Now()},
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

	// Reconcile should handle deletion
	_, err := reconciler.Reconcile(ctx, req)
	require.NoError(t, err)

	// Verify the cluster was either deleted or finalizer was removed
	updatedCluster := &langopv1alpha1.LanguageCluster{}
	err = fakeClient.Get(ctx, types.NamespacedName{
		Name:      cluster.Name,
		Namespace: cluster.Namespace,
	}, updatedCluster)

	if err == nil {
		// Cluster still exists, check that finalizer was removed
		for _, finalizer := range updatedCluster.Finalizers {
			if finalizer == FinalizerName {
				t.Error("Expected finalizer to be removed after successful cleanup")
			}
		}
	}
	// If cluster was deleted, that's also acceptable
}

func TestLanguageClusterController_DeletionWithDependents(t *testing.T) {
	scheme := testutil.SetupTestScheme(t)

	cluster := &langopv1alpha1.LanguageCluster{
		ObjectMeta: metav1.ObjectMeta{
			Name:              "test-cluster-with-deps",
			Namespace:         "default",
			Finalizers:        []string{FinalizerName},
			DeletionTimestamp: &metav1.Time{Time: time.Now()},
		},
		Spec: langopv1alpha1.LanguageClusterSpec{},
	}

	// Create dependent agent
	agent := &langopv1alpha1.LanguageAgent{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-agent",
			Namespace: "default",
		},
		Spec: langopv1alpha1.LanguageAgentSpec{
			ClusterRef:   "test-cluster-with-deps",
			Instructions: "test agent",
		},
	}

	// Create dependent tool
	tool := &langopv1alpha1.LanguageTool{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-tool",
			Namespace: "default",
		},
		Spec: langopv1alpha1.LanguageToolSpec{
			ClusterRef: "test-cluster-with-deps",
			Type:       "shell",
		},
	}

	fakeClient := fake.NewClientBuilder().
		WithScheme(scheme).
		WithObjects(cluster, agent, tool).
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

	// Reconcile should handle deletion and cleanup dependents
	_, err := reconciler.Reconcile(ctx, req)
	require.NoError(t, err)

	// Verify dependent agent was deleted
	updatedAgent := &langopv1alpha1.LanguageAgent{}
	err = fakeClient.Get(ctx, types.NamespacedName{
		Name:      agent.Name,
		Namespace: agent.Namespace,
	}, updatedAgent)
	if err == nil {
		// Agent should have deletion timestamp set
		if updatedAgent.DeletionTimestamp.IsZero() {
			t.Error("Expected agent to be marked for deletion")
		}
	}

	// Verify dependent tool was deleted
	updatedTool := &langopv1alpha1.LanguageTool{}
	err = fakeClient.Get(ctx, types.NamespacedName{
		Name:      tool.Name,
		Namespace: tool.Namespace,
	}, updatedTool)
	if err == nil {
		// Tool should have deletion timestamp set
		if updatedTool.DeletionTimestamp.IsZero() {
			t.Error("Expected tool to be marked for deletion")
		}
	}

	// Verify cluster finalizer was removed
	updatedCluster := &langopv1alpha1.LanguageCluster{}
	err = fakeClient.Get(ctx, types.NamespacedName{
		Name:      cluster.Name,
		Namespace: cluster.Namespace,
	}, updatedCluster)

	if err == nil {
		// Cluster still exists, check that finalizer was removed
		for _, finalizer := range updatedCluster.Finalizers {
			if finalizer == FinalizerName {
				t.Error("Expected finalizer to be removed after successful cleanup")
			}
		}
	}
}
