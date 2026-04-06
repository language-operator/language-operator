package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// LanguageModelSpec defines the desired state of LanguageModel
type LanguageModelSpec struct {
	// Provider specifies the LLM provider type
	// +kubebuilder:validation:Required
	// +kubebuilder:validation:Enum=openai;anthropic;openai-compatible;azure;bedrock;vertex;custom
	Provider string `json:"provider"`

	// ModelName is the specific model identifier (e.g., "gpt-4", "claude-3-opus")
	// +kubebuilder:validation:Required
	// +kubebuilder:validation:MinLength=1
	ModelName string `json:"modelName"`

	// Endpoint is the API endpoint URL (required for openai-compatible, azure, custom)
	// +optional
	Endpoint string `json:"endpoint,omitempty"`

	// APIKeySecretRef references a secret containing the API key
	// +optional
	APIKeySecretRef *SecretReference `json:"apiKeySecretRef,omitempty"`

	// RateLimits defines rate limiting configuration
	// +optional
	RateLimits *RateLimitSpec `json:"rateLimits,omitempty"`

	// Timeout specifies request timeout duration (e.g., "5m", "30s")
	// +kubebuilder:validation:Pattern=`^[0-9]+(ns|us|µs|ms|s|m|h)$`
	// +kubebuilder:default="5m"
	// +optional
	Timeout string `json:"timeout,omitempty"`
}

// SecretReference references a Kubernetes Secret
type SecretReference struct {
	// Name is the name of the secret
	// +kubebuilder:validation:Required
	Name string `json:"name"`

	// Namespace is the namespace of the secret (defaults to same namespace as LanguageModel)
	// +optional
	Namespace string `json:"namespace,omitempty"`

	// Key is the key within the secret containing the value
	// +kubebuilder:default="api-key"
	// +optional
	Key string `json:"key,omitempty"`
}

// RateLimitSpec defines rate limiting configuration
type RateLimitSpec struct {
	// RequestsPerMinute limits requests per minute
	// +optional
	RequestsPerMinute *int32 `json:"requestsPerMinute,omitempty"`

	// TokensPerMinute limits tokens per minute
	// +optional
	TokensPerMinute *int32 `json:"tokensPerMinute,omitempty"`
}

// LanguageModelStatus defines the observed state of LanguageModel
type LanguageModelStatus struct {
	// ObservedGeneration reflects the generation of the most recently observed LanguageModel
	// +optional
	ObservedGeneration int64 `json:"observedGeneration,omitempty"`

	// Phase represents the current phase of the model (Pending, Ready, Failed)
	// +kubebuilder:validation:Enum=Pending;Ready;Failed
	// +optional
	Phase string `json:"phase,omitempty"`

	// Conditions represent the latest available observations of the model's state
	// +optional
	// +listType=map
	// +listMapKey=type
	// +patchMergeKey=type
	// +patchStrategy=merge
	Conditions []metav1.Condition `json:"conditions,omitempty" patchStrategy:"merge" patchMergeKey:"type"`

	// Message provides human-readable details about the current state
	// +optional
	Message string `json:"message,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:resource:scope=Namespaced,shortName=lmodel
// +kubebuilder:printcolumn:name="Provider",type=string,JSONPath=`.spec.provider`
// +kubebuilder:printcolumn:name="Model",type=string,JSONPath=`.spec.modelName`
// +kubebuilder:printcolumn:name="Phase",type=string,JSONPath=`.status.phase`
// +kubebuilder:printcolumn:name="Age",type=date,JSONPath=`.metadata.creationTimestamp`

// LanguageModel is the Schema for the languagemodels API
type LanguageModel struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   LanguageModelSpec   `json:"spec,omitempty"`
	Status LanguageModelStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// LanguageModelList contains a list of LanguageModel
type LanguageModelList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []LanguageModel `json:"items"`
}

func init() {
	SchemeBuilder.Register(&LanguageModel{}, &LanguageModelList{})
}
