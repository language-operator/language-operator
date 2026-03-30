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

func TestLanguagePersonaController_BasicReconciliation(t *testing.T) {
	scheme := testutil.SetupTestScheme(t)

	persona := gen.LanguagePersona("test-persona", "default",
		gen.SetPersonaPersonality("You are a helpful assistant specialising in Ruby programming."),
		gen.SetPersonaTone("technical"),
	)

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

	// Verify ConfigMap contains the three persona fields
	if cm.Data["personality"] != persona.Spec.Personality {
		t.Errorf("Expected personality %q, got %q", persona.Spec.Personality, cm.Data["personality"])
	}
	if cm.Data["tone"] != persona.Spec.Tone {
		t.Errorf("Expected tone %q, got %q", persona.Spec.Tone, cm.Data["tone"])
	}
	if cm.Data["persona.json"] == "" {
		t.Error("Expected persona.json to contain serialised spec")
	}
}

func TestLanguagePersonaController_StatusUpdates(t *testing.T) {
	scheme := testutil.SetupTestScheme(t)

	persona := gen.LanguagePersona("test-persona-status", "default",
		gen.SetPersonaPersonality("You are a helpful assistant."),
	)
	persona.Generation = 1

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

func TestLanguagePersonaController_AllThreeFields(t *testing.T) {
	scheme := testutil.SetupTestScheme(t)

	persona := gen.LanguagePersona("full-persona", "default",
		gen.SetPersonaTone("concise and direct"),
		gen.SetPersonaPersonality("Curious and methodical, always explains reasoning step by step"),
		gen.SetPersonaExpertise("Senior software engineer specialising in distributed systems and Go"),
	)

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
	req := ctrl.Request{NamespacedName: types.NamespacedName{Name: persona.Name, Namespace: persona.Namespace}}

	_, _ = reconciler.Reconcile(ctx, req)
	_, _ = reconciler.Reconcile(ctx, req)

	cm := &corev1.ConfigMap{}
	if err := fakeClient.Get(ctx, types.NamespacedName{Name: GenerateConfigMapName(persona.Name, "persona"), Namespace: persona.Namespace}, cm); err != nil {
		t.Fatalf("Expected ConfigMap to exist: %v", err)
	}

	if cm.Data["tone"] != persona.Spec.Tone {
		t.Errorf("tone: want %q, got %q", persona.Spec.Tone, cm.Data["tone"])
	}
	if cm.Data["personality"] != persona.Spec.Personality {
		t.Errorf("personality: want %q, got %q", persona.Spec.Personality, cm.Data["personality"])
	}
	if cm.Data["expertise"] != persona.Spec.Expertise {
		t.Errorf("expertise: want %q, got %q", persona.Spec.Expertise, cm.Data["expertise"])
	}
	if cm.Data["persona.json"] == "" {
		t.Error("Expected persona.json key in ConfigMap")
	}
}

func TestLanguagePersonaController_Deletion(t *testing.T) {
	scheme := testutil.SetupTestScheme(t)

	persona := gen.LanguagePersona("del-persona", "default")
	persona.Finalizers = []string{FinalizerName}
	persona.DeletionTimestamp = &metav1.Time{Time: time.Now()}

	cmName := GenerateConfigMapName(persona.Name, "persona")
	cm := &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{Name: cmName, Namespace: persona.Namespace},
	}

	fakeClient := fake.NewClientBuilder().
		WithScheme(scheme).
		WithObjects(persona, cm).
		WithStatusSubresource(persona).
		Build()

	reconciler := &LanguagePersonaReconciler{
		Client: fakeClient,
		Scheme: scheme,
		Log:    logr.Discard(),
	}

	ctx := context.Background()
	req := ctrl.Request{NamespacedName: types.NamespacedName{Name: persona.Name, Namespace: persona.Namespace}}
	_, err := reconciler.Reconcile(ctx, req)
	if err != nil {
		t.Fatalf("Reconcile returned unexpected error: %v", err)
	}

	// ConfigMap should be deleted
	deletedCM := &corev1.ConfigMap{}
	err = fakeClient.Get(ctx, types.NamespacedName{Name: cmName, Namespace: persona.Namespace}, deletedCM)
	if !k8serrors.IsNotFound(err) {
		t.Errorf("Expected ConfigMap to be deleted, got: %v", err)
	}

	// Finalizer should be removed
	updated := &langopv1alpha1.LanguagePersona{}
	if err := fakeClient.Get(ctx, types.NamespacedName{Name: persona.Name, Namespace: persona.Namespace}, updated); err == nil {
		if controllerutil.ContainsFinalizer(updated, FinalizerName) {
			t.Error("Expected finalizer to be removed after deletion")
		}
	}
}
