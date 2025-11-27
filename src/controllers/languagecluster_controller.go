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
	"fmt"
	"net"
	"os"
	"time"

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

// LanguageClusterReconciler reconciles a LanguageCluster object
type LanguageClusterReconciler struct {
	client.Client
	Scheme *runtime.Scheme
	Log    logr.Logger
}

//+kubebuilder:rbac:groups=langop.io,resources=languageclusters,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=langop.io,resources=languageclusters/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=langop.io,resources=languageclusters/finalizers,verbs=update
//+kubebuilder:rbac:groups=langop.io,resources=languageagents,verbs=get;list;delete
//+kubebuilder:rbac:groups=langop.io,resources=languagetools,verbs=get;list;delete

// Reconcile is part of the main kubernetes reconciliation loop
func (r *LanguageClusterReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	// Use the reconciler helper for common setup
	helper := &reconciler.ReconcileHelper[*langopv1alpha1.LanguageCluster]{
		Client:       r.Client,
		TracerName:   "language-operator/cluster-controller",
		ResourceType: "cluster",
	}

	cluster := &langopv1alpha1.LanguageCluster{}
	result, err := helper.StartReconcile(ctx, req, cluster)
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

	// Handle deletion
	if !cluster.DeletionTimestamp.IsZero() {
		if controllerutil.ContainsFinalizer(cluster, FinalizerName) {
			// Cleanup dependent resources
			if err := r.cleanupDependentResources(ctx, cluster); err != nil {
				log.Error(err, "Failed to cleanup dependent resources")
				span.RecordError(err)
				span.SetStatus(codes.Error, "Failed to cleanup dependent resources")
				reconcileErr = err
				return ctrl.Result{}, err
			}

			// Remove finalizer
			controllerutil.RemoveFinalizer(cluster, FinalizerName)
			if err := r.Update(ctx, cluster); err != nil {
				log.Error(err, "Failed to remove finalizer")
				span.SetStatus(codes.Error, "Failed to remove finalizer")
				reconcileErr = err
				return ctrl.Result{}, err
			}
		}
		return ctrl.Result{}, nil
	}

	// Add finalizer if not present
	if !controllerutil.ContainsFinalizer(cluster, FinalizerName) {
		controllerutil.AddFinalizer(cluster, FinalizerName)
		if err := r.Update(ctx, cluster); err != nil {
			log.Error(err, "Failed to add finalizer")
			span.RecordError(err)
			span.SetStatus(codes.Error, "Failed to add finalizer")
			reconcileErr = err
			return ctrl.Result{}, err
		}
		// Requeue after adding finalizer
		return ctrl.Result{Requeue: true}, nil
	}

	// Validate DNS configuration if domain is set
	if cluster.Spec.Domain != "" {
		r.validateDNS(ctx, cluster)
	}

	// LanguageCluster is now just a logical grouping - no namespace management
	// Child resources reference the cluster and live in the same namespace
	cluster.Status.Phase = "Ready"
	SetCondition(&cluster.Status.Conditions, "Ready", metav1.ConditionTrue,
		"ReconcileSuccess", "LanguageCluster is ready", cluster.Generation)

	if err := r.Status().Update(ctx, cluster); err != nil {
		log.Error(err, "Failed to update status")
		span.RecordError(err)
		span.SetStatus(codes.Error, "Failed to update status")
		reconcileErr = err
		return ctrl.Result{}, err
	}

	span.SetStatus(codes.Ok, "Reconciliation successful")
	return ctrl.Result{}, nil
}

// cleanupDependentResources removes all resources that reference this cluster
func (r *LanguageClusterReconciler) cleanupDependentResources(ctx context.Context, cluster *langopv1alpha1.LanguageCluster) error {
	log := log.FromContext(ctx)
	clusterName := cluster.Name
	namespace := cluster.Namespace

	log.Info("Cleaning up dependent resources", "cluster", clusterName, "namespace", namespace)

	// Delete all LanguageAgents that reference this cluster
	agentList := &langopv1alpha1.LanguageAgentList{}
	if err := r.List(ctx, agentList, client.InNamespace(namespace)); err != nil {
		return fmt.Errorf("failed to list agents in namespace %s: %w", namespace, err)
	}

	for _, agent := range agentList.Items {
		if agent.Spec.ClusterRef == clusterName {
			log.Info("Deleting agent", "agent", agent.Name, "cluster", clusterName)
			if err := r.Delete(ctx, &agent, client.PropagationPolicy(metav1.DeletePropagationBackground)); err != nil {
				if client.IgnoreNotFound(err) != nil {
					log.Error(err, "Failed to delete agent", "agent", agent.Name)
					// Continue with other resources, don't fail completely
				}
			}
		}
	}

	// Delete all LanguageTools that reference this cluster
	toolList := &langopv1alpha1.LanguageToolList{}
	if err := r.List(ctx, toolList, client.InNamespace(namespace)); err != nil {
		return fmt.Errorf("failed to list tools in namespace %s: %w", namespace, err)
	}

	for _, tool := range toolList.Items {
		if tool.Spec.ClusterRef == clusterName {
			log.Info("Deleting tool", "tool", tool.Name, "cluster", clusterName)
			if err := r.Delete(ctx, &tool, client.PropagationPolicy(metav1.DeletePropagationBackground)); err != nil {
				if client.IgnoreNotFound(err) != nil {
					log.Error(err, "Failed to delete tool", "tool", tool.Name)
					// Continue with other resources, don't fail completely
				}
			}
		}
	}

	log.Info("Completed cleanup of dependent resources", "cluster", clusterName)
	return nil
}

// validateDNS checks if wildcard DNS is configured for the cluster domain
// This is optional validation that can be disabled via environment variable
func (r *LanguageClusterReconciler) validateDNS(ctx context.Context, cluster *langopv1alpha1.LanguageCluster) {
	log := log.FromContext(ctx)

	// Check if DNS validation is disabled (for air-gapped environments)
	if os.Getenv("DISABLE_DNS_VALIDATION") == "true" {
		log.V(1).Info("DNS validation disabled via environment variable")
		return
	}

	domain := cluster.Spec.Domain
	log.V(1).Info("Validating DNS configuration", "domain", domain)

	// Test DNS resolution with a test subdomain
	testHost := fmt.Sprintf("test-validation.%s", domain)

	// Set a reasonable timeout for DNS lookup
	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	// Perform DNS lookup
	resolver := &net.Resolver{}
	_, err := resolver.LookupHost(ctx, testHost)

	if err != nil {
		// DNS resolution failed - this is expected if wildcard DNS isn't configured
		log.V(1).Info("Wildcard DNS not configured or not accessible",
			"domain", domain, "test_host", testHost, "error", err.Error())

		SetCondition(&cluster.Status.Conditions, "DNSConfigured", metav1.ConditionFalse,
			"WildcardDNSMissing",
			fmt.Sprintf("Wildcard DNS (*.%s) not configured or not accessible. See docs/dns.md for setup instructions.", domain),
			cluster.Generation)

		// Log a helpful message for users
		log.Info("DNS configuration notice",
			"domain", domain,
			"required_dns", fmt.Sprintf("*.%s", domain),
			"documentation", "See docs/dns.md for DNS setup instructions")
	} else {
		// DNS resolution succeeded
		log.V(1).Info("Wildcard DNS configured correctly", "domain", domain)

		SetCondition(&cluster.Status.Conditions, "DNSConfigured", metav1.ConditionTrue,
			"WildcardDNSReady",
			fmt.Sprintf("Wildcard DNS (*.%s) is correctly configured", domain),
			cluster.Generation)
	}
}

// SetupWithManager sets up the controller with the Manager.
func (r *LanguageClusterReconciler) SetupWithManager(mgr ctrl.Manager, concurrency int) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&langopv1alpha1.LanguageCluster{}).
		Complete(r)
}
