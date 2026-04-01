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
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/log"

	langopv1alpha1 "github.com/language-operator/language-operator/api/v1alpha1"
	"github.com/language-operator/language-operator/pkg/reconciler"
)

// LanguageAgentRuntimeReconciler reconciles a LanguageAgentRuntime object.
// The runtime is a cluster-scoped preset; it owns no child resources.
// The agent controller reads runtimes at reconcile time to merge defaults into agents.
type LanguageAgentRuntimeReconciler struct {
	client.Client
	Scheme *runtime.Scheme
	Log    logr.Logger
}

//+kubebuilder:rbac:groups=langop.io,resources=languageagentruntimes,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=langop.io,resources=languageagentruntimes/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=langop.io,resources=languageagentruntimes/finalizers,verbs=update

// Reconcile reconciles a LanguageAgentRuntime resource.
func (r *LanguageAgentRuntimeReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	helper := &reconciler.ReconcileHelper[*langopv1alpha1.LanguageAgentRuntime]{
		Client:       r.Client,
		TracerName:   "language-operator/agentruntime-controller",
		ResourceType: "agentruntime",
	}

	rt := &langopv1alpha1.LanguageAgentRuntime{}
	result, err := helper.StartReconcile(ctx, req, rt)
	if err != nil {
		return ctrl.Result{}, err
	}
	if result == nil {
		return ctrl.Result{}, nil
	}

	var reconcileErr error
	defer func() {
		result.CompleteReconcile(reconcileErr)
	}()

	ctx = result.Ctx
	span := result.Span
	log := log.FromContext(ctx)

	// Handle deletion
	if !rt.ObjectMeta.DeletionTimestamp.IsZero() {
		return r.handleDeletion(ctx, rt)
	}

	// Add finalizer if not present
	if !controllerutil.ContainsFinalizer(rt, FinalizerName) {
		controllerutil.AddFinalizer(rt, FinalizerName)
		if err := r.Update(ctx, rt); err != nil {
			log.Error(err, "Failed to add finalizer")
			reconcileErr = err
			return ctrl.Result{}, err
		}
		return ctrl.Result{Requeue: true}, nil
	}

	// Mark as Ready
	rt.Status.ObservedGeneration = rt.Generation
	SetCondition(&rt.Status.Conditions, langopv1alpha1.ConditionReady, metav1.ConditionTrue,
		langopv1alpha1.ReasonReconcileSuccess, "LanguageAgentRuntime is ready", rt.Generation)

	if err := r.Status().Update(ctx, rt); err != nil {
		log.Error(err, "Failed to update LanguageAgentRuntime status")
		span.RecordError(err)
		span.SetStatus(codes.Error, "Failed to update status")
		reconcileErr = err
		return ctrl.Result{}, err
	}

	log.Info("Successfully reconciled LanguageAgentRuntime")
	span.SetStatus(codes.Ok, "Reconciliation successful")
	return ctrl.Result{}, nil
}

func (r *LanguageAgentRuntimeReconciler) handleDeletion(ctx context.Context, rt *langopv1alpha1.LanguageAgentRuntime) (ctrl.Result, error) {
	return ctrl.Result{}, RemoveFinalizer(ctx, r.Client, rt)
}

// SetupWithManager sets up the controller with the Manager.
func (r *LanguageAgentRuntimeReconciler) SetupWithManager(mgr ctrl.Manager, concurrency int) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&langopv1alpha1.LanguageAgentRuntime{}).
		Complete(r)
}
