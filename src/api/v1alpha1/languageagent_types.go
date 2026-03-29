package v1alpha1

import (
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// LanguageAgentSpec defines the desired state of LanguageAgent
type LanguageAgentSpec struct {
	// Image is the container image to run for this agent
	// +kubebuilder:validation:Required
	// +kubebuilder:validation:MinLength=1
	// +kubebuilder:validation:Pattern=`^([a-z0-9]+([._-][a-z0-9]+)*\/)*[a-z0-9]+([._-][a-z0-9]+)*(:[a-z0-9]+([._-][a-z0-9]+)*)?$`
	Image string `json:"image"`

	// ImagePullPolicy defines when to pull the container image
	// +kubebuilder:validation:Enum=Always;Never;IfNotPresent
	// +kubebuilder:default=IfNotPresent
	// +optional
	ImagePullPolicy corev1.PullPolicy `json:"imagePullPolicy,omitempty"`

	// ImagePullSecrets is a list of references to secrets for pulling images
	// +optional
	ImagePullSecrets []corev1.LocalObjectReference `json:"imagePullSecrets,omitempty"`

	// Models is a list of LanguageModel references this agent can use
	// +optional
	Models []ModelReference `json:"models,omitempty"`

	// Tools is a list of LanguageTool references available to this agent
	// +optional
	Tools []ToolReference `json:"tools,omitempty"`

	// Personas is a list of LanguagePersona references that compose in order of importance
	// Personas are merged with later personas taking precedence over earlier ones
	// +optional
	Personas []PersonaReference `json:"personas,omitempty"`

	// Instructions provides system instructions for the agent.
	// Mounted at /etc/agent/instructions.txt if set.
	// +optional
	Instructions string `json:"instructions,omitempty"`

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

	// Workspace defines persistent storage for the agent
	// +optional
	Workspace *WorkspaceSpec `json:"workspace,omitempty"`

	// Egress defines external network access rules for this agent
	// By default, agents can access all resources within the cluster but no external endpoints
	// +optional
	Egress []NetworkRule `json:"egress,omitempty"`

	// Port is the port the agent container listens on.
	// Used for the ClusterIP Service and NetworkPolicy ingress rules.
	// Defaults to 8080.
	// +kubebuilder:validation:Minimum=1
	// +kubebuilder:validation:Maximum=65535
	// +optional
	Port *int32 `json:"port,omitempty"`

	// LivenessProbe defines the liveness probe for the agent container.
	// If not set, no liveness probe is configured.
	// +optional
	LivenessProbe *corev1.Probe `json:"livenessProbe,omitempty"`

	// ReadinessProbe defines the readiness probe for the agent container.
	// If not set, no readiness probe is configured.
	// +optional
	ReadinessProbe *corev1.Probe `json:"readinessProbe,omitempty"`

	// InitContainers are additional init containers injected before the agent container starts.
	// Useful for seeding config, migrating workspace data, or other pre-start setup.
	// +optional
	InitContainers []corev1.Container `json:"initContainers,omitempty"`
}

// ModelReference references a LanguageModel
type ModelReference struct {
	// Name is the name of the LanguageModel
	// +kubebuilder:validation:Required
	// +kubebuilder:validation:Pattern=`^[a-z0-9]([-a-z0-9]*[a-z0-9])?$`
	// +kubebuilder:validation:MaxLength=63
	Name string `json:"name"`

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
	// +kubebuilder:validation:Pattern=`^[a-z0-9]([-a-z0-9]*[a-z0-9])?$`
	// +kubebuilder:validation:MaxLength=63
	Name string `json:"name"`

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
	// +kubebuilder:validation:Pattern=`^[a-z0-9]([-a-z0-9]*[a-z0-9])?$`
	// +kubebuilder:validation:MaxLength=63
	Name string `json:"name"`
}

// WorkspaceSpec defines persistent workspace storage for an agent
type WorkspaceSpec struct {
	// Enabled controls whether to create a workspace volume
	// +kubebuilder:default=true
	// +optional
	Enabled bool `json:"enabled,omitempty"`

	// Size is the requested storage size (e.g., "10Gi", "1.5Ti", "500Mi")
	// Supports integer and decimal quantities with standard Kubernetes suffixes
	// +kubebuilder:validation:Pattern=`^([0-9]*\.?[0-9]+)(Ei|Pi|Ti|Gi|Mi|Ki|E|P|T|G|M|K|m)?$`
	// +kubebuilder:validation:MinLength=1
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

	// UUID is a unique identifier for this agent instance
	// Used for webhook routing (e.g., <uuid>.domain.com)
	// +optional
	UUID string `json:"uuid,omitempty"`

	// WebhookURLs contains the URLs where this agent can receive webhooks
	// +optional
	WebhookURLs []string `json:"webhookURLs,omitempty"`
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
// +kubebuilder:printcolumn:name="Phase",type=string,JSONPath=`.status.phase`
// +kubebuilder:printcolumn:name="Replicas",type=integer,JSONPath=`.status.activeReplicas`
// +kubebuilder:printcolumn:name="Executions",type=integer,JSONPath=`.status.executionCount`
// +kubebuilder:printcolumn:name="Success Rate",type=string,JSONPath=`.status.metrics.successRate`
// +kubebuilder:printcolumn:name="Age",type=date,JSONPath=`.metadata.creationTimestamp`

// LanguageAgent is the Schema for the languageagents API
// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
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
