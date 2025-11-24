package controllers

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/go-logr/logr"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/tools/record"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"

	langopv1alpha1 "github.com/language-operator/language-operator/api/v1alpha1"
	"github.com/language-operator/language-operator/pkg/synthesis"
)

// LearningReconciler reconciles learning events and triggers re-synthesis
type LearningReconciler struct {
	client.Client
	Scheme               *runtime.Scheme
	Log                  logr.Logger
	Recorder             record.EventRecorder
	Synthesizer          synthesis.AgentSynthesizer  // For re-synthesis with task_synthesis.tmpl
	ConfigMapManager     *synthesis.ConfigMapManager // For versioned ConfigMap management
	LearningEnabled      bool
	LearningThreshold    int32         // Number of execution traces before triggering learning
	LearningInterval     time.Duration // Minimum interval between learning attempts
	MaxVersions          int32         // Maximum number of ConfigMap versions to keep
	PatternConfidenceMin float64       // Minimum confidence threshold for pattern detection
}

// LearningEvent represents a learning trigger event
type LearningEvent struct {
	AgentName  string    `json:"agentName"`
	Namespace  string    `json:"namespace"`
	TaskName   string    `json:"taskName"`
	EventType  string    `json:"eventType"` // "traces_accumulated", "error_threshold", "manual_trigger"
	TraceCount int32     `json:"traceCount,omitempty"`
	ErrorRate  float64   `json:"errorRate,omitempty"`
	Confidence float64   `json:"confidence,omitempty"`
	Timestamp  time.Time `json:"timestamp"`
}

// TaskLearningStatus tracks learning status for individual tasks
type TaskLearningStatus struct {
	TaskName            string    `json:"taskName"`
	TraceCount          int32     `json:"traceCount"`
	LastLearningAttempt time.Time `json:"lastLearningAttempt"`
	LearningAttempts    int32     `json:"learningAttempts"`
	CurrentVersion      int32     `json:"currentVersion"`
	IsSymbolic          bool      `json:"isSymbolic"`
	PatternConfidence   float64   `json:"patternConfidence"`
	LastTraceTimestamp  time.Time `json:"lastTraceTimestamp"`
	ErrorRate           float64   `json:"errorRate"`
	CommonPattern       string    `json:"commonPattern"`
	UniquePatternCount  int32     `json:"uniquePatternCount"`
}

// TaskTrace represents an execution trace for pattern detection
type TaskTrace struct {
	TaskName     string                 `json:"taskName"`
	Timestamp    time.Time              `json:"timestamp"`
	Inputs       map[string]interface{} `json:"inputs"`
	Outputs      map[string]interface{} `json:"outputs"`
	ToolCalls    []ToolCall             `json:"toolCalls"`
	Duration     time.Duration          `json:"duration"`
	Success      bool                   `json:"success"`
	ErrorMessage string                 `json:"errorMessage,omitempty"`
}

// ToolCall represents a tool call within a task execution
type ToolCall struct {
	ToolName   string                 `json:"toolName"`
	Method     string                 `json:"method"`
	Parameters map[string]interface{} `json:"parameters"`
	Result     interface{}            `json:"result"`
	Duration   time.Duration          `json:"duration"`
	Success    bool                   `json:"success"`
}

// PatternAnalysis represents the result of pattern detection analysis
type PatternAnalysis struct {
	TaskName           string  `json:"taskName"`
	IsDeterministic    bool    `json:"isDeterministic"`
	Confidence         float64 `json:"confidence"`
	CommonPattern      string  `json:"commonPattern"`
	ConsistencyScore   float64 `json:"consistencyScore"`
	UniquePatternCount int32   `json:"uniquePatternCount"`
	RecommendedCode    string  `json:"recommendedCode,omitempty"`
	Explanation        string  `json:"explanation"`
}

var learningTracer = otel.Tracer("language-operator/learning")

//+kubebuilder:rbac:groups=langop.io,resources=languageagents,verbs=get;list;watch;update;patch
//+kubebuilder:rbac:groups="",resources=configmaps,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups="",resources=events,verbs=create;patch
//+kubebuilder:rbac:groups=apps,resources=deployments,verbs=get;list;watch;update;patch

// Reconcile handles learning events and triggers re-synthesis when appropriate
func (r *LearningReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	ctx, span := learningTracer.Start(ctx, "learning.reconcile")
	defer span.End()

	span.SetAttributes(
		attribute.String("learning.agent_name", req.Name),
		attribute.String("learning.namespace", req.Namespace),
	)

	log := r.Log.WithValues("agent", req.NamespacedName)

	if !r.LearningEnabled {
		log.V(1).Info("Learning disabled, skipping reconciliation")
		return ctrl.Result{}, nil
	}

	log.Info("Processing learning event", "agent", req.NamespacedName)

	// Get the LanguageAgent
	var agent langopv1alpha1.LanguageAgent
	if err := r.Get(ctx, req.NamespacedName, &agent); err != nil {
		if errors.IsNotFound(err) {
			log.Info("LanguageAgent not found, may have been deleted")
			return ctrl.Result{}, nil
		}
		span.RecordError(err)
		span.SetStatus(codes.Error, "Failed to get LanguageAgent")
		return ctrl.Result{}, fmt.Errorf("failed to get LanguageAgent: %w", err)
	}

	// Check if learning is enabled for this agent
	if !r.isLearningEnabled(&agent) {
		log.V(1).Info("Learning disabled for this agent")
		return ctrl.Result{}, nil
	}

	// Get learning status from ConfigMap
	learningStatus, err := r.getLearningStatus(ctx, &agent)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "Failed to get learning status")
		return ctrl.Result{}, fmt.Errorf("failed to get learning status: %w", err)
	}

	// Check for learning triggers
	learningTriggers, err := r.checkLearningTriggers(ctx, &agent, learningStatus)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "Failed to check learning triggers")
		return ctrl.Result{}, fmt.Errorf("failed to check learning triggers: %w", err)
	}

	// Process any learning triggers
	requeue := false
	for _, trigger := range learningTriggers {
		if err := r.processLearningTrigger(ctx, &agent, trigger, learningStatus); err != nil {
			log.Error(err, "Failed to process learning trigger", "trigger", trigger.EventType, "task", trigger.TaskName)
			r.Recorder.Event(&agent, corev1.EventTypeWarning, "LearningFailed",
				fmt.Sprintf("Failed to process learning for task %s: %v", trigger.TaskName, err))
			// Don't return error - continue processing other triggers
		} else {
			log.Info("Successfully processed learning trigger", "trigger", trigger.EventType, "task", trigger.TaskName)
			requeue = true
		}
	}

	// Update learning status
	if err := r.updateLearningStatus(ctx, &agent, learningStatus); err != nil {
		span.RecordError(err)
		return ctrl.Result{}, fmt.Errorf("failed to update learning status: %w", err)
	}

	span.SetAttributes(attribute.Int("learning.triggers_processed", len(learningTriggers)))
	span.SetStatus(codes.Ok, "Learning reconciliation completed")

	// Requeue to check for new learning opportunities
	requeueAfter := r.LearningInterval
	if requeue {
		requeueAfter = time.Minute // Faster requeue after learning events
	}

	log.V(1).Info("Learning reconciliation completed",
		"triggers", len(learningTriggers),
		"requeue_after", requeueAfter)

	return ctrl.Result{RequeueAfter: requeueAfter}, nil
}

// isLearningEnabled checks if learning is enabled for the given agent
func (r *LearningReconciler) isLearningEnabled(agent *langopv1alpha1.LanguageAgent) bool {
	// Check agent annotations for learning configuration
	if annotations := agent.GetAnnotations(); annotations != nil {
		if disabled, exists := annotations["langop.io/learning-disabled"]; exists && disabled == "true" {
			return false
		}
	}
	return true
}

// getLearningStatus retrieves the current learning status from the agent's ConfigMap
func (r *LearningReconciler) getLearningStatus(ctx context.Context, agent *langopv1alpha1.LanguageAgent) (map[string]*TaskLearningStatus, error) {
	ctx, span := learningTracer.Start(ctx, "learning.get_status")
	defer span.End()

	configMapName := fmt.Sprintf("%s-learning-status", agent.Name)
	var configMap corev1.ConfigMap

	err := r.Get(ctx, types.NamespacedName{
		Name:      configMapName,
		Namespace: agent.Namespace,
	}, &configMap)

	if errors.IsNotFound(err) {
		// Create initial learning status
		return make(map[string]*TaskLearningStatus), nil
	}

	if err != nil {
		span.RecordError(err)
		return nil, fmt.Errorf("failed to get learning status ConfigMap: %w", err)
	}

	// Parse learning status from ConfigMap data
	learningStatus := make(map[string]*TaskLearningStatus)
	for taskName, statusData := range configMap.Data {
		if strings.HasSuffix(taskName, "-status") {
			taskNameClean := strings.TrimSuffix(taskName, "-status")
			status, err := r.parseTaskLearningStatus(statusData)
			if err != nil {
				r.Log.Error(err, "Failed to parse learning status for task", "task", taskNameClean)
				continue
			}
			learningStatus[taskNameClean] = status
		}
	}

	span.SetAttributes(attribute.Int("learning.tasks_tracked", len(learningStatus)))
	return learningStatus, nil
}

// parseTaskLearningStatus parses learning status from JSON string
func (r *LearningReconciler) parseTaskLearningStatus(data string) (*TaskLearningStatus, error) {
	// For now, return a basic status - in production this would parse JSON
	// TODO: Implement JSON parsing once we define the full status structure
	return &TaskLearningStatus{
		TraceCount:        0,
		LearningAttempts:  0,
		CurrentVersion:    1,
		IsSymbolic:        false,
		PatternConfidence: 0.0,
	}, nil
}

// checkLearningTriggers checks for conditions that should trigger learning
func (r *LearningReconciler) checkLearningTriggers(ctx context.Context, agent *langopv1alpha1.LanguageAgent, learningStatus map[string]*TaskLearningStatus) ([]LearningEvent, error) {
	ctx, span := learningTracer.Start(ctx, "learning.check_triggers")
	defer span.End()

	var triggers []LearningEvent

	// Get execution traces for analysis
	traces, err := r.getExecutionTraces(ctx, agent)
	if err != nil {
		span.RecordError(err)
		return nil, fmt.Errorf("failed to get execution traces: %w", err)
	}

	// Group traces by task
	taskTraces := r.groupTracesByTask(traces)

	// Check each task for learning triggers
	for taskName, taskTraceList := range taskTraces {
		status, exists := learningStatus[taskName]
		if !exists {
			status = &TaskLearningStatus{
				TaskName:       taskName,
				CurrentVersion: 1,
			}
			learningStatus[taskName] = status
		}

		// Update trace count and latest trace timestamp
		status.TraceCount = int32(len(taskTraceList))
		if len(taskTraceList) > 0 {
			status.LastTraceTimestamp = taskTraceList[len(taskTraceList)-1].Timestamp
		}

		// Skip if already symbolic (learned)
		if status.IsSymbolic {
			continue
		}

		// Skip if in cooldown period
		if time.Since(status.LastLearningAttempt) < r.LearningInterval {
			continue
		}

		// Check if we have enough traces for pattern analysis
		if status.TraceCount < r.LearningThreshold {
			continue
		}

		// Perform pattern analysis
		analysis, err := r.analyzeTaskPatterns(taskName, taskTraceList)
		if err != nil {
			r.Log.Error(err, "Failed to analyze patterns for task", "task", taskName)
			continue
		}

		// Update learning status with pattern analysis results
		status.PatternConfidence = analysis.Confidence
		status.CommonPattern = analysis.CommonPattern
		status.UniquePatternCount = analysis.UniquePatternCount
		status.ErrorRate = r.calculateErrorRate(taskTraceList)

		// Check if pattern confidence meets threshold
		if analysis.Confidence >= r.PatternConfidenceMin && analysis.IsDeterministic {
			trigger := LearningEvent{
				AgentName:  agent.Name,
				Namespace:  agent.Namespace,
				TaskName:   taskName,
				EventType:  "traces_accumulated",
				TraceCount: status.TraceCount,
				Confidence: analysis.Confidence,
				Timestamp:  time.Now(),
			}
			triggers = append(triggers, trigger)
		}

		// Check for high error rate that might benefit from optimization
		if status.ErrorRate > 0.2 && analysis.Confidence > 0.5 { // 20% error rate
			trigger := LearningEvent{
				AgentName:  agent.Name,
				Namespace:  agent.Namespace,
				TaskName:   taskName,
				EventType:  "error_threshold",
				TraceCount: status.TraceCount,
				ErrorRate:  status.ErrorRate,
				Confidence: analysis.Confidence,
				Timestamp:  time.Now(),
			}
			triggers = append(triggers, trigger)
		}
	}

	span.SetAttributes(attribute.Int("learning.triggers_found", len(triggers)))
	return triggers, nil
}

// processLearningTrigger processes a single learning trigger
func (r *LearningReconciler) processLearningTrigger(ctx context.Context, agent *langopv1alpha1.LanguageAgent, trigger LearningEvent, learningStatus map[string]*TaskLearningStatus) error {
	ctx, span := learningTracer.Start(ctx, "learning.process_trigger")
	defer span.End()

	span.SetAttributes(
		attribute.String("learning.trigger_type", trigger.EventType),
		attribute.String("learning.task_name", trigger.TaskName),
		attribute.Float64("learning.confidence", trigger.Confidence),
	)

	log := r.Log.WithValues(
		"agent", agent.Name,
		"task", trigger.TaskName,
		"trigger", trigger.EventType,
	)

	log.Info("Processing learning trigger")

	// Get or initialize task status
	taskStatus, exists := learningStatus[trigger.TaskName]
	if !exists {
		taskStatus = &TaskLearningStatus{
			TaskName:       trigger.TaskName,
			CurrentVersion: 1,
		}
		learningStatus[trigger.TaskName] = taskStatus
	}

	// Check learning cooldown
	if time.Since(taskStatus.LastLearningAttempt) < r.LearningInterval {
		log.V(1).Info("Learning cooldown active, skipping",
			"last_attempt", taskStatus.LastLearningAttempt,
			"interval", r.LearningInterval)
		return nil
	}

	// Check confidence threshold
	if trigger.Confidence < r.PatternConfidenceMin {
		log.V(1).Info("Pattern confidence below threshold, skipping",
			"confidence", trigger.Confidence,
			"threshold", r.PatternConfidenceMin)
		return nil
	}

	// Record learning attempt
	taskStatus.LastLearningAttempt = time.Now()
	taskStatus.LearningAttempts++

	// Generate learned code
	learnedCode, err := r.generateLearnedCode(ctx, agent, trigger)
	if err != nil {
		span.RecordError(err)
		return fmt.Errorf("failed to generate learned code: %w", err)
	}

	// Create new versioned ConfigMap using ConfigMapManager
	newVersion := taskStatus.CurrentVersion + 1

	// Get previous version for tracking
	var previousVersion *int32
	if taskStatus.CurrentVersion > 0 {
		previousVersion = &taskStatus.CurrentVersion
	}

	configMapOptions := &synthesis.ConfigMapOptions{
		Code:            learnedCode,
		Version:         newVersion,
		SynthesisType:   "learned",
		PreviousVersion: previousVersion,
		LearnedTask:     trigger.TaskName,
		LearningSource:  trigger.EventType, // pattern-detection, error-recovery, etc.
	}

	if _, err := r.ConfigMapManager.CreateVersionedConfigMap(ctx, agent, configMapOptions); err != nil {
		span.RecordError(err)
		return fmt.Errorf("failed to create versioned ConfigMap: %w", err)
	}

	// Apply retention policy if configured
	if r.MaxVersions > 0 {
		retentionPolicy := &synthesis.RetentionPolicy{
			KeepLastN:         r.MaxVersions,
			AlwaysKeepInitial: true, // Always preserve v1 as specified in DSL v1 proposal
		}
		if err := r.ConfigMapManager.ApplyRetentionPolicy(ctx, agent, retentionPolicy); err != nil {
			// Log error but don't fail the learning process
			r.Log.Error(err, "Failed to apply retention policy", "agent", agent.Name)
		}
	}

	// Update deployment
	if err := r.updateDeployment(ctx, agent, trigger.TaskName, newVersion); err != nil {
		span.RecordError(err)
		return fmt.Errorf("failed to update deployment: %w", err)
	}

	// Update task status
	taskStatus.CurrentVersion = newVersion
	taskStatus.IsSymbolic = true
	taskStatus.PatternConfidence = trigger.Confidence

	// Record learning event
	r.recordLearningEvent(agent, trigger, newVersion)

	span.SetAttributes(
		attribute.Int("learning.new_version", int(newVersion)),
		attribute.Bool("learning.success", true),
	)

	span.SetStatus(codes.Ok, "Learning trigger processed successfully")
	return nil
}

// generateLearnedCode generates optimized code for a task based on learning triggers
func (r *LearningReconciler) generateLearnedCode(ctx context.Context, agent *langopv1alpha1.LanguageAgent, trigger LearningEvent) (string, error) {
	ctx, span := learningTracer.Start(ctx, "learning.generate_code")
	defer span.End()

	// Get recent traces for the task
	traces, err := r.getExecutionTraces(ctx, agent)
	if err != nil {
		span.RecordError(err)
		return "", fmt.Errorf("failed to get execution traces: %w", err)
	}

	// Filter traces for the specific task
	var taskTraces []TaskTrace
	for _, trace := range traces {
		if trace.TaskName == trigger.TaskName {
			taskTraces = append(taskTraces, trace)
		}
	}

	// Analyze patterns to generate optimized code
	analysis, err := r.analyzeTaskPatterns(trigger.TaskName, taskTraces)
	if err != nil {
		span.RecordError(err)
		return "", fmt.Errorf("failed to analyze task patterns: %w", err)
	}

	// Use the synthesis service with task_synthesis.tmpl for learned code generation
	synthesisReq := synthesis.AgentSynthesisRequest{
		Instructions: fmt.Sprintf("Optimize task %s based on %d execution traces with pattern: %s", trigger.TaskName, trigger.TraceCount, analysis.CommonPattern),
		AgentName:    agent.Name,
		Namespace:    agent.Namespace,
	}

	response, err := r.Synthesizer.SynthesizeAgent(ctx, synthesisReq)
	if err != nil {
		// Fallback to pattern-based code generation if synthesis fails
		r.Log.Info("Synthesis service failed, using fallback pattern generation",
			"error", err, "task", trigger.TaskName)
		return r.generatePatternBasedCode(trigger.TaskName, analysis), nil
	}

	if response.Error != "" {
		return "", fmt.Errorf("synthesis failed: %s", response.Error)
	}

	span.SetAttributes(
		attribute.Int("learning.generated_code_length", len(response.DSLCode)),
		attribute.Float64("learning.synthesis_duration", response.DurationSeconds),
		attribute.Float64("learning.confidence", trigger.Confidence),
	)

	return response.DSLCode, nil
}

// generatePatternBasedCode generates optimized code based on detected patterns (fallback)
func (r *LearningReconciler) generatePatternBasedCode(taskName string, analysis *PatternAnalysis) string {
	if analysis.RecommendedCode != "" {
		return analysis.RecommendedCode
	}

	// Generate symbolic task based on common patterns
	var codeBuilder strings.Builder

	codeBuilder.WriteString(fmt.Sprintf("task :%s do |inputs|\n", taskName))
	codeBuilder.WriteString(fmt.Sprintf("  # Learned implementation (confidence: %.2f)\n", analysis.Confidence))
	codeBuilder.WriteString(fmt.Sprintf("  # Pattern: %s\n", analysis.CommonPattern))
	codeBuilder.WriteString("  \n")

	// Generate code based on detected pattern type
	switch {
	case strings.Contains(analysis.CommonPattern, "deterministic_tool_sequence"):
		codeBuilder.WriteString("  # Optimized tool sequence based on execution patterns\n")
		codeBuilder.WriteString("  result = execute_optimized_sequence(inputs)\n")
	case strings.Contains(analysis.CommonPattern, "simple_transformation"):
		codeBuilder.WriteString("  # Direct data transformation without tool calls\n")
		codeBuilder.WriteString("  result = transform_data(inputs)\n")
	case strings.Contains(analysis.CommonPattern, "conditional_logic"):
		codeBuilder.WriteString("  # Conditional logic based on input patterns\n")
		codeBuilder.WriteString("  if condition_check(inputs)\n")
		codeBuilder.WriteString("    result = primary_path(inputs)\n")
		codeBuilder.WriteString("  else\n")
		codeBuilder.WriteString("    result = alternative_path(inputs)\n")
		codeBuilder.WriteString("  end\n")
	default:
		codeBuilder.WriteString("  # Generic optimization based on observed patterns\n")
		codeBuilder.WriteString("  result = execute_learned_pattern(inputs)\n")
	}

	codeBuilder.WriteString("  \n")
	codeBuilder.WriteString("  { result: result }\n")
	codeBuilder.WriteString("end")

	return codeBuilder.String()
}

// Note: createVersionedConfigMap method removed - now using ConfigMapManager

// updateDeployment updates the deployment to use the new versioned ConfigMap
func (r *LearningReconciler) updateDeployment(ctx context.Context, agent *langopv1alpha1.LanguageAgent, taskName string, version int32) error {
	ctx, span := learningTracer.Start(ctx, "learning.update_deployment")
	defer span.End()

	span.SetAttributes(
		attribute.String("learning.task_name", taskName),
		attribute.Int("learning.new_version", int(version)),
	)

	log := r.Log.WithValues("agent", agent.Name, "task", taskName, "version", version)
	log.Info("Updating deployment for learned task")

	// Find the agent's deployment
	deployment, err := r.findAgentDeployment(ctx, agent)
	if err != nil {
		span.RecordError(err)
		return fmt.Errorf("failed to find agent deployment: %w", err)
	}

	if deployment == nil {
		// No deployment found - this might be a CronJob or other workload type
		log.V(1).Info("No deployment found, checking for other workload types")
		return r.updateAlternativeWorkload(ctx, agent, taskName, version)
	}

	// Store original ConfigMap reference for rollback
	originalConfigMap := r.extractConfigMapReference(deployment)

	// Update the deployment with the new ConfigMap version
	newConfigMapName := fmt.Sprintf("%s-v%d", agent.Name, version)

	// Create deployment patch
	if err := r.patchDeploymentConfigMap(ctx, deployment, newConfigMapName); err != nil {
		span.RecordError(err)
		return fmt.Errorf("failed to patch deployment: %w", err)
	}

	// Wait for rollout to complete
	if err := r.waitForDeploymentRollout(ctx, deployment, time.Minute*5); err != nil {
		span.RecordError(err)
		log.Error(err, "Deployment rollout failed, attempting rollback")

		// Attempt rollback
		if rollbackErr := r.rollbackDeployment(ctx, deployment, originalConfigMap); rollbackErr != nil {
			log.Error(rollbackErr, "Failed to rollback deployment")
			return fmt.Errorf("deployment update failed and rollback failed: %w", err)
		}

		r.Recorder.Event(agent, corev1.EventTypeWarning, "LearningRollback",
			fmt.Sprintf("Rolled back deployment after failed learning update for task %s", taskName))

		return fmt.Errorf("deployment rollout failed, rolled back: %w", err)
	}

	// Verify the deployment is healthy
	if err := r.verifyDeploymentHealth(ctx, deployment); err != nil {
		span.RecordError(err)
		log.Error(err, "Deployment health check failed after learning update")

		// Don't rollback automatically here - let monitoring/alerting handle it
		r.Recorder.Event(agent, corev1.EventTypeWarning, "LearningHealthCheck",
			fmt.Sprintf("Deployment health check failed after learning update for task %s", taskName))

		return fmt.Errorf("deployment health check failed: %w", err)
	}

	span.SetAttributes(
		attribute.String("learning.original_configmap", originalConfigMap),
		attribute.String("learning.new_configmap", newConfigMapName),
		attribute.Bool("learning.deployment_updated", true),
	)

	log.Info("Successfully updated deployment for learned task")
	r.Recorder.Event(agent, corev1.EventTypeNormal, "LearningDeploymentUpdated",
		fmt.Sprintf("Updated deployment to use learned task %s (v%d)", taskName, version))

	return nil
}

// findAgentDeployment finds the deployment associated with the agent
func (r *LearningReconciler) findAgentDeployment(ctx context.Context, agent *langopv1alpha1.LanguageAgent) (*appsv1.Deployment, error) {
	ctx, span := learningTracer.Start(ctx, "learning.find_deployment")
	defer span.End()

	// List deployments with the agent label
	deploymentList := &appsv1.DeploymentList{}
	labelSelector := client.MatchingLabels{
		"langop.io/agent": agent.Name,
	}

	err := r.List(ctx, deploymentList,
		client.InNamespace(agent.Namespace),
		labelSelector)
	if err != nil {
		span.RecordError(err)
		return nil, fmt.Errorf("failed to list deployments: %w", err)
	}

	if len(deploymentList.Items) == 0 {
		return nil, nil // No deployment found
	}

	if len(deploymentList.Items) > 1 {
		r.Log.Info("Multiple deployments found for agent, using first one",
			"agent", agent.Name, "count", len(deploymentList.Items))
	}

	deployment := &deploymentList.Items[0]
	span.SetAttributes(
		attribute.String("learning.deployment_name", deployment.Name),
		attribute.String("learning.deployment_generation", fmt.Sprintf("%d", deployment.Generation)),
	)

	return deployment, nil
}

// updateAlternativeWorkload handles non-deployment workloads (CronJob, Job, etc.)
func (r *LearningReconciler) updateAlternativeWorkload(ctx context.Context, agent *langopv1alpha1.LanguageAgent, taskName string, version int32) error {
	// TODO: Implement updates for CronJob and other workload types
	// For now, just log that we would update alternative workloads
	r.Log.Info("Alternative workload update not yet implemented",
		"agent", agent.Name, "task", taskName, "version", version)

	return nil
}

// extractConfigMapReference extracts the current ConfigMap reference from deployment
func (r *LearningReconciler) extractConfigMapReference(deployment *appsv1.Deployment) string {
	// Look through volumes for ConfigMap references
	for _, volume := range deployment.Spec.Template.Spec.Volumes {
		if volume.ConfigMap != nil && strings.Contains(volume.ConfigMap.Name, deployment.Labels["langop.io/agent"]) {
			return volume.ConfigMap.Name
		}
	}

	// Look through environment variables for ConfigMap references
	for _, container := range deployment.Spec.Template.Spec.Containers {
		for _, envFrom := range container.EnvFrom {
			if envFrom.ConfigMapRef != nil && strings.Contains(envFrom.ConfigMapRef.Name, deployment.Labels["langop.io/agent"]) {
				return envFrom.ConfigMapRef.Name
			}
		}
	}

	// Default fallback
	return fmt.Sprintf("%s-v1", deployment.Labels["langop.io/agent"])
}

// patchDeploymentConfigMap updates the deployment to use a new ConfigMap
func (r *LearningReconciler) patchDeploymentConfigMap(ctx context.Context, deployment *appsv1.Deployment, newConfigMapName string) error {
	ctx, span := learningTracer.Start(ctx, "learning.patch_deployment")
	defer span.End()

	// Create a copy to modify
	updatedDeployment := deployment.DeepCopy()

	// Update ConfigMap references in volumes
	for i, volume := range updatedDeployment.Spec.Template.Spec.Volumes {
		if volume.ConfigMap != nil && strings.Contains(volume.ConfigMap.Name, deployment.Labels["langop.io/agent"]) {
			updatedDeployment.Spec.Template.Spec.Volumes[i].ConfigMap.Name = newConfigMapName
		}
	}

	// Update ConfigMap references in environment
	for containerIdx, container := range updatedDeployment.Spec.Template.Spec.Containers {
		for envIdx, envFrom := range container.EnvFrom {
			if envFrom.ConfigMapRef != nil && strings.Contains(envFrom.ConfigMapRef.Name, deployment.Labels["langop.io/agent"]) {
				updatedDeployment.Spec.Template.Spec.Containers[containerIdx].EnvFrom[envIdx].ConfigMapRef.Name = newConfigMapName
			}
		}
	}

	// Add annotation to trigger rolling update
	if updatedDeployment.Spec.Template.Annotations == nil {
		updatedDeployment.Spec.Template.Annotations = make(map[string]string)
	}
	updatedDeployment.Spec.Template.Annotations["langop.io/learning-update"] = time.Now().Format(time.RFC3339)
	updatedDeployment.Spec.Template.Annotations["langop.io/learned-configmap"] = newConfigMapName

	// Update the deployment
	if err := r.Update(ctx, updatedDeployment); err != nil {
		span.RecordError(err)
		return fmt.Errorf("failed to update deployment: %w", err)
	}

	span.SetAttributes(
		attribute.String("learning.new_configmap", newConfigMapName),
		attribute.String("learning.update_timestamp", time.Now().Format(time.RFC3339)),
	)

	return nil
}

// waitForDeploymentRollout waits for the deployment rollout to complete
func (r *LearningReconciler) waitForDeploymentRollout(ctx context.Context, deployment *appsv1.Deployment, timeout time.Duration) error {
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	ctx, span := learningTracer.Start(ctx, "learning.wait_rollout")
	defer span.End()

	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			span.SetStatus(codes.Error, "Rollout timeout")
			return fmt.Errorf("rollout timeout after %v", timeout)
		case <-ticker.C:
			// Get fresh deployment status
			var currentDeployment appsv1.Deployment
			if err := r.Get(ctx, types.NamespacedName{
				Name:      deployment.Name,
				Namespace: deployment.Namespace,
			}, &currentDeployment); err != nil {
				span.RecordError(err)
				return fmt.Errorf("failed to get deployment status: %w", err)
			}

			// Check if rollout is complete
			if currentDeployment.Status.UpdatedReplicas == currentDeployment.Status.Replicas &&
				currentDeployment.Status.ReadyReplicas == currentDeployment.Status.Replicas &&
				currentDeployment.Status.AvailableReplicas == currentDeployment.Status.Replicas {
				span.SetAttributes(
					attribute.Int("learning.final_replicas", int(currentDeployment.Status.Replicas)),
					attribute.Int("learning.ready_replicas", int(currentDeployment.Status.ReadyReplicas)),
				)
				return nil
			}

			r.Log.V(1).Info("Waiting for deployment rollout",
				"deployment", deployment.Name,
				"replicas", currentDeployment.Status.Replicas,
				"updated", currentDeployment.Status.UpdatedReplicas,
				"ready", currentDeployment.Status.ReadyReplicas)
		}
	}
}

// rollbackDeployment rolls back the deployment to use the previous ConfigMap
func (r *LearningReconciler) rollbackDeployment(ctx context.Context, deployment *appsv1.Deployment, originalConfigMap string) error {
	ctx, span := learningTracer.Start(ctx, "learning.rollback_deployment")
	defer span.End()

	r.Log.Info("Rolling back deployment",
		"deployment", deployment.Name,
		"original_configmap", originalConfigMap)

	// Update deployment to use original ConfigMap
	return r.patchDeploymentConfigMap(ctx, deployment, originalConfigMap)
}

// verifyDeploymentHealth performs basic health checks on the deployment
func (r *LearningReconciler) verifyDeploymentHealth(ctx context.Context, deployment *appsv1.Deployment) error {
	ctx, span := learningTracer.Start(ctx, "learning.verify_health")
	defer span.End()

	// Get fresh deployment status
	var currentDeployment appsv1.Deployment
	if err := r.Get(ctx, types.NamespacedName{
		Name:      deployment.Name,
		Namespace: deployment.Namespace,
	}, &currentDeployment); err != nil {
		span.RecordError(err)
		return fmt.Errorf("failed to get deployment for health check: %w", err)
	}

	// Check basic health indicators
	if currentDeployment.Status.ReadyReplicas == 0 {
		return fmt.Errorf("no ready replicas")
	}

	if currentDeployment.Status.UnavailableReplicas > 0 {
		return fmt.Errorf("%d unavailable replicas", currentDeployment.Status.UnavailableReplicas)
	}

	// TODO: Add more sophisticated health checks:
	// - Pod readiness probes
	// - Custom health endpoints
	// - Performance metrics comparison
	// - Error rate monitoring

	span.SetAttributes(
		attribute.Int("learning.ready_replicas", int(currentDeployment.Status.ReadyReplicas)),
		attribute.Int("learning.unavailable_replicas", int(currentDeployment.Status.UnavailableReplicas)),
		attribute.Bool("learning.health_check_passed", true),
	)

	return nil
}

// updateLearningStatus saves the current learning status to the ConfigMap
func (r *LearningReconciler) updateLearningStatus(ctx context.Context, agent *langopv1alpha1.LanguageAgent, learningStatus map[string]*TaskLearningStatus) error {
	ctx, span := learningTracer.Start(ctx, "learning.update_status")
	defer span.End()

	configMapName := fmt.Sprintf("%s-learning-status", agent.Name)

	configMap := &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      configMapName,
			Namespace: agent.Namespace,
			Labels: map[string]string{
				"langop.io/agent":     agent.Name,
				"langop.io/component": "learning-status",
			},
		},
		Data: make(map[string]string),
	}

	// Set owner reference
	if err := controllerutil.SetControllerReference(agent, configMap, r.Scheme); err != nil {
		span.RecordError(err)
		return fmt.Errorf("failed to set controller reference: %w", err)
	}

	// Serialize learning status
	for taskName, status := range learningStatus {
		statusData := r.serializeTaskLearningStatus(status)
		configMap.Data[fmt.Sprintf("%s-status", taskName)] = statusData
	}

	// Create or update ConfigMap
	var existing corev1.ConfigMap
	err := r.Get(ctx, types.NamespacedName{
		Name:      configMapName,
		Namespace: agent.Namespace,
	}, &existing)

	if errors.IsNotFound(err) {
		if err := r.Create(ctx, configMap); err != nil {
			span.RecordError(err)
			return fmt.Errorf("failed to create learning status ConfigMap: %w", err)
		}
	} else if err != nil {
		span.RecordError(err)
		return fmt.Errorf("failed to get learning status ConfigMap: %w", err)
	} else {
		existing.Data = configMap.Data
		if err := r.Update(ctx, &existing); err != nil {
			span.RecordError(err)
			return fmt.Errorf("failed to update learning status ConfigMap: %w", err)
		}
	}

	span.SetAttributes(attribute.Int("learning.status_entries", len(learningStatus)))
	return nil
}

// serializeTaskLearningStatus serializes task learning status to string
func (r *LearningReconciler) serializeTaskLearningStatus(status *TaskLearningStatus) string {
	// For now, return a simple string representation
	// TODO: Implement JSON serialization
	return fmt.Sprintf("traces:%d,attempts:%d,version:%d,symbolic:%t,confidence:%.2f",
		status.TraceCount,
		status.LearningAttempts,
		status.CurrentVersion,
		status.IsSymbolic,
		status.PatternConfidence,
	)
}

// recordLearningEvent records a learning event for auditing and monitoring
func (r *LearningReconciler) recordLearningEvent(agent *langopv1alpha1.LanguageAgent, trigger LearningEvent, newVersion int32) {
	message := fmt.Sprintf("Learned optimization for task %s (v%d) with confidence %.2f from %s",
		trigger.TaskName, newVersion, trigger.Confidence, trigger.EventType)

	r.Recorder.Event(agent, corev1.EventTypeNormal, "LearningSucceeded", message)

	r.Log.Info("Recorded learning event",
		"agent", agent.Name,
		"task", trigger.TaskName,
		"version", newVersion,
		"confidence", trigger.Confidence,
		"trigger", trigger.EventType)
}

// SetupWithManager sets up the controller with the Manager
func (r *LearningReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&langopv1alpha1.LanguageAgent{}).
		Owns(&corev1.ConfigMap{}).
		Named("learning").
		Complete(r)
}

// getExecutionTraces retrieves execution traces for pattern analysis
func (r *LearningReconciler) getExecutionTraces(ctx context.Context, agent *langopv1alpha1.LanguageAgent) ([]TaskTrace, error) {
	ctx, span := learningTracer.Start(ctx, "learning.get_traces")
	defer span.End()

	// TODO: Implement actual trace retrieval from monitoring/observability system
	// This would typically query:
	// - OpenTelemetry traces
	// - Application logs
	// - Metrics storage
	// - Custom trace storage

	// For now, return empty traces - this will be implemented when we integrate
	// with the actual agent execution environment
	var traces []TaskTrace

	span.SetAttributes(attribute.Int("learning.traces_retrieved", len(traces)))
	return traces, nil
}

// groupTracesByTask groups execution traces by task name
func (r *LearningReconciler) groupTracesByTask(traces []TaskTrace) map[string][]TaskTrace {
	taskGroups := make(map[string][]TaskTrace)

	for _, trace := range traces {
		taskGroups[trace.TaskName] = append(taskGroups[trace.TaskName], trace)
	}

	return taskGroups
}

// analyzeTaskPatterns performs pattern analysis on task execution traces
func (r *LearningReconciler) analyzeTaskPatterns(taskName string, traces []TaskTrace) (*PatternAnalysis, error) {
	if len(traces) == 0 {
		return &PatternAnalysis{
			TaskName:           taskName,
			IsDeterministic:    false,
			Confidence:         0.0,
			CommonPattern:      "insufficient_data",
			ConsistencyScore:   0.0,
			UniquePatternCount: 0,
			Explanation:        "No execution traces available for analysis",
		}, nil
	}

	// Analyze tool call patterns
	toolCallPatterns := r.analyzeToolCallPatterns(traces)

	// Analyze input/output patterns
	ioConsistency := r.analyzeInputOutputConsistency(traces)

	// Determine if the task is deterministic based on patterns
	isDeterministic := r.isDeterministicTask(traces, toolCallPatterns)

	// Calculate confidence based on pattern consistency
	confidence := r.calculatePatternConfidence(traces, toolCallPatterns, ioConsistency)

	// Identify the most common execution pattern
	commonPattern := r.identifyCommonPattern(traces, toolCallPatterns)

	// Count unique patterns
	uniquePatterns := r.countUniquePatterns(traces)

	analysis := &PatternAnalysis{
		TaskName:           taskName,
		IsDeterministic:    isDeterministic,
		Confidence:         confidence,
		CommonPattern:      commonPattern,
		ConsistencyScore:   ioConsistency,
		UniquePatternCount: int32(uniquePatterns),
		Explanation:        r.generatePatternExplanation(taskName, traces, confidence, isDeterministic),
	}

	// Generate recommended code if confidence is high enough
	if confidence > 0.7 && isDeterministic {
		analysis.RecommendedCode = r.generatePatternBasedCode(taskName, analysis)
	}

	return analysis, nil
}

// analyzeToolCallPatterns analyzes patterns in tool calls across traces
func (r *LearningReconciler) analyzeToolCallPatterns(traces []TaskTrace) map[string]int {
	patterns := make(map[string]int)

	for _, trace := range traces {
		// Create a pattern signature from the sequence of tool calls
		var patternParts []string
		for _, toolCall := range trace.ToolCalls {
			patternParts = append(patternParts, fmt.Sprintf("%s.%s", toolCall.ToolName, toolCall.Method))
		}
		pattern := strings.Join(patternParts, "->")
		patterns[pattern]++
	}

	return patterns
}

// analyzeInputOutputConsistency measures how consistent inputs and outputs are
func (r *LearningReconciler) analyzeInputOutputConsistency(traces []TaskTrace) float64 {
	if len(traces) < 2 {
		return 0.0
	}

	// TODO: Implement sophisticated I/O consistency analysis
	// This would analyze:
	// - Input parameter stability
	// - Output format consistency
	// - Value ranges and types
	// - Transformation patterns

	// For now, return a placeholder based on success rate
	successCount := 0
	for _, trace := range traces {
		if trace.Success {
			successCount++
		}
	}

	return float64(successCount) / float64(len(traces))
}

// isDeterministicTask determines if a task exhibits deterministic behavior
func (r *LearningReconciler) isDeterministicTask(traces []TaskTrace, toolCallPatterns map[string]int) bool {
	if len(traces) < 3 {
		return false // Need sufficient samples
	}

	// Check if there's a dominant pattern (>70% of executions)
	totalTraces := len(traces)
	for _, count := range toolCallPatterns {
		if float64(count)/float64(totalTraces) > 0.7 {
			return true
		}
	}

	// Check consistency of successful executions
	successfulTraces := 0
	for _, trace := range traces {
		if trace.Success {
			successfulTraces++
		}
	}

	// High success rate with low pattern variation indicates determinism
	successRate := float64(successfulTraces) / float64(totalTraces)
	patternVariation := len(toolCallPatterns)

	return successRate > 0.8 && patternVariation <= 3
}

// calculatePatternConfidence calculates confidence in the detected patterns
func (r *LearningReconciler) calculatePatternConfidence(traces []TaskTrace, toolCallPatterns map[string]int, ioConsistency float64) float64 {
	if len(traces) == 0 {
		return 0.0
	}

	// Base confidence on pattern frequency
	totalTraces := len(traces)
	maxPatternCount := 0
	for _, count := range toolCallPatterns {
		if count > maxPatternCount {
			maxPatternCount = count
		}
	}

	patternConfidence := float64(maxPatternCount) / float64(totalTraces)

	// Weight by I/O consistency
	combinedConfidence := (patternConfidence + ioConsistency) / 2.0

	// Boost confidence with more traces (up to a point)
	traceBonus := 1.0
	if totalTraces >= 10 {
		traceBonus = 1.1
	}
	if totalTraces >= 20 {
		traceBonus = 1.2
	}

	finalConfidence := combinedConfidence * traceBonus

	// Cap at 0.95 to leave room for uncertainty
	if finalConfidence > 0.95 {
		finalConfidence = 0.95
	}

	return finalConfidence
}

// identifyCommonPattern identifies the most common execution pattern
func (r *LearningReconciler) identifyCommonPattern(traces []TaskTrace, toolCallPatterns map[string]int) string {
	if len(toolCallPatterns) == 0 {
		return "no_pattern"
	}

	// Find the most frequent pattern
	maxCount := 0
	var mostCommonPattern string
	for pattern, count := range toolCallPatterns {
		if count > maxCount {
			maxCount = count
			mostCommonPattern = pattern
		}
	}

	// Classify pattern types
	if strings.Contains(mostCommonPattern, "->") {
		if strings.Count(mostCommonPattern, "->") <= 2 {
			return "simple_tool_sequence"
		} else {
			return "complex_tool_sequence"
		}
	}

	// Analyze for deterministic patterns
	if len(toolCallPatterns) == 1 {
		return "deterministic_tool_sequence"
	}

	// Check for transformation patterns
	transformationKeywords := []string{"transform", "convert", "process", "format"}
	for _, keyword := range transformationKeywords {
		if strings.Contains(strings.ToLower(mostCommonPattern), keyword) {
			return "simple_transformation"
		}
	}

	// Check for conditional patterns (multiple but limited patterns)
	if len(toolCallPatterns) > 1 && len(toolCallPatterns) <= 3 {
		return "conditional_logic"
	}

	return "variable_pattern"
}

// countUniquePatterns counts the number of unique execution patterns
func (r *LearningReconciler) countUniquePatterns(traces []TaskTrace) int {
	patterns := make(map[string]bool)

	for _, trace := range traces {
		// Create a signature for each trace
		var signature []string
		signature = append(signature, fmt.Sprintf("success:%t", trace.Success))
		signature = append(signature, fmt.Sprintf("tools:%d", len(trace.ToolCalls)))

		for _, toolCall := range trace.ToolCalls {
			signature = append(signature, fmt.Sprintf("%s.%s", toolCall.ToolName, toolCall.Method))
		}

		patternKey := strings.Join(signature, "|")
		patterns[patternKey] = true
	}

	return len(patterns)
}

// calculateErrorRate calculates the error rate from execution traces
func (r *LearningReconciler) calculateErrorRate(traces []TaskTrace) float64 {
	if len(traces) == 0 {
		return 0.0
	}

	errorCount := 0
	for _, trace := range traces {
		if !trace.Success {
			errorCount++
		}
	}

	return float64(errorCount) / float64(len(traces))
}

// generatePatternExplanation generates a human-readable explanation of the detected patterns
func (r *LearningReconciler) generatePatternExplanation(taskName string, traces []TaskTrace, confidence float64, isDeterministic bool) string {
	var explanation strings.Builder

	explanation.WriteString(fmt.Sprintf("Analysis of %d execution traces for task '%s': ", len(traces), taskName))

	if confidence > 0.8 {
		explanation.WriteString("Strong patterns detected. ")
	} else if confidence > 0.6 {
		explanation.WriteString("Moderate patterns detected. ")
	} else {
		explanation.WriteString("Weak or inconsistent patterns. ")
	}

	if isDeterministic {
		explanation.WriteString("Task behavior is highly deterministic and suitable for symbolic optimization. ")
	} else {
		explanation.WriteString("Task behavior shows variation that may limit optimization potential. ")
	}

	errorRate := r.calculateErrorRate(traces)
	if errorRate > 0.1 {
		explanation.WriteString(fmt.Sprintf("Error rate of %.1f%% suggests optimization could improve reliability. ", errorRate*100))
	} else {
		explanation.WriteString("Low error rate indicates stable execution. ")
	}

	return explanation.String()
}
