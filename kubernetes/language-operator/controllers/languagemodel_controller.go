/*
Copyright 2025 Based Team.

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
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	langopv1alpha1 "github.com/based/language-operator/api/v1alpha1"
)

// LanguageModelReconciler reconciles a LanguageModel object
type LanguageModelReconciler struct {
	client.Client
	Scheme *runtime.Scheme
	Log    logr.Logger
}

//+kubebuilder:rbac:groups=langop.io,resources=languagemodels,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=langop.io,resources=languagemodels/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=langop.io,resources=languagemodels/finalizers,verbs=update
//+kubebuilder:rbac:groups="",resources=configmaps,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups="",resources=secrets,verbs=get;list;watch

// Reconcile reconciles a LanguageModel resource
func (r *LanguageModelReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log := r.Log.WithValues("languagemodel", req.NamespacedName)

	// Fetch the LanguageModel instance
	model := &langopv1alpha1.LanguageModel{}
	if err := r.Get(ctx, req.NamespacedName, model); err != nil {
		if apierrors.IsNotFound(err) {
			log.Info("LanguageModel resource not found. Ignoring since object must be deleted")
			return ctrl.Result{}, nil
		}
		log.Error(err, "Failed to get LanguageModel")
		return ctrl.Result{}, err
	}

	// Handle deletion
	if !model.ObjectMeta.DeletionTimestamp.IsZero() {
		return r.handleDeletion(ctx, model)
	}

	// Add finalizer if it doesn't exist
	if !HasFinalizer(model) {
		AddFinalizer(model)
		if err := r.Update(ctx, model); err != nil {
			log.Error(err, "Failed to add finalizer")
			return ctrl.Result{}, err
		}
		return ctrl.Result{Requeue: true}, nil
	}

	// Reconcile the ConfigMap
	if err := r.reconcileConfigMap(ctx, model); err != nil {
		log.Error(err, "Failed to reconcile ConfigMap")
		SetCondition(&model.Status.Conditions, "Ready", metav1.ConditionFalse, "ReconcileError", err.Error(), model.Generation)
		model.Status.Phase = "Failed"
		if statusErr := r.Status().Update(ctx, model); statusErr != nil {
			log.Error(statusErr, "Failed to update status")
		}
		return ctrl.Result{}, err
	}

	// Update status
	model.Status.ObservedGeneration = model.Generation
	model.Status.Phase = "Ready"
	// Status fields updated
	SetCondition(&model.Status.Conditions, "Ready", metav1.ConditionTrue, "ReconcileSuccess", "Model configuration is ready", model.Generation)

	if err := r.Status().Update(ctx, model); err != nil {
		log.Error(err, "Failed to update status")
		return ctrl.Result{}, err
	}

	log.Info("Successfully reconciled LanguageModel")
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

	// Add fallbacks if specified
	if len(model.Spec.Fallbacks) > 0 {
		fallbacksJSON, err := json.Marshal(model.Spec.Fallbacks)
		if err != nil {
			return err
		}
		data["fallbacks.json"] = string(fallbacksJSON)
	}

	// Create or update the ConfigMap
	configMapName := GenerateConfigMapName(model.Name, "model")
	return CreateOrUpdateConfigMap(ctx, r.Client, r.Scheme, model, configMapName, model.Namespace, data)
}

// handleDeletion handles the deletion of the LanguageModel
func (r *LanguageModelReconciler) handleDeletion(ctx context.Context, model *langopv1alpha1.LanguageModel) (ctrl.Result, error) {
	log := r.Log.WithValues("languagemodel", client.ObjectKeyFromObject(model))

	if HasFinalizer(model) {
		// Delete the ConfigMap
		configMapName := GenerateConfigMapName(model.Name, "model")
		if err := DeleteConfigMap(ctx, r.Client, configMapName, model.Namespace); err != nil {
			log.Error(err, "Failed to delete ConfigMap")
			return ctrl.Result{}, err
		}

		// Remove finalizer
		RemoveFinalizer(model)
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
