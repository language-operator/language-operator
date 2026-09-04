package controllers

import (
	"context"
	"fmt"
	"strings"
	"time"

	wfv1 "github.com/argoproj/argo-workflows/v4/pkg/apis/workflow/v1alpha1"
	"github.com/go-logr/logr"
	"go.opentelemetry.io/otel/codes"
	corev1 "k8s.io/api/core/v1"
	networkingv1 "k8s.io/api/networking/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/tools/record"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	langopv1alpha1 "github.com/language-operator/language-operator/api/v1alpha1"
	"github.com/language-operator/language-operator/pkg/cni"
	"github.com/language-operator/language-operator/pkg/events"
	"github.com/language-operator/language-operator/pkg/merge"
	"github.com/language-operator/language-operator/pkg/reconciler"
	"github.com/language-operator/language-operator/pkg/validation"
)

// LanguageAgentReconciler reconciles a LanguageAgent object
type LanguageAgentReconciler struct {
	client.Client
	Scheme                     *runtime.Scheme
	Log                        logr.Logger
	Recorder                   record.EventRecorder
	EventManager               *events.EventManager
	RegistryManager            validation.RegistryManager
	NetworkPolicyTimeout       time.Duration
	NetworkPolicyRetries       int
	NetworkIsolationEnabled    bool
	DefaultIngressClassName    string
	DefaultStorageClassName    string
	DefaultTLSIssuerName       string
	DefaultTLSIssuerKind       string
	IngressControllerNamespace string
	OAuth2ProxyImage           string
	CNICapabilities            *cni.CNICapabilities
	// MCPBridgeImage is the stdio→Streamable-HTTP bridge image injected for transport=stdio
	// sidecar tools. When empty, langopv1alpha1.DefaultMCPBridgeImage is used.
	MCPBridgeImage string
	// MCPBridgeImagePullPolicy is the pull policy for the injected bridge image.
	MCPBridgeImagePullPolicy corev1.PullPolicy
}

//+kubebuilder:rbac:groups=langop.io,resources=languageagents,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=langop.io,resources=languageagents/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=langop.io,resources=languageagents/finalizers,verbs=update
//+kubebuilder:rbac:groups=langop.io,resources=languagepersonas,verbs=get;list;watch
//+kubebuilder:rbac:groups=langop.io,resources=languageclusters,verbs=get;list;watch
//+kubebuilder:rbac:groups=argoproj.io,resources=workflows;workflowtemplates;cronworkflows,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=argoproj.io,resources=workflows/status;cronworkflows/status,verbs=get
// The operator grants workflowtaskresults to each agent's ServiceAccount so the Argo
// executor can report node outcomes. RBAC forbids granting a permission the granter
// does not hold, so the operator must carry it too.
//+kubebuilder:rbac:groups=argoproj.io,resources=workflowtaskresults,verbs=create;patch
//+kubebuilder:rbac:groups=apps,resources=daemonsets,verbs=get;list;watch
//+kubebuilder:rbac:groups="",resources=configmaps,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups="",resources=persistentvolumeclaims,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups="",resources=services,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups="",resources=pods,verbs=get;list;watch
//+kubebuilder:rbac:groups="",resources=pods/log,verbs=get
// Read to discover the API server's real listening port when building agent egress rules.
//+kubebuilder:rbac:groups="",resources=endpoints,verbs=get;list;watch
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
			SetPhase(&agent.Status.Phase, &agent.Status.ObservedGeneration, events.PhaseStatusFailed, agent.Generation)
			reconcileErr = err
			return ctrl.Result{}, err
		}
		effectiveSpec := agent.Spec.DeepCopy()
		merge.ApplyRuntimeDefaults(effectiveSpec, &rt.Spec)
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
		r.EventManager.RecordRegistryValidationFailed(agent, agent.Spec.Image)
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

	// Repository clone is performed in-pod by the "repository" init container (clone-once).
	// The condition reflects that the clone is configured, mirroring WorkspaceSeeded.
	if agentHasRepository(workingAgent) {
		SetCondition(&agent.Status.Conditions, langopv1alpha1.ConditionRepositoryCloned, metav1.ConditionTrue, langopv1alpha1.ReasonRepositoryConfigured,
			fmt.Sprintf("Repository clone configured: %s", workingAgent.Spec.Repository.URL), agent.Generation)
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
				r.EventManager.RecordNetworkPolicyTimeout(agent)

				// Don't fail the entire reconciliation for timeout - continue with degraded state
			} else {
				// For non-timeout errors, fail the reconciliation
				span.SetStatus(codes.Error, "NetworkPolicy reconciliation failed")
				SetCondition(&agent.Status.Conditions, langopv1alpha1.ConditionReady, metav1.ConditionFalse, langopv1alpha1.ReasonNetworkPolicyError, err.Error(), agent.Generation)
				SetPhase(&agent.Status.Phase, &agent.Status.ObservedGeneration, events.PhaseStatusFailed, agent.Generation)
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
			r.EventManager.RecordNetworkPolicyUnsupported(agent, cniName)
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

	// Use the agent's Kubernetes UID as the stable agent UUID — it's assigned at creation,
	// never changes, and is always available without a status round-trip.
	agent.Status.UUID = string(agent.UID)
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

	// The WorkflowTemplate is the agent's podspec and the unit a manual run is
	// submitted against; it exists in both execution modes.
	if err := r.reconcileWorkflowTemplate(ctx, workingAgent, configHash); err != nil {
		log.Error(err, "Failed to reconcile WorkflowTemplate")
		span.RecordError(err)
		span.SetStatus(codes.Error, "WorkflowTemplate reconciliation failed")
		SetCondition(&agent.Status.Conditions, langopv1alpha1.ConditionReady, metav1.ConditionFalse, langopv1alpha1.ReasonWorkflowError, err.Error(), agent.Generation)
		agent.Status.Phase = events.PhaseStatusFailed
		reconcileErr = err
		return ctrl.Result{}, err
	}
	agent.Status.WorkflowTemplateName = agent.Name

	// Then whatever the execution mode derives from it: an always-on Workflow for
	// a service agent, a CronWorkflow for a scheduled task. Both reconcilers also
	// remove the object belonging to the other mode, so flipping mode is clean.
	if workingAgent.Spec.Execution.EffectiveExecutionMode() == langopv1alpha1.ExecutionModeService {
		if err := r.reconcileCronWorkflow(ctx, workingAgent, configHash); err != nil {
			log.Error(err, "Failed to remove CronWorkflow")
			span.RecordError(err)
			reconcileErr = err
			return ctrl.Result{}, err
		}
		if err := r.reconcileServiceWorkflow(ctx, workingAgent, configHash); err != nil {
			log.Error(err, "Failed to reconcile agent Workflow")
			span.RecordError(err)
			span.SetStatus(codes.Error, "Workflow reconciliation failed")
			SetCondition(&agent.Status.Conditions, langopv1alpha1.ConditionReady, metav1.ConditionFalse, langopv1alpha1.ReasonWorkflowError, err.Error(), agent.Generation)
			agent.Status.Phase = events.PhaseStatusFailed
			reconcileErr = err
			return ctrl.Result{}, err
		}
	} else {
		if err := r.deleteWorkflowIfExists(ctx, agent.Name, agent.Namespace); err != nil {
			log.Error(err, "Failed to remove service Workflow")
			span.RecordError(err)
			reconcileErr = err
			return ctrl.Result{}, err
		}
		if err := r.reconcileCronWorkflow(ctx, workingAgent, configHash); err != nil {
			log.Error(err, "Failed to reconcile CronWorkflow")
			span.RecordError(err)
			span.SetStatus(codes.Error, "CronWorkflow reconciliation failed")
			SetCondition(&agent.Status.Conditions, langopv1alpha1.ConditionReady, metav1.ConditionFalse, langopv1alpha1.ReasonWorkflowError, err.Error(), agent.Generation)
			agent.Status.Phase = events.PhaseStatusFailed
			reconcileErr = err
			return ctrl.Result{}, err
		}
	}

	if err := r.reconcileServiceMonitor(ctx, workingAgent); err != nil {
		log.Error(err, "Failed to reconcile ServiceMonitor")
		span.RecordError(err)
		span.SetStatus(codes.Error, "ServiceMonitor reconciliation failed")
		SetCondition(&agent.Status.Conditions, langopv1alpha1.ConditionReady, metav1.ConditionFalse, langopv1alpha1.ReasonWorkflowError, err.Error(), agent.Generation)
		agent.Status.Phase = events.PhaseStatusFailed
		reconcileErr = err
		return ctrl.Result{}, err
	}

	if err := r.reconcilePrometheusRule(ctx, workingAgent); err != nil {
		log.Error(err, "Failed to reconcile PrometheusRule")
		span.RecordError(err)
		span.SetStatus(codes.Error, "PrometheusRule reconciliation failed")
		SetCondition(&agent.Status.Conditions, langopv1alpha1.ConditionReady, metav1.ConditionFalse, langopv1alpha1.ReasonWorkflowError, err.Error(), agent.Generation)
		agent.Status.Phase = events.PhaseStatusFailed
		reconcileErr = err
		return ctrl.Result{}, err
	}

	// Update status only if something changed
	statusChanged := false

	// Derive phase and run history from Argo Workflow state.
	newPhase, runStatusChanged, err := r.syncWorkflowStatus(ctx, agent, workingAgent)
	if runStatusChanged {
		statusChanged = true
	}
	if err != nil {
		log.Error(err, "Failed to read Workflow status")
		span.RecordError(err)
		reconcileErr = err
		return ctrl.Result{}, err
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
		SetPhase(&agent.Status.Phase, &agent.Status.ObservedGeneration, newPhase, agent.Generation)
		statusChanged = true
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
		Owns(&wfv1.WorkflowTemplate{}).
		Owns(&wfv1.Workflow{}).
		Owns(&wfv1.CronWorkflow{}).
		Owns(&corev1.ConfigMap{}).
		Owns(&corev1.Service{}).
		Owns(&networkingv1.NetworkPolicy{}).
		Owns(&networkingv1.Ingress{}).
		Owns(&corev1.PersistentVolumeClaim{}).
		Watches(&langopv1alpha1.LanguageTool{}, handler.EnqueueRequestsFromMapFunc(enqueue)).
		Watches(&langopv1alpha1.LanguageModel{}, handler.EnqueueRequestsFromMapFunc(enqueue)).
		Watches(&langopv1alpha1.LanguagePersona{}, handler.EnqueueRequestsFromMapFunc(enqueue)).
		Watches(&langopv1alpha1.LanguageAgentRuntime{}, handler.EnqueueRequestsFromMapFunc(r.enqueueAgentsByRuntime())).
		WithOptions(controller.Options{MaxConcurrentReconciles: concurrency}).
		Complete(r)
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
		{Group: "argoproj.io", Kind: "WorkflowTemplate", Name: agent.Name, Namespace: ns},
		{Kind: "ConfigMap", Name: GenerateConfigMapName(agent.Name, "agent"), Namespace: ns},
	}

	// A service agent is backed by a long-lived Workflow and is addressable; a
	// scheduled task agent by a CronWorkflow. An unscheduled task agent has only
	// the WorkflowTemplate, invoked by hand.
	if agentAddressable(workingAgent) {
		resources = append(resources,
			langopv1alpha1.ManagedResource{Group: "argoproj.io", Kind: "Workflow", Name: agent.Name, Namespace: ns},
			langopv1alpha1.ManagedResource{Kind: "Service", Name: agent.Name, Namespace: ns},
		)
	} else if workingAgent.Spec.Execution.Schedule != "" {
		resources = append(resources, langopv1alpha1.ManagedResource{
			Group: "argoproj.io", Kind: "CronWorkflow", Name: agent.Name, Namespace: ns,
		})
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

	// Runtime Secret is managed when any credential is inline or auto-generated
	// (i.e. lacks a ValueFrom pointing at an existing Secret).
	hasSecret := false
	for _, c := range workingAgent.Spec.Credentials {
		if c.ValueFrom == nil {
			hasSecret = true
			break
		}
	}
	if hasSecret {
		resources = append(resources, langopv1alpha1.ManagedResource{
			Kind: "Secret", Name: agent.Name + "-runtime", Namespace: ns,
		})
	}

	// Ingress is created when the agent is addressable and the cluster has a domain.
	if agentAddressable(workingAgent) && clusterDomain != "" {
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
