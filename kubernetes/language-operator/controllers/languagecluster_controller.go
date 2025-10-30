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
	"fmt"
	"strings"
	"time"

	"github.com/go-logr/logr"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	networkingv1 "k8s.io/api/networking/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/log"

	langopv1alpha1 "github.com/based/language-operator/api/v1alpha1"
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
//+kubebuilder:rbac:groups="",resources=namespaces,verbs=get;list;watch;create;update;patch
//+kubebuilder:rbac:groups=networking.k8s.io,resources=networkpolicies,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=cilium.io,resources=ciliumnetworkpolicies,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=apps,resources=daemonsets,verbs=get;list;watch

// Reconcile is part of the main kubernetes reconciliation loop
func (r *LanguageClusterReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log := log.FromContext(ctx)

	// Fetch LanguageCluster
	cluster := &langopv1alpha1.LanguageCluster{}
	if err := r.Get(ctx, req.NamespacedName, cluster); err != nil {
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	// Handle deletion with finalizer
	if !cluster.DeletionTimestamp.IsZero() {
		return r.handleDeletion(ctx, cluster)
	}

	// Add finalizer if not present
	if !controllerutil.ContainsFinalizer(cluster, FinalizerName) {
		controllerutil.AddFinalizer(cluster, FinalizerName)
		if err := r.Update(ctx, cluster); err != nil {
			return ctrl.Result{}, err
		}
	}

	// Check/Install Cilium
	ciliumStatus, err := r.ensureCilium(ctx, cluster)
	if err != nil {
		log.Error(err, "Failed to ensure Cilium is ready")
		cluster.Status.Phase = "Failed"
		SetCondition(&cluster.Status.Conditions, "CiliumReady", metav1.ConditionFalse,
			"CiliumError", err.Error(), cluster.Generation)
		r.Status().Update(ctx, cluster)
		return ctrl.Result{RequeueAfter: 30 * time.Second}, err
	}
	cluster.Status.CiliumStatus = ciliumStatus

	// Create/verify namespace
	namespace, err := r.ensureNamespace(ctx, cluster)
	if err != nil {
		log.Error(err, "Failed to create/verify namespace")
		return ctrl.Result{}, err
	}
	cluster.Status.Namespace = namespace

	// Apply default-deny NetworkPolicy
	if err := r.ensureDefaultDenyPolicy(ctx, cluster, namespace); err != nil {
		log.Error(err, "Failed to apply default-deny policy")
		return ctrl.Result{}, err
	}

	// Discover group members
	membership, err := r.discoverGroupMembers(ctx, cluster)
	if err != nil {
		log.Error(err, "Failed to discover group members")
		return ctrl.Result{}, err
	}
	cluster.Status.GroupMembership = membership

	// Validate groups
	if err := r.validateGroups(cluster, membership); err != nil {
		cluster.Status.Phase = "Failed"
		SetCondition(&cluster.Status.Conditions, "Valid", metav1.ConditionFalse,
			"ValidationError", err.Error(), cluster.Generation)
		r.Status().Update(ctx, cluster)
		return ctrl.Result{}, err
	}

	// Generate and apply NetworkPolicies
	netpols, err := r.reconcileNetworkPolicies(ctx, cluster, namespace, membership)
	if err != nil {
		log.Error(err, "Failed to reconcile NetworkPolicies")
		return ctrl.Result{}, err
	}
	cluster.Status.NetworkPolicies = netpols

	// Generate and apply Cilium policies
	ciliumPols, err := r.reconcileCiliumPolicies(ctx, cluster, namespace, membership)
	if err != nil {
		log.Error(err, "Failed to reconcile Cilium policies")
		return ctrl.Result{}, err
	}
	cluster.Status.CiliumPolicies = ciliumPols

	// Update status
	cluster.Status.Phase = "Ready"
	SetCondition(&cluster.Status.Conditions, "Ready", metav1.ConditionTrue,
		"ReconcileSuccess", "LanguageCluster is ready", cluster.Generation)

	if err := r.Status().Update(ctx, cluster); err != nil {
		log.Error(err, "Failed to update status")
		return ctrl.Result{}, err
	}

	return ctrl.Result{}, nil
}

func (r *LanguageClusterReconciler) handleDeletion(ctx context.Context, cluster *langopv1alpha1.LanguageCluster) (ctrl.Result, error) {
	log := log.FromContext(ctx)

	if !controllerutil.ContainsFinalizer(cluster, FinalizerName) {
		return ctrl.Result{}, nil
	}

	// Check for member resources
	members, err := r.discoverGroupMembers(ctx, cluster)
	if err != nil {
		return ctrl.Result{}, err
	}

	totalMembers := 0
	for _, info := range members {
		totalMembers += info.Count
	}

	if totalMembers > 0 {
		log.Info("Cannot delete LanguageCluster with active members",
			"cluster", cluster.Name, "memberCount", totalMembers)
		return ctrl.Result{}, fmt.Errorf(
			"cannot delete LanguageCluster %s: still has %d member resources",
			cluster.Name, totalMembers)
	}

	// Cleanup resources
	if err := r.cleanupResources(ctx, cluster); err != nil {
		return ctrl.Result{}, err
	}

	// Remove finalizer
	controllerutil.RemoveFinalizer(cluster, FinalizerName)
	if err := r.Update(ctx, cluster); err != nil {
		return ctrl.Result{}, err
	}

	return ctrl.Result{}, nil
}

func (r *LanguageClusterReconciler) ensureCilium(ctx context.Context, cluster *langopv1alpha1.LanguageCluster) (langopv1alpha1.CiliumStatus, error) {
	// Check if Cilium DaemonSet exists
	ds := &appsv1.DaemonSet{}
	err := r.Get(ctx, types.NamespacedName{Name: "cilium", Namespace: "kube-system"}, ds)

	if err == nil {
		// Cilium is installed
		return langopv1alpha1.CiliumStatus{
			Installed: true,
			Version:   extractCiliumVersion(ds),
			Ready:     isCiliumReady(ds),
		}, nil
	}

	if !errors.IsNotFound(err) {
		return langopv1alpha1.CiliumStatus{}, err
	}

	// Cilium not found - return error with helpful message
	return langopv1alpha1.CiliumStatus{}, fmt.Errorf("Cilium is not installed. Please install Cilium before creating a LanguageCluster. See: https://docs.cilium.io/en/stable/gettingstarted/k8s-install-default/")
}

func extractCiliumVersion(ds *appsv1.DaemonSet) string {
	for _, container := range ds.Spec.Template.Spec.Containers {
		if container.Name == "cilium-agent" {
			// Extract version from image tag
			// Format: quay.io/cilium/cilium:v1.15.0
			parts := strings.Split(container.Image, ":")
			if len(parts) == 2 {
				return strings.TrimPrefix(parts[1], "v")
			}
		}
	}
	return "unknown"
}

func isCiliumReady(ds *appsv1.DaemonSet) bool {
	return ds.Status.NumberReady > 0 && ds.Status.NumberReady == ds.Status.DesiredNumberScheduled
}

func (r *LanguageClusterReconciler) ensureNamespace(ctx context.Context, cluster *langopv1alpha1.LanguageCluster) (string, error) {
	namespace := cluster.Spec.Namespace
	if namespace == "" {
		namespace = cluster.Name + "-ns"
	}

	ns := &corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name: namespace,
			Labels: map[string]string{
				"langop.io/cluster": cluster.Name,
				"langop.io/managed": "true",
			},
		},
	}

	err := r.Get(ctx, types.NamespacedName{Name: namespace}, ns)
	if errors.IsNotFound(err) {
		// Create namespace
		if err := r.Create(ctx, ns); err != nil {
			return "", fmt.Errorf("failed to create namespace: %w", err)
		}
	} else if err != nil {
		return "", err
	}

	return namespace, nil
}

func (r *LanguageClusterReconciler) ensureDefaultDenyPolicy(ctx context.Context, cluster *langopv1alpha1.LanguageCluster, namespace string) error {
	// Only create default-deny if policy is "deny"
	if cluster.Spec.Network.DefaultPolicy != "deny" && cluster.Spec.Network.DefaultPolicy != "" {
		return nil
	}

	// Default-deny policy handled in networkpolicy_builder.go
	policy := buildDefaultDenyPolicy(cluster, namespace)

	existing := &networkingv1.NetworkPolicy{}
	err := r.Get(ctx, types.NamespacedName{Name: policy.Name, Namespace: namespace}, existing)

	if errors.IsNotFound(err) {
		return r.Create(ctx, policy)
	} else if err != nil {
		return err
	}

	// Update if needed
	existing.Spec = policy.Spec
	return r.Update(ctx, existing)
}

func (r *LanguageClusterReconciler) discoverGroupMembers(ctx context.Context, cluster *langopv1alpha1.LanguageCluster) (map[string]langopv1alpha1.GroupMembershipInfo, error) {
	membership := make(map[string]langopv1alpha1.GroupMembershipInfo)

	// Initialize all groups
	for _, group := range cluster.Spec.Groups {
		membership[group.Name] = langopv1alpha1.GroupMembershipInfo{
			Count:     0,
			Resources: []string{},
		}
	}

	// Add default group
	membership["default"] = langopv1alpha1.GroupMembershipInfo{Count: 0, Resources: []string{}}

	// Scan LanguageTools
	tools := &langopv1alpha1.LanguageToolList{}
	if err := r.List(ctx, tools); err != nil {
		return nil, err
	}
	for _, tool := range tools.Items {
		if tool.Spec.ClusterRef == cluster.Name {
			group := tool.Spec.Group
			if group == "" {
				group = "default"
			}
			info := membership[group]
			info.Count++
			info.Resources = append(info.Resources, fmt.Sprintf("LanguageTool/%s", tool.Name))
			membership[group] = info
		}
	}

	// Scan LanguageAgents
	agents := &langopv1alpha1.LanguageAgentList{}
	if err := r.List(ctx, agents); err != nil {
		return nil, err
	}
	for _, agent := range agents.Items {
		if agent.Spec.ClusterRef == cluster.Name {
			group := agent.Spec.Group
			if group == "" {
				group = "default"
			}
			info := membership[group]
			info.Count++
			info.Resources = append(info.Resources, fmt.Sprintf("LanguageAgent/%s", agent.Name))
			membership[group] = info
		}
	}

	// Scan LanguageClients
	clients := &langopv1alpha1.LanguageClientList{}
	if err := r.List(ctx, clients); err != nil {
		return nil, err
	}
	for _, client := range clients.Items {
		if client.Spec.ClusterRef == cluster.Name {
			group := client.Spec.Group
			if group == "" {
				group = "default"
			}
			info := membership[group]
			info.Count++
			info.Resources = append(info.Resources, fmt.Sprintf("LanguageClient/%s", client.Name))
			membership[group] = info
		}
	}

	return membership, nil
}

func (r *LanguageClusterReconciler) validateGroups(cluster *langopv1alpha1.LanguageCluster, membership map[string]langopv1alpha1.GroupMembershipInfo) error {
	// Allow empty groups for now
	return nil
}

func (r *LanguageClusterReconciler) cleanupResources(ctx context.Context, cluster *langopv1alpha1.LanguageCluster) error {
	namespace := cluster.Status.Namespace
	if namespace == "" {
		return nil
	}

	// Delete namespace (cascades to all resources within)
	ns := &corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name: namespace,
		},
	}

	return client.IgnoreNotFound(r.Delete(ctx, ns))
}

// SetupWithManager sets up the controller with the Manager.
func (r *LanguageClusterReconciler) SetupWithManager(mgr ctrl.Manager, concurrency int) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&langopv1alpha1.LanguageCluster{}).
		Owns(&corev1.Namespace{}).
		Complete(r)
}
