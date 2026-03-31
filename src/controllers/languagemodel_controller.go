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
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
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

// LanguageModelReconciler reconciles a LanguageModel object
type LanguageModelReconciler struct {
	client.Client
	Scheme                  *runtime.Scheme
	Log                     logr.Logger
	Recorder                record.EventRecorder
	EventManager            *events.EventManager
	NetworkIsolationEnabled bool
}

//+kubebuilder:rbac:groups=langop.io,resources=languagemodels,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=langop.io,resources=languagemodels/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=langop.io,resources=languagemodels/finalizers,verbs=update
//+kubebuilder:rbac:groups="",resources=secrets,verbs=get;list;watch

// Reconcile reconciles a LanguageModel resource
func (r *LanguageModelReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	// Use the reconciler helper for common setup
	helper := &reconciler.ReconcileHelper[*langopv1alpha1.LanguageModel]{
		Client:       r.Client,
		TracerName:   "language-operator/model-controller",
		ResourceType: "model",
	}

	model := &langopv1alpha1.LanguageModel{}
	result, err := helper.StartReconcile(ctx, req, model)
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

	// Add model-specific attributes to span
	span.SetAttributes(
		attribute.String("model.provider", model.Spec.Provider),
	)

	// Handle deletion
	if !model.ObjectMeta.DeletionTimestamp.IsZero() {
		return r.handleDeletion(ctx, model)
	}

	// Add finalizer if it doesn't exist
	if !controllerutil.ContainsFinalizer(model, FinalizerName) {
		controllerutil.AddFinalizer(model, FinalizerName)
		if err := r.Update(ctx, model); err != nil {
			log.Error(err, "Failed to add finalizer")
			reconcileErr = err
			return ctrl.Result{}, err
		}
		if r.EventManager != nil {
			r.EventManager.RecordModelCreated(model)
		}
		return ctrl.Result{Requeue: true}, nil
	}

	// Update status — model is managed by the cluster's shared gateway
	model.Status.ObservedGeneration = model.Generation
	model.Status.Phase = events.PhaseStatusReady
	model.Status.Message = "Model is managed by the cluster shared gateway"
	SetCondition(&model.Status.Conditions, langopv1alpha1.ConditionReady, metav1.ConditionTrue, "ReconcileSuccess", "Model spec registered with cluster gateway", model.Generation)

	if r.EventManager != nil {
		r.EventManager.RecordModelReady(model, model.Spec.Provider)
	}

	if err := r.Status().Update(ctx, model); err != nil {
		log.Error(err, "Failed to update status")
		span.RecordError(err)
		span.SetStatus(codes.Error, "Failed to update status")
		reconcileErr = err
		return ctrl.Result{}, err
	}

	log.Info("Successfully reconciled LanguageModel")
	span.SetStatus(codes.Ok, "Reconciliation successful")
	return ctrl.Result{}, nil
}

// handleDeletion handles the deletion of the LanguageModel
func (r *LanguageModelReconciler) handleDeletion(ctx context.Context, model *langopv1alpha1.LanguageModel) (ctrl.Result, error) {
	log := log.FromContext(ctx)

	if controllerutil.ContainsFinalizer(model, FinalizerName) {
		// Remove finalizer
		controllerutil.RemoveFinalizer(model, FinalizerName)
		if err := r.Update(ctx, model); err != nil {
			log.Error(err, "Failed to remove finalizer")
			return ctrl.Result{}, err
		}
	}

	return ctrl.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager
func (r *LanguageModelReconciler) SetupWithManager(mgr ctrl.Manager, concurrency int) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&langopv1alpha1.LanguageModel{}).
		Complete(r)
}
