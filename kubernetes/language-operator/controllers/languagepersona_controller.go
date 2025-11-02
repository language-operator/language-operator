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
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	langopv1alpha1 "github.com/based/language-operator/api/v1alpha1"
)

// LanguagePersonaReconciler reconciles a LanguagePersona object
type LanguagePersonaReconciler struct {
	client.Client
	Scheme *runtime.Scheme
	Log    logr.Logger
}

//+kubebuilder:rbac:groups=langop.io,resources=languagepersonas,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=langop.io,resources=languagepersonas/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=langop.io,resources=languagepersonas/finalizers,verbs=update
//+kubebuilder:rbac:groups="",resources=configmaps,verbs=get;list;watch;create;update;patch;delete

// Reconcile reconciles a LanguagePersona resource
func (r *LanguagePersonaReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log := r.Log.WithValues("languagepersona", req.NamespacedName)

	// Fetch the LanguagePersona instance
	persona := &langopv1alpha1.LanguagePersona{}
	if err := r.Get(ctx, req.NamespacedName, persona); err != nil {
		if apierrors.IsNotFound(err) {
			log.Info("LanguagePersona resource not found. Ignoring since object must be deleted")
			return ctrl.Result{}, nil
		}
		log.Error(err, "Failed to get LanguagePersona")
		return ctrl.Result{}, err
	}

	// Handle deletion
	if !persona.ObjectMeta.DeletionTimestamp.IsZero() {
		return r.handleDeletion(ctx, persona)
	}

	// Add finalizer if it doesn't exist
	if !HasFinalizer(persona) {
		AddFinalizer(persona)
		if err := r.Update(ctx, persona); err != nil {
			log.Error(err, "Failed to add finalizer")
			return ctrl.Result{}, err
		}
		return ctrl.Result{Requeue: true}, nil
	}

	// Reconcile the ConfigMap
	if err := r.reconcileConfigMap(ctx, persona); err != nil {
		log.Error(err, "Failed to reconcile ConfigMap")
		SetCondition(&persona.Status.Conditions, "Ready", metav1.ConditionFalse, "ReconcileError", err.Error(), persona.Generation)
		persona.Status.Phase = "Failed"
		if statusErr := r.Status().Update(ctx, persona); statusErr != nil {
			log.Error(statusErr, "Failed to update status")
		}
		return ctrl.Result{}, err
	}

	// Update status
	persona.Status.ObservedGeneration = persona.Generation
	persona.Status.Phase = "Ready"
	// Status fields updated
	SetCondition(&persona.Status.Conditions, "Ready", metav1.ConditionTrue, "ReconcileSuccess", "Persona configuration is ready", persona.Generation)

	if err := r.Status().Update(ctx, persona); err != nil {
		log.Error(err, "Failed to update status")
		return ctrl.Result{}, err
	}

	log.Info("Successfully reconciled LanguagePersona")
	return ctrl.Result{}, nil
}

// reconcileConfigMap creates or updates the ConfigMap for the persona
func (r *LanguagePersonaReconciler) reconcileConfigMap(ctx context.Context, persona *langopv1alpha1.LanguagePersona) error {
	// Create ConfigMap data from persona spec
	data := make(map[string]string)

	// Serialize the spec as JSON
	specJSON, err := json.Marshal(persona.Spec)
	if err != nil {
		return err
	}
	data["persona.json"] = string(specJSON)

	// Add individual fields for easy access
	data["displayName"] = persona.Spec.DisplayName
	data["description"] = persona.Spec.Description
	data["systemPrompt"] = persona.Spec.SystemPrompt
	if persona.Spec.Tone != "" {
		data["tone"] = persona.Spec.Tone
	}
	if persona.Spec.Language != "" {
		data["language"] = persona.Spec.Language
	}

	// Serialize instructions
	if len(persona.Spec.Instructions) > 0 {
		instructionsJSON, err := json.Marshal(persona.Spec.Instructions)
		if err != nil {
			return err
		}
		data["instructions.json"] = string(instructionsJSON)
	}

	// Create or update the ConfigMap
	configMapName := GenerateConfigMapName(persona.Name, "persona")
	return CreateOrUpdateConfigMap(ctx, r.Client, r.Scheme, persona, configMapName, persona.Namespace, data)
}

// handleDeletion handles the deletion of the LanguagePersona
func (r *LanguagePersonaReconciler) handleDeletion(ctx context.Context, persona *langopv1alpha1.LanguagePersona) (ctrl.Result, error) {
	log := r.Log.WithValues("languagepersona", client.ObjectKeyFromObject(persona))

	if HasFinalizer(persona) {
		// Delete the ConfigMap
		configMapName := GenerateConfigMapName(persona.Name, "persona")
		if err := DeleteConfigMap(ctx, r.Client, configMapName, persona.Namespace); err != nil {
			log.Error(err, "Failed to delete ConfigMap")
			return ctrl.Result{}, err
		}

		// Remove finalizer
		RemoveFinalizer(persona)
		if err := r.Update(ctx, persona); err != nil {
			log.Error(err, "Failed to remove finalizer")
			return ctrl.Result{}, err
		}
	}

	return ctrl.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager
func (r *LanguagePersonaReconciler) SetupWithManager(mgr ctrl.Manager, concurrency int) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&langopv1alpha1.LanguagePersona{}).
		Complete(r)
}
