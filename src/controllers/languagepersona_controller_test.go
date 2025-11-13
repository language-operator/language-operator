package controllers

import (
	"context"
	"testing"

	langopv1alpha1 "github.com/based/language-operator/api/v1alpha1"
	"github.com/based/language-operator/controllers/testutil"
	"github.com/go-logr/logr"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

func TestLanguagePersonaController_BasicReconciliation(t *testing.T) {
	scheme := testutil.SetupTestScheme(t)

	persona := &langopv1alpha1.LanguagePersona{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-persona",
			Namespace: "default",
		},
		Spec: langopv1alpha1.LanguagePersonaSpec{
			SystemPrompt: "You are a helpful assistant specializing in Ruby programming.",
		},
	}

	fakeClient := fake.NewClientBuilder().
		WithScheme(scheme).
		WithObjects(persona).
		WithStatusSubresource(persona).
		Build()

	reconciler := &LanguagePersonaReconciler{
		Client: fakeClient,
		Scheme: scheme,
		Log:    logr.Discard(),
	}

	ctx := context.Background()
	req := ctrl.Request{
		NamespacedName: types.NamespacedName{
			Name:      persona.Name,
			Namespace: persona.Namespace,
		},
	}

	// First reconcile adds finalizer
	_, err := reconciler.Reconcile(ctx, req)
	if err != nil {
		t.Fatalf("First reconcile failed: %v", err)
	}

	// Second reconcile creates ConfigMap
	_, err = reconciler.Reconcile(ctx, req)
	if err != nil {
		t.Fatalf("Second reconcile failed: %v", err)
	}

	// Verify ConfigMap was created
	cm := &corev1.ConfigMap{}
	err = fakeClient.Get(ctx, types.NamespacedName{
		Name:      GenerateConfigMapName(persona.Name, "persona"),
		Namespace: persona.Namespace,
	}, cm)
	if err != nil {
		t.Fatalf("Expected ConfigMap to exist, but got error: %v", err)
	}

	// Verify ConfigMap contains persona data
	if cm.Data["systemPrompt"] != persona.Spec.SystemPrompt {
		t.Errorf("Expected systemPrompt '%s', got '%s'", persona.Spec.SystemPrompt, cm.Data["systemPrompt"])
	}
}

func TestLanguagePersonaController_StatusUpdates(t *testing.T) {
	scheme := testutil.SetupTestScheme(t)

	persona := &langopv1alpha1.LanguagePersona{
		ObjectMeta: metav1.ObjectMeta{
			Name:       "test-persona-status",
			Namespace:  "default",
			Generation: 1,
		},
		Spec: langopv1alpha1.LanguagePersonaSpec{
			SystemPrompt: "You are a helpful assistant.",
		},
	}

	fakeClient := fake.NewClientBuilder().
		WithScheme(scheme).
		WithObjects(persona).
		WithStatusSubresource(persona).
		Build()

	reconciler := &LanguagePersonaReconciler{
		Client: fakeClient,
		Scheme: scheme,
		Log:    logr.Discard(),
	}

	ctx := context.Background()
	req := ctrl.Request{
		NamespacedName: types.NamespacedName{
			Name:      persona.Name,
			Namespace: persona.Namespace,
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

	// Fetch updated persona to check status
	updatedPersona := &langopv1alpha1.LanguagePersona{}
	err = fakeClient.Get(ctx, types.NamespacedName{
		Name:      persona.Name,
		Namespace: persona.Namespace,
	}, updatedPersona)
	if err != nil {
		t.Fatalf("Failed to fetch updated persona: %v", err)
	}

	// Verify status phase
	if updatedPersona.Status.Phase != "Ready" {
		t.Errorf("Expected phase 'Ready', got '%s'", updatedPersona.Status.Phase)
	}

	// Verify ObservedGeneration
	if updatedPersona.Status.ObservedGeneration != persona.Generation {
		t.Errorf("Expected ObservedGeneration %d, got %d", persona.Generation, updatedPersona.Status.ObservedGeneration)
	}

	// Verify Ready condition
	var readyCondition *metav1.Condition
	for i := range updatedPersona.Status.Conditions {
		if updatedPersona.Status.Conditions[i].Type == "Ready" {
			readyCondition = &updatedPersona.Status.Conditions[i]
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

func TestLanguagePersonaController_NotFoundHandling(t *testing.T) {
	scheme := testutil.SetupTestScheme(t)

	fakeClient := fake.NewClientBuilder().
		WithScheme(scheme).
		Build()

	reconciler := &LanguagePersonaReconciler{
		Client: fakeClient,
		Scheme: scheme,
		Log:    logr.Discard(),
	}

	ctx := context.Background()
	result, err := reconciler.Reconcile(ctx, ctrl.Request{
		NamespacedName: types.NamespacedName{
			Name:      "non-existent-persona",
			Namespace: "default",
		},
	})

	// Should not return error for not found
	if err != nil {
		t.Errorf("Expected no error for not found persona, got: %v", err)
	}

	// Should not requeue
	if result.Requeue {
		t.Error("Expected no requeue for not found persona")
	}
}
