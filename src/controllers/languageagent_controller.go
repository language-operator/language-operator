package controllers

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"maps"
	"os"
	"strings"
	"time"

	"github.com/go-logr/logr"
	"github.com/google/uuid"
	"go.opentelemetry.io/otel/codes"
	appsv1 "k8s.io/api/apps/v1"
	autoscalingv2 "k8s.io/api/autoscaling/v2"
	corev1 "k8s.io/api/core/v1"
	networkingv1 "k8s.io/api/networking/v1"
	policyv1 "k8s.io/api/policy/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/meta"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/client-go/tools/record"
	"k8s.io/utils/ptr"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	"sigs.k8s.io/yaml"

	langopv1alpha1 "github.com/language-operator/language-operator/api/v1alpha1"
	"github.com/language-operator/language-operator/pkg/cni"
	"github.com/language-operator/language-operator/pkg/events"
	"github.com/language-operator/language-operator/pkg/reconciler"
	"github.com/language-operator/language-operator/pkg/validation"
)

// RegistryManager interface for registry configuration management
type RegistryManager interface {
	GetRegistries() []string
}

// LanguageAgentReconciler reconciles a LanguageAgent object
type LanguageAgentReconciler struct {
	client.Client
	Scheme                     *runtime.Scheme
	Log                        logr.Logger
	Recorder                   record.EventRecorder
	EventManager               *events.EventManager
	RegistryManager            RegistryManager
	NetworkPolicyTimeout       time.Duration
	NetworkPolicyRetries       int
	NetworkIsolationEnabled    bool
	DefaultIngressClassName    string
	DefaultStorageClassName    string
	DefaultTLSIssuerName       string
	DefaultTLSIssuerKind       string
	IngressControllerNamespace string
	CNICapabilities            *cni.CNICapabilities
}

// agentConfigYAML is the structure marshaled into /etc/agent/config.yaml.
// sigs.k8s.io/yaml marshals via JSON, so json tags control the output key names.
type agentConfigYAML struct {
	Agent        agentIdentityYAML          `json:"agent"`
	Instructions string                     `json:"instructions,omitempty"`
	Personas     []personaConfigYAML        `json:"personas,omitempty"`
	Tools        map[string]toolConfigYAML  `json:"tools,omitempty"`
	Models       map[string]modelConfigYAML `json:"models,omitempty"`
}

type agentIdentityYAML struct {
	Name      string `json:"name"`
	Namespace string `json:"namespace"`
}

type personaConfigYAML struct {
	Name        string `json:"name"`
	Tone        string `json:"tone,omitempty"`
	Personality string `json:"personality,omitempty"`
	Expertise   string `json:"expertise,omitempty"`
}

type toolConfigYAML struct {
	Endpoint string `json:"endpoint"`
	Protocol string `json:"protocol"`
}

type modelConfigYAML struct {
	Role     string `json:"role,omitempty"`
	Provider string `json:"provider"`
	Model    string `json:"model"`
	Endpoint string `json:"endpoint"`
	Priority *int32 `json:"priority,omitempty"`
}

//+kubebuilder:rbac:groups=langop.io,resources=languageagents,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=langop.io,resources=languageagents/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=langop.io,resources=languageagents/finalizers,verbs=update
//+kubebuilder:rbac:groups=langop.io,resources=languagepersonas,verbs=get;list;watch
//+kubebuilder:rbac:groups=langop.io,resources=languageclusters,verbs=get;list;watch
//+kubebuilder:rbac:groups=apps,resources=deployments,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=apps,resources=daemonsets,verbs=get;list;watch
//+kubebuilder:rbac:groups="",resources=configmaps,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups="",resources=persistentvolumeclaims,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups="",resources=services,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups="",resources=pods,verbs=get;list;watch
//+kubebuilder:rbac:groups="",resources=pods/log,verbs=get
//+kubebuilder:rbac:groups=networking.k8s.io,resources=networkpolicies,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=networking.k8s.io,resources=ingresses,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=networking.k8s.io,resources=ingressclasses,verbs=get;list;watch
//+kubebuilder:rbac:groups="",resources=serviceaccounts,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=rbac.authorization.k8s.io,resources=roles,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=rbac.authorization.k8s.io,resources=rolebindings,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=rbac.authorization.k8s.io,resources=clusterrolebindings,verbs=get;delete
//+kubebuilder:rbac:groups=langop.io,resources=languagetools,verbs=get;list;watch
//+kubebuilder:rbac:groups=langop.io,resources=languagemodels,verbs=get;list;watch
//+kubebuilder:rbac:groups=langop.io,resources=languageagentruntimes,verbs=get;list;watch
//+kubebuilder:rbac:groups="",resources=secrets,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=policy,resources=poddisruptionbudgets,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=autoscaling,resources=horizontalpodautoscalers,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=monitoring.coreos.com,resources=servicemonitors,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=monitoring.coreos.com,resources=prometheusrules,verbs=get;list;watch;create;update;patch;delete

// Reconcile is part of the main kubernetes reconciliation loop
func (r *LanguageAgentReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	// Use the reconciler helper for common setup
	helper := &reconciler.ReconcileHelper[*langopv1alpha1.LanguageAgent]{
		Client:       r.Client,
		TracerName:   "language-operator/agent-controller",
		ResourceType: "agent",
	}

	agent := &langopv1alpha1.LanguageAgent{}
	result, err := helper.StartReconcile(ctx, req, agent)
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

	// Write status exactly once on error exit paths — prevents phase flicker and
	// ensures ObservedGeneration is always set, even on early returns.
	defer func() {
		if reconcileErr == nil || !agent.DeletionTimestamp.IsZero() {
			return
		}
		agent.Status.ObservedGeneration = agent.Generation
		if updateErr := r.Status().Update(ctx, agent); updateErr != nil && !apierrors.IsNotFound(updateErr) {
			log.Error(updateErr, "Failed to update LanguageAgent status")
		}
	}()

	// Handle deletion
	if !agent.DeletionTimestamp.IsZero() {
		span.AddEvent("Deleting agent")
		if controllerutil.ContainsFinalizer(agent, FinalizerName) {
			// Per-agent RBAC resources (ServiceAccount, Role, RoleBinding named
			// "language-agent-<agent>") have no owner reference and are not GC'd
			// automatically; clean them up explicitly on deletion.
			if err := r.cleanupResources(ctx, agent); err != nil {
				span.RecordError(err)
				span.SetStatus(codes.Error, "Failed to clean up agent resources")
				reconcileErr = err
				return ctrl.Result{}, err
			}
			controllerutil.RemoveFinalizer(agent, FinalizerName)
			if err := r.Update(ctx, agent); err != nil {
				span.RecordError(err)
				span.SetStatus(codes.Error, "Failed to remove finalizer")
				reconcileErr = err
				return ctrl.Result{}, err
			}
		}
		span.SetStatus(codes.Ok, "Agent deleted successfully")
		return ctrl.Result{}, nil
	}

	// Add finalizer if not present
	if !controllerutil.ContainsFinalizer(agent, FinalizerName) {
		controllerutil.AddFinalizer(agent, FinalizerName)
		if err := r.Update(ctx, agent); err != nil {
			span.RecordError(err)
			span.SetStatus(codes.Error, "Failed to add finalizer")
			reconcileErr = err
			return ctrl.Result{}, err
		}
	}

	// Resolve LanguageAgentRuntime if referenced, and build an effective spec for this
	// reconcile cycle. We work on a local copy so the stored agent object is never
	// mutated with runtime-derived values (controller-time resolution, not admission-time).
	workingAgent := agent
	if agent.Spec.Runtime != "" {
		rt := &langopv1alpha1.LanguageAgentRuntime{}
		if err := r.Get(ctx, types.NamespacedName{Name: agent.Spec.Runtime}, rt); err != nil {
			if apierrors.IsNotFound(err) {
				SetCondition(&agent.Status.Conditions, langopv1alpha1.ConditionRuntimeResolved,
					metav1.ConditionFalse, langopv1alpha1.ReasonRuntimeNotFound,
					fmt.Sprintf("LanguageAgentRuntime %q not found", agent.Spec.Runtime), agent.Generation)
			}
			agent.Status.Phase = events.PhaseStatusFailed
			reconcileErr = err
			return ctrl.Result{}, err
		}
		effectiveSpec := agent.Spec.DeepCopy()
		langopv1alpha1.ApplyRuntimeDefaults(effectiveSpec, &rt.Spec)
		workingAgent = agent.DeepCopy()
		workingAgent.Spec = *effectiveSpec
		SetCondition(&agent.Status.Conditions, langopv1alpha1.ConditionRuntimeResolved,
			metav1.ConditionTrue, langopv1alpha1.ReasonRuntimeApplied,
			fmt.Sprintf("Runtime %q applied", agent.Spec.Runtime), agent.Generation)
	}

	// Reconcile runtime-specific credential Secret (opencode, openclaw).
	// Must run after runtime resolution so workingAgent has the merged spec.
	if err := r.reconcileRuntimeSecret(ctx, agent, workingAgent); err != nil {
		log.Error(err, "Failed to reconcile runtime credential secret")
		reconcileErr = err
		return ctrl.Result{}, err
	}

	// Validate image registry against whitelist
	if err := r.validateImageRegistry(workingAgent); err != nil {
		log.Error(err, "Image registry validation failed", "image", agent.Spec.Image)
		span.RecordError(err)
		span.SetStatus(codes.Error, "Image registry validation failed")
		SetCondition(&agent.Status.Conditions, langopv1alpha1.ConditionRegistryValidated, metav1.ConditionFalse, langopv1alpha1.ReasonRegistryNotAllowed, err.Error(), agent.Generation)
		if r.EventManager != nil {
			r.EventManager.RecordRegistryValidationFailed(agent, agent.Spec.Image)
		}
		agent.Status.Phase = events.PhaseStatusFailed
		reconcileErr = err
		return ctrl.Result{}, err
	}
	SetCondition(&agent.Status.Conditions, langopv1alpha1.ConditionRegistryValidated, metav1.ConditionTrue, langopv1alpha1.ReasonValidated, "Image registry is in whitelist", agent.Generation)

	// Reconcile ConfigMap
	configHash, err := r.reconcileConfigMap(ctx, workingAgent)
	if err != nil {
		log.Error(err, "Failed to reconcile ConfigMap")
		span.RecordError(err)
		span.SetStatus(codes.Error, "ConfigMap reconciliation failed")
		SetCondition(&agent.Status.Conditions, langopv1alpha1.ConditionReady, metav1.ConditionFalse, langopv1alpha1.ReasonConfigMapError, err.Error(), agent.Generation)
		agent.Status.Phase = events.PhaseStatusFailed
		reconcileErr = err
		return ctrl.Result{}, err
	}

	// Reconcile PVC for workspace if enabled
	if err := r.reconcilePVC(ctx, workingAgent); err != nil {
		log.Error(err, "Failed to reconcile PVC")
		span.RecordError(err)
		span.SetStatus(codes.Error, "PVC reconciliation failed")
		SetCondition(&agent.Status.Conditions, langopv1alpha1.ConditionReady, metav1.ConditionFalse, langopv1alpha1.ReasonPVCError, err.Error(), agent.Generation)
		agent.Status.Phase = events.PhaseStatusFailed
		reconcileErr = err
		return ctrl.Result{}, err
	}

	// Reconcile workspace seed ConfigMap (holds InitialFiles for seed-once init container)
	if err := r.reconcileWorkspaceSeedConfigMap(ctx, workingAgent); err != nil {
		log.Error(err, "Failed to reconcile workspace seed ConfigMap")
		span.RecordError(err)
		span.SetStatus(codes.Error, "Workspace seed ConfigMap reconciliation failed")
		SetCondition(&agent.Status.Conditions, langopv1alpha1.ConditionWorkspaceSeeded, metav1.ConditionFalse, langopv1alpha1.ReasonWorkspaceSeedError, err.Error(), agent.Generation)
		agent.Status.Phase = events.PhaseStatusFailed
		reconcileErr = err
		return ctrl.Result{}, err
	}
	if workspaceSeedEnabled(workingAgent) {
		SetCondition(&agent.Status.Conditions, langopv1alpha1.ConditionWorkspaceSeeded, metav1.ConditionTrue, langopv1alpha1.ReasonWorkspaceSeedReady, "Workspace seed ConfigMap reconciled", agent.Generation)
	}

	// Reconcile NetworkPolicy for network isolation (if enabled)
	if r.NetworkIsolationEnabled {
		if err := r.reconcileNetworkPolicy(ctx, workingAgent); err != nil {
			log.Error(err, "Failed to reconcile NetworkPolicy")
			span.RecordError(err)

			// Determine if this is a timeout error vs other error
			isTimeout := strings.Contains(err.Error(), "context deadline exceeded") ||
				strings.Contains(err.Error(), "timeout")

			if isTimeout {
				// For timeout errors, set a specific condition but continue reconciliation
				SetCondition(&agent.Status.Conditions, langopv1alpha1.ConditionNetworkPolicyReady, metav1.ConditionFalse, langopv1alpha1.ReasonNetworkPolicyTimeout,
					fmt.Sprintf("NetworkPolicy creation timed out after %v with %d retries. This may indicate slow CNI response. The operator will continue to retry. Error: %v",
						r.NetworkPolicyTimeout, r.NetworkPolicyRetries, err), agent.Generation)

				log.Info("NetworkPolicy timeout detected - continuing reconciliation with degraded network isolation",
					"timeout", r.NetworkPolicyTimeout,
					"retries", r.NetworkPolicyRetries,
					"error", err.Error())

				// Record warning event
				if r.EventManager != nil {
					r.EventManager.RecordNetworkPolicyTimeout(agent)
				}

				// Don't fail the entire reconciliation for timeout - continue with degraded state
			} else {
				// For non-timeout errors, fail the reconciliation
				span.SetStatus(codes.Error, "NetworkPolicy reconciliation failed")
				SetCondition(&agent.Status.Conditions, langopv1alpha1.ConditionReady, metav1.ConditionFalse, langopv1alpha1.ReasonNetworkPolicyError, err.Error(), agent.Generation)
				agent.Status.Phase = events.PhaseStatusFailed
				reconcileErr = err
				return ctrl.Result{}, err
			}
		} else {
			// NetworkPolicy succeeded
			SetCondition(&agent.Status.Conditions, langopv1alpha1.ConditionNetworkPolicyReady, metav1.ConditionTrue, langopv1alpha1.ReasonNetworkPolicyReady,
				"NetworkPolicy created successfully", agent.Generation)
		}

		// Use startup-cached CNI capabilities to determine NetworkPolicy enforcement support.
		cniName := "unknown"
		cniSupported := false
		if r.CNICapabilities != nil {
			cniName = r.CNICapabilities.Name
			cniSupported = r.CNICapabilities.SupportsNetworkPolicy
		}
		if !cniSupported {
			message := fmt.Sprintf("NetworkPolicy created but may not be enforced. CNI plugin '%s' does not support NetworkPolicy. Consider installing Cilium, Calico, Weave Net, or Antrea for network isolation.", cniName)
			SetCondition(&agent.Status.Conditions, langopv1alpha1.ConditionNetworkPolicyEnforced, metav1.ConditionFalse, langopv1alpha1.ReasonCNINotSupported, message, agent.Generation)
			if r.Recorder != nil {
				r.EventManager.RecordNetworkPolicyUnsupported(agent, cniName)
			}
			log.Info("NetworkPolicy enforcement not supported", "cni", cniName)
		} else {
			message := fmt.Sprintf("NetworkPolicy enforcement active (CNI: %s)", cniName)
			SetCondition(&agent.Status.Conditions, langopv1alpha1.ConditionNetworkPolicyEnforced, metav1.ConditionTrue, langopv1alpha1.ReasonEnforced, message, agent.Generation)
			log.V(1).Info("NetworkPolicy enforcement supported", "cni", cniName)
		}
	} else {
		// Network isolation disabled - skip NetworkPolicy creation
		SetCondition(&agent.Status.Conditions, langopv1alpha1.ConditionNetworkPolicyReady, metav1.ConditionTrue, langopv1alpha1.ReasonNetworkPolicyDisabled,
			"NetworkPolicy creation disabled via networkIsolation.enabled=false", agent.Generation)
		SetCondition(&agent.Status.Conditions, langopv1alpha1.ConditionNetworkPolicyEnforced, metav1.ConditionTrue, langopv1alpha1.ReasonDisabled,
			"NetworkPolicy enforcement disabled - unrestricted network access allowed", agent.Generation)
		log.V(1).Info("Network isolation disabled - skipping NetworkPolicy creation")
	}

	// Ensure agent has a UUID for webhook routing; persisted by the end-of-reconcile status write.
	if agent.Status.UUID == "" {
		agent.Status.UUID = uuid.New().String()
		log.Info("Generated UUID for agent", "uuid", agent.Status.UUID)
	}
	// Propagate to workingAgent so AGENT_UUID is correct in the first Deployment on agents
	// that use a runtime (workingAgent is a DeepCopy made before the UUID was generated).
	workingAgent.Status.UUID = agent.Status.UUID

	// Reconcile Service for agent webhook server
	if err := r.reconcileService(ctx, workingAgent); err != nil {
		log.Error(err, "Failed to reconcile Service")
		span.RecordError(err)
		span.SetStatus(codes.Error, "Service reconciliation failed")
		SetCondition(&agent.Status.Conditions, langopv1alpha1.ConditionReady, metav1.ConditionFalse, langopv1alpha1.ReasonServiceError, err.Error(), agent.Generation)
		agent.Status.Phase = events.PhaseStatusFailed
		reconcileErr = err
		return ctrl.Result{}, err
	}

	// Reconcile webhooks (Ingress for webhook access)
	if err := r.reconcileWebhooks(ctx, workingAgent); err != nil {
		// Log webhook errors but don't fail reconciliation if domain not configured
		log.Info("Webhook reconciliation skipped or pending", "reason", err.Error())
	}

	// Reconcile ServiceAccount for agent pods
	if err := r.reconcileAgentServiceAccount(ctx, workingAgent); err != nil {
		log.Error(err, "Failed to reconcile agent ServiceAccount")
		span.RecordError(err)
		span.SetStatus(codes.Error, "ServiceAccount reconciliation failed")
		SetCondition(&agent.Status.Conditions, langopv1alpha1.ConditionReady, metav1.ConditionFalse, langopv1alpha1.ReasonServiceAccountError, err.Error(), agent.Generation)
		agent.Status.Phase = events.PhaseStatusFailed
		reconcileErr = err
		return ctrl.Result{}, err
	}

	if err := r.reconcileDeployment(ctx, workingAgent, configHash); err != nil {
		log.Error(err, "Failed to reconcile Deployment")
		span.RecordError(err)
		span.SetStatus(codes.Error, "Deployment reconciliation failed")
		SetCondition(&agent.Status.Conditions, langopv1alpha1.ConditionReady, metav1.ConditionFalse, langopv1alpha1.ReasonDeploymentError, err.Error(), agent.Generation)
		agent.Status.Phase = events.PhaseStatusFailed
		reconcileErr = err
		return ctrl.Result{}, err
	}

	if err := r.reconcilePodDisruptionBudget(ctx, workingAgent); err != nil {
		log.Error(err, "Failed to reconcile PodDisruptionBudget")
		span.RecordError(err)
		span.SetStatus(codes.Error, "PodDisruptionBudget reconciliation failed")
		SetCondition(&agent.Status.Conditions, langopv1alpha1.ConditionReady, metav1.ConditionFalse, langopv1alpha1.ReasonDeploymentError, err.Error(), agent.Generation)
		agent.Status.Phase = events.PhaseStatusFailed
		reconcileErr = err
		return ctrl.Result{}, err
	}

	if err := r.reconcileHPA(ctx, workingAgent); err != nil {
		log.Error(err, "Failed to reconcile HorizontalPodAutoscaler")
		span.RecordError(err)
		span.SetStatus(codes.Error, "HPA reconciliation failed")
		SetCondition(&agent.Status.Conditions, langopv1alpha1.ConditionReady, metav1.ConditionFalse, langopv1alpha1.ReasonDeploymentError, err.Error(), agent.Generation)
		agent.Status.Phase = events.PhaseStatusFailed
		reconcileErr = err
		return ctrl.Result{}, err
	}

	if err := r.reconcileServiceMonitor(ctx, workingAgent); err != nil {
		log.Error(err, "Failed to reconcile ServiceMonitor")
		span.RecordError(err)
		span.SetStatus(codes.Error, "ServiceMonitor reconciliation failed")
		SetCondition(&agent.Status.Conditions, langopv1alpha1.ConditionReady, metav1.ConditionFalse, langopv1alpha1.ReasonDeploymentError, err.Error(), agent.Generation)
		agent.Status.Phase = events.PhaseStatusFailed
		reconcileErr = err
		return ctrl.Result{}, err
	}

	if err := r.reconcilePrometheusRule(ctx, workingAgent); err != nil {
		log.Error(err, "Failed to reconcile PrometheusRule")
		span.RecordError(err)
		span.SetStatus(codes.Error, "PrometheusRule reconciliation failed")
		SetCondition(&agent.Status.Conditions, langopv1alpha1.ConditionReady, metav1.ConditionFalse, langopv1alpha1.ReasonDeploymentError, err.Error(), agent.Generation)
		agent.Status.Phase = events.PhaseStatusFailed
		reconcileErr = err
		return ctrl.Result{}, err
	}

	// Update status only if something changed
	statusChanged := false

	// Sync replica counts and derive phase from Deployment state.
	existingDeploy := &appsv1.Deployment{}
	if err := r.Get(ctx, types.NamespacedName{Name: agent.Name, Namespace: agent.Namespace}, existingDeploy); err == nil {
		if agent.Status.ActiveReplicas != existingDeploy.Status.Replicas ||
			agent.Status.ReadyReplicas != existingDeploy.Status.ReadyReplicas {
			agent.Status.ActiveReplicas = existingDeploy.Status.Replicas
			agent.Status.ReadyReplicas = existingDeploy.Status.ReadyReplicas
			statusChanged = true
		}

		// When HPA is active, desired count is managed by the HPA and stored in
		// existingDeploy.Spec.Replicas. Use that so Updating reflects the full rollout.
		desiredReplicas := int32(1)
		if agent.Spec.Deployment.Autoscaling != nil && existingDeploy.Spec.Replicas != nil {
			desiredReplicas = *existingDeploy.Spec.Replicas
		} else if agent.Spec.Deployment.Replicas != nil {
			desiredReplicas = *agent.Spec.Deployment.Replicas
		}

		newPhase := events.PhaseStatusPending
		if existingDeploy.Status.Replicas > 0 && existingDeploy.Status.UpdatedReplicas < desiredReplicas {
			// Pods exist but rollout is in progress — distinct from Pending (no pods yet).
			newPhase = events.PhaseStatusUpdating
		} else if existingDeploy.Status.ReadyReplicas > 0 {
			newPhase = events.PhaseStatusRunning
		} else if existingDeploy.Status.Replicas > 0 {
			// Pods exist but none ready — check Deployment conditions to distinguish
			// a transient rollout (Pending) from a crash/config error (Failed).
			for _, c := range existingDeploy.Status.Conditions {
				if c.Type == appsv1.DeploymentAvailable &&
					c.Status == corev1.ConditionFalse &&
					c.Reason == "MinimumReplicasUnavailable" {
					newPhase = events.PhaseStatusFailed
					break
				}
			}
		}
		// Downgrade Running to Degraded when a non-critical subsystem has failed
		// (e.g. NetworkPolicy timed out). The agent is operational but at reduced capability.
		if newPhase == events.PhaseStatusRunning {
			for _, c := range agent.Status.Conditions {
				if c.Type == langopv1alpha1.ConditionNetworkPolicyReady && c.Status == metav1.ConditionFalse {
					newPhase = events.PhaseStatusDegraded
					break
				}
			}
		}
		if agent.Status.Phase != newPhase {
			agent.Status.Phase = newPhase
			statusChanged = true
		}
	} else if apierrors.IsNotFound(err) {
		if agent.Status.Phase != events.PhaseStatusPending {
			agent.Status.Phase = events.PhaseStatusPending
			statusChanged = true
		}
	}

	if agent.Status.ObservedGeneration != agent.Generation {
		agent.Status.ObservedGeneration = agent.Generation
		statusChanged = true
	}

	// Build managed resources inventory. Fetch cluster domain for Ingress detection (non-fatal).
	clusterDomain := ""
	if cl := (&langopv1alpha1.LanguageCluster{}); r.Get(ctx, types.NamespacedName{Name: agent.Namespace}, cl) == nil {
		clusterDomain = cl.Spec.Domain
	}
	managed := r.buildAgentManagedResources(agent, workingAgent, clusterDomain)
	if !managedResourcesEqual(agent.Status.ManagedResources, managed) {
		agent.Status.ManagedResources = managed
		statusChanged = true
	}

	if SetCondition(&agent.Status.Conditions, langopv1alpha1.ConditionReady, metav1.ConditionTrue, langopv1alpha1.ReasonReconcileSuccess, "LanguageAgent is ready", agent.Generation) {
		statusChanged = true
	}

	if statusChanged {
		if err := r.Status().Update(ctx, agent); err != nil {
			log.Error(err, "Failed to update LanguageAgent status")
			span.RecordError(err)
			span.SetStatus(codes.Error, "Failed to update status")
			return ctrl.Result{}, err
		}
	}

	// Reconciliation successful
	span.SetStatus(codes.Ok, "Reconciliation successful")

	// No need for periodic requeues - owner-reference events from Deployment, Service, ConfigMap,
	// and other owned resources drive re-reconciliation via SetupWithManager watches.
	return ctrl.Result{}, nil
}

func (r *LanguageAgentReconciler) reconcileConfigMap(ctx context.Context, agent *langopv1alpha1.LanguageAgent) (string, error) {
	l := log.FromContext(ctx)

	cfg := agentConfigYAML{
		Agent: agentIdentityYAML{
			Name:      agent.Name,
			Namespace: agent.Namespace,
		},
		Instructions: agent.Spec.Instructions,
	}

	// Persona
	persona, err := r.fetchPersona(ctx, agent)
	if err != nil {
		l.Error(err, "Failed to fetch persona, continuing without it")
	}
	if persona != nil {
		cfg.Personas = []personaConfigYAML{{
			Name:        persona.Name,
			Tone:        persona.Spec.Tone,
			Personality: persona.Spec.Personality,
			Expertise:   persona.Spec.Expertise,
		}}
	}

	// Tools
	for _, toolRef := range agent.Spec.Tools {
		if toolRef.Enabled != nil && !*toolRef.Enabled {
			continue
		}
		tool := &langopv1alpha1.LanguageTool{}
		if err := r.Get(ctx, types.NamespacedName{Name: toolRef.Name, Namespace: agent.Namespace}, tool); err != nil {
			l.Error(err, "Failed to get tool for config.yaml, skipping", "tool", toolRef.Name)
			continue
		}
		port := tool.Spec.Port
		if port == 0 {
			port = 8080
		}
		endpoint := serviceURL(tool.Name, agent.Namespace, port)
		if tool.Spec.DeploymentMode == "sidecar" {
			endpoint = fmt.Sprintf("http://localhost:%d", port)
		}
		if cfg.Tools == nil {
			cfg.Tools = make(map[string]toolConfigYAML)
		}
		cfg.Tools[tool.Name] = toolConfigYAML{Endpoint: endpoint, Protocol: "mcp"}
	}

	// Models — all served via the shared namespace gateway
	gatewayURL := serviceURL("gateway", agent.Namespace, GatewayServicePort)
	for _, modelRef := range agent.Spec.Models {
		model := &langopv1alpha1.LanguageModel{}
		if err := r.Get(ctx, types.NamespacedName{Name: modelRef.Name, Namespace: agent.Namespace}, model); err != nil {
			l.Error(err, "Failed to get model for config.yaml, skipping", "model", modelRef.Name)
			continue
		}
		if cfg.Models == nil {
			cfg.Models = make(map[string]modelConfigYAML)
		}
		cfg.Models[modelRef.Name] = modelConfigYAML{
			Role:     modelRef.Role,
			Provider: model.Spec.Provider,
			Model:    model.Spec.ModelName,
			Endpoint: gatewayURL,
			Priority: modelRef.Priority,
		}
	}

	configYAMLBytes, err := yaml.Marshal(cfg)
	if err != nil {
		return "", fmt.Errorf("failed to marshal config.yaml: %w", err)
	}

	configHash := hashString(string(configYAMLBytes))[:16]

	data := map[string]string{
		"config.yaml": string(configYAMLBytes),
	}

	configMapName := GenerateConfigMapName(agent.Name, "agent")
	return configHash, CreateOrUpdateConfigMap(ctx, r.Client, r.Scheme, agent, configMapName, agent.Namespace, data)
}

// getToolNames extracts tool names from agent's tools
func (r *LanguageAgentReconciler) getToolNames(agent *langopv1alpha1.LanguageAgent) []string {
	var names []string
	for _, ref := range agent.Spec.Tools {
		names = append(names, ref.Name)
	}
	return names
}

// getModelNames extracts model names from agent's models
func (r *LanguageAgentReconciler) getModelNames(agent *langopv1alpha1.LanguageAgent) []string {
	var names []string
	for _, ref := range agent.Spec.Models {
		names = append(names, ref.Name)
	}
	return names
}

// getPersonaNames returns the persona name for the agent, if set
func (r *LanguageAgentReconciler) getPersonaNames(agent *langopv1alpha1.LanguageAgent) []string {
	if agent.Spec.Persona == "" {
		return nil
	}
	return []string{agent.Spec.Persona}
}

// hashString creates a SHA256 hash of a string for change detection
func hashString(s string) string {
	h := sha256.New()
	h.Write([]byte(s))
	return fmt.Sprintf("%x", h.Sum(nil))
}

func (r *LanguageAgentReconciler) reconcilePVC(ctx context.Context, agent *langopv1alpha1.LanguageAgent) error {
	// Skip if workspace is not enabled
	if agent.Spec.Workspace == nil || (agent.Spec.Workspace.Enabled != nil && !*agent.Spec.Workspace.Enabled) {
		return nil
	}

	// Determine target namespace - always use agent's namespace
	targetNamespace := agent.Namespace
	if err := ValidateClusterReference(ctx, r.Client, agent.Namespace); err != nil {
		return err
	}

	// Set defaults from WorkspaceSpec
	size := agent.Spec.Workspace.Size
	if size == "" {
		size = "10Gi"
	}

	accessMode := corev1.PersistentVolumeAccessMode(agent.Spec.Workspace.AccessMode)
	if accessMode == "" {
		accessMode = corev1.ReadWriteOnce
	}

	pvc := &corev1.PersistentVolumeClaim{
		ObjectMeta: metav1.ObjectMeta{
			Name:      GeneratePVCName(agent.Name),
			Namespace: targetNamespace,
			Labels:    GetCommonLabels(agent.Name, "LanguageAgent"),
		},
	}

	err := CreateOrUpdateOwned(ctx, r.Client, r.Scheme, agent, pvc, func() error {
		// Only set spec on creation (PVCs are immutable after creation)
		if pvc.CreationTimestamp.IsZero() {
			// Parse storage size safely to avoid controller panic
			quantity, err := resource.ParseQuantity(size)
			if err != nil {
				return fmt.Errorf("invalid workspace size %q: %w", size, err)
			}

			// Validate minimum size - PVCs cannot have zero storage
			if quantity.IsZero() {
				return fmt.Errorf("workspace size cannot be zero, got: %s", size)
			}

			pvc.Spec = corev1.PersistentVolumeClaimSpec{
				AccessModes: []corev1.PersistentVolumeAccessMode{accessMode},
				Resources: corev1.VolumeResourceRequirements{
					Requests: corev1.ResourceList{
						corev1.ResourceStorage: quantity,
					},
				},
			}

			if agent.Spec.Workspace.StorageClassName != nil {
				pvc.Spec.StorageClassName = agent.Spec.Workspace.StorageClassName
			} else if r.DefaultStorageClassName != "" {
				pvc.Spec.StorageClassName = &r.DefaultStorageClassName
			}
		}

		return nil
	})
	if err != nil {
		return err
	}

	if agent.Spec.Workspace.Retain != nil && *agent.Spec.Workspace.Retain {
		agent.Status.WorkspacePVCName = GeneratePVCName(agent.Name)
	}
	return nil
}

// reconcileWorkspaceSeedConfigMap creates or deletes the workspace-seed ConfigMap
// that holds the contents of spec.workspace.initialFiles.
// When InitialFiles is empty (or workspace is nil/disabled), the ConfigMap is deleted
// so stale seed data is not left behind.
func (r *LanguageAgentReconciler) reconcileWorkspaceSeedConfigMap(ctx context.Context, agent *langopv1alpha1.LanguageAgent) error {
	cmName := GenerateConfigMapName(agent.Name, "workspace-seed")

	wsEnabled := agent.Spec.Workspace != nil &&
		(agent.Spec.Workspace.Enabled == nil || *agent.Spec.Workspace.Enabled)

	if !wsEnabled || len(agent.Spec.Workspace.InitialFiles) == 0 {
		// No InitialFiles — remove any previously-created seed ConfigMap.
		return DeleteConfigMap(ctx, r.Client, cmName, agent.Namespace)
	}

	return CreateOrUpdateConfigMap(ctx, r.Client, r.Scheme, agent, cmName, agent.Namespace, agent.Spec.Workspace.InitialFiles)
}

// workspaceSeedEnabled reports whether workspace seeding is configured on the agent.
func workspaceSeedEnabled(agent *langopv1alpha1.LanguageAgent) bool {
	if agent.Spec.Workspace == nil {
		return false
	}
	if agent.Spec.Workspace.Enabled != nil && !*agent.Spec.Workspace.Enabled {
		return false
	}
	return len(agent.Spec.Workspace.InitialFiles) > 0 || agent.Spec.Workspace.SeedConfigMapRef != nil
}

// buildWorkspaceSeedVolumes returns pod-level volumes required by the workspace-seeder
// init container. These are not mounted in the main agent container.
func buildWorkspaceSeedVolumes(agent *langopv1alpha1.LanguageAgent) []corev1.Volume {
	if !workspaceSeedEnabled(agent) {
		return nil
	}
	var vols []corev1.Volume
	if len(agent.Spec.Workspace.InitialFiles) > 0 {
		vols = append(vols, corev1.Volume{
			Name: "workspace-seed-init",
			VolumeSource: corev1.VolumeSource{
				ConfigMap: &corev1.ConfigMapVolumeSource{
					LocalObjectReference: corev1.LocalObjectReference{
						Name: GenerateConfigMapName(agent.Name, "workspace-seed"),
					},
				},
			},
		})
	}
	if agent.Spec.Workspace.SeedConfigMapRef != nil {
		vols = append(vols, corev1.Volume{
			Name: "workspace-seed-ref",
			VolumeSource: corev1.VolumeSource{
				ConfigMap: &corev1.ConfigMapVolumeSource{
					LocalObjectReference: *agent.Spec.Workspace.SeedConfigMapRef,
				},
			},
		})
	}
	return vols
}

// buildWorkspaceSeedInitContainer returns the workspace-seeder init container when seeding
// is configured, or nil otherwise. The container uses seed-once semantics: files are only
// copied if they do not already exist at the destination path, preserving any agent edits.
// InitialFiles are processed first (higher priority), SeedConfigMapRef second.
func buildWorkspaceSeedInitContainer(agent *langopv1alpha1.LanguageAgent) *corev1.Container {
	if !workspaceSeedEnabled(agent) {
		return nil
	}

	mountPath := agent.Spec.Workspace.MountPath
	if mountPath == "" {
		mountPath = "/workspace"
	}

	// Build the shell script. Both loops use seed-once semantics (test -f).
	script := fmt.Sprintf(`set -e
WORKSPACE=%s
if [ -d /seed-init ]; then
  for f in /seed-init/*; do
    [ -f "$f" ] || continue
    dest="$WORKSPACE/$(basename "$f")"
    [ -f "$dest" ] || cp "$f" "$dest"
  done
fi
if [ -d /seed-ref ]; then
  for f in /seed-ref/*; do
    [ -f "$f" ] || continue
    dest="$WORKSPACE/$(basename "$f")"
    [ -f "$dest" ] || cp "$f" "$dest"
  done
fi`, mountPath)

	mounts := []corev1.VolumeMount{
		{
			Name:      "workspace",
			MountPath: mountPath,
		},
	}
	if len(agent.Spec.Workspace.InitialFiles) > 0 {
		mounts = append(mounts, corev1.VolumeMount{
			Name:      "workspace-seed-init",
			MountPath: "/seed-init",
			ReadOnly:  true,
		})
	}
	if agent.Spec.Workspace.SeedConfigMapRef != nil {
		mounts = append(mounts, corev1.VolumeMount{
			Name:      "workspace-seed-ref",
			MountPath: "/seed-ref",
			ReadOnly:  true,
		})
	}

	return &corev1.Container{
		Name:         "workspace-seeder",
		Image:        "busybox:latest",
		Command:      []string{"/bin/sh", "-c", script},
		VolumeMounts: mounts,
	}
}

// buildVolumes creates the volumes and volume mounts for agent pods
func (r *LanguageAgentReconciler) buildVolumes(ctx context.Context, agent *langopv1alpha1.LanguageAgent) ([]corev1.Volume, []corev1.VolumeMount) {
	volumes := []corev1.Volume{}
	volumeMounts := []corev1.VolumeMount{}

	// Add tmpfs volumes for read-only root filesystem
	// /tmp - general temporary files
	volumes = append(volumes, corev1.Volume{
		Name: "tmp",
		VolumeSource: corev1.VolumeSource{
			EmptyDir: &corev1.EmptyDirVolumeSource{
				Medium: corev1.StorageMediumMemory, // Use tmpfs
			},
		},
	})
	volumeMounts = append(volumeMounts, corev1.VolumeMount{
		Name:      "tmp",
		MountPath: "/tmp",
	})

	// Mount agent config ConfigMap at /etc/agent/ (provides config.yaml)
	agentConfigMapName := GenerateConfigMapName(agent.Name, "agent")
	volumes = append(volumes, corev1.Volume{
		Name: "agent-config",
		VolumeSource: corev1.VolumeSource{
			ConfigMap: &corev1.ConfigMapVolumeSource{
				LocalObjectReference: corev1.LocalObjectReference{
					Name: agentConfigMapName,
				},
			},
		},
	})
	volumeMounts = append(volumeMounts, corev1.VolumeMount{
		Name:      "agent-config",
		MountPath: "/etc/agent",
		ReadOnly:  true,
	})

	// Add workspace volume if enabled
	if agent.Spec.Workspace != nil && (agent.Spec.Workspace.Enabled == nil || *agent.Spec.Workspace.Enabled) {
		mountPath := agent.Spec.Workspace.MountPath
		if mountPath == "" {
			mountPath = "/workspace"
		}

		volumes = append(volumes, corev1.Volume{
			Name: "workspace",
			VolumeSource: corev1.VolumeSource{
				PersistentVolumeClaim: &corev1.PersistentVolumeClaimVolumeSource{
					ClaimName: GeneratePVCName(agent.Name),
				},
			},
		})

		volumeMounts = append(volumeMounts, corev1.VolumeMount{
			Name:      "workspace",
			MountPath: mountPath,
		})
	}

	return volumes, volumeMounts
}

func (r *LanguageAgentReconciler) reconcileDeployment(ctx context.Context, agent *langopv1alpha1.LanguageAgent, configHash string) error {
	// Resolve model URLs and names
	modelURLs, modelNames, err := r.resolveModels(ctx, agent)
	if err != nil {
		return fmt.Errorf("failed to resolve models: %w", err)
	}

	// Resolve tool URLs
	toolURLs, err := r.resolveTools(ctx, agent)
	if err != nil {
		return fmt.Errorf("failed to resolve tools: %w", err)
	}

	// Resolve sidecar tools
	sidecarContainers, err := r.resolveSidecarTools(ctx, agent)
	if err != nil {
		return fmt.Errorf("failed to resolve sidecar tools: %w", err)
	}

	// Determine target namespace and labels
	targetNamespace := agent.Namespace
	labels := GetCommonLabels(agent.Name, "LanguageAgent")
	labels[LabelKeyLangopComponent] = "agent" // Distinguish from trigger pods

	if err := ValidateClusterReference(ctx, r.Client, agent.Namespace); err != nil {
		return err
	}

	cluster := &langopv1alpha1.LanguageCluster{}
	if err := r.Get(ctx, types.NamespacedName{Name: agent.Namespace}, cluster); err != nil {
		return fmt.Errorf("failed to get cluster %s: %w", agent.Namespace, err)
	}

	labels[LabelKeyLangopCluster] = agent.Namespace

	deployment := &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      agent.Name,
			Namespace: targetNamespace,
			Labels:    labels,
		},
	}

	err = CreateOrUpdateOwned(ctx, r.Client, r.Scheme, agent, deployment, func() error {
		replicas := int32(1)
		if agent.Spec.Deployment.Replicas != nil {
			replicas = *agent.Spec.Deployment.Replicas
		}
		// When HPA is active, preserve the current replica count rather than overwriting it.
		// The HPA controller owns spec.replicas; if we reset it every reconcile the
		// Deployment will never scale.
		deploymentReplicas := &replicas
		if agent.Spec.Deployment.Autoscaling != nil && deployment.Spec.Replicas != nil {
			deploymentReplicas = deployment.Spec.Replicas
		}

		// Build container list starting with the agent
		containers := []corev1.Container{
			{
				Name:            "agent",
				Image:           agent.Spec.Image,
				ImagePullPolicy: agent.Spec.Deployment.ImagePullPolicy,
				Command:         agent.Spec.Deployment.Command,
				Args:            agent.Spec.Deployment.Args,
				Env:             r.buildAgentEnv(ctx, agent, cluster, modelURLs, modelNames, toolURLs),
				EnvFrom:         agent.Spec.Deployment.EnvFrom,
				Resources:       agent.Spec.Deployment.Resources,
				LivenessProbe:   agent.Spec.Deployment.LivenessProbe,
				ReadinessProbe:  agent.Spec.Deployment.ReadinessProbe,
				StartupProbe:    agent.Spec.Deployment.StartupProbe,
				Ports:           buildAgentContainerPorts(agentPorts(agent)),
			},
		}

		// Inject operator-managed env vars and volume mounts into user-specified init containers.
		// Per spec/agents.md, all contracted env vars must be present in every
		// container including init containers. The agent-config volume mount is also
		// injected so init containers (e.g. runtime adapters) can read /etc/agent/config.yaml.
		userInitContainers := make([]corev1.Container, len(agent.Spec.Deployment.InitContainers))
		copy(userInitContainers, agent.Spec.Deployment.InitContainers)
		agentEnv := r.buildAgentEnv(ctx, agent, cluster, modelURLs, modelNames, toolURLs)
		agentConfigMount := corev1.VolumeMount{
			Name:      "agent-config",
			MountPath: "/etc/agent",
			ReadOnly:  true,
		}
		for i := range userInitContainers {
			userInitContainers[i].Env = append(agentEnv, userInitContainers[i].Env...)
			userInitContainers[i].VolumeMounts = append([]corev1.VolumeMount{agentConfigMount}, userInitContainers[i].VolumeMounts...)
		}

		// Prepend the workspace-seeder init container so the workspace is populated
		// before any user init containers or sidecar tools run.
		allInitContainers := userInitContainers
		if seedContainer := buildWorkspaceSeedInitContainer(agent); seedContainer != nil {
			allInitContainers = append([]corev1.Container{*seedContainer}, allInitContainers...)
		}
		allInitContainers = append(allInitContainers, sidecarContainers...)

		// Merge user pod labels; operator-managed labels take precedence to protect selector stability.
		podLabels := make(map[string]string, len(labels)+len(agent.Spec.Deployment.PodLabels))
		for k, v := range agent.Spec.Deployment.PodLabels {
			podLabels[k] = v
		}
		for k, v := range labels {
			podLabels[k] = v
		}

		// Use user-supplied pod security context if set, otherwise apply operator defaults.
		podSecCtx := buildPodSecurityContext()
		if agent.Spec.Deployment.SecurityContext != nil {
			podSecCtx = agent.Spec.Deployment.SecurityContext
		}

		// Seed pod annotations with the operator-managed config-hash, then overlay user annotations.
		podAnnotations := map[string]string{LabelKeyLangopConfigHash: configHash}
		maps.Copy(podAnnotations, agent.Spec.Deployment.PodAnnotations)

		deployment.Spec = appsv1.DeploymentSpec{
			Replicas: deploymentReplicas,
			Selector: &metav1.LabelSelector{
				MatchLabels: labels,
			},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels:      podLabels,
					Annotations: podAnnotations,
				},
				Spec: corev1.PodSpec{
					ServiceAccountName:        r.getServiceAccountName(agent),
					ShareProcessNamespace:     &[]bool{len(allInitContainers) > 0}[0],
					InitContainers:            allInitContainers,
					Containers:                containers,
					SecurityContext:           podSecCtx,
					ImagePullSecrets:          agent.Spec.Deployment.ImagePullSecrets,
					NodeSelector:              agent.Spec.Deployment.NodeSelector,
					Tolerations:               agent.Spec.Deployment.Tolerations,
					TopologySpreadConstraints: agent.Spec.Deployment.TopologySpreadConstraints,
					Affinity:                  agent.Spec.Deployment.Affinity,
				},
			},
		}

		// Add container security context for agent container
		deployment.Spec.Template.Spec.Containers[0].SecurityContext = buildContainerSecurityContext()

		// Build operator-managed volumes and volume mounts, then append user-supplied ones.
		volumes, volumeMounts := r.buildVolumes(ctx, agent)
		// Append seed ConfigMap volumes (not mounted in main container; used by workspace-seeder init container).
		volumes = append(volumes, buildWorkspaceSeedVolumes(agent)...)
		volumes = append(volumes, agent.Spec.Deployment.Volumes...)
		volumeMounts = append(volumeMounts, agent.Spec.Deployment.VolumeMounts...)
		if len(volumes) > 0 {
			deployment.Spec.Template.Spec.Volumes = volumes
		}
		if len(volumeMounts) > 0 {
			deployment.Spec.Template.Spec.Containers[0].VolumeMounts = volumeMounts
		}

		return nil
	})

	return err
}

func (r *LanguageAgentReconciler) reconcileNetworkPolicy(ctx context.Context, agent *langopv1alpha1.LanguageAgent) error {
	labels := GetCommonLabels(agent.Name, "LanguageAgent")

	// Get OTEL endpoint from operator environment
	// This ensures agents can send traces to the collector
	otelEndpoint := os.Getenv("OTEL_EXPORTER_OTLP_ENDPOINT")

	// Build NetworkPolicy using helper from utils.go
	networkPolicy := BuildEgressNetworkPolicy(
		ctx,
		r.Client,
		agent.Name,
		agent.Namespace,
		labels,
		otelEndpoint,
		agent.Spec.NetworkPolicies,
	)

	// Add ingress rules to allow trigger pods to connect to agent.
	// Build NetworkPolicy port list from all agent ports.
	var npPorts []networkingv1.NetworkPolicyPort
	for _, ap := range agentPorts(agent) {
		proto := ap.Protocol
		if proto == "" {
			proto = corev1.ProtocolTCP
		}
		portCopy := intstr.FromInt32(ap.Port) // copy per iteration to avoid pointer aliasing
		npPorts = append(npPorts, networkingv1.NetworkPolicyPort{
			Protocol: ptr.To(proto),
			Port:     &portCopy,
		})
	}
	networkPolicy.Spec.PolicyTypes = append(networkPolicy.Spec.PolicyTypes, networkingv1.PolicyTypeIngress)
	networkPolicy.Spec.Ingress = []networkingv1.NetworkPolicyIngressRule{
		{
			// Allow trigger pods in same namespace to connect to agent
			From: []networkingv1.NetworkPolicyPeer{
				{
					PodSelector: &metav1.LabelSelector{
						MatchLabels: map[string]string{
							LabelKeyLangopComponent: "trigger",
						},
					},
				},
			},
			Ports: npPorts,
		},
		{
			// Allow other agent pods to connect (agent-to-agent traffic)
			From: []networkingv1.NetworkPolicyPeer{
				{
					PodSelector: &metav1.LabelSelector{
						MatchLabels: map[string]string{
							LabelKeyLangopKind: "LanguageAgent",
						},
					},
				},
			},
			Ports: npPorts,
		},
	}

	// Allow ingress controller pods to reach agent ports (needed when an Ingress routes external
	// traffic to the agent). Only added when the ingress controller namespace is configured.
	if r.IngressControllerNamespace != "" {
		networkPolicy.Spec.Ingress = append(networkPolicy.Spec.Ingress, networkingv1.NetworkPolicyIngressRule{
			From: []networkingv1.NetworkPolicyPeer{
				{
					NamespaceSelector: &metav1.LabelSelector{
						MatchLabels: map[string]string{
							LabelKeyMetadataName: r.IngressControllerNamespace,
						},
					},
				},
			},
			Ports: npPorts,
		})
	}

	// Append user-defined ingress rules from spec.networkPolicies.ingress
	if agent.Spec.NetworkPolicies != nil {
		for _, rule := range agent.Spec.NetworkPolicies.Ingress {
			ingressRule := networkingv1.NetworkPolicyIngressRule{}
			for _, peer := range rule.From {
				peer := peer
				ingressRule.From = append(ingressRule.From, buildIngressPeerFromNetworkPeer(&peer, agent.Namespace))
			}
			for _, p := range rule.Ports {
				protocol := corev1.Protocol(p.Protocol)
				if protocol == "" {
					protocol = corev1.ProtocolTCP
				}
				ingressRule.Ports = append(ingressRule.Ports, networkingv1.NetworkPolicyPort{
					Protocol: ptr.To(protocol),
					Port:     &intstr.IntOrString{Type: intstr.Int, IntVal: p.Port},
				})
			}
			networkPolicy.Spec.Ingress = append(networkPolicy.Spec.Ingress, ingressRule)
		}
	}

	// Create or update the NetworkPolicy with owner reference and configured timeout/retries
	return CreateOrUpdateNetworkPolicyWithTimeout(ctx, r.Client, r.Scheme, agent, networkPolicy, r.NetworkPolicyTimeout, r.NetworkPolicyRetries)
}

func (r *LanguageAgentReconciler) resolveModels(ctx context.Context, agent *langopv1alpha1.LanguageAgent) ([]string, []string, error) {
	var modelURLs []string
	var modelNames []string

	for _, modelRef := range agent.Spec.Models {
		// Fetch the LanguageModel (always in the agent's namespace)
		model := &langopv1alpha1.LanguageModel{}
		if err := r.Get(ctx, types.NamespacedName{Name: modelRef.Name, Namespace: agent.Namespace}, model); err != nil {
			return nil, nil, fmt.Errorf("failed to get model %s/%s: %w", agent.Namespace, modelRef.Name, err)
		}

		// All models in a cluster are served by the shared gateway
		// in the cluster namespace. Deduplicate: only add the gateway URL once.
		gatewayURL := serviceURL("gateway", agent.Namespace, GatewayServicePort)
		alreadyAdded := false
		for _, u := range modelURLs {
			if u == gatewayURL {
				alreadyAdded = true
				break
			}
		}
		if !alreadyAdded {
			modelURLs = append(modelURLs, gatewayURL)
		}

		// Collect model name from spec
		if model.Spec.ModelName != "" {
			modelNames = append(modelNames, model.Spec.ModelName)
		}
	}

	return modelURLs, modelNames, nil
}

func (r *LanguageAgentReconciler) resolveSidecarTools(ctx context.Context, agent *langopv1alpha1.LanguageAgent) ([]corev1.Container, error) {
	var sidecarContainers []corev1.Container

	for _, toolRef := range agent.Spec.Tools {
		// Skip tools explicitly disabled by the user
		if toolRef.Enabled != nil && !*toolRef.Enabled {
			continue
		}

		// Fetch the LanguageTool (always in the agent's namespace)
		tool := &langopv1alpha1.LanguageTool{}
		if err := r.Get(ctx, types.NamespacedName{Name: toolRef.Name, Namespace: agent.Namespace}, tool); err != nil {
			return nil, fmt.Errorf("failed to get tool %s/%s: %w", agent.Namespace, toolRef.Name, err)
		}

		// Only process sidecar tools
		if tool.Spec.DeploymentMode != "sidecar" {
			continue
		}

		// Build sidecar container spec
		port := tool.Spec.Port
		if port == 0 {
			port = 8080 // Default MCP port
		}

		// Use native sidecar support (Kubernetes 1.28+)
		// Sidecars with restartPolicy: Always will terminate automatically
		// when the main container completes
		restartPolicy := corev1.ContainerRestartPolicyAlways
		container := corev1.Container{
			Name:          fmt.Sprintf("tool-%s", tool.Name),
			Image:         tool.Spec.Image,
			RestartPolicy: &restartPolicy,
			Ports: []corev1.ContainerPort{
				{
					Name:          "mcp",
					ContainerPort: port,
					Protocol:      corev1.ProtocolTCP,
				},
			},
			Env: tool.Spec.Deployment.Env,
			ReadinessProbe: &corev1.Probe{
				ProbeHandler: corev1.ProbeHandler{
					TCPSocket: &corev1.TCPSocketAction{
						Port: intstr.FromInt(int(port)),
					},
				},
				InitialDelaySeconds: 2,
				PeriodSeconds:       2,
				TimeoutSeconds:      1,
				SuccessThreshold:    1,
				FailureThreshold:    3,
			},
		}

		// Add resource requirements if specified, otherwise use sensible defaults for tool containers
		if tool.Spec.Deployment.Resources.Requests == nil && tool.Spec.Deployment.Resources.Limits == nil {
			container.Resources = corev1.ResourceRequirements{
				Requests: corev1.ResourceList{
					corev1.ResourceCPU:    resource.MustParse("50m"),
					corev1.ResourceMemory: resource.MustParse("128Mi"),
				},
				Limits: corev1.ResourceList{
					corev1.ResourceCPU:    resource.MustParse("500m"),
					corev1.ResourceMemory: resource.MustParse("1Gi"),
				},
			}
		} else {
			container.Resources = tool.Spec.Deployment.Resources
		}

		// Mount workspace if agent has workspace enabled
		if agent.Spec.Workspace != nil && (agent.Spec.Workspace.Enabled == nil || *agent.Spec.Workspace.Enabled) {
			mountPath := agent.Spec.Workspace.MountPath
			if mountPath == "" {
				mountPath = "/workspace"
			}

			container.VolumeMounts = []corev1.VolumeMount{
				{
					Name:      "workspace",
					MountPath: mountPath,
				},
			}
		}

		sidecarContainers = append(sidecarContainers, container)
	}

	return sidecarContainers, nil
}

func (r *LanguageAgentReconciler) resolveTools(ctx context.Context, agent *langopv1alpha1.LanguageAgent) ([]string, error) {
	var toolURLs []string

	for _, toolRef := range agent.Spec.Tools {
		// Skip tools explicitly disabled by the user
		if toolRef.Enabled != nil && !*toolRef.Enabled {
			continue
		}

		// Fetch the LanguageTool (always in the agent's namespace)
		tool := &langopv1alpha1.LanguageTool{}
		if err := r.Get(ctx, types.NamespacedName{Name: toolRef.Name, Namespace: agent.Namespace}, tool); err != nil {
			return nil, fmt.Errorf("failed to get tool %s/%s: %w", agent.Namespace, toolRef.Name, err)
		}

		port := tool.Spec.Port
		if port == 0 {
			port = 8080 // Default MCP port
		}

		// Sidecar tools use localhost URLs
		if tool.Spec.DeploymentMode == "sidecar" {
			// Format: http://localhost:<port>
			localhostURL := fmt.Sprintf("http://localhost:%d", port)
			toolURLs = append(toolURLs, localhostURL)
			continue
		}

		// Build MCP server URL (service mode)
		mcpURL := serviceURL(tool.Name, agent.Namespace, port)
		toolURLs = append(toolURLs, mcpURL)
	}

	return toolURLs, nil
}

func (r *LanguageAgentReconciler) buildAgentEnv(ctx context.Context, agent *langopv1alpha1.LanguageAgent, cluster *langopv1alpha1.LanguageCluster, modelURLs []string, modelNames []string, toolURLs []string) []corev1.EnvVar {
	env := []corev1.EnvVar{
		{
			Name:  "AGENT_NAME",
			Value: agent.Name,
		},
		{
			Name:  "AGENT_NAMESPACE",
			Value: agent.Namespace,
		},
		{
			Name:  "AGENT_UUID",
			Value: agent.Status.UUID,
		},
		{
			Name:  "AGENT_CLUSTER_NAME",
			Value: cluster.Name,
		},
		{
			Name:  "AGENT_CLUSTER_UUID",
			Value: string(cluster.UID),
		},
	}

	// Pass through OpenTelemetry collector endpoint from operator environment.
	// Agents are responsible for configuring their own OTEL SDK.
	if endpoint := os.Getenv("OTEL_EXPORTER_OTLP_ENDPOINT"); endpoint != "" {
		env = append(env, corev1.EnvVar{
			Name:  "OTEL_EXPORTER_OTLP_ENDPOINT",
			Value: endpoint,
		})
		env = append(env, corev1.EnvVar{
			Name:  "OTEL_SERVICE_NAME",
			Value: fmt.Sprintf("agent-%s", agent.Name),
		})

		if resourceAttrs := os.Getenv("OTEL_RESOURCE_ATTRIBUTES"); resourceAttrs != "" {
			env = append(env, corev1.EnvVar{
				Name:  "OTEL_RESOURCE_ATTRIBUTES",
				Value: resourceAttrs,
			})
		}
		if sampler := os.Getenv("OTEL_TRACES_SAMPLER"); sampler != "" {
			env = append(env, corev1.EnvVar{
				Name:  "OTEL_TRACES_SAMPLER",
				Value: sampler,
			})
		}
		if samplerArg := os.Getenv("OTEL_TRACES_SAMPLER_ARG"); samplerArg != "" {
			env = append(env, corev1.EnvVar{
				Name:  "OTEL_TRACES_SAMPLER_ARG",
				Value: samplerArg,
			})
		}
	}

	if agent.Spec.Instructions != "" {
		env = append(env, corev1.EnvVar{
			Name:  "AGENT_INSTRUCTIONS",
			Value: agent.Spec.Instructions,
		})
	}

	// Model gateway URLs and names (comma-separated)
	if len(modelURLs) > 0 {
		env = append(env, corev1.EnvVar{
			Name:  "MODEL_ENDPOINT",
			Value: strings.Join(modelURLs, ","),
		})
	}
	if len(modelNames) > 0 {
		env = append(env, corev1.EnvVar{
			Name:  "LLM_MODEL",
			Value: strings.Join(modelNames, ","),
		})
	}

	// MCP tool server URLs (comma-separated)
	if len(toolURLs) > 0 {
		env = append(env, corev1.EnvVar{
			Name:  "MCP_SERVERS",
			Value: strings.Join(toolURLs, ","),
		})
	}

	// User-specified env vars (may override any of the above)
	env = append(env, agent.Spec.Deployment.Env...)

	return env
}

func (r *LanguageAgentReconciler) fetchPersona(ctx context.Context, agent *langopv1alpha1.LanguageAgent) (*langopv1alpha1.LanguagePersona, error) {
	if agent.Spec.Persona == "" {
		return nil, nil
	}

	persona := &langopv1alpha1.LanguagePersona{}
	if err := r.Get(ctx, types.NamespacedName{Name: agent.Spec.Persona, Namespace: agent.Namespace}, persona); err != nil {
		if apierrors.IsNotFound(err) {
			return nil, fmt.Errorf("persona %s/%s not found", agent.Namespace, agent.Spec.Persona)
		}
		return nil, fmt.Errorf("failed to get persona %s/%s: %w", agent.Namespace, agent.Spec.Persona, err)
	}

	if persona.Status.Phase != events.PhaseStatusReady {
		return nil, fmt.Errorf("persona %s/%s is not ready (phase: %s)", agent.Namespace, agent.Spec.Persona, persona.Status.Phase)
	}

	return persona, nil
}

func (r *LanguageAgentReconciler) cleanupResources(ctx context.Context, agent *langopv1alpha1.LanguageAgent) error {
	// Child resources (Service, Ingress, PVC, Deployment, NetworkPolicy) all carry owner
	// references set via SetControllerReference. Kubernetes GC deletes them automatically
	// when the agent is deleted; no explicit polling is needed.
	//
	// Exception: when spec.workspace.retain is true, strip the PVC's ownerReference
	// before the finalizer is removed so GC does not collect it.
	//
	// Per-agent RBAC resources (ServiceAccount, Role, RoleBinding) are not owned by the
	// agent (ServiceAccounts cannot be owner-referenced from Pods across namespaces) so
	// they must be deleted explicitly here.
	if agent.Spec.Workspace != nil &&
		(agent.Spec.Workspace.Enabled == nil || *agent.Spec.Workspace.Enabled) &&
		agent.Spec.Workspace.Retain != nil && *agent.Spec.Workspace.Retain {
		if err := r.orphanWorkspacePVC(ctx, agent); err != nil {
			return err
		}
	}
	return r.cleanupPerAgentRBAC(ctx, agent)
}

// orphanWorkspacePVC removes the ownerReference from the workspace PVC so that
// Kubernetes GC does not delete it when the LanguageAgent is deleted.
func (r *LanguageAgentReconciler) orphanWorkspacePVC(ctx context.Context, agent *langopv1alpha1.LanguageAgent) error {
	pvc := &corev1.PersistentVolumeClaim{}
	err := r.Get(ctx, types.NamespacedName{Name: GeneratePVCName(agent.Name), Namespace: agent.Namespace}, pvc)
	if apierrors.IsNotFound(err) {
		return nil
	}
	if err != nil {
		return fmt.Errorf("failed to get workspace PVC: %w", err)
	}
	patch := client.MergeFrom(pvc.DeepCopy())
	pvc.OwnerReferences = nil
	if err := r.Patch(ctx, pvc, patch); err != nil {
		return fmt.Errorf("failed to patch workspace PVC owner refs: %w", err)
	}
	log.FromContext(ctx).Info("Retained workspace PVC", "pvc", pvc.Name)
	return nil
}

// cleanupPerAgentRBAC deletes the per-agent ServiceAccount, Role, and RoleBinding.
// These are not covered by owner-reference GC because ServiceAccounts cannot be
// owner-referenced from Pods across namespaces, so they must be deleted explicitly.
// Skipped when a custom ServiceAccountName is set (user manages their own SA).
func (r *LanguageAgentReconciler) cleanupPerAgentRBAC(ctx context.Context, agent *langopv1alpha1.LanguageAgent) error {
	if agent.Spec.Deployment.ServiceAccountName != "" {
		return nil
	}
	log := log.FromContext(ctx)
	ns := agent.Namespace
	saName := GenerateServiceAccountName(agent.Name)

	toDelete := []client.Object{
		&rbacv1.RoleBinding{ObjectMeta: metav1.ObjectMeta{Name: saName, Namespace: ns}},
		&rbacv1.Role{ObjectMeta: metav1.ObjectMeta{Name: saName, Namespace: ns}},
		&corev1.ServiceAccount{ObjectMeta: metav1.ObjectMeta{Name: saName, Namespace: ns}},
	}
	for _, obj := range toDelete {
		if err := r.Delete(ctx, obj); err != nil && !apierrors.IsNotFound(err) {
			return fmt.Errorf("failed to delete %T %s: %w", obj, saName, err)
		}
		log.Info("Deleted per-agent RBAC resource", "kind", fmt.Sprintf("%T", obj), "name", saName, "namespace", ns)
	}
	return nil
}

// agentPorts returns the effective port list for the agent.
// Falls back to a single http:8080 port when spec.ports is not set.
func agentPorts(agent *langopv1alpha1.LanguageAgent) []langopv1alpha1.AgentPort {
	if len(agent.Spec.Ports) > 0 {
		return agent.Spec.Ports
	}
	return []langopv1alpha1.AgentPort{
		{Name: "http", Port: 8080, Protocol: corev1.ProtocolTCP, Expose: ptr.To(true)},
	}
}

// agentIngressPort returns the port that ingress/HTTPRoute should route to.
// Uses the first port with Expose: true; falls back to the first port; falls back to 8080.
func agentIngressPort(agent *langopv1alpha1.LanguageAgent) int32 {
	ports := agentPorts(agent)
	for _, p := range ports {
		if p.Expose != nil && *p.Expose {
			return p.Port
		}
	}
	if len(ports) > 0 {
		return ports[0].Port
	}
	return 8080
}

// buildAgentContainerPorts converts AgentPorts to ContainerPorts for visibility in the Deployment.
// Kubernetes does not enforce declared container ports; this is informational only.
func buildAgentContainerPorts(ports []langopv1alpha1.AgentPort) []corev1.ContainerPort {
	out := make([]corev1.ContainerPort, 0, len(ports))
	for _, p := range ports {
		proto := p.Protocol
		if proto == "" {
			proto = corev1.ProtocolTCP
		}
		out = append(out, corev1.ContainerPort{
			Name:          p.Name,
			ContainerPort: p.Port,
			Protocol:      proto,
		})
	}
	return out
}

// reconcileService creates a Service for the agent
func (r *LanguageAgentReconciler) reconcileService(ctx context.Context, agent *langopv1alpha1.LanguageAgent) error {
	labels := GetCommonLabels(agent.Name, "LanguageAgent")
	labels[LabelKeyLangopComponent] = "agent" // Only route to agent pods, not trigger pods

	service := &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      agent.Name,
			Namespace: agent.Namespace,
			Labels:    labels,
		},
	}

	err := CreateOrUpdateOwned(ctx, r.Client, r.Scheme, agent, service, func() error {
		svcType := agent.Spec.Deployment.ServiceType
		if svcType == "" {
			svcType = corev1.ServiceTypeClusterIP
		}
		svcPorts := make([]corev1.ServicePort, 0)
		for _, ap := range agentPorts(agent) {
			proto := ap.Protocol
			if proto == "" {
				proto = corev1.ProtocolTCP
			}
			svcPorts = append(svcPorts, corev1.ServicePort{
				Name:       ap.Name,
				Port:       ap.Port,
				TargetPort: intstr.FromInt32(ap.Port),
				Protocol:   proto,
			})
		}
		service.Spec = corev1.ServiceSpec{
			Selector: labels,
			Ports:    svcPorts,
			Type:     svcType,
		}
		if len(agent.Spec.Deployment.ServiceAnnotations) > 0 {
			if service.Annotations == nil {
				service.Annotations = make(map[string]string)
			}
			for k, v := range agent.Spec.Deployment.ServiceAnnotations {
				service.Annotations[k] = v
			}
		}

		return nil
	})

	return err
}

// reconcilePodDisruptionBudget creates a PodDisruptionBudget for agents with more than one replica,
// and deletes any existing PDB when the agent is scaled down to a single replica.
func (r *LanguageAgentReconciler) reconcilePodDisruptionBudget(ctx context.Context, agent *langopv1alpha1.LanguageAgent) error {
	replicas := int32(1)
	if agent.Spec.Deployment.Replicas != nil {
		replicas = *agent.Spec.Deployment.Replicas
	}

	pdb := &policyv1.PodDisruptionBudget{
		ObjectMeta: metav1.ObjectMeta{
			Name:      agent.Name,
			Namespace: agent.Namespace,
		},
	}

	if replicas <= 1 {
		if err := r.Delete(ctx, pdb); err != nil && !apierrors.IsNotFound(err) {
			return fmt.Errorf("failed to delete PodDisruptionBudget: %w", err)
		}
		return nil
	}

	minAvailable := intstr.FromInt32(1)
	labels := GetCommonLabels(agent.Name, "LanguageAgent")

	err := CreateOrUpdateOwned(ctx, r.Client, r.Scheme, agent, pdb, func() error {
		pdb.Labels = labels
		pdb.Spec = policyv1.PodDisruptionBudgetSpec{
			MinAvailable: &minAvailable,
			Selector: &metav1.LabelSelector{
				MatchLabels: labels,
			},
		}
		return nil
	})
	return err
}

// reconcileHPA creates a HorizontalPodAutoscaler for agents with autoscaling configured,
// and deletes any existing HPA when autoscaling is removed from the spec.
func (r *LanguageAgentReconciler) reconcileHPA(ctx context.Context, agent *langopv1alpha1.LanguageAgent) error {
	hpa := &autoscalingv2.HorizontalPodAutoscaler{
		ObjectMeta: metav1.ObjectMeta{
			Name:      agent.Name,
			Namespace: agent.Namespace,
		},
	}

	if agent.Spec.Deployment.Autoscaling == nil {
		if err := r.Delete(ctx, hpa); err != nil && !apierrors.IsNotFound(err) {
			return fmt.Errorf("failed to delete HorizontalPodAutoscaler: %w", err)
		}
		return nil
	}

	as := agent.Spec.Deployment.Autoscaling
	labels := GetCommonLabels(agent.Name, "LanguageAgent")

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

	err := CreateOrUpdateOwned(ctx, r.Client, r.Scheme, agent, hpa, func() error {
		hpa.Labels = labels
		hpa.Spec = autoscalingv2.HorizontalPodAutoscalerSpec{
			ScaleTargetRef: autoscalingv2.CrossVersionObjectReference{
				APIVersion: "apps/v1",
				Kind:       "Deployment",
				Name:       agent.Name,
			},
			MinReplicas: as.MinReplicas,
			MaxReplicas: as.MaxReplicas,
			Metrics:     metrics,
		}
		return nil
	})
	return err
}

// reconcileServiceMonitor creates or deletes a Prometheus Operator ServiceMonitor for the agent.
// When spec.monitoring.serviceMonitor.enabled is true, a ServiceMonitor is created that selects
// the agent's Service. When disabled or nil, any existing ServiceMonitor is deleted.
// If prometheus-operator is not installed, the CRD will be absent and all errors are silently
// suppressed (meta.IsNoMatchError).
func (r *LanguageAgentReconciler) reconcileServiceMonitor(ctx context.Context, agent *langopv1alpha1.LanguageAgent) error {
	sm := &unstructured.Unstructured{}
	sm.SetGroupVersionKind(schema.GroupVersionKind{
		Group:   "monitoring.coreos.com",
		Version: "v1",
		Kind:    "ServiceMonitor",
	})
	sm.SetName(agent.Name)
	sm.SetNamespace(agent.Namespace)

	smSpec := agent.Spec.Monitoring
	if smSpec == nil || smSpec.ServiceMonitor == nil || !smSpec.ServiceMonitor.Enabled {
		if err := r.Delete(ctx, sm); err != nil && !apierrors.IsNotFound(err) && !meta.IsNoMatchError(err) {
			return fmt.Errorf("failed to delete ServiceMonitor: %w", err)
		}
		return nil
	}

	cfg := smSpec.ServiceMonitor
	port := cfg.Port
	if port == "" {
		port = "http"
		if len(agent.Spec.Ports) > 0 {
			port = agent.Spec.Ports[0].Name
		}
	}
	path := cfg.Path
	if path == "" {
		path = "/metrics"
	}

	endpoint := map[string]any{
		"port": port,
		"path": path,
	}
	if cfg.Interval != "" {
		endpoint["interval"] = cfg.Interval
	}
	if cfg.ScrapeTimeout != "" {
		endpoint["scrapeTimeout"] = cfg.ScrapeTimeout
	}

	labels := GetCommonLabels(agent.Name, "LanguageAgent")
	maps.Copy(labels, cfg.Labels)

	err := CreateOrUpdateOwned(ctx, r.Client, r.Scheme, agent, sm, func() error {
		sm.SetLabels(labels)
		return unstructured.SetNestedField(sm.Object, map[string]any{
			"selector": map[string]any{
				"matchLabels": map[string]any{
					LabelKeyK8sName: agent.Name,
				},
			},
			"endpoints": []any{endpoint},
		}, "spec")
	})
	if meta.IsNoMatchError(err) {
		return nil
	}
	return err
}

// reconcilePrometheusRule creates or deletes a Prometheus Operator PrometheusRule for the agent.
// When spec.monitoring.rules is non-empty, a PrometheusRule is created with the provided groups.
// When empty or nil, any existing PrometheusRule is deleted.
// If prometheus-operator is not installed the CRD will be absent and errors are silently suppressed.
func (r *LanguageAgentReconciler) reconcilePrometheusRule(ctx context.Context, agent *langopv1alpha1.LanguageAgent) error {
	pr := &unstructured.Unstructured{}
	pr.SetGroupVersionKind(schema.GroupVersionKind{
		Group:   "monitoring.coreos.com",
		Version: "v1",
		Kind:    "PrometheusRule",
	})
	pr.SetName(agent.Name)
	pr.SetNamespace(agent.Namespace)

	if agent.Spec.Monitoring == nil || len(agent.Spec.Monitoring.Rules) == 0 {
		if err := r.Delete(ctx, pr); err != nil && !apierrors.IsNotFound(err) && !meta.IsNoMatchError(err) {
			return fmt.Errorf("failed to delete PrometheusRule: %w", err)
		}
		return nil
	}

	groups := make([]any, 0, len(agent.Spec.Monitoring.Rules))
	for _, g := range agent.Spec.Monitoring.Rules {
		rules := make([]any, 0, len(g.Rules))
		for _, rule := range g.Rules {
			rm := map[string]any{
				"expr": rule.Expr,
			}
			if rule.Alert != "" {
				rm["alert"] = rule.Alert
			}
			if rule.Record != "" {
				rm["record"] = rule.Record
			}
			if rule.For != "" {
				rm["for"] = rule.For
			}
			if len(rule.Labels) > 0 {
				lm := make(map[string]any, len(rule.Labels))
				for k, v := range rule.Labels {
					lm[k] = v
				}
				rm["labels"] = lm
			}
			if len(rule.Annotations) > 0 {
				am := make(map[string]any, len(rule.Annotations))
				for k, v := range rule.Annotations {
					am[k] = v
				}
				rm["annotations"] = am
			}
			rules = append(rules, rm)
		}
		group := map[string]any{
			"name":  g.Name,
			"rules": rules,
		}
		if g.Interval != "" {
			group["interval"] = g.Interval
		}
		groups = append(groups, group)
	}

	labels := GetCommonLabels(agent.Name, "LanguageAgent")

	err := CreateOrUpdateOwned(ctx, r.Client, r.Scheme, agent, pr, func() error {
		pr.SetLabels(labels)
		return unstructured.SetNestedField(pr.Object, map[string]any{
			"groups": groups,
		}, "spec")
	})
	if meta.IsNoMatchError(err) {
		return nil
	}
	return err
}

// reconcileWebhooks creates an Ingress for webhook access
func (r *LanguageAgentReconciler) reconcileWebhooks(ctx context.Context, agent *langopv1alpha1.LanguageAgent) error {
	log := log.FromContext(ctx)

	// Get the cluster to check for domain configuration
	cluster := &langopv1alpha1.LanguageCluster{}
	if err := r.Get(ctx, types.NamespacedName{Name: agent.Namespace}, cluster); err != nil {
		if apierrors.IsNotFound(err) {
			log.Info("Cluster not found, skipping webhook reconciliation", "cluster", agent.Namespace)
			return nil
		}
		return err
	}

	// Skip webhook reconciliation if no domain is configured
	if cluster.Spec.Domain == "" {
		log.Info("No domain configured, skipping webhook reconciliation")
		return nil
	}

	// Build agent hostname: <agent-name>.<domain>
	hostname := fmt.Sprintf("%s.%s", agent.Name, cluster.Spec.Domain)

	log.Info("Creating Ingress for webhook", "hostname", hostname)
	if err := r.reconcileIngress(ctx, agent, hostname); err != nil {
		SetCondition(&agent.Status.Conditions, langopv1alpha1.ConditionWebhookRouteCreated, metav1.ConditionFalse, langopv1alpha1.ReasonIngressCreationFailed, err.Error(), agent.Generation)
		return fmt.Errorf("failed to reconcile Ingress: %w", err)
	}
	SetCondition(&agent.Status.Conditions, langopv1alpha1.ConditionWebhookRouteCreated, metav1.ConditionTrue, langopv1alpha1.ReasonIngressCreated, "Ingress created successfully", agent.Generation)

	var routeReady bool
	var routeReadyMsg string
	{
		ready, msg, err := r.checkIngressReadiness(ctx, agent.Name, agent.Namespace)
		if err != nil {
			log.Error(err, "Failed to check Ingress readiness")
			routeReadyMsg = fmt.Sprintf("Failed to check readiness: %v", err)
		} else {
			routeReady = ready
			routeReadyMsg = msg
		}
	}

	// Set WebhookRouteReady condition based on readiness check
	if routeReady {
		SetCondition(&agent.Status.Conditions, langopv1alpha1.ConditionWebhookRouteReady, metav1.ConditionTrue, langopv1alpha1.ReasonWebhookRouteReady, routeReadyMsg, agent.Generation)

		// Only populate WebhookURLs when route is ready
		webhookURL := fmt.Sprintf("https://%s", hostname)
		if agent.Status.WebhookURLs == nil || len(agent.Status.WebhookURLs) == 0 || agent.Status.WebhookURLs[0] != webhookURL {
			agent.Status.WebhookURLs = []string{webhookURL}
			log.Info("Updated webhook URL in status", "url", webhookURL)
		}
	} else {
		SetCondition(&agent.Status.Conditions, langopv1alpha1.ConditionWebhookRouteReady, metav1.ConditionFalse, langopv1alpha1.ReasonWebhookRouteNotReady, routeReadyMsg, agent.Generation)

		// Clear webhook URLs when route is not ready
		if len(agent.Status.WebhookURLs) > 0 {
			agent.Status.WebhookURLs = nil
			log.Info("Cleared webhook URLs from status - route not ready")
		}
	}

	return nil
}

// reconcileIngress creates or updates an Ingress for the agent
func (r *LanguageAgentReconciler) reconcileIngress(ctx context.Context, agent *langopv1alpha1.LanguageAgent, hostname string) error {
	labels := GetCommonLabels(agent.Name, "LanguageAgent")

	ingress := &networkingv1.Ingress{
		ObjectMeta: metav1.ObjectMeta{
			Name:      agent.Name,
			Namespace: agent.Namespace,
			Labels:    labels,
		},
	}

	err := CreateOrUpdateOwned(ctx, r.Client, r.Scheme, agent, ingress, func() error {
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
											Name: agent.Name,
											Port: networkingv1.ServiceBackendPort{
												Number: agentIngressPort(agent),
											},
										},
									},
								},
							},
						},
					},
				},
			},
		}

		// Add TLS and ClassName from cluster config (with operator-level defaults)
		ingressClass := r.DefaultIngressClassName
		{
			cluster := &langopv1alpha1.LanguageCluster{}
			if err := r.Get(ctx, types.NamespacedName{Name: agent.Namespace}, cluster); err == nil {
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
							annotationKey := "cert-manager.io/" + certManagerIssuerAnnotationSuffix(kind)
							ingress.Annotations[annotationKey] = r.DefaultTLSIssuerName
						}
						secretName = GenerateTLSSecretName(agent.Name)
					}
					ingress.Spec.TLS = []networkingv1.IngressTLS{
						{
							Hosts:      []string{hostname},
							SecretName: secretName,
						},
					}
				}

				// Cluster-level ClassName overrides operator default
				if cluster.Spec.Ingress != nil && cluster.Spec.Ingress.ClassName != "" {
					ingressClass = cluster.Spec.Ingress.ClassName
				}
			}
		}
		if ingressClass != "" {
			ingress.Spec.IngressClassName = &ingressClass
		}

		return nil
	})

	return err
}

// detectPodFailures checks for pod failures and updates agent status
// extractPodErrorInfo extracts error details and logs from a failed pod
// validateImageRegistry validates that the agent's container image registry is in the whitelist
func (r *LanguageAgentReconciler) validateImageRegistry(agent *langopv1alpha1.LanguageAgent) error {
	// Skip validation if no whitelist configured
	allowedRegistries := r.RegistryManager.GetRegistries()
	if len(allowedRegistries) == 0 {
		return nil
	}

	return validation.ValidateImageRegistry(agent.Spec.Image, allowedRegistries)
}

// checkIngressReadiness checks if an Ingress is ready to serve traffic
// Returns (isReady, statusMessage, error)
func (r *LanguageAgentReconciler) checkIngressReadiness(ctx context.Context, name, namespace string) (bool, string, error) {
	ingress := &networkingv1.Ingress{}
	err := r.Get(ctx, types.NamespacedName{Name: name, Namespace: namespace}, ingress)
	if err != nil {
		if apierrors.IsNotFound(err) {
			return false, "Ingress not found", nil
		}
		return false, "", fmt.Errorf("failed to get Ingress: %w", err)
	}

	// Check if load balancer is ready
	if len(ingress.Status.LoadBalancer.Ingress) == 0 {
		return false, "Ingress load balancer not ready - no ingress points assigned", nil
	}

	// Check if any ingress point has an IP or hostname
	for _, lbIngress := range ingress.Status.LoadBalancer.Ingress {
		if lbIngress.IP != "" || lbIngress.Hostname != "" {
			return true, "Ingress is ready with load balancer", nil
		}
	}

	return false, "Ingress load balancer assigned but no IP or hostname available", nil
}

// enqueueAgentsByRuntime returns a handler that lists all LanguageAgents across all
// namespaces that reference the changed LanguageAgentRuntime and enqueues them.
// LanguageAgentRuntime is cluster-scoped, so we list across all namespaces.
func (r *LanguageAgentReconciler) enqueueAgentsByRuntime() handler.MapFunc {
	return func(ctx context.Context, obj client.Object) []reconcile.Request {
		runtimeName := obj.GetName()
		agentList := &langopv1alpha1.LanguageAgentList{}
		if err := r.List(ctx, agentList); err != nil {
			return nil
		}
		var reqs []reconcile.Request
		for _, agent := range agentList.Items {
			if agent.Spec.Runtime == runtimeName {
				reqs = append(reqs, reconcile.Request{
					NamespacedName: types.NamespacedName{
						Name:      agent.Name,
						Namespace: agent.Namespace,
					},
				})
			}
		}
		return reqs
	}
}

// enqueueAgentsInNamespace returns a handler that lists all LanguageAgents in the
// same namespace as the changed object and enqueues a reconcile request for each.
func (r *LanguageAgentReconciler) enqueueAgentsInNamespace() handler.MapFunc {
	return func(ctx context.Context, obj client.Object) []reconcile.Request {
		agentList := &langopv1alpha1.LanguageAgentList{}
		if err := r.List(ctx, agentList, client.InNamespace(obj.GetNamespace())); err != nil {
			return nil
		}
		reqs := make([]reconcile.Request, len(agentList.Items))
		for i, agent := range agentList.Items {
			reqs[i] = reconcile.Request{
				NamespacedName: types.NamespacedName{
					Name:      agent.Name,
					Namespace: agent.Namespace,
				},
			}
		}
		return reqs
	}
}

// SetupWithManager sets up the controller with the Manager.
func (r *LanguageAgentReconciler) SetupWithManager(mgr ctrl.Manager, concurrency int) error {
	enqueue := r.enqueueAgentsInNamespace()
	return ctrl.NewControllerManagedBy(mgr).
		For(&langopv1alpha1.LanguageAgent{}).
		Owns(&appsv1.Deployment{}).
		Owns(&corev1.ConfigMap{}).
		Owns(&corev1.Service{}).
		Owns(&networkingv1.NetworkPolicy{}).
		Owns(&networkingv1.Ingress{}).
		Owns(&corev1.PersistentVolumeClaim{}).
		Owns(&policyv1.PodDisruptionBudget{}).
		Owns(&autoscalingv2.HorizontalPodAutoscaler{}).
		Watches(&langopv1alpha1.LanguageTool{}, handler.EnqueueRequestsFromMapFunc(enqueue)).
		Watches(&langopv1alpha1.LanguageModel{}, handler.EnqueueRequestsFromMapFunc(enqueue)).
		Watches(&langopv1alpha1.LanguagePersona{}, handler.EnqueueRequestsFromMapFunc(enqueue)).
		Watches(&langopv1alpha1.LanguageAgentRuntime{}, handler.EnqueueRequestsFromMapFunc(r.enqueueAgentsByRuntime())).
		WithOptions(controller.Options{MaxConcurrentReconciles: concurrency}).
		Complete(r)
}

// reconcileAgentServiceAccount ensures the ServiceAccount for agent pods exists with proper permissions
func (r *LanguageAgentReconciler) reconcileAgentServiceAccount(ctx context.Context, agent *langopv1alpha1.LanguageAgent) error {
	log := log.FromContext(ctx)

	// Skip if custom ServiceAccount is specified - assume it exists and has proper permissions
	if agent.Spec.Deployment.ServiceAccountName != "" {
		return nil
	}

	// ServiceAccount always lives in the agent's own namespace
	targetNamespace := agent.Namespace
	saName := GenerateServiceAccountName(agent.Name)

	// Create ServiceAccount
	serviceAccount := &corev1.ServiceAccount{
		ObjectMeta: metav1.ObjectMeta{
			Name:      saName,
			Namespace: targetNamespace,
		},
	}

	_, err := controllerutil.CreateOrUpdate(ctx, r.Client, serviceAccount, func() error {
		if serviceAccount.Labels == nil {
			serviceAccount.Labels = make(map[string]string)
		}
		serviceAccount.Labels[LabelKeyK8sName] = saName
		serviceAccount.Labels[LabelKeyK8sComponent] = "serviceaccount"
		serviceAccount.Labels[LabelKeyK8sManagedBy] = "language-operator"

		// Merge user-supplied annotations (e.g. IRSA, GCP WI, AKS WI)
		if len(agent.Spec.Deployment.ServiceAccountAnnotations) > 0 {
			if serviceAccount.Annotations == nil {
				serviceAccount.Annotations = make(map[string]string)
			}
			maps.Copy(serviceAccount.Annotations, agent.Spec.Deployment.ServiceAccountAnnotations)
		}

		return nil
	})

	if err != nil {
		return fmt.Errorf("failed to create/update ServiceAccount: %w", err)
	}

	// Create namespace-scoped Role with minimal permissions for agent pods
	role := &rbacv1.Role{
		ObjectMeta: metav1.ObjectMeta{
			Name:      saName,
			Namespace: targetNamespace,
		},
	}

	_, err = controllerutil.CreateOrUpdate(ctx, r.Client, role, func() error {
		if role.Labels == nil {
			role.Labels = make(map[string]string)
		}
		role.Labels[LabelKeyK8sName] = saName
		role.Labels[LabelKeyK8sComponent] = "role"
		role.Labels[LabelKeyK8sManagedBy] = "language-operator"
		role.Rules = []rbacv1.PolicyRule{
			{
				APIGroups: []string{""},
				Resources: []string{"configmaps"},
				Verbs:     []string{"get", "list"},
			},
			{
				APIGroups: []string{""},
				Resources: []string{"pods"},
				Verbs:     []string{"get", "list", "watch"},
			},
		}
		// When self-configure is enabled, grant the agent's SA permission to
		// create LanguageAgentSelfConfig requests targeting itself.
		if agent.Spec.SelfConfigure != nil && agent.Spec.SelfConfigure.Enabled != nil && *agent.Spec.SelfConfigure.Enabled {
			role.Rules = append(role.Rules, rbacv1.PolicyRule{
				APIGroups: []string{"langop.io"},
				Resources: []string{"languageagentselfconfigs"},
				Verbs:     []string{"create"},
			})
		}
		role.Rules = append(role.Rules, agent.Spec.Deployment.RoleRules...)
		return nil
	})

	if err != nil {
		return fmt.Errorf("failed to create/update Role: %w", err)
	}

	// Create namespace-scoped RoleBinding binding the ServiceAccount to the Role
	roleBinding := &rbacv1.RoleBinding{
		ObjectMeta: metav1.ObjectMeta{
			Name:      saName,
			Namespace: targetNamespace,
		},
	}

	_, err = controllerutil.CreateOrUpdate(ctx, r.Client, roleBinding, func() error {
		if roleBinding.Labels == nil {
			roleBinding.Labels = make(map[string]string)
		}
		roleBinding.Labels[LabelKeyK8sName] = saName
		roleBinding.Labels[LabelKeyK8sComponent] = "rolebinding"
		roleBinding.Labels[LabelKeyK8sManagedBy] = "language-operator"
		roleBinding.RoleRef = rbacv1.RoleRef{
			APIGroup: "rbac.authorization.k8s.io",
			Kind:     "Role",
			Name:     saName,
		}
		roleBinding.Subjects = []rbacv1.Subject{
			{
				Kind:      "ServiceAccount",
				Name:      saName,
				Namespace: targetNamespace,
			},
		}
		return nil
	})

	if err != nil {
		return fmt.Errorf("failed to create/update RoleBinding: %w", err)
	}

	log.Info("Reconciled agent ServiceAccount and permissions",
		"serviceAccount", saName,
		"namespace", targetNamespace,
		"role", role.Name,
		"roleBinding", roleBinding.Name)

	return nil
}

// getServiceAccountName returns the ServiceAccount name to use for agent pods
func (r *LanguageAgentReconciler) getServiceAccountName(agent *langopv1alpha1.LanguageAgent) string {
	// If explicitly specified in the agent spec, use that
	if agent.Spec.Deployment.ServiceAccountName != "" {
		return agent.Spec.Deployment.ServiceAccountName
	}

	// Default to an operator-managed per-agent ServiceAccount
	return GenerateServiceAccountName(agent.Name)
}

// generateCredential returns a cryptographically random 32-byte hex string.
func generateCredential() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}

// reconcileRuntimeSecret creates or updates a managed Secret containing credentials for
// runtime-specific configuration (opencode, openclaw). When inline values are provided,
// the operator owns the Secret and GC's it on agent deletion. When *Ref variants are used,
// the referenced Secret is injected into workingAgent's envFrom directly. When neither is
// set, credentials are auto-generated once and preserved on subsequent reconciles.
func (r *LanguageAgentReconciler) reconcileRuntimeSecret(
	ctx context.Context,
	agent *langopv1alpha1.LanguageAgent,
	workingAgent *langopv1alpha1.LanguageAgent,
) error {
	secretName := agent.Name + "-runtime"
	secretData := map[string][]byte{}
	var extraEnv []corev1.EnvVar
	var refEnvFrom []corev1.EnvFromSource

	// Load existing secret so auto-generated values can be preserved across reconciles.
	existing := &corev1.Secret{}
	existingData := map[string][]byte{}
	if err := r.Get(ctx, types.NamespacedName{Name: secretName, Namespace: agent.Namespace}, existing); err == nil {
		existingData = existing.Data
	}

	// opencode inline credentials → managed secret
	// Use workingAgent so runtime-provided config (e.g. spec.opencode from a LanguageAgentRuntime) is respected.
	if workingAgent.Spec.Opencode != nil && workingAgent.Spec.Opencode.Enabled != nil && *workingAgent.Spec.Opencode.Enabled {
		oc := workingAgent.Spec.Opencode
		if oc.Password != "" {
			username := oc.Username
			if username == "" {
				username = "opencode"
			}
			secretData["OPENCODE_SERVER_USERNAME"] = []byte(username)
			secretData["OPENCODE_SERVER_PASSWORD"] = []byte(oc.Password)
		} else if oc.PasswordRef != nil {
			// Username as literal env var if specified alongside a ref
			if oc.Username != "" {
				extraEnv = append(extraEnv, corev1.EnvVar{
					Name:  "OPENCODE_SERVER_USERNAME",
					Value: oc.Username,
				})
			}
			refEnvFrom = append(refEnvFrom, corev1.EnvFromSource{
				SecretRef: &corev1.SecretEnvSource{
					LocalObjectReference: corev1.LocalObjectReference{Name: oc.PasswordRef.Name},
				},
			})
		} else {
			// Auto-generate: preserve existing value if present, otherwise generate new.
			password := string(existingData["OPENCODE_SERVER_PASSWORD"])
			if password == "" {
				var err error
				password, err = generateCredential()
				if err != nil {
					return fmt.Errorf("generating opencode password: %w", err)
				}
			}
			username := oc.Username
			if username == "" {
				username = "opencode"
			}
			secretData["OPENCODE_SERVER_USERNAME"] = []byte(username)
			secretData["OPENCODE_SERVER_PASSWORD"] = []byte(password)
		}
	}

	// openclaw inline credentials → managed secret
	if workingAgent.Spec.Openclaw != nil && workingAgent.Spec.Openclaw.Enabled != nil && *workingAgent.Spec.Openclaw.Enabled {
		oc := workingAgent.Spec.Openclaw
		if oc.Token != "" {
			secretData["OPENCLAW_GATEWAY_TOKEN"] = []byte(oc.Token)
		} else if oc.TokenRef != nil {
			refEnvFrom = append(refEnvFrom, corev1.EnvFromSource{
				SecretRef: &corev1.SecretEnvSource{
					LocalObjectReference: corev1.LocalObjectReference{Name: oc.TokenRef.Name},
				},
			})
		} else {
			// Auto-generate: preserve existing value if present, otherwise generate new.
			token := string(existingData["OPENCLAW_GATEWAY_TOKEN"])
			if token == "" {
				var err error
				token, err = generateCredential()
				if err != nil {
					return fmt.Errorf("generating openclaw token: %w", err)
				}
			}
			secretData["OPENCLAW_GATEWAY_TOKEN"] = []byte(token)
		}
	}

	// claude-code credentials → managed secret, ref envFrom, or gateway-routed placeholder
	if workingAgent.Spec.ClaudeCode != nil && workingAgent.Spec.ClaudeCode.Enabled != nil && *workingAgent.Spec.ClaudeCode.Enabled {
		cc := workingAgent.Spec.ClaudeCode
		if cc.APIKey != "" {
			secretData["ANTHROPIC_API_KEY"] = []byte(cc.APIKey)
		} else if cc.APIKeyRef != nil {
			refEnvFrom = append(refEnvFrom, corev1.EnvFromSource{
				SecretRef: &corev1.SecretEnvSource{
					LocalObjectReference: corev1.LocalObjectReference{Name: cc.APIKeyRef.Name},
				},
			})
		} else {
			// Gateway-routed mode: inject placeholder key and route SDK traffic to the
			// LiteLLM gateway via ANTHROPIC_BASE_URL so calls never reach api.anthropic.com.
			extraEnv = append(extraEnv, corev1.EnvVar{
				Name:  "ANTHROPIC_API_KEY",
				Value: "sk-langop-proxy",
			})
			extraEnv = append(extraEnv, corev1.EnvVar{
				Name:  "ANTHROPIC_BASE_URL",
				Value: serviceURL("gateway", agent.Namespace, GatewayServicePort),
			})
		}
		if cc.MaxTurns != nil {
			extraEnv = append(extraEnv, corev1.EnvVar{
				Name:  "CLAUDE_CODE_MAX_TURNS",
				Value: fmt.Sprintf("%d", *cc.MaxTurns),
			})
		}
	}

	if len(secretData) > 0 {
		secret := &corev1.Secret{
			ObjectMeta: metav1.ObjectMeta{
				Name:      secretName,
				Namespace: agent.Namespace,
			},
		}
		if err := CreateOrUpdateOwned(ctx, r.Client, r.Scheme, agent, secret, func() error {
			secret.Data = secretData
			return nil
		}); err != nil {
			return fmt.Errorf("reconciling runtime secret %s/%s: %w", agent.Namespace, secretName, err)
		}
		// Prepend managed secret to envFrom so it takes precedence
		workingAgent.Spec.Deployment.EnvFrom = append(
			[]corev1.EnvFromSource{{SecretRef: &corev1.SecretEnvSource{
				LocalObjectReference: corev1.LocalObjectReference{Name: secretName},
			}}},
			workingAgent.Spec.Deployment.EnvFrom...,
		)
	} else {
		// No inline credentials — delete managed secret if it exists
		secret := &corev1.Secret{}
		err := r.Get(ctx, types.NamespacedName{Name: secretName, Namespace: agent.Namespace}, secret)
		if err == nil {
			if err := r.Delete(ctx, secret); err != nil && !apierrors.IsNotFound(err) {
				return fmt.Errorf("deleting stale runtime secret %s/%s: %w", agent.Namespace, secretName, err)
			}
		}
	}

	// Append ref-based envFrom entries
	workingAgent.Spec.Deployment.EnvFrom = append(workingAgent.Spec.Deployment.EnvFrom, refEnvFrom...)
	// Prepend any literal env vars (e.g. username alongside a passwordRef)
	workingAgent.Spec.Deployment.Env = append(extraEnv, workingAgent.Spec.Deployment.Env...)

	return nil
}

// managedResourcesEqual returns true when a and b contain the same entries in the same order.
// ManagedResource is a pure-value struct (all string fields), so direct struct comparison is valid.
func managedResourcesEqual(a, b []langopv1alpha1.ManagedResource) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

// buildAgentManagedResources returns the inventory of Kubernetes resources managed by this
// controller on behalf of agent. workingAgent carries the merged spec (after runtime resolution).
// clusterDomain is cluster.Spec.Domain, empty if the cluster has no domain configured.
func (r *LanguageAgentReconciler) buildAgentManagedResources(
	agent *langopv1alpha1.LanguageAgent,
	workingAgent *langopv1alpha1.LanguageAgent,
	clusterDomain string,
) []langopv1alpha1.ManagedResource {
	ns := agent.Namespace
	resources := []langopv1alpha1.ManagedResource{
		{Group: "apps", Kind: "Deployment", Name: agent.Name, Namespace: ns},
		{Kind: "Service", Name: agent.Name, Namespace: ns},
		{Kind: "ConfigMap", Name: GenerateConfigMapName(agent.Name, "agent"), Namespace: ns},
	}

	if r.NetworkIsolationEnabled {
		resources = append(resources, langopv1alpha1.ManagedResource{
			Group: "networking.k8s.io", Kind: "NetworkPolicy", Name: agent.Name, Namespace: ns,
		})
	}

	// PVC is created when workspace is nil (defaults enabled) or explicitly enabled.
	ws := workingAgent.Spec.Workspace
	if ws == nil || ws.Enabled == nil || *ws.Enabled {
		resources = append(resources, langopv1alpha1.ManagedResource{
			Kind: "PersistentVolumeClaim", Name: GeneratePVCName(agent.Name), Namespace: ns,
		})
	}

	// Runtime Secret is managed when opencode/openclaw/claude-code is enabled and configured
	// with inline credentials rather than a *Ref pointing at an existing Secret.
	hasSecret := (workingAgent.Spec.Opencode != nil && workingAgent.Spec.Opencode.Enabled != nil && *workingAgent.Spec.Opencode.Enabled && workingAgent.Spec.Opencode.PasswordRef == nil) ||
		(workingAgent.Spec.Openclaw != nil && workingAgent.Spec.Openclaw.Enabled != nil && *workingAgent.Spec.Openclaw.Enabled && workingAgent.Spec.Openclaw.TokenRef == nil) ||
		(workingAgent.Spec.ClaudeCode != nil && workingAgent.Spec.ClaudeCode.Enabled != nil && *workingAgent.Spec.ClaudeCode.Enabled && workingAgent.Spec.ClaudeCode.APIKey != "")
	if hasSecret {
		resources = append(resources, langopv1alpha1.ManagedResource{
			Kind: "Secret", Name: agent.Name + "-runtime", Namespace: ns,
		})
	}

	// Ingress is created when the cluster has a domain configured.
	if clusterDomain != "" {
		resources = append(resources, langopv1alpha1.ManagedResource{
			Group: "networking.k8s.io", Kind: "Ingress", Name: agent.Name, Namespace: ns,
		})
	}

	// Per-agent RBAC resources are created when no custom ServiceAccount is specified.
	if agent.Spec.Deployment.ServiceAccountName == "" {
		saName := GenerateServiceAccountName(agent.Name)
		resources = append(resources,
			langopv1alpha1.ManagedResource{Kind: "ServiceAccount", Name: saName, Namespace: ns},
			langopv1alpha1.ManagedResource{Group: "rbac.authorization.k8s.io", Kind: "Role", Name: saName, Namespace: ns},
			langopv1alpha1.ManagedResource{Group: "rbac.authorization.k8s.io", Kind: "RoleBinding", Name: saName, Namespace: ns},
		)
	}

	// PodDisruptionBudget is created only when replicas > 1.
	replicas := int32(1)
	if workingAgent.Spec.Deployment.Replicas != nil {
		replicas = *workingAgent.Spec.Deployment.Replicas
	}
	if replicas > 1 {
		resources = append(resources, langopv1alpha1.ManagedResource{
			Group: "policy", Kind: "PodDisruptionBudget", Name: agent.Name, Namespace: ns,
		})
	}

	// HorizontalPodAutoscaler is created when autoscaling is configured.
	if workingAgent.Spec.Deployment.Autoscaling != nil {
		resources = append(resources, langopv1alpha1.ManagedResource{
			Group: "autoscaling", Kind: "HorizontalPodAutoscaler", Name: agent.Name, Namespace: ns,
		})
	}

	// ServiceMonitor is created when monitoring.serviceMonitor.enabled is true.
	if workingAgent.Spec.Monitoring != nil &&
		workingAgent.Spec.Monitoring.ServiceMonitor != nil &&
		workingAgent.Spec.Monitoring.ServiceMonitor.Enabled {
		resources = append(resources, langopv1alpha1.ManagedResource{
			Group: "monitoring.coreos.com", Kind: "ServiceMonitor", Name: agent.Name, Namespace: ns,
		})
	}

	// PrometheusRule is created when monitoring.rules is non-empty.
	if workingAgent.Spec.Monitoring != nil && len(workingAgent.Spec.Monitoring.Rules) > 0 {
		resources = append(resources, langopv1alpha1.ManagedResource{
			Group: "monitoring.coreos.com", Kind: "PrometheusRule", Name: agent.Name, Namespace: ns,
		})
	}

	return resources
}

// resolveCodeConfigMapName resolves the ConfigMap name for agent code based on AgentVersionRef
