package controllers

import (
	"context"
	"crypto/sha256"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/go-logr/logr"
	"github.com/google/uuid"
	"go.opentelemetry.io/otel/codes"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	networkingv1 "k8s.io/api/networking/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
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
	Scheme                  *runtime.Scheme
	Log                     logr.Logger
	Recorder                record.EventRecorder
	EventManager            *events.EventManager
	RegistryManager         RegistryManager
	NetworkPolicyTimeout    time.Duration
	NetworkPolicyRetries    int
	NetworkIsolationEnabled bool
	DefaultIngressClassName string
	CNICapabilities         *cni.CNICapabilities
}

// agentConfigYAML is the structure marshaled into /etc/agent/config.yaml.
type agentConfigYAML struct {
	Agent        agentIdentityYAML          `yaml:"agent"`
	Instructions string                     `yaml:"instructions,omitempty"`
	Personas     []personaConfigYAML        `yaml:"personas,omitempty"`
	Tools        map[string]toolConfigYAML  `yaml:"tools,omitempty"`
	Models       map[string]modelConfigYAML `yaml:"models,omitempty"`
}

type agentIdentityYAML struct {
	Name      string `yaml:"name"`
	Namespace string `yaml:"namespace"`
}

type personaConfigYAML struct {
	Name        string `yaml:"name"`
	Tone        string `yaml:"tone,omitempty"`
	Personality string `yaml:"personality,omitempty"`
	Expertise   string `yaml:"expertise,omitempty"`
}

type toolConfigYAML struct {
	Endpoint string `yaml:"endpoint"`
	Protocol string `yaml:"protocol"`
}

type modelConfigYAML struct {
	Role     string `yaml:"role,omitempty"`
	Provider string `yaml:"provider"`
	Model    string `yaml:"model"`
	Endpoint string `yaml:"endpoint"`
	Priority *int32 `yaml:"priority,omitempty"`
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
//+kubebuilder:rbac:groups="",resources=serviceaccounts,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=rbac.authorization.k8s.io,resources=roles,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=rbac.authorization.k8s.io,resources=rolebindings,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=rbac.authorization.k8s.io,resources=clusterrolebindings,verbs=get;delete
//+kubebuilder:rbac:groups=langop.io,resources=languagetools,verbs=get;list;watch
//+kubebuilder:rbac:groups=langop.io,resources=languagemodels,verbs=get;list;watch

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
			// Shared RBAC resources (ServiceAccount, Role, RoleBinding named "language-agent")
			// have no owner reference and are not GC'd automatically; clean them up explicitly
			// when the last agent in the namespace is deleted.
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

	// Validate image registry against whitelist
	if err := r.validateImageRegistry(agent); err != nil {
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
	if err := r.reconcileConfigMap(ctx, agent); err != nil {
		log.Error(err, "Failed to reconcile ConfigMap")
		span.RecordError(err)
		span.SetStatus(codes.Error, "ConfigMap reconciliation failed")
		SetCondition(&agent.Status.Conditions, langopv1alpha1.ConditionReady, metav1.ConditionFalse, langopv1alpha1.ReasonConfigMapError, err.Error(), agent.Generation)
		agent.Status.Phase = events.PhaseStatusFailed
		reconcileErr = err
		return ctrl.Result{}, err
	}

	// Reconcile PVC for workspace if enabled
	if err := r.reconcilePVC(ctx, agent); err != nil {
		log.Error(err, "Failed to reconcile PVC")
		span.RecordError(err)
		span.SetStatus(codes.Error, "PVC reconciliation failed")
		SetCondition(&agent.Status.Conditions, langopv1alpha1.ConditionReady, metav1.ConditionFalse, langopv1alpha1.ReasonPVCError, err.Error(), agent.Generation)
		agent.Status.Phase = events.PhaseStatusFailed
		reconcileErr = err
		return ctrl.Result{}, err
	}

	// Reconcile NetworkPolicy for network isolation (if enabled)
	if r.NetworkIsolationEnabled {
		if err := r.reconcileNetworkPolicy(ctx, agent); err != nil {
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

	// Reconcile Service for agent webhook server (all agents expose port 8080)
	if err := r.reconcileService(ctx, agent); err != nil {
		log.Error(err, "Failed to reconcile Service")
		span.RecordError(err)
		span.SetStatus(codes.Error, "Service reconciliation failed")
		SetCondition(&agent.Status.Conditions, langopv1alpha1.ConditionReady, metav1.ConditionFalse, langopv1alpha1.ReasonServiceError, err.Error(), agent.Generation)
		agent.Status.Phase = events.PhaseStatusFailed
		reconcileErr = err
		return ctrl.Result{}, err
	}

	// Reconcile webhooks (Ingress for webhook access)
	if err := r.reconcileWebhooks(ctx, agent); err != nil {
		// Log webhook errors but don't fail reconciliation if domain not configured
		log.Info("Webhook reconciliation skipped or pending", "reason", err.Error())
		SetCondition(&agent.Status.Conditions, langopv1alpha1.ConditionWebhooksReady, metav1.ConditionFalse, langopv1alpha1.ReasonPending, err.Error(), agent.Generation)
	} else {
		SetCondition(&agent.Status.Conditions, langopv1alpha1.ConditionWebhooksReady, metav1.ConditionTrue, langopv1alpha1.ReasonConfigured, "Webhook routing configured", agent.Generation)
	}

	// Reconcile ServiceAccount for agent pods
	if err := r.reconcileAgentServiceAccount(ctx, agent); err != nil {
		log.Error(err, "Failed to reconcile agent ServiceAccount")
		span.RecordError(err)
		span.SetStatus(codes.Error, "ServiceAccount reconciliation failed")
		SetCondition(&agent.Status.Conditions, langopv1alpha1.ConditionReady, metav1.ConditionFalse, langopv1alpha1.ReasonServiceAccountError, err.Error(), agent.Generation)
		agent.Status.Phase = events.PhaseStatusFailed
		reconcileErr = err
		return ctrl.Result{}, err
	}

	if err := r.reconcileDeployment(ctx, agent); err != nil {
		log.Error(err, "Failed to reconcile Deployment")
		span.RecordError(err)
		span.SetStatus(codes.Error, "Deployment reconciliation failed")
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

		newPhase := events.PhaseStatusPending
		if existingDeploy.Status.ReadyReplicas > 0 {
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

func (r *LanguageAgentReconciler) reconcileConfigMap(ctx context.Context, agent *langopv1alpha1.LanguageAgent) error {
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
		endpoint := fmt.Sprintf("http://%s.%s.svc.cluster.local:%d", tool.Name, agent.Namespace, port)
		if tool.Spec.DeploymentMode == "sidecar" {
			endpoint = fmt.Sprintf("http://localhost:%d", port)
		}
		if cfg.Tools == nil {
			cfg.Tools = make(map[string]toolConfigYAML)
		}
		cfg.Tools[tool.Name] = toolConfigYAML{Endpoint: endpoint, Protocol: "mcp"}
	}

	// Models — all served via the shared namespace gateway
	gatewayURL := fmt.Sprintf("http://gateway.%s.svc.cluster.local:8000", agent.Namespace)
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
		return fmt.Errorf("failed to marshal config.yaml: %w", err)
	}

	data := map[string]string{
		"config.yaml": string(configYAMLBytes),
	}

	configMapName := GenerateConfigMapName(agent.Name, "agent")
	return CreateOrUpdateConfigMap(ctx, r.Client, r.Scheme, agent, configMapName, agent.Namespace, data)
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
			Name:      agent.Name + "-workspace",
			Namespace: targetNamespace,
			Labels:    GetCommonLabels(agent.Name, "LanguageAgent"),
		},
	}

	_, err := controllerutil.CreateOrUpdate(ctx, r.Client, pvc, func() error {
		if err := controllerutil.SetControllerReference(agent, pvc, r.Scheme); err != nil {
			return err
		}

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
			}
		}

		return nil
	})

	return err
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
					ClaimName: agent.Name + "-workspace",
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

func (r *LanguageAgentReconciler) reconcileDeployment(ctx context.Context, agent *langopv1alpha1.LanguageAgent) error {
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

	_, err = controllerutil.CreateOrUpdate(ctx, r.Client, deployment, func() error {
		if err := controllerutil.SetControllerReference(agent, deployment, r.Scheme); err != nil {
			return err
		}

		replicas := int32(1)
		if agent.Spec.Deployment.Replicas != nil {
			replicas = *agent.Spec.Deployment.Replicas
		}

		// Build container list starting with the agent
		containers := []corev1.Container{
			{
				Name:            "agent",
				Image:           agent.Spec.Image,
				ImagePullPolicy: agent.Spec.Deployment.ImagePullPolicy,
				Env:             r.buildAgentEnv(ctx, agent, cluster, modelURLs, modelNames, toolURLs),
				EnvFrom:         agent.Spec.Deployment.EnvFrom,
				Resources:       agent.Spec.Deployment.Resources,
				LivenessProbe:   agent.Spec.Deployment.LivenessProbe,
				ReadinessProbe:  agent.Spec.Deployment.ReadinessProbe,
				StartupProbe:    agent.Spec.Deployment.StartupProbe,
			},
		}

		// Inject operator-managed env vars into user-specified init containers.
		// Per spec/agents.md, all contracted env vars must be present in every
		// container including init containers.
		userInitContainers := make([]corev1.Container, len(agent.Spec.Deployment.InitContainers))
		copy(userInitContainers, agent.Spec.Deployment.InitContainers)
		agentEnv := r.buildAgentEnv(ctx, agent, cluster, modelURLs, modelNames, toolURLs)
		for i := range userInitContainers {
			userInitContainers[i].Env = append(agentEnv, userInitContainers[i].Env...)
		}

		// User-specified init containers run before operator-managed sidecar containers
		allInitContainers := append(userInitContainers, sidecarContainers...)

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

		deployment.Spec = appsv1.DeploymentSpec{
			Replicas: &replicas,
			Selector: &metav1.LabelSelector{
				MatchLabels: labels,
			},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels:      podLabels,
					Annotations: agent.Spec.Deployment.PodAnnotations,
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

	// Add ingress rules to allow trigger pods and dashboard to connect to agent
	port := agentPort(agent)
	agentPortIntStr := intstr.IntOrString{Type: intstr.Int, IntVal: port}
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
			Ports: []networkingv1.NetworkPolicyPort{
				{
					Protocol: protocolPtr(corev1.ProtocolTCP),
					Port:     &agentPortIntStr,
				},
			},
		},
		{
			// Allow dashboard pods from language-operator namespace to connect
			From: []networkingv1.NetworkPolicyPeer{
				{
					NamespaceSelector: &metav1.LabelSelector{
						MatchLabels: map[string]string{
							LabelKeyMetadataName: "language-operator",
						},
					},
					PodSelector: &metav1.LabelSelector{
						MatchLabels: map[string]string{
							"app.kubernetes.io/name": "language-operator-dashboard",
						},
					},
				},
			},
			Ports: []networkingv1.NetworkPolicyPort{
				{
					Protocol: protocolPtr(corev1.ProtocolTCP),
					Port:     &agentPortIntStr,
				},
			},
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
			Ports: []networkingv1.NetworkPolicyPort{
				{
					Protocol: protocolPtr(corev1.ProtocolTCP),
					Port:     &agentPortIntStr,
				},
			},
		},
	}

	// Append user-defined ingress rules from spec.networkPolicies[].from
	for _, rule := range agent.Spec.NetworkPolicies {
		if rule.From == nil {
			continue
		}
		ingressRule := networkingv1.NetworkPolicyIngressRule{
			From: []networkingv1.NetworkPolicyPeer{
				buildIngressPeerFromNetworkPeer(rule.From, agent.Namespace),
			},
		}
		for _, p := range rule.Ports {
			protocol := corev1.Protocol(p.Protocol)
			if protocol == "" {
				protocol = corev1.ProtocolTCP
			}
			ingressRule.Ports = append(ingressRule.Ports, networkingv1.NetworkPolicyPort{
				Protocol: protocolPtr(protocol),
				Port:     &intstr.IntOrString{Type: intstr.Int, IntVal: p.Port},
			})
		}
		networkPolicy.Spec.Ingress = append(networkPolicy.Spec.Ingress, ingressRule)
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
		gatewayURL := fmt.Sprintf("http://gateway.%s.svc.cluster.local:8000", agent.Namespace)
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
		// Format: http://<service-name>.<namespace>.svc.cluster.local:<port>
		serviceURL := fmt.Sprintf("http://%s.%s.svc.cluster.local:%d", tool.Name, agent.Namespace, port)
		toolURLs = append(toolURLs, serviceURL)
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

	if agent.Spec.ExecutionMode != "" {
		env = append(env, corev1.EnvVar{
			Name:  "AGENT_MODE",
			Value: agent.Spec.ExecutionMode,
		})
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
			Name:  "MODEL_ENDPOINTS",
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
	// The only manual cleanup required is shared RBAC resources, which are not owned by
	// any single agent and must be removed when the last agent in the namespace is deleted.
	return r.cleanupSharedRBAC(ctx, agent)
}

// cleanupSharedRBAC deletes the shared ServiceAccount, Role, and RoleBinding for agent pods
// in the given namespace, but only when no other LanguageAgents remain in that namespace.
func (r *LanguageAgentReconciler) cleanupSharedRBAC(ctx context.Context, agent *langopv1alpha1.LanguageAgent) error {
	log := log.FromContext(ctx)
	ns := agent.Namespace

	// Check whether any other agents remain in this namespace
	agentList := &langopv1alpha1.LanguageAgentList{}
	if err := r.List(ctx, agentList, client.InNamespace(ns)); err != nil {
		return fmt.Errorf("failed to list LanguageAgents: %w", err)
	}
	for _, a := range agentList.Items {
		if a.Name != agent.Name && a.DeletionTimestamp.IsZero() {
			// Another live agent still exists; leave shared resources intact
			return nil
		}
	}

	// Last agent in namespace — delete shared RBAC resources
	toDelete := []client.Object{
		&rbacv1.RoleBinding{ObjectMeta: metav1.ObjectMeta{Name: "language-agent", Namespace: ns}},
		&rbacv1.Role{ObjectMeta: metav1.ObjectMeta{Name: "language-agent", Namespace: ns}},
		&corev1.ServiceAccount{ObjectMeta: metav1.ObjectMeta{Name: "language-agent", Namespace: ns}},
	}
	for _, obj := range toDelete {
		if err := r.Delete(ctx, obj); err != nil && !apierrors.IsNotFound(err) {
			return fmt.Errorf("failed to delete %T %s: %w", obj, "language-agent", err)
		}
		log.Info("Deleted shared RBAC resource", "kind", fmt.Sprintf("%T", obj), "namespace", ns)
	}
	return nil
}

// agentPort returns the port the agent container listens on.
func agentPort(agent *langopv1alpha1.LanguageAgent) int32 {
	if agent.Spec.Port != nil {
		return *agent.Spec.Port
	}
	return 8080
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

	_, err := controllerutil.CreateOrUpdate(ctx, r.Client, service, func() error {
		if err := controllerutil.SetControllerReference(agent, service, r.Scheme); err != nil {
			return err
		}

		port := agentPort(agent)
		svcType := agent.Spec.Deployment.ServiceType
		if svcType == "" {
			svcType = corev1.ServiceTypeClusterIP
		}
		service.Spec = corev1.ServiceSpec{
			Selector: labels,
			Ports: []corev1.ServicePort{
				{
					Name:       "http",
					Port:       port,
					TargetPort: intstr.FromInt32(port),
					Protocol:   corev1.ProtocolTCP,
				},
			},
			Type: svcType,
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

	_, err := controllerutil.CreateOrUpdate(ctx, r.Client, ingress, func() error {
		if err := controllerutil.SetControllerReference(agent, ingress, r.Scheme); err != nil {
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
											Name: agent.Name,
											Port: networkingv1.ServiceBackendPort{
												Number: agentPort(agent),
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
				if cluster.Spec.Ingress != nil && cluster.Spec.Ingress.TLS != nil && (cluster.Spec.Ingress.TLS.Enabled == nil || *cluster.Spec.Ingress.TLS.Enabled) {
					secretName := cluster.Spec.Ingress.TLS.SecretName
					if secretName == "" {
						// Use cert-manager annotation for automatic certificate provisioning
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
						secretName = agent.Name + "-tls"
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
		Watches(&langopv1alpha1.LanguageTool{}, handler.EnqueueRequestsFromMapFunc(enqueue)).
		Watches(&langopv1alpha1.LanguageModel{}, handler.EnqueueRequestsFromMapFunc(enqueue)).
		Watches(&langopv1alpha1.LanguagePersona{}, handler.EnqueueRequestsFromMapFunc(enqueue)).
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

	// Create ServiceAccount
	serviceAccount := &corev1.ServiceAccount{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "language-agent",
			Namespace: targetNamespace,
		},
	}

	_, err := controllerutil.CreateOrUpdate(ctx, r.Client, serviceAccount, func() error {
		// Set labels
		if serviceAccount.Labels == nil {
			serviceAccount.Labels = make(map[string]string)
		}
		serviceAccount.Labels["app.kubernetes.io/name"] = "language-agent"
		serviceAccount.Labels["app.kubernetes.io/component"] = "serviceaccount"
		serviceAccount.Labels["app.kubernetes.io/managed-by"] = "language-operator"

		return nil
	})

	if err != nil {
		return fmt.Errorf("failed to create/update ServiceAccount: %w", err)
	}

	// Delete legacy ClusterRoleBinding if present (migration cleanup)
	legacyCRBName := fmt.Sprintf("language-agent-%s-language-agent", targetNamespace)
	legacyCRB := &rbacv1.ClusterRoleBinding{}
	if err := r.Get(ctx, types.NamespacedName{Name: legacyCRBName}, legacyCRB); err == nil {
		if err := r.Delete(ctx, legacyCRB); err != nil && !apierrors.IsNotFound(err) {
			return fmt.Errorf("failed to delete legacy ClusterRoleBinding: %w", err)
		}
	}

	// Create namespace-scoped Role with minimal permissions for agent pods
	role := &rbacv1.Role{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "language-agent",
			Namespace: targetNamespace,
		},
	}

	_, err = controllerutil.CreateOrUpdate(ctx, r.Client, role, func() error {
		if role.Labels == nil {
			role.Labels = make(map[string]string)
		}
		role.Labels["app.kubernetes.io/name"] = "language-agent"
		role.Labels["app.kubernetes.io/component"] = "role"
		role.Labels["app.kubernetes.io/managed-by"] = "language-operator"
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
		return nil
	})

	if err != nil {
		return fmt.Errorf("failed to create/update Role: %w", err)
	}

	// Create namespace-scoped RoleBinding binding the ServiceAccount to the Role
	roleBinding := &rbacv1.RoleBinding{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "language-agent",
			Namespace: targetNamespace,
		},
	}

	_, err = controllerutil.CreateOrUpdate(ctx, r.Client, roleBinding, func() error {
		if roleBinding.Labels == nil {
			roleBinding.Labels = make(map[string]string)
		}
		roleBinding.Labels["app.kubernetes.io/name"] = "language-agent"
		roleBinding.Labels["app.kubernetes.io/component"] = "rolebinding"
		roleBinding.Labels["app.kubernetes.io/managed-by"] = "language-operator"
		roleBinding.RoleRef = rbacv1.RoleRef{
			APIGroup: "rbac.authorization.k8s.io",
			Kind:     "Role",
			Name:     "language-agent",
		}
		roleBinding.Subjects = []rbacv1.Subject{
			{
				Kind:      "ServiceAccount",
				Name:      "language-agent",
				Namespace: targetNamespace,
			},
		}
		return nil
	})

	if err != nil {
		return fmt.Errorf("failed to create/update RoleBinding: %w", err)
	}

	log.Info("Reconciled agent ServiceAccount and permissions",
		"serviceAccount", "language-agent",
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

	// Default to a language-agent ServiceAccount that will be created in the agent's namespace
	return "language-agent"
}

// resolveCodeConfigMapName resolves the ConfigMap name for agent code based on AgentVersionRef
