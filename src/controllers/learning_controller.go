package controllers

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"time"

	"github.com/go-logr/logr"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/tools/record"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/builder"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	langopv1alpha1 "github.com/language-operator/language-operator/api/v1alpha1"
	"github.com/language-operator/language-operator/pkg/learning"
	"github.com/language-operator/language-operator/pkg/reconciler"
	"github.com/language-operator/language-operator/pkg/synthesis"
	"github.com/language-operator/language-operator/pkg/telemetry"
)

// OptimizedTask contains both the optimized code and its schema information
type OptimizedTask struct {
	Code    string // The optimized task body code from LLM
	Inputs  string // JSON schema for inputs
	Outputs string // JSON schema for outputs
}

// LearningReconciler reconciles learning events and triggers re-synthesis
type LearningReconciler struct {
	client.Client
	Scheme                *runtime.Scheme
	Log                   logr.Logger
	Recorder              record.EventRecorder
	ConfigMapManager      *synthesis.ConfigMapManager                        // For versioned ConfigMap management
	MetricsCollector      *learning.MetricsCollector                         // For learning metrics collection
	EventProcessor        *learning.LearningEventProcessor                   // For processing learning events with metrics
	TelemetryAdapter      telemetry.TelemetryAdapter                         // For querying historical execution data
	SuccessRateAggregator map[string]*learning.LearningSuccessRateAggregator // Per-agent success rate tracking
	LearningEnabled       bool
	LearningThreshold     int32         // Number of execution traces before triggering learning
	LearningInterval      time.Duration // Minimum interval between learning attempts
	MaxVersions           int32         // Maximum number of ConfigMap versions to keep
	PatternConfidenceMin  float64       // Minimum confidence threshold for pattern detection

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


	// Execution metrics for learning status command
	TotalExecutions      int32     `json:"totalExecutions"`
	SuccessfulExecutions int32     `json:"successfulExecutions"`
	FailedExecutions     int32     `json:"failedExecutions"`
	LastExecutionTime    time.Time `json:"lastExecutionTime"`
	SuccessRate          float64   `json:"successRate"`
	LearningStatus       string    `json:"learningStatus"` // "learning", "ready_for_symbolic", "symbolic"
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

// TaskExecutionStatus represents execution status for a single task in the summary
type TaskExecutionStatus struct {
	Executions  int32   `json:"executions"`
	SuccessRate float64 `json:"successRate"`
	Confidence  float64 `json:"confidence"`
	Status      string  `json:"status"` // "learning", "ready_for_symbolic", "symbolic"
}

// AgentExecutionSummary provides a summary of agent execution metrics for the Ruby gem
type AgentExecutionSummary struct {
	TotalExecutions   int32                          `json:"totalExecutions"`
	LastExecution     time.Time                      `json:"lastExecution"`
	SuccessRate       float64                        `json:"successRate"`
	LearningThreshold int32                          `json:"learningThreshold"`
	Tasks             map[string]TaskExecutionStatus `json:"tasks"`
	Created           time.Time                      `json:"created"`
	Updated           time.Time                      `json:"updated"`
}

// learningTracer is used by helper methods for detailed tracing
var learningTracer = otel.Tracer("language-operator/learning")

//+kubebuilder:rbac:groups=langop.io,resources=languageagents,verbs=get;list;watch;update;patch
//+kubebuilder:rbac:groups="",resources=configmaps,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups="",resources=events,verbs=create;get;list;patch;watch;update
//+kubebuilder:rbac:groups=apps,resources=deployments,verbs=get;list;watch;update;patch
//+kubebuilder:rbac:groups=apps,resources=daemonsets,verbs=get;list;watch;update;patch

// Reconcile handles learning events and triggers re-synthesis when appropriate
func (r *LearningReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	// Use the reconciler helper for common setup
	helper := &reconciler.ReconcileHelper[*langopv1alpha1.LanguageAgent]{
		Client:       r.Client,
		TracerName:   "language-operator/learning-controller",
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
	log := r.Log.WithValues("agent", req.NamespacedName)

	if !r.LearningEnabled {
		log.V(1).Info("Learning disabled, skipping reconciliation")
		return ctrl.Result{}, nil
	}

	// Check if learning is enabled for this agent
	if !r.isLearningEnabled(agent) {
		log.V(1).Info("Learning disabled for this agent")
		return ctrl.Result{}, nil
	}

	log.Info("Processing simplified learning check",
		"agent", req.NamespacedName,
		"runsPendingLearning", agent.Status.RunsPendingLearning,
		"threshold", r.LearningThreshold)

	// Simplified learning logic: check if runsPendingLearning reaches threshold
	if agent.Status.RunsPendingLearning >= r.LearningThreshold {
		log.Info("Learning threshold reached, triggering optimization",
			"runs", agent.Status.RunsPendingLearning,
			"threshold", r.LearningThreshold)

		// Trigger optimization
		if err := r.triggerOptimization(ctx, agent); err != nil {
			log.Error(err, "Failed to trigger optimization")
			reconcileErr = fmt.Errorf("failed to trigger optimization: %w", err)
			return ctrl.Result{}, reconcileErr
		}

		// Reset counter after successful optimization
		agent.Status.RunsPendingLearning = 0
		if err := r.Status().Update(ctx, agent); err != nil {
			log.Error(err, "Failed to reset runsPendingLearning counter")
			reconcileErr = fmt.Errorf("failed to reset counter: %w", err)
			return ctrl.Result{}, reconcileErr
		}

		log.Info("Optimization completed and counter reset")
	}

	span.SetStatus(codes.Ok, "Learning reconciliation completed")
	return ctrl.Result{}, nil
}

// triggerOptimization triggers agent code optimization based on telemetry data
func (r *LearningReconciler) triggerOptimization(ctx context.Context, agent *langopv1alpha1.LanguageAgent) error {
	ctx, span := learningTracer.Start(ctx, "learning.trigger_optimization")
	defer span.End()

	log := r.Log.WithValues("agent", agent.Name, "namespace", agent.Namespace)
	log.Info("Triggering agent optimization based on accumulated runs")

	// Create synthesizer from agent's model for optimization
	synthesizer, err := r.createSynthesizerForAgent(ctx, agent)
	if err != nil {
		span.RecordError(err)
		log.Error(err, "Failed to create synthesizer for optimization")
		return fmt.Errorf("failed to create synthesizer: %w", err)
	}

	// Get tasks to optimize from agent's telemetry data
	tasksToOptimize, err := r.identifyTasksForOptimization(ctx, agent)
	if err != nil {
		span.RecordError(err)
		log.Error(err, "Failed to identify tasks for optimization")
		return fmt.Errorf("failed to identify tasks: %w", err)
	}

	if len(tasksToOptimize) == 0 {
		log.Info("No tasks identified for optimization", "agent", agent.Name)
		// Reset the counter since we processed the optimization request
		agent.Status.RunsPendingLearning = 0
		if err := r.Status().Update(ctx, agent); err != nil {
			log.Error(err, "Failed to reset learning counter")
		}
		return nil
	}

	log.Info("Identified tasks for optimization", "agent", agent.Name, "taskCount", len(tasksToOptimize))

	// Optimize each task individually and collect results
	optimizedTasks := make(map[string]OptimizedTask) // taskName -> optimized task with schema
	for _, taskReq := range tasksToOptimize {
		log.Info("Optimizing task", "task", taskReq.TaskName, "agent", agent.Name)

		taskResponse, err := synthesizer.SynthesizeTask(ctx, taskReq)
		if err != nil {
			log.Error(err, "Failed to synthesize task", "task", taskReq.TaskName)
			continue // Continue with other tasks
		}

		if taskResponse.Error != "" {
			log.Error(fmt.Errorf(taskResponse.Error), "Task synthesis returned error", "task", taskReq.TaskName)
			continue
		}

		// Check if task was determined to be optimizable
		if !taskResponse.IsDeterministic || taskResponse.Code == nil {
			log.Info("Task not suitable for optimization",
				"task", taskReq.TaskName,
				"isDeterministic", taskResponse.IsDeterministic,
				"confidence", taskResponse.Confidence,
				"explanation", taskResponse.Explanation)
			continue
		}

		log.Info("Task optimization successful",
			"task", taskReq.TaskName,
			"confidence", taskResponse.Confidence,
			"explanation", taskResponse.Explanation,
			"codeLength", len(*taskResponse.Code))

		// Store optimized task with schema information
		optimizedTasks[taskReq.TaskName] = OptimizedTask{
			Code:    *taskResponse.Code,
			Inputs:  taskReq.Inputs,
			Outputs: taskReq.Outputs,
		}
	}

	if len(optimizedTasks) == 0 {
		log.Info("No tasks were successfully optimized", "agent", agent.Name)
	} else {
		// Create LanguageAgentVersion resource with optimized code
		if err := r.createAgentVersion(ctx, agent, optimizedTasks, synthesizer); err != nil {
			log.Error(err, "Failed to create LanguageAgentVersion")
			// Don't return error - learning was successful, version creation failed
		} else {
			log.Info("Successfully created LanguageAgentVersion with optimized tasks",
				"agent", agent.Name,
				"optimizedTaskCount", len(optimizedTasks))
		}
	}

	// Reset the counter after processing optimization
	agent.Status.RunsPendingLearning = 0
	if err := r.Status().Update(ctx, agent); err != nil {
		log.Error(err, "Failed to reset learning counter after optimization")
	}

	// Record successful optimization event
	r.Recorder.Event(agent, corev1.EventTypeNormal, "OptimizationTriggered",
		fmt.Sprintf("Agent code optimized after %d runs", agent.Status.RunsPendingLearning))

	log.Info("Successfully triggered agent optimization")
	span.SetStatus(codes.Ok, "Optimization completed")
	return nil
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

	if apierrors.IsNotFound(err) {
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
	// Try JSON parsing first (new format)
	var status TaskLearningStatus
	if err := json.Unmarshal([]byte(data), &status); err == nil {
		// Validate that version is non-negative
		if status.CurrentVersion < 0 {
			r.Log.Error(fmt.Errorf("invalid version in task status: %d", status.CurrentVersion),
				"Invalid version number in task status", "task", status.TaskName, "version", status.CurrentVersion)
			status.CurrentVersion = 0 // Reset to safe default
		}
		return &status, nil
	}

	// Fall back to legacy string parsing for backward compatibility
	r.Log.V(1).Info("JSON parsing failed, attempting legacy string parsing", "data", data)
	return r.parseTaskLearningStatusLegacy(data)
}

// parseTaskLearningStatusLegacy parses learning status from legacy string format
func (r *LearningReconciler) parseTaskLearningStatusLegacy(data string) (*TaskLearningStatus, error) {
	// Parse the simple string format: "task:task_name,traces:5,attempts:2,version:1,symbolic:false,confidence:0.85"
	status := &TaskLearningStatus{
		TraceCount:        0,
		LearningAttempts:  0,
		CurrentVersion:    1,
		IsSymbolic:        false,
		PatternConfidence: 0.0,
	}

	// Simple parsing for key:value pairs
	parts := strings.Split(data, ",")
	for _, part := range parts {
		keyValue := strings.Split(part, ":")
		if len(keyValue) != 2 {
			continue
		}

		key := strings.TrimSpace(keyValue[0])
		value := strings.TrimSpace(keyValue[1])

		switch key {
		case "task":
			status.TaskName = value
		case "traces":
			if v, err := fmt.Sscanf(value, "%d", &status.TraceCount); err == nil && v == 1 {
				// parsed successfully
			}
		case "attempts":
			if v, err := fmt.Sscanf(value, "%d", &status.LearningAttempts); err == nil && v == 1 {
				// parsed successfully
			}
		case "version":
			if v, err := fmt.Sscanf(value, "%d", &status.CurrentVersion); err == nil && v == 1 {
				// Validate that version is non-negative
				if status.CurrentVersion < 0 {
					r.Log.Error(fmt.Errorf("invalid version in task status: %d", status.CurrentVersion),
						"Invalid version number in task status", "task", status.TaskName, "version", status.CurrentVersion)
					status.CurrentVersion = 0 // Reset to safe default
				}
			}
		case "symbolic":
			status.IsSymbolic = value == "true"
		case "confidence":
			if v, err := fmt.Sscanf(value, "%f", &status.PatternConfidence); err == nil && v == 1 {
				// parsed successfully
			}
		}
	}

	return status, nil
}

// validateConfigMapSize validates that the ConfigMap won't exceed Kubernetes size limits
func (r *LearningReconciler) validateConfigMapSize(configMap *corev1.ConfigMap) error {
	const maxConfigMapSize = 800 * 1024 // 800KB (staying under 1MB limit with margin)

	totalSize := 0

	// Calculate size of all data entries
	for key, value := range configMap.Data {
		totalSize += len(key) + len(value)
	}

	// Add metadata overhead (rough estimate)
	totalSize += len(configMap.Name) + len(configMap.Namespace) + 1024 // metadata overhead

	if totalSize > maxConfigMapSize {
		return fmt.Errorf("ConfigMap size (%d bytes) exceeds limit (%d bytes)", totalSize, maxConfigMapSize)
	}

	return nil
}

// cleanupStatusData removes old or less important status data to reduce ConfigMap size
func (r *LearningReconciler) cleanupStatusData(configMap *corev1.ConfigMap, agent *langopv1alpha1.LanguageAgent) (*corev1.ConfigMap, error) {
	r.Log.Info("Attempting to cleanup learning status data to reduce ConfigMap size",
		"configmap", configMap.Name, "current_entries", len(configMap.Data))

	// Parse all status entries to prioritize which to keep
	type statusEntry struct {
		key    string
		status *TaskLearningStatus
		size   int
	}

	var entries []statusEntry
	for key, value := range configMap.Data {
		if !strings.HasSuffix(key, "-status") {
			continue // Keep non-status data
		}

		status, err := r.parseTaskLearningStatus(value)
		if err != nil {
			r.Log.Error(err, "Failed to parse status during cleanup", "key", key)
			// Remove invalid entries
			delete(configMap.Data, key)
			continue
		}

		entries = append(entries, statusEntry{
			key:    key,
			status: status,
			size:   len(key) + len(value),
		})
	}

	// Sort by importance: symbolic tasks first (they're learned), then by trace count
	sort.Slice(entries, func(i, j int) bool {
		// Prioritize symbolic tasks (already learned)
		if entries[i].status.IsSymbolic && !entries[j].status.IsSymbolic {
			return true
		}
		if !entries[i].status.IsSymbolic && entries[j].status.IsSymbolic {
			return false
		}

		// Then prioritize by trace count (more traces = more valuable)
		return entries[i].status.TraceCount > entries[j].status.TraceCount
	})

	// Keep most important entries that fit within size limit
	const targetSize = 600 * 1024 // Target 600KB to leave room for growth
	currentSize := 0
	cleanedData := make(map[string]string)

	// Copy non-status data first
	for key, value := range configMap.Data {
		if !strings.HasSuffix(key, "-status") {
			cleanedData[key] = value
			currentSize += len(key) + len(value)
		}
	}

	// Add status entries in priority order until we hit size limit
	entriesKept := 0
	for _, entry := range entries {
		if currentSize+entry.size > targetSize {
			break // Would exceed target size
		}

		cleanedData[entry.key] = configMap.Data[entry.key]
		currentSize += entry.size
		entriesKept++
	}

	configMap.Data = cleanedData

	r.Log.Info("Cleaned up learning status data",
		"original_entries", len(entries),
		"kept_entries", entriesKept,
		"removed_entries", len(entries)-entriesKept,
		"estimated_size_kb", currentSize/1024)

	return configMap, nil
}

// checkLearningTriggers checks for conditions that should trigger learning
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
		statusData, err := r.serializeTaskLearningStatus(status)
		if err != nil {
			span.RecordError(err)
			r.Log.Error(err, "Failed to serialize task learning status", "task", taskName)
			continue // Skip this task status rather than failing the entire operation
		}
		configMap.Data[fmt.Sprintf("%s-status", taskName)] = statusData
	}

	// Validate ConfigMap size before update
	if err := r.validateConfigMapSize(configMap); err != nil {
		span.RecordError(err)
		r.Log.Error(err, "ConfigMap size validation failed", "configmap", configMapName)

		// Attempt to clean up old status data and retry
		if cleanedConfigMap, cleanupErr := r.cleanupStatusData(configMap, agent); cleanupErr == nil {
			configMap = cleanedConfigMap
			if validateErr := r.validateConfigMapSize(configMap); validateErr != nil {
				span.RecordError(validateErr)
				return fmt.Errorf("failed to update learning status: ConfigMap too large even after cleanup: %w", validateErr)
			}
		} else {
			span.RecordError(cleanupErr)
			return fmt.Errorf("failed to update learning status: %w, cleanup also failed: %w", err, cleanupErr)
		}
	}

	// Create or update ConfigMap
	var existing corev1.ConfigMap
	err := r.Get(ctx, types.NamespacedName{
		Name:      configMapName,
		Namespace: agent.Namespace,
	}, &existing)

	if apierrors.IsNotFound(err) {
		r.Log.V(1).Info("Creating new learning status ConfigMap", "configmap", configMapName)
		if err := r.Create(ctx, configMap); err != nil {
			span.RecordError(err)
			r.Log.Error(err, "Failed to create learning status ConfigMap",
				"configmap", configMapName, "agent", agent.Name)
			return fmt.Errorf("failed to create learning status ConfigMap: %w", err)
		}
		r.Log.Info("Successfully created learning status ConfigMap", "configmap", configMapName)
	} else if err != nil {
		span.RecordError(err)
		r.Log.Error(err, "Failed to get learning status ConfigMap",
			"configmap", configMapName, "agent", agent.Name)
		return fmt.Errorf("failed to get learning status ConfigMap: %w", err)
	} else {
		r.Log.V(1).Info("Updating existing learning status ConfigMap", "configmap", configMapName)
		existing.Data = configMap.Data
		if err := r.Update(ctx, &existing); err != nil {
			span.RecordError(err)
			r.Log.Error(err, "Failed to update learning status ConfigMap",
				"configmap", configMapName, "agent", agent.Name, "data_entries", len(configMap.Data))
			return fmt.Errorf("failed to update learning status ConfigMap: %w", err)
		}
		r.Log.Info("Successfully updated learning status ConfigMap",
			"configmap", configMapName, "status_entries", len(learningStatus))
	}

	// Update execution summary for Ruby gem consumption
	if err := r.updateExecutionSummary(ctx, agent, learningStatus); err != nil {
		span.RecordError(err)
		r.Log.Error(err, "Failed to update execution summary", "agent", agent.Name)
		// Don't fail the entire operation if summary update fails
	}

	// Add telemetry attributes for monitoring
	totalSize := 0
	for key, value := range configMap.Data {
		totalSize += len(key) + len(value)
	}

	span.SetAttributes(
		attribute.Int("learning.status_entries", len(learningStatus)),
		attribute.Int("learning.configmap_size_bytes", totalSize),
		attribute.Int("learning.configmap_size_kb", totalSize/1024),
		attribute.Bool("learning.status_update_success", true),
	)

	return nil
}

// updateAgentHealthMetrics calculates and updates learning health metrics in agent status
func (r *LearningReconciler) updateAgentHealthMetrics(ctx context.Context, agent *langopv1alpha1.LanguageAgent, learningStatus map[string]*TaskLearningStatus) error {
	ctx, span := learningTracer.Start(ctx, "learning.update_health_metrics")
	defer span.End()

	// Get or create the success rate aggregator for this agent
	agentKey := fmt.Sprintf("%s/%s", agent.Namespace, agent.Name)
	if r.SuccessRateAggregator == nil {
		r.SuccessRateAggregator = make(map[string]*learning.LearningSuccessRateAggregator)
	}
	if r.SuccessRateAggregator[agentKey] == nil {
		r.SuccessRateAggregator[agentKey] = learning.NewLearningSuccessRateAggregator(agent.Namespace, agent.Name, 24) // 24-hour window
	}
	aggregator := r.SuccessRateAggregator[agentKey]

	// Check if aggregator needs reset
	if aggregator.ShouldReset() {
		aggregator.Reset()
		r.Log.V(1).Info("Reset learning success rate aggregator window", "agent", agent.Name)
	}

	// Calculate health metrics from learning status
	var totalConfidence float64
	var confidenceCount int
	var symbolicTaskCount, neuralTaskCount int32
	var totalCostSavings float64

	for _, status := range learningStatus {
		if status.IsSymbolic {
			symbolicTaskCount++
			// Calculate cost savings for symbolic tasks
			if r.MetricsCollector != nil {
				costSavings := r.MetricsCollector.EstimateCostSavings(ctx, agent.Namespace, agent.Name, status.TaskName, 0.01, 0.0001, 10) // Sample values
				totalCostSavings += costSavings
			}
		} else {
			neuralTaskCount++
		}

		// Include pattern confidence in health calculation
		if status.PatternConfidence > 0 {
			totalConfidence += status.PatternConfidence
			confidenceCount++
		}
	}

	// Calculate average confidence
	avgConfidence := 0.0
	if confidenceCount > 0 {
		avgConfidence = totalConfidence / float64(confidenceCount)
	}

	// Get success rate from aggregator
	successRate := aggregator.CalculateSuccessRate()

	// Calculate error rate from recent learning attempts
	errorRate := 0.0
	totalAttempts := int64(0)
	failedAttempts := int64(0)
	for _, status := range learningStatus {
		if status.LearningAttempts > 0 {
			totalAttempts += int64(status.LearningAttempts)
			// Use error rate from task status if available
			failedAttempts += int64(float64(status.LearningAttempts) * status.ErrorRate)
		}
	}
	if totalAttempts > 0 {
		errorRate = float64(failedAttempts) / float64(totalAttempts)
	}

	// Calculate overall health score using HealthMetrics
	healthMetrics := learning.NewHealthMetrics(agent.Namespace, agent.Name)
	healthScore := healthMetrics.CalculateOverallLearningHealth(successRate, avgConfidence, errorRate)
	healthCategory := healthMetrics.GetHealthCategory(healthScore)

	// Project monthly cost savings
	monthlyCostSavings := totalCostSavings * 30 // Assuming daily savings

	// Initialize agent metrics if needed
	if agent.Status.Metrics == nil {
		agent.Status.Metrics = &langopv1alpha1.AgentMetrics{}
	}

	// Update learning health metrics
	agent.Status.Metrics.LearningHealthScore = &healthScore
	agent.Status.Metrics.LearningHealthCategory = healthCategory
	agent.Status.Metrics.LearningSuccessRate = &successRate
	agent.Status.Metrics.SymbolicTaskCount = symbolicTaskCount
	agent.Status.Metrics.NeuralTaskCount = neuralTaskCount
	agent.Status.Metrics.ProjectedMonthlyCostSavings = &monthlyCostSavings

	// Update agent status
	if err := r.Status().Update(ctx, agent); err != nil {
		span.RecordError(err)
		return fmt.Errorf("failed to update agent health metrics in status: %w", err)
	}

	// Update success rate metrics in telemetry
	if r.MetricsCollector != nil {
		r.MetricsCollector.UpdateLearningSuccessRates(ctx, agent.Namespace, agent.Name, aggregator)
	}

	span.SetAttributes(
		attribute.Float64("learning.health_score", healthScore),
		attribute.String("learning.health_category", healthCategory),
		attribute.Float64("learning.success_rate", successRate),
		attribute.Int("learning.symbolic_tasks", int(symbolicTaskCount)),
		attribute.Int("learning.neural_tasks", int(neuralTaskCount)),
		attribute.Float64("learning.monthly_cost_savings", monthlyCostSavings),
	)

	r.Log.V(1).Info("Updated agent health metrics",
		"agent", agent.Name,
		"health_score", healthScore,
		"health_category", healthCategory,
		"success_rate", successRate,
		"symbolic_tasks", symbolicTaskCount,
		"neural_tasks", neuralTaskCount,
		"monthly_savings", monthlyCostSavings)

	return nil
}

// serializeTaskLearningStatus serializes task learning status to JSON string
func (r *LearningReconciler) serializeTaskLearningStatus(status *TaskLearningStatus) (string, error) {
	data, err := json.Marshal(status)
	if err != nil {
		return "", fmt.Errorf("failed to marshal task learning status: %w", err)
	}
	return string(data), nil
}

// recordLearningEvent records a learning event for auditing and monitoring
func (r *LearningReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&langopv1alpha1.LanguageAgent{}).
		Watches(&batchv1.Job{},
			handler.EnqueueRequestsFromMapFunc(r.mapJobToAgent)).
		Watches(&corev1.Event{},
			handler.EnqueueRequestsFromMapFunc(r.mapEventToAgent),
			builder.WithPredicates(predicate.NewPredicateFuncs(func(object client.Object) bool {
				if event, ok := object.(*corev1.Event); ok {
					return event.Reason == "TaskCompleted" && event.InvolvedObject.Kind == "LanguageAgent"
				}
				return false
			}))).
		Named("learning").
		Complete(r)
}

// mapJobToAgent maps Job completion events to LanguageAgent reconciliation requests
func (r *LearningReconciler) mapJobToAgent(ctx context.Context, obj client.Object) []reconcile.Request {
	job, ok := obj.(*batchv1.Job)
	if !ok {
		return nil
	}

	// Check if this job belongs to a language operator agent
	agentName, exists := job.Labels["app.kubernetes.io/name"]
	if !exists {
		return nil
	}

	// Only trigger learning reconciliation on job completion (success or failure)
	if job.Status.Succeeded == 0 && job.Status.Failed == 0 {
		// Job is still running, not completed yet
		return nil
	}

	r.Log.V(1).Info("Job completion detected, triggering learning reconciliation",
		"job", job.Name,
		"agent", agentName,
		"namespace", job.Namespace,
		"succeeded", job.Status.Succeeded,
		"failed", job.Status.Failed)

	return []reconcile.Request{
		{
			NamespacedName: types.NamespacedName{
				Name:      agentName,
				Namespace: job.Namespace,
			},
		},
	}
}

// getExecutionTraces retrieves execution traces for pattern analysis
func (r *LearningReconciler) getExecutionTraces(ctx context.Context, agent *langopv1alpha1.LanguageAgent) ([]TaskTrace, error) {
	ctx, span := learningTracer.Start(ctx, "learning.get_traces")
	defer span.End()

	// Check if telemetry adapter is available
	if r.TelemetryAdapter == nil || !r.TelemetryAdapter.Available() {
		r.Log.V(1).Info("Telemetry adapter not available, returning empty traces",
			"agent", agent.Name, "namespace", agent.Namespace)
		span.SetAttributes(
			attribute.String("learning.adapter_status", "unavailable"),
			attribute.Int("learning.traces_retrieved", 0),
		)
		return []TaskTrace{}, nil
	}

	// Query spans from the last 24 hours for this agent
	timeRange := telemetry.TimeRange{
		Start: time.Now().Add(-24 * time.Hour),
		End:   time.Now(),
	}

	filter := telemetry.SpanFilter{
		TimeRange: timeRange,
		// Use semantic attributes like the original gem implementation
		Attributes: map[string]string{
			"service.name": fmt.Sprintf("language-operator-agent-%s", agent.Name),
		},
		Limit: 1000, // Reasonable limit for pattern analysis
	}

	spans, err := r.TelemetryAdapter.QuerySpans(ctx, filter)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "Failed to query spans")
		r.Log.Error(err, "Failed to query execution traces, continuing with empty traces",
			"agent", agent.Name, "namespace", agent.Namespace)
		// Don't fail - continue with empty traces to allow ConfigMap creation
		span.SetAttributes(
			attribute.String("learning.adapter_status", "error"),
			attribute.Int("learning.traces_retrieved", 0),
		)
		return []TaskTrace{}, nil
	}

	// Convert telemetry spans to TaskTrace format
	traces := r.convertSpansToTaskTraces(spans)

	// Summarize traces to reduce data size for ConfigMap storage
	summarizedTraces := r.summarizeTraces(traces)

	span.SetAttributes(
		attribute.String("learning.adapter_status", "available"),
		attribute.Int("learning.spans_queried", len(spans)),
		attribute.Int("learning.traces_retrieved", len(traces)),
		attribute.Int("learning.traces_summarized", len(summarizedTraces)),
	)

	r.Log.V(1).Info("Retrieved execution traces for learning",
		"agent", agent.Name,
		"namespace", agent.Namespace,
		"traces_count", len(traces),
		"summarized_count", len(summarizedTraces))

	return summarizedTraces, nil
}

// convertSpansToTaskTraces converts telemetry spans to TaskTrace format
func (r *LearningReconciler) convertSpansToTaskTraces(spans []telemetry.Span) []TaskTrace {
	var traces []TaskTrace

	for _, span := range spans {
		// Only process spans that represent task executions
		if span.OperationName != "execute_task" || span.TaskName == "" {
			continue
		}

		// Extract task inputs/outputs from span attributes
		inputs := r.parseJSONAttribute(span.Attributes["task.inputs"])
		outputs := r.parseJSONAttribute(span.Attributes["task.outputs"])

		// Extract tool calls from child spans (simplified)
		toolCalls := r.extractToolCallsFromSpan(span)

		trace := TaskTrace{
			TaskName:     span.TaskName,
			Timestamp:    span.StartTime,
			Inputs:       inputs,
			Outputs:      outputs,
			ToolCalls:    toolCalls,
			Duration:     span.Duration,
			Success:      span.Status,
			ErrorMessage: span.ErrorMessage,
		}

		traces = append(traces, trace)
	}

	return traces
}

// parseJSONAttribute safely parses JSON from span attributes
func (r *LearningReconciler) parseJSONAttribute(jsonStr string) map[string]interface{} {
	if jsonStr == "" {
		return map[string]interface{}{}
	}

	var result map[string]interface{}
	if err := json.Unmarshal([]byte(jsonStr), &result); err != nil {
		r.Log.V(1).Info("Failed to parse JSON attribute", "json", jsonStr, "error", err)
		return map[string]interface{}{}
	}

	return result
}

// extractToolCallsFromSpan extracts tool calls from span attributes
func (r *LearningReconciler) extractToolCallsFromSpan(span telemetry.Span) []ToolCall {
	var toolCalls []ToolCall

	// Check if this is a tool execution span
	if span.Attributes["gen_ai.operation.name"] == "execute_tool" {
		toolCall := ToolCall{
			ToolName:   span.Attributes["gen_ai.tool.name"],
			Method:     "execute", // Default method for MCP tools
			Parameters: r.parseJSONAttribute(span.Attributes["gen_ai.tool.call.arguments"]),
			Result:     r.parseJSONAttribute(span.Attributes["gen_ai.tool.call.result"]),
		}

		if toolCall.ToolName != "" {
			toolCalls = append(toolCalls, toolCall)
		}
	}

	// Also check for tool call sequence in task.tool_calls attribute (if available)
	if toolCallsData := span.Attributes["task.tool_calls"]; toolCallsData != "" {
		parsedCalls := r.parseToolCallSequence(toolCallsData)
		toolCalls = append(toolCalls, parsedCalls...)
	}

	return toolCalls
}

// parseToolCallSequence parses a JSON array of tool calls from span attributes
func (r *LearningReconciler) parseToolCallSequence(toolCallsJSON string) []ToolCall {
	var toolCalls []ToolCall

	var rawCalls []map[string]interface{}
	if err := json.Unmarshal([]byte(toolCallsJSON), &rawCalls); err != nil {
		r.Log.V(1).Info("Failed to parse tool call sequence", "json", toolCallsJSON, "error", err)
		return toolCalls
	}

	for _, call := range rawCalls {
		toolCall := ToolCall{
			ToolName: getString(call, "tool_name"),
			Method:   getString(call, "method"),
		}

		if params, ok := call["parameters"]; ok {
			if paramsMap, ok := params.(map[string]interface{}); ok {
				toolCall.Parameters = paramsMap
			}
		}

		if result, ok := call["result"]; ok {
			if resultMap, ok := result.(map[string]interface{}); ok {
				toolCall.Result = resultMap
			}
		}

		if toolCall.ToolName != "" {
			toolCalls = append(toolCalls, toolCall)
		}
	}

	return toolCalls
}

// getString safely extracts a string value from a map
func getString(m map[string]interface{}, key string) string {
	if val, ok := m[key]; ok {
		if str, ok := val.(string); ok {
			return str
		}
	}
	return ""
}

// groupTracesByTask groups execution traces by task name
func (r *LearningReconciler) groupTracesByTask(traces []TaskTrace) map[string][]TaskTrace {
	taskGroups := make(map[string][]TaskTrace)

	for _, trace := range traces {
		taskGroups[trace.TaskName] = append(taskGroups[trace.TaskName], trace)
	}

	return taskGroups
}

// summarizeTraces reduces trace data size by deduplicating patterns and summarizing outputs
func (r *LearningReconciler) summarizeTraces(traces []TaskTrace) []TaskTrace {
	if len(traces) == 0 {
		return traces
	}

	// Group traces by task for pattern-based summarization
	taskGroups := r.groupTracesByTask(traces)
	var summarizedTraces []TaskTrace

	for taskName, taskTraceList := range taskGroups {
		// Keep only recent and unique pattern traces per task
		summarized := r.summarizeTaskTraces(taskName, taskTraceList)
		summarizedTraces = append(summarizedTraces, summarized...)
	}

	// Sort by timestamp to maintain chronological order
	sort.Slice(summarizedTraces, func(i, j int) bool {
		return summarizedTraces[i].Timestamp.Before(summarizedTraces[j].Timestamp)
	})

	// Limit total number of traces to prevent excessive ConfigMap size
	maxTraces := 50 // Configurable limit
	if len(summarizedTraces) > maxTraces {
		// Keep the most recent traces
		summarizedTraces = summarizedTraces[len(summarizedTraces)-maxTraces:]
	}

	return summarizedTraces
}

// summarizeTaskTraces summarizes traces for a specific task
func (r *LearningReconciler) summarizeTaskTraces(taskName string, traces []TaskTrace) []TaskTrace {
	if len(traces) <= 5 {
		// Keep all traces if there are few of them
		return traces
	}

	// Sort by timestamp (most recent first)
	sort.Slice(traces, func(i, j int) bool {
		return traces[i].Timestamp.After(traces[j].Timestamp)
	})

	// Keep patterns: recent traces + representative examples of each pattern
	var result []TaskTrace
	toolCallPatterns := make(map[string]bool)

	// Always keep the most recent 3 traces
	recentCount := 3
	if len(traces) < recentCount {
		recentCount = len(traces)
	}
	result = append(result, traces[:recentCount]...)

	// Add representative examples of different tool call patterns
	for i := recentCount; i < len(traces) && len(result) < 15; i++ {
		trace := traces[i]

		// Create a simplified pattern signature from tool calls
		patternSignature := r.createToolCallSignature(trace.ToolCalls)

		// Keep this trace if it represents a new pattern
		if !toolCallPatterns[patternSignature] {
			toolCallPatterns[patternSignature] = true

			// Summarize large inputs/outputs to reduce size
			summarizedTrace := r.summarizeTraceData(trace)
			result = append(result, summarizedTrace)
		}
	}

	return result
}

// createToolCallSignature creates a pattern signature from tool calls
func (r *LearningReconciler) createToolCallSignature(toolCalls []ToolCall) string {
	if len(toolCalls) == 0 {
		return "no_tools"
	}

	var signature strings.Builder
	for i, call := range toolCalls {
		if i > 0 {
			signature.WriteString("->")
		}
		signature.WriteString(fmt.Sprintf("%s.%s", call.ToolName, call.Method))
	}

	return signature.String()
}

// summarizeTraceData reduces the size of individual trace data
func (r *LearningReconciler) summarizeTraceData(trace TaskTrace) TaskTrace {
	summarized := trace

	// Summarize large input/output data
	summarized.Inputs = r.summarizeData(trace.Inputs)
	summarized.Outputs = r.summarizeData(trace.Outputs)

	// Summarize tool call results
	for i, toolCall := range trace.ToolCalls {
		summarized.ToolCalls[i].Parameters = r.summarizeData(toolCall.Parameters)

		// Handle Result which may be of different types
		if resultMap, ok := toolCall.Result.(map[string]interface{}); ok {
			summarized.ToolCalls[i].Result = r.summarizeData(resultMap)
		} else {
			// Keep non-map results as-is (strings, numbers, etc.)
			summarized.ToolCalls[i].Result = toolCall.Result
		}
	}

	return summarized
}

// summarizeData reduces the size of data maps by truncating large values
func (r *LearningReconciler) summarizeData(data map[string]interface{}) map[string]interface{} {
	if len(data) == 0 {
		return data
	}

	summarized := make(map[string]interface{})
	const maxFieldSize = 256 // Maximum size for individual field values

	for key, value := range data {
		switch v := value.(type) {
		case string:
			if len(v) > maxFieldSize {
				summarized[key] = fmt.Sprintf("%s...[truncated %d bytes]",
					v[:maxFieldSize-50], len(v)-maxFieldSize+50)
			} else {
				summarized[key] = v
			}
		case []interface{}:
			if len(v) > 10 {
				// Truncate large arrays
				truncated := v[:10]
				summarized[key] = append(truncated, fmt.Sprintf("...[%d more items]", len(v)-10))
			} else {
				summarized[key] = v
			}
		case map[string]interface{}:
			// Recursively summarize nested maps
			if len(v) > 0 {
				summarized[key] = r.summarizeData(v)
			} else {
				summarized[key] = v
			}
		default:
			summarized[key] = value
		}
	}

	return summarized
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

	// Note: Recommended code generation now handled by threshold-based learning via LLM synthesis
	// (removed event-based generatePatternBasedCode)

	return analysis, nil
}

// analyzeToolCallPatterns analyzes patterns in tool calls across traces
func (r *LearningReconciler) analyzeToolCallPatterns(traces []TaskTrace) map[string]int {
	patterns := make(map[string]int)

	for _, trace := range traces {
		// Use the same normalization logic as the consistency analysis
		pattern := r.normalizeToolCallPattern(trace.ToolCalls)
		if pattern != "" {
			patterns[pattern]++
		}
	}

	return patterns
}

// analyzeInputOutputConsistency measures how consistent inputs and outputs are
func (r *LearningReconciler) analyzeInputOutputConsistency(traces []TaskTrace) float64 {
	if len(traces) < 2 {
		return 0.0
	}

	// Group traces by input signature (following Ruby implementation)
	inputGroups := r.groupTracesByInputSignature(traces)

	// Calculate consistency for each input signature
	var totalExecutions int
	var weightedConsistency float64

	for inputSignature, groupTraces := range inputGroups {
		// Analyze tool call patterns within this input group
		toolPatterns := make(map[string]int)
		for _, trace := range groupTraces {
			pattern := r.normalizeToolCallPattern(trace.ToolCalls)
			toolPatterns[pattern]++
		}

		// Find the most common pattern for this input signature
		var mostCommon string
		var mostCommonCount int
		for pattern, count := range toolPatterns {
			if count > mostCommonCount {
				mostCommon = pattern
				mostCommonCount = count
			}
		}

		// Calculate consistency for this input signature
		groupTotal := len(groupTraces)
		groupConsistency := float64(mostCommonCount) / float64(groupTotal)

		// Weight by number of executions
		weight := float64(groupTotal)
		weightedConsistency += weight * groupConsistency
		totalExecutions += groupTotal

		r.Log.V(2).Info("Input signature consistency analysis",
			"inputSignature", inputSignature,
			"executions", groupTotal,
			"mostCommonPattern", mostCommon,
			"consistency", groupConsistency)
	}

	if totalExecutions == 0 {
		return 0.0
	}

	// Return weighted average consistency across all input signatures
	overallConsistency := weightedConsistency / float64(totalExecutions)
	return overallConsistency
}

// groupTracesByInputSignature groups traces by their normalized input signature
func (r *LearningReconciler) groupTracesByInputSignature(traces []TaskTrace) map[string][]TaskTrace {
	inputGroups := make(map[string][]TaskTrace)

	for _, trace := range traces {
		signature := r.normalizeInputSignature(trace.Inputs)
		inputGroups[signature] = append(inputGroups[signature], trace)
	}

	return inputGroups
}

// normalizeInputSignature creates a consistent signature from input parameters
func (r *LearningReconciler) normalizeInputSignature(inputs map[string]interface{}) string {
	if len(inputs) == 0 {
		return ""
	}

	// Convert to deterministic string representation
	// Sort keys to ensure consistent ordering
	keys := make([]string, 0, len(inputs))
	for k := range inputs {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	var parts []string
	for _, key := range keys {
		value := inputs[key]
		parts = append(parts, fmt.Sprintf("%s:%v", key, value))
	}

	return strings.Join(parts, ",")
}

// normalizeToolCallPattern creates a pattern string from tool call sequence
func (r *LearningReconciler) normalizeToolCallPattern(toolCalls []ToolCall) string {
	if len(toolCalls) == 0 {
		return ""
	}

	// Extract tool names in sequence
	var toolNames []string
	for _, call := range toolCalls {
		toolNames = append(toolNames, call.ToolName)
	}

	// Collapse consecutive duplicates (following Ruby implementation)
	return r.collapseConsecutiveDuplicates(toolNames)
}

// collapseConsecutiveDuplicates collapses consecutive duplicate tool calls
func (r *LearningReconciler) collapseConsecutiveDuplicates(toolNames []string) string {
	if len(toolNames) == 0 {
		return ""
	}

	var collapsed []string
	currentTool := ""
	count := 0

	for _, tool := range toolNames {
		if tool == currentTool {
			count++
		} else {
			// Add previous tool (if any) to result
			if currentTool != "" {
				collapsed = append(collapsed, r.formatToolWithCount(currentTool, count))
			}
			// Start new sequence
			currentTool = tool
			count = 1
		}
	}

	// Add final tool
	if currentTool != "" {
		collapsed = append(collapsed, r.formatToolWithCount(currentTool, count))
	}

	return strings.Join(collapsed, " → ")
}

// formatToolWithCount formats tool name with count if > 1
func (r *LearningReconciler) formatToolWithCount(toolName string, count int) string {
	if count > 1 {
		return fmt.Sprintf("%s (×%d)", toolName, count)
	}
	return toolName
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

	// Check for conditional patterns (multiple but limited patterns) first
	if len(toolCallPatterns) > 1 && len(toolCallPatterns) <= 3 {
		return "conditional_logic"
	}

	// Check for transformation patterns (only if not conditional)
	transformationKeywords := []string{"transform", "convert", "process", "format"}
	for _, keyword := range transformationKeywords {
		if strings.Contains(strings.ToLower(mostCommonPattern), keyword) {
			return "simple_transformation"
		}
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

// ProcessAgentExecution processes agent execution events and updates learning status metrics
func (r *LearningReconciler) ProcessAgentExecution(ctx context.Context, agent *langopv1alpha1.LanguageAgent, taskName string, success bool, executionTime time.Time) error {
	ctx, span := learningTracer.Start(ctx, "learning.process_agent_execution")
	defer span.End()

	span.SetAttributes(
		attribute.String("learning.agent", agent.Name),
		attribute.String("learning.namespace", agent.Namespace),
		attribute.String("learning.task", taskName),
		attribute.Bool("learning.success", success),
	)

	// Get current learning status
	learningStatus, err := r.getLearningStatus(ctx, agent)
	if err != nil {
		span.RecordError(err)
		return fmt.Errorf("failed to get learning status: %w", err)
	}

	// Update task execution metrics
	taskStatus, exists := learningStatus[taskName]
	if !exists {
		taskStatus = &TaskLearningStatus{
			TaskName:       taskName,
			CurrentVersion: 1,
		}
		learningStatus[taskName] = taskStatus
	}

	// Update execution counters
	taskStatus.TotalExecutions++
	taskStatus.LastExecutionTime = executionTime

	if success {
		taskStatus.SuccessfulExecutions++
	} else {
		taskStatus.FailedExecutions++
	}

	// Calculate success rate
	if taskStatus.TotalExecutions > 0 {
		taskStatus.SuccessRate = float64(taskStatus.SuccessfulExecutions) / float64(taskStatus.TotalExecutions)
	}

	// Determine learning status
	taskStatus.LearningStatus = r.determineLearningStatus(taskStatus)

	// Update learning status in ConfigMap
	if err := r.updateLearningStatus(ctx, agent, learningStatus); err != nil {
		span.RecordError(err)
		return fmt.Errorf("failed to update learning status: %w", err)
	}

	// Update execution summary
	if err := r.updateExecutionSummary(ctx, agent, learningStatus); err != nil {
		span.RecordError(err)
		r.Log.Error(err, "Failed to update execution summary", "agent", agent.Name, "task", taskName)
		// Don't fail the entire operation if summary update fails
	}

	span.SetAttributes(
		attribute.Int("learning.total_executions", int(taskStatus.TotalExecutions)),
		attribute.Float64("learning.success_rate", taskStatus.SuccessRate),
		attribute.String("learning.status", taskStatus.LearningStatus),
	)

	r.Log.V(1).Info("Updated execution metrics for task",
		"agent", agent.Name,
		"task", taskName,
		"total_executions", taskStatus.TotalExecutions,
		"success_rate", taskStatus.SuccessRate,
		"status", taskStatus.LearningStatus)

	return nil
}

// determineLearningStatus determines the current learning status based on execution metrics
func (r *LearningReconciler) determineLearningStatus(status *TaskLearningStatus) string {
	// If already symbolic, return symbolic
	if status.IsSymbolic {
		return "symbolic"
	}

	// Check if ready for symbolic conversion
	if status.TotalExecutions >= r.LearningThreshold &&
		status.SuccessRate >= 0.8 &&
		status.PatternConfidence >= r.PatternConfidenceMin {
		return "ready_for_symbolic"
	}

	// Otherwise still learning
	return "learning"
}

// GenerateExecutionSummary creates an execution summary for Ruby gem consumption
func (r *LearningReconciler) GenerateExecutionSummary(ctx context.Context, agent *langopv1alpha1.LanguageAgent, learningStatus map[string]*TaskLearningStatus) (*AgentExecutionSummary, error) {
	ctx, span := learningTracer.Start(ctx, "learning.generate_execution_summary")
	defer span.End()

	summary := &AgentExecutionSummary{
		LearningThreshold: r.LearningThreshold,
		Tasks:             make(map[string]TaskExecutionStatus),
		Created:           time.Now(),
		Updated:           time.Now(),
	}

	var totalExecutions int32
	var totalSuccessful int32
	var lastExecution time.Time

	// Aggregate data from all tasks
	for taskName, status := range learningStatus {
		totalExecutions += status.TotalExecutions
		totalSuccessful += status.SuccessfulExecutions

		if status.LastExecutionTime.After(lastExecution) {
			lastExecution = status.LastExecutionTime
		}

		// Create task status entry
		summary.Tasks[taskName] = TaskExecutionStatus{
			Executions:  status.TotalExecutions,
			SuccessRate: status.SuccessRate,
			Confidence:  status.PatternConfidence,
			Status:      status.LearningStatus,
		}
	}

	// Calculate overall metrics
	summary.TotalExecutions = totalExecutions
	summary.LastExecution = lastExecution

	if totalExecutions > 0 {
		summary.SuccessRate = float64(totalSuccessful) / float64(totalExecutions)
	}

	span.SetAttributes(
		attribute.Int("learning.total_executions", int(totalExecutions)),
		attribute.Float64("learning.overall_success_rate", summary.SuccessRate),
		attribute.Int("learning.task_count", len(summary.Tasks)),
	)

	return summary, nil
}

// updateExecutionSummary updates the execution summary in the ConfigMap
func (r *LearningReconciler) updateExecutionSummary(ctx context.Context, agent *langopv1alpha1.LanguageAgent, learningStatus map[string]*TaskLearningStatus) error {
	ctx, span := learningTracer.Start(ctx, "learning.update_execution_summary")
	defer span.End()

	// Generate execution summary
	summary, err := r.GenerateExecutionSummary(ctx, agent, learningStatus)
	if err != nil {
		span.RecordError(err)
		return fmt.Errorf("failed to generate execution summary: %w", err)
	}

	// Get existing ConfigMap
	configMapName := fmt.Sprintf("%s-learning-status", agent.Name)
	configMap := &corev1.ConfigMap{}
	err = r.Get(ctx, types.NamespacedName{
		Name:      configMapName,
		Namespace: agent.Namespace,
	}, configMap)

	if apierrors.IsNotFound(err) {
		// Create new ConfigMap if it doesn't exist
		configMap = &corev1.ConfigMap{
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

		if err := controllerutil.SetControllerReference(agent, configMap, r.Scheme); err != nil {
			span.RecordError(err)
			return fmt.Errorf("failed to set owner reference: %w", err)
		}
	} else if err != nil {
		span.RecordError(err)
		return fmt.Errorf("failed to get ConfigMap: %w", err)
	}

	// Ensure Data field is initialized
	if configMap.Data == nil {
		configMap.Data = make(map[string]string)
	}

	// Serialize execution summary
	summaryData, err := json.Marshal(summary)
	if err != nil {
		span.RecordError(err)
		return fmt.Errorf("failed to marshal execution summary: %w", err)
	}

	// Update ConfigMap with execution summary
	configMap.Data["execution-summary"] = string(summaryData)

	// Create or update ConfigMap
	if configMap.ResourceVersion == "" {
		err = r.Create(ctx, configMap)
	} else {
		err = r.Update(ctx, configMap)
	}

	if err != nil {
		span.RecordError(err)
		return fmt.Errorf("failed to update ConfigMap: %w", err)
	}

	span.SetAttributes(
		attribute.Int("learning.summary_size", len(summaryData)),
		attribute.Int("learning.tasks_included", len(summary.Tasks)),
	)

	r.Log.V(1).Info("Updated execution summary",
		"agent", agent.Name,
		"total_executions", summary.TotalExecutions,
		"success_rate", summary.SuccessRate,
		"tasks", len(summary.Tasks))

	return nil
}

// processAgentExecutions finds and processes TaskCompleted events and completed Jobs for an agent to track execution metrics
func (r *LearningReconciler) processAgentExecutions(ctx context.Context, agent *langopv1alpha1.LanguageAgent) error {
	ctx, span := learningTracer.Start(ctx, "learning.process_agent_executions")
	defer span.End()

	log := r.Log.WithValues("agent", agent.Name, "namespace", agent.Namespace)

	// Process TaskCompleted events (preferred method)
	if err := r.processTaskCompletedEvents(ctx, agent); err != nil {
		log.Error(err, "Failed to process TaskCompleted events")
		// Don't return error - continue with fallback Job processing
	}

	// Fallback: Process completed Jobs (for backward compatibility)
	jobList := &batchv1.JobList{}
	listOpts := []client.ListOption{
		client.InNamespace(agent.Namespace),
		client.MatchingLabels{
			"app.kubernetes.io/name": agent.Name,
		},
	}

	if err := r.List(ctx, jobList, listOpts...); err != nil {
		return fmt.Errorf("failed to list Jobs for agent %s: %w", agent.Name, err)
	}

	log.V(1).Info("Found Jobs for agent", "count", len(jobList.Items))

	// Process each completed Job
	for _, job := range jobList.Items {
		if err := r.processJobExecution(ctx, agent, &job); err != nil {
			log.Error(err, "Failed to process Job execution", "job", job.Name)
			// Continue processing other jobs
		}
	}

	return nil
}

// processJobExecution processes a single Job execution and calls ProcessAgentExecution
func (r *LearningReconciler) processJobExecution(ctx context.Context, agent *langopv1alpha1.LanguageAgent, job *batchv1.Job) error {
	ctx, span := learningTracer.Start(ctx, "learning.process_job_execution")
	defer span.End()

	log := r.Log.WithValues("agent", agent.Name, "job", job.Name)

	// Only process completed jobs
	if job.Status.Succeeded == 0 && job.Status.Failed == 0 {
		log.V(2).Info("Job not yet completed, skipping")
		return nil
	}

	// Check if we've already processed this job
	if r.isJobProcessed(ctx, agent, job) {
		log.V(2).Info("Job already processed, skipping")
		return nil
	}

	// Determine success/failure
	success := job.Status.Succeeded > 0
	executionTime := job.Status.CompletionTime
	if executionTime == nil {
		// Fallback to creation time if completion time is not available
		executionTime = &job.CreationTimestamp
	}

	// Extract task name from job labels or annotations
	taskName := r.extractTaskName(job)

	log.Info("Processing Job execution",
		"task", taskName,
		"success", success,
		"completion_time", executionTime.Time)

	// Call ProcessAgentExecution to update metrics
	if err := r.ProcessAgentExecution(ctx, agent, taskName, success, executionTime.Time); err != nil {
		return fmt.Errorf("failed to process agent execution: %w", err)
	}

	// Mark this job as processed
	if err := r.markJobAsProcessed(ctx, agent, job); err != nil {
		log.Error(err, "Failed to mark job as processed", "job", job.Name)
		// Don't fail - this is just for optimization
	}

	return nil
}

// isJobProcessed checks if we've already processed this Job
func (r *LearningReconciler) isJobProcessed(ctx context.Context, agent *langopv1alpha1.LanguageAgent, job *batchv1.Job) bool {
	// Check if job has our processing annotation
	if job.Annotations != nil {
		if _, exists := job.Annotations["langop.io/learning-processed"]; exists {
			return true
		}
	}
	return false
}

// markJobAsProcessed marks a Job as processed to avoid duplicate processing
func (r *LearningReconciler) markJobAsProcessed(ctx context.Context, agent *langopv1alpha1.LanguageAgent, job *batchv1.Job) error {
	if job.Annotations == nil {
		job.Annotations = make(map[string]string)
	}
	job.Annotations["langop.io/learning-processed"] = time.Now().Format(time.RFC3339)

	return r.Update(ctx, job)
}

// extractTaskName extracts the task name from Job metadata
func (r *LearningReconciler) extractTaskName(job *batchv1.Job) string {
	// Try to get task name from labels first
	if job.Labels != nil {
		if taskName, exists := job.Labels["langop.io/task"]; exists {
			return taskName
		}
	}

	// Try annotations
	if job.Annotations != nil {
		if taskName, exists := job.Annotations["langop.io/task"]; exists {
			return taskName
		}
	}

	// Fallback to job name or a default
	if strings.Contains(job.Name, "-") {
		// For CronJob generated jobs like "agent-name-123456", extract the task
		parts := strings.Split(job.Name, "-")
		if len(parts) > 1 {
			return strings.Join(parts[:len(parts)-1], "-")
		}
	}

	return "main_task" // Default task name
}

// mapEventToAgent maps TaskCompleted events to LanguageAgent reconciliation requests
func (r *LearningReconciler) mapEventToAgent(ctx context.Context, obj client.Object) []reconcile.Request {
	event, ok := obj.(*corev1.Event)
	if !ok {
		return nil
	}

	// Only process TaskCompleted events
	if event.Reason != "TaskCompleted" {
		return nil
	}

	// Extract agent name from the involved object
	if event.InvolvedObject.Kind != "LanguageAgent" {
		return nil
	}

	agentName := event.InvolvedObject.Name
	namespace := event.InvolvedObject.Namespace

	r.Log.V(1).Info("TaskCompleted event detected, triggering learning reconciliation",
		"event", event.Name,
		"agent", agentName,
		"namespace", namespace,
		"task", r.extractTaskFromEventMessage(event.Message))

	return []reconcile.Request{
		{
			NamespacedName: types.NamespacedName{
				Name:      agentName,
				Namespace: namespace,
			},
		},
	}
}

// extractTaskFromEventMessage parses task name from TaskCompleted event message
func (r *LearningReconciler) extractTaskFromEventMessage(message string) string {
	// Expected format: "Task 'task_name' completed successfully in 1234ms (task_type: neural)"
	// Use regex to extract task name between single quotes
	if strings.Contains(message, "Task '") && strings.Contains(message, "' completed") {
		start := strings.Index(message, "Task '") + 6
		end := strings.Index(message[start:], "' completed")
		if end > 0 {
			return message[start : start+end]
		}
	}
	return "unknown_task"
}

// processTaskCompletedEvents finds and processes recent TaskCompleted events for an agent
func (r *LearningReconciler) processTaskCompletedEvents(ctx context.Context, agent *langopv1alpha1.LanguageAgent) error {
	ctx, span := learningTracer.Start(ctx, "learning.process_task_completed_events")
	defer span.End()

	log := r.Log.WithValues("agent", agent.Name, "namespace", agent.Namespace)

	// Find TaskCompleted events for this agent
	eventList := &corev1.EventList{}
	listOpts := []client.ListOption{
		client.InNamespace(agent.Namespace),
	}

	if err := r.List(ctx, eventList, listOpts...); err != nil {
		return fmt.Errorf("failed to list Events for agent %s: %w", agent.Name, err)
	}

	log.V(1).Info("Found events for namespace", "count", len(eventList.Items))

	// Process TaskCompleted events for this agent
	processedCount := 0
	for _, event := range eventList.Items {
		if err := r.processTaskCompletedEvent(ctx, agent, &event); err != nil {
			log.Error(err, "Failed to process TaskCompleted event", "event", event.Name)
			// Continue processing other events
		} else if r.isRelevantTaskEvent(&event, agent) {
			processedCount++
		}
	}

	log.V(1).Info("Processed TaskCompleted events", "count", processedCount)
	return nil
}

// processTaskCompletedEvent processes a single TaskCompleted event
func (r *LearningReconciler) processTaskCompletedEvent(ctx context.Context, agent *langopv1alpha1.LanguageAgent, event *corev1.Event) error {
	// Only process TaskCompleted events for this agent
	if !r.isRelevantTaskEvent(event, agent) {
		return nil
	}

	ctx, span := learningTracer.Start(ctx, "learning.process_task_completed_event")
	defer span.End()

	log := r.Log.WithValues("agent", agent.Name, "event", event.Name)

	// Check if we've already processed this event
	if r.isEventProcessed(ctx, agent, event) {
		log.V(2).Info("Event already processed, skipping")
		return nil
	}

	// Parse event data
	taskName := r.extractTaskFromEventMessage(event.Message)
	success := r.isTaskSuccessfulFromEventMessage(event.Message)
	executionTime := event.CreationTimestamp.Time

	log.Info("Processing TaskCompleted event",
		"task", taskName,
		"success", success,
		"timestamp", executionTime)

	// Call ProcessAgentExecution to update metrics
	if err := r.ProcessAgentExecution(ctx, agent, taskName, success, executionTime); err != nil {
		return fmt.Errorf("failed to process agent execution from event: %w", err)
	}

	// Mark this event as processed
	if err := r.markEventAsProcessed(ctx, agent, event); err != nil {
		log.Error(err, "Failed to mark event as processed", "event", event.Name)
		// Don't fail - this is just for optimization
	}

	return nil
}

// isRelevantTaskEvent checks if an event is a TaskCompleted event for the given agent
func (r *LearningReconciler) isRelevantTaskEvent(event *corev1.Event, agent *langopv1alpha1.LanguageAgent) bool {
	return event.Reason == "TaskCompleted" &&
		event.InvolvedObject.Kind == "LanguageAgent" &&
		event.InvolvedObject.Name == agent.Name &&
		event.InvolvedObject.Namespace == agent.Namespace
}

// isTaskSuccessfulFromEventMessage parses success status from TaskCompleted event message
func (r *LearningReconciler) isTaskSuccessfulFromEventMessage(message string) bool {
	// Expected format: "Task 'task_name' completed successfully in 1234ms (task_type: neural)"
	return strings.Contains(message, "completed successfully")
}

// isEventProcessed checks if we've already processed this Event
func (r *LearningReconciler) isEventProcessed(ctx context.Context, agent *langopv1alpha1.LanguageAgent, event *corev1.Event) bool {
	// Check if event has our processing annotation
	if event.Annotations != nil {
		if _, exists := event.Annotations["langop.io/learning-processed"]; exists {
			return true
		}
	}
	return false
}

// markEventAsProcessed marks an Event as processed to avoid duplicate processing
func (r *LearningReconciler) markEventAsProcessed(ctx context.Context, agent *langopv1alpha1.LanguageAgent, event *corev1.Event) error {
	if event.Annotations == nil {
		event.Annotations = make(map[string]string)
	}
	event.Annotations["langop.io/learning-processed"] = time.Now().Format(time.RFC3339)

	return r.Update(ctx, event)
}

// createSynthesizerForAgent creates a synthesizer from the agent's model for optimization
func (r *LearningReconciler) createSynthesizerForAgent(ctx context.Context, agent *langopv1alpha1.LanguageAgent) (synthesis.AgentSynthesizer, error) {
	// Get the primary model from the agent
	if len(agent.Spec.ModelRefs) == 0 {
		return nil, fmt.Errorf("agent has no model references")
	}

	var primaryModel *langopv1alpha1.LanguageModel
	for _, modelRef := range agent.Spec.ModelRefs {
		if modelRef.Role == "primary" {
			model := &langopv1alpha1.LanguageModel{}
			modelNamespace := agent.Namespace
			if modelRef.Namespace != "" {
				modelNamespace = modelRef.Namespace
			}

			err := r.Get(ctx, types.NamespacedName{
				Name:      modelRef.Name,
				Namespace: modelNamespace,
			}, model)
			if err != nil {
				return nil, fmt.Errorf("failed to get model %s: %w", modelRef.Name, err)
			}
			primaryModel = model
			break
		}
	}

	if primaryModel == nil {
		// Fallback to first model if no primary role found
		model := &langopv1alpha1.LanguageModel{}
		modelNamespace := agent.Namespace
		if agent.Spec.ModelRefs[0].Namespace != "" {
			modelNamespace = agent.Spec.ModelRefs[0].Namespace
		}

		err := r.Get(ctx, types.NamespacedName{
			Name:      agent.Spec.ModelRefs[0].Name,
			Namespace: modelNamespace,
		}, model)
		if err != nil {
			return nil, fmt.Errorf("failed to get fallback model %s: %w", agent.Spec.ModelRefs[0].Name, err)
		}
		primaryModel = model
	}

	// Create synthesizer from the model
	synthesizer, err := synthesis.NewSynthesizerFromLanguageModel(ctx, r.Client, primaryModel, r.Log.WithName("learning-synthesis"))
	if err != nil {
		return nil, fmt.Errorf("failed to create synthesizer: %w", err)
	}

	return synthesizer, nil
}

// identifyTasksForOptimization analyzes agent telemetry to identify tasks that could benefit from optimization
func (r *LearningReconciler) identifyTasksForOptimization(ctx context.Context, agent *langopv1alpha1.LanguageAgent) ([]synthesis.TaskSynthesisRequest, error) {
	log := r.Log.WithValues("agent", agent.Name, "namespace", agent.Namespace)
	log.Info("Identifying tasks for optimization")

	// Query learning status for tasks with sufficient data
	learningStatus, err := r.getLearningStatus(ctx, agent)
	if err != nil {
		return nil, fmt.Errorf("failed to get learning status: %w", err)
	}

	if len(learningStatus) == 0 {
		log.Info("No task learning status found, no tasks to optimize")
		return []synthesis.TaskSynthesisRequest{}, nil
	}

	var tasks []synthesis.TaskSynthesisRequest

	// Analyze each task to determine if it's ready for optimization
	for taskName, status := range learningStatus {
		log.V(1).Info("Evaluating task for optimization",
			"task", taskName,
			"traces", status.TraceCount,
			"confidence", status.PatternConfidence,
			"symbolic", status.IsSymbolic)

		// Skip if already symbolic (already optimized)
		if status.IsSymbolic {
			log.V(1).Info("Skipping task - already symbolic", "task", taskName)
			continue
		}

		// Skip if not enough traces
		if status.TraceCount < r.LearningThreshold {
			log.V(1).Info("Skipping task - insufficient traces",
				"task", taskName,
				"traces", status.TraceCount,
				"threshold", r.LearningThreshold)
			continue
		}

		// Skip if already optimized recently
		if !status.LastLearningAttempt.IsZero() && time.Since(status.LastLearningAttempt) < r.LearningInterval {
			log.V(1).Info("Skipping task - recently optimized",
				"task", taskName,
				"last_attempt", status.LastLearningAttempt,
				"interval", r.LearningInterval)
			continue
		}

		// Skip if pattern confidence too low
		if status.PatternConfidence < r.PatternConfidenceMin {
			log.V(1).Info("Skipping task - low confidence",
				"task", taskName,
				"confidence", status.PatternConfidence,
				"threshold", r.PatternConfidenceMin)
			continue
		}

		log.Info("Task eligible for optimization",
			"task", taskName,
			"traces", status.TraceCount,
			"confidence", status.PatternConfidence)

		// Get execution traces for this task
		allTraces, err := r.getExecutionTraces(ctx, agent)
		if err != nil {
			log.Error(err, "Failed to get execution traces", "task", taskName)
			continue
		}

		// Group traces by task
		tracesByTask := r.groupTracesByTask(allTraces)
		taskTraces, exists := tracesByTask[taskName]
		if !exists || len(taskTraces) == 0 {
			log.V(1).Info("No traces found for task", "task", taskName)
			continue
		}

		// Format traces for synthesis
		formattedTraces := r.formatTracesForSynthesis(taskTraces)

		// Extract tools used from traces
		toolsList := r.extractToolsFromTraces(taskTraces)

		// Build synthesis request from actual data
		tasks = append(tasks, synthesis.TaskSynthesisRequest{
			TaskName:           taskName,
			Instructions:       fmt.Sprintf("Optimize task %s based on observed execution patterns", taskName),
			Inputs:             `{}`, // TODO: Extract from agent code
			Outputs:            `{}`, // TODO: Extract from agent code
			TaskCode:           "",   // TODO: Extract from agent code
			Traces:             formattedTraces,
			TraceCount:         int(status.TraceCount),
			CommonPattern:      status.CommonPattern,
			ConsistencyScore:   int(status.PatternConfidence * 100),
			UniquePatternCount: int(status.UniquePatternCount),
			ToolsList:          toolsList,
			AgentName:          agent.Name,
			Namespace:          agent.Namespace,
		})
	}

	log.Info("Identified tasks for optimization", "count", len(tasks))
	return tasks, nil
}

// formatTracesForSynthesis converts task traces to a synthesis-friendly format
func (r *LearningReconciler) formatTracesForSynthesis(traces []TaskTrace) string {
	if len(traces) == 0 {
		return "No execution traces available"
	}

	var formatted strings.Builder
	formatted.WriteString(fmt.Sprintf("Analyzed %d execution traces:\n", len(traces)))

	// Summarize top 5 traces
	maxTraces := 5
	if len(traces) < maxTraces {
		maxTraces = len(traces)
	}

	for i := 0; i < maxTraces; i++ {
		trace := traces[i]
		formatted.WriteString(fmt.Sprintf("\nTrace %d:\n", i+1))
		formatted.WriteString(fmt.Sprintf("  Duration: %s\n", trace.Duration))
		formatted.WriteString(fmt.Sprintf("  Success: %v\n", trace.Success))
		formatted.WriteString(fmt.Sprintf("  Tool calls: %d\n", len(trace.ToolCalls)))

		if len(trace.ToolCalls) > 0 {
			formatted.WriteString("  Tools used: ")
			toolNames := make([]string, len(trace.ToolCalls))
			for j, tc := range trace.ToolCalls {
				toolNames[j] = fmt.Sprintf("%s.%s", tc.ToolName, tc.Method)
			}
			formatted.WriteString(strings.Join(toolNames, ", "))
			formatted.WriteString("\n")
		}
	}

	return formatted.String()
}

// extractToolsFromTraces extracts unique tool names from traces
func (r *LearningReconciler) extractToolsFromTraces(traces []TaskTrace) string {
	toolMap := make(map[string]bool)

	for _, trace := range traces {
		for _, toolCall := range trace.ToolCalls {
			toolMap[toolCall.ToolName] = true
		}
	}

	if len(toolMap) == 0 {
		return "No tools detected"
	}

	tools := make([]string, 0, len(toolMap))
	for tool := range toolMap {
		tools = append(tools, tool)
	}

	sort.Strings(tools)
	return strings.Join(tools, ", ")
}

// applyOptimizedTasks merges optimized task implementations into the agent's DSL and creates a versioned ConfigMap
func (r *LearningReconciler) applyOptimizedTasks(ctx context.Context, agent *langopv1alpha1.LanguageAgent, optimizedTasks map[string]OptimizedTask, synthesizer synthesis.AgentSynthesizer) error {
	log := r.Log.WithValues("agent", agent.Name, "namespace", agent.Namespace)

	// Get the current agent code from the existing ConfigMap
	currentCode, err := r.getCurrentAgentCode(ctx, agent)
	if err != nil {
		return fmt.Errorf("failed to get current agent code: %w", err)
	}

	// Merge optimized tasks into the current agent code
	optimizedCode, err := r.mergeOptimizedTasks(currentCode, optimizedTasks)
	if err != nil {
		return fmt.Errorf("failed to merge optimized tasks: %w", err)
	}

	// Create versioned ConfigMap with optimized code
	if r.ConfigMapManager != nil {
		// Get the latest version and increment it
		latestVersion, err := r.ConfigMapManager.GetLatestVersion(ctx, agent)
		if err != nil {
			log.Error(err, "Failed to get latest ConfigMap version")
			return fmt.Errorf("failed to get latest version: %w", err)
		}

		newVersion := latestVersion + 1
		if newVersion <= 0 {
			newVersion = 1 // Ensure version is always positive
		}

		options := &synthesis.ConfigMapOptions{
			Code:            optimizedCode,
			Version:         newVersion,
			SynthesisType:   "learned",
			LearningSource:  "task-level-optimization",
			PreviousVersion: &latestVersion,
		}

		if _, err := r.ConfigMapManager.CreateVersionedConfigMap(ctx, agent, options); err != nil {
			log.Error(err, "Failed to create optimized code ConfigMap")
			return fmt.Errorf("failed to create optimized code: %w", err)
		}

		log.Info("Created versioned ConfigMap for optimized agent code",
			"version", newVersion,
			"previousVersion", latestVersion,
			"codeLength", len(optimizedCode),
			"optimizedTasks", len(optimizedTasks))
	}

	return nil
}

// getCurrentAgentCode retrieves the current agent DSL code from the active LanguageAgentVersion
func (r *LearningReconciler) getCurrentAgentCode(ctx context.Context, agent *langopv1alpha1.LanguageAgent) (string, error) {
	// Step 1: Find the current active LanguageAgentVersion
	var activeVersion *langopv1alpha1.LanguageAgentVersion
	var err error

	// Check if agent has an explicit version reference
	if agent.Spec.AgentVersionRef != nil && agent.Spec.AgentVersionRef.Name != "" {
		// Use the explicitly referenced version
		activeVersion = &langopv1alpha1.LanguageAgentVersion{}
		err = r.Get(ctx, types.NamespacedName{
			Name:      agent.Spec.AgentVersionRef.Name,
			Namespace: agent.Namespace,
		}, activeVersion)

		if err != nil {
			if apierrors.IsNotFound(err) {
				r.Log.Info("Referenced AgentVersion not found, falling back to latest",
					"agent", agent.Name,
					"referencedVersion", agent.Spec.AgentVersionRef.Name)
			} else {
				return "", fmt.Errorf("failed to get referenced LanguageAgentVersion %s: %w",
					agent.Spec.AgentVersionRef.Name, err)
			}
		}
	}

	// If we don't have an active version yet, find the latest one
	if activeVersion == nil {
		activeVersion, err = r.getLatestAgentVersion(ctx, agent)
		if err != nil {
			return "", fmt.Errorf("failed to get latest LanguageAgentVersion: %w", err)
		}
		if activeVersion == nil {
			// No versions exist yet - return empty code
			r.Log.Info("No LanguageAgentVersion found for agent", "agent", agent.Name)
			return "", nil
		}
	}

	// Step 2: Get the agent code from the version
	if activeVersion.Spec.Code != "" {
		// Code is stored directly in the LanguageAgentVersion spec
		return activeVersion.Spec.Code, nil
	}

	// Step 3: Fall back to ConfigMap if code is not in spec
	configMapName := activeVersion.Status.ConfigMapName
	if configMapName == "" {
		return "", fmt.Errorf("LanguageAgentVersion %s has no code in spec and no ConfigMap name in status", activeVersion.Name)
	}

	configMap := &corev1.ConfigMap{}
	err = r.Get(ctx, types.NamespacedName{
		Name:      configMapName,
		Namespace: agent.Namespace,
	}, configMap)

	if err != nil {
		if apierrors.IsNotFound(err) {
			return "", fmt.Errorf("ConfigMap %s referenced by LanguageAgentVersion %s not found",
				configMapName, activeVersion.Name)
		}
		return "", fmt.Errorf("failed to get ConfigMap %s: %w", configMapName, err)
	}

	// Look for the agent code in the ConfigMap data
	if code, exists := configMap.Data["agent.rb"]; exists {
		return code, nil
	}

	return "", fmt.Errorf("no agent code found in ConfigMap %s", configMapName)
}

// getLatestAgentVersion finds the latest LanguageAgentVersion for the given agent
func (r *LearningReconciler) getLatestAgentVersion(ctx context.Context, agent *langopv1alpha1.LanguageAgent) (*langopv1alpha1.LanguageAgentVersion, error) {
	// List all LanguageAgentVersion resources for this agent
	versionList := &langopv1alpha1.LanguageAgentVersionList{}
	listOpts := []client.ListOption{
		client.InNamespace(agent.Namespace),
		client.MatchingLabels{"langop.io/agent": agent.Name},
	}

	if err := r.List(ctx, versionList, listOpts...); err != nil {
		return nil, fmt.Errorf("failed to list LanguageAgentVersions: %w", err)
	}

	if len(versionList.Items) == 0 {
		return nil, nil
	}

	// Find the version with the highest version number
	var latestVersion *langopv1alpha1.LanguageAgentVersion
	var maxVersion int32 = -1

	for i := range versionList.Items {
		version := &versionList.Items[i]
		if version.Spec.Version > maxVersion {
			maxVersion = version.Spec.Version
			latestVersion = version
		}
	}

	return latestVersion, nil
}

// mergeOptimizedTasks merges optimized task implementations into the current agent DSL code
// using Ruby AST parsing for proper code reconstruction
func (r *LearningReconciler) mergeOptimizedTasks(currentCode string, optimizedTasks map[string]OptimizedTask) (string, error) {
	if len(optimizedTasks) == 0 {
		return currentCode, nil
	}

	// Use Ruby AST-based reconstruction for proper code merging
	reconstructedCode, err := r.reconstructAgentCodeWithRuby(currentCode, optimizedTasks)
	if err != nil {
		r.Log.Error(err, "Ruby AST reconstruction failed, falling back to simple concatenation")
		// Fallback to simple concatenation if AST reconstruction fails
		return r.fallbackMergeOptimizedTasks(currentCode, optimizedTasks), nil
	}

	r.Log.Info("Successfully reconstructed agent code using Ruby AST",
		"originalCodeLength", len(currentCode),
		"reconstructedCodeLength", len(reconstructedCode),
		"optimizedTaskCount", len(optimizedTasks))

	return reconstructedCode, nil
}

// reconstructAgentCodeWithRuby uses Ruby script for AST-based code reconstruction
func (r *LearningReconciler) reconstructAgentCodeWithRuby(originalCode string, optimizedTasks map[string]OptimizedTask) (string, error) {
	// Prepare input JSON for Ruby script
	input := map[string]interface{}{
		"original_code":   originalCode,
		"optimized_tasks": optimizedTasks,
	}

	inputJSON, err := json.Marshal(input)
	if err != nil {
		return "", fmt.Errorf("failed to marshal input JSON: %w", err)
	}

	// Find the reconstruction script
	scriptPath := r.findReconstructionScript()

	// Execute Ruby script with timeout
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	cmd := exec.CommandContext(ctx, "ruby", scriptPath)
	cmd.Stdin = strings.NewReader(string(inputJSON))

	output, err := cmd.Output()
	if err != nil {
		if ctx.Err() == context.DeadlineExceeded {
			return "", fmt.Errorf("reconstruction timeout: code too complex (>10s)")
		}
		// Try to get stderr for better error reporting
		if exitErr, ok := err.(*exec.ExitError); ok {
			return "", fmt.Errorf("reconstruction script failed: %s", string(exitErr.Stderr))
		}
		return "", fmt.Errorf("reconstruction script execution failed: %w", err)
	}

	reconstructedCode := strings.TrimSpace(string(output))
	if reconstructedCode == "" {
		return "", fmt.Errorf("reconstruction script produced empty output")
	}

	// Validate Ruby syntax before returning
	if err := r.validateRubySyntax(reconstructedCode); err != nil {
		r.Log.Error(err, "Generated code has Ruby syntax errors, falling back")
		return "", fmt.Errorf("generated code has syntax errors: %w", err)
	}

	return reconstructedCode, nil
}

// findReconstructionScript locates the Ruby reconstruction script
func (r *LearningReconciler) findReconstructionScript() string {
	locations := []string{
		"/usr/local/bin/reconstruct-agent-code.rb",                              // Docker container
		"scripts/reconstruct-agent-code.rb",                                     // CI from src/ directory
		"../../../scripts/reconstruct-agent-code.rb",                            // Test from src/pkg/validation
		filepath.Join("..", "..", "..", "scripts", "reconstruct-agent-code.rb"), // Alternative path
		"../../scripts/reconstruct-agent-code.rb",                               // From src subdirectory
		"../scripts/reconstruct-agent-code.rb",                                  // From pkg/validation
	}

	for _, path := range locations {
		if absPath, err := filepath.Abs(path); err == nil {
			if _, err := os.Stat(absPath); err == nil {
				return absPath
			}
		}
	}

	// Default to container location
	return "/usr/local/bin/reconstruct-agent-code.rb"
}

// validateRubySyntax checks if the given Ruby code has valid syntax using ruby -c
func (r *LearningReconciler) validateRubySyntax(rubyCode string) error {
	// Skip validation if Ruby is not available
	if _, err := exec.LookPath("ruby"); err != nil {
		r.Log.Info("Ruby not available, skipping syntax validation")
		return nil
	}

	// Create a context with timeout for syntax checking
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Use ruby -c to check syntax
	cmd := exec.CommandContext(ctx, "ruby", "-c")
	cmd.Stdin = strings.NewReader(rubyCode)

	// Capture stderr which contains syntax error messages
	var stderr bytes.Buffer
	cmd.Stderr = &stderr

	err := cmd.Run()
	if err != nil {
		if ctx.Err() == context.DeadlineExceeded {
			return fmt.Errorf("syntax validation timeout (>5s)")
		}

		// Extract syntax error message from stderr
		errorMsg := strings.TrimSpace(stderr.String())
		if errorMsg != "" {
			return fmt.Errorf("Ruby syntax error: %s", errorMsg)
		}

		return fmt.Errorf("Ruby syntax validation failed: %w", err)
	}

	return nil
}

// fallbackMergeOptimizedTasks provides fallback behavior if Ruby reconstruction fails
// It replaces existing task definitions with optimized versions
func (r *LearningReconciler) fallbackMergeOptimizedTasks(currentCode string, optimizedTasks map[string]OptimizedTask) string {
	mergedCode := currentCode

	for taskName, optimized := range optimizedTasks {
		// Try to replace existing task definition first
		replacedCode := r.replaceTaskDefinition(mergedCode, taskName, optimized)

		if replacedCode != mergedCode {
			// Successfully replaced existing task
			mergedCode = replacedCode
			r.Log.Info("Replaced existing task definition (fallback mode)",
				"taskName", taskName,
				"optimizedCodeLength", len(optimized.Code),
				"hasInputsSchema", optimized.Inputs != "",
				"hasOutputsSchema", optimized.Outputs != "")
		} else {
			// Task not found, add as new task
			taskDef := r.buildTaskDefinition(taskName, optimized)
			mergedCode = r.insertNewTask(mergedCode, taskName, taskDef)
			r.Log.Info("Added new optimized task (fallback mode)",
				"taskName", taskName,
				"optimizedCodeLength", len(optimized.Code),
				"hasInputsSchema", optimized.Inputs != "",
				"hasOutputsSchema", optimized.Outputs != "")
		}
	}

	// Validate final syntax
	if err := r.validateRubySyntax(mergedCode); err != nil {
		r.Log.Error(err, "Fallback code generation produced syntax errors",
			"codeLength", len(mergedCode))
		// Return original code if fallback also fails syntax validation
		return currentCode
	}

	return mergedCode
}

// replaceTaskDefinition attempts to replace an existing task definition with an optimized version
func (r *LearningReconciler) replaceTaskDefinition(code, taskName string, optimized OptimizedTask) string {
	// Look for neural task pattern: task(:taskname, ...)
	neuralStart := fmt.Sprintf("task(:%s,", taskName)
	if strings.Contains(code, neuralStart) {
		return r.replaceNeuralTask(code, taskName, optimized)
	}

	// Look for symbolic task pattern: task :taskname do
	symbolicStart := fmt.Sprintf("task :%s do", taskName)
	if strings.Contains(code, symbolicStart) {
		return r.replaceSymbolicTask(code, taskName, optimized)
	}

	// No match found - return unchanged
	return code
}

// replaceNeuralTask replaces a neural task (with parentheses syntax)
func (r *LearningReconciler) replaceNeuralTask(code, taskName string, optimized OptimizedTask) string {
	lines := strings.Split(code, "\n")
	result := []string{}
	i := 0

	for i < len(lines) {
		line := lines[i]

		// Look for the start of our target task
		if strings.Contains(line, fmt.Sprintf("task(:%s,", taskName)) {
			// Extract indentation from this line
			indentation := ""
			for _, char := range line {
				if char == ' ' || char == '\t' {
					indentation += string(char)
				} else {
					break
				}
			}

			// Skip lines until we find the line with outputs (neural tasks don't have a do block)
			// Neural tasks are single statements ending with )
			for i < len(lines) && !strings.Contains(lines[i], "outputs:") {
				i++
			}
			// Skip one more line to get past the outputs line
			if i < len(lines) {
				i++
			}

			// Generate replacement task
			replacement := r.buildTaskDefinitionWithIndentation(taskName, optimized, indentation)
			result = append(result, replacement)
		} else {
			result = append(result, line)
		}
		i++
	}

	return strings.Join(result, "\n")
}

// replaceSymbolicTask replaces a symbolic task (with do..end syntax)
func (r *LearningReconciler) replaceSymbolicTask(code, taskName string, optimized OptimizedTask) string {
	lines := strings.Split(code, "\n")
	result := []string{}
	i := 0

	for i < len(lines) {
		line := lines[i]

		// Look for the start of our target task
		if strings.Contains(line, fmt.Sprintf("task :%s do", taskName)) {
			// Extract indentation from this line
			indentation := ""
			for _, char := range line {
				if char == ' ' || char == '\t' {
					indentation += string(char)
				} else {
					break
				}
			}

			// Skip lines until we find the matching end
			depth := 1
			i++ // Move past the task line
			for i < len(lines) && depth > 0 {
				if strings.Contains(strings.TrimSpace(lines[i]), " do") {
					depth++
				} else if strings.TrimSpace(lines[i]) == "end" {
					depth--
				}
				i++
			}

			// Generate replacement task
			replacement := r.buildTaskDefinitionWithIndentation(taskName, optimized, indentation)
			result = append(result, replacement)
		} else {
			result = append(result, line)
		}
		i++
	}

	return strings.Join(result, "\n")
}

// buildTaskDefinition creates the Ruby task definition string
func (r *LearningReconciler) buildTaskDefinition(taskName string, optimized OptimizedTask) string {
	return r.buildTaskDefinitionWithIndentation(taskName, optimized, "  ")
}

// buildTaskDefinitionWithIndentation creates the Ruby task definition with proper indentation
func (r *LearningReconciler) buildTaskDefinitionWithIndentation(taskName string, optimized OptimizedTask, indentation string) string {
	if optimized.Inputs != "" && optimized.Outputs != "" {
		// Convert JSON schemas to Ruby hash syntax with proper quoting
		inputsRuby := r.convertSchemaToRuby(optimized.Inputs)
		outputsRuby := r.convertSchemaToRuby(optimized.Outputs)

		// Format with schema for neural tasks
		return fmt.Sprintf("%stask(:%s,\n%s       inputs: { %s },\n%s       outputs: { %s }) do |inputs|\n%s\n%send",
			indentation, taskName, indentation, inputsRuby, indentation, outputsRuby,
			r.indentCode(optimized.Code, indentation+"  "), indentation)
	} else {
		// Format without schema for symbolic tasks
		return fmt.Sprintf("%stask :%s do |inputs|\n%s\n%send",
			indentation, taskName, r.indentCode(optimized.Code, indentation+"  "), indentation)
	}
}

// convertSchemaToRuby converts JSON schema to Ruby hash syntax with proper type quoting
func (r *LearningReconciler) convertSchemaToRuby(jsonSchema string) string {
	trimmed := strings.TrimSpace(jsonSchema)
	if jsonSchema == "" || trimmed == "{}" {
		return ""
	}

	// Parse JSON to get the hash structure
	var schemaMap map[string]interface{}
	if err := json.Unmarshal([]byte(jsonSchema), &schemaMap); err != nil {
		// If JSON parsing fails, try to handle simple key:value pairs manually
		content := strings.Trim(jsonSchema, "{}")
		content = strings.TrimSpace(content)
		if content == "" {
			return ""
		}

		// Split by commas and convert each pair
		pairs := strings.Split(content, ",")
		var rubyPairs []string
		for _, pair := range pairs {
			pair = strings.TrimSpace(pair)
			// Match "key": "value" or key: value patterns
			if parts := strings.SplitN(pair, ":", 2); len(parts) == 2 {
				key := strings.Trim(strings.TrimSpace(parts[0]), `"`)
				value := strings.Trim(strings.TrimSpace(parts[1]), `"`)
				rubyPairs = append(rubyPairs, fmt.Sprintf(`%s: "%s"`, key, value))
			}
		}
		return strings.Join(rubyPairs, ", ")
	}

	// Convert parsed map to Ruby hash syntax with quoted type values
	var rubyPairs []string
	for key, value := range schemaMap {
		var quotedValue string
		if strValue, ok := value.(string); ok {
			quotedValue = fmt.Sprintf(`"%s"`, strValue)
		} else {
			quotedValue = fmt.Sprintf("%v", value)
		}
		rubyPairs = append(rubyPairs, fmt.Sprintf(`%s: %s`, key, quotedValue))
	}

	return strings.Join(rubyPairs, ", ")
}

// indentCode adds indentation to each line of code
func (r *LearningReconciler) indentCode(code, indentation string) string {
	lines := strings.Split(strings.TrimSpace(code), "\n")
	indentedLines := make([]string, len(lines))
	for i, line := range lines {
		if strings.TrimSpace(line) == "" {
			indentedLines[i] = "" // Keep empty lines empty
		} else {
			indentedLines[i] = indentation + line
		}
	}
	return strings.Join(indentedLines, "\n")
}

// insertNewTask inserts a new task definition before the main block or final end
func (r *LearningReconciler) insertNewTask(code, taskName, taskDef string) string {
	// Try to insert before main block
	mainPattern := `(\n\s*)(main\s+do)`
	if strings.Contains(code, "main do") {
		replacement := fmt.Sprintf("$1\n  # Optimized task: %s\n%s\n$1$2", taskName, taskDef)
		return r.replaceWithPattern(code, mainPattern, replacement)
	}

	// Try to insert before final end
	finalEndPattern := `(\n\s*)(end\s*$)`
	if strings.HasSuffix(strings.TrimSpace(code), "end") {
		replacement := fmt.Sprintf("$1\n  # Optimized task: %s\n%s\n$1$2", taskName, taskDef)
		return r.replaceWithPattern(code, finalEndPattern, replacement)
	}

	// Fallback: append at end
	return fmt.Sprintf("%s\n\n  # Optimized task: %s\n%s", code, taskName, taskDef)
}

// matchAndExtractIndentation checks if a pattern matches and extracts indentation
func (r *LearningReconciler) matchAndExtractIndentation(code, pattern string) (bool, string) {
	re, err := regexp.Compile(pattern)
	if err != nil {
		r.Log.Error(err, "Invalid regex pattern", "pattern", pattern)
		return false, ""
	}

	matches := re.FindStringSubmatch(code)
	if len(matches) < 2 {
		return false, ""
	}

	// First captured group should be the indentation
	indentation := matches[1]
	return true, indentation
}

// replaceWithPattern replaces text using a regex pattern
func (r *LearningReconciler) replaceWithPattern(code, pattern, replacement string) string {
	re, err := regexp.Compile(pattern)
	if err != nil {
		r.Log.Error(err, "Invalid regex pattern", "pattern", pattern)
		return code
	}

	return re.ReplaceAllString(code, replacement)
}

// setAgentVersionRef updates the agent's versionRef field to point to an optimized ConfigMap
func (r *LearningReconciler) setAgentVersionRef(ctx context.Context, agent *langopv1alpha1.LanguageAgent, configMapName string) error {
	log := r.Log.WithValues("agent", agent.Name, "versionRef", configMapName)

	// DEPRECATED: This method is kept for compatibility but no longer functional
	log.Info("setAgentVersionRef is deprecated, use LanguageAgentVersion resources instead")
	return nil
}

// createAgentVersion creates a new LanguageAgentVersion resource with optimized code
func (r *LearningReconciler) createAgentVersion(ctx context.Context, agent *langopv1alpha1.LanguageAgent, optimizedTasks map[string]OptimizedTask, synthesizer synthesis.AgentSynthesizer) error {
	log := r.Log.WithValues("agent", agent.Name, "namespace", agent.Namespace)

	// Get the current agent code from the existing ConfigMap
	currentCode, err := r.getCurrentAgentCode(ctx, agent)
	if err != nil {
		return fmt.Errorf("failed to get current agent code: %w", err)
	}

	// Merge optimized tasks into the current agent code
	optimizedCode, err := r.mergeOptimizedTasks(currentCode, optimizedTasks)
	if err != nil {
		return fmt.Errorf("failed to merge optimized tasks: %w", err)
	}

	// Get next version number
	nextVersion, err := r.getNextVersionNumber(ctx, agent)
	if err != nil {
		return fmt.Errorf("failed to get next version number: %w", err)
	}

	// Build optimized tasks metadata
	optimizedTasksMetadata := make(map[string]langopv1alpha1.OptimizedTask)
	for taskName := range optimizedTasks {
		optimizedTasksMetadata[taskName] = langopv1alpha1.OptimizedTask{
			Name:               taskName,
			ConfidenceScore:    95, // Default confidence score for learning-optimized tasks
			OptimizationReason: "Task optimized by learning system based on execution patterns",
		}
	}

	// Create LanguageAgentVersion resource
	agentVersion := &langopv1alpha1.LanguageAgentVersion{
		ObjectMeta: metav1.ObjectMeta{
			Name:      fmt.Sprintf("%s-v%d", agent.Name, nextVersion),
			Namespace: agent.Namespace,
			Labels: map[string]string{
				"langop.io/agent":      agent.Name,
				"langop.io/source":     "learning",
				"langop.io/managed-by": "language-operator",
			},
			Annotations: map[string]string{
				"langop.io/generated-by": "learning-controller",
			},
		},
		Spec: langopv1alpha1.LanguageAgentVersionSpec{
			AgentRef: langopv1alpha1.AgentReference{
				Name:      agent.Name,
				Namespace: agent.Namespace,
			},
			Version:        nextVersion,
			Code:           optimizedCode,
			OptimizedTasks: optimizedTasksMetadata,
			SourceType:     "learning",
			Description:    fmt.Sprintf("Optimized version with %d improved tasks", len(optimizedTasks)),
			CreatedBy:      "learning-system",
		},
	}

	// Set owner reference so LanguageAgentVersion is cleaned up when LanguageAgent is deleted
	if err := controllerutil.SetControllerReference(agent, agentVersion, r.Scheme); err != nil {
		return fmt.Errorf("failed to set controller reference: %w", err)
	}

	// Create the LanguageAgentVersion
	if err := r.Create(ctx, agentVersion); err != nil {
		return fmt.Errorf("failed to create LanguageAgentVersion: %w", err)
	}

	log.Info("Created LanguageAgentVersion",
		"versionName", agentVersion.Name,
		"version", nextVersion,
		"optimizedTasks", len(optimizedTasks))

	// Update agent to reference the new version with retry for conflicts
	if err := r.setAgentVersionReferenceWithRetry(ctx, agent, agentVersion.Name); err != nil {
		log.Error(err, "Failed to update agent's AgentVersionRef after retries",
			"versionName", agentVersion.Name)
		return fmt.Errorf("failed to update agent to use LanguageAgentVersion %s: %w", agentVersion.Name, err)
	}

	log.Info("Successfully updated agent to use new LanguageAgentVersion",
		"agent", agent.Name,
		"versionName", agentVersion.Name,
		"version", nextVersion)

	return nil
}

// getNextVersionNumber determines the next version number for an agent
func (r *LearningReconciler) getNextVersionNumber(ctx context.Context, agent *langopv1alpha1.LanguageAgent) (int32, error) {
	// List all LanguageAgentVersion resources for this agent
	versionList := &langopv1alpha1.LanguageAgentVersionList{}
	listOpts := []client.ListOption{
		client.InNamespace(agent.Namespace),
		client.MatchingLabels{"langop.io/agent": agent.Name},
	}

	if err := r.List(ctx, versionList, listOpts...); err != nil {
		return 0, fmt.Errorf("failed to list existing versions: %w", err)
	}

	// Find highest version number
	var maxVersion int32 = 0
	for _, version := range versionList.Items {
		if version.Spec.Version > maxVersion {
			maxVersion = version.Spec.Version
		}
	}

	return maxVersion + 1, nil
}

// setAgentVersionReference updates the agent's AgentVersionRef field
func (r *LearningReconciler) setAgentVersionReference(ctx context.Context, agent *langopv1alpha1.LanguageAgent, versionName string) error {
	log := r.Log.WithValues("agent", agent.Name, "versionName", versionName)

	// Check if the version reference is locked
	if agent.Spec.AgentVersionRef != nil && agent.Spec.AgentVersionRef.Lock {
		log.Info("Agent version reference is locked, skipping automatic update",
			"lockedVersion", agent.Spec.AgentVersionRef.Name)
		return nil
	}

	// Update the agent's AgentVersionRef
	agent.Spec.AgentVersionRef = &langopv1alpha1.AgentVersionReference{
		Name:      versionName,
		Namespace: agent.Namespace,
		Lock:      false, // Default to unlocked when learning creates new versions
	}

	if err := r.Update(ctx, agent); err != nil {
		log.Error(err, "Failed to update agent AgentVersionRef")
		return fmt.Errorf("failed to update agent AgentVersionRef: %w", err)
	}

	log.Info("Successfully updated agent AgentVersionRef")
	return nil
}

// setAgentVersionReferenceWithRetry handles race conditions when updating agent version reference
func (r *LearningReconciler) setAgentVersionReferenceWithRetry(ctx context.Context, agent *langopv1alpha1.LanguageAgent, versionName string) error {
	log := r.Log.WithValues("agent", agent.Name, "versionName", versionName)

	maxRetries := 5
	for attempt := 0; attempt < maxRetries; attempt++ {
		// Fetch fresh agent resource to avoid conflicts
		freshAgent := &langopv1alpha1.LanguageAgent{}
		if err := r.Get(ctx, types.NamespacedName{Name: agent.Name, Namespace: agent.Namespace}, freshAgent); err != nil {
			log.Error(err, "Failed to get fresh agent for version reference update", "attempt", attempt)
			return fmt.Errorf("failed to get fresh agent: %w", err)
		}

		// Check if the version reference is locked
		if freshAgent.Spec.AgentVersionRef != nil && freshAgent.Spec.AgentVersionRef.Lock {
			log.Info("Agent version reference is locked, skipping automatic update",
				"lockedVersion", freshAgent.Spec.AgentVersionRef.Name)
			return nil
		}

		// Update the agent's AgentVersionRef on the fresh object
		freshAgent.Spec.AgentVersionRef = &langopv1alpha1.AgentVersionReference{
			Name:      versionName,
			Namespace: agent.Namespace,
			Lock:      false, // Default to unlocked when learning creates new versions
		}

		if err := r.Update(ctx, freshAgent); err != nil {
			if apierrors.IsConflict(err) && attempt < maxRetries-1 {
				log.V(1).Info("Resource conflict updating agent version reference, retrying",
					"attempt", attempt+1, "maxRetries", maxRetries)
				// Brief sleep to avoid tight retry loop
				time.Sleep(time.Millisecond * time.Duration(100*(attempt+1)))
				continue
			}
			log.Error(err, "Failed to update agent AgentVersionRef", "attempt", attempt)
			return fmt.Errorf("failed to update agent AgentVersionRef after %d attempts: %w", attempt+1, err)
		}

		log.Info("Successfully updated agent AgentVersionRef", "attempts", attempt+1)
		return nil
	}

	return fmt.Errorf("failed to update agent AgentVersionRef after %d attempts", maxRetries)
}
