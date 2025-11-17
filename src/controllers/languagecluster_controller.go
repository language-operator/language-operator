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
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"

	langopv1alpha1 "github.com/language-operator/language-operator/api/v1alpha1"
)

// LanguageClusterReconciler reconciles a LanguageCluster object
type LanguageClusterReconciler struct {
	client.Client
	Scheme *runtime.Scheme
	Log    logr.Logger
}

// tracer is the OpenTelemetry tracer for the LanguageCluster controller
var clusterTracer = otel.Tracer("language-operator/cluster-controller")

//+kubebuilder:rbac:groups=langop.io,resources=languageclusters,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=langop.io,resources=languageclusters/status,verbs=get;update;patch

// Reconcile is part of the main kubernetes reconciliation loop
func (r *LanguageClusterReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	// Start OpenTelemetry span for reconciliation
	ctx, span := clusterTracer.Start(ctx, "cluster.reconcile")
	defer span.End()

	// Add basic span attributes from request
	span.SetAttributes(
		attribute.String("cluster.name", req.Name),
		attribute.String("cluster.namespace", req.Namespace),
	)

	log := log.FromContext(ctx)

	// Fetch LanguageCluster
	cluster := &langopv1alpha1.LanguageCluster{}
	if err := r.Get(ctx, req.NamespacedName, cluster); err != nil {
		if client.IgnoreNotFound(err) == nil {
			span.SetStatus(codes.Ok, "Resource not found (deleted)")
		} else {
			span.RecordError(err)
			span.SetStatus(codes.Error, "Failed to get LanguageCluster")
		}
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	// Add cluster-specific attributes to span
	span.SetAttributes(
		attribute.Int64("cluster.generation", cluster.Generation),
	)

	// LanguageCluster is now just a logical grouping - no namespace management
	// Child resources reference the cluster and live in the same namespace
	cluster.Status.Phase = "Ready"
	SetCondition(&cluster.Status.Conditions, "Ready", metav1.ConditionTrue,
		"ReconcileSuccess", "LanguageCluster is ready", cluster.Generation)

	if err := r.Status().Update(ctx, cluster); err != nil {
		log.Error(err, "Failed to update status")
		span.RecordError(err)
		span.SetStatus(codes.Error, "Failed to update status")
		return ctrl.Result{}, err
	}

	span.SetStatus(codes.Ok, "Reconciliation successful")
	return ctrl.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *LanguageClusterReconciler) SetupWithManager(mgr ctrl.Manager, concurrency int) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&langopv1alpha1.LanguageCluster{}).
		Complete(r)
}
