package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// LanguageAgentVersionSpec defines the desired state of LanguageAgentVersion
type LanguageAgentVersionSpec struct {
	// AgentRef references the LanguageAgent this version belongs to
	// +kubebuilder:validation:Required
	AgentRef AgentReference `json:"agentRef"`

	// Version is the semantic version number of this agent code
	// +kubebuilder:validation:Minimum=1
	// +kubebuilder:validation:Required
	Version int32 `json:"version"`

	// Code contains the complete Ruby DSL agent code for this version
	// +kubebuilder:validation:Required
	// +kubebuilder:validation:MinLength=1
	Code string `json:"code"`

	// OptimizedTasks contains information about tasks that were optimized in this version
	// +optional
	OptimizedTasks map[string]OptimizedTask `json:"optimizedTasks,omitempty"`

	// SourceType indicates how this version was created
	// +kubebuilder:validation:Enum=learning;manual;import
	// +kubebuilder:default=manual
	// +optional
	SourceType string `json:"sourceType,omitempty"`

	// Description provides human-readable information about this version
	// +optional
	Description string `json:"description,omitempty"`

	// CreatedBy indicates who/what created this version (user, learning system, etc.)
	// +optional
	CreatedBy string `json:"createdBy,omitempty"`
}

// LanguageAgentVersionStatus defines the observed state of LanguageAgentVersion
type LanguageAgentVersionStatus struct {
	// Phase represents the current lifecycle phase of this version
	// +kubebuilder:validation:Enum=Pending;Ready;Failed;Deprecated
	// +kubebuilder:default=Pending
	Phase string `json:"phase,omitempty"`

	// AppliedAt indicates when this version was successfully applied
	// +optional
	AppliedAt *metav1.Time `json:"appliedAt,omitempty"`

	// ConfigMapName is the name of the underlying ConfigMap created for this version
	// +optional
	ConfigMapName string `json:"configMapName,omitempty"`

	// Conditions represent the latest available observations of the version's state
	// +optional
	Conditions []metav1.Condition `json:"conditions,omitempty"`

	// Message provides additional information about the current state
	// +optional
	Message string `json:"message,omitempty"`
}

// AgentReference references a LanguageAgent
type AgentReference struct {
	// Name is the name of the LanguageAgent
	// +kubebuilder:validation:Required
	Name string `json:"name"`

	// Namespace is the namespace of the LanguageAgent (defaults to same namespace)
	// +optional
	Namespace string `json:"namespace,omitempty"`
}

// OptimizedTask contains metadata about an optimized task in this version
type OptimizedTask struct {
	// Name of the task that was optimized
	// +kubebuilder:validation:Required
	Name string `json:"name"`

	// ConfidenceScore is the confidence level (0-100) that this optimization is beneficial
	// +kubebuilder:validation:Minimum=0
	// +kubebuilder:validation:Maximum=100
	// +optional
	ConfidenceScore int32 `json:"confidenceScore,omitempty"`

	// OptimizationReason describes why this task was optimized
	// +optional
	OptimizationReason string `json:"optimizationReason,omitempty"`

	// OriginalComplexity describes the original task complexity before optimization
	// +optional
	OriginalComplexity string `json:"originalComplexity,omitempty"`

	// OptimizedComplexity describes the task complexity after optimization
	// +optional
	OptimizedComplexity string `json:"optimizedComplexity,omitempty"`
}

// AgentVersionReference references a specific LanguageAgentVersion
type AgentVersionReference struct {
	// Name is the name of the LanguageAgentVersion
	// +kubebuilder:validation:Required
	Name string `json:"name"`

	// Namespace is the namespace of the LanguageAgentVersion (defaults to same namespace)
	// +optional
	Namespace string `json:"namespace,omitempty"`

	// Lock prevents automatic version updates by the learning controller
	// When true, the learning controller will not modify this agentVersionRef
	// +kubebuilder:default=false
	// +optional
	Lock bool `json:"lock,omitempty"`
}

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status
//+kubebuilder:printcolumn:name="Agent",type=string,JSONPath=`.spec.agentRef.name`
//+kubebuilder:printcolumn:name="Version",type=integer,JSONPath=`.spec.version`
//+kubebuilder:printcolumn:name="Source",type=string,JSONPath=`.spec.sourceType`
//+kubebuilder:printcolumn:name="Phase",type=string,JSONPath=`.status.phase`
//+kubebuilder:printcolumn:name="Age",type=date,JSONPath=`.metadata.creationTimestamp`

// LanguageAgentVersion represents a specific version of optimized agent code
type LanguageAgentVersion struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   LanguageAgentVersionSpec   `json:"spec,omitempty"`
	Status LanguageAgentVersionStatus `json:"status,omitempty"`
}

//+kubebuilder:object:root=true

// LanguageAgentVersionList contains a list of LanguageAgentVersion
type LanguageAgentVersionList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []LanguageAgentVersion `json:"items"`
}

func init() {
	SchemeBuilder.Register(&LanguageAgentVersion{}, &LanguageAgentVersionList{})
}
