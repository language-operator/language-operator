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
	rbacv1 "k8s.io/api/rbac/v1"
	"k8s.io/apimachinery/pkg/api/errors"
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
//+kubebuilder:rbac:groups=langop.io,resources=languagemodels,verbs=get;list;delete
//+kubebuilder:rbac:groups=langop.io,resources=languagepersonas,verbs=get;list;delete
//+kubebuilder:rbac:groups=rbac.authorization.k8s.io,resources=roles,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=rbac.authorization.k8s.io,resources=rolebindings,verbs=get;list;watch;create;update;patch;delete

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

	// Ensure agent RBAC resources exist in the cluster namespace
	if err := r.reconcileAgentRBAC(ctx, cluster); err != nil {
		log.Error(err, "Failed to reconcile agent RBAC")
		span.RecordError(err)
		span.SetStatus(codes.Error, "Failed to reconcile agent RBAC")
		SetCondition(&cluster.Status.Conditions, "Ready", metav1.ConditionFalse,
			"RBACError", err.Error(), cluster.Generation)
		if updateErr := r.Status().Update(ctx, cluster); updateErr != nil {
			log.Error(updateErr, "Failed to update status after RBAC error")
		}
		reconcileErr = err
		return ctrl.Result{}, err
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

// reconcileAgentRBAC ensures the agent RBAC resources exist in the cluster namespace
func (r *LanguageClusterReconciler) reconcileAgentRBAC(ctx context.Context, cluster *langopv1alpha1.LanguageCluster) error {
	log := log.FromContext(ctx)
	namespace := cluster.Namespace

	log.V(1).Info("Reconciling agent RBAC", "cluster", cluster.Name, "namespace", namespace)

	// Create the agents Role
	agentsRole := &rbacv1.Role{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "agents",
			Namespace: namespace,
			Labels: map[string]string{
				"app.kubernetes.io/name":       "language-operator",
				"app.kubernetes.io/managed-by": "language-operator",
				"app.kubernetes.io/component":  "agent-rbac",
				"langop.io/cluster":            cluster.Name,
			},
		},
		Rules: []rbacv1.PolicyRule{
			{
				APIGroups: []string{""},
				Resources: []string{"events"},
				Verbs:     []string{"create"},
			},
		},
	}

	// Set cluster as owner so RBAC gets cleaned up when cluster is deleted
	if err := controllerutil.SetControllerReference(cluster, agentsRole, r.Scheme); err != nil {
		return fmt.Errorf("failed to set controller reference for agents Role: %w", err)
	}

	// Create or update the Role
	if err := r.Create(ctx, agentsRole); err != nil {
		if !errors.IsAlreadyExists(err) {
			return fmt.Errorf("failed to create agents Role: %w", err)
		}
		// Role already exists, update it if needed
		existingRole := &rbacv1.Role{}
		if err := r.Get(ctx, client.ObjectKeyFromObject(agentsRole), existingRole); err != nil {
			return fmt.Errorf("failed to get existing agents Role: %w", err)
		}
		existingRole.Rules = agentsRole.Rules
		if err := r.Update(ctx, existingRole); err != nil {
			return fmt.Errorf("failed to update agents Role: %w", err)
		}
		log.V(1).Info("Updated existing agents Role", "namespace", namespace)
	} else {
		log.Info("Created agents Role", "namespace", namespace)
	}

	// Create the agents RoleBinding
	agentsRoleBinding := &rbacv1.RoleBinding{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "agents",
			Namespace: namespace,
			Labels: map[string]string{
				"app.kubernetes.io/name":       "language-operator",
				"app.kubernetes.io/managed-by": "language-operator",
				"app.kubernetes.io/component":  "agent-rbac",
				"langop.io/cluster":            cluster.Name,
			},
		},
		Subjects: []rbacv1.Subject{
			{
				Kind:      "ServiceAccount",
				Name:      "default",
				Namespace: namespace,
			},
		},
		RoleRef: rbacv1.RoleRef{
			APIGroup: "rbac.authorization.k8s.io",
			Kind:     "Role",
			Name:     "agents",
		},
	}

	// Set cluster as owner so RBAC gets cleaned up when cluster is deleted
	if err := controllerutil.SetControllerReference(cluster, agentsRoleBinding, r.Scheme); err != nil {
		return fmt.Errorf("failed to set controller reference for agents RoleBinding: %w", err)
	}

	// Create or update the RoleBinding
	if err := r.Create(ctx, agentsRoleBinding); err != nil {
		if !errors.IsAlreadyExists(err) {
			return fmt.Errorf("failed to create agents RoleBinding: %w", err)
		}
		// RoleBinding already exists, update it if needed
		existingRoleBinding := &rbacv1.RoleBinding{}
		if err := r.Get(ctx, client.ObjectKeyFromObject(agentsRoleBinding), existingRoleBinding); err != nil {
			return fmt.Errorf("failed to get existing agents RoleBinding: %w", err)
		}
		existingRoleBinding.Subjects = agentsRoleBinding.Subjects
		existingRoleBinding.RoleRef = agentsRoleBinding.RoleRef
		if err := r.Update(ctx, existingRoleBinding); err != nil {
			return fmt.Errorf("failed to update agents RoleBinding: %w", err)
		}
		log.V(1).Info("Updated existing agents RoleBinding", "namespace", namespace)
	} else {
		log.Info("Created agents RoleBinding", "namespace", namespace)
	}

	log.V(1).Info("Successfully reconciled agent RBAC", "cluster", cluster.Name, "namespace", namespace)
	return nil
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

	// Delete all LanguageModels that reference this cluster
	modelList := &langopv1alpha1.LanguageModelList{}
	if err := r.List(ctx, modelList, client.InNamespace(namespace)); err != nil {
		return fmt.Errorf("failed to list models in namespace %s: %w", namespace, err)
	}

	for _, model := range modelList.Items {
		if model.Spec.ClusterRef == clusterName {
			log.Info("Deleting model", "model", model.Name, "cluster", clusterName)
			if err := r.Delete(ctx, &model, client.PropagationPolicy(metav1.DeletePropagationBackground)); err != nil {
				if client.IgnoreNotFound(err) != nil {
					log.Error(err, "Failed to delete model", "model", model.Name)
					// Continue with other resources, don't fail completely
				}
			}
		}
	}

	// Delete all LanguagePersonas that reference this cluster
	personaList := &langopv1alpha1.LanguagePersonaList{}
	if err := r.List(ctx, personaList, client.InNamespace(namespace)); err != nil {
		return fmt.Errorf("failed to list personas in namespace %s: %w", namespace, err)
	}

	for _, persona := range personaList.Items {
		if persona.Spec.ClusterRef == clusterName {
			log.Info("Deleting persona", "persona", persona.Name, "cluster", clusterName)
			if err := r.Delete(ctx, &persona, client.PropagationPolicy(metav1.DeletePropagationBackground)); err != nil {
				if client.IgnoreNotFound(err) != nil {
					log.Error(err, "Failed to delete persona", "persona", persona.Name)
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
