package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// LanguageAgentTaskSpec defines the desired state of LanguageAgentTask
type LanguageAgentTaskSpec struct {
	// AgentRef is the name of the LanguageAgent that owns this task
	// +kubebuilder:validation:Required
	AgentRef string `json:"agentRef"`

	// TaskID is the A2A task ID assigned by the agent
	// +kubebuilder:validation:Required
	TaskID string `json:"taskId"`

	// ContextID is the A2A context (conversation session) this task belongs to
	// +optional
	ContextID string `json:"contextId,omitempty"`

	// InitialMessage is the message that created this task
	// +optional
	InitialMessage string `json:"initialMessage,omitempty"`
}

// LanguageAgentTaskStatus defines the observed state of LanguageAgentTask
type LanguageAgentTaskStatus struct {
	// ObservedGeneration reflects the generation of the most recently observed LanguageAgentTask
	// +optional
	ObservedGeneration int64 `json:"observedGeneration,omitempty"`

	// State is the current A2A task state
	// +kubebuilder:validation:Enum=submitted;working;input-required;auth-required;completed;failed;canceled;rejected
	// +optional
	State string `json:"state,omitempty"`

	// Message is the agent's human-readable explanation of the current state.
	// When state is input-required or auth-required, this describes what is needed.
	// +optional
	Message string `json:"message,omitempty"`

	// InputRequired holds details when the task is paused awaiting user or caller input
	// +optional
	InputRequired *InputRequiredStatus `json:"inputRequired,omitempty"`

	// AuthRequired holds details when the task is paused awaiting credentials
	// +optional
	AuthRequired *AuthRequiredStatus `json:"authRequired,omitempty"`

	// Artifacts lists output artifacts produced by the task
	// +optional
	Artifacts []TaskArtifact `json:"artifacts,omitempty"`

	// StartedAt is when the task was first observed
	// +optional
	StartedAt *metav1.Time `json:"startedAt,omitempty"`

	// CompletedAt is when the task reached a terminal state
	// +optional
	CompletedAt *metav1.Time `json:"completedAt,omitempty"`

	// LastUpdatedAt is when the status was last updated
	// +optional
	LastUpdatedAt *metav1.Time `json:"lastUpdatedAt,omitempty"`

	// Conditions represent the latest available observations of the task's state
	// +optional
	// +listType=map
	// +listMapKey=type
	// +patchMergeKey=type
	// +patchStrategy=merge
	Conditions []metav1.Condition `json:"conditions,omitempty" patchStrategy:"merge" patchMergeKey:"type"`
}

// InputRequiredStatus describes what input the agent needs to continue
type InputRequiredStatus struct {
	// Prompt is the agent's description of what input is needed
	// +optional
	Prompt string `json:"prompt,omitempty"`

	// Since is when the task entered input-required state
	// +optional
	Since *metav1.Time `json:"since,omitempty"`
}

// AuthRequiredStatus describes what credentials the agent needs to continue
type AuthRequiredStatus struct {
	// Scheme is the authentication scheme required (e.g. Bearer, APIKey)
	// +optional
	Scheme string `json:"scheme,omitempty"`

	// Description is the agent's explanation of why credentials are needed
	// +optional
	Description string `json:"description,omitempty"`

	// Since is when the task entered auth-required state
	// +optional
	Since *metav1.Time `json:"since,omitempty"`
}

// TaskArtifact represents an output produced by the task
type TaskArtifact struct {
	// Name identifies the artifact
	// +optional
	Name string `json:"name,omitempty"`

	// ArtifactID is the A2A artifact identifier
	// +optional
	ArtifactID string `json:"artifactId,omitempty"`

	// Type is the MIME type of the artifact content
	// +optional
	Type string `json:"type,omitempty"`

	// Description is a human-readable description of the artifact
	// +optional
	Description string `json:"description,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:resource:scope=Namespaced,shortName=latask
// +kubebuilder:printcolumn:name="Agent",type=string,JSONPath=`.spec.agentRef`
// +kubebuilder:printcolumn:name="State",type=string,JSONPath=`.status.state`
// +kubebuilder:printcolumn:name="Context",type=string,JSONPath=`.spec.contextId`
// +kubebuilder:printcolumn:name="Age",type=date,JSONPath=`.metadata.creationTimestamp`

// LanguageAgentTask tracks an in-flight A2A task running on a LanguageAgent.
// Created by the operator when an agent reports a new task via push notification.
// Surfaces INPUT_REQUIRED and AUTH_REQUIRED states as Kubernetes-native signals
// so external actors (humans, controllers) can observe and resolve blocked tasks.
type LanguageAgentTask struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   LanguageAgentTaskSpec   `json:"spec,omitempty"`
	Status LanguageAgentTaskStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// LanguageAgentTaskList contains a list of LanguageAgentTask
type LanguageAgentTaskList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []LanguageAgentTask `json:"items"`
}

func init() {
	SchemeBuilder.Register(&LanguageAgentTask{}, &LanguageAgentTaskList{})
}
