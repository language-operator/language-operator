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
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

func clusterRequest(name string) ctrl.Request {
	return ctrl.Request{NamespacedName: types.NamespacedName{Name: name}}
}

func TestLanguageClusterController_BasicReconciliation(t *testing.T) {
	scheme := testutil.SetupTestScheme(t)

	cluster := &langopv1alpha1.LanguageCluster{
		ObjectMeta: metav1.ObjectMeta{Name: "test-cluster"},
		Spec:       langopv1alpha1.LanguageClusterSpec{},
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
	req := clusterRequest(cluster.Name)

	// First reconcile adds finalizer and requeues
	result, err := reconciler.Reconcile(ctx, req)
	if err != nil {
		t.Fatalf("First reconcile failed: %v", err)
	}
	if !result.Requeue {
		t.Error("Expected requeue after adding finalizer")
	}

	// Second reconcile should set status
	result, err = reconciler.Reconcile(ctx, req)
	if err != nil {
		t.Fatalf("Second reconcile failed: %v", err)
	}
	if result.Requeue {
		t.Error("Expected no requeue after status update")
	}

	updatedCluster := &langopv1alpha1.LanguageCluster{}
	require.NoError(t, fakeClient.Get(ctx, types.NamespacedName{Name: cluster.Name}, updatedCluster))

	if updatedCluster.Status.Phase != "Ready" {
		t.Errorf("Expected phase 'Ready', got '%s'", updatedCluster.Status.Phase)
	}
	if !controllerutil.ContainsFinalizer(updatedCluster, FinalizerName) {
		t.Error("Expected finalizer to be added")
	}

	// Namespace should be created
	ns := &corev1.Namespace{}
	require.NoError(t, fakeClient.Get(ctx, types.NamespacedName{Name: cluster.Name}, ns))
}

func TestLanguageClusterController_ReadyCondition(t *testing.T) {
	scheme := testutil.SetupTestScheme(t)

	cluster := &langopv1alpha1.LanguageCluster{
		ObjectMeta: metav1.ObjectMeta{Name: "test-cluster-condition", Generation: 5},
		Spec:       langopv1alpha1.LanguageClusterSpec{},
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
	req := clusterRequest(cluster.Name)

	_, err := reconciler.Reconcile(ctx, req)
	require.NoError(t, err)
	_, err = reconciler.Reconcile(ctx, req)
	require.NoError(t, err)

	updatedCluster := &langopv1alpha1.LanguageCluster{}
	require.NoError(t, fakeClient.Get(ctx, types.NamespacedName{Name: cluster.Name}, updatedCluster))

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
	if readyCondition.Status != metav1.ConditionTrue {
		t.Errorf("Expected condition status True, got %s", readyCondition.Status)
	}
	if readyCondition.Reason != "ReconcileSuccess" {
		t.Errorf("Expected reason 'ReconcileSuccess', got '%s'", readyCondition.Reason)
	}
	if readyCondition.Message != "LanguageCluster is ready" {
		t.Errorf("Expected message 'LanguageCluster is ready', got '%s'", readyCondition.Message)
	}
	if readyCondition.ObservedGeneration != cluster.Generation {
		t.Errorf("Expected ObservedGeneration %d, got %d", cluster.Generation, readyCondition.ObservedGeneration)
	}
}

func TestLanguageClusterController_NotFoundHandling(t *testing.T) {
	scheme := testutil.SetupTestScheme(t)

	fakeClient := fake.NewClientBuilder().WithScheme(scheme).Build()

	reconciler := &LanguageClusterReconciler{
		Client: fakeClient,
		Scheme: scheme,
		Log:    logr.Discard(),
	}

	ctx := context.Background()
	result, err := reconciler.Reconcile(ctx, clusterRequest("non-existent-cluster"))

	if err != nil {
		t.Errorf("Expected no error for not found cluster, got: %v", err)
	}
	if result.Requeue {
		t.Error("Expected no requeue for not found cluster")
	}
}

func TestLanguageClusterController_MultipleReconciles(t *testing.T) {
	scheme := testutil.SetupTestScheme(t)

	cluster := &langopv1alpha1.LanguageCluster{
		ObjectMeta: metav1.ObjectMeta{Name: "test-cluster-multiple", Generation: 2},
		Spec:       langopv1alpha1.LanguageClusterSpec{},
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
	req := clusterRequest(cluster.Name)

	_, err := reconciler.Reconcile(ctx, req)
	require.NoError(t, err)
	_, err = reconciler.Reconcile(ctx, req)
	require.NoError(t, err)

	updatedCluster := &langopv1alpha1.LanguageCluster{}
	require.NoError(t, fakeClient.Get(ctx, types.NamespacedName{Name: cluster.Name}, updatedCluster))

	if updatedCluster.Status.Phase != "Ready" {
		t.Errorf("Expected phase 'Ready' after multiple reconciles, got '%s'", updatedCluster.Status.Phase)
	}
}

func TestLanguageClusterController_Finalizer(t *testing.T) {
	scheme := testutil.SetupTestScheme(t)

	cluster := &langopv1alpha1.LanguageCluster{
		ObjectMeta: metav1.ObjectMeta{Name: "test-cluster-finalizer"},
		Spec:       langopv1alpha1.LanguageClusterSpec{},
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
	req := clusterRequest(cluster.Name)

	result, err := reconciler.Reconcile(ctx, req)
	require.NoError(t, err)
	if !result.Requeue {
		t.Error("Expected requeue after adding finalizer")
	}

	updatedCluster := &langopv1alpha1.LanguageCluster{}
	require.NoError(t, fakeClient.Get(ctx, types.NamespacedName{Name: cluster.Name}, updatedCluster))
	if !controllerutil.ContainsFinalizer(updatedCluster, FinalizerName) {
		t.Error("Expected finalizer to be added after first reconcile")
	}

	result, err = reconciler.Reconcile(ctx, req)
	require.NoError(t, err)
	assert.False(t, result.Requeue)
}

func TestLanguageClusterController_DeletionWithoutDependents(t *testing.T) {
	scheme := testutil.SetupTestScheme(t)

	cluster := &langopv1alpha1.LanguageCluster{
		ObjectMeta: metav1.ObjectMeta{
			Name:              "test-cluster-deletion",
			Finalizers:        []string{FinalizerName},
			DeletionTimestamp: &metav1.Time{Time: time.Now()},
		},
		Spec: langopv1alpha1.LanguageClusterSpec{},
	}

	ns := &corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{Name: cluster.Name},
	}

	fakeClient := fake.NewClientBuilder().
		WithScheme(scheme).
		WithObjects(cluster, ns).
		WithStatusSubresource(cluster).
		Build()

	reconciler := &LanguageClusterReconciler{
		Client: fakeClient,
		Scheme: scheme,
		Log:    logr.Discard(),
	}

	ctx := context.Background()
	_, err := reconciler.Reconcile(ctx, clusterRequest(cluster.Name))
	require.NoError(t, err)

	updatedCluster := &langopv1alpha1.LanguageCluster{}
	err = fakeClient.Get(ctx, types.NamespacedName{Name: cluster.Name}, updatedCluster)
	if err == nil {
		for _, finalizer := range updatedCluster.Finalizers {
			if finalizer == FinalizerName {
				t.Error("Expected finalizer to be removed after successful cleanup")
			}
		}
	}

	// Namespace should be deleted
	deletedNs := &corev1.Namespace{}
	err = fakeClient.Get(ctx, types.NamespacedName{Name: cluster.Name}, deletedNs)
	require.True(t, errors.IsNotFound(err), "Expected namespace to be deleted")
}

func TestLanguageClusterController_DeletionWithDependents(t *testing.T) {
	scheme := testutil.SetupTestScheme(t)

	cluster := &langopv1alpha1.LanguageCluster{
		ObjectMeta: metav1.ObjectMeta{
			Name:              "test-cluster-with-deps",
			Finalizers:        []string{FinalizerName},
			DeletionTimestamp: &metav1.Time{Time: time.Now()},
		},
		Spec: langopv1alpha1.LanguageClusterSpec{},
	}

	agent := &langopv1alpha1.LanguageAgent{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-agent",
			Namespace: "test-cluster-with-deps",
		},
		Spec: langopv1alpha1.LanguageAgentSpec{
			ClusterRef:   "test-cluster-with-deps",
			Instructions: "test agent",
		},
	}

	tool := &langopv1alpha1.LanguageTool{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-tool",
			Namespace: "test-cluster-with-deps",
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
	_, err := reconciler.Reconcile(ctx, clusterRequest(cluster.Name))
	require.NoError(t, err)

	updatedAgent := &langopv1alpha1.LanguageAgent{}
	err = fakeClient.Get(ctx, types.NamespacedName{Name: agent.Name, Namespace: agent.Namespace}, updatedAgent)
	if err == nil {
		if updatedAgent.DeletionTimestamp.IsZero() {
			t.Error("Expected agent to be marked for deletion")
		}
	}

	updatedTool := &langopv1alpha1.LanguageTool{}
	err = fakeClient.Get(ctx, types.NamespacedName{Name: tool.Name, Namespace: tool.Namespace}, updatedTool)
	if err == nil {
		if updatedTool.DeletionTimestamp.IsZero() {
			t.Error("Expected tool to be marked for deletion")
		}
	}

	updatedCluster := &langopv1alpha1.LanguageCluster{}
	err = fakeClient.Get(ctx, types.NamespacedName{Name: cluster.Name}, updatedCluster)
	if err == nil {
		for _, finalizer := range updatedCluster.Finalizers {
			if finalizer == FinalizerName {
				t.Error("Expected finalizer to be removed after successful cleanup")
			}
		}
	}
}
