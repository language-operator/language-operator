package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// LanguageAgentSpec defines the desired state of LanguageAgent
type LanguageAgentSpec struct {
	// Image is the container image to run for this agent
	// +kubebuilder:validation:Required
	// +kubebuilder:validation:MinLength=1
	// +kubebuilder:validation:Pattern=`^([a-z0-9]+([._-][a-z0-9]+)*\/)*[a-z0-9]+([._-][a-z0-9]+)*(:[a-z0-9]+([._-][a-z0-9]+)*)?$`
	Image string `json:"image"`

	// Models is a list of LanguageModel references this agent can use
	// +optional
	Models []ModelReference `json:"models,omitempty"`

	// Tools is a list of LanguageTool references available to this agent
	// +optional
	Tools []ToolReference `json:"tools,omitempty"`

	// Persona is the name of a LanguagePersona this agent uses
	// +optional
	Persona string `json:"persona,omitempty"`

	// Instructions provides system instructions for the agent.
	// Mounted at /etc/agent/instructions.txt if set.
	// +optional
	Instructions string `json:"instructions,omitempty"`

	// Timeout is the maximum execution time (e.g., "10m", "1h")
	// +kubebuilder:validation:Pattern=`^[0-9]+(ns|us|µs|ms|s|m|h)$`
	// +kubebuilder:default="10m"
	// +optional
	Timeout string `json:"timeout,omitempty"`

	// Workspace defines persistent storage for the agent
	// +optional
	Workspace *WorkspaceSpec `json:"workspace,omitempty"`

	// NetworkPolicies defines network access rules for this agent
	// By default, agents can access all resources within the cluster but no external endpoints
	// +optional
	NetworkPolicies []NetworkRule `json:"networkPolicies,omitempty"`

	// Port is the port the agent container listens on.
	// Used for the ClusterIP Service and NetworkPolicy ingress rules.
	// Defaults to 8080.
	// +kubebuilder:validation:Minimum=1
	// +kubebuilder:validation:Maximum=65535
	// +optional
	Port *int32 `json:"port,omitempty"`

	// Deployment groups Kubernetes-specific pod and container configuration.
	// +optional
	Deployment DeploymentSpec `json:"deployment,omitempty"`
}

// ModelReference references a LanguageModel
type ModelReference struct {
	// Name is the name of the LanguageModel
	// +kubebuilder:validation:Required
	// +kubebuilder:validation:Pattern=`^[a-z0-9]([-a-z0-9]*[a-z0-9])?$`
	// +kubebuilder:validation:MaxLength=63
	Name string `json:"name"`

	// Role defines the purpose of this model — a hint for the agent runtime for model selection
	// (e.g. prefer role=primary for general calls, role=reasoning for chain-of-thought).
	// The operator does not enforce routing by role; it is surfaced in the agent config (agent.json).
	// +kubebuilder:validation:Enum=primary;fallback;reasoning;tool-calling;summarization
	// +kubebuilder:default=primary
	// +optional
	Role string `json:"role,omitempty"`

	// Priority for model selection — a hint for the agent runtime (lower value = higher priority).
	// The operator does not enforce priority; it is surfaced in the agent config (agent.json).
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

	// Enabled indicates if this tool is available to the agent.
	// Defaults to true. Set to false to explicitly disable the tool without removing it.
	// +kubebuilder:default=true
	// +optional
	Enabled *bool `json:"enabled,omitempty"`
}

// WorkspaceSpec defines persistent workspace storage for an agent
type WorkspaceSpec struct {
	// Enabled controls whether to create a workspace volume.
	// Defaults to true. Set to false to explicitly disable without removing the workspace config.
	// +kubebuilder:default=true
	// +optional
	Enabled *bool `json:"enabled,omitempty"`

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
	// Phase represents the current phase of the agent
	// +kubebuilder:validation:Enum=Running
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

	// UUID is a unique identifier for this agent instance
	// Used for webhook routing (e.g., <uuid>.domain.com)
	// +optional
	UUID string `json:"uuid,omitempty"`

	// WebhookURLs contains the URLs where this agent can receive webhooks
	// +optional
	WebhookURLs []string `json:"webhookURLs,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// Condition types for LanguageAgent
const (
	// WebhookRouteCreatedCondition indicates that the webhook Ingress has been created
	WebhookRouteCreatedCondition = "WebhookRouteCreated"
	// WebhookRouteReadyCondition indicates that the webhook route is ready and serving traffic
	WebhookRouteReadyCondition = "WebhookRouteReady"
)

// +kubebuilder:resource:scope=Namespaced,shortName=lagent
// +kubebuilder:printcolumn:name="Phase",type=string,JSONPath=`.status.phase`
// +kubebuilder:printcolumn:name="Replicas",type=integer,JSONPath=`.status.activeReplicas`
// +kubebuilder:printcolumn:name="Ready",type=integer,JSONPath=`.status.readyReplicas`
// +kubebuilder:printcolumn:name="UUID",type=string,JSONPath=`.status.uuid`
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
