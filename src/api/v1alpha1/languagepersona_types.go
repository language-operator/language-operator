package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// LanguagePersonaSpec defines the desired state of LanguagePersona
type LanguagePersonaSpec struct {
	// Tone describes the agent's communication style
	// e.g. "professional", "concise and direct", "warm and encouraging"
	// +optional
	Tone string `json:"tone,omitempty"`

	// Personality describes the agent's character and behavioural traits
	// e.g. "curious and methodical, always explains reasoning step by step"
	// +optional
	Personality string `json:"personality,omitempty"`

	// Expertise describes the agent's domain knowledge and skills
	// e.g. "senior software engineer specialising in distributed systems and Go"
	// +optional
	Expertise string `json:"expertise,omitempty"`
}

// LanguagePersonaStatus defines the observed state of LanguagePersona
type LanguagePersonaStatus struct {
	// ObservedGeneration reflects the generation of the most recently observed LanguagePersona
	// +optional
	ObservedGeneration int64 `json:"observedGeneration,omitempty"`

	// Phase represents the current phase
	// +kubebuilder:validation:Enum=Ready
	// +optional
	Phase string `json:"phase,omitempty"`

	// Conditions represent the latest available observations of the persona's state
	// +optional
	// +listType=map
	// +listMapKey=type
	// +patchMergeKey=type
	// +patchStrategy=merge
	Conditions []metav1.Condition `json:"conditions,omitempty" patchStrategy:"merge" patchMergeKey:"type"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:resource:scope=Namespaced,shortName=lpersona
// +kubebuilder:printcolumn:name="Personality",type=string,JSONPath=`.spec.personality`
// +kubebuilder:printcolumn:name="Tone",type=string,JSONPath=`.spec.tone`
// +kubebuilder:printcolumn:name="Phase",type=string,JSONPath=`.status.phase`
// +kubebuilder:printcolumn:name="Age",type=date,JSONPath=`.metadata.creationTimestamp`

// LanguagePersona is the Schema for the languagepersonas API
type LanguagePersona struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   LanguagePersonaSpec   `json:"spec,omitempty"`
	Status LanguagePersonaStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// LanguagePersonaList contains a list of LanguagePersona
type LanguagePersonaList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []LanguagePersona `json:"items"`
}

func init() {
	SchemeBuilder.Register(&LanguagePersona{}, &LanguagePersonaList{})
}
