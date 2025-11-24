package v1alpha1

import (
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// LanguageAgentSpec defines the desired state of LanguageAgent
type LanguageAgentSpec struct {
	// ClusterRef references a LanguageCluster to deploy this agent into
	// +optional
	ClusterRef string `json:"clusterRef,omitempty"`

	// Image is the container image to run for this agent
	// +kubebuilder:validation:Required
	// +kubebuilder:validation:MinLength=1
	Image string `json:"image"`

	// ImagePullPolicy defines when to pull the container image
	// +kubebuilder:validation:Enum=Always;Never;IfNotPresent
	// +kubebuilder:default=IfNotPresent
	// +optional
	ImagePullPolicy corev1.PullPolicy `json:"imagePullPolicy,omitempty"`

	// ImagePullSecrets is a list of references to secrets for pulling images
	// +optional
	ImagePullSecrets []corev1.LocalObjectReference `json:"imagePullSecrets,omitempty"`

	// ModelRefs is a list of LanguageModel references this agent can use
	// +kubebuilder:validation:MinItems=1
	// +kubebuilder:validation:Required
	ModelRefs []ModelReference `json:"modelRefs"`

	// ToolRefs is a list of LanguageTool references available to this agent
	// +optional
	ToolRefs []ToolReference `json:"toolRefs,omitempty"`

	// PersonaRefs is a list of LanguagePersona references that compose in order of importance
	// Personas are merged with later personas taking precedence over earlier ones
	// +optional
	PersonaRefs []PersonaReference `json:"personaRefs,omitempty"`

	// Goal defines the agent's objective (for autonomous agents)
	// +optional
	Goal string `json:"goal,omitempty"`

	// Instructions provides system instructions for the agent
	// +optional
	Instructions string `json:"instructions,omitempty"`

	// ExecutionMode defines how the agent operates
	// +kubebuilder:validation:Enum=autonomous;interactive;scheduled;event-driven
	// +kubebuilder:default=autonomous
	ExecutionMode string `json:"executionMode,omitempty"`

	// Schedule defines when the agent runs (cron format, for scheduled mode)
	// +optional
	Schedule string `json:"schedule,omitempty"`

	// EventTriggers defines events that trigger the agent (for event-driven mode)
	// +optional
	EventTriggers []EventTriggerSpec `json:"eventTriggers,omitempty"`

	// MaxIterations limits the number of reasoning/action loops
	// +kubebuilder:validation:Minimum=1
	// +kubebuilder:validation:Maximum=1000
	// +kubebuilder:default=50
	// +optional
	MaxIterations *int32 `json:"maxIterations,omitempty"`

	// Timeout is the maximum execution time (e.g., "10m", "1h")
	// +kubebuilder:validation:Pattern=`^[0-9]+(ns|us|µs|ms|s|m|h)$`
	// +kubebuilder:default="10m"
	// +optional
	Timeout string `json:"timeout,omitempty"`

	// Replicas is the number of agent instances to run
	// +kubebuilder:validation:Minimum=0
	// +kubebuilder:default=1
	// +optional
	Replicas *int32 `json:"replicas,omitempty"`

	// Env contains environment variables for the agent container
	// +optional
	Env []corev1.EnvVar `json:"env,omitempty"`

	// EnvFrom sources to populate environment variables
	// +optional
	EnvFrom []corev1.EnvFromSource `json:"envFrom,omitempty"`

	// Resources defines compute resource requirements
	// +optional
	Resources corev1.ResourceRequirements `json:"resources,omitempty"`

	// NodeSelector is a selector which must match a node's labels
	// +optional
	NodeSelector map[string]string `json:"nodeSelector,omitempty"`

	// Affinity defines pod affinity and anti-affinity rules
	// +optional
	Affinity *corev1.Affinity `json:"affinity,omitempty"`

	// Tolerations allow pods to schedule onto nodes with matching taints
	// +optional
	Tolerations []corev1.Toleration `json:"tolerations,omitempty"`

	// ServiceAccountName is the name of the ServiceAccount to use
	// +optional
	ServiceAccountName string `json:"serviceAccountName,omitempty"`

	// SecurityContext holds pod-level security attributes
	// +optional
	SecurityContext *corev1.PodSecurityContext `json:"securityContext,omitempty"`

	// VolumeMounts to mount into the agent container
	// +optional
	VolumeMounts []corev1.VolumeMount `json:"volumeMounts,omitempty"`

	// Volumes to attach to the pod
	// +optional
	Volumes []corev1.Volume `json:"volumes,omitempty"`

	// PodAnnotations are annotations to add to the Pods
	// +optional
	PodAnnotations map[string]string `json:"podAnnotations,omitempty"`

	// PodLabels are additional labels to add to the Pods
	// +optional
	PodLabels map[string]string `json:"podLabels,omitempty"`

	// RestartPolicy defines when to restart the agent
	// +kubebuilder:validation:Enum=Always;OnFailure;Never
	// +kubebuilder:default=OnFailure
	// +optional
	RestartPolicy corev1.RestartPolicy `json:"restartPolicy,omitempty"`

	// BackoffLimit specifies the number of retries before marking as Failed
	// +kubebuilder:validation:Minimum=0
	// +kubebuilder:default=3
	// +optional
	BackoffLimit *int32 `json:"backoffLimit,omitempty"`

	// MemoryStore configures conversation memory persistence
	// +optional
	MemoryStore *MemoryStoreSpec `json:"memoryStore,omitempty"`

	// Observability defines monitoring and tracing configuration
	// +optional
	Observability *AgentObservabilitySpec `json:"observability,omitempty"`

	// RateLimits defines rate limiting for this agent
	// +optional
	RateLimits *AgentRateLimitSpec `json:"rateLimits,omitempty"`

	// SafetyConfig defines safety constraints and guardrails
	// +optional
	SafetyConfig *SafetyConfigSpec `json:"safetyConfig,omitempty"`

	// Workspace defines persistent storage for the agent
	// +optional
	Workspace *WorkspaceSpec `json:"workspace,omitempty"`

	// Egress defines external network access rules for this agent
	// By default, agents can access all resources within the cluster but no external endpoints
	// +optional
	Egress []NetworkRule `json:"egress,omitempty"`
}

// ModelReference references a LanguageModel
type ModelReference struct {
	// Name is the name of the LanguageModel
	// +kubebuilder:validation:Required
	Name string `json:"name"`

	// Namespace is the namespace of the LanguageModel (defaults to same namespace)
	// +optional
	Namespace string `json:"namespace,omitempty"`

	// Role defines the purpose of this model (primary, fallback, specialized)
	// +kubebuilder:validation:Enum=primary;fallback;reasoning;tool-calling;summarization
	// +kubebuilder:default=primary
	// +optional
	Role string `json:"role,omitempty"`

	// Priority for model selection (lower is higher priority)
	// +optional
	Priority *int32 `json:"priority,omitempty"`
}

// ToolReference references a LanguageTool
type ToolReference struct {
	// Name is the name of the LanguageTool
	// +kubebuilder:validation:Required
	Name string `json:"name"`

	// Namespace is the namespace of the LanguageTool (defaults to same namespace)
	// +optional
	Namespace string `json:"namespace,omitempty"`

	// Enabled indicates if this tool is available to the agent
	// +kubebuilder:default=true
	Enabled bool `json:"enabled,omitempty"`

	// RequireApproval requires human approval before tool execution
	// +kubebuilder:default=false
	// +optional
	RequireApproval bool `json:"requireApproval,omitempty"`
}

// PersonaReference references a LanguagePersona
type PersonaReference struct {
	// Name is the name of the LanguagePersona
	// +kubebuilder:validation:Required
	Name string `json:"name"`

	// Namespace is the namespace of the LanguagePersona (defaults to same namespace)
	// +optional
	Namespace string `json:"namespace,omitempty"`
}

// EventTriggerSpec defines an event trigger
type EventTriggerSpec struct {
	// Type is the event type (webhook, kubernetes-event, message-queue)
	// +kubebuilder:validation:Enum=webhook;kubernetes-event;message-queue;schedule
	// +kubebuilder:validation:Required
	Type string `json:"type"`

	// Source identifies the event source
	// +optional
	Source string `json:"source,omitempty"`

	// Filter defines filtering criteria for events
	// +optional
	Filter map[string]string `json:"filter,omitempty"`
}

// MemoryStoreSpec configures conversation memory
type MemoryStoreSpec struct {
	// Type specifies the memory backend
	// +kubebuilder:validation:Enum=ephemeral;redis;postgres;s3
	// +kubebuilder:default=ephemeral
	Type string `json:"type,omitempty"`

	// ConnectionSecretRef references a secret with connection details
	// +optional
	ConnectionSecretRef *SecretReference `json:"connectionSecretRef,omitempty"`

	// RetentionPolicy defines how long to keep conversation history
	// +optional
	RetentionPolicy *RetentionPolicySpec `json:"retentionPolicy,omitempty"`

	// MaxConversations limits the number of concurrent conversations
	// +optional
	MaxConversations *int32 `json:"maxConversations,omitempty"`
}

// RetentionPolicySpec defines data retention policy
type RetentionPolicySpec struct {
	// MaxAge is the maximum age of data to retain (e.g., "7d", "30d")
	// +kubebuilder:validation:Pattern=`^[0-9]+(d|w|m|y)$`
	// +optional
	MaxAge string `json:"maxAge,omitempty"`

	// MaxMessages is the maximum number of messages to retain per conversation
	// +optional
	MaxMessages *int32 `json:"maxMessages,omitempty"`
}

// AgentObservabilitySpec defines agent monitoring
type AgentObservabilitySpec struct {
	// Metrics enables metrics collection
	// +kubebuilder:default=true
	Metrics bool `json:"metrics,omitempty"`

	// Tracing enables distributed tracing
	// +kubebuilder:default=false
	Tracing bool `json:"tracing,omitempty"`

	// LogLevel defines the logging level
	// +kubebuilder:validation:Enum=debug;info;warn;error
	// +kubebuilder:default=info
	LogLevel string `json:"logLevel,omitempty"`

	// LogConversations enables conversation logging
	// +kubebuilder:default=true
	LogConversations bool `json:"logConversations,omitempty"`
}

// AgentRateLimitSpec defines agent-level rate limiting
type AgentRateLimitSpec struct {
	// RequestsPerMinute limits requests per minute
	// +optional
	RequestsPerMinute *int32 `json:"requestsPerMinute,omitempty"`

	// TokensPerMinute limits tokens per minute
	// +optional
	TokensPerMinute *int32 `json:"tokensPerMinute,omitempty"`

	// ToolCallsPerMinute limits tool invocations per minute
	// +optional
	ToolCallsPerMinute *int32 `json:"toolCallsPerMinute,omitempty"`
}

// SafetyConfigSpec defines safety constraints
type SafetyConfigSpec struct {
	// MaxToolCallsPerIteration limits tool calls per reasoning loop
	// +kubebuilder:validation:Minimum=0
	// +kubebuilder:default=10
	// +optional
	MaxToolCallsPerIteration *int32 `json:"maxToolCallsPerIteration,omitempty"`

	// BlockedTools lists tools that are explicitly blocked
	// +optional
	BlockedTools []string `json:"blockedTools,omitempty"`

	// RequireApprovalFor lists tools requiring human approval
	// +optional
	RequireApprovalFor []string `json:"requireApprovalFor,omitempty"`

	// ContentFilters defines content filtering rules
	// +optional
	ContentFilters []AgentContentFilterSpec `json:"contentFilters,omitempty"`

	// MaxCostPerExecution limits cost per execution
	// +optional
	MaxCostPerExecution *float64 `json:"maxCostPerExecution,omitempty"`
}

// AgentContentFilterSpec defines a content filter
type AgentContentFilterSpec struct {
	// Type is the filter type (profanity, pii, toxic, custom)
	// +kubebuilder:validation:Enum=profanity;pii;toxic;custom
	Type string `json:"type"`

	// Action defines what to do when filter matches (block, warn, log)
	// +kubebuilder:validation:Enum=block;warn;log
	// +kubebuilder:default=block
	Action string `json:"action,omitempty"`

	// Pattern is a regex pattern for custom filters
	// +optional
	Pattern string `json:"pattern,omitempty"`
}

// WorkspaceSpec defines persistent workspace storage for an agent
type WorkspaceSpec struct {
	// Enabled controls whether to create a workspace volume
	// +kubebuilder:default=true
	// +optional
	Enabled bool `json:"enabled,omitempty"`

	// Size is the requested storage size (e.g., "10Gi", "1Ti")
	// +kubebuilder:validation:Pattern=`^[0-9]+(Ei|Pi|Ti|Gi|Mi|Ki|E|P|T|G|M|K)$`
	// +kubebuilder:default="10Gi"
	// +optional
	Size string `json:"size,omitempty"`

	// StorageClassName specifies the StorageClass for the PVC
	// If not specified, uses the cluster default
	// +optional
	StorageClassName *string `json:"storageClassName,omitempty"`

	// AccessMode defines the volume access mode
	// +kubebuilder:validation:Enum=ReadWriteOnce;ReadWriteMany
	// +kubebuilder:default=ReadWriteOnce
	// +optional
	AccessMode string `json:"accessMode,omitempty"`

	// MountPath is where the workspace is mounted in containers
	// +kubebuilder:default="/workspace"
	// +optional
	MountPath string `json:"mountPath,omitempty"`
}

// LanguageAgentStatus defines the observed state of LanguageAgent
type LanguageAgentStatus struct {
	// ObservedGeneration reflects the generation of the most recently observed LanguageAgent
	// +optional
	ObservedGeneration int64 `json:"observedGeneration,omitempty"`

	// Phase represents the current phase (Pending, Running, Succeeded, Failed, Unknown)
	// +kubebuilder:validation:Enum=Pending;Running;Succeeded;Failed;Unknown;Suspended
	// +optional
	Phase string `json:"phase,omitempty"`

	// Conditions represent the latest available observations of the agent's state
	// +optional
	// +listType=map
	// +listMapKey=type
	// +patchMergeKey=type
	// +patchStrategy=merge
	Conditions []metav1.Condition `json:"conditions,omitempty" patchStrategy:"merge" patchMergeKey:"type"`

	// ActiveReplicas is the number of agent pods currently running
	// +optional
	ActiveReplicas int32 `json:"activeReplicas,omitempty"`

	// ReadyReplicas is the number of agent pods ready
	// +optional
	ReadyReplicas int32 `json:"readyReplicas,omitempty"`

	// ExecutionCount is the total number of executions
	// +optional
	ExecutionCount int64 `json:"executionCount,omitempty"`

	// SuccessfulExecutions is the number of successful executions
	// +optional
	SuccessfulExecutions int64 `json:"successfulExecutions,omitempty"`

	// FailedExecutions is the number of failed executions
	// +optional
	FailedExecutions int64 `json:"failedExecutions,omitempty"`

	// LastExecutionTime is the timestamp of the last execution
	// +optional
	LastExecutionTime *metav1.Time `json:"lastExecutionTime,omitempty"`

	// LastExecutionResult describes the result of the last execution
	// +optional
	LastExecutionResult string `json:"lastExecutionResult,omitempty"`

	// CurrentGoal is the current goal being pursued (for autonomous agents)
	// +optional
	CurrentGoal string `json:"currentGoal,omitempty"`

	// IterationCount is the current iteration in the reasoning loop
	// +optional
	IterationCount int32 `json:"iterationCount,omitempty"`

	// Metrics contains execution metrics
	// +optional
	Metrics *AgentMetrics `json:"metrics,omitempty"`

	// ActiveConversations is the number of active conversations
	// +optional
	ActiveConversations int32 `json:"activeConversations,omitempty"`

	// ToolUsage tracks tool invocation statistics
	// +optional
	ToolUsage []ToolUsageSpec `json:"toolUsage,omitempty"`

	// ModelUsage tracks model usage statistics
	// +optional
	ModelUsage []ModelUsageSpec `json:"modelUsage,omitempty"`

	// CostMetrics contains cost tracking data
	// +optional
	CostMetrics *AgentCostMetrics `json:"costMetrics,omitempty"`

	// LastUpdateTime is the last time the status was updated
	// +optional
	LastUpdateTime *metav1.Time `json:"lastUpdateTime,omitempty"`

	// Message provides human-readable details about the current state
	// +optional
	Message string `json:"message,omitempty"`

	// Reason provides a machine-readable reason for the current state
	// +optional
	Reason string `json:"reason,omitempty"`

	// SynthesisInfo contains information about code synthesis
	// +optional
	SynthesisInfo *SynthesisInfo `json:"synthesisInfo,omitempty"`

	// UUID is a unique identifier for this agent instance
	// Used for webhook routing (e.g., <uuid>.agents.domain.com)
	// +optional
	UUID string `json:"uuid,omitempty"`

	// WebhookURLs contains the URLs where this agent can receive webhooks
	// +optional
	WebhookURLs []string `json:"webhookURLs,omitempty"`

	// RuntimeErrors contains recent runtime errors for self-healing
	// +optional
	RuntimeErrors []RuntimeError `json:"runtimeErrors,omitempty"`

	// LastCrashLog contains the last 100 lines of logs before crash
	// +optional
	LastCrashLog string `json:"lastCrashLog,omitempty"`

	// ConsecutiveFailures tracks consecutive pod failures
	// +optional
	ConsecutiveFailures int32 `json:"consecutiveFailures,omitempty"`

	// FailureReason categorizes the failure type (Synthesis|Runtime|Infrastructure)
	// +optional
	FailureReason string `json:"failureReason,omitempty"`

	// SelfHealingAttempts tracks how many self-healing synthesis attempts have been made
	// +optional
	SelfHealingAttempts int32 `json:"selfHealingAttempts,omitempty"`

	// LastSuccessfulCode stores the last known working code for rollback
	// +optional
	LastSuccessfulCode string `json:"lastSuccessfulCode,omitempty"`
}

// SynthesisInfo contains metadata about agent code synthesis
type SynthesisInfo struct {
	// LastSynthesisTime is when the code was last synthesized
	// +optional
	LastSynthesisTime *metav1.Time `json:"lastSynthesisTime,omitempty"`

	// SynthesisModel is the LLM model used for synthesis
	// +optional
	SynthesisModel string `json:"synthesisModel,omitempty"`

	// SynthesisDuration is how long synthesis took (in seconds)
	// +optional
	SynthesisDuration float64 `json:"synthesisDuration,omitempty"`

	// CodeHash is the SHA256 hash of the current synthesized code
	// +optional
	CodeHash string `json:"codeHash,omitempty"`

	// InstructionsHash is the SHA256 hash of the instructions that generated the code
	// +optional
	InstructionsHash string `json:"instructionsHash,omitempty"`

	// ValidationErrors contains any validation errors from the last synthesis
	// +optional
	ValidationErrors []string `json:"validationErrors,omitempty"`

	// SynthesisAttempts is the number of synthesis attempts for current instructions
	// +optional
	SynthesisAttempts int32 `json:"synthesisAttempts,omitempty"`
}

// RuntimeError captures runtime failure information for self-healing
type RuntimeError struct {
	// Timestamp is when the error occurred
	// +optional
	Timestamp metav1.Time `json:"timestamp"`

	// ErrorType is the exception class or error type
	// +optional
	ErrorType string `json:"errorType,omitempty"`

	// ErrorMessage is the error message
	// +optional
	ErrorMessage string `json:"errorMessage,omitempty"`

	// StackTrace contains the error stack trace
	// +optional
	StackTrace []string `json:"stackTrace,omitempty"`

	// ContainerExitCode is the container exit code
	// +optional
	ContainerExitCode int32 `json:"exitCode,omitempty"`

	// SynthesisAttempt indicates which synthesis iteration this error occurred in
	// +optional
	SynthesisAttempt int32 `json:"synthesisAttempt,omitempty"`
}

// AgentMetrics contains agent execution metrics
type AgentMetrics struct {
	// AverageIterations is the average number of iterations per execution
	// +optional
	AverageIterations *float64 `json:"averageIterations,omitempty"`

	// AverageExecutionTime is the average execution time in seconds
	// +optional
	AverageExecutionTime *float64 `json:"averageExecutionTime,omitempty"`

	// TotalTokens is the total number of tokens consumed
	// +optional
	TotalTokens int64 `json:"totalTokens,omitempty"`

	// TotalToolCalls is the total number of tool invocations
	// +optional
	TotalToolCalls int64 `json:"totalToolCalls,omitempty"`

	// SuccessRate is the percentage of successful executions
	// +optional
	SuccessRate *float64 `json:"successRate,omitempty"`
}

// ToolUsageSpec tracks tool usage
type ToolUsageSpec struct {
	// ToolName is the name of the tool
	ToolName string `json:"toolName"`

	// InvocationCount is the number of times this tool was invoked
	InvocationCount int64 `json:"invocationCount"`

	// SuccessCount is the number of successful invocations
	SuccessCount int64 `json:"successCount"`

	// FailureCount is the number of failed invocations
	FailureCount int64 `json:"failureCount"`

	// AverageLatency is the average latency in milliseconds
	// +optional
	AverageLatency *int32 `json:"averageLatency,omitempty"`
}

// ModelUsageSpec tracks model usage
type ModelUsageSpec struct {
	// ModelName is the name of the model
	ModelName string `json:"modelName"`

	// RequestCount is the number of requests to this model
	RequestCount int64 `json:"requestCount"`

	// TotalTokens is the total tokens consumed by this model
	TotalTokens int64 `json:"totalTokens"`

	// InputTokens is the total input tokens
	// +optional
	InputTokens int64 `json:"inputTokens,omitempty"`

	// OutputTokens is the total output tokens
	// +optional
	OutputTokens int64 `json:"outputTokens,omitempty"`
}

// AgentCostMetrics contains agent cost tracking
type AgentCostMetrics struct {
	// TotalCost is the total cost incurred by this agent
	// +optional
	TotalCost *float64 `json:"totalCost,omitempty"`

	// ModelCosts breaks down cost by model
	// +optional
	ModelCosts []ModelCostSpec `json:"modelCosts,omitempty"`

	// Currency is the currency for cost metrics
	// +optional
	Currency string `json:"currency,omitempty"`

	// LastReset is when cost metrics were last reset
	// +optional
	LastReset *metav1.Time `json:"lastReset,omitempty"`
}

// ModelCostSpec tracks cost per model
type ModelCostSpec struct {
	// ModelName is the name of the model
	ModelName string `json:"modelName"`

	// Cost is the total cost for this model
	Cost float64 `json:"cost"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// Condition types for LanguageAgent
const (
	// WebhookRouteCreatedCondition indicates that the webhook route (HTTPRoute/Ingress) has been created
	WebhookRouteCreatedCondition = "WebhookRouteCreated"
	// WebhookRouteReadyCondition indicates that the webhook route is ready and serving traffic
	WebhookRouteReadyCondition = "WebhookRouteReady"
)

// +kubebuilder:resource:scope=Namespaced,shortName=lagent
// +kubebuilder:printcolumn:name="Mode",type=string,JSONPath=`.spec.executionMode`
// +kubebuilder:printcolumn:name="Phase",type=string,JSONPath=`.status.phase`
// +kubebuilder:printcolumn:name="Replicas",type=integer,JSONPath=`.status.activeReplicas`
// +kubebuilder:printcolumn:name="Executions",type=integer,JSONPath=`.status.executionCount`
// +kubebuilder:printcolumn:name="Success Rate",type=string,JSONPath=`.status.metrics.successRate`
// +kubebuilder:printcolumn:name="Age",type=date,JSONPath=`.metadata.creationTimestamp`

// LanguageAgent is the Schema for the languageagents API
type LanguageAgent struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   LanguageAgentSpec   `json:"spec,omitempty"`
	Status LanguageAgentStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// LanguageAgentList contains a list of LanguageAgent
type LanguageAgentList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []LanguageAgent `json:"items"`
}

func init() {
	SchemeBuilder.Register(&LanguageAgent{}, &LanguageAgentList{})
}
