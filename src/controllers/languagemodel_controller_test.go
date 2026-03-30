package controllers

import (
	"context"
	"testing"
	"time"

	"github.com/go-logr/logr"
	langopv1alpha1 "github.com/language-operator/language-operator/api/v1alpha1"
	"github.com/language-operator/language-operator/controllers/testutil"
	"github.com/language-operator/language-operator/internal/testutil/gen"
	corev1 "k8s.io/api/core/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

func TestLanguageModelController_ConfigMapCreation(t *testing.T) {
	scheme := testutil.SetupTestScheme(t)

	model := gen.LanguageModel("test-model", "default",
		gen.SetModelName("claude-3-5-sonnet-20241022"),
	)

	fakeClient := fake.NewClientBuilder().
		WithScheme(scheme).
		WithObjects(model).
		WithStatusSubresource(model).
		Build()

	reconciler := &LanguageModelReconciler{
		Client:                  fakeClient,
		Scheme:                  scheme,
		Log:                     logr.Discard(),
		NetworkIsolationEnabled: true,
	}

	ctx := context.Background()
	req := ctrl.Request{
		NamespacedName: types.NamespacedName{
			Name:      model.Name,
			Namespace: model.Namespace,
		},
	}

	// First reconcile adds finalizer
	_, err := reconciler.Reconcile(ctx, req)
	if err != nil {
		t.Fatalf("First reconcile failed: %v", err)
	}

	// Second reconcile creates resources
	_, err = reconciler.Reconcile(ctx, req)
	if err != nil {
		t.Fatalf("Second reconcile failed: %v", err)
	}

	// Verify ConfigMap was created
	cm := &corev1.ConfigMap{}
	err = fakeClient.Get(ctx, types.NamespacedName{
		Name:      GenerateConfigMapName(model.Name, "model"),
		Namespace: model.Namespace,
	}, cm)
	if err != nil {
		t.Fatalf("Expected ConfigMap to exist, but got error: %v", err)
	}

	// Verify ConfigMap contains model configuration
	if cm.Data["provider"] != "anthropic" {
		t.Errorf("Expected provider 'anthropic', got '%s'", cm.Data["provider"])
	}
	if cm.Data["modelName"] != "claude-3-5-sonnet-20241022" {
		t.Errorf("Expected modelName 'claude-3-5-sonnet-20241022', got '%s'", cm.Data["modelName"])
	}
	if cm.Data["model.json"] == "" {
		t.Error("Expected model.json to contain serialized spec")
	}
}

// TestLanguageModelController_OnlyCreatesConfigMap verifies that the LanguageModel
// controller no longer creates a Deployment or Service — those are owned by the
// cluster's shared proxy. Only a ConfigMap should be created.
func TestLanguageModelController_OnlyCreatesConfigMap(t *testing.T) {
	scheme := testutil.SetupTestScheme(t)

	model := gen.LanguageModel("test-model-deployment", "default",
		gen.SetModelProvider("openai"),
		gen.SetModelName("gpt-4"),
	)

	fakeClient := fake.NewClientBuilder().
		WithScheme(scheme).
		WithObjects(model).
		WithStatusSubresource(model).
		Build()

	reconciler := &LanguageModelReconciler{
		Client: fakeClient,
		Scheme: scheme,
		Log:    logr.Discard(),
	}

	ctx := context.Background()
	req := ctrl.Request{NamespacedName: types.NamespacedName{Name: model.Name, Namespace: model.Namespace}}

	_, _ = reconciler.Reconcile(ctx, req)
	_, _ = reconciler.Reconcile(ctx, req)

	// ConfigMap must exist
	cm := &corev1.ConfigMap{}
	if err := fakeClient.Get(ctx, types.NamespacedName{Name: GenerateConfigMapName(model.Name, "model"), Namespace: model.Namespace}, cm); err != nil {
		t.Fatalf("Expected ConfigMap to exist: %v", err)
	}
	if cm.Data["model.json"] == "" {
		t.Error("Expected ConfigMap to contain model.json key")
	}

	// No Service should be created
	svc := &corev1.Service{}
	err := fakeClient.Get(ctx, types.NamespacedName{Name: model.Name, Namespace: model.Namespace}, svc)
	if err == nil {
		t.Error("LanguageModel controller should not create a Service; that is the cluster proxy's responsibility")
	}
}

func TestLanguageModelController_StatusUpdates(t *testing.T) {
	scheme := testutil.SetupTestScheme(t)

	model := gen.LanguageModel("test-model-status", "default",
		gen.SetModelName("claude-3-5-sonnet-20241022"),
	)
	model.Generation = 1

	fakeClient := fake.NewClientBuilder().
		WithScheme(scheme).
		WithObjects(model).
		WithStatusSubresource(model).
		Build()

	reconciler := &LanguageModelReconciler{
		Client:                  fakeClient,
		Scheme:                  scheme,
		Log:                     logr.Discard(),
		NetworkIsolationEnabled: true,
	}

	ctx := context.Background()
	req := ctrl.Request{
		NamespacedName: types.NamespacedName{
			Name:      model.Name,
			Namespace: model.Namespace,
		},
	}

	// First reconcile adds finalizer, second creates resources
	_, err := reconciler.Reconcile(ctx, req)
	if err != nil {
		t.Fatalf("First reconcile failed: %v", err)
	}
	_, err = reconciler.Reconcile(ctx, req)
	if err != nil {
		t.Fatalf("Second reconcile failed: %v", err)
	}

	// Fetch updated model to check status
	updatedModel := &langopv1alpha1.LanguageModel{}
	err = fakeClient.Get(ctx, types.NamespacedName{
		Name:      model.Name,
		Namespace: model.Namespace,
	}, updatedModel)
	if err != nil {
		t.Fatalf("Failed to fetch updated model: %v", err)
	}

	// Model is now managed by the cluster proxy, not its own deployment
	if updatedModel.Status.Phase != "Ready" {
		t.Errorf("Expected phase 'Ready', got '%s'", updatedModel.Status.Phase)
	}

	// Verify ObservedGeneration
	if updatedModel.Status.ObservedGeneration != model.Generation {
		t.Errorf("Expected ObservedGeneration %d, got %d", model.Generation, updatedModel.Status.ObservedGeneration)
	}

	// Verify Ready condition
	var readyCondition *metav1.Condition
	for i := range updatedModel.Status.Conditions {
		if updatedModel.Status.Conditions[i].Type == "Ready" {
			readyCondition = &updatedModel.Status.Conditions[i]
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
}

// TestLanguageModelController_APIKeySecretRefInConfigMap verifies that the secret
// reference is persisted in the ConfigMap so the cluster proxy can mount it.
func TestLanguageModelController_APIKeySecretRefInConfigMap(t *testing.T) {
	scheme := testutil.SetupTestScheme(t)

	model := gen.LanguageModel("test-model-secret", "default",
		gen.SetModelProvider("openai"),
		gen.SetModelName("gpt-4"),
		gen.SetModelAPIKeySecretRef("openai-api-key", "api-key"),
	)

	fakeClient := fake.NewClientBuilder().
		WithScheme(scheme).
		WithObjects(model).
		WithStatusSubresource(model).
		Build()

	reconciler := &LanguageModelReconciler{
		Client: fakeClient,
		Scheme: scheme,
		Log:    logr.Discard(),
	}

	ctx := context.Background()
	req := ctrl.Request{NamespacedName: types.NamespacedName{Name: model.Name, Namespace: model.Namespace}}

	_, _ = reconciler.Reconcile(ctx, req)
	_, _ = reconciler.Reconcile(ctx, req)

	cm := &corev1.ConfigMap{}
	if err := fakeClient.Get(ctx, types.NamespacedName{Name: GenerateConfigMapName(model.Name, "model"), Namespace: model.Namespace}, cm); err != nil {
		t.Fatalf("Expected ConfigMap to exist: %v", err)
	}
	if cm.Data["apiKeySecretRef.json"] == "" {
		t.Error("Expected ConfigMap to contain apiKeySecretRef.json so the cluster proxy can mount the secret")
	}
}

func TestLanguageModelController_NotFoundHandling(t *testing.T) {
	scheme := testutil.SetupTestScheme(t)

	fakeClient := fake.NewClientBuilder().
		WithScheme(scheme).
		Build()

	reconciler := &LanguageModelReconciler{
		Client:                  fakeClient,
		Scheme:                  scheme,
		Log:                     logr.Discard(),
		NetworkIsolationEnabled: true,
	}

	ctx := context.Background()
	result, err := reconciler.Reconcile(ctx, ctrl.Request{
		NamespacedName: types.NamespacedName{
			Name:      "non-existent-model",
			Namespace: "default",
		},
	})

	// Should not return error for not found
	if err != nil {
		t.Errorf("Expected no error for not found model, got: %v", err)
	}

	// Should not requeue
	if result.Requeue {
		t.Error("Expected no requeue for not found model")
	}
}

// NetworkPolicy tests removed: LanguageModel no longer owns NetworkPolicies.
// The cluster controller manages network isolation for the shared proxy.

func TestLanguageModelController_Finalizer(t *testing.T) {
	scheme := testutil.SetupTestScheme(t)

	model := gen.LanguageModel("test-model-finalizer", "default",
		gen.SetModelName("gpt-4"),
	)

	fakeClient := fake.NewClientBuilder().
		WithScheme(scheme).
		WithObjects(model).
		WithStatusSubresource(model).
		Build()

	reconciler := &LanguageModelReconciler{
		Client: fakeClient,
		Scheme: scheme,
		Log:    logr.Discard(),
	}

	ctx := context.Background()
	req := ctrl.Request{
		NamespacedName: types.NamespacedName{
			Name:      model.Name,
			Namespace: model.Namespace,
		},
	}

	// First reconcile should add finalizer
	result, err := reconciler.Reconcile(ctx, req)
	if err != nil {
		t.Fatalf("First reconcile failed: %v", err)
	}
	if !result.Requeue {
		t.Error("Expected requeue after adding finalizer")
	}

	updatedModel := &langopv1alpha1.LanguageModel{}
	if err := fakeClient.Get(ctx, types.NamespacedName{Name: model.Name, Namespace: model.Namespace}, updatedModel); err != nil {
		t.Fatalf("Failed to fetch updated model: %v", err)
	}
	if !controllerutil.ContainsFinalizer(updatedModel, FinalizerName) {
		t.Error("Expected finalizer to be added after first reconcile")
	}
}

func TestLanguageModelController_Deletion(t *testing.T) {
	scheme := testutil.SetupTestScheme(t)

	model := gen.LanguageModel("del-model", "default")
	model.Finalizers = []string{FinalizerName}
	model.DeletionTimestamp = &metav1.Time{Time: time.Now()}

	cmName := GenerateConfigMapName(model.Name, "model")
	cm := &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{Name: cmName, Namespace: model.Namespace},
	}

	fakeClient := fake.NewClientBuilder().
		WithScheme(scheme).
		WithObjects(model, cm).
		WithStatusSubresource(model).
		Build()

	reconciler := &LanguageModelReconciler{
		Client: fakeClient,
		Scheme: scheme,
		Log:    logr.Discard(),
	}

	ctx := context.Background()
	req := ctrl.Request{NamespacedName: types.NamespacedName{Name: model.Name, Namespace: model.Namespace}}
	_, err := reconciler.Reconcile(ctx, req)
	if err != nil {
		t.Fatalf("Reconcile returned unexpected error: %v", err)
	}

	// ConfigMap should be deleted
	deletedCM := &corev1.ConfigMap{}
	err = fakeClient.Get(ctx, types.NamespacedName{Name: cmName, Namespace: model.Namespace}, deletedCM)
	if !k8serrors.IsNotFound(err) {
		t.Errorf("Expected ConfigMap to be deleted, got: %v", err)
	}

	// Finalizer should be removed
	updated := &langopv1alpha1.LanguageModel{}
	if err := fakeClient.Get(ctx, types.NamespacedName{Name: model.Name, Namespace: model.Namespace}, updated); err == nil {
		if controllerutil.ContainsFinalizer(updated, FinalizerName) {
			t.Error("Expected finalizer to be removed after deletion")
		}
	}
}
