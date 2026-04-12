/*
Copyright 2025 Langop Team.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package controllers

import (
	"context"

	"github.com/go-logr/logr"
	"go.opentelemetry.io/otel/codes"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/tools/record"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/log"

	langopv1alpha1 "github.com/language-operator/language-operator/api/v1alpha1"
	"github.com/language-operator/language-operator/pkg/events"
	"github.com/language-operator/language-operator/pkg/reconciler"
)

// LanguagePersonaReconciler reconciles a LanguagePersona object
type LanguagePersonaReconciler struct {
	client.Client
	Scheme       *runtime.Scheme
	Log          logr.Logger
	Recorder     record.EventRecorder
	EventManager *events.EventManager
}

//+kubebuilder:rbac:groups=langop.io,resources=languagepersonas,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=langop.io,resources=languagepersonas/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=langop.io,resources=languagepersonas/finalizers,verbs=update

// Reconcile reconciles a LanguagePersona resource
func (r *LanguagePersonaReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	// Use the reconciler helper for common setup
	helper := &reconciler.ReconcileHelper[*langopv1alpha1.LanguagePersona]{
		Client:       r.Client,
		TracerName:   "language-operator/persona-controller",
		ResourceType: "persona",
	}

	persona := &langopv1alpha1.LanguagePersona{}
	result, err := helper.StartReconcile(ctx, req, persona)
	if err != nil {
		return ctrl.Result{}, err
	}
	if result == nil {
		// Resource was deleted
		return ctrl.Result{}, nil
	}

	// Capture the error for proper span completion
	var reconcileErr error
	defer func() {
		result.CompleteReconcile(reconcileErr)
	}()

	ctx = result.Ctx
	span := result.Span
	log := log.FromContext(ctx)

	// Write Failed phase on error exit paths via deferred status update.
	defer func() {
		if reconcileErr == nil || !persona.DeletionTimestamp.IsZero() {
			return
		}
		persona.Status.ObservedGeneration = persona.Generation
		if updateErr := r.Status().Update(ctx, persona); updateErr != nil && !apierrors.IsNotFound(updateErr) {
			log.Error(updateErr, "Failed to update LanguagePersona status")
		}
	}()

	// Handle deletion
	if !persona.ObjectMeta.DeletionTimestamp.IsZero() {
		return r.handleDeletion(ctx, persona)
	}

	// Add finalizer if it doesn't exist
	if !controllerutil.ContainsFinalizer(persona, FinalizerName) {
		SetPhase(&persona.Status.Phase, &persona.Status.ObservedGeneration, events.PhaseStatusPending, persona.Generation)
		SetCondition(&persona.Status.Conditions, langopv1alpha1.ConditionReady, metav1.ConditionFalse, langopv1alpha1.ReasonPending, "Persona initializing", persona.Generation)
		if err := r.Status().Update(ctx, persona); err != nil {
			log.Error(err, "Failed to write Pending status")
			reconcileErr = err
			return ctrl.Result{}, err
		}

		controllerutil.AddFinalizer(persona, FinalizerName)
		if err := r.Update(ctx, persona); err != nil {
			log.Error(err, "Failed to add finalizer")
			reconcileErr = err
			return ctrl.Result{}, err
		}
		r.EventManager.RecordPersonaCreated(persona, persona.Name)
		return ctrl.Result{Requeue: true}, nil
	}

	// Update status
	SetPhase(&persona.Status.Phase, &persona.Status.ObservedGeneration, events.PhaseStatusReady, persona.Generation)
	SetCondition(&persona.Status.Conditions, langopv1alpha1.ConditionReady, metav1.ConditionTrue, langopv1alpha1.ReasonReconcileSuccess, "Persona configuration is ready", persona.Generation)

	r.EventManager.RecordPersonaReady(persona, persona.Name)

	if err := r.Status().Update(ctx, persona); err != nil {
		log.Error(err, "Failed to update status")
		span.RecordError(err)
		span.SetStatus(codes.Error, "Failed to update status")
		SetPhase(&persona.Status.Phase, &persona.Status.ObservedGeneration, events.PhaseStatusFailed, persona.Generation)
		SetCondition(&persona.Status.Conditions, langopv1alpha1.ConditionReady, metav1.ConditionFalse, langopv1alpha1.ReasonServiceError, err.Error(), persona.Generation)
		reconcileErr = err
		return ctrl.Result{}, err
	}

	log.Info("Successfully reconciled LanguagePersona")
	span.SetStatus(codes.Ok, "Reconciliation successful")
	return ctrl.Result{}, nil
}

// handleDeletion handles the deletion of the LanguagePersona
func (r *LanguagePersonaReconciler) handleDeletion(ctx context.Context, persona *langopv1alpha1.LanguagePersona) (ctrl.Result, error) {
	return ctrl.Result{}, RemoveFinalizer(ctx, r.Client, persona)
}

// SetupWithManager sets up the controller with the Manager
func (r *LanguagePersonaReconciler) SetupWithManager(mgr ctrl.Manager, concurrency int) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&langopv1alpha1.LanguagePersona{}).
		Complete(r)
}
