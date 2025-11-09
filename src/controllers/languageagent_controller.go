package controllers

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/go-logr/logr"
	"github.com/google/uuid"
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

	langopv1alpha1 "github.com/based/language-operator/api/v1alpha1"
	"github.com/based/language-operator/pkg/synthesis"
)

// LanguageAgentReconciler reconciles a LanguageAgent object
type LanguageAgentReconciler struct {
	client.Client
	Scheme         *runtime.Scheme
	Log            logr.Logger
	Synthesizer    synthesis.AgentSynthesizer
	SynthesisModel string
	Recorder       record.EventRecorder
}

//+kubebuilder:rbac:groups=langop.io,resources=languageagents,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=langop.io,resources=languageagents/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=langop.io,resources=languageagents/finalizers,verbs=update
//+kubebuilder:rbac:groups=langop.io,resources=languagepersonas,verbs=get;list;watch
//+kubebuilder:rbac:groups=langop.io,resources=languageclusters,verbs=get;list;watch
//+kubebuilder:rbac:groups=apps,resources=deployments,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=batch,resources=cronjobs,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups="",resources=configmaps,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups="",resources=persistentvolumeclaims,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups="",resources=services,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=networking.k8s.io,resources=networkpolicies,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=networking.k8s.io,resources=ingresses,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=gateway.networking.k8s.io,resources=httproutes,verbs=get;list;watch;create;update;patch;delete

// Reconcile is part of the main kubernetes reconciliation loop
func (r *LanguageAgentReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log := log.FromContext(ctx)

	// Fetch the LanguageAgent instance
	agent := &langopv1alpha1.LanguageAgent{}
	if err := r.Get(ctx, req.NamespacedName, agent); err != nil {
		if errors.IsNotFound(err) {
			return ctrl.Result{}, nil
		}
		log.Error(err, "Failed to get LanguageAgent")
		return ctrl.Result{}, err
	}

	// Handle deletion
	if !agent.DeletionTimestamp.IsZero() {
		if controllerutil.ContainsFinalizer(agent, FinalizerName) {
			if err := r.cleanupResources(ctx, agent); err != nil {
				return ctrl.Result{}, err
			}
			controllerutil.RemoveFinalizer(agent, FinalizerName)
			if err := r.Update(ctx, agent); err != nil {
				return ctrl.Result{}, err
			}
		}
		return ctrl.Result{}, nil
	}

	// Add finalizer if not present
	if !controllerutil.ContainsFinalizer(agent, FinalizerName) {
		controllerutil.AddFinalizer(agent, FinalizerName)
		if err := r.Update(ctx, agent); err != nil {
			return ctrl.Result{}, err
		}
	}

	// Synthesize agent code from instructions (if synthesizer is configured)
	if r.Synthesizer != nil && agent.Spec.Instructions != "" {
		if err := r.reconcileCodeConfigMap(ctx, agent); err != nil {
			log.Error(err, "Failed to synthesize/reconcile agent code")
			SetCondition(&agent.Status.Conditions, "Synthesized", metav1.ConditionFalse, "SynthesisFailed", err.Error(), agent.Generation)
			r.Status().Update(ctx, agent)
			return ctrl.Result{}, err
		}
		SetCondition(&agent.Status.Conditions, "Synthesized", metav1.ConditionTrue, "CodeGenerated", "Agent code synthesized successfully", agent.Generation)
	}

	// Reconcile ConfigMap
	if err := r.reconcileConfigMap(ctx, agent); err != nil {
		log.Error(err, "Failed to reconcile ConfigMap")
		SetCondition(&agent.Status.Conditions, "Ready", metav1.ConditionFalse, "ConfigMapError", err.Error(), agent.Generation)
		r.Status().Update(ctx, agent)
		return ctrl.Result{}, err
	}

	// Reconcile PVC for workspace if enabled
	if err := r.reconcilePVC(ctx, agent); err != nil {
		log.Error(err, "Failed to reconcile PVC")
		SetCondition(&agent.Status.Conditions, "Ready", metav1.ConditionFalse, "PVCError", err.Error(), agent.Generation)
		r.Status().Update(ctx, agent)
		return ctrl.Result{}, err
	}

	// Reconcile NetworkPolicy for network isolation
	if err := r.reconcileNetworkPolicy(ctx, agent); err != nil {
		log.Error(err, "Failed to reconcile NetworkPolicy")
		SetCondition(&agent.Status.Conditions, "Ready", metav1.ConditionFalse, "NetworkPolicyError", err.Error(), agent.Generation)
		r.Status().Update(ctx, agent)
		return ctrl.Result{}, err
	}

	// Ensure agent has a UUID for webhook routing
	if agent.Status.UUID == "" {
		agent.Status.UUID = uuid.New().String()
		if err := r.Status().Update(ctx, agent); err != nil {
			log.Error(err, "Failed to update agent UUID")
			return ctrl.Result{}, err
		}
		log.Info("Generated UUID for agent", "uuid", agent.Status.UUID)
	}

	// Reconcile Service for agent webhook server (all agents expose port 8080)
	if err := r.reconcileService(ctx, agent); err != nil {
		log.Error(err, "Failed to reconcile Service")
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
	switch agent.Spec.ExecutionMode {
	case "autonomous", "interactive", "event-driven", "":
		if err := r.reconcileDeployment(ctx, agent); err != nil {
			log.Error(err, "Failed to reconcile Deployment")
			SetCondition(&agent.Status.Conditions, "Ready", metav1.ConditionFalse, "DeploymentError", err.Error(), agent.Generation)
			r.Status().Update(ctx, agent)
			return ctrl.Result{}, err
		}
	case "scheduled":
		if err := r.reconcileCronJob(ctx, agent); err != nil {
			log.Error(err, "Failed to reconcile CronJob")
			SetCondition(&agent.Status.Conditions, "Ready", metav1.ConditionFalse, "CronJobError", err.Error(), agent.Generation)
			r.Status().Update(ctx, agent)
			return ctrl.Result{}, err
		}
	}

	// Update status
	agent.Status.Phase = "Running"
	SetCondition(&agent.Status.Conditions, "Ready", metav1.ConditionTrue, "ReconcileSuccess", "LanguageAgent is ready", agent.Generation)

	if err := r.Status().Update(ctx, agent); err != nil {
		log.Error(err, "Failed to update LanguageAgent status")
		return ctrl.Result{}, err
	}

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
		// Compare current vs previous hashes for smart change detection
		currentInstructionsHash := hashString(agent.Spec.Instructions)
		previousInstructionsHash := existingCM.Annotations["langop.io/instructions-hash"]

		currentToolsHash := hashString(strings.Join(r.getToolNames(agent), ","))
		previousToolsHash := existingCM.Annotations["langop.io/tools-hash"]

		currentModelsHash := hashString(strings.Join(r.getModelNames(agent), ","))
		previousModelsHash := existingCM.Annotations["langop.io/models-hash"]

		personaRef := ""
		if agent.Spec.PersonaRef != nil {
			personaRef = agent.Spec.PersonaRef.Name
		}
		currentPersonaHash := hashString(personaRef)
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

		// Build synthesis request
		synthReq := synthesis.AgentSynthesisRequest{
			Instructions: agent.Spec.Instructions,
			Tools:        r.getToolNames(agent),
			Models:       r.getModelNames(agent),
			PersonaText:  distilledPersona,
			AgentName:    agent.Name,
			Namespace:    agent.Namespace,
		}

		// Synthesize code
		log.Info("Synthesizing agent code", "agent", agent.Name)
		if r.Recorder != nil {
			r.Recorder.Event(agent, corev1.EventTypeNormal, "SynthesisStarted", "Starting code synthesis from natural language instructions")
		}

		resp, err := r.Synthesizer.SynthesizeAgent(ctx, synthReq)
		if err != nil {
			if r.Recorder != nil {
				r.Recorder.Eventf(agent, corev1.EventTypeWarning, "SynthesisFailed", "Code synthesis failed: %v", err)
			}
			return fmt.Errorf("synthesis failed: %w", err)
		}

		if resp.Error != "" {
			if r.Recorder != nil {
				r.Recorder.Eventf(agent, corev1.EventTypeWarning, "ValidationFailed", "Synthesized code validation failed: %s", resp.Error)
			}
			return fmt.Errorf("synthesis validation failed: %s", resp.Error)
		}

		dslCode = resp.DSLCode
		log.Info("Agent code synthesized successfully",
			"agent", agent.Name,
			"codeLength", len(dslCode),
			"duration", resp.DurationSeconds)

		if r.Recorder != nil {
			r.Recorder.Eventf(agent, corev1.EventTypeNormal, "SynthesisSucceeded", "Code synthesized successfully in %.2fs", resp.DurationSeconds)
		}

		// Update synthesis info in status
		now := metav1.Now()
		if agent.Status.SynthesisInfo == nil {
			agent.Status.SynthesisInfo = &langopv1alpha1.SynthesisInfo{}
		}
		agent.Status.SynthesisInfo.LastSynthesisTime = &now
		agent.Status.SynthesisInfo.SynthesisModel = r.SynthesisModel
		agent.Status.SynthesisInfo.SynthesisDuration = resp.DurationSeconds
		agent.Status.SynthesisInfo.CodeHash = hashString(dslCode)
		agent.Status.SynthesisInfo.InstructionsHash = hashString(agent.Spec.Instructions)
		agent.Status.SynthesisInfo.ValidationErrors = resp.ValidationErrors
		if agent.Status.SynthesisInfo.SynthesisAttempts == 0 || needsSynthesis {
			agent.Status.SynthesisInfo.SynthesisAttempts++
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
	personaRef := ""
	if agent.Spec.PersonaRef != nil {
		personaRef = agent.Spec.PersonaRef.Name
	}

	annotations := map[string]string{
		"langop.io/instructions-hash": hashString(agent.Spec.Instructions),
		"langop.io/tools-hash":        hashString(strings.Join(r.getToolNames(agent), ",")),
		"langop.io/models-hash":       hashString(strings.Join(r.getModelNames(agent), ",")),
		"langop.io/persona-hash":      hashString(personaRef),
		"langop.io/synthesized-at":    metav1.Now().Format("2006-01-02T15:04:05Z"),
	}

	return CreateOrUpdateConfigMapWithAnnotations(ctx, r.Client, r.Scheme, agent, codeConfigMapName, agent.Namespace, data, annotations)
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

	return r.Synthesizer.DistillPersona(ctx, personaInfo, agentCtx)
}

// getToolNames extracts tool names from agent's toolRefs
func (r *LanguageAgentReconciler) getToolNames(agent *langopv1alpha1.LanguageAgent) []string {
	var names []string
	for _, ref := range agent.Spec.ToolRefs {
		names = append(names, ref.Name)
	}
	return names
}

// getModelNames extracts model names from agent's modelRefs
func (r *LanguageAgentReconciler) getModelNames(agent *langopv1alpha1.LanguageAgent) []string {
	var names []string
	for _, ref := range agent.Spec.ModelRefs {
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

func (r *LanguageAgentReconciler) reconcilePVC(ctx context.Context, agent *langopv1alpha1.LanguageAgent) error {
	// Skip if workspace is not enabled
	if agent.Spec.Workspace == nil || !agent.Spec.Workspace.Enabled {
		return nil
	}

	// Determine target namespace - always use agent's namespace
	// If cluster ref is set, verify cluster exists in same namespace
	targetNamespace := agent.Namespace
	if agent.Spec.ClusterRef != "" {
		cluster := &langopv1alpha1.LanguageCluster{}
		if err := r.Get(ctx, types.NamespacedName{Name: agent.Spec.ClusterRef, Namespace: agent.Namespace}, cluster); err != nil {
			return err
		}
		if cluster.Status.Phase != "Ready" {
			return fmt.Errorf("cluster %s is not ready yet", agent.Spec.ClusterRef)
		}
		// Cluster is in same namespace - agent.Namespace is correct
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

	// If cluster ref is set, verify cluster exists in same namespace
	if agent.Spec.ClusterRef != "" {
		cluster := &langopv1alpha1.LanguageCluster{}
		if err := r.Get(ctx, types.NamespacedName{Name: agent.Spec.ClusterRef, Namespace: agent.Namespace}, cluster); err != nil {
			return err
		}

		// Wait for cluster to be ready
		if cluster.Status.Phase != "Ready" {
			return fmt.Errorf("cluster %s is not ready yet", agent.Spec.ClusterRef)
		}

		// Add cluster label
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
				Env:   r.buildAgentEnv(agent, modelURLs, modelNames, toolURLs, persona),
			},
		}

		// Append sidecar tool containers
		containers = append(containers, sidecarContainers...)

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
					Containers: containers,
					SecurityContext: &corev1.PodSecurityContext{
						FSGroup: ptr.To[int64](101), // langop group
					},
				},
			},
		}

		// Add resource requirements if specified
		deployment.Spec.Template.Spec.Containers[0].Resources = agent.Spec.Resources

		// Initialize volumes and volume mounts
		volumes := []corev1.Volume{}
		volumeMounts := []corev1.VolumeMount{}

		// Add code ConfigMap volume if synthesizer is configured and instructions exist
		if r.Synthesizer != nil && agent.Spec.Instructions != "" {
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

		// Apply volumes and volume mounts to deployment
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

	// If cluster ref is set, verify cluster exists in same namespace
	if agent.Spec.ClusterRef != "" {
		cluster := &langopv1alpha1.LanguageCluster{}
		if err := r.Get(ctx, types.NamespacedName{Name: agent.Spec.ClusterRef, Namespace: agent.Namespace}, cluster); err != nil {
			return err
		}

		// Wait for cluster to be ready
		if cluster.Status.Phase != "Ready" {
			return fmt.Errorf("cluster %s is not ready yet", agent.Spec.ClusterRef)
		}

		// Add cluster label
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
				Env:   r.buildAgentEnv(agent, modelURLs, modelNames, toolURLs, persona),
			},
		}

		// Append sidecar tool containers
		containers = append(containers, sidecarContainers...)

		cronJob.Spec = batchv1.CronJobSpec{
			Schedule: schedule,
			JobTemplate: batchv1.JobTemplateSpec{
				Spec: batchv1.JobSpec{
					Template: corev1.PodTemplateSpec{
						ObjectMeta: metav1.ObjectMeta{
							Labels: labels,
						},
						Spec: corev1.PodSpec{
							RestartPolicy: corev1.RestartPolicyOnFailure,
							Containers:    containers,
							SecurityContext: &corev1.PodSecurityContext{
								FSGroup: ptr.To[int64](101), // langop group
							},
						},
					},
				},
			},
		}

		// Add resource requirements if specified
		cronJob.Spec.JobTemplate.Spec.Template.Spec.Containers[0].Resources = agent.Spec.Resources

		// Initialize volumes and volume mounts
		volumes := []corev1.Volume{}
		volumeMounts := []corev1.VolumeMount{}

		// Add code ConfigMap volume if synthesizer is configured and instructions exist
		if r.Synthesizer != nil && agent.Spec.Instructions != "" {
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

		// Apply volumes and volume mounts to cronjob
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

	// Build NetworkPolicy using helper from utils.go
	networkPolicy := BuildEgressNetworkPolicy(
		agent.Name,
		agent.Namespace,
		labels,
		agent.Spec.Egress,
	)

	// Set owner reference so NetworkPolicy is cleaned up with agent
	if err := controllerutil.SetControllerReference(agent, networkPolicy, r.Scheme); err != nil {
		return fmt.Errorf("failed to set owner reference: %w", err)
	}

	// Create or update the NetworkPolicy
	existingPolicy := &networkingv1.NetworkPolicy{}
	err := r.Get(ctx, types.NamespacedName{Name: networkPolicy.Name, Namespace: networkPolicy.Namespace}, existingPolicy)

	if err != nil {
		if errors.IsNotFound(err) {
			// Create new NetworkPolicy
			if err := r.Create(ctx, networkPolicy); err != nil {
				return fmt.Errorf("failed to create NetworkPolicy: %w", err)
			}
			return nil
		}
		return fmt.Errorf("failed to get NetworkPolicy: %w", err)
	}

	// Update existing NetworkPolicy
	existingPolicy.Spec = networkPolicy.Spec
	existingPolicy.Labels = networkPolicy.Labels
	if err := r.Update(ctx, existingPolicy); err != nil {
		return fmt.Errorf("failed to update NetworkPolicy: %w", err)
	}

	return nil
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

		container := corev1.Container{
			Name:  fmt.Sprintf("tool-%s", tool.Name),
			Image: tool.Spec.Image,
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

func (r *LanguageAgentReconciler) buildAgentEnv(agent *langopv1alpha1.LanguageAgent, modelURLs []string, modelNames []string, toolURLs []string, persona *langopv1alpha1.LanguagePersona) []corev1.EnvVar {
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
	// Return nil if no persona is referenced
	if agent.Spec.PersonaRef == nil {
		return nil, nil
	}

	// Determine namespace
	namespace := agent.Spec.PersonaRef.Namespace
	if namespace == "" {
		namespace = agent.Namespace
	}

	// Fetch the LanguagePersona
	persona := &langopv1alpha1.LanguagePersona{}
	if err := r.Get(ctx, types.NamespacedName{Name: agent.Spec.PersonaRef.Name, Namespace: namespace}, persona); err != nil {
		if errors.IsNotFound(err) {
			return nil, fmt.Errorf("persona %s/%s not found", namespace, agent.Spec.PersonaRef.Name)
		}
		return nil, fmt.Errorf("failed to get persona %s/%s: %w", namespace, agent.Spec.PersonaRef.Name, err)
	}

	// Check if persona is ready
	if persona.Status.Phase != "Ready" {
		return nil, fmt.Errorf("persona %s/%s is not ready (phase: %s)", namespace, agent.Spec.PersonaRef.Name, persona.Status.Phase)
	}

	return persona, nil
}

func (r *LanguageAgentReconciler) cleanupResources(ctx context.Context, agent *langopv1alpha1.LanguageAgent) error {
	// Resources will be cleaned up automatically via owner references
	return nil
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

	if hasGateway {
		log.Info("Gateway API detected, creating HTTPRoute", "hostname", hostname)
		if err := r.reconcileHTTPRoute(ctx, agent, hostname); err != nil {
			return fmt.Errorf("failed to reconcile HTTPRoute: %w", err)
		}
	} else {
		log.Info("Gateway API not available, creating Ingress fallback", "hostname", hostname)
		if err := r.reconcileIngress(ctx, agent, hostname); err != nil {
			return fmt.Errorf("failed to reconcile Ingress: %w", err)
		}
	}

	// Update agent status with webhook URL
	webhookURL := fmt.Sprintf("https://%s", hostname)
	if agent.Status.WebhookURLs == nil || len(agent.Status.WebhookURLs) == 0 || agent.Status.WebhookURLs[0] != webhookURL {
		agent.Status.WebhookURLs = []string{webhookURL}
		if err := r.Status().Update(ctx, agent); err != nil {
			log.Error(err, "Failed to update webhook URLs in status")
			return err
		}
		log.Info("Updated webhook URL in status", "url", webhookURL)
	}

	return nil
}

// hasGatewayAPI checks if Gateway API CRDs are available in the cluster
func (r *LanguageAgentReconciler) hasGatewayAPI(ctx context.Context) (bool, error) {
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

// reconcileHTTPRoute creates or updates a Gateway API HTTPRoute for the agent
func (r *LanguageAgentReconciler) reconcileHTTPRoute(ctx context.Context, agent *langopv1alpha1.LanguageAgent, hostname string) error {
	log := log.FromContext(ctx)
	labels := GetCommonLabels(agent.Name, "LanguageAgent")

	// Get cluster config for Gateway configuration
	var gatewayName, gatewayNamespace string
	if agent.Spec.ClusterRef != "" {
		cluster := &langopv1alpha1.LanguageCluster{}
		if err := r.Get(ctx, types.NamespacedName{Name: agent.Spec.ClusterRef, Namespace: agent.Namespace}, cluster); err == nil {
			if cluster.Spec.IngressConfig != nil && cluster.Spec.IngressConfig.GatewayClassName != "" {
				// Use specified gateway
				gatewayName = cluster.Spec.IngressConfig.GatewayClassName
				gatewayNamespace = agent.Namespace
			}
		}
	}

	// Default to "default" gateway if not specified
	if gatewayName == "" {
		gatewayName = "default"
		gatewayNamespace = "default"
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
	err := r.Get(ctx, types.NamespacedName{Name: agent.Name, Namespace: agent.Namespace}, existing)

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

// SetupWithManager sets up the controller with the Manager.
func (r *LanguageAgentReconciler) SetupWithManager(mgr ctrl.Manager, concurrency int) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&langopv1alpha1.LanguageAgent{}).
		Owns(&appsv1.Deployment{}).
		Owns(&batchv1.CronJob{}).
		Owns(&corev1.ConfigMap{}).
		Owns(&corev1.Service{}).
		Owns(&networkingv1.NetworkPolicy{}).
		Owns(&networkingv1.Ingress{}).
		Complete(r)
}
