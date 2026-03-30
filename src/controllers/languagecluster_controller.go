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
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"maps"
	"net"
	"os"
	"strings"
	"time"

	"github.com/go-logr/logr"
	"go.opentelemetry.io/otel/codes"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	networkingv1 "k8s.io/api/networking/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/client-go/tools/record"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	langopv1alpha1 "github.com/language-operator/language-operator/api/v1alpha1"
	"github.com/language-operator/language-operator/pkg/events"
	"github.com/language-operator/language-operator/pkg/reconciler"
	"k8s.io/utils/ptr"
)

// LanguageClusterReconciler reconciles a LanguageCluster object
type LanguageClusterReconciler struct {
	client.Client
	Scheme                  *runtime.Scheme
	Log                     logr.Logger
	Recorder                record.EventRecorder
	EventManager            *events.EventManager
	NetworkIsolationEnabled bool
	GatewayImage            string
	GatewayImagePullPolicy  corev1.PullPolicy
}

func (r *LanguageClusterReconciler) gatewayImage() string {
	if r.GatewayImage != "" {
		return r.GatewayImage
	}
	return "ghcr.io/language-operator/model:latest"
}

func (r *LanguageClusterReconciler) gatewayImagePullPolicy(cluster *langopv1alpha1.LanguageCluster) corev1.PullPolicy {
	if cluster.Spec.Gateway != nil && cluster.Spec.Gateway.Deployment.ImagePullPolicy != "" {
		return cluster.Spec.Gateway.Deployment.ImagePullPolicy
	}
	return r.GatewayImagePullPolicy
}

func defaultGatewayLivenessProbe() *corev1.Probe {
	return &corev1.Probe{
		ProbeHandler: corev1.ProbeHandler{
			HTTPGet: &corev1.HTTPGetAction{
				Path: "/health/liveliness",
				Port: intstr.FromInt(4000),
			},
		},
		InitialDelaySeconds: 30,
		PeriodSeconds:       300,
		TimeoutSeconds:      30,
	}
}

func defaultGatewayReadinessProbe() *corev1.Probe {
	return &corev1.Probe{
		ProbeHandler: corev1.ProbeHandler{
			HTTPGet: &corev1.HTTPGetAction{
				Path: "/health/readiness",
				Port: intstr.FromInt(4000),
			},
		},
		InitialDelaySeconds: 30,
		PeriodSeconds:       300,
		TimeoutSeconds:      30,
	}
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
//+kubebuilder:rbac:groups=networking.k8s.io,resources=networkpolicies,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=apps,resources=deployments,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups="",resources=services,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups="",resources=configmaps,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=networking.k8s.io,resources=ingresses,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups="",resources=resourcequotas,verbs=get;list;watch;create;update;patch;delete

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
		if r.EventManager != nil {
			r.EventManager.RecordClusterCreated(cluster)
		}
		// Requeue after adding finalizer
		return ctrl.Result{Requeue: true}, nil
	}

	// Mark Pending on first real reconcile so kubectl get shows something meaningful
	// before resources are fully provisioned.
	if cluster.Status.Phase == "" {
		cluster.Status.Phase = events.PhaseStatusPending
		if updateErr := r.Status().Update(ctx, cluster); updateErr != nil {
			log.Error(updateErr, "Failed to update cluster phase to Pending")
		}
	}

	// Ensure the namespace for this cluster exists
	if err := r.reconcileNamespace(ctx, cluster); err != nil {
		log.Error(err, "Failed to reconcile namespace")
		span.RecordError(err)
		span.SetStatus(codes.Error, "Failed to reconcile namespace")
		cluster.Status.Phase = events.PhaseStatusFailed
		SetCondition(&cluster.Status.Conditions, "Ready", metav1.ConditionFalse,
			"NamespaceError", err.Error(), cluster.Generation)
		if updateErr := r.Status().Update(ctx, cluster); updateErr != nil {
			log.Error(updateErr, "Failed to update status after namespace error")
		}
		reconcileErr = err
		return ctrl.Result{}, err
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
		if r.EventManager != nil {
			r.EventManager.RecordRBACFailed(cluster, err)
		}
		cluster.Status.Phase = events.PhaseStatusFailed
		SetCondition(&cluster.Status.Conditions, "Ready", metav1.ConditionFalse,
			"RBACError", err.Error(), cluster.Generation)
		if updateErr := r.Status().Update(ctx, cluster); updateErr != nil {
			log.Error(updateErr, "Failed to update status after RBAC error")
		}
		reconcileErr = err
		return ctrl.Result{}, err
	}

	// Ensure NetworkPolicy exists for agent communication (if enabled)
	if r.NetworkIsolationEnabled {
		if err := r.reconcileNetworkPolicy(ctx, cluster); err != nil {
			log.Error(err, "Failed to reconcile NetworkPolicy")
			span.RecordError(err)
			span.SetStatus(codes.Error, "Failed to reconcile NetworkPolicy")
			if r.EventManager != nil {
				r.EventManager.RecordNetworkPolicyFailed(cluster, err)
			}
			cluster.Status.Phase = events.PhaseStatusFailed
			SetCondition(&cluster.Status.Conditions, "Ready", metav1.ConditionFalse,
				"NetworkPolicyError", err.Error(), cluster.Generation)
			if updateErr := r.Status().Update(ctx, cluster); updateErr != nil {
				log.Error(updateErr, "Failed to update status after NetworkPolicy error")
			}
			reconcileErr = err
			return ctrl.Result{}, err
		} else {
			SetCondition(&cluster.Status.Conditions, "NetworkPolicyReady", metav1.ConditionTrue, "NetworkPolicyReady",
				"NetworkPolicy created successfully", cluster.Generation)
		}
	} else {
		SetCondition(&cluster.Status.Conditions, "NetworkPolicyReady", metav1.ConditionTrue, "NetworkPolicyDisabled",
			"NetworkPolicy creation disabled via networkIsolation.enabled=false", cluster.Generation)
		log.V(1).Info("Network isolation disabled - skipping NetworkPolicy creation")
	}

	// Reconcile shared LiteLLM gateway Deployment + Service + ConfigMap
	if err := r.reconcileGateway(ctx, cluster); err != nil {
		log.Error(err, "Failed to reconcile gateway")
		span.RecordError(err)
		span.SetStatus(codes.Error, "Failed to reconcile gateway")
		cluster.Status.Phase = events.PhaseStatusFailed
		cluster.Status.GatewayReady = ptr.To(false)
		SetCondition(&cluster.Status.Conditions, "GatewayReady", metav1.ConditionFalse,
			"GatewayError", err.Error(), cluster.Generation)
		if updateErr := r.Status().Update(ctx, cluster); updateErr != nil {
			log.Error(updateErr, "Failed to update status after gateway error")
		}
		reconcileErr = err
		return ctrl.Result{}, err
	}

	// Reconcile gateway Ingress if domain is configured
	if cluster.Spec.Domain != "" {
		if err := r.reconcileGatewayIngress(ctx, cluster); err != nil {
			log.Error(err, "Failed to reconcile gateway ingress")
			span.RecordError(err)
			span.SetStatus(codes.Error, "Failed to reconcile gateway ingress")
			cluster.Status.Phase = events.PhaseStatusFailed
			cluster.Status.GatewayReady = ptr.To(false)
			SetCondition(&cluster.Status.Conditions, "GatewayReady", metav1.ConditionFalse,
				"GatewayIngressError", err.Error(), cluster.Generation)
			if updateErr := r.Status().Update(ctx, cluster); updateErr != nil {
				log.Error(updateErr, "Failed to update status after gateway ingress error")
			}
			reconcileErr = err
			return ctrl.Result{}, err
		}
	}

	SetCondition(&cluster.Status.Conditions, "GatewayReady", metav1.ConditionTrue,
		"GatewayReady", "Shared LiteLLM gateway is ready", cluster.Generation)
	cluster.Status.GatewayEndpoint = fmt.Sprintf("http://gateway.%s.svc.cluster.local:8000", cluster.Name)
	cluster.Status.GatewayReady = ptr.To(true)

	// Populate status.capacity with observed usage. Runs before reconcileCapacity so the
	// field is always written even if ResourceQuota management fails (e.g. RBAC not yet applied).
	if err := r.reconcileCapacityStatus(ctx, cluster); err != nil {
		log.Error(err, "Failed to reconcile capacity status")
		// Non-fatal: capacity counts are best-effort
	}

	// Reconcile ResourceQuota for capacity limits
	if err := r.reconcileCapacity(ctx, cluster); err != nil {
		log.Error(err, "Failed to reconcile capacity quota")
		span.RecordError(err)
		span.SetStatus(codes.Error, "Failed to reconcile capacity quota")
		cluster.Status.Phase = events.PhaseStatusFailed
		SetCondition(&cluster.Status.Conditions, "CapacityReady", metav1.ConditionFalse,
			"CapacityError", err.Error(), cluster.Generation)
		if updateErr := r.Status().Update(ctx, cluster); updateErr != nil {
			log.Error(updateErr, "Failed to update status after capacity error")
		}
		reconcileErr = err
		return ctrl.Result{}, err
	}

	cluster.Status.Phase = events.PhaseStatusReady
	SetCondition(&cluster.Status.Conditions, "Ready", metav1.ConditionTrue,
		"ReconcileSuccess", "LanguageCluster is ready", cluster.Generation)

	if r.EventManager != nil {
		r.EventManager.RecordClusterReady(cluster)
	}

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

// reconcileNamespace ensures a namespace named cluster.Name exists
func (r *LanguageClusterReconciler) reconcileNamespace(ctx context.Context, cluster *langopv1alpha1.LanguageCluster) error {
	log := log.FromContext(ctx)
	ns := &corev1.Namespace{}
	err := r.Get(ctx, client.ObjectKey{Name: cluster.Name}, ns)
	if err == nil {
		return nil // already exists
	}
	if !errors.IsNotFound(err) {
		return fmt.Errorf("failed to get namespace %s: %w", cluster.Name, err)
	}
	ns = &corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name: cluster.Name,
			Labels: map[string]string{
				"app.kubernetes.io/managed-by": "language-operator",
				LabelKeyLangopCluster:          cluster.Name,
			},
		},
	}
	if err := r.Create(ctx, ns); err != nil {
		return fmt.Errorf("failed to create namespace %s: %w", cluster.Name, err)
	}
	log.Info("Created namespace", "namespace", cluster.Name)
	return nil
}

// reconcileAgentRBAC ensures the agent RBAC resources exist in the cluster namespace
func (r *LanguageClusterReconciler) reconcileAgentRBAC(ctx context.Context, cluster *langopv1alpha1.LanguageCluster) error {
	log := log.FromContext(ctx)
	namespace := cluster.Name

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
				LabelKeyLangopCluster:          cluster.Name,
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
				LabelKeyLangopCluster:          cluster.Name,
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

// reconcileNetworkPolicy ensures the NetworkPolicy for agent communication exists
func (r *LanguageClusterReconciler) reconcileNetworkPolicy(ctx context.Context, cluster *langopv1alpha1.LanguageCluster) error {
	log := log.FromContext(ctx)
	namespace := cluster.Name

	log.V(1).Info("Reconciling NetworkPolicy", "cluster", cluster.Name, "namespace", namespace)

	// Build egress rules starting with K8s API server access
	tcpProtocol := corev1.ProtocolTCP
	udpProtocol := corev1.ProtocolUDP
	egressRules := []networkingv1.NetworkPolicyEgressRule{
		{
			// Allow access to Kubernetes API server via service discovery (kubernetes.default.svc)
			To: []networkingv1.NetworkPolicyPeer{
				{
					// Target the default namespace where the kubernetes service exists
					NamespaceSelector: &metav1.LabelSelector{
						MatchLabels: map[string]string{
							LabelKeyMetadataName: "default",
						},
					},
				},
			},
			Ports: []networkingv1.NetworkPolicyPort{
				{
					Protocol: &tcpProtocol,
					Port:     &intstr.IntOrString{Type: intstr.Int, IntVal: 443},
				},
			},
		},
		{
			// Allow DNS access for name resolution (kube-dns in kube-system namespace)
			To: []networkingv1.NetworkPolicyPeer{
				{
					NamespaceSelector: &metav1.LabelSelector{
						MatchLabels: map[string]string{
							LabelKeyMetadataName: "kube-system",
						},
					},
				},
			},
			Ports: []networkingv1.NetworkPolicyPort{
				{
					Protocol: &udpProtocol,
					Port:     &intstr.IntOrString{Type: intstr.Int, IntVal: 53},
				},
				{
					Protocol: &tcpProtocol,
					Port:     &intstr.IntOrString{Type: intstr.Int, IntVal: 53},
				},
			},
		},
	}

	// Add user-defined network policies
	for _, rule := range cluster.Spec.NetworkPolicies {
		if rule.To != nil {
			egressRule := networkingv1.NetworkPolicyEgressRule{}

			// Convert langop NetworkPeer to k8s NetworkPolicyPeer
			peer := networkingv1.NetworkPolicyPeer{}

			if rule.To.CIDR != "" {
				peer.IPBlock = &networkingv1.IPBlock{
					CIDR: rule.To.CIDR,
				}
			}

			if rule.To.Service != nil {
				serviceNamespace := rule.To.Service.Namespace
				if serviceNamespace == "" {
					serviceNamespace = namespace
				}
				peer.NamespaceSelector = &metav1.LabelSelector{
					MatchLabels: map[string]string{
						LabelKeyMetadataName: serviceNamespace,
					},
				}
			}

			if rule.To.Group != "" {
				peer.PodSelector = &metav1.LabelSelector{
					MatchLabels: map[string]string{
						LabelKeyLangopGroup: rule.To.Group,
					},
				}
			}

			if rule.To.PodSelector != nil {
				peer.PodSelector = rule.To.PodSelector
			}

			if rule.To.NamespaceSelector != nil {
				peer.NamespaceSelector = rule.To.NamespaceSelector
			}

			// Only include the peer if it has actual selectors
			if peer.IPBlock != nil || peer.PodSelector != nil || peer.NamespaceSelector != nil {
				egressRule.To = append(egressRule.To, peer)
			}

			// Resolve DNS hostnames to CIDR blocks at policy creation time (fail-closed)
			if len(rule.To.DNS) > 0 {
				resolvedCIDRs, err := resolveDNSToCIDRs(rule.To.DNS)
				if err == nil && len(resolvedCIDRs) > 0 {
					for _, cidr := range resolvedCIDRs {
						egressRule.To = append(egressRule.To, networkingv1.NetworkPolicyPeer{
							IPBlock: &networkingv1.IPBlock{CIDR: cidr},
						})
					}
				}
				// DNS resolution failure = fail-closed: no destinations added
			}

			// Convert ports
			if len(rule.Ports) > 0 {
				for _, port := range rule.Ports {
					protocol := corev1.ProtocolTCP
					if port.Protocol != "" {
						protocol = corev1.Protocol(port.Protocol)
					}
					egressRule.Ports = append(egressRule.Ports, networkingv1.NetworkPolicyPort{
						Protocol: &protocol,
						Port:     &intstr.IntOrString{Type: intstr.Int, IntVal: port.Port},
					})
				}
			}

			egressRules = append(egressRules, egressRule)
		}
	}

	// Wire user-defined ingress rules from spec.networkPolicies[].from
	var ingressRules []networkingv1.NetworkPolicyIngressRule
	for _, rule := range cluster.Spec.NetworkPolicies {
		if rule.From == nil {
			continue
		}
		ingressRule := networkingv1.NetworkPolicyIngressRule{
			From: []networkingv1.NetworkPolicyPeer{
				buildIngressPeerFromNetworkPeer(rule.From, namespace),
			},
		}
		for _, port := range rule.Ports {
			protocol := corev1.ProtocolTCP
			if port.Protocol != "" {
				protocol = corev1.Protocol(port.Protocol)
			}
			ingressRule.Ports = append(ingressRule.Ports, networkingv1.NetworkPolicyPort{
				Protocol: &protocol,
				Port:     &intstr.IntOrString{Type: intstr.Int, IntVal: port.Port},
			})
		}
		ingressRules = append(ingressRules, ingressRule)
	}

	policyTypes := []networkingv1.PolicyType{networkingv1.PolicyTypeEgress}
	if len(ingressRules) > 0 {
		policyTypes = append(policyTypes, networkingv1.PolicyTypeIngress)
	}

	// Create NetworkPolicy
	networkPolicy := &networkingv1.NetworkPolicy{
		ObjectMeta: metav1.ObjectMeta{
			Name:      fmt.Sprintf("%s-agents", cluster.Name),
			Namespace: namespace,
			Labels: map[string]string{
				"app.kubernetes.io/name":       "language-operator",
				"app.kubernetes.io/managed-by": "language-operator",
				"app.kubernetes.io/component":  "agent-network-policy",
				LabelKeyLangopCluster:          cluster.Name,
			},
		},
		Spec: networkingv1.NetworkPolicySpec{
			PodSelector: metav1.LabelSelector{
				MatchLabels: map[string]string{
					LabelKeyLangopCluster: cluster.Name,
				},
			},
			PolicyTypes: policyTypes,
			Egress:      egressRules,
			Ingress:     ingressRules,
		},
	}

	// Set cluster as owner so NetworkPolicy gets cleaned up when cluster is deleted
	if err := controllerutil.SetControllerReference(cluster, networkPolicy, r.Scheme); err != nil {
		return fmt.Errorf("failed to set controller reference for NetworkPolicy: %w", err)
	}

	// Create or update the NetworkPolicy
	if err := r.Create(ctx, networkPolicy); err != nil {
		if !errors.IsAlreadyExists(err) {
			return fmt.Errorf("failed to create NetworkPolicy: %w", err)
		}
		// NetworkPolicy already exists, update it if needed
		existingNetworkPolicy := &networkingv1.NetworkPolicy{}
		if err := r.Get(ctx, client.ObjectKeyFromObject(networkPolicy), existingNetworkPolicy); err != nil {
			return fmt.Errorf("failed to get existing NetworkPolicy: %w", err)
		}
		existingNetworkPolicy.Spec = networkPolicy.Spec
		if err := r.Update(ctx, existingNetworkPolicy); err != nil {
			return fmt.Errorf("failed to update NetworkPolicy: %w", err)
		}
		log.V(1).Info("Updated existing NetworkPolicy", "namespace", namespace)
	} else {
		log.Info("Created NetworkPolicy", "namespace", namespace, "name", networkPolicy.Name)
	}

	log.V(1).Info("Successfully reconciled NetworkPolicy", "cluster", cluster.Name, "namespace", namespace)
	return nil
}

// cleanupDependentResources removes all resources that reference this cluster
func (r *LanguageClusterReconciler) cleanupDependentResources(ctx context.Context, cluster *langopv1alpha1.LanguageCluster) error {
	log := log.FromContext(ctx)
	clusterName := cluster.Name
	namespace := cluster.Name

	log.Info("Cleaning up dependent resources", "cluster", clusterName, "namespace", namespace)

	// Delete all LanguageAgents that reference this cluster
	agentList := &langopv1alpha1.LanguageAgentList{}
	if err := r.List(ctx, agentList, client.InNamespace(namespace)); err != nil {
		return fmt.Errorf("failed to list agents in namespace %s: %w", namespace, err)
	}

	for _, agent := range agentList.Items {
		log.Info("Deleting agent", "agent", agent.Name, "cluster", clusterName)
		if err := r.Delete(ctx, &agent, client.PropagationPolicy(metav1.DeletePropagationBackground)); err != nil {
			if client.IgnoreNotFound(err) != nil {
				log.Error(err, "Failed to delete agent", "agent", agent.Name)
				// Continue with other resources, don't fail completely
			}
		}
	}

	// Delete all LanguageTools that reference this cluster
	toolList := &langopv1alpha1.LanguageToolList{}
	if err := r.List(ctx, toolList, client.InNamespace(namespace)); err != nil {
		return fmt.Errorf("failed to list tools in namespace %s: %w", namespace, err)
	}

	for _, tool := range toolList.Items {
		log.Info("Deleting tool", "tool", tool.Name, "cluster", clusterName)
		if err := r.Delete(ctx, &tool, client.PropagationPolicy(metav1.DeletePropagationBackground)); err != nil {
			if client.IgnoreNotFound(err) != nil {
				log.Error(err, "Failed to delete tool", "tool", tool.Name)
				// Continue with other resources, don't fail completely
			}
		}
	}

	// Delete all LanguageModels that reference this cluster
	modelList := &langopv1alpha1.LanguageModelList{}
	if err := r.List(ctx, modelList, client.InNamespace(namespace)); err != nil {
		return fmt.Errorf("failed to list models in namespace %s: %w", namespace, err)
	}

	for _, model := range modelList.Items {
		log.Info("Deleting model", "model", model.Name, "cluster", clusterName)
		if err := r.Delete(ctx, &model, client.PropagationPolicy(metav1.DeletePropagationBackground)); err != nil {
			if client.IgnoreNotFound(err) != nil {
				log.Error(err, "Failed to delete model", "model", model.Name)
				// Continue with other resources, don't fail completely
			}
		}
	}

	// Delete all LanguagePersonas that reference this cluster
	personaList := &langopv1alpha1.LanguagePersonaList{}
	if err := r.List(ctx, personaList, client.InNamespace(namespace)); err != nil {
		return fmt.Errorf("failed to list personas in namespace %s: %w", namespace, err)
	}

	for _, persona := range personaList.Items {
		log.Info("Deleting persona", "persona", persona.Name, "cluster", clusterName)
		if err := r.Delete(ctx, &persona, client.PropagationPolicy(metav1.DeletePropagationBackground)); err != nil {
			if client.IgnoreNotFound(err) != nil {
				log.Error(err, "Failed to delete persona", "persona", persona.Name)
				// Continue with other resources, don't fail completely
			}
		}
	}

	// Delete the namespace for this cluster
	ns := &corev1.Namespace{}
	if err := r.Get(ctx, client.ObjectKey{Name: clusterName}, ns); err == nil {
		log.Info("Deleting namespace", "namespace", clusterName)
		if err := r.Delete(ctx, ns); err != nil && !errors.IsNotFound(err) {
			return fmt.Errorf("failed to delete namespace %s: %w", clusterName, err)
		}
	} else if !errors.IsNotFound(err) {
		return fmt.Errorf("failed to get namespace %s: %w", clusterName, err)
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

// reconcileGateway creates/updates the shared LiteLLM gateway Deployment, Service, and ConfigMap.
func (r *LanguageClusterReconciler) reconcileGateway(ctx context.Context, cluster *langopv1alpha1.LanguageCluster) error {
	log := log.FromContext(ctx)
	namespace := cluster.Name

	// List all LanguageModels in this cluster's namespace
	modelList := &langopv1alpha1.LanguageModelList{}
	if err := r.List(ctx, modelList, client.InNamespace(namespace)); err != nil {
		return fmt.Errorf("failed to list models: %w", err)
	}

	// Build ConfigMap data: one model.json entry per model under models/<name>.json
	cmData := make(map[string]string)
	for _, model := range modelList.Items {
		specJSON, err := json.Marshal(model.Spec)
		if err != nil {
			return fmt.Errorf("failed to marshal model spec %s: %w", model.Name, err)
		}
		cmData["model__"+model.Name+".json"] = string(specJSON)
	}

	// Compute a hash of the config for rolling-restart annotation
	h := sha256.New()
	keys := make([]string, 0, len(cmData))
	for k := range cmData {
		keys = append(keys, k)
	}
	for _, k := range keys {
		h.Write([]byte(k))
		h.Write([]byte(cmData[k]))
	}
	configHash := hex.EncodeToString(h.Sum(nil))[:16]

	// Reconcile ConfigMap
	gatewayLabels := map[string]string{
		"app.kubernetes.io/name":       "language-operator",
		"app.kubernetes.io/managed-by": "language-operator",
		"app.kubernetes.io/component":  "gateway",
		LabelKeyLangopCluster:          cluster.Name,
		LabelKeyLangopKind:             "gateway",
	}
	if err := CreateOrUpdateConfigMap(ctx, r.Client, r.Scheme, cluster, "gateway-config", namespace, cmData); err != nil {
		return fmt.Errorf("failed to reconcile gateway ConfigMap: %w", err)
	}

	// Build volumes and mounts: ConfigMap at /etc/langop/models/ + one Secret per model API key
	volumes := []corev1.Volume{
		{
			Name: "models-config",
			VolumeSource: corev1.VolumeSource{
				ConfigMap: &corev1.ConfigMapVolumeSource{
					LocalObjectReference: corev1.LocalObjectReference{Name: "gateway-config"},
				},
			},
		},
	}
	mounts := []corev1.VolumeMount{
		{Name: "models-config", MountPath: "/etc/langop/models", ReadOnly: true},
	}
	mountedSecrets := map[string]bool{}
	for _, model := range modelList.Items {
		if model.Spec.APIKeySecretRef == nil {
			continue
		}
		secretName := model.Spec.APIKeySecretRef.Name
		if mountedSecrets[secretName] {
			continue
		}
		mountedSecrets[secretName] = true
		secretKey := model.Spec.APIKeySecretRef.Key
		if secretKey == "" {
			secretKey = "api-key"
		}
		volName := "secret-" + strings.ReplaceAll(secretName, ".", "-")
		volumes = append(volumes, corev1.Volume{
			Name: volName,
			VolumeSource: corev1.VolumeSource{
				Secret: &corev1.SecretVolumeSource{
					SecretName: secretName,
					Items: []corev1.KeyToPath{
						{Key: secretKey, Path: secretName + "/" + secretKey},
					},
				},
			},
		})
		mounts = append(mounts, corev1.VolumeMount{
			Name:      volName,
			MountPath: "/etc/secrets",
			ReadOnly:  true,
		})
	}

	// Extract the gateway DeploymentSpec once to avoid repeated nil checks below.
	var gatewayDeploy langopv1alpha1.DeploymentSpec
	if cluster.Spec.Gateway != nil {
		gatewayDeploy = cluster.Spec.Gateway.Deployment
	}

	// Append user-supplied volumes/mounts after the operator-managed ones.
	mounts = append(mounts, gatewayDeploy.VolumeMounts...)
	volumes = append(volumes, gatewayDeploy.Volumes...)

	// Determine replicas and resources from GatewaySpec
	replicas := int32(1)
	resources := corev1.ResourceRequirements{
		Requests: corev1.ResourceList{
			corev1.ResourceCPU:    resource.MustParse("100m"),
			corev1.ResourceMemory: resource.MustParse("128Mi"),
		},
		Limits: corev1.ResourceList{
			corev1.ResourceCPU:    resource.MustParse("1000m"),
			corev1.ResourceMemory: resource.MustParse("512Mi"),
		},
	}
	gatewaySvcType := corev1.ServiceTypeClusterIP
	var gatewaySvcAnnotations map[string]string
	if gatewayDeploy.Replicas != nil {
		replicas = *gatewayDeploy.Replicas
	}
	if gatewayDeploy.Resources.Requests != nil || gatewayDeploy.Resources.Limits != nil {
		resources = gatewayDeploy.Resources
	}
	if gatewayDeploy.ServiceType != "" {
		gatewaySvcType = gatewayDeploy.ServiceType
	}
	gatewaySvcAnnotations = gatewayDeploy.ServiceAnnotations

	// Merge user pod annotations with the operator-managed config-hash annotation.
	podAnnotations := map[string]string{LabelKeyLangopConfigHash: configHash}
	maps.Copy(podAnnotations, gatewayDeploy.PodAnnotations)

	// Merge user pod labels with the operator-managed gateway labels.
	podLabels := make(map[string]string, len(gatewayLabels)+len(gatewayDeploy.PodLabels))
	maps.Copy(podLabels, gatewayLabels)
	maps.Copy(podLabels, gatewayDeploy.PodLabels)

	// Probe defaults; overridden by spec when non-nil.
	livenessProbe := defaultGatewayLivenessProbe()
	readinessProbe := defaultGatewayReadinessProbe()
	if gatewayDeploy.LivenessProbe != nil {
		livenessProbe = gatewayDeploy.LivenessProbe
	}
	if gatewayDeploy.ReadinessProbe != nil {
		readinessProbe = gatewayDeploy.ReadinessProbe
	}

	// Reconcile Deployment
	deployment := &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{Name: "gateway", Namespace: namespace},
	}
	_, err := controllerutil.CreateOrUpdate(ctx, r.Client, deployment, func() error {
		if err := controllerutil.SetControllerReference(cluster, deployment, r.Scheme); err != nil {
			return err
		}
		deployment.Labels = gatewayLabels
		maxUnavailable := intstr.FromInt(0)
		maxSurge := intstr.FromInt(1)
		deployment.Spec = appsv1.DeploymentSpec{
			Replicas: &replicas,
			Selector: &metav1.LabelSelector{MatchLabels: gatewayLabels},
			Strategy: appsv1.DeploymentStrategy{
				Type: appsv1.RollingUpdateDeploymentStrategyType,
				RollingUpdate: &appsv1.RollingUpdateDeployment{
					MaxUnavailable: &maxUnavailable,
					MaxSurge:       &maxSurge,
				},
			},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels:      podLabels,
					Annotations: podAnnotations,
				},
				Spec: corev1.PodSpec{
					InitContainers: gatewayDeploy.InitContainers,
					Containers: []corev1.Container{
						{
							Name:            "gateway",
							Image:           r.gatewayImage(),
							ImagePullPolicy: r.gatewayImagePullPolicy(cluster),
							Resources:       resources,
							Env:             gatewayDeploy.Env,
							EnvFrom:         gatewayDeploy.EnvFrom,
							VolumeMounts:    mounts,
							Ports: []corev1.ContainerPort{
								{Name: "http", ContainerPort: 4000, Protocol: corev1.ProtocolTCP},
							},
							LivenessProbe:  livenessProbe,
							ReadinessProbe: readinessProbe,
							StartupProbe:   gatewayDeploy.StartupProbe,
						},
					},
					Volumes:                   volumes,
					NodeSelector:              gatewayDeploy.NodeSelector,
					Tolerations:               gatewayDeploy.Tolerations,
					TopologySpreadConstraints: gatewayDeploy.TopologySpreadConstraints,
					ImagePullSecrets:          gatewayDeploy.ImagePullSecrets,
				},
			},
		}
		podSpec := &deployment.Spec.Template.Spec
		if gatewayDeploy.Affinity != nil {
			podSpec.Affinity = gatewayDeploy.Affinity
		}
		if gatewayDeploy.SecurityContext != nil {
			podSpec.SecurityContext = gatewayDeploy.SecurityContext
		}
		if gatewayDeploy.ServiceAccountName != "" {
			podSpec.ServiceAccountName = gatewayDeploy.ServiceAccountName
		}
		return nil
	})
	if err != nil {
		return fmt.Errorf("failed to reconcile gateway Deployment: %w", err)
	}

	// Reconcile Service
	svc := &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{Name: "gateway", Namespace: namespace},
	}
	_, err = controllerutil.CreateOrUpdate(ctx, r.Client, svc, func() error {
		if err := controllerutil.SetControllerReference(cluster, svc, r.Scheme); err != nil {
			return err
		}
		svc.Labels = gatewayLabels
		svc.Spec = corev1.ServiceSpec{
			Selector: gatewayLabels,
			Type:     gatewaySvcType,
			Ports: []corev1.ServicePort{
				{
					Name:       "http",
					Protocol:   corev1.ProtocolTCP,
					Port:       8000,
					TargetPort: intstr.FromInt(4000),
				},
			},
		}
		if len(gatewaySvcAnnotations) > 0 {
			if svc.Annotations == nil {
				svc.Annotations = make(map[string]string)
			}
			for k, v := range gatewaySvcAnnotations {
				svc.Annotations[k] = v
			}
		}
		return nil
	})
	if err != nil {
		return fmt.Errorf("failed to reconcile gateway Service: %w", err)
	}

	log.Info("Reconciled shared gateway", "namespace", namespace, "models", len(modelList.Items))
	return nil
}

// reconcileGatewayIngress creates an Ingress for gateway.<cluster.domain>.
func (r *LanguageClusterReconciler) reconcileGatewayIngress(ctx context.Context, cluster *langopv1alpha1.LanguageCluster) error {
	log := log.FromContext(ctx)

	// Skip if gateway ingress explicitly disabled
	if cluster.Spec.Ingress != nil && cluster.Spec.Ingress.Enabled != nil && !*cluster.Spec.Ingress.Enabled {
		log.V(1).Info("Gateway ingress disabled, skipping")
		return nil
	}

	hostname := fmt.Sprintf("gateway.%s", cluster.Spec.Domain)
	namespace := cluster.Name

	ingressClass := ""
	if cluster.Spec.Ingress != nil {
		ingressClass = cluster.Spec.Ingress.ClassName
	}

	ingress := &networkingv1.Ingress{
		ObjectMeta: metav1.ObjectMeta{Name: "gateway", Namespace: namespace},
	}
	_, err := controllerutil.CreateOrUpdate(ctx, r.Client, ingress, func() error {
		if err := controllerutil.SetControllerReference(cluster, ingress, r.Scheme); err != nil {
			return err
		}
		pathType := networkingv1.PathTypePrefix
		ingress.Spec = networkingv1.IngressSpec{
			Rules: []networkingv1.IngressRule{
				{
					Host: hostname,
					IngressRuleValue: networkingv1.IngressRuleValue{
						HTTP: &networkingv1.HTTPIngressRuleValue{
							Paths: []networkingv1.HTTPIngressPath{
								{
									Path:     "/",
									PathType: &pathType,
									Backend: networkingv1.IngressBackend{
										Service: &networkingv1.IngressServiceBackend{
											Name: "gateway",
											Port: networkingv1.ServiceBackendPort{Number: 8000},
										},
									},
								},
							},
						},
					},
				},
			},
		}
		if cluster.Spec.Ingress != nil && cluster.Spec.Ingress.TLS != nil && (cluster.Spec.Ingress.TLS.Enabled == nil || *cluster.Spec.Ingress.TLS.Enabled) {
			secretName := cluster.Spec.Ingress.TLS.SecretName
			if secretName == "" {
				if ingress.Annotations == nil {
					ingress.Annotations = make(map[string]string)
				}
				if cluster.Spec.Ingress.TLS.IssuerRef != nil {
					kind := cluster.Spec.Ingress.TLS.IssuerRef.Kind
					if kind == "" {
						kind = "ClusterIssuer"
					}
					ingress.Annotations["cert-manager.io/"+strings.ToLower(kind)] = cluster.Spec.Ingress.TLS.IssuerRef.Name
				}
				secretName = "gateway-tls"
			}
			ingress.Spec.TLS = []networkingv1.IngressTLS{
				{Hosts: []string{hostname}, SecretName: secretName},
			}
		}
		if ingressClass != "" {
			ingress.Spec.IngressClassName = &ingressClass
		}
		return nil
	})
	if err != nil {
		return fmt.Errorf("failed to reconcile gateway Ingress: %w", err)
	}
	log.Info("Reconciled gateway Ingress", "hostname", hostname)
	return nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *LanguageClusterReconciler) SetupWithManager(mgr ctrl.Manager, concurrency int) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&langopv1alpha1.LanguageCluster{}).
		Owns(&appsv1.Deployment{}).
		Owns(&corev1.Service{}).
		Owns(&corev1.ConfigMap{}).
		Watches(&langopv1alpha1.LanguageModel{}, handler.EnqueueRequestsFromMapFunc(
			func(ctx context.Context, obj client.Object) []reconcile.Request {
				// Re-reconcile the LanguageCluster whose namespace matches the model's namespace
				return []reconcile.Request{
					{NamespacedName: types.NamespacedName{Name: obj.GetNamespace()}},
				}
			},
		)).
		WithOptions(controller.Options{MaxConcurrentReconciles: concurrency}).
		Complete(r)
}

// reconcileCapacity creates or deletes a ResourceQuota named "langop-quota" in the cluster's
// namespace based on cluster.spec.capacity. When spec.capacity is nil the quota is deleted.
func (r *LanguageClusterReconciler) reconcileCapacity(ctx context.Context, cluster *langopv1alpha1.LanguageCluster) error {
	log := log.FromContext(ctx)
	namespace := cluster.Name
	quotaName := "langop-quota"
	key := client.ObjectKey{Name: quotaName, Namespace: namespace}

	if cluster.Spec.Capacity == nil {
		// Remove quota if it exists
		existing := &corev1.ResourceQuota{}
		if err := r.Get(ctx, key, existing); err == nil {
			if err := r.Delete(ctx, existing); err != nil && !errors.IsNotFound(err) {
				return fmt.Errorf("failed to delete resourcequota: %w", err)
			}
			log.Info("Deleted ResourceQuota", "namespace", namespace, "name", quotaName)
		} else if !errors.IsNotFound(err) {
			return fmt.Errorf("failed to get resourcequota: %w", err)
		}
		return nil
	}

	// Build hard limits from spec
	hard := corev1.ResourceList{}
	cap := cluster.Spec.Capacity
	if cap.MaxAgents != nil {
		hard[corev1.ResourceName("count/languageagents.langop.io")] = *resource.NewQuantity(int64(*cap.MaxAgents), resource.DecimalSI)
	}
	if cap.MaxModels != nil {
		hard[corev1.ResourceName("count/languagemodels.langop.io")] = *resource.NewQuantity(int64(*cap.MaxModels), resource.DecimalSI)
	}
	if cap.MaxTools != nil {
		hard[corev1.ResourceName("count/languagetools.langop.io")] = *resource.NewQuantity(int64(*cap.MaxTools), resource.DecimalSI)
	}
	if cap.MaxPersonas != nil {
		hard[corev1.ResourceName("count/languagepersonas.langop.io")] = *resource.NewQuantity(int64(*cap.MaxPersonas), resource.DecimalSI)
	}
	if cap.MaxCPU != nil {
		hard[corev1.ResourceLimitsCPU] = *cap.MaxCPU
	}
	if cap.MaxMemory != nil {
		hard[corev1.ResourceLimitsMemory] = *cap.MaxMemory
	}

	quota := &corev1.ResourceQuota{
		ObjectMeta: metav1.ObjectMeta{
			Name:      quotaName,
			Namespace: namespace,
			Labels: map[string]string{
				"app.kubernetes.io/name":       "language-operator",
				"app.kubernetes.io/managed-by": "language-operator",
				"app.kubernetes.io/component":  "capacity",
				LabelKeyLangopCluster:          cluster.Name,
			},
		},
	}
	_, err := controllerutil.CreateOrUpdate(ctx, r.Client, quota, func() error {
		quota.Spec.Hard = hard
		return controllerutil.SetControllerReference(cluster, quota, r.Scheme)
	})
	if err != nil {
		return fmt.Errorf("failed to reconcile resourcequota: %w", err)
	}
	log.V(1).Info("Reconciled ResourceQuota", "namespace", namespace, "name", quotaName)
	return nil
}

// reconcileCapacityStatus populates cluster.status.capacity with observed counts and
// the sum of limits.cpu / limits.memory across all LanguageAgent specs.
// status.capacity is assigned unconditionally at the start so it is always written,
// even when all counts are zero or a list call returns an error.
func (r *LanguageClusterReconciler) reconcileCapacityStatus(ctx context.Context, cluster *langopv1alpha1.LanguageCluster) error {
	namespace := cluster.Name
	status := &langopv1alpha1.ClusterCapacityStatus{}
	// Assign immediately — ensures the field is always written to the API even on early return.
	cluster.Status.Capacity = status

	agents := &langopv1alpha1.LanguageAgentList{}
	if err := r.List(ctx, agents, client.InNamespace(namespace)); err != nil {
		return fmt.Errorf("failed to list agents: %w", err)
	}
	status.AgentCount = int32(len(agents.Items))

	models := &langopv1alpha1.LanguageModelList{}
	if err := r.List(ctx, models, client.InNamespace(namespace)); err != nil {
		return fmt.Errorf("failed to list models: %w", err)
	}
	status.ModelCount = int32(len(models.Items))

	tools := &langopv1alpha1.LanguageToolList{}
	if err := r.List(ctx, tools, client.InNamespace(namespace)); err != nil {
		return fmt.Errorf("failed to list tools: %w", err)
	}
	status.ToolCount = int32(len(tools.Items))

	personas := &langopv1alpha1.LanguagePersonaList{}
	if err := r.List(ctx, personas, client.InNamespace(namespace)); err != nil {
		return fmt.Errorf("failed to list personas: %w", err)
	}
	status.PersonaCount = int32(len(personas.Items))

	// Sum limits.cpu and limits.memory across all agent specs
	totalCPU := resource.Quantity{}
	totalMem := resource.Quantity{}
	for _, a := range agents.Items {
		if a.Spec.Deployment.Resources.Limits != nil {
			totalCPU.Add(*a.Spec.Deployment.Resources.Limits.Cpu())
			totalMem.Add(*a.Spec.Deployment.Resources.Limits.Memory())
		}
	}
	status.TotalCPULimits = totalCPU
	status.TotalMemoryLimits = totalMem
	return nil
}
