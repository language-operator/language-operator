package controllers

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"os"
	"regexp"
	"strings"
	"sync"
	"time"

	"github.com/go-logr/logr"
	"github.com/google/uuid"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	appsv1 "k8s.io/api/apps/v1"
	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	networkingv1 "k8s.io/api/networking/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/client-go/discovery"
	"k8s.io/client-go/tools/record"
	"k8s.io/utils/ptr"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/log"

	langopv1alpha1 "github.com/language-operator/language-operator/api/v1alpha1"
	"github.com/language-operator/language-operator/pkg/synthesis"
	"github.com/language-operator/language-operator/pkg/validation"
)

// gatewayAPICache holds cached Gateway API availability information
type gatewayAPICache struct {
	available bool
	lastCheck time.Time
	mutex     sync.RWMutex
}

const (
	// Gateway API cache TTL - how long to cache the availability result
	gatewayAPICacheTTL = 5 * time.Minute
)

// LanguageAgentReconciler reconciles a LanguageAgent object
type LanguageAgentReconciler struct {
	client.Client
	Scheme                 *runtime.Scheme
	Log                    logr.Logger
	Recorder               record.EventRecorder
	MaxSelfHealingAttempts int32
	SelfHealingEnabled     bool
	RateLimiter            *synthesis.RateLimiter
	QuotaManager           *synthesis.QuotaManager
	AllowedRegistries      []string
	gatewayCache           *gatewayAPICache
}

// agentTracer is the OpenTelemetry tracer for the LanguageAgent controller
var agentTracer = otel.Tracer("language-operator/agent-controller")

const (
	// LangopUserID is the user ID for the langop user (matches Dockerfile)
	LangopUserID = 1000
	// LangopGroupID is the group ID for the langop group
	LangopGroupID = 101
)

// InitializeGatewayCache initializes the Gateway API cache
func (r *LanguageAgentReconciler) InitializeGatewayCache() {
	r.gatewayCache = &gatewayAPICache{}
}

//+kubebuilder:rbac:groups=langop.io,resources=languageagents,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=langop.io,resources=languageagents/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=langop.io,resources=languageagents/finalizers,verbs=update
//+kubebuilder:rbac:groups=langop.io,resources=languagepersonas,verbs=get;list;watch
//+kubebuilder:rbac:groups=langop.io,resources=languageclusters,verbs=get;list;watch
//+kubebuilder:rbac:groups=apps,resources=deployments,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=apps,resources=daemonsets,verbs=get;list;watch
//+kubebuilder:rbac:groups=batch,resources=cronjobs,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups="",resources=configmaps,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups="",resources=persistentvolumeclaims,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups="",resources=services,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups="",resources=pods,verbs=get;list;watch
//+kubebuilder:rbac:groups="",resources=pods/log,verbs=get
//+kubebuilder:rbac:groups=networking.k8s.io,resources=networkpolicies,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=networking.k8s.io,resources=ingresses,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=gateway.networking.k8s.io,resources=httproutes,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=gateway.networking.k8s.io,resources=referencegrants,verbs=get;list;watch;create;update;patch;delete

// Reconcile is part of the main kubernetes reconciliation loop
func (r *LanguageAgentReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	// Start OpenTelemetry span for reconciliation
	ctx, span := agentTracer.Start(ctx, "agent.reconcile")
	defer span.End()

	// Add basic span attributes from request
	span.SetAttributes(
		attribute.String("agent.name", req.Name),
		attribute.String("agent.namespace", req.Namespace),
	)

	log := log.FromContext(ctx)

	// Fetch the LanguageAgent instance
	agent := &langopv1alpha1.LanguageAgent{}
	if err := r.Get(ctx, req.NamespacedName, agent); err != nil {
		if errors.IsNotFound(err) {
			// Resource not found, likely deleted - this is not an error
			span.SetStatus(codes.Ok, "Resource not found (deleted)")
			return ctrl.Result{}, nil
		}
		log.Error(err, "Failed to get LanguageAgent")
		span.RecordError(err)
		span.SetStatus(codes.Error, "Failed to get LanguageAgent")
		return ctrl.Result{}, err
	}

	// Add agent-specific attributes to span
	span.SetAttributes(
		attribute.String("agent.mode", agent.Spec.ExecutionMode),
		attribute.Int64("agent.generation", agent.Generation),
	)

	// Handle deletion
	if !agent.DeletionTimestamp.IsZero() {
		span.AddEvent("Deleting agent")
		if controllerutil.ContainsFinalizer(agent, FinalizerName) {
			if err := r.cleanupResources(ctx, agent); err != nil {
				span.RecordError(err)
				span.SetStatus(codes.Error, "Failed to cleanup resources")
				return ctrl.Result{}, err
			}
			controllerutil.RemoveFinalizer(agent, FinalizerName)
			if err := r.Update(ctx, agent); err != nil {
				span.RecordError(err)
				span.SetStatus(codes.Error, "Failed to remove finalizer")
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
			return ctrl.Result{}, err
		}
	}

	// Validate image registry against whitelist
	if err := r.validateImageRegistry(agent); err != nil {
		log.Error(err, "Image registry validation failed", "image", agent.Spec.Image)
		span.RecordError(err)
		span.SetStatus(codes.Error, "Image registry validation failed")
		SetCondition(&agent.Status.Conditions, "RegistryValidated", metav1.ConditionFalse, "RegistryNotAllowed", err.Error(), agent.Generation)
		if r.Recorder != nil {
			r.Recorder.Eventf(agent, corev1.EventTypeWarning, "RegistryValidationFailed", "Image registry not in whitelist: %s", agent.Spec.Image)
		}
		if updateErr := r.Status().Update(ctx, agent); updateErr != nil {
			log.Error(updateErr, "Failed to update status after registry validation failure")
		}
		return ctrl.Result{}, err
	}
	SetCondition(&agent.Status.Conditions, "RegistryValidated", metav1.ConditionTrue, "Validated", "Image registry is in whitelist", agent.Generation)

	// Detect pod failures for self-healing (if enabled)
	if r.SelfHealingEnabled {
		if err := r.detectPodFailures(ctx, agent); err != nil {
			log.Error(err, "Failed to detect pod failures")
			// Don't fail reconciliation, just log the error
		}
	}

	// Synthesize agent code from instructions (if agent has modelRefs and instructions)
	if len(agent.Spec.ModelRefs) > 0 && agent.Spec.Instructions != "" {
		if err := r.reconcileCodeConfigMap(ctx, agent); err != nil {
			log.Error(err, "Failed to synthesize/reconcile agent code")
			span.RecordError(err)
			span.SetStatus(codes.Error, "Synthesis failed")
			SetCondition(&agent.Status.Conditions, "Synthesized", metav1.ConditionFalse, "SynthesisFailed", err.Error(), agent.Generation)
			r.Status().Update(ctx, agent)
			return ctrl.Result{}, err
		}
		SetCondition(&agent.Status.Conditions, "Synthesized", metav1.ConditionTrue, "CodeGenerated", "Agent code synthesized successfully", agent.Generation)
	}

	// Reconcile ConfigMap
	if err := r.reconcileConfigMap(ctx, agent); err != nil {
		log.Error(err, "Failed to reconcile ConfigMap")
		span.RecordError(err)
		span.SetStatus(codes.Error, "ConfigMap reconciliation failed")
		SetCondition(&agent.Status.Conditions, "Ready", metav1.ConditionFalse, "ConfigMapError", err.Error(), agent.Generation)
		r.Status().Update(ctx, agent)
		return ctrl.Result{}, err
	}

	// Reconcile PVC for workspace if enabled
	if err := r.reconcilePVC(ctx, agent); err != nil {
		log.Error(err, "Failed to reconcile PVC")
		span.RecordError(err)
		span.SetStatus(codes.Error, "PVC reconciliation failed")
		SetCondition(&agent.Status.Conditions, "Ready", metav1.ConditionFalse, "PVCError", err.Error(), agent.Generation)
		r.Status().Update(ctx, agent)
		return ctrl.Result{}, err
	}

	// Reconcile NetworkPolicy for network isolation
	if err := r.reconcileNetworkPolicy(ctx, agent); err != nil {
		log.Error(err, "Failed to reconcile NetworkPolicy")
		span.RecordError(err)
		span.SetStatus(codes.Error, "NetworkPolicy reconciliation failed")
		SetCondition(&agent.Status.Conditions, "Ready", metav1.ConditionFalse, "NetworkPolicyError", err.Error(), agent.Generation)
		r.Status().Update(ctx, agent)
		return ctrl.Result{}, err
	}

	// Detect if NetworkPolicy enforcement is supported
	if supported, cni := r.detectNetworkPolicySupport(ctx); !supported {
		message := fmt.Sprintf("NetworkPolicy created but may not be enforced. CNI plugin '%s' does not support NetworkPolicy. Consider installing Cilium, Calico, Weave Net, or Antrea for network isolation.", cni)
		SetCondition(&agent.Status.Conditions, "NetworkPolicyEnforced", metav1.ConditionFalse, "CNINotSupported", message, agent.Generation)
		if r.Recorder != nil {
			r.Recorder.Eventf(agent, corev1.EventTypeWarning, "NetworkPolicyUnsupported", "CNI '%s' does not enforce NetworkPolicy", cni)
		}
		log.Info("NetworkPolicy enforcement not supported", "cni", cni)
	} else {
		message := fmt.Sprintf("NetworkPolicy enforcement active (CNI: %s)", cni)
		SetCondition(&agent.Status.Conditions, "NetworkPolicyEnforced", metav1.ConditionTrue, "Enforced", message, agent.Generation)
		log.V(1).Info("NetworkPolicy enforcement supported", "cni", cni)
	}

	// Ensure agent has a UUID for webhook routing
	if agent.Status.UUID == "" {
		agent.Status.UUID = uuid.New().String()
		if err := r.Status().Update(ctx, agent); err != nil {
			log.Error(err, "Failed to update agent UUID")
			span.RecordError(err)
			span.SetStatus(codes.Error, "Failed to update agent UUID")
			return ctrl.Result{}, err
		}
		log.Info("Generated UUID for agent", "uuid", agent.Status.UUID)
	}

	// Reconcile Service for agent webhook server (all agents expose port 8080)
	if err := r.reconcileService(ctx, agent); err != nil {
		log.Error(err, "Failed to reconcile Service")
		span.RecordError(err)
		span.SetStatus(codes.Error, "Service reconciliation failed")
		SetCondition(&agent.Status.Conditions, "Ready", metav1.ConditionFalse, "ServiceError", err.Error(), agent.Generation)
		r.Status().Update(ctx, agent)
		return ctrl.Result{}, err
	}

	// Reconcile webhooks (HTTPRoute/Ingress for webhook access)
	if err := r.reconcileWebhooks(ctx, agent); err != nil {
		// Log webhook errors but don't fail reconciliation if domain not configured
		log.Info("Webhook reconciliation skipped or pending", "reason", err.Error())
		SetCondition(&agent.Status.Conditions, "WebhooksReady", metav1.ConditionFalse, "Pending", err.Error(), agent.Generation)
	} else {
		SetCondition(&agent.Status.Conditions, "WebhooksReady", metav1.ConditionTrue, "Configured", "Webhook routing configured", agent.Generation)
	}

	// Reconcile workload based on execution mode
	// If executionMode is empty, skip workload reconciliation until synthesis completes and detects the mode
	switch agent.Spec.ExecutionMode {
	case "autonomous", "interactive", "event-driven":
		if err := r.reconcileDeployment(ctx, agent); err != nil {
			log.Error(err, "Failed to reconcile Deployment")
			span.RecordError(err)
			span.SetStatus(codes.Error, "Deployment reconciliation failed")
			SetCondition(&agent.Status.Conditions, "Ready", metav1.ConditionFalse, "DeploymentError", err.Error(), agent.Generation)
			r.Status().Update(ctx, agent)
			return ctrl.Result{}, err
		}
	case "scheduled":
		if err := r.reconcileCronJob(ctx, agent); err != nil {
			log.Error(err, "Failed to reconcile CronJob")
			span.RecordError(err)
			span.SetStatus(codes.Error, "CronJob reconciliation failed")
			SetCondition(&agent.Status.Conditions, "Ready", metav1.ConditionFalse, "CronJobError", err.Error(), agent.Generation)
			r.Status().Update(ctx, agent)
			return ctrl.Result{}, err
		}
	case "":
		// ExecutionMode not yet set - wait for synthesis to complete and detect the mode
		log.V(1).Info("ExecutionMode not set, skipping workload reconciliation until synthesis completes")
	}

	// Update status only if something changed
	statusChanged := false
	if agent.Status.Phase != "Running" {
		agent.Status.Phase = "Running"
		statusChanged = true
	}
	if SetCondition(&agent.Status.Conditions, "Ready", metav1.ConditionTrue, "ReconcileSuccess", "LanguageAgent is ready", agent.Generation) {
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
	return ctrl.Result{}, nil
}

func (r *LanguageAgentReconciler) reconcileConfigMap(ctx context.Context, agent *langopv1alpha1.LanguageAgent) error {
	log := log.FromContext(ctx)
	data := make(map[string]string)

	// Fetch persona if referenced
	persona, err := r.fetchPersona(ctx, agent)
	if err != nil {
		// Log warning but continue without persona
		log.Error(err, "Failed to fetch persona, continuing without it")
	}

	// Merge instructions with persona systemPrompt if persona is available
	instructions := agent.Spec.Instructions
	if persona != nil {
		if persona.Spec.SystemPrompt != "" {
			if instructions != "" {
				instructions = persona.Spec.SystemPrompt + "\n\n" + instructions
			} else {
				instructions = persona.Spec.SystemPrompt
			}
		}

		// Add persona instructions if available
		if len(persona.Spec.Instructions) > 0 {
			instructions = instructions + "\n\nAdditional Guidelines:\n"
			for _, inst := range persona.Spec.Instructions {
				instructions = instructions + "- " + inst + "\n"
			}
		}
	}

	// Add agent spec as JSON
	specJSON, err := json.Marshal(agent.Spec)
	if err != nil {
		return err
	}
	data["agent.json"] = string(specJSON)

	// Add persona data as JSON if available
	if persona != nil {
		personaJSON, err := json.Marshal(persona.Spec)
		if err != nil {
			return err
		}
		data["persona.json"] = string(personaJSON)
		data["persona_name"] = persona.Name
	}

	// Add other useful data
	data["name"] = agent.Name
	data["namespace"] = agent.Namespace
	data["mode"] = agent.Spec.ExecutionMode
	if agent.Spec.Goal != "" {
		data["goal"] = agent.Spec.Goal
	}

	// Add merged instructions
	if instructions != "" {
		data["instructions"] = instructions
	}

	configMapName := GenerateConfigMapName(agent.Name, "agent")
	return CreateOrUpdateConfigMap(ctx, r.Client, r.Scheme, agent, configMapName, agent.Namespace, data)
}

// reconcileCodeConfigMap synthesizes agent DSL code and stores it in a ConfigMap
func (r *LanguageAgentReconciler) reconcileCodeConfigMap(ctx context.Context, agent *langopv1alpha1.LanguageAgent) error {
	log := log.FromContext(ctx)

	// ConfigMap name for synthesized code
	codeConfigMapName := GenerateConfigMapName(agent.Name, "code")

	// Check if agent is in failure state requiring self-healing
	if r.shouldAttemptSelfHealing(agent) {
		log.Info("Agent in failure state, checking self-healing eligibility",
			"consecutiveFailures", agent.Status.ConsecutiveFailures,
			"selfHealingAttempts", agent.Status.SelfHealingAttempts)

		// Check if we've exceeded max self-healing attempts
		if agent.Status.SelfHealingAttempts >= r.MaxSelfHealingAttempts {
			log.Info("Max self-healing attempts reached, marking agent as failed")
			SetCondition(&agent.Status.Conditions, "Synthesized", metav1.ConditionFalse,
				"MaxAttemptsExceeded",
				fmt.Sprintf("Self-healing failed after %d attempts", r.MaxSelfHealingAttempts),
				agent.Generation)
			agent.Status.Phase = "Failed"
			if err := r.Status().Update(ctx, agent); err != nil {
				return err
			}
			if r.Recorder != nil {
				r.Recorder.Eventf(agent, corev1.EventTypeWarning, "SelfHealingMaxAttempts",
					"Self-healing max attempts (%d) reached, agent marked as failed", r.MaxSelfHealingAttempts)
			}
			return fmt.Errorf("max self-healing attempts exceeded")
		}

		// Implement exponential backoff
		if agent.Status.SynthesisInfo != nil && agent.Status.SynthesisInfo.LastSynthesisTime != nil {
			backoffDuration := calculateBackoff(agent.Status.SelfHealingAttempts)
			timeSinceLastSynthesis := time.Since(agent.Status.SynthesisInfo.LastSynthesisTime.Time)
			if timeSinceLastSynthesis < backoffDuration {
				log.V(1).Info("In backoff period, skipping synthesis",
					"backoffRemaining", backoffDuration-timeSinceLastSynthesis)
				return nil
			}
		}

		// Trigger self-healing synthesis
		log.Info("Triggering self-healing synthesis",
			"attempt", agent.Status.SelfHealingAttempts+1,
			"maxAttempts", r.MaxSelfHealingAttempts)
		if r.Recorder != nil {
			r.Recorder.Eventf(agent, corev1.EventTypeNormal, "SelfHealingTriggered",
				"Self-healing synthesis triggered after %d consecutive failures (attempt %d/%d)",
				agent.Status.ConsecutiveFailures, agent.Status.SelfHealingAttempts+1, r.MaxSelfHealingAttempts)
		}

		agent.Status.SelfHealingAttempts++
		return r.performSelfHealingSynthesis(ctx, agent)
	}

	// Normal synthesis flow
	// Check if we need to synthesize
	// Smart change detection:
	// 1. ConfigMap doesn't exist → full synthesis
	// 2. Instructions changed → full synthesis
	// 3. Persona changed → re-distill only (update existing code's context)
	// 4. Tools/models changed → env var update only (no synthesis needed)
	existingCM := &corev1.ConfigMap{}
	err := r.Get(ctx, types.NamespacedName{Name: codeConfigMapName, Namespace: agent.Namespace}, existingCM)

	needsSynthesis := false
	needsPersonaUpdate := false

	if errors.IsNotFound(err) {
		needsSynthesis = true
		log.Info("Code ConfigMap not found, will synthesize")
	} else if err != nil {
		return err
	} else {
		// Check if ConfigMap has been optimized by CLI - skip synthesis but ensure owner reference
		if existingCM.Annotations["langop.io/optimized"] == "true" {
			log.Info("ConfigMap has langop.io/optimized annotation, skipping synthesis",
				"optimizedAt", existingCM.Annotations["langop.io/optimized-at"],
				"optimizedTask", existingCM.Annotations["langop.io/optimized-task"])

			// Ensure owner reference is set for proper garbage collection
			if err := controllerutil.SetControllerReference(agent, existingCM, r.Scheme); err != nil {
				return fmt.Errorf("failed to set owner reference on optimized ConfigMap: %w", err)
			}
			if err := r.Update(ctx, existingCM); err != nil {
				return fmt.Errorf("failed to update optimized ConfigMap with owner reference: %w", err)
			}

			return nil
		}

		// Compare current vs previous hashes for smart change detection
		currentInstructionsHash := hashString(agent.Spec.Instructions)
		previousInstructionsHash := existingCM.Annotations["langop.io/instructions-hash"]

		currentToolsHash := hashString(strings.Join(r.getToolNames(agent), ","))
		previousToolsHash := existingCM.Annotations["langop.io/tools-hash"]

		currentModelsHash := hashString(strings.Join(r.getModelNames(agent), ","))
		previousModelsHash := existingCM.Annotations["langop.io/models-hash"]

		personaRefs := r.getPersonaNames(agent)
		currentPersonaHash := hashString(strings.Join(personaRefs, ","))
		previousPersonaHash := existingCM.Annotations["langop.io/persona-hash"]

		// Instructions changed → full re-synthesis
		if currentInstructionsHash != previousInstructionsHash {
			needsSynthesis = true
			log.Info("Instructions changed, will re-synthesize",
				"previousHash", previousInstructionsHash,
				"currentHash", currentInstructionsHash)
			// Persona changed → re-distill without full synthesis
		} else if currentPersonaHash != previousPersonaHash {
			needsPersonaUpdate = true
			log.Info("Persona changed, will re-distill",
				"previousPersona", previousPersonaHash,
				"currentPersona", currentPersonaHash)
			// Tools/models changed → logged but no synthesis needed (env vars handle this)
		} else if currentToolsHash != previousToolsHash || currentModelsHash != previousModelsHash {
			log.Info("Tools or models changed, deployment will update env vars",
				"toolsChanged", currentToolsHash != previousToolsHash,
				"modelsChanged", currentModelsHash != previousModelsHash)
			// No synthesis needed - deployment reconciliation handles env var updates
		}
	}

	var dslCode string
	if needsSynthesis {
		// Start synthesis span
		ctx, span := agentTracer.Start(ctx, "agent.synthesize")
		defer span.End()

		// Add synthesis span attributes
		tools := r.getToolNames(agent)
		models := r.getModelNames(agent)
		span.SetAttributes(
			attribute.String("synthesis.agent_name", agent.Name),
			attribute.String("synthesis.namespace", agent.Namespace),
			attribute.Int("synthesis.tools_count", len(tools)),
			attribute.Int("synthesis.models_count", len(models)),
			attribute.Bool("synthesis.is_retry", needsSynthesis),
		)

		// Add attempt number if available
		if agent.Status.SynthesisInfo != nil {
			span.SetAttributes(
				attribute.Int("synthesis.attempt", int(agent.Status.SynthesisInfo.SynthesisAttempts)+1),
			)
		}

		// Fetch persona for distillation
		persona, err := r.fetchPersona(ctx, agent)
		if err != nil {
			log.Error(err, "Failed to fetch persona, continuing without it")
		}

		// Distill persona if available
		var distilledPersona string
		if persona != nil {
			distilledPersona, err = r.distillPersona(ctx, persona, agent)
			if err != nil {
				log.Error(err, "Failed to distill persona, continuing without it")
				distilledPersona = ""
			}
		}

		// Get complete tool schemas for better synthesis quality
		toolSchemas := r.getToolSchemas(ctx, agent)

		// Build synthesis request
		synthReq := synthesis.AgentSynthesisRequest{
			Instructions: agent.Spec.Instructions,
			Tools:        tools,       // Kept for backward compatibility
			ToolSchemas:  toolSchemas, // Complete schemas for better synthesis
			Models:       models,
			PersonaText:  distilledPersona,
			AgentName:    agent.Name,
			Namespace:    agent.Namespace,
		}

		// Check rate limit before synthesis
		if r.RateLimiter != nil {
			if err := r.RateLimiter.CheckAndConsume(ctx, agent.Namespace); err != nil {
				if r.Recorder != nil {
					r.Recorder.Eventf(agent, corev1.EventTypeWarning, "RateLimitExceeded", "Synthesis rate limit exceeded: %v", err)
				}
				log.Info("Synthesis rate limit exceeded", "agent", agent.Name, "namespace", agent.Namespace)
				// Record rate limit metric
				synthesis.RecordSynthesisRateLimitExceeded(agent.Namespace)
				// Record error in span
				span.RecordError(err)
				span.SetStatus(codes.Error, "Rate limit exceeded")
				// Return error to retry later
				return fmt.Errorf("synthesis rate limit exceeded: %w", err)
			}
		}

		// Check quota before synthesis
		if r.QuotaManager != nil {
			// Check attempt quota
			if err := r.QuotaManager.CheckAttemptQuota(ctx, agent.Namespace); err != nil {
				if r.Recorder != nil {
					r.Recorder.Eventf(agent, corev1.EventTypeWarning, "QuotaExceeded", "Synthesis attempt quota exceeded: %v", err)
				}
				log.Info("Synthesis attempt quota exceeded", "agent", agent.Name, "namespace", agent.Namespace)
				// Record quota exceeded metric
				synthesis.RecordSynthesisQuotaExceeded(agent.Namespace, "attempts")
				// Record error in span
				span.RecordError(err)
				span.SetStatus(codes.Error, "Quota exceeded")
				return fmt.Errorf("synthesis attempt quota exceeded: %w", err)
			}
		}

		// Synthesize code
		log.Info("Synthesizing agent code", "agent", agent.Name)
		if r.Recorder != nil {
			r.Recorder.Event(agent, corev1.EventTypeNormal, "SynthesisStarted", "Starting code synthesis from natural language instructions")
		}

		// Create synthesizer from agent's model
		synthesizer, synthesisModelName, err := r.createSynthesizer(ctx, agent)
		if err != nil {
			return fmt.Errorf("failed to create synthesizer: %w", err)
		}

		resp, err := synthesizer.SynthesizeAgent(ctx, synthReq)

		// Record synthesis attempt
		if r.QuotaManager != nil {
			success := err == nil && resp.Error == ""
			errorMsg := ""
			if err != nil {
				errorMsg = err.Error()
			} else if resp.Error != "" {
				errorMsg = resp.Error
			}
			r.QuotaManager.RecordAttempt(ctx, agent.Namespace, agent.Name, success, errorMsg)
		}
		if err != nil {
			if r.Recorder != nil {
				r.Recorder.Eventf(agent, corev1.EventTypeWarning, "SynthesisFailed", "Code synthesis failed: %v", err)
			}
			// Record failure metrics
			synthesis.RecordSynthesisRequest(agent.Namespace, "failed")
			synthesis.RecordSynthesisDuration(agent.Namespace, "failed", time.Since(time.Now()).Seconds())
			// Record error in span
			span.RecordError(err)
			span.SetStatus(codes.Error, "Synthesis failed")
			return fmt.Errorf("synthesis failed: %w", err)
		}

		if resp.Error != "" {
			if r.Recorder != nil {
				r.Recorder.Eventf(agent, corev1.EventTypeWarning, "ValidationFailed", "Synthesized code validation failed: %s", resp.Error)
			}
			// Record validation failure metrics
			synthesis.RecordSynthesisRequest(agent.Namespace, "validation_failed")
			synthesis.RecordSynthesisDuration(agent.Namespace, "validation_failed", resp.DurationSeconds)
			// Record error in span
			validationErr := fmt.Errorf("validation failed: %s", resp.Error)
			span.RecordError(validationErr)
			span.SetStatus(codes.Error, "Validation failed")
			return fmt.Errorf("synthesis validation failed: %s", resp.Error)
		}

		dslCode = resp.DSLCode
		log.Info("Agent code synthesized successfully",
			"agent", agent.Name,
			"codeLength", len(dslCode),
			"duration", resp.DurationSeconds)

		// Add success metrics to span
		span.SetAttributes(
			attribute.Int("synthesis.code_length", len(dslCode)),
			attribute.Float64("synthesis.duration_seconds", resp.DurationSeconds),
		)
		span.SetStatus(codes.Ok, "Synthesis successful")

		if r.Recorder != nil {
			r.Recorder.Eventf(agent, corev1.EventTypeNormal, "SynthesisSucceeded", "Code synthesized successfully in %.2fs", resp.DurationSeconds)
		}

		// Record synthesis cost if available
		if r.QuotaManager != nil && resp.Cost != nil {
			if err := r.QuotaManager.RecordCost(ctx, agent.Namespace, agent.Name, resp.Cost); err != nil {
				log.Error(err, "Failed to record synthesis cost")
			} else {
				log.Info("Synthesis cost recorded",
					"agent", agent.Name,
					"cost", resp.Cost.TotalCost,
					"currency", resp.Cost.Currency,
					"inputTokens", resp.Cost.InputTokens,
					"outputTokens", resp.Cost.OutputTokens)
			}

			// Record metrics
			synthesis.RecordSynthesisTokens(agent.Namespace, resp.Cost.InputTokens, resp.Cost.OutputTokens)
			synthesis.RecordSynthesisCost(agent.Namespace, resp.Cost.TotalCost)
		}

		// Record synthesis success metric
		synthesis.RecordSynthesisRequest(agent.Namespace, "success")
		synthesis.RecordSynthesisDuration(agent.Namespace, "success", resp.DurationSeconds)

		// Update remaining quota metrics
		if r.QuotaManager != nil {
			remainingCost, remainingAttempts := r.QuotaManager.GetRemainingQuota(agent.Namespace)
			synthesis.UpdateNamespaceQuotaRemaining(agent.Namespace, "cost", remainingCost)
			synthesis.UpdateNamespaceQuotaRemaining(agent.Namespace, "attempts", float64(remainingAttempts))
		}

		// Update synthesis info in status
		now := metav1.Now()
		if agent.Status.SynthesisInfo == nil {
			agent.Status.SynthesisInfo = &langopv1alpha1.SynthesisInfo{}
		}
		agent.Status.SynthesisInfo.LastSynthesisTime = &now
		agent.Status.SynthesisInfo.SynthesisModel = synthesisModelName
		agent.Status.SynthesisInfo.SynthesisDuration = resp.DurationSeconds
		agent.Status.SynthesisInfo.CodeHash = hashString(dslCode)
		agent.Status.SynthesisInfo.InstructionsHash = hashString(agent.Spec.Instructions)
		agent.Status.SynthesisInfo.ValidationErrors = resp.ValidationErrors
		if agent.Status.SynthesisInfo.SynthesisAttempts == 0 || needsSynthesis {
			agent.Status.SynthesisInfo.SynthesisAttempts++
		}

		// Update cost metrics in status if available
		if resp.Cost != nil {
			agent.Status.CostMetrics = resp.Cost.ToAgentCostMetrics()
		}

		// Update agent status
		if err := r.Status().Update(ctx, agent); err != nil {
			log.Error(err, "Failed to update synthesis info in status")
		}
	} else if needsPersonaUpdate {
		// Persona changed but instructions didn't → re-distill only
		// This updates the persona context without re-synthesizing the entire code
		dslCode = existingCM.Data["agent.rb"]

		persona, err := r.fetchPersona(ctx, agent)
		if err != nil {
			log.Error(err, "Failed to fetch persona for update")
		} else if persona != nil {
			_, err = r.distillPersona(ctx, persona, agent)
			if err != nil {
				log.Error(err, "Failed to re-distill persona")
			} else {
				log.Info("Persona re-distilled successfully")
				if r.Recorder != nil {
					r.Recorder.Event(agent, corev1.EventTypeNormal, "PersonaUpdated", "Persona re-distilled without code re-synthesis")
				}
			}
		}
	} else {
		// Use existing code
		dslCode = existingCM.Data["agent.rb"]
		log.Info("Using existing synthesized code", "agent", agent.Name)
	}

	// Create or update ConfigMap with synthesized code
	data := map[string]string{
		"agent.rb": dslCode,
	}

	// Store all hashes for smart change detection
	annotations := map[string]string{
		"langop.io/instructions-hash": hashString(agent.Spec.Instructions),
		"langop.io/tools-hash":        hashString(strings.Join(r.getToolNames(agent), ",")),
		"langop.io/models-hash":       hashString(strings.Join(r.getModelNames(agent), ",")),
		"langop.io/persona-hash":      hashString(strings.Join(r.getPersonaNames(agent), ",")),
	}

	// Only update synthesized-at timestamp when we actually synthesized new code
	if needsSynthesis || needsPersonaUpdate {
		annotations["langop.io/synthesized-at"] = metav1.Now().Format("2006-01-02T15:04:05Z")
	} else if existingCM != nil && existingCM.Annotations != nil {
		// Preserve existing timestamp when reusing code
		if existingTimestamp, ok := existingCM.Annotations["langop.io/synthesized-at"]; ok {
			annotations["langop.io/synthesized-at"] = existingTimestamp
		}
	}

	if err := CreateOrUpdateConfigMapWithAnnotations(ctx, r.Client, r.Scheme, agent, codeConfigMapName, agent.Namespace, data, annotations); err != nil {
		return err
	}

	// Parse DSL to extract mode and schedule, then update spec if needed
	detectedMode, detectedSchedule := parseDSLMode(dslCode)
	specNeedsUpdate := false

	// Check if executionMode needs to be updated
	if agent.Spec.ExecutionMode == "" || agent.Spec.ExecutionMode != detectedMode {
		log.Info("Auto-detected executionMode from synthesized DSL",
			"agent", agent.Name,
			"previousMode", agent.Spec.ExecutionMode,
			"detectedMode", detectedMode)
		agent.Spec.ExecutionMode = detectedMode
		specNeedsUpdate = true
	}

	// Check if schedule needs to be updated (only for scheduled mode)
	if detectedMode == "scheduled" && detectedSchedule != "" && agent.Spec.Schedule != detectedSchedule {
		log.Info("Auto-detected schedule from synthesized DSL",
			"agent", agent.Name,
			"previousSchedule", agent.Spec.Schedule,
			"detectedSchedule", detectedSchedule)
		agent.Spec.Schedule = detectedSchedule
		specNeedsUpdate = true
	}

	// Update the agent spec if changes were detected
	if specNeedsUpdate {
		if err := r.Update(ctx, agent); err != nil {
			log.Error(err, "Failed to update agent spec with auto-detected mode and schedule")
			return err
		}
		log.Info("Agent spec updated with auto-detected execution mode and schedule",
			"agent", agent.Name,
			"executionMode", agent.Spec.ExecutionMode,
			"schedule", agent.Spec.Schedule)
		if r.Recorder != nil {
			r.Recorder.Eventf(agent, corev1.EventTypeNormal, "ExecutionModeDetected",
				"Auto-detected executionMode: %s", detectedMode)
		}
	}

	return nil
}

// distillPersona calls the synthesizer to distill a persona into a system message
func (r *LanguageAgentReconciler) distillPersona(ctx context.Context, persona *langopv1alpha1.LanguagePersona, agent *langopv1alpha1.LanguageAgent) (string, error) {
	personaInfo := synthesis.PersonaInfo{
		Name:         persona.Name,
		Description:  persona.Spec.Description,
		SystemPrompt: persona.Spec.SystemPrompt,
		Tone:         persona.Spec.Tone,
		Language:     persona.Spec.Language,
	}

	agentCtx := synthesis.AgentContext{
		AgentName:    agent.Name,
		Instructions: agent.Spec.Instructions,
		Tools:        strings.Join(r.getToolNames(agent), ", "),
	}

	// Create synthesizer from agent's model for persona distillation
	synthesizer, _, err := r.createSynthesizer(ctx, agent)
	if err != nil {
		return "", fmt.Errorf("failed to create synthesizer for persona distillation: %w", err)
	}

	return synthesizer.DistillPersona(ctx, personaInfo, agentCtx)
}

// getToolNames extracts tool names from agent's toolRefs
func (r *LanguageAgentReconciler) getToolNames(agent *langopv1alpha1.LanguageAgent) []string {
	var names []string
	for _, ref := range agent.Spec.ToolRefs {
		names = append(names, ref.Name)
	}
	return names
}

// getToolSchemas extracts complete tool schemas from agent's toolRefs
func (r *LanguageAgentReconciler) getToolSchemas(ctx context.Context, agent *langopv1alpha1.LanguageAgent) []langopv1alpha1.ToolSchema {
	var allSchemas []langopv1alpha1.ToolSchema

	for _, ref := range agent.Spec.ToolRefs {
		// Get the LanguageTool CR
		tool := &langopv1alpha1.LanguageTool{}
		err := r.Get(ctx, types.NamespacedName{
			Name:      ref.Name,
			Namespace: agent.Namespace,
		}, tool)

		if err != nil {
			// Log error but continue - don't fail synthesis for missing tools
			log := log.FromContext(ctx)
			log.Error(err, "Failed to get LanguageTool for schema", "tool", ref.Name, "agent", agent.Name)
			continue
		}

		// Add schemas from this tool to the collection
		if len(tool.Status.ToolSchemas) > 0 {
			allSchemas = append(allSchemas, tool.Status.ToolSchemas...)
		}
	}

	return allSchemas
}

// getModelNames extracts model names from agent's modelRefs
func (r *LanguageAgentReconciler) getModelNames(agent *langopv1alpha1.LanguageAgent) []string {
	var names []string
	for _, ref := range agent.Spec.ModelRefs {
		names = append(names, ref.Name)
	}
	return names
}

// getSynthesisModel returns the LanguageModel to use for synthesis
// Prefers model with role "primary", otherwise uses the first model
func (r *LanguageAgentReconciler) getSynthesisModel(ctx context.Context, agent *langopv1alpha1.LanguageAgent) (*langopv1alpha1.LanguageModel, error) {
	if len(agent.Spec.ModelRefs) == 0 {
		return nil, fmt.Errorf("agent has no modelRefs configured")
	}

	// Find primary model or use first one
	var modelRef *langopv1alpha1.ModelReference
	for i := range agent.Spec.ModelRefs {
		ref := &agent.Spec.ModelRefs[i]
		if ref.Role == "primary" || ref.Role == "" {
			modelRef = ref
			break
		}
	}
	if modelRef == nil {
		modelRef = &agent.Spec.ModelRefs[0]
	}

	// Fetch the LanguageModel
	namespace := modelRef.Namespace
	if namespace == "" {
		namespace = agent.Namespace
	}

	model := &langopv1alpha1.LanguageModel{}
	if err := r.Get(ctx, types.NamespacedName{Name: modelRef.Name, Namespace: namespace}, model); err != nil {
		return nil, fmt.Errorf("failed to get synthesis model %s/%s: %w", namespace, modelRef.Name, err)
	}

	return model, nil
}

// createSynthesizer creates a synthesizer from the agent's model
func (r *LanguageAgentReconciler) createSynthesizer(ctx context.Context, agent *langopv1alpha1.LanguageAgent) (synthesis.AgentSynthesizer, string, error) {
	model, err := r.getSynthesisModel(ctx, agent)
	if err != nil {
		return nil, "", err
	}

	synth, err := synthesis.NewSynthesizerFromLanguageModel(ctx, r.Client, model, r.Log.WithName("synthesis"))
	if err != nil {
		return nil, "", fmt.Errorf("failed to create synthesizer: %w", err)
	}

	return synth, model.Spec.ModelName, nil
}

// getPersonaNames extracts persona names from agent's personaRefs
func (r *LanguageAgentReconciler) getPersonaNames(agent *langopv1alpha1.LanguageAgent) []string {
	var names []string
	for _, ref := range agent.Spec.PersonaRefs {
		names = append(names, ref.Name)
	}
	return names
}

// hashString creates a SHA256 hash of a string for change detection
func hashString(s string) string {
	h := sha256.New()
	h.Write([]byte(s))
	return fmt.Sprintf("%x", h.Sum(nil))
}

// parseDSLMode extracts the mode and schedule from synthesized DSL code
func parseDSLMode(dslCode string) (mode string, schedule string) {
	// Default to autonomous if no mode directive found
	mode = "autonomous"
	schedule = ""

	// Match "mode :scheduled" or "mode :autonomous"
	modeRegex := regexp.MustCompile(`(?m)^\s*mode\s+:(\w+)`)
	if matches := modeRegex.FindStringSubmatch(dslCode); len(matches) > 1 {
		dslMode := matches[1]
		switch dslMode {
		case "scheduled":
			mode = "scheduled"
		case "autonomous":
			mode = "autonomous"
		case "interactive":
			mode = "interactive"
		case "event_driven":
			mode = "event-driven"
		}
	}

	// Match schedule "*/10 * * * *" or schedule '*/10 * * * *'
	scheduleRegex := regexp.MustCompile(`(?m)^\s*schedule\s+["']([^"']+)["']`)
	if matches := scheduleRegex.FindStringSubmatch(dslCode); len(matches) > 1 {
		schedule = matches[1]
		// If a schedule is present, the mode should be "scheduled"
		mode = "scheduled"
	}

	return mode, schedule
}

func (r *LanguageAgentReconciler) reconcilePVC(ctx context.Context, agent *langopv1alpha1.LanguageAgent) error {
	// Skip if workspace is not enabled
	if agent.Spec.Workspace == nil || !agent.Spec.Workspace.Enabled {
		return nil
	}

	// Determine target namespace - always use agent's namespace
	// If cluster ref is set, verify cluster exists in same namespace
	targetNamespace := agent.Namespace
	if err := ValidateClusterReference(ctx, r.Client, agent.Spec.ClusterRef, agent.Namespace); err != nil {
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
			pvc.Spec = corev1.PersistentVolumeClaimSpec{
				AccessModes: []corev1.PersistentVolumeAccessMode{accessMode},
				Resources: corev1.VolumeResourceRequirements{
					Requests: corev1.ResourceList{
						corev1.ResourceStorage: resource.MustParse(size),
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

// buildPodSecurityContext creates the pod-level security context for agent pods
func (r *LanguageAgentReconciler) buildPodSecurityContext() *corev1.PodSecurityContext {
	return &corev1.PodSecurityContext{
		RunAsNonRoot: ptr.To(true),
		RunAsUser:    ptr.To[int64](LangopUserID),
		FSGroup:      ptr.To[int64](LangopGroupID),
		SeccompProfile: &corev1.SeccompProfile{
			Type: corev1.SeccompProfileTypeRuntimeDefault,
		},
	}
}

// buildContainerSecurityContext creates the container-level security context for agent containers
func (r *LanguageAgentReconciler) buildContainerSecurityContext() *corev1.SecurityContext {
	return &corev1.SecurityContext{
		AllowPrivilegeEscalation: ptr.To(false),
		RunAsNonRoot:             ptr.To(true),
		RunAsUser:                ptr.To[int64](LangopUserID),
		ReadOnlyRootFilesystem:   ptr.To(true),
		Capabilities: &corev1.Capabilities{
			Drop: []corev1.Capability{"ALL"},
		},
	}
}

// buildVolumes creates the volumes and volume mounts for agent pods
func (r *LanguageAgentReconciler) buildVolumes(agent *langopv1alpha1.LanguageAgent) ([]corev1.Volume, []corev1.VolumeMount) {
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

	// /home/langop/.bundle - Ruby bundler cache
	volumes = append(volumes, corev1.Volume{
		Name: "ruby-bundle",
		VolumeSource: corev1.VolumeSource{
			EmptyDir: &corev1.EmptyDirVolumeSource{
				Medium: corev1.StorageMediumMemory, // Use tmpfs
			},
		},
	})
	volumeMounts = append(volumeMounts, corev1.VolumeMount{
		Name:      "ruby-bundle",
		MountPath: "/home/langop/.bundle",
	})

	// /home/langop/.gem - Ruby gem installation directory
	volumes = append(volumes, corev1.Volume{
		Name: "ruby-gem",
		VolumeSource: corev1.VolumeSource{
			EmptyDir: &corev1.EmptyDirVolumeSource{
				Medium: corev1.StorageMediumMemory, // Use tmpfs
			},
		},
	})
	volumeMounts = append(volumeMounts, corev1.VolumeMount{
		Name:      "ruby-gem",
		MountPath: "/home/langop/.gem",
	})

	// Add code ConfigMap volume if agent has modelRefs and instructions (synthesis enabled)
	if len(agent.Spec.ModelRefs) > 0 && agent.Spec.Instructions != "" {
		codeConfigMapName := GenerateConfigMapName(agent.Name, "code")
		volumes = append(volumes, corev1.Volume{
			Name: "agent-code",
			VolumeSource: corev1.VolumeSource{
				ConfigMap: &corev1.ConfigMapVolumeSource{
					LocalObjectReference: corev1.LocalObjectReference{
						Name: codeConfigMapName,
					},
				},
			},
		})
		volumeMounts = append(volumeMounts, corev1.VolumeMount{
			Name:      "agent-code",
			MountPath: "/etc/agent/code",
			ReadOnly:  true,
		})
	}

	// Add workspace volume if enabled
	if agent.Spec.Workspace != nil && agent.Spec.Workspace.Enabled {
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
	log := log.FromContext(ctx)

	// Fetch persona if referenced
	persona, err := r.fetchPersona(ctx, agent)
	if err != nil {
		// Log warning but continue without persona
		log.Error(err, "Failed to fetch persona for deployment, continuing without it")
	}

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

	// If cluster ref is set, verify cluster exists and is ready
	if err := ValidateClusterReference(ctx, r.Client, agent.Spec.ClusterRef, agent.Namespace); err != nil {
		return err
	}

	// Add cluster label if cluster ref is set
	if agent.Spec.ClusterRef != "" {
		labels["langop.io/cluster"] = agent.Spec.ClusterRef
	}

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
		if agent.Spec.Replicas != nil {
			replicas = *agent.Spec.Replicas
		}

		// Build container list starting with the agent
		containers := []corev1.Container{
			{
				Name:  "agent",
				Image: agent.Spec.Image,
				Env:   r.buildAgentEnv(ctx, agent, modelURLs, modelNames, toolURLs, persona),
			},
		}

		deployment.Spec = appsv1.DeploymentSpec{
			Replicas: &replicas,
			Selector: &metav1.LabelSelector{
				MatchLabels: labels,
			},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: labels,
				},
				Spec: corev1.PodSpec{
					ShareProcessNamespace: &[]bool{len(sidecarContainers) > 0}[0],
					InitContainers:        sidecarContainers, // Sidecars as init containers with restartPolicy: Always
					Containers:            containers,
					SecurityContext:       r.buildPodSecurityContext(),
				},
			},
		}

		// Add container security context for agent container
		deployment.Spec.Template.Spec.Containers[0].SecurityContext = r.buildContainerSecurityContext()

		// Add resource requirements if specified
		deployment.Spec.Template.Spec.Containers[0].Resources = agent.Spec.Resources

		// Build and apply volumes and volume mounts
		volumes, volumeMounts := r.buildVolumes(agent)
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

func (r *LanguageAgentReconciler) reconcileCronJob(ctx context.Context, agent *langopv1alpha1.LanguageAgent) error {
	log := log.FromContext(ctx)

	// Fetch persona if referenced
	persona, err := r.fetchPersona(ctx, agent)
	if err != nil {
		// Log warning but continue without persona
		log.Error(err, "Failed to fetch persona for cronjob, continuing without it")
	}

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

	// If cluster ref is set, verify cluster exists and is ready
	if err := ValidateClusterReference(ctx, r.Client, agent.Spec.ClusterRef, agent.Namespace); err != nil {
		return err
	}

	// Add cluster label if cluster ref is set
	if agent.Spec.ClusterRef != "" {
		labels["langop.io/cluster"] = agent.Spec.ClusterRef
	}

	cronJob := &batchv1.CronJob{
		ObjectMeta: metav1.ObjectMeta{
			Name:      agent.Name,
			Namespace: targetNamespace,
			Labels:    labels,
		},
	}

	_, err = controllerutil.CreateOrUpdate(ctx, r.Client, cronJob, func() error {
		if err := controllerutil.SetControllerReference(agent, cronJob, r.Scheme); err != nil {
			return err
		}

		schedule := "0 * * * *" // Default: hourly
		if agent.Spec.Schedule != "" {
			schedule = agent.Spec.Schedule
		}

		// Build container list starting with the agent
		containers := []corev1.Container{
			{
				Name:  "agent",
				Image: agent.Spec.Image,
				Env:   r.buildAgentEnv(ctx, agent, modelURLs, modelNames, toolURLs, persona),
			},
		}

		cronJob.Spec = batchv1.CronJobSpec{
			Schedule: schedule,
			JobTemplate: batchv1.JobTemplateSpec{
				Spec: batchv1.JobSpec{
					Template: corev1.PodTemplateSpec{
						ObjectMeta: metav1.ObjectMeta{
							Labels: labels,
						},
						Spec: corev1.PodSpec{
							RestartPolicy:         corev1.RestartPolicyOnFailure,
							ShareProcessNamespace: &[]bool{len(sidecarContainers) > 0}[0],
							InitContainers:        sidecarContainers, // Sidecars as init containers with restartPolicy: Always
							Containers:            containers,
							SecurityContext:       r.buildPodSecurityContext(),
						},
					},
				},
			},
		}

		// Add container security context for agent container
		cronJob.Spec.JobTemplate.Spec.Template.Spec.Containers[0].SecurityContext = r.buildContainerSecurityContext()

		// Add resource requirements if specified
		cronJob.Spec.JobTemplate.Spec.Template.Spec.Containers[0].Resources = agent.Spec.Resources

		// Build and apply volumes and volume mounts
		volumes, volumeMounts := r.buildVolumes(agent)
		if len(volumes) > 0 {
			cronJob.Spec.JobTemplate.Spec.Template.Spec.Volumes = volumes
		}
		if len(volumeMounts) > 0 {
			cronJob.Spec.JobTemplate.Spec.Template.Spec.Containers[0].VolumeMounts = volumeMounts
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
		agent.Name,
		agent.Namespace,
		labels,
		"", // provider - not applicable for agents
		"", // endpoint - not applicable for agents
		otelEndpoint,
		agent.Spec.Egress,
	)

	// Create or update the NetworkPolicy with owner reference
	return CreateOrUpdateNetworkPolicy(ctx, r.Client, r.Scheme, agent, networkPolicy)
}

func (r *LanguageAgentReconciler) resolveModels(ctx context.Context, agent *langopv1alpha1.LanguageAgent) ([]string, []string, error) {
	var modelURLs []string
	var modelNames []string

	for _, modelRef := range agent.Spec.ModelRefs {
		// Determine namespace
		namespace := modelRef.Namespace
		if namespace == "" {
			namespace = agent.Namespace
		}

		// Fetch the LanguageModel
		model := &langopv1alpha1.LanguageModel{}
		if err := r.Get(ctx, types.NamespacedName{Name: modelRef.Name, Namespace: namespace}, model); err != nil {
			return nil, nil, fmt.Errorf("failed to get model %s/%s: %w", namespace, modelRef.Name, err)
		}

		// Build LiteLLM proxy URL
		// Format: http://<service-name>.<namespace>.svc.cluster.local:<port>
		// TODO: Once LanguageModel controller creates Service, get actual port from service
		port := 8000 // Default LiteLLM port

		serviceURL := fmt.Sprintf("http://%s.%s.svc.cluster.local:%d", model.Name, namespace, port)
		modelURLs = append(modelURLs, serviceURL)

		// Collect model name from spec
		if model.Spec.ModelName != "" {
			modelNames = append(modelNames, model.Spec.ModelName)
		}
	}

	return modelURLs, modelNames, nil
}

func (r *LanguageAgentReconciler) resolveSidecarTools(ctx context.Context, agent *langopv1alpha1.LanguageAgent) ([]corev1.Container, error) {
	var sidecarContainers []corev1.Container

	for _, toolRef := range agent.Spec.ToolRefs {
		// Determine namespace
		namespace := toolRef.Namespace
		if namespace == "" {
			namespace = agent.Namespace
		}

		// Fetch the LanguageTool
		tool := &langopv1alpha1.LanguageTool{}
		if err := r.Get(ctx, types.NamespacedName{Name: toolRef.Name, Namespace: namespace}, tool); err != nil {
			return nil, fmt.Errorf("failed to get tool %s/%s: %w", namespace, toolRef.Name, err)
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
			Env: tool.Spec.Env,
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

		// Add resource requirements if specified
		container.Resources = tool.Spec.Resources

		// Mount workspace if agent has workspace enabled
		if agent.Spec.Workspace != nil && agent.Spec.Workspace.Enabled {
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

	for _, toolRef := range agent.Spec.ToolRefs {
		// Determine namespace
		namespace := toolRef.Namespace
		if namespace == "" {
			namespace = agent.Namespace
		}

		// Fetch the LanguageTool
		tool := &langopv1alpha1.LanguageTool{}
		if err := r.Get(ctx, types.NamespacedName{Name: toolRef.Name, Namespace: namespace}, tool); err != nil {
			return nil, fmt.Errorf("failed to get tool %s/%s: %w", namespace, toolRef.Name, err)
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
		serviceURL := fmt.Sprintf("http://%s.%s.svc.cluster.local:%d", tool.Name, namespace, port)
		toolURLs = append(toolURLs, serviceURL)
	}

	return toolURLs, nil
}

func (r *LanguageAgentReconciler) buildAgentEnv(ctx context.Context, agent *langopv1alpha1.LanguageAgent, modelURLs []string, modelNames []string, toolURLs []string, persona *langopv1alpha1.LanguagePersona) []corev1.EnvVar {
	env := []corev1.EnvVar{
		{
			Name:  "CONFIG_PATH",
			Value: "/nonexistent/config.yaml", // Force config to load from env vars
		},
		{
			Name:  "AGENT_NAME",
			Value: agent.Name,
		},
		{
			Name:  "AGENT_NAMESPACE",
			Value: agent.Namespace,
		},
		{
			Name:  "AGENT_MODE",
			Value: agent.Spec.ExecutionMode,
		},
	}

	// Note: We don't inject TRACEPARENT here because it changes on every reconciliation
	// (new span ID each time), which would cause unnecessary CronJob/Deployment updates
	// and trigger reconciliation loops. The agent pod will create its own traces.

	// Inject OpenTelemetry configuration from operator environment
	if endpoint := os.Getenv("OTEL_EXPORTER_OTLP_ENDPOINT"); endpoint != "" {
		// Ruby OpenTelemetry exporter uses HTTP (port 4318) not gRPC (port 4317)
		// Replace :4317 with :4318 for Ruby agents
		agentEndpoint := strings.Replace(endpoint, ":4317", ":4318", 1)

		// Ensure http:// protocol is present (required by Ruby OTLP exporter)
		if !strings.HasPrefix(agentEndpoint, "http://") && !strings.HasPrefix(agentEndpoint, "https://") {
			agentEndpoint = "http://" + agentEndpoint
		}

		// Configure Ruby OpenTelemetry auto-instrumentation via standard env vars
		env = append(env, corev1.EnvVar{
			Name:  "OTEL_EXPORTER_OTLP_ENDPOINT",
			Value: agentEndpoint,
		})
		env = append(env, corev1.EnvVar{
			Name:  "OTEL_TRACES_EXPORTER",
			Value: "otlp",
		})
		env = append(env, corev1.EnvVar{
			Name:  "OTEL_EXPORTER_OTLP_PROTOCOL",
			Value: "http/protobuf",
		})
		env = append(env, corev1.EnvVar{
			Name:  "OTEL_LOGS_EXPORTER",
			Value: "otlp",
		})
	}

	// Set unique service name for agent
	env = append(env, corev1.EnvVar{
		Name:  "OTEL_SERVICE_NAME",
		Value: fmt.Sprintf("language-operator-agent-%s", agent.Name),
	})

	if agent.Spec.Goal != "" {
		env = append(env, corev1.EnvVar{
			Name:  "AGENT_GOAL",
			Value: agent.Spec.Goal,
		})
	}

	if agent.Spec.Instructions != "" {
		env = append(env, corev1.EnvVar{
			Name:  "AGENT_INSTRUCTIONS",
			Value: agent.Spec.Instructions,
		})
	}

	// Add persona environment variables if persona is set
	if persona != nil {
		env = append(env, corev1.EnvVar{
			Name:  "PERSONA_NAME",
			Value: persona.Name,
		})
		if persona.Spec.Tone != "" {
			env = append(env, corev1.EnvVar{
				Name:  "PERSONA_TONE",
				Value: persona.Spec.Tone,
			})
		}
		if persona.Spec.Language != "" {
			env = append(env, corev1.EnvVar{
				Name:  "PERSONA_LANGUAGE",
				Value: persona.Spec.Language,
			})
		}
	}

	// Add LiteLLM model proxy URLs (comma-separated)
	if len(modelURLs) > 0 {
		env = append(env, corev1.EnvVar{
			Name:  "MODEL_ENDPOINTS",
			Value: strings.Join(modelURLs, ","),
		})
	}

	// Add model names (comma-separated)
	// This tells the agent which model to request from the proxy
	if len(modelNames) > 0 {
		env = append(env, corev1.EnvVar{
			Name:  "LLM_MODEL",
			Value: strings.Join(modelNames, ","),
		})
	}

	// Add dummy API key for local proxies (LiteLLM doesn't need auth)
	// RubyLLM requires an API key to be set, so we provide a placeholder
	if len(modelURLs) > 0 {
		env = append(env, corev1.EnvVar{
			Name:  "OPENAI_API_KEY",
			Value: "sk-dummy-key-for-local-proxy",
		})
	}

	// Disable HTTPX io_uring to avoid permission errors in containers
	// HTTPX's io_uring implementation can fail with EPERM in restricted environments
	env = append(env, corev1.EnvVar{
		Name:  "HTTPX_NO_IO_URING",
		Value: "1",
	})

	// Add MCP tool server URLs (comma-separated)
	if len(toolURLs) > 0 {
		env = append(env, corev1.EnvVar{
			Name:  "MCP_SERVERS",
			Value: strings.Join(toolURLs, ","),
		})
	}

	// Add environment variables from spec
	env = append(env, agent.Spec.Env...)

	return env
}

func (r *LanguageAgentReconciler) fetchPersona(ctx context.Context, agent *langopv1alpha1.LanguageAgent) (*langopv1alpha1.LanguagePersona, error) {
	// Return nil if no personas are referenced
	if len(agent.Spec.PersonaRefs) == 0 {
		return nil, nil
	}

	// Fetch all personas
	var personas []*langopv1alpha1.LanguagePersona
	for _, ref := range agent.Spec.PersonaRefs {
		// Determine namespace
		namespace := ref.Namespace
		if namespace == "" {
			namespace = agent.Namespace
		}

		// Fetch the LanguagePersona
		persona := &langopv1alpha1.LanguagePersona{}
		if err := r.Get(ctx, types.NamespacedName{Name: ref.Name, Namespace: namespace}, persona); err != nil {
			if errors.IsNotFound(err) {
				return nil, fmt.Errorf("persona %s/%s not found", namespace, ref.Name)
			}
			return nil, fmt.Errorf("failed to get persona %s/%s: %w", namespace, ref.Name, err)
		}

		// Check if persona is ready
		if persona.Status.Phase != "Ready" {
			return nil, fmt.Errorf("persona %s/%s is not ready (phase: %s)", namespace, ref.Name, persona.Status.Phase)
		}

		personas = append(personas, persona)
	}

	// Compose personas in order of importance (later personas override earlier ones)
	return r.composePersonas(personas), nil
}

// composePersonas merges multiple personas with later personas taking precedence
func (r *LanguageAgentReconciler) composePersonas(personas []*langopv1alpha1.LanguagePersona) *langopv1alpha1.LanguagePersona {
	if len(personas) == 0 {
		return nil
	}
	if len(personas) == 1 {
		return personas[0]
	}

	// Start with a copy of the first persona
	composed := personas[0].DeepCopy()

	// Merge each subsequent persona, with later ones taking precedence
	for i := 1; i < len(personas); i++ {
		p := personas[i]

		// Override scalar fields if non-empty
		if p.Spec.DisplayName != "" {
			composed.Spec.DisplayName = p.Spec.DisplayName
		}
		if p.Spec.Description != "" {
			composed.Spec.Description = p.Spec.Description
		}
		if p.Spec.SystemPrompt != "" {
			composed.Spec.SystemPrompt = p.Spec.SystemPrompt
		}
		if p.Spec.Tone != "" {
			composed.Spec.Tone = p.Spec.Tone
		}
		if p.Spec.Language != "" {
			composed.Spec.Language = p.Spec.Language
		}
		if p.Spec.ResponseFormat != nil {
			composed.Spec.ResponseFormat = p.Spec.ResponseFormat
		}

		// Append array fields (capabilities, limitations, etc.)
		composed.Spec.Capabilities = append(composed.Spec.Capabilities, p.Spec.Capabilities...)
		composed.Spec.Limitations = append(composed.Spec.Limitations, p.Spec.Limitations...)
		composed.Spec.Examples = append(composed.Spec.Examples, p.Spec.Examples...)
		composed.Spec.Rules = append(composed.Spec.Rules, p.Spec.Rules...)
		composed.Spec.Instructions = append(composed.Spec.Instructions, p.Spec.Instructions...)
		composed.Spec.KnowledgeSources = append(composed.Spec.KnowledgeSources, p.Spec.KnowledgeSources...)

		// Merge tool preferences
		if p.Spec.ToolPreferences != nil {
			if composed.Spec.ToolPreferences == nil {
				composed.Spec.ToolPreferences = &langopv1alpha1.ToolPreferencesSpec{}
			}
			composed.Spec.ToolPreferences.PreferredTools = append(composed.Spec.ToolPreferences.PreferredTools, p.Spec.ToolPreferences.PreferredTools...)
			composed.Spec.ToolPreferences.AvoidTools = append(composed.Spec.ToolPreferences.AvoidTools, p.Spec.ToolPreferences.AvoidTools...)
			if p.Spec.ToolPreferences.Strategy != "" {
				composed.Spec.ToolPreferences.Strategy = p.Spec.ToolPreferences.Strategy
			}
			if p.Spec.ToolPreferences.AlwaysConfirm {
				composed.Spec.ToolPreferences.AlwaysConfirm = p.Spec.ToolPreferences.AlwaysConfirm
			}
			if p.Spec.ToolPreferences.ExplainToolUse {
				composed.Spec.ToolPreferences.ExplainToolUse = p.Spec.ToolPreferences.ExplainToolUse
			}
		}

		// Merge constraints
		if p.Spec.Constraints != nil {
			if composed.Spec.Constraints == nil {
				composed.Spec.Constraints = &langopv1alpha1.PersonaConstraints{}
			}
			if p.Spec.Constraints.MaxResponseTokens != nil {
				composed.Spec.Constraints.MaxResponseTokens = p.Spec.Constraints.MaxResponseTokens
			}
			if p.Spec.Constraints.MaxToolCalls != nil {
				composed.Spec.Constraints.MaxToolCalls = p.Spec.Constraints.MaxToolCalls
			}
			if p.Spec.Constraints.MaxKnowledgeQueries != nil {
				composed.Spec.Constraints.MaxKnowledgeQueries = p.Spec.Constraints.MaxKnowledgeQueries
			}
			if p.Spec.Constraints.ResponseTimeout != "" {
				composed.Spec.Constraints.ResponseTimeout = p.Spec.Constraints.ResponseTimeout
			}
			if p.Spec.Constraints.RequireDocumentation {
				composed.Spec.Constraints.RequireDocumentation = p.Spec.Constraints.RequireDocumentation
			}
			if len(p.Spec.Constraints.AllowedDomains) > 0 {
				composed.Spec.Constraints.AllowedDomains = p.Spec.Constraints.AllowedDomains
			}
			composed.Spec.Constraints.BlockedTopics = append(composed.Spec.Constraints.BlockedTopics, p.Spec.Constraints.BlockedTopics...)
		}
	}

	return composed
}

func (r *LanguageAgentReconciler) cleanupResources(ctx context.Context, agent *langopv1alpha1.LanguageAgent) error {
	log := log.FromContext(ctx)
	log.Info("Starting explicit resource cleanup", "agent", agent.Name, "namespace", agent.Namespace)

	// Set cleanup timeout
	cleanupTimeout := 30 * time.Second
	cleanupCtx, cancel := context.WithTimeout(ctx, cleanupTimeout)
	defer cancel()

	var cleanupErrors []error

	// 1. Cleanup HTTPRoutes
	if err := r.cleanupHTTPRoutes(cleanupCtx, agent); err != nil {
		cleanupErrors = append(cleanupErrors, fmt.Errorf("HTTPRoute cleanup failed: %w", err))
		log.Error(err, "Failed to cleanup HTTPRoutes", "agent", agent.Name)
	}

	// 2. Cleanup Ingresses
	if err := r.cleanupIngresses(cleanupCtx, agent); err != nil {
		cleanupErrors = append(cleanupErrors, fmt.Errorf("Ingress cleanup failed: %w", err))
		log.Error(err, "Failed to cleanup Ingresses", "agent", agent.Name)
	}

	// 3. Cleanup Services
	if err := r.cleanupServices(cleanupCtx, agent); err != nil {
		cleanupErrors = append(cleanupErrors, fmt.Errorf("Service cleanup failed: %w", err))
		log.Error(err, "Failed to cleanup Services", "agent", agent.Name)
	}

	// 4. Cleanup cross-namespace ReferenceGrants
	if err := r.cleanupReferenceGrants(cleanupCtx, agent); err != nil {
		cleanupErrors = append(cleanupErrors, fmt.Errorf("ReferenceGrant cleanup failed: %w", err))
		log.Error(err, "Failed to cleanup ReferenceGrants", "agent", agent.Name)
	}

	// Log summary
	if len(cleanupErrors) == 0 {
		log.Info("Resource cleanup completed successfully", "agent", agent.Name)
		return nil
	}

	// Return combined error for critical failures, but don't block deletion indefinitely
	log.Info("Resource cleanup completed with errors", "agent", agent.Name, "errorCount", len(cleanupErrors))
	for _, err := range cleanupErrors {
		log.Error(err, "Cleanup error details")
	}

	// Don't block agent deletion for cleanup failures - log and continue
	return nil
}

// cleanupHTTPRoutes deletes HTTPRoutes owned by the agent and verifies deletion
func (r *LanguageAgentReconciler) cleanupHTTPRoutes(ctx context.Context, agent *langopv1alpha1.LanguageAgent) error {
	log := log.FromContext(ctx)

	// List HTTPRoutes owned by this agent
	httpRouteList := &unstructured.UnstructuredList{}
	httpRouteList.SetGroupVersionKind(schema.GroupVersionKind{
		Group:   "gateway.networking.k8s.io",
		Version: "v1",
		Kind:    "HTTPRouteList",
	})

	// Use label selector to find resources owned by this agent
	labels := GetCommonLabels(agent.Name, "LanguageAgent")
	labelSelector := client.MatchingLabels(labels)

	if err := r.List(ctx, httpRouteList, client.InNamespace(agent.Namespace), labelSelector); err != nil {
		if errors.IsNotFound(err) {
			return nil
		}
		return fmt.Errorf("failed to list HTTPRoutes: %w", err)
	}

	// Delete each HTTPRoute and verify deletion
	for _, route := range httpRouteList.Items {
		if err := r.deleteAndVerifyResource(ctx, &route, "HTTPRoute"); err != nil {
			return fmt.Errorf("failed to delete HTTPRoute %s: %w", route.GetName(), err)
		}
		log.Info("Successfully deleted HTTPRoute", "name", route.GetName(), "namespace", route.GetNamespace())
	}

	return nil
}

// cleanupIngresses deletes Ingresses owned by the agent and verifies deletion
func (r *LanguageAgentReconciler) cleanupIngresses(ctx context.Context, agent *langopv1alpha1.LanguageAgent) error {
	log := log.FromContext(ctx)

	ingressList := &networkingv1.IngressList{}
	labels := GetCommonLabels(agent.Name, "LanguageAgent")
	labelSelector := client.MatchingLabels(labels)

	if err := r.List(ctx, ingressList, client.InNamespace(agent.Namespace), labelSelector); err != nil {
		if errors.IsNotFound(err) {
			return nil
		}
		return fmt.Errorf("failed to list Ingresses: %w", err)
	}

	// Delete each Ingress and verify deletion
	for _, ingress := range ingressList.Items {
		ingressObj := ingress // Create a copy to avoid pointer issues
		if err := r.deleteAndVerifyResource(ctx, &ingressObj, "Ingress"); err != nil {
			return fmt.Errorf("failed to delete Ingress %s: %w", ingress.Name, err)
		}
		log.Info("Successfully deleted Ingress", "name", ingress.Name, "namespace", ingress.Namespace)
	}

	return nil
}

// cleanupServices deletes Services owned by the agent and verifies deletion
func (r *LanguageAgentReconciler) cleanupServices(ctx context.Context, agent *langopv1alpha1.LanguageAgent) error {
	log := log.FromContext(ctx)

	serviceList := &corev1.ServiceList{}
	labels := GetCommonLabels(agent.Name, "LanguageAgent")
	labelSelector := client.MatchingLabels(labels)

	if err := r.List(ctx, serviceList, client.InNamespace(agent.Namespace), labelSelector); err != nil {
		if errors.IsNotFound(err) {
			return nil
		}
		return fmt.Errorf("failed to list Services: %w", err)
	}

	// Delete each Service and verify deletion
	for _, service := range serviceList.Items {
		serviceObj := service // Create a copy to avoid pointer issues
		if err := r.deleteAndVerifyResource(ctx, &serviceObj, "Service"); err != nil {
			return fmt.Errorf("failed to delete Service %s: %w", service.Name, err)
		}
		log.Info("Successfully deleted Service", "name", service.Name, "namespace", service.Namespace)
	}

	return nil
}

// cleanupReferenceGrants deletes ReferenceGrants created for cross-namespace Gateway access
func (r *LanguageAgentReconciler) cleanupReferenceGrants(ctx context.Context, agent *langopv1alpha1.LanguageAgent) error {
	log := log.FromContext(ctx)

	// ReferenceGrants are created with specific naming pattern: {agent-name}-{agent-namespace}-referencegrant
	// They could be in different namespaces (gateway namespaces), so we need to search across namespaces

	referenceGrantList := &unstructured.UnstructuredList{}
	referenceGrantList.SetGroupVersionKind(schema.GroupVersionKind{
		Group:   "gateway.networking.k8s.io",
		Version: "v1beta1",
		Kind:    "ReferenceGrantList",
	})

	// List all ReferenceGrants across all namespaces to find ones created for this agent
	if err := r.List(ctx, referenceGrantList); err != nil {
		if errors.IsNotFound(err) {
			return nil
		}
		return fmt.Errorf("failed to list ReferenceGrants: %w", err)
	}

	expectedName := fmt.Sprintf("%s-%s-referencegrant", agent.Name, agent.Namespace)

	// Delete ReferenceGrants with matching name pattern
	for _, refGrant := range referenceGrantList.Items {
		if refGrant.GetName() == expectedName {
			if err := r.deleteAndVerifyResource(ctx, &refGrant, "ReferenceGrant"); err != nil {
				return fmt.Errorf("failed to delete ReferenceGrant %s in namespace %s: %w",
					refGrant.GetName(), refGrant.GetNamespace(), err)
			}
			log.Info("Successfully deleted ReferenceGrant", "name", refGrant.GetName(),
				"namespace", refGrant.GetNamespace())
		}
	}

	return nil
}

// deleteAndVerifyResource deletes a resource and waits for deletion to be confirmed
func (r *LanguageAgentReconciler) deleteAndVerifyResource(ctx context.Context, obj client.Object, resourceType string) error {
	log := log.FromContext(ctx)

	// Delete the resource
	if err := r.Delete(ctx, obj); err != nil {
		if errors.IsNotFound(err) {
			// Already deleted
			return nil
		}
		return fmt.Errorf("failed to delete %s: %w", resourceType, err)
	}

	// Wait for deletion to be confirmed with polling
	name := obj.GetName()
	namespace := obj.GetNamespace()

	// Poll for deletion with timeout
	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return fmt.Errorf("timeout waiting for %s %s/%s deletion: %w", resourceType, namespace, name, ctx.Err())
		case <-ticker.C:
			// Check if resource still exists
			err := r.Get(ctx, types.NamespacedName{Name: name, Namespace: namespace}, obj)
			if errors.IsNotFound(err) {
				// Resource has been deleted successfully
				log.V(1).Info("Verified resource deletion", "type", resourceType, "name", name, "namespace", namespace)
				return nil
			}
			if err != nil {
				return fmt.Errorf("error checking %s deletion status: %w", resourceType, err)
			}
			// Resource still exists, continue polling
		}
	}
}

// reconcileService creates a Service for the agent's webhook server
func (r *LanguageAgentReconciler) reconcileService(ctx context.Context, agent *langopv1alpha1.LanguageAgent) error {
	labels := GetCommonLabels(agent.Name, "LanguageAgent")

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

		// All agents expose webhook server on port 8080
		service.Spec = corev1.ServiceSpec{
			Selector: labels,
			Ports: []corev1.ServicePort{
				{
					Name:       "http",
					Port:       80,
					TargetPort: intstr.FromInt(8080),
					Protocol:   corev1.ProtocolTCP,
				},
			},
			Type: corev1.ServiceTypeClusterIP,
		}

		return nil
	})

	return err
}

// reconcileWebhooks creates HTTPRoute or Ingress for webhook access
func (r *LanguageAgentReconciler) reconcileWebhooks(ctx context.Context, agent *langopv1alpha1.LanguageAgent) error {
	log := log.FromContext(ctx)

	// Get the cluster to check for domain configuration
	var domain string
	if agent.Spec.ClusterRef != "" {
		cluster := &langopv1alpha1.LanguageCluster{}
		if err := r.Get(ctx, types.NamespacedName{Name: agent.Spec.ClusterRef, Namespace: agent.Namespace}, cluster); err != nil {
			if errors.IsNotFound(err) {
				log.Info("Cluster not found, skipping webhook reconciliation", "cluster", agent.Spec.ClusterRef)
				return nil
			}
			return err
		}
		domain = cluster.Spec.Domain
	}

	// Skip webhook reconciliation if no domain is configured
	if domain == "" {
		log.Info("No domain configured, skipping webhook reconciliation")
		return nil
	}

	// Build webhook hostname: <uuid>.agents.<domain>
	hostname := fmt.Sprintf("%s.agents.%s", agent.Status.UUID, domain)

	// Check if Gateway API is available
	hasGateway, err := r.hasGatewayAPI(ctx)
	if err != nil {
		log.Error(err, "Failed to detect Gateway API availability")
		// Fall back to Ingress on detection error
		hasGateway = false
	}

	var routeReady bool
	var routeReadyMsg string

	if hasGateway {
		log.Info("Gateway API detected, creating HTTPRoute", "hostname", hostname)
		if err := r.reconcileHTTPRoute(ctx, agent, hostname); err != nil {
			// Set WebhookRouteCreated condition to false on failure
			SetCondition(&agent.Status.Conditions, langopv1alpha1.WebhookRouteCreatedCondition, metav1.ConditionFalse, "HTTPRouteCreationFailed", err.Error(), agent.Generation)
			return fmt.Errorf("failed to reconcile HTTPRoute: %w", err)
		}

		// Set WebhookRouteCreated condition to true on success
		SetCondition(&agent.Status.Conditions, langopv1alpha1.WebhookRouteCreatedCondition, metav1.ConditionTrue, "HTTPRouteCreated", "HTTPRoute created successfully", agent.Generation)

		// Check if HTTPRoute is ready
		ready, msg, err := r.checkHTTPRouteReadiness(ctx, agent.Name, agent.Namespace)
		if err != nil {
			log.Error(err, "Failed to check HTTPRoute readiness")
			routeReady = false
			routeReadyMsg = fmt.Sprintf("Failed to check readiness: %v", err)
		} else {
			routeReady = ready
			routeReadyMsg = msg
		}
	} else {
		log.Info("Gateway API not available, creating Ingress fallback", "hostname", hostname)
		if err := r.reconcileIngress(ctx, agent, hostname); err != nil {
			// Set WebhookRouteCreated condition to false on failure
			SetCondition(&agent.Status.Conditions, langopv1alpha1.WebhookRouteCreatedCondition, metav1.ConditionFalse, "IngressCreationFailed", err.Error(), agent.Generation)
			return fmt.Errorf("failed to reconcile Ingress: %w", err)
		}

		// Set WebhookRouteCreated condition to true on success
		SetCondition(&agent.Status.Conditions, langopv1alpha1.WebhookRouteCreatedCondition, metav1.ConditionTrue, "IngressCreated", "Ingress created successfully", agent.Generation)

		// Check if Ingress is ready
		ready, msg, err := r.checkIngressReadiness(ctx, agent.Name, agent.Namespace)
		if err != nil {
			log.Error(err, "Failed to check Ingress readiness")
			routeReady = false
			routeReadyMsg = fmt.Sprintf("Failed to check readiness: %v", err)
		} else {
			routeReady = ready
			routeReadyMsg = msg
		}
	}

	// Set WebhookRouteReady condition based on readiness check
	if routeReady {
		SetCondition(&agent.Status.Conditions, langopv1alpha1.WebhookRouteReadyCondition, metav1.ConditionTrue, "WebhookRouteReady", routeReadyMsg, agent.Generation)

		// Only populate WebhookURLs when route is ready
		webhookURL := fmt.Sprintf("https://%s", hostname)
		if agent.Status.WebhookURLs == nil || len(agent.Status.WebhookURLs) == 0 || agent.Status.WebhookURLs[0] != webhookURL {
			agent.Status.WebhookURLs = []string{webhookURL}
			log.Info("Updated webhook URL in status", "url", webhookURL)
		}
	} else {
		SetCondition(&agent.Status.Conditions, langopv1alpha1.WebhookRouteReadyCondition, metav1.ConditionFalse, "WebhookRouteNotReady", routeReadyMsg, agent.Generation)

		// Clear webhook URLs when route is not ready
		if len(agent.Status.WebhookURLs) > 0 {
			agent.Status.WebhookURLs = nil
			log.Info("Cleared webhook URLs from status - route not ready")
		}
	}

	// Update agent status with conditions and potentially webhook URLs
	if err := r.Status().Update(ctx, agent); err != nil {
		log.Error(err, "Failed to update agent status")
		return err
	}

	return nil
}

// detectNetworkPolicySupport detects if the cluster CNI supports NetworkPolicy enforcement
func (r *LanguageAgentReconciler) detectNetworkPolicySupport(ctx context.Context) (bool, string) {
	// Check for known CNI plugins that support NetworkPolicy
	// We detect by looking for DaemonSets or pods in kube-system namespace

	// Check for Cilium
	ciliumDS := &appsv1.DaemonSet{}
	if err := r.Get(ctx, types.NamespacedName{Name: "cilium", Namespace: "kube-system"}, ciliumDS); err == nil {
		return true, "cilium"
	}

	// Check for Calico
	calicoDS := &appsv1.DaemonSet{}
	if err := r.Get(ctx, types.NamespacedName{Name: "calico-node", Namespace: "kube-system"}, calicoDS); err == nil {
		return true, "calico"
	}

	// Check for Weave Net
	weaveDS := &appsv1.DaemonSet{}
	if err := r.Get(ctx, types.NamespacedName{Name: "weave-net", Namespace: "kube-system"}, weaveDS); err == nil {
		return true, "weave-net"
	}

	// Check for Antrea
	antreaDS := &appsv1.DaemonSet{}
	if err := r.Get(ctx, types.NamespacedName{Name: "antrea-agent", Namespace: "kube-system"}, antreaDS); err == nil {
		return true, "antrea"
	}

	// Check for Flannel (does NOT support NetworkPolicy)
	flannelDS := &appsv1.DaemonSet{}
	if err := r.Get(ctx, types.NamespacedName{Name: "kube-flannel-ds", Namespace: "kube-system"}, flannelDS); err == nil {
		return false, "flannel"
	}

	// Unknown CNI - assume not supported and warn
	return false, "unknown"
}

// hasGatewayAPI checks if Gateway API CRDs are available in the cluster with caching
func (r *LanguageAgentReconciler) hasGatewayAPI(ctx context.Context) (bool, error) {
	// Quick read lock check for cached result
	r.gatewayCache.mutex.RLock()
	if time.Since(r.gatewayCache.lastCheck) < gatewayAPICacheTTL {
		available := r.gatewayCache.available
		r.gatewayCache.mutex.RUnlock()
		return available, nil
	}
	r.gatewayCache.mutex.RUnlock()

	// Cache is stale, acquire write lock and refresh
	r.gatewayCache.mutex.Lock()
	defer r.gatewayCache.mutex.Unlock()

	// Check again in case another goroutine already refreshed
	if time.Since(r.gatewayCache.lastCheck) < gatewayAPICacheTTL {
		return r.gatewayCache.available, nil
	}

	// Perform expensive discovery
	available, err := r.discoverGatewayAPI(ctx)
	if err != nil {
		// Don't update cache on error, return stale data if available
		if !r.gatewayCache.lastCheck.IsZero() {
			return r.gatewayCache.available, nil
		}
		return false, err
	}

	// Update cache with fresh result
	r.gatewayCache.available = available
	r.gatewayCache.lastCheck = time.Now()
	return available, nil
}

// discoverGatewayAPI performs the actual API discovery without caching
func (r *LanguageAgentReconciler) discoverGatewayAPI(ctx context.Context) (bool, error) {
	// Create a discovery client from the existing client
	cfg, err := ctrl.GetConfig()
	if err != nil {
		return false, err
	}

	discoveryClient, err := discovery.NewDiscoveryClientForConfig(cfg)
	if err != nil {
		return false, err
	}

	// Check if HTTPRoute CRD exists (gateway.networking.k8s.io/v1)
	gvr := schema.GroupVersionResource{
		Group:    "gateway.networking.k8s.io",
		Version:  "v1",
		Resource: "httproutes",
	}

	_, apiResourcesList, err := discoveryClient.ServerGroupsAndResources()
	if err != nil {
		// Partial errors are acceptable - some API groups might be unavailable
		if discovery.IsGroupDiscoveryFailedError(err) {
			// Continue with partial results
		} else {
			return false, err
		}
	}

	for _, apiResources := range apiResourcesList {
		if apiResources.GroupVersion == gvr.Group+"/"+gvr.Version {
			for _, resource := range apiResources.APIResources {
				if resource.Name == gvr.Resource {
					return true, nil
				}
			}
		}
	}

	return false, nil
}

// reconcileReferenceGrant creates or updates a Gateway API ReferenceGrant for cross-namespace access
func (r *LanguageAgentReconciler) reconcileReferenceGrant(ctx context.Context, agent *langopv1alpha1.LanguageAgent, gatewayName, gatewayNamespace string) error {
	log := log.FromContext(ctx)

	// Only create ReferenceGrant if gateway is in a different namespace
	if agent.Namespace == gatewayNamespace {
		return nil
	}

	labels := GetCommonLabels(agent.Name, "LanguageAgent")
	referenceGrantName := fmt.Sprintf("%s-%s-referencegrant", agent.Name, agent.Namespace)

	// Create ReferenceGrant using unstructured to avoid Gateway API dependency
	referenceGrant := &unstructured.Unstructured{}
	referenceGrant.SetGroupVersionKind(schema.GroupVersionKind{
		Group:   "gateway.networking.k8s.io",
		Version: "v1beta1", // ReferenceGrant is in beta
		Kind:    "ReferenceGrant",
	})
	referenceGrant.SetName(referenceGrantName)
	referenceGrant.SetNamespace(gatewayNamespace) // Must be created in gateway namespace
	referenceGrant.SetLabels(labels)

	// Build ReferenceGrant spec
	spec := map[string]interface{}{
		"from": []interface{}{
			map[string]interface{}{
				"group":     "gateway.networking.k8s.io",
				"kind":      "HTTPRoute",
				"namespace": agent.Namespace,
			},
		},
		"to": []interface{}{
			map[string]interface{}{
				"group": "gateway.networking.k8s.io",
				"kind":  "Gateway",
				"name":  gatewayName,
			},
		},
	}

	// Check if ReferenceGrant already exists
	existing := &unstructured.Unstructured{}
	existing.SetGroupVersionKind(referenceGrant.GroupVersionKind())
	err := r.Get(ctx, types.NamespacedName{Name: referenceGrantName, Namespace: gatewayNamespace}, existing)

	if err != nil {
		if !errors.IsNotFound(err) {
			return err
		}
		// Create new ReferenceGrant
		referenceGrant.Object["spec"] = spec
		log.Info("Creating ReferenceGrant", "name", referenceGrantName, "namespace", gatewayNamespace, "gateway", gatewayName)
		if err := r.Create(ctx, referenceGrant); err != nil {
			return fmt.Errorf("failed to create ReferenceGrant: %w", err)
		}
	} else {
		// Update existing ReferenceGrant
		existing.Object["spec"] = spec
		existing.SetLabels(labels)
		log.Info("Updating ReferenceGrant", "name", referenceGrantName, "namespace", gatewayNamespace, "gateway", gatewayName)
		if err := r.Update(ctx, existing); err != nil {
			return fmt.Errorf("failed to update ReferenceGrant: %w", err)
		}
	}

	return nil
}

// validateGatewayTLS validates that the Gateway has appropriate TLS configuration
// Returns the protocol that should be used for webhook URLs (http or https)
func (r *LanguageAgentReconciler) validateGatewayTLS(ctx context.Context, gatewayName, gatewayNamespace string, tlsEnabled bool) (string, error) {
	log := log.FromContext(ctx)

	// Query the Gateway to check its listeners
	gateway := &unstructured.Unstructured{}
	gateway.SetGroupVersionKind(schema.GroupVersionKind{
		Group:   "gateway.networking.k8s.io",
		Version: "v1",
		Kind:    "Gateway",
	})

	err := r.Get(ctx, types.NamespacedName{Name: gatewayName, Namespace: gatewayNamespace}, gateway)
	if err != nil {
		if errors.IsNotFound(err) {
			if tlsEnabled {
				return "", fmt.Errorf("Gateway %s/%s not found, but TLS is enabled in cluster config", gatewayNamespace, gatewayName)
			}
			// Gateway doesn't exist, but TLS not required - assume HTTP
			log.Info("Gateway not found, assuming HTTP protocol", "gateway", gatewayName+"/"+gatewayNamespace)
			return "http", nil
		}
		return "", fmt.Errorf("failed to get Gateway %s/%s: %w", gatewayNamespace, gatewayName, err)
	}

	// Extract listeners from Gateway spec
	spec, exists := gateway.Object["spec"].(map[string]interface{})
	if !exists {
		return "", fmt.Errorf("Gateway %s/%s has invalid spec", gatewayNamespace, gatewayName)
	}

	listeners, exists := spec["listeners"].([]interface{})
	if !exists || len(listeners) == 0 {
		if tlsEnabled {
			return "", fmt.Errorf("Gateway %s/%s has no listeners, but TLS is enabled in cluster config", gatewayNamespace, gatewayName)
		}
		return "http", nil
	}

	// Check listeners for TLS configuration
	hasHTTPS := false
	hasHTTP := false

	for _, listenerInterface := range listeners {
		listener, ok := listenerInterface.(map[string]interface{})
		if !ok {
			continue
		}

		// Check port and protocol
		port, portExists := listener["port"]
		protocol, protocolExists := listener["protocol"]

		if portExists && protocolExists {
			portNum := int64(0)
			switch p := port.(type) {
			case int64:
				portNum = p
			case float64:
				portNum = int64(p)
			case int:
				portNum = int64(p)
			}

			protocolStr, _ := protocol.(string)

			// Check for HTTPS (port 443 or TLS in protocol/listener config)
			if portNum == 443 || protocolStr == "HTTPS" {
				hasHTTPS = true
			}
			// Check for TLS configuration in listener
			if tls, tlsExists := listener["tls"]; tlsExists && tls != nil {
				hasHTTPS = true
			}

			// Check for HTTP
			if portNum == 80 || protocolStr == "HTTP" {
				hasHTTP = true
			}
		}
	}

	// Determine protocol and validate against TLS requirements
	if tlsEnabled {
		if !hasHTTPS {
			return "", fmt.Errorf("TLS is enabled in cluster config, but Gateway %s/%s has no HTTPS listeners (port 443 or TLS configuration)", gatewayNamespace, gatewayName)
		}
		log.Info("Gateway has HTTPS listeners, using HTTPS protocol", "gateway", gatewayName+"/"+gatewayNamespace)
		return "https", nil
	} else {
		// TLS not required, prefer HTTPS if available, otherwise HTTP
		if hasHTTPS {
			log.Info("Gateway has HTTPS listeners, using HTTPS protocol", "gateway", gatewayName+"/"+gatewayNamespace)
			return "https", nil
		} else if hasHTTP {
			log.Info("Gateway has HTTP listeners, using HTTP protocol", "gateway", gatewayName+"/"+gatewayNamespace)
			return "http", nil
		} else {
			log.Info("Gateway has no recognized HTTP/HTTPS listeners, defaulting to HTTP", "gateway", gatewayName+"/"+gatewayNamespace)
			return "http", nil
		}
	}
}

// reconcileHTTPRoute creates or updates a Gateway API HTTPRoute for the agent
func (r *LanguageAgentReconciler) reconcileHTTPRoute(ctx context.Context, agent *langopv1alpha1.LanguageAgent, hostname string) error {
	log := log.FromContext(ctx)
	labels := GetCommonLabels(agent.Name, "LanguageAgent")

	// Get cluster config for Gateway configuration and TLS settings
	var gatewayName, gatewayNamespace string
	var tlsEnabled bool
	if agent.Spec.ClusterRef != "" {
		cluster := &langopv1alpha1.LanguageCluster{}
		if err := r.Get(ctx, types.NamespacedName{Name: agent.Spec.ClusterRef, Namespace: agent.Namespace}, cluster); err == nil {
			if cluster.Spec.IngressConfig != nil {
				// Extract TLS configuration
				if cluster.Spec.IngressConfig.TLS != nil {
					tlsEnabled = cluster.Spec.IngressConfig.TLS.Enabled
				}

				// Prefer new GatewayName field, fall back to deprecated GatewayClassName for backward compatibility
				if cluster.Spec.IngressConfig.GatewayName != "" {
					gatewayName = cluster.Spec.IngressConfig.GatewayName
					// Use specified namespace or default to cluster namespace
					if cluster.Spec.IngressConfig.GatewayNamespace != "" {
						gatewayNamespace = cluster.Spec.IngressConfig.GatewayNamespace
					} else {
						gatewayNamespace = agent.Namespace
					}
				} else if cluster.Spec.IngressConfig.GatewayClassName != "" {
					// Backward compatibility: treat GatewayClassName as Gateway resource name
					gatewayName = cluster.Spec.IngressConfig.GatewayClassName
					gatewayNamespace = agent.Namespace
				}
			}
		}
	}

	// Default to "default" gateway if not specified
	if gatewayName == "" {
		gatewayName = "default"
		gatewayNamespace = "default"
	}

	// Validate Gateway TLS configuration and determine protocol
	_, err := r.validateGatewayTLS(ctx, gatewayName, gatewayNamespace, tlsEnabled)
	if err != nil {
		return fmt.Errorf("Gateway TLS validation failed: %w", err)
	}

	// Create ReferenceGrant if cross-namespace Gateway reference is needed
	if err := r.reconcileReferenceGrant(ctx, agent, gatewayName, gatewayNamespace); err != nil {
		return fmt.Errorf("failed to reconcile ReferenceGrant: %w", err)
	}

	// Create HTTPRoute using unstructured to avoid Gateway API dependency
	httpRoute := &unstructured.Unstructured{}
	httpRoute.SetGroupVersionKind(schema.GroupVersionKind{
		Group:   "gateway.networking.k8s.io",
		Version: "v1",
		Kind:    "HTTPRoute",
	})
	httpRoute.SetName(agent.Name)
	httpRoute.SetNamespace(agent.Namespace)
	httpRoute.SetLabels(labels)

	// Build HTTPRoute spec
	spec := map[string]interface{}{
		"parentRefs": []map[string]interface{}{
			{
				"name":      gatewayName,
				"namespace": gatewayNamespace,
			},
		},
		"hostnames": []string{hostname},
		"rules": []map[string]interface{}{
			{
				"matches": []map[string]interface{}{
					{
						"path": map[string]interface{}{
							"type":  "PathPrefix",
							"value": "/",
						},
					},
				},
				"backendRefs": []map[string]interface{}{
					{
						"name": agent.Name,
						"port": int64(80),
					},
				},
			},
		},
	}

	// Check if HTTPRoute already exists
	existing := &unstructured.Unstructured{}
	existing.SetGroupVersionKind(httpRoute.GroupVersionKind())
	err = r.Get(ctx, types.NamespacedName{Name: agent.Name, Namespace: agent.Namespace}, existing)

	if err != nil {
		if !errors.IsNotFound(err) {
			return err
		}
		// Create new HTTPRoute
		httpRoute.Object["spec"] = spec
		// Set owner reference for automatic cleanup
		if err := controllerutil.SetControllerReference(agent, httpRoute, r.Scheme); err != nil {
			return err
		}
		log.Info("Creating HTTPRoute", "hostname", hostname, "gateway", gatewayName+"/"+gatewayNamespace)
		if err := r.Create(ctx, httpRoute); err != nil {
			return err
		}
	} else {
		// Update existing HTTPRoute
		existing.Object["spec"] = spec
		existing.SetLabels(labels)
		log.Info("Updating HTTPRoute", "hostname", hostname, "gateway", gatewayName+"/"+gatewayNamespace)
		if err := r.Update(ctx, existing); err != nil {
			return err
		}
	}

	return nil
}

// reconcileIngress creates or updates an Ingress for the agent (fallback when Gateway API unavailable)
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
												Number: 80,
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

		// Add TLS configuration if cluster has TLS enabled
		if agent.Spec.ClusterRef != "" {
			cluster := &langopv1alpha1.LanguageCluster{}
			if err := r.Get(ctx, types.NamespacedName{Name: agent.Spec.ClusterRef, Namespace: agent.Namespace}, cluster); err == nil {
				if cluster.Spec.IngressConfig != nil && cluster.Spec.IngressConfig.TLS != nil && cluster.Spec.IngressConfig.TLS.Enabled {
					secretName := cluster.Spec.IngressConfig.TLS.SecretName
					if secretName == "" {
						// Use cert-manager annotation for automatic certificate provisioning
						if ingress.Annotations == nil {
							ingress.Annotations = make(map[string]string)
						}
						if cluster.Spec.IngressConfig.TLS.IssuerRef != nil {
							kind := cluster.Spec.IngressConfig.TLS.IssuerRef.Kind
							if kind == "" {
								kind = "ClusterIssuer"
							}
							ingress.Annotations["cert-manager.io/"+strings.ToLower(kind)] = cluster.Spec.IngressConfig.TLS.IssuerRef.Name
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

				// Add IngressClassName if specified
				if cluster.Spec.IngressConfig != nil && cluster.Spec.IngressConfig.IngressClassName != "" {
					ingress.Spec.IngressClassName = &cluster.Spec.IngressConfig.IngressClassName
				}
			}
		}

		return nil
	})

	return err
}

// performSelfHealingSynthesis performs synthesis with error context for self-healing
func (r *LanguageAgentReconciler) performSelfHealingSynthesis(ctx context.Context, agent *langopv1alpha1.LanguageAgent) error {
	// Start OpenTelemetry span for self-healing synthesis
	ctx, span := agentTracer.Start(ctx, "agent.self_healing.synthesize")
	defer span.End()

	log := log.FromContext(ctx)

	// Fetch persona if referenced
	persona, err := r.fetchPersona(ctx, agent)
	if err != nil {
		log.Error(err, "Failed to fetch persona, continuing without it")
	}

	// Distill persona if available
	var distilledPersona string
	if persona != nil {
		distilledPersona, err = r.distillPersona(ctx, persona, agent)
		if err != nil {
			log.Error(err, "Failed to distill persona, continuing without it")
			distilledPersona = ""
		}
	}

	// Build error context for self-healing
	errorContext := r.buildErrorContext(agent)

	// Get last known good code for reference
	lastKnownGoodCode := ""
	if agent.Status.LastSuccessfulCode != "" {
		lastKnownGoodCode = agent.Status.LastSuccessfulCode
	}

	// Get complete tool schemas for better self-healing synthesis quality
	toolSchemas := r.getToolSchemas(ctx, agent)

	// Build synthesis request with error context
	synthReq := synthesis.AgentSynthesisRequest{
		Instructions:      agent.Spec.Instructions,
		Tools:             r.getToolNames(agent), // Kept for backward compatibility
		ToolSchemas:       toolSchemas,           // Complete schemas for better synthesis
		Models:            r.getModelNames(agent),
		PersonaText:       distilledPersona,
		AgentName:         agent.Name,
		Namespace:         agent.Namespace,
		ErrorContext:      errorContext,
		IsRetry:           true,
		AttemptNumber:     agent.Status.SelfHealingAttempts,
		LastKnownGoodCode: lastKnownGoodCode,
	}

	// Build error context string for span attribute
	errorContextStr := ""
	if len(errorContext.RuntimeErrors) > 0 {
		// Include first runtime error message
		errorContextStr = errorContext.RuntimeErrors[0].ErrorMessage
	} else if len(errorContext.ValidationErrors) > 0 {
		// Or first validation error if no runtime errors
		errorContextStr = errorContext.ValidationErrors[0]
	}

	// Add span attributes
	span.SetAttributes(
		attribute.String("agent.name", agent.Name),
		attribute.String("agent.namespace", agent.Namespace),
		attribute.Int("self_healing.attempt_number", int(agent.Status.SelfHealingAttempts)),
		attribute.String("self_healing.error_context", errorContextStr),
		attribute.Int("self_healing.runtime_errors_count", len(errorContext.RuntimeErrors)),
		attribute.Int("self_healing.validation_errors_count", len(errorContext.ValidationErrors)),
	)

	// Synthesize code with error context
	log.Info("Performing self-healing synthesis with error context",
		"agent", agent.Name,
		"attempt", agent.Status.SelfHealingAttempts,
		"runtimeErrors", len(errorContext.RuntimeErrors),
		"validationErrors", len(errorContext.ValidationErrors))

	if r.Recorder != nil {
		r.Recorder.Event(agent, corev1.EventTypeNormal, "SelfHealingSynthesisStarted",
			"Starting self-healing code synthesis with error context")
	}

	// Create synthesizer from agent's model
	synthesizer, synthesisModelName, err := r.createSynthesizer(ctx, agent)
	if err != nil {
		return fmt.Errorf("failed to create synthesizer for self-healing: %w", err)
	}

	resp, err := synthesizer.SynthesizeAgent(ctx, synthReq)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "Self-healing synthesis failed")
		if r.Recorder != nil {
			r.Recorder.Eventf(agent, corev1.EventTypeWarning, "SelfHealingSynthesisFailed",
				"Self-healing synthesis failed: %v", err)
		}
		return fmt.Errorf("self-healing synthesis failed: %w", err)
	}

	if resp.Error != "" {
		synthesisErr := fmt.Errorf("validation failed: %s", resp.Error)
		span.RecordError(synthesisErr)
		span.SetStatus(codes.Error, "Self-healing validation failed")
		if r.Recorder != nil {
			r.Recorder.Eventf(agent, corev1.EventTypeWarning, "SelfHealingValidationFailed",
				"Self-healing validation failed: %s", resp.Error)
		}
		return fmt.Errorf("self-healing validation failed: %s", resp.Error)
	}

	// Store synthesized code in ConfigMap
	codeConfigMapName := GenerateConfigMapName(agent.Name, "code")
	data := map[string]string{
		"agent.rb": resp.DSLCode,
	}

	// Store all hashes for smart change detection
	annotations := map[string]string{
		"langop.io/instructions-hash": hashString(agent.Spec.Instructions),
		"langop.io/tools-hash":        hashString(strings.Join(r.getToolNames(agent), ",")),
		"langop.io/models-hash":       hashString(strings.Join(r.getModelNames(agent), ",")),
		"langop.io/persona-hash":      hashString(strings.Join(r.getPersonaNames(agent), ",")),
		"langop.io/synthesized-at":    metav1.Now().Format("2006-01-02T15:04:05Z"),
		"langop.io/self-healing":      "true",
	}

	if err := CreateOrUpdateConfigMapWithAnnotations(ctx, r.Client, r.Scheme, agent, codeConfigMapName, agent.Namespace, data, annotations); err != nil {
		return err
	}

	// Update synthesis info in status
	now := metav1.Now()
	if agent.Status.SynthesisInfo == nil {
		agent.Status.SynthesisInfo = &langopv1alpha1.SynthesisInfo{}
	}
	agent.Status.SynthesisInfo.LastSynthesisTime = &now
	agent.Status.SynthesisInfo.SynthesisModel = synthesisModelName
	agent.Status.SynthesisInfo.SynthesisDuration = resp.DurationSeconds
	agent.Status.SynthesisInfo.CodeHash = hashString(resp.DSLCode)
	agent.Status.SynthesisInfo.InstructionsHash = hashString(agent.Spec.Instructions)
	agent.Status.SynthesisInfo.ValidationErrors = resp.ValidationErrors

	// Update agent status
	if err := r.Status().Update(ctx, agent); err != nil {
		log.Error(err, "Failed to update synthesis info in status")
		return err
	}

	log.Info("Self-healing synthesis completed successfully",
		"agent", agent.Name,
		"codeLength", len(resp.DSLCode),
		"duration", resp.DurationSeconds,
		"attempt", agent.Status.SelfHealingAttempts)

	if r.Recorder != nil {
		r.Recorder.Eventf(agent, corev1.EventTypeNormal, "SelfHealingSynthesisSucceeded",
			"Self-healing synthesis succeeded in %.2fs (attempt %d)", resp.DurationSeconds, agent.Status.SelfHealingAttempts)
	}

	// Mark span as successful
	span.SetStatus(codes.Ok, "Self-healing synthesis succeeded")
	span.SetAttributes(
		attribute.Float64("synthesis.duration_seconds", resp.DurationSeconds),
		attribute.Int("synthesis.code_length", len(resp.DSLCode)),
	)

	return nil
}

// shouldAttemptSelfHealing determines if self-healing should be triggered
func (r *LanguageAgentReconciler) shouldAttemptSelfHealing(agent *langopv1alpha1.LanguageAgent) bool {
	// Self-healing must be enabled
	if !r.SelfHealingEnabled {
		return false
	}

	// Agent has consecutive runtime failures
	if agent.Status.ConsecutiveFailures >= 2 {
		return true
	}

	// Agent has validation errors and hasn't exceeded max attempts
	if len(agent.Status.SynthesisInfo.ValidationErrors) > 0 &&
		agent.Status.SelfHealingAttempts < r.MaxSelfHealingAttempts {
		return true
	}

	return false
}

// buildErrorContext constructs error context for self-healing synthesis
func (r *LanguageAgentReconciler) buildErrorContext(agent *langopv1alpha1.LanguageAgent) *synthesis.ErrorContext {
	// Convert langopv1alpha1.RuntimeError to synthesis.RuntimeError
	var runtimeErrors []synthesis.RuntimeError
	for _, re := range agent.Status.RuntimeErrors {
		runtimeErrors = append(runtimeErrors, synthesis.RuntimeError{
			Timestamp:         re.Timestamp.Format("2006-01-02 15:04:05"),
			ErrorType:         re.ErrorType,
			ErrorMessage:      re.ErrorMessage,
			StackTrace:        re.StackTrace,
			ContainerExitCode: re.ContainerExitCode,
			SynthesisAttempt:  re.SynthesisAttempt,
		})
	}

	var validationErrors []string
	if agent.Status.SynthesisInfo != nil {
		validationErrors = agent.Status.SynthesisInfo.ValidationErrors
	}

	return &synthesis.ErrorContext{
		RuntimeErrors:       runtimeErrors,
		ValidationErrors:    validationErrors,
		LastCrashLog:        agent.Status.LastCrashLog,
		ConsecutiveFailures: agent.Status.ConsecutiveFailures,
		PreviousAttempts:    agent.Status.SelfHealingAttempts,
	}
}

// calculateBackoff returns exponential backoff duration based on attempt count
func calculateBackoff(attempts int32) time.Duration {
	// Exponential backoff: 1m, 2m, 4m, 8m, 16m (max)
	backoff := time.Minute * time.Duration(1<<attempts)
	maxBackoff := 16 * time.Minute
	if backoff > maxBackoff {
		return maxBackoff
	}
	return backoff
}

// detectPodFailures checks for pod failures and updates agent status
func (r *LanguageAgentReconciler) detectPodFailures(ctx context.Context, agent *langopv1alpha1.LanguageAgent) error {
	// Start OpenTelemetry span for failure detection
	ctx, span := agentTracer.Start(ctx, "agent.self_healing.detect")
	defer span.End()

	// Add span attributes
	span.SetAttributes(
		attribute.String("agent.name", agent.Name),
		attribute.String("agent.namespace", agent.Namespace),
	)

	log := log.FromContext(ctx)

	// List pods owned by this agent
	podList := &corev1.PodList{}
	labels := GetCommonLabels(agent.Name, "LanguageAgent")
	if err := r.List(ctx, podList, client.InNamespace(agent.Namespace), client.MatchingLabels(labels)); err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "Failed to list pods")
		return err
	}

	// Track failure detection metrics
	podFailureCount := 0
	errorPatterns := []string{}

	// Check each pod for failures
	for _, pod := range podList.Items {
		// Detect failure states
		if r.isPodFailed(&pod) {
			podFailureCount++
			log.Info("Pod failure detected", "pod", pod.Name, "status", pod.Status.Phase)

			// Extract error information
			runtimeError, crashLog, err := r.extractPodErrorInfo(ctx, &pod, agent)
			if err != nil {
				log.Error(err, "Failed to extract pod error info")
				continue
			}

			// Update agent status with runtime error
			if runtimeError != nil {
				// Collect error pattern for span
				if runtimeError.ErrorMessage != "" {
					errorPatterns = append(errorPatterns, runtimeError.ErrorMessage)
				}

				// Append to runtime errors (keep last 10)
				agent.Status.RuntimeErrors = append(agent.Status.RuntimeErrors, *runtimeError)
				if len(agent.Status.RuntimeErrors) > 10 {
					agent.Status.RuntimeErrors = agent.Status.RuntimeErrors[len(agent.Status.RuntimeErrors)-10:]
				}

				agent.Status.LastCrashLog = crashLog
				agent.Status.ConsecutiveFailures++
				agent.Status.FailureReason = "Runtime"

				// Update status
				if err := r.Status().Update(ctx, agent); err != nil {
					log.Error(err, "Failed to update agent status with runtime error")
					span.RecordError(err)
					span.SetStatus(codes.Error, "Failed to update agent status")
					return err
				}

				// Record event
				if r.Recorder != nil {
					r.Recorder.Eventf(agent, corev1.EventTypeWarning, "RuntimeError",
						"Pod %s failed: %s", pod.Name, runtimeError.ErrorMessage)
				}
			}
		}
	}

	// Add failure detection metrics to span
	span.SetAttributes(
		attribute.Int("agent.pod_failures", podFailureCount),
		attribute.StringSlice("agent.error_patterns", errorPatterns),
	)

	// Set span status based on detection results
	if podFailureCount > 0 {
		span.SetStatus(codes.Ok, fmt.Sprintf("Detected %d pod failures", podFailureCount))
	} else {
		span.SetStatus(codes.Ok, "No pod failures detected")
	}

	return nil
}

// isPodFailed checks if a pod is in a failed state
func (r *LanguageAgentReconciler) isPodFailed(pod *corev1.Pod) bool {
	// Check pod phase
	if pod.Status.Phase == corev1.PodFailed {
		return true
	}

	// Check for CrashLoopBackOff and other failure states
	for _, containerStatus := range pod.Status.ContainerStatuses {
		if containerStatus.State.Waiting != nil {
			reason := containerStatus.State.Waiting.Reason
			if reason == "CrashLoopBackOff" || reason == "Error" ||
				reason == "RunContainerError" || reason == "ImagePullBackOff" {
				return true
			}
		}

		// Check terminated state with non-zero exit code
		if containerStatus.State.Terminated != nil {
			if containerStatus.State.Terminated.ExitCode != 0 {
				return true
			}
		}
	}

	return false
}

// extractPodErrorInfo extracts error details and logs from a failed pod
func (r *LanguageAgentReconciler) extractPodErrorInfo(ctx context.Context, pod *corev1.Pod, agent *langopv1alpha1.LanguageAgent) (*langopv1alpha1.RuntimeError, string, error) {
	runtimeError := &langopv1alpha1.RuntimeError{
		Timestamp:        metav1.Now(),
		SynthesisAttempt: agent.Status.SelfHealingAttempts,
	}

	// Extract exit code and error message from container status
	for _, containerStatus := range pod.Status.ContainerStatuses {
		if containerStatus.Name == "agent" {
			if containerStatus.State.Waiting != nil {
				runtimeError.ErrorType = containerStatus.State.Waiting.Reason
				runtimeError.ErrorMessage = containerStatus.State.Waiting.Message
			}

			if containerStatus.State.Terminated != nil {
				runtimeError.ContainerExitCode = containerStatus.State.Terminated.ExitCode
				if runtimeError.ErrorMessage == "" {
					runtimeError.ErrorMessage = containerStatus.State.Terminated.Message
				}
				if containerStatus.State.Terminated.Reason != "" {
					runtimeError.ErrorType = containerStatus.State.Terminated.Reason
				}
			}
		}
	}

	// Extract last 100 lines of logs from the agent container
	// TODO: Implement proper log extraction using Kubernetes client-go
	// For now, we'll use the error message from the container status
	crashLog := fmt.Sprintf("Pod %s failed. Container status: %+v", pod.Name, pod.Status.ContainerStatuses)

	// If we still don't have an error message, use a generic one
	if runtimeError.ErrorMessage == "" {
		runtimeError.ErrorMessage = fmt.Sprintf("Pod %s failed with status %s", pod.Name, pod.Status.Phase)
	}

	return runtimeError, crashLog, nil
}

// validateImageRegistry validates that the agent's container image registry is in the whitelist
func (r *LanguageAgentReconciler) validateImageRegistry(agent *langopv1alpha1.LanguageAgent) error {
	// Skip validation if no whitelist configured
	if len(r.AllowedRegistries) == 0 {
		return nil
	}

	return validation.ValidateImageRegistry(agent.Spec.Image, r.AllowedRegistries)
}

// checkHTTPRouteReadiness checks if an HTTPRoute is ready to serve traffic
// Returns (isReady, statusMessage, error)
func (r *LanguageAgentReconciler) checkHTTPRouteReadiness(ctx context.Context, name, namespace string) (bool, string, error) {
	// Get HTTPRoute using unstructured to avoid Gateway API dependency
	httpRoute := &unstructured.Unstructured{}
	httpRoute.SetGroupVersionKind(schema.GroupVersionKind{
		Group:   "gateway.networking.k8s.io",
		Version: "v1",
		Kind:    "HTTPRoute",
	})

	err := r.Get(ctx, types.NamespacedName{Name: name, Namespace: namespace}, httpRoute)
	if err != nil {
		if errors.IsNotFound(err) {
			return false, "HTTPRoute not found", nil
		}
		return false, "", fmt.Errorf("failed to get HTTPRoute: %w", err)
	}

	// Check status conditions
	status, found, err := unstructured.NestedMap(httpRoute.Object, "status")
	if err != nil {
		return false, "", fmt.Errorf("failed to get HTTPRoute status: %w", err)
	}
	if !found {
		return false, "HTTPRoute status not available", nil
	}

	// Check parents (Gateway references)
	parents, found, err := unstructured.NestedSlice(status, "parents")
	if err != nil {
		return false, "", fmt.Errorf("failed to get HTTPRoute parents status: %w", err)
	}
	if !found || len(parents) == 0 {
		return false, "HTTPRoute has no parent Gateway status", nil
	}

	// Check if any parent is ready (Accepted and Programmed)
	for _, parentInterface := range parents {
		parent, ok := parentInterface.(map[string]interface{})
		if !ok {
			continue
		}

		conditions, found, err := unstructured.NestedSlice(parent, "conditions")
		if err != nil || !found {
			continue
		}

		var accepted, programmed bool
		for _, condInterface := range conditions {
			cond, ok := condInterface.(map[string]interface{})
			if !ok {
				continue
			}

			condType, _, _ := unstructured.NestedString(cond, "type")
			condStatus, _, _ := unstructured.NestedString(cond, "status")

			if condType == "Accepted" && condStatus == "True" {
				accepted = true
			}
			if condType == "Programmed" && condStatus == "True" {
				programmed = true
			}
		}

		if accepted && programmed {
			return true, "HTTPRoute is ready and programmed", nil
		}
	}

	return false, "HTTPRoute is not ready - waiting for Gateway to accept and program route", nil
}

// checkIngressReadiness checks if an Ingress is ready to serve traffic
// Returns (isReady, statusMessage, error)
func (r *LanguageAgentReconciler) checkIngressReadiness(ctx context.Context, name, namespace string) (bool, string, error) {
	ingress := &networkingv1.Ingress{}
	err := r.Get(ctx, types.NamespacedName{Name: name, Namespace: namespace}, ingress)
	if err != nil {
		if errors.IsNotFound(err) {
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

// SetupWithManager sets up the controller with the Manager.
func (r *LanguageAgentReconciler) SetupWithManager(mgr ctrl.Manager, concurrency int) error {
	// Set defaults for self-healing
	if r.MaxSelfHealingAttempts == 0 {
		r.MaxSelfHealingAttempts = 5
	}

	return ctrl.NewControllerManagedBy(mgr).
		For(&langopv1alpha1.LanguageAgent{}).
		Owns(&appsv1.Deployment{}).
		Owns(&batchv1.CronJob{}).
		Owns(&corev1.ConfigMap{}).
		Owns(&corev1.Service{}).
		Owns(&networkingv1.NetworkPolicy{}).
		Owns(&networkingv1.Ingress{}).
		Owns(&corev1.Pod{}).
		Complete(r)
}
