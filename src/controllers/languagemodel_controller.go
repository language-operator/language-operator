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
	"encoding/json"

	"github.com/go-logr/logr"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	corev1 "k8s.io/api/core/v1"
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

// modelTracer is used by methods that haven't been refactored yet
var modelTracer = otel.Tracer("language-operator/model-controller")

//+kubebuilder:rbac:groups=langop.io,resources=languagemodels,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=langop.io,resources=languagemodels/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=langop.io,resources=languagemodels/finalizers,verbs=update
//+kubebuilder:rbac:groups="",resources=configmaps,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups="",resources=secrets,verbs=get;list;watch
//+kubebuilder:rbac:groups=apps,resources=deployments,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups="",resources=services,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=networking.k8s.io,resources=networkpolicies,verbs=get;list;watch;create;update;patch;delete

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

	// Reconcile the ConfigMap (read by the cluster's shared gateway)
	if err := r.reconcileConfigMap(ctx, model); err != nil {
		log.Error(err, "Failed to reconcile ConfigMap")
		span.RecordError(err)
		span.SetStatus(codes.Error, "Failed to reconcile ConfigMap")
		if r.EventManager != nil {
			r.EventManager.RecordConfigMapFailed(model, err)
		}
		SetCondition(&model.Status.Conditions, "Ready", metav1.ConditionFalse, "ReconcileError", err.Error(), model.Generation)
		model.Status.Phase = "Error"
		if statusErr := r.Status().Update(ctx, model); statusErr != nil {
			log.Error(statusErr, "Failed to update status")
		}
		reconcileErr = err
		return ctrl.Result{}, err
	}

	// Update status — model is managed by the cluster's shared gateway
	model.Status.ObservedGeneration = model.Generation
	model.Status.Phase = "Ready"
	model.Status.Message = "Model is managed by the cluster shared gateway"
	SetCondition(&model.Status.Conditions, "Ready", metav1.ConditionTrue, "ReconcileSuccess", "Model spec registered with cluster gateway", model.Generation)

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

// reconcileConfigMap creates or updates the ConfigMap for the model
func (r *LanguageModelReconciler) reconcileConfigMap(ctx context.Context, model *langopv1alpha1.LanguageModel) error {
	// Create ConfigMap data from model spec
	data := make(map[string]string)

	// Serialize the spec as JSON
	specJSON, err := json.Marshal(model.Spec)
	if err != nil {
		return err
	}
	data["model.json"] = string(specJSON)

	// Add individual fields for easy access
	data["provider"] = model.Spec.Provider
	data["modelName"] = model.Spec.ModelName
	if model.Spec.Endpoint != "" {
		data["endpoint"] = model.Spec.Endpoint
	}
	if model.Spec.Timeout != "" {
		data["timeout"] = model.Spec.Timeout
	}

	// Add API key secret reference info (not the actual secret)
	if model.Spec.APIKeySecretRef != nil {
		secretRefJSON, err := json.Marshal(model.Spec.APIKeySecretRef)
		if err != nil {
			return err
		}
		data["apiKeySecretRef.json"] = string(secretRefJSON)
	}

	// Add rate limits if specified
	if model.Spec.RateLimits != nil {
		rateLimitsJSON, err := json.Marshal(model.Spec.RateLimits)
		if err != nil {
			return err
		}
		data["rateLimits.json"] = string(rateLimitsJSON)
	}

	// Create or update the ConfigMap
	configMapName := GenerateConfigMapName(model.Name, "model")
	return CreateOrUpdateConfigMap(ctx, r.Client, r.Scheme, model, configMapName, model.Namespace, data)
}

// handleDeletion handles the deletion of the LanguageModel
func (r *LanguageModelReconciler) handleDeletion(ctx context.Context, model *langopv1alpha1.LanguageModel) (ctrl.Result, error) {
	log := log.FromContext(ctx)

	if controllerutil.ContainsFinalizer(model, FinalizerName) {
		// Delete the ConfigMap
		configMapName := GenerateConfigMapName(model.Name, "model")
		if err := DeleteConfigMap(ctx, r.Client, configMapName, model.Namespace); err != nil {
			log.Error(err, "Failed to delete ConfigMap")
			return ctrl.Result{}, err
		}

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
		Owns(&corev1.ConfigMap{}).
		Complete(r)
}
