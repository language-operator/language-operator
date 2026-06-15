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
	"sort"
	"strings"
	"time"

	"github.com/go-logr/logr"
	"go.opentelemetry.io/otel/codes"
	appsv1 "k8s.io/api/apps/v1"
	autoscalingv2 "k8s.io/api/autoscaling/v2"
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
	langoplabels "github.com/language-operator/language-operator/pkg/labels"
	"github.com/language-operator/language-operator/pkg/network"
	"github.com/language-operator/language-operator/pkg/reconciler"
	"k8s.io/utils/ptr"
)

// errChildrenDraining is returned by cleanupDependentResources when child CRs still have
// finalizers. The caller should requeue rather than proceed to namespace deletion.
var errChildrenDraining = fmt.Errorf("children still draining, requeuing")

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
	DexImage                string
	OAuth2ProxyImage        string
	DefaultIngressClassName string
	DefaultTLSIssuerName    string
	DefaultTLSIssuerKind    string
	// DNSLookup replaces the live net.Resolver lookup when non-nil.
	// Used only in unit tests to inject controlled success/failure.
	DNSLookup func(ctx context.Context, host string) error
	// dnsTestDone, when non-nil, is closed by the async DNS goroutine after the
	// status update completes. Used only in unit tests to synchronise on completion.
	dnsTestDone chan struct{}
}

func (r *LanguageClusterReconciler) gatewayImage() string {
	if r.GatewayImage != "" {
		return r.GatewayImage
	}
	return "ghcr.io/language-operator/model-gateway:latest"
}

// gatewayEnv merges user-supplied env vars with operator-managed ones.
// When spec.auth.enabled, LITELLM_JWT_PUBLIC_KEY_URL is injected so that the
// model gateway enables LiteLLM JWT auth pointed at the cluster's Dex instance.
func (r *LanguageClusterReconciler) gatewayEnv(cluster *langopv1alpha1.LanguageCluster, userEnv []corev1.EnvVar) []corev1.EnvVar {
	issuer := dexIssuerURL(cluster)
	if cluster.Spec.Auth == nil || !cluster.Spec.Auth.Enabled || issuer == "" {
		return userEnv
	}
	jwtKeyURL := issuer + "/keys"
	managed := corev1.EnvVar{Name: "LITELLM_JWT_PUBLIC_KEY_URL", Value: jwtKeyURL}
	// User env takes precedence — don't override if they set it explicitly.
	for _, e := range userEnv {
		if e.Name == "LITELLM_JWT_PUBLIC_KEY_URL" {
			return userEnv
		}
	}
	return append([]corev1.EnvVar{managed}, userEnv...)
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
				Port: intstr.FromInt(network.GatewayContainerPort),
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
				Port: intstr.FromInt(network.GatewayContainerPort),
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
//+kubebuilder:rbac:groups="",resources=secrets,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=networking.k8s.io,resources=ingresses,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups="",resources=resourcequotas,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups="",resources=namespaces,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=autoscaling,resources=horizontalpodautoscalers,verbs=get;list;watch;create;update;patch;delete

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
				if err == errChildrenDraining {
					// Children still have finalizers; wait for them before deleting namespace
					return ctrl.Result{RequeueAfter: 5 * time.Second}, nil
				}
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
		r.EventManager.RecordClusterCreated(cluster)
		// Requeue after adding finalizer
		return ctrl.Result{Requeue: true}, nil
	}

	// Track the generation being reconciled so watchers can detect stale status.
	cluster.Status.ObservedGeneration = cluster.Generation

	// Mark Pending on first real reconcile so kubectl get shows something meaningful
	// before resources are fully provisioned.
	if cluster.Status.Phase == "" {
		SetPhase(&cluster.Status.Phase, &cluster.Status.ObservedGeneration, events.PhaseStatusPending, cluster.Generation)
		if updateErr := r.Status().Update(ctx, cluster); updateErr != nil {
			log.Error(updateErr, "Failed to update cluster phase to Pending")
		}
	}

	// Ensure the namespace for this cluster exists
	if err := r.reconcileNamespace(ctx, cluster); err != nil {
		log.Error(err, "Failed to reconcile namespace")
		span.RecordError(err)
		span.SetStatus(codes.Error, "Failed to reconcile namespace")
		SetPhase(&cluster.Status.Phase, &cluster.Status.ObservedGeneration, events.PhaseStatusFailed, cluster.Generation)
		SetCondition(&cluster.Status.Conditions, langopv1alpha1.ConditionReady, metav1.ConditionFalse,
			langopv1alpha1.ReasonNamespaceError, err.Error(), cluster.Generation)
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
		r.EventManager.RecordRBACFailed(cluster, err)
		SetPhase(&cluster.Status.Phase, &cluster.Status.ObservedGeneration, events.PhaseStatusFailed, cluster.Generation)
		SetCondition(&cluster.Status.Conditions, langopv1alpha1.ConditionReady, metav1.ConditionFalse,
			langopv1alpha1.ReasonRBACError, err.Error(), cluster.Generation)
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
			r.EventManager.RecordNetworkPolicyFailed(cluster, err)
			SetPhase(&cluster.Status.Phase, &cluster.Status.ObservedGeneration, events.PhaseStatusFailed, cluster.Generation)
			SetCondition(&cluster.Status.Conditions, langopv1alpha1.ConditionReady, metav1.ConditionFalse,
				langopv1alpha1.ReasonNetworkPolicyError, err.Error(), cluster.Generation)
			if updateErr := r.Status().Update(ctx, cluster); updateErr != nil {
				log.Error(updateErr, "Failed to update status after NetworkPolicy error")
			}
			reconcileErr = err
			return ctrl.Result{}, err
		} else {
			SetCondition(&cluster.Status.Conditions, langopv1alpha1.ConditionNetworkPolicyReady, metav1.ConditionTrue, langopv1alpha1.ReasonNetworkPolicyReady,
				"NetworkPolicy created successfully", cluster.Generation)
		}
	} else {
		SetCondition(&cluster.Status.Conditions, langopv1alpha1.ConditionNetworkPolicyReady, metav1.ConditionTrue, langopv1alpha1.ReasonNetworkPolicyDisabled,
			"NetworkPolicy creation disabled via networkIsolation.enabled=false", cluster.Generation)
		log.V(1).Info("Network isolation disabled - skipping NetworkPolicy creation")
	}

	// Reconcile shared LiteLLM gateway Deployment + Service + ConfigMap
	if err := r.reconcileGateway(ctx, cluster); err != nil {
		log.Error(err, "Failed to reconcile gateway")
		span.RecordError(err)
		span.SetStatus(codes.Error, "Failed to reconcile gateway")
		SetPhase(&cluster.Status.Phase, &cluster.Status.ObservedGeneration, events.PhaseStatusFailed, cluster.Generation)
		cluster.Status.GatewayReady = ptr.To(false)
		SetCondition(&cluster.Status.Conditions, langopv1alpha1.ConditionGatewayReady, metav1.ConditionFalse,
			langopv1alpha1.ReasonGatewayError, err.Error(), cluster.Generation)
		if updateErr := r.Status().Update(ctx, cluster); updateErr != nil {
			log.Error(updateErr, "Failed to update status after gateway error")
		}
		reconcileErr = err
		return ctrl.Result{}, err
	}

	// Reconcile gateway HPA (create when autoscaling configured, delete when removed)
	if err := r.reconcileGatewayHPA(ctx, cluster); err != nil {
		log.Error(err, "Failed to reconcile gateway HPA")
		span.RecordError(err)
		span.SetStatus(codes.Error, "Failed to reconcile gateway HPA")
		SetPhase(&cluster.Status.Phase, &cluster.Status.ObservedGeneration, events.PhaseStatusFailed, cluster.Generation)
		cluster.Status.GatewayReady = ptr.To(false)
		SetCondition(&cluster.Status.Conditions, langopv1alpha1.ConditionGatewayReady, metav1.ConditionFalse,
			langopv1alpha1.ReasonGatewayError, err.Error(), cluster.Generation)
		if updateErr := r.Status().Update(ctx, cluster); updateErr != nil {
			log.Error(updateErr, "Failed to update status after gateway HPA error")
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
			SetPhase(&cluster.Status.Phase, &cluster.Status.ObservedGeneration, events.PhaseStatusFailed, cluster.Generation)
			cluster.Status.GatewayReady = ptr.To(false)
			SetCondition(&cluster.Status.Conditions, langopv1alpha1.ConditionGatewayReady, metav1.ConditionFalse,
				langopv1alpha1.ReasonGatewayIngressError, err.Error(), cluster.Generation)
			if updateErr := r.Status().Update(ctx, cluster); updateErr != nil {
				log.Error(updateErr, "Failed to update status after gateway ingress error")
			}
			reconcileErr = err
			return ctrl.Result{}, err
		}
	}

	SetCondition(&cluster.Status.Conditions, langopv1alpha1.ConditionGatewayReady, metav1.ConditionTrue,
		langopv1alpha1.ReasonGatewayReady, "Shared LiteLLM gateway is ready", cluster.Generation)
	cluster.Status.GatewayEndpoint = serviceURL("gateway", cluster.Name, network.GatewayServicePort)
	cluster.Status.GatewayReady = ptr.To(true)

	// Reconcile Dex OIDC provider (created when spec.auth.enabled and no external issuer)
	if err := r.reconcileDex(ctx, cluster); err != nil {
		log.Error(err, "Failed to reconcile Dex OIDC provider")
		span.RecordError(err)
		span.SetStatus(codes.Error, "Failed to reconcile Dex OIDC provider")
		SetPhase(&cluster.Status.Phase, &cluster.Status.ObservedGeneration, events.PhaseStatusFailed, cluster.Generation)
		if updateErr := r.Status().Update(ctx, cluster); updateErr != nil {
			log.Error(updateErr, "Failed to update status after Dex error")
		}
		return ctrl.Result{}, err
	}

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
		SetPhase(&cluster.Status.Phase, &cluster.Status.ObservedGeneration, events.PhaseStatusFailed, cluster.Generation)
		SetCondition(&cluster.Status.Conditions, langopv1alpha1.ConditionCapacityReady, metav1.ConditionFalse,
			langopv1alpha1.ReasonCapacityError, err.Error(), cluster.Generation)
		if updateErr := r.Status().Update(ctx, cluster); updateErr != nil {
			log.Error(updateErr, "Failed to update status after capacity error")
		}
		reconcileErr = err
		return ctrl.Result{}, err
	}
	SetCondition(&cluster.Status.Conditions, langopv1alpha1.ConditionCapacityReady, metav1.ConditionTrue,
		langopv1alpha1.ReasonReconcileSuccess, "Capacity reconciled", cluster.Generation)

	SetPhase(&cluster.Status.Phase, &cluster.Status.ObservedGeneration, events.PhaseStatusReady, cluster.Generation)
	SetCondition(&cluster.Status.Conditions, langopv1alpha1.ConditionReady, metav1.ConditionTrue,
		langopv1alpha1.ReasonReconcileSuccess, "LanguageCluster is ready", cluster.Generation)

	cluster.Status.ManagedResources = r.buildClusterManagedResources(cluster)

	r.EventManager.RecordClusterReady(cluster)

	// Touch all member resources in the namespace so their controllers re-reconcile.
	// This handles the adoption case: resources that existed before the cluster was
	// created (or before a prior reconcile completed) are discovered and re-queued.
	r.adoptPreExistingMembers(ctx, cluster)

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

// adoptPreExistingMembers stamps langop.io/cluster-generation on every member resource
// (LanguageAgent, LanguageModel, LanguagePersona, LanguageTool) in the cluster namespace.
// The annotation change triggers each resource's controller to re-reconcile, which
// handles the case where members existed before the cluster was created. The method is
// best-effort: individual list or patch failures are logged and ignored.
func (r *LanguageClusterReconciler) adoptPreExistingMembers(ctx context.Context, cluster *langopv1alpha1.LanguageCluster) {
	log := log.FromContext(ctx)
	ns := cluster.Name
	gen := fmt.Sprintf("%d", cluster.Generation)

	touch := func(obj client.Object) {
		anns := obj.GetAnnotations()
		if anns != nil && anns[langoplabels.AnnotationKeyClusterGeneration] == gen {
			return
		}
		base := obj.DeepCopyObject().(client.Object)
		if anns == nil {
			anns = map[string]string{}
		}
		anns[langoplabels.AnnotationKeyClusterGeneration] = gen
		obj.SetAnnotations(anns)
		if err := r.Patch(ctx, obj, client.MergeFrom(base)); err != nil && !errors.IsNotFound(err) {
			log.V(1).Info("adoptPreExistingMembers: patch failed", "name", obj.GetName(), "err", err)
		}
	}

	agents := &langopv1alpha1.LanguageAgentList{}
	if err := r.List(ctx, agents, client.InNamespace(ns)); err == nil {
		for i := range agents.Items {
			touch(&agents.Items[i])
		}
	}
	models := &langopv1alpha1.LanguageModelList{}
	if err := r.List(ctx, models, client.InNamespace(ns)); err == nil {
		for i := range models.Items {
			touch(&models.Items[i])
		}
	}
	personas := &langopv1alpha1.LanguagePersonaList{}
	if err := r.List(ctx, personas, client.InNamespace(ns)); err == nil {
		for i := range personas.Items {
			touch(&personas.Items[i])
		}
	}
	tools := &langopv1alpha1.LanguageToolList{}
	if err := r.List(ctx, tools, client.InNamespace(ns)); err == nil {
		for i := range tools.Items {
			touch(&tools.Items[i])
		}
	}
}

// reconcileNamespace ensures a namespace named cluster.Name exists.
// If the namespace already exists (whether managed or unmanaged), the operator
// labels it and adopts it without setting an owner reference. A new namespace is
// created with an owner reference so it is deleted when the cluster is removed.
func (r *LanguageClusterReconciler) reconcileNamespace(ctx context.Context, cluster *langopv1alpha1.LanguageCluster) error {
	log := log.FromContext(ctx)
	ns := &corev1.Namespace{}
	err := r.Get(ctx, client.ObjectKey{Name: cluster.Name}, ns)
	if err == nil {
		// Ensure our labels are present on the existing namespace.
		if ns.Labels[langoplabels.LabelKeyLangopCluster] == cluster.Name {
			return nil // already labelled
		}
		patch := client.MergeFrom(ns.DeepCopy())
		if ns.Labels == nil {
			ns.Labels = map[string]string{}
		}
		ns.Labels[langoplabels.LabelKeyK8sManagedBy] = "language-operator"
		ns.Labels[langoplabels.LabelKeyLangopCluster] = cluster.Name
		if err := r.Patch(ctx, ns, patch); err != nil {
			return fmt.Errorf("failed to label adopted namespace %s: %w", cluster.Name, err)
		}
		log.Info("Adopted existing namespace", "namespace", cluster.Name)
		return nil
	}
	if !errors.IsNotFound(err) {
		return fmt.Errorf("failed to get namespace %s: %w", cluster.Name, err)
	}
	ns = &corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name: cluster.Name,
			Labels: map[string]string{
				langoplabels.LabelKeyK8sManagedBy:  "language-operator",
				langoplabels.LabelKeyLangopCluster: cluster.Name,
			},
		},
	}
	if err := controllerutil.SetControllerReference(cluster, ns, r.Scheme); err != nil {
		return fmt.Errorf("failed to set owner reference on namespace %s: %w", cluster.Name, err)
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

	rbacLabels := GetCommonLabels("language-operator", "LanguageCluster")
	rbacLabels[langoplabels.LabelKeyK8sComponent] = "agent-rbac"
	rbacLabels[langoplabels.LabelKeyLangopCluster] = cluster.Name

	agentsRole := &rbacv1.Role{
		ObjectMeta: metav1.ObjectMeta{Name: "agents", Namespace: namespace},
	}
	if err := CreateOrUpdateOwned(ctx, r.Client, r.Scheme, cluster, agentsRole, func() error {
		agentsRole.Labels = rbacLabels
		agentsRole.Rules = []rbacv1.PolicyRule{
			{
				APIGroups: []string{""},
				Resources: []string{"events"},
				Verbs:     []string{"create"},
			},
		}
		return nil
	}); err != nil {
		return fmt.Errorf("failed to reconcile agents Role: %w", err)
	}
	log.V(1).Info("Reconciled agents Role", "namespace", namespace)

	agentsRoleBinding := &rbacv1.RoleBinding{
		ObjectMeta: metav1.ObjectMeta{Name: "agents", Namespace: namespace},
	}
	if err := CreateOrUpdateOwned(ctx, r.Client, r.Scheme, cluster, agentsRoleBinding, func() error {
		agentsRoleBinding.Labels = rbacLabels
		agentsRoleBinding.Subjects = []rbacv1.Subject{
			{
				Kind:      "ServiceAccount",
				Name:      "default",
				Namespace: namespace,
			},
		}
		agentsRoleBinding.RoleRef = rbacv1.RoleRef{
			APIGroup: "rbac.authorization.k8s.io",
			Kind:     "Role",
			Name:     "agents",
		}
		return nil
	}); err != nil {
		return fmt.Errorf("failed to reconcile agents RoleBinding: %w", err)
	}
	log.V(1).Info("Reconciled agents RoleBinding", "namespace", namespace)

	log.V(1).Info("Successfully reconciled agent RBAC", "cluster", cluster.Name, "namespace", namespace)
	return nil
}

// reconcileGatewaySA creates/updates the ServiceAccount (and optional Role/RoleBinding) for the
// gateway pod when spec.gateway.deployment.serviceAccountName is not set. This allows users to
// attach workload-identity annotations (IRSA, GCP WI, AKS WI) and extend gateway RBAC via
// spec.gateway.deployment.serviceAccountAnnotations and spec.gateway.deployment.roleRules.
// When serviceAccountName is explicitly set the function is a no-op; the referenced SA is
// assumed to be managed externally.
func (r *LanguageClusterReconciler) reconcileGatewaySA(
	ctx context.Context,
	cluster *langopv1alpha1.LanguageCluster,
	namespace string,
	gatewayDeploy langopv1alpha1.DeploymentSpec,
) error {
	log := log.FromContext(ctx)

	// User supplied their own SA — nothing for us to manage.
	if gatewayDeploy.ServiceAccountName != "" {
		return nil
	}

	saLabels := GetCommonLabels("language-operator", "LanguageCluster")
	saLabels[langoplabels.LabelKeyK8sComponent] = "gateway-sa"
	saLabels[langoplabels.LabelKeyLangopCluster] = cluster.Name

	sa := &corev1.ServiceAccount{
		ObjectMeta: metav1.ObjectMeta{Name: GatewayResourceName, Namespace: namespace},
	}
	if err := CreateOrUpdateOwned(ctx, r.Client, r.Scheme, cluster, sa, func() error {
		sa.Labels = saLabels
		if len(gatewayDeploy.ServiceAccountAnnotations) > 0 {
			if sa.Annotations == nil {
				sa.Annotations = make(map[string]string)
			}
			maps.Copy(sa.Annotations, gatewayDeploy.ServiceAccountAnnotations)
		}
		return nil
	}); err != nil {
		return fmt.Errorf("failed to create/update gateway ServiceAccount: %w", err)
	}
	log.V(1).Info("Reconciled gateway ServiceAccount", "namespace", namespace)

	// Only create Role/RoleBinding when the user has supplied extra rules.
	if len(gatewayDeploy.RoleRules) == 0 {
		return nil
	}

	roleLabels := GetCommonLabels("language-operator", "LanguageCluster")
	roleLabels[langoplabels.LabelKeyK8sComponent] = "gateway-rbac"
	roleLabels[langoplabels.LabelKeyLangopCluster] = cluster.Name

	role := &rbacv1.Role{
		ObjectMeta: metav1.ObjectMeta{Name: GatewayResourceName, Namespace: namespace},
	}
	if err := CreateOrUpdateOwned(ctx, r.Client, r.Scheme, cluster, role, func() error {
		role.Labels = roleLabels
		role.Rules = gatewayDeploy.RoleRules
		return nil
	}); err != nil {
		return fmt.Errorf("failed to create/update gateway Role: %w", err)
	}

	rb := &rbacv1.RoleBinding{
		ObjectMeta: metav1.ObjectMeta{Name: GatewayResourceName, Namespace: namespace},
	}
	if err := CreateOrUpdateOwned(ctx, r.Client, r.Scheme, cluster, rb, func() error {
		rb.Labels = roleLabels
		rb.RoleRef = rbacv1.RoleRef{
			APIGroup: "rbac.authorization.k8s.io",
			Kind:     "Role",
			Name:     GatewayResourceName,
		}
		rb.Subjects = []rbacv1.Subject{
			{
				Kind:      "ServiceAccount",
				Name:      GatewayResourceName,
				Namespace: namespace,
			},
		}
		return nil
	}); err != nil {
		return fmt.Errorf("failed to create/update gateway RoleBinding: %w", err)
	}
	log.V(1).Info("Reconciled gateway Role and RoleBinding", "namespace", namespace)

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
							langoplabels.LabelKeyMetadataName: "default",
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
							langoplabels.LabelKeyMetadataName: "kube-system",
						},
					},
				},
			},
			Ports: []networkingv1.NetworkPolicyPort{
				{
					Protocol: &udpProtocol,
					Port:     &intstr.IntOrString{Type: intstr.Int, IntVal: network.DNSPort},
				},
				{
					Protocol: &tcpProtocol,
					Port:     &intstr.IntOrString{Type: intstr.Int, IntVal: network.DNSPort},
				},
			},
		},
	}

	// Add user-defined egress rules from spec.networkPolicies.egress
	if cluster.Spec.NetworkPolicies != nil {
		for _, rule := range cluster.Spec.NetworkPolicies.Egress {
			egressRule := networkingv1.NetworkPolicyEgressRule{}
			for _, peer := range rule.To {
				peer := peer
				kpeer := networkingv1.NetworkPolicyPeer{}
				if peer.CIDR != "" {
					kpeer.IPBlock = &networkingv1.IPBlock{CIDR: peer.CIDR}
				}
				if peer.Service != nil {
					ns := peer.Service.Namespace
					if ns == "" {
						ns = namespace
					}
					kpeer.NamespaceSelector = &metav1.LabelSelector{
						MatchLabels: map[string]string{langoplabels.LabelKeyMetadataName: ns},
					}
				}
				if peer.Group != "" {
					kpeer.PodSelector = &metav1.LabelSelector{
						MatchLabels: map[string]string{langoplabels.LabelKeyLangopGroup: peer.Group},
					}
				}
				if peer.PodSelector != nil {
					kpeer.PodSelector = peer.PodSelector
				}
				if peer.NamespaceSelector != nil {
					kpeer.NamespaceSelector = peer.NamespaceSelector
				}
				if kpeer.IPBlock != nil || kpeer.PodSelector != nil || kpeer.NamespaceSelector != nil {
					egressRule.To = append(egressRule.To, kpeer)
				}
				if len(peer.DNS) > 0 {
					resolvedCIDRs, err := resolveDNSToCIDRs(ctx, peer.DNS)
					if err == nil {
						for _, cidr := range resolvedCIDRs {
							egressRule.To = append(egressRule.To, networkingv1.NetworkPolicyPeer{
								IPBlock: &networkingv1.IPBlock{CIDR: cidr},
							})
						}
					}
				}
			}
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
			egressRules = append(egressRules, egressRule)
		}
	}

	// Wire user-defined ingress rules from spec.networkPolicies.ingress
	var ingressRules []networkingv1.NetworkPolicyIngressRule
	if cluster.Spec.NetworkPolicies != nil {
		for _, rule := range cluster.Spec.NetworkPolicies.Ingress {
			ingressRule := networkingv1.NetworkPolicyIngressRule{}
			for _, peer := range rule.From {
				peer := peer
				ingressRule.From = append(ingressRule.From, buildIngressPeerFromNetworkPeer(&peer, namespace))
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
	}

	policyTypes := []networkingv1.PolicyType{networkingv1.PolicyTypeEgress}
	if len(ingressRules) > 0 {
		policyTypes = append(policyTypes, networkingv1.PolicyTypeIngress)
	}

	netpolLabels := GetCommonLabels("language-operator", "LanguageCluster")
	netpolLabels[langoplabels.LabelKeyK8sComponent] = "agent-network-policy"
	netpolLabels[langoplabels.LabelKeyLangopCluster] = cluster.Name

	// Create NetworkPolicy
	networkPolicy := &networkingv1.NetworkPolicy{
		ObjectMeta: metav1.ObjectMeta{
			Name:      fmt.Sprintf("%s-agents", cluster.Name),
			Namespace: namespace,
			Labels:    netpolLabels,
		},
		Spec: networkingv1.NetworkPolicySpec{
			PodSelector: metav1.LabelSelector{
				MatchLabels: map[string]string{
					langoplabels.LabelKeyLangopCluster: cluster.Name,
					langoplabels.LabelKeyLangopKind:    "LanguageAgent",
				},
			},
			PolicyTypes: policyTypes,
			Egress:      egressRules,
			Ingress:     ingressRules,
		},
	}

	// Create or update the NetworkPolicy (sets owner reference internally)
	if err := CreateOrUpdateNetworkPolicy(ctx, r.Client, r.Scheme, cluster, networkPolicy); err != nil {
		return fmt.Errorf("failed to reconcile NetworkPolicy: %w", err)
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

	// Re-list all child CRs; if any still exist they are waiting for finalizer removal.
	// Do not delete the namespace until all children are gone — otherwise the namespace
	// enters Terminating with finalizer-bearing objects inside, which gets stuck.
	drainAgents := &langopv1alpha1.LanguageAgentList{}
	if err := r.List(ctx, drainAgents, client.InNamespace(namespace)); err != nil {
		return fmt.Errorf("failed to list agents during drain check: %w", err)
	}
	drainTools := &langopv1alpha1.LanguageToolList{}
	if err := r.List(ctx, drainTools, client.InNamespace(namespace)); err != nil {
		return fmt.Errorf("failed to list tools during drain check: %w", err)
	}
	drainModels := &langopv1alpha1.LanguageModelList{}
	if err := r.List(ctx, drainModels, client.InNamespace(namespace)); err != nil {
		return fmt.Errorf("failed to list models during drain check: %w", err)
	}
	drainPersonas := &langopv1alpha1.LanguagePersonaList{}
	if err := r.List(ctx, drainPersonas, client.InNamespace(namespace)); err != nil {
		return fmt.Errorf("failed to list personas during drain check: %w", err)
	}
	remaining := len(drainAgents.Items) + len(drainTools.Items) + len(drainModels.Items) + len(drainPersonas.Items)
	if remaining > 0 {
		log.Info("Waiting for child CRs to finish finalizing before deleting namespace", "remaining", remaining, "cluster", clusterName)
		return errChildrenDraining
	}

	// Delete the namespace only if the operator created it (owner reference present).
	// Adopted namespaces have no owner reference and must not be deleted.
	ns := &corev1.Namespace{}
	if err := r.Get(ctx, client.ObjectKey{Name: clusterName}, ns); err == nil {
		ownsNamespace := false
		for _, ref := range ns.OwnerReferences {
			if ref.Kind == "LanguageCluster" && ref.Name == clusterName {
				ownsNamespace = true
				break
			}
		}
		if ownsNamespace {
			log.Info("Deleting namespace", "namespace", clusterName)
			if err := r.Delete(ctx, ns); err != nil && !errors.IsNotFound(err) {
				return fmt.Errorf("failed to delete namespace %s: %w", clusterName, err)
			}
		} else {
			log.Info("Skipping namespace deletion — namespace was adopted, not created by operator", "namespace", clusterName)
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

	// Skip DNS lookup if the condition already reflects the current generation.
	// This prevents blocking the reconcile worker on every loop for clusters
	// whose domain has not changed.
	for _, cond := range cluster.Status.Conditions {
		if cond.Type == langopv1alpha1.ConditionDNSConfigured &&
			cond.ObservedGeneration == cluster.Generation {
			log.V(1).Info("DNS condition up-to-date for current generation, skipping lookup",
				"domain", domain, "generation", cluster.Generation)
			return
		}
	}

	// DNS condition is stale — run lookup asynchronously to avoid blocking the reconcile worker.
	clusterKey := types.NamespacedName{Name: cluster.Name}
	clusterGen := cluster.Generation
	log.V(1).Info("Starting async DNS validation", "domain", domain, "generation", clusterGen)

	go func() {
		dnsCtx, cancel := context.WithTimeout(context.Background(), 12*time.Second)
		defer cancel()
		if r.dnsTestDone != nil {
			defer close(r.dnsTestDone)
		}

		testHost := fmt.Sprintf("test-validation.%s", domain)
		var err error
		if r.DNSLookup != nil {
			err = r.DNSLookup(dnsCtx, testHost)
		} else {
			resolver := &net.Resolver{}
			_, err = resolver.LookupHost(dnsCtx, testHost)
		}

		var fresh langopv1alpha1.LanguageCluster
		if getErr := r.Client.Get(dnsCtx, clusterKey, &fresh); getErr != nil {
			log.V(1).Info("Cluster not found during async DNS update, skipping")
			return
		}
		if fresh.Generation != clusterGen {
			log.V(1).Info("Cluster generation changed during async DNS validation, discarding result")
			return
		}

		if err != nil {
			log.V(1).Info("Wildcard DNS not configured or not accessible",
				"domain", domain, "test_host", testHost, "error", err.Error())
			SetCondition(&fresh.Status.Conditions, langopv1alpha1.ConditionDNSConfigured, metav1.ConditionFalse,
				langopv1alpha1.ReasonWildcardDNSMissing,
				fmt.Sprintf("Wildcard DNS (*.%s) not configured or not accessible. See docs/dns.md for setup instructions.", domain),
				clusterGen)
			log.Info("DNS configuration notice",
				"domain", domain,
				"required_dns", fmt.Sprintf("*.%s", domain),
				"documentation", "See docs/dns.md for DNS setup instructions")
		} else {
			log.V(1).Info("Wildcard DNS configured correctly", "domain", domain)
			SetCondition(&fresh.Status.Conditions, langopv1alpha1.ConditionDNSConfigured, metav1.ConditionTrue,
				langopv1alpha1.ReasonWildcardDNSReady,
				fmt.Sprintf("Wildcard DNS (*.%s) is correctly configured", domain),
				clusterGen)
		}

		if updateErr := r.Status().Update(dnsCtx, &fresh); updateErr != nil {
			log.Error(updateErr, "Failed to update DNS condition asynchronously")
		}
	}()
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
	sort.Strings(keys)
	for _, k := range keys {
		h.Write([]byte(k))
		h.Write([]byte(cmData[k]))
	}
	configHash := hex.EncodeToString(h.Sum(nil))[:16]

	// Reconcile ConfigMap
	gatewayLabels := GetCommonLabels("language-operator", GatewayResourceName)
	gatewayLabels[langoplabels.LabelKeyK8sComponent] = GatewayResourceName
	gatewayLabels[langoplabels.LabelKeyLangopCluster] = cluster.Name
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
						{Key: secretKey, Path: secretKey},
					},
				},
			},
		})
		mounts = append(mounts, corev1.VolumeMount{
			Name:      volName,
			MountPath: "/etc/secrets/" + secretName,
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
	if gatewayDeploy.Replicas != nil {
		replicas = *gatewayDeploy.Replicas
	}
	// When HPA is active, preserve the current replica count rather than overwriting it.
	// The HPA controller owns spec.replicas; resetting it every reconcile prevents scaling.
	deploymentReplicas := &replicas
	if gatewayDeploy.Autoscaling != nil {
		existing := &appsv1.Deployment{}
		if err := r.Get(ctx, types.NamespacedName{Name: GatewayResourceName, Namespace: namespace}, existing); err == nil && existing.Spec.Replicas != nil {
			deploymentReplicas = existing.Spec.Replicas
		}
	}
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
	if gatewayDeploy.Resources.Requests != nil || gatewayDeploy.Resources.Limits != nil {
		resources = gatewayDeploy.Resources
	}
	if gatewayDeploy.ServiceType != "" {
		gatewaySvcType = gatewayDeploy.ServiceType
	}
	gatewaySvcAnnotations = gatewayDeploy.ServiceAnnotations

	// Merge user pod annotations with the operator-managed config-hash annotation.
	podAnnotations := map[string]string{langoplabels.LabelKeyLangopConfigHash: configHash}
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

	// Reconcile gateway ServiceAccount (and optional Role/RoleBinding)
	if err := r.reconcileGatewaySA(ctx, cluster, namespace, gatewayDeploy); err != nil {
		return fmt.Errorf("failed to reconcile gateway ServiceAccount: %w", err)
	}

	// Reconcile Deployment
	deployment := &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{Name: GatewayResourceName, Namespace: namespace},
	}
	err := CreateOrUpdateOwned(ctx, r.Client, r.Scheme, cluster, deployment, func() error {
		deployment.Labels = gatewayLabels
		maxUnavailable := intstr.FromInt(0)
		maxSurge := intstr.FromInt(1)
		deployment.Spec = appsv1.DeploymentSpec{
			Replicas: deploymentReplicas,
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
							Name:            GatewayResourceName,
							Image:           r.gatewayImage(),
							Command:         gatewayDeploy.Command,
							Args:            gatewayDeploy.Args,
							ImagePullPolicy: r.gatewayImagePullPolicy(cluster),
							Resources:       resources,
							Env:             r.gatewayEnv(cluster, gatewayDeploy.Env),
							EnvFrom:         gatewayDeploy.EnvFrom,
							VolumeMounts:    mounts,
							Ports: []corev1.ContainerPort{
								{Name: "http", ContainerPort: network.GatewayContainerPort, Protocol: corev1.ProtocolTCP},
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
		} else {
			podSpec.ServiceAccountName = GatewayResourceName
		}
		return nil
	})
	if err != nil {
		return fmt.Errorf("failed to reconcile gateway Deployment: %w", err)
	}

	// Reconcile Service
	svc := &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{Name: GatewayResourceName, Namespace: namespace},
	}
	err = CreateOrUpdateOwned(ctx, r.Client, r.Scheme, cluster, svc, func() error {
		svc.Labels = gatewayLabels
		svc.Spec = corev1.ServiceSpec{
			Selector: gatewayLabels,
			Type:     gatewaySvcType,
			Ports: []corev1.ServicePort{
				{
					Name:       "http",
					Protocol:   corev1.ProtocolTCP,
					Port:       network.GatewayServicePort,
					TargetPort: intstr.FromInt(network.GatewayContainerPort),
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

// reconcileGatewayHPA creates a HorizontalPodAutoscaler for the gateway when autoscaling is
// configured, and deletes any existing HPA when autoscaling is removed from the spec.
func (r *LanguageClusterReconciler) reconcileGatewayHPA(ctx context.Context, cluster *langopv1alpha1.LanguageCluster) error {
	var gatewayDeploy langopv1alpha1.DeploymentSpec
	if cluster.Spec.Gateway != nil {
		gatewayDeploy = cluster.Spec.Gateway.Deployment
	}

	hpa := &autoscalingv2.HorizontalPodAutoscaler{
		ObjectMeta: metav1.ObjectMeta{
			Name:      GatewayResourceName,
			Namespace: cluster.Name,
		},
	}

	if gatewayDeploy.Autoscaling == nil {
		if err := r.Delete(ctx, hpa); err != nil && !errors.IsNotFound(err) {
			return fmt.Errorf("failed to delete gateway HorizontalPodAutoscaler: %w", err)
		}
		return nil
	}

	as := gatewayDeploy.Autoscaling
	labels := GetCommonLabels("language-operator", GatewayResourceName)
	labels[langoplabels.LabelKeyK8sComponent] = GatewayResourceName
	labels[langoplabels.LabelKeyLangopCluster] = cluster.Name

	metrics := as.Metrics
	if len(metrics) == 0 {
		// Default: 80% average CPU utilization
		avgUtil := int32(80)
		metrics = []autoscalingv2.MetricSpec{
			{
				Type: autoscalingv2.ResourceMetricSourceType,
				Resource: &autoscalingv2.ResourceMetricSource{
					Name: corev1.ResourceCPU,
					Target: autoscalingv2.MetricTarget{
						Type:               autoscalingv2.UtilizationMetricType,
						AverageUtilization: &avgUtil,
					},
				},
			},
		}
	}

	err := CreateOrUpdateOwned(ctx, r.Client, r.Scheme, cluster, hpa, func() error {
		hpa.Labels = labels
		hpa.Spec = autoscalingv2.HorizontalPodAutoscalerSpec{
			ScaleTargetRef: autoscalingv2.CrossVersionObjectReference{
				APIVersion: "apps/v1",
				Kind:       "Deployment",
				Name:       GatewayResourceName,
			},
			MinReplicas: as.MinReplicas,
			MaxReplicas: as.MaxReplicas,
			Metrics:     metrics,
		}
		return nil
	})
	return err
}

// reconcileGatewayIngress creates an Ingress for gateway.<cluster.domain>.
func (r *LanguageClusterReconciler) reconcileGatewayIngress(ctx context.Context, cluster *langopv1alpha1.LanguageCluster) error {
	log := log.FromContext(ctx)

	// Gateway ingress is opt-in. Create it only when explicitly enabled; otherwise
	// make sure any previously-created ingress is removed (upgrade or runtime toggle-off).
	enabled := cluster.Spec.Ingress != nil && cluster.Spec.Ingress.Enabled != nil && *cluster.Spec.Ingress.Enabled
	if !enabled {
		log.V(1).Info("Gateway ingress not enabled, ensuring absent")
		return r.deleteGatewayIngress(ctx, cluster)
	}

	hostname := cluster.Spec.Domain
	namespace := cluster.Name

	ingressClass := r.DefaultIngressClassName
	if cluster.Spec.Ingress != nil && cluster.Spec.Ingress.ClassName != "" {
		ingressClass = cluster.Spec.Ingress.ClassName
	}

	ingress := &networkingv1.Ingress{
		ObjectMeta: metav1.ObjectMeta{Name: GatewayResourceName, Namespace: namespace},
	}
	err := CreateOrUpdateOwned(ctx, r.Client, r.Scheme, cluster, ingress, func() error {
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
											Name: GatewayResourceName,
											Port: networkingv1.ServiceBackendPort{Number: network.GatewayServicePort},
										},
									},
								},
							},
						},
					},
				},
			},
		}
		tlsEnabled := r.DefaultTLSIssuerName != ""
		if cluster.Spec.Ingress != nil && cluster.Spec.Ingress.TLS != nil {
			if cluster.Spec.Ingress.TLS.Enabled != nil {
				tlsEnabled = *cluster.Spec.Ingress.TLS.Enabled
			} else {
				tlsEnabled = true
			}
		}
		if tlsEnabled {
			secretName := ""
			if cluster.Spec.Ingress != nil && cluster.Spec.Ingress.TLS != nil {
				secretName = cluster.Spec.Ingress.TLS.SecretName
			}
			if secretName == "" {
				if r.DefaultTLSIssuerName != "" {
					if ingress.Annotations == nil {
						ingress.Annotations = make(map[string]string)
					}
					kind := r.DefaultTLSIssuerKind
					if kind == "" {
						kind = "ClusterIssuer"
					}
					ingress.Annotations["cert-manager.io/"+certManagerIssuerAnnotationSuffix(kind)] = r.DefaultTLSIssuerName
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

// deleteGatewayIngress removes the gateway Ingress if it exists. Used when the gateway
// ingress is not enabled, to clean up an ingress left over from a prior reconcile.
func (r *LanguageClusterReconciler) deleteGatewayIngress(ctx context.Context, cluster *langopv1alpha1.LanguageCluster) error {
	ingress := &networkingv1.Ingress{
		ObjectMeta: metav1.ObjectMeta{Name: GatewayResourceName, Namespace: cluster.Name},
	}
	if err := r.Delete(ctx, ingress); err != nil && !errors.IsNotFound(err) {
		return fmt.Errorf("failed to delete gateway Ingress: %w", err)
	}
	return nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *LanguageClusterReconciler) SetupWithManager(mgr ctrl.Manager, concurrency int) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&langopv1alpha1.LanguageCluster{}).
		Owns(&appsv1.Deployment{}).
		Owns(&corev1.Service{}).
		Owns(&corev1.ConfigMap{}).
		Owns(&networkingv1.NetworkPolicy{}).
		Owns(&networkingv1.Ingress{}).
		Owns(&corev1.ResourceQuota{}).
		Watches(&langopv1alpha1.LanguageModel{}, handler.EnqueueRequestsFromMapFunc(
			func(ctx context.Context, obj client.Object) []reconcile.Request {
				// Re-reconcile the LanguageCluster whose namespace matches the model's namespace
				return []reconcile.Request{
					{NamespacedName: types.NamespacedName{Name: obj.GetNamespace()}},
				}
			},
		)).
		Watches(&langopv1alpha1.LanguageAgent{}, handler.EnqueueRequestsFromMapFunc(
			func(ctx context.Context, obj client.Object) []reconcile.Request {
				// Re-reconcile when agents change so Dex redirect URIs stay current
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

	quotaLabels := GetCommonLabels("language-operator", "LanguageCluster")
	quotaLabels[langoplabels.LabelKeyK8sComponent] = "capacity"
	quotaLabels[langoplabels.LabelKeyLangopCluster] = cluster.Name

	quota := &corev1.ResourceQuota{
		ObjectMeta: metav1.ObjectMeta{
			Name:      quotaName,
			Namespace: namespace,
			Labels:    quotaLabels,
		},
	}
	err := CreateOrUpdateOwned(ctx, r.Client, r.Scheme, cluster, quota, func() error {
		quota.Spec.Hard = hard
		return nil
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

// buildClusterManagedResources returns the inventory of Kubernetes resources managed by this
// controller on behalf of cluster.
func (r *LanguageClusterReconciler) buildClusterManagedResources(
	cluster *langopv1alpha1.LanguageCluster,
) []langopv1alpha1.ManagedResource {
	ns := cluster.Name
	resources := []langopv1alpha1.ManagedResource{
		// Namespace is cluster-scoped: Namespace field intentionally empty.
		{Kind: "Namespace", Name: cluster.Name},
		{Group: "rbac.authorization.k8s.io", Kind: "Role", Name: "agents", Namespace: ns},
		{Group: "rbac.authorization.k8s.io", Kind: "RoleBinding", Name: "agents", Namespace: ns},
	}

	if r.NetworkIsolationEnabled {
		resources = append(resources, langopv1alpha1.ManagedResource{
			Group: "networking.k8s.io", Kind: "NetworkPolicy",
			Name: fmt.Sprintf("%s-agents", cluster.Name), Namespace: ns,
		})
	}

	resources = append(resources,
		langopv1alpha1.ManagedResource{Kind: "ConfigMap", Name: "gateway-config", Namespace: ns},
		langopv1alpha1.ManagedResource{Group: "apps", Kind: "Deployment", Name: GatewayResourceName, Namespace: ns},
		langopv1alpha1.ManagedResource{Kind: "Service", Name: GatewayResourceName, Namespace: ns},
	)

	// Managed gateway SA — always created when serviceAccountName is not user-supplied.
	if cluster.Spec.Gateway == nil || cluster.Spec.Gateway.Deployment.ServiceAccountName == "" {
		resources = append(resources,
			langopv1alpha1.ManagedResource{Kind: "ServiceAccount", Name: GatewayResourceName, Namespace: ns},
		)
		if cluster.Spec.Gateway != nil && len(cluster.Spec.Gateway.Deployment.RoleRules) > 0 {
			resources = append(resources,
				langopv1alpha1.ManagedResource{Group: "rbac.authorization.k8s.io", Kind: "Role", Name: GatewayResourceName, Namespace: ns},
				langopv1alpha1.ManagedResource{Group: "rbac.authorization.k8s.io", Kind: "RoleBinding", Name: GatewayResourceName, Namespace: ns},
			)
		}
	}

	// Ingress is opt-in: created only when a domain is configured and explicitly enabled.
	if cluster.Spec.Domain != "" {
		ingressEnabled := cluster.Spec.Ingress != nil && cluster.Spec.Ingress.Enabled != nil && *cluster.Spec.Ingress.Enabled
		if ingressEnabled {
			resources = append(resources, langopv1alpha1.ManagedResource{
				Group: "networking.k8s.io", Kind: "Ingress", Name: GatewayResourceName, Namespace: ns,
			})
		}
	}

	// HPA is created when gateway autoscaling is configured.
	if cluster.Spec.Gateway != nil && cluster.Spec.Gateway.Deployment.Autoscaling != nil {
		resources = append(resources, langopv1alpha1.ManagedResource{
			Group: "autoscaling", Kind: "HorizontalPodAutoscaler", Name: GatewayResourceName, Namespace: ns,
		})
	}

	// ResourceQuota is created when capacity limits are configured.
	if cluster.Spec.Capacity != nil {
		resources = append(resources, langopv1alpha1.ManagedResource{
			Kind: "ResourceQuota", Name: "langop-quota", Namespace: ns,
		})
	}

	return resources
}
