package v1alpha1

import (
	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// LanguageModelSpec defines the desired state of LanguageModel
// +kubebuilder:validation:XValidation:rule="has(self.provider) != has(self.litellmProvider)",message="set exactly one of provider or litellmProvider"
type LanguageModelSpec struct {
	// Provider is one of the providers the operator documents and validates.
	// For any other LiteLLM provider, set litellmProvider instead.
	// "custom" is deprecated and behaves exactly like "openai-compatible".
	// +kubebuilder:validation:Enum=openai;anthropic;gemini;openai-compatible;azure;bedrock;vertex;custom
	// +optional
	Provider string `json:"provider,omitempty"`

	// LiteLLMProvider is a LiteLLM provider prefix (e.g. "deepseek", "dashscope",
	// "hosted_vllm") for providers not covered by Provider. The gateway calls the
	// model as "<litellmProvider>/<modelName>".
	// +kubebuilder:validation:Pattern=`^[a-z0-9_]+$`
	// +optional
	LiteLLMProvider string `json:"litellmProvider,omitempty"`

	// ModelName is the specific model identifier (e.g., "gpt-4", "claude-3-opus"), or
	// "*" for a wildcard model that stands for the provider's whole catalogue: agents
	// then pick a model with spec.models[].model and call it as "<name>/<model>".
	// +kubebuilder:validation:Required
	// +kubebuilder:validation:MinLength=1
	ModelName string `json:"modelName"`

	// Endpoint is the API endpoint URL (required for openai-compatible and azure)
	// +optional
	Endpoint string `json:"endpoint,omitempty"`

	// APIKeySecretRef references a secret containing the API key. Shorthand for
	// a single key; takes precedence over an API key in CredentialsSecretRef.
	// +optional
	APIKeySecretRef *SecretReference `json:"apiKeySecretRef,omitempty"`

	// CredentialsSecretRef references a Secret whose keys are this model's
	// credentials, for providers that need more than one value: AWS access keys
	// or a Bedrock bearer token, a Vertex service-account JSON, Azure AD app
	// credentials. Keys are matched by name (e.g. AWS_ACCESS_KEY_ID,
	// AWS_BEARER_TOKEN_BEDROCK, VERTEX_CREDENTIALS, AZURE_CLIENT_SECRET) and
	// applied to this model only.
	// +optional
	CredentialsSecretRef *CredentialsSecretReference `json:"credentialsSecretRef,omitempty"`

	// Region is the cloud region (Bedrock: the AWS region).
	// +optional
	Region string `json:"region,omitempty"`

	// Project is the cloud project (Vertex: the GCP project ID).
	// +optional
	Project string `json:"project,omitempty"`

	// Location is the cloud location (Vertex: e.g. "us-central1").
	// +optional
	Location string `json:"location,omitempty"`

	// APIVersion is the provider API version (Azure: e.g. "2025-01-01-preview").
	// +optional
	APIVersion string `json:"apiVersion,omitempty"`

	// Params are passed through into this model's LiteLLM params, overriding the
	// values derived from the fields above (e.g. aws_bedrock_runtime_endpoint,
	// extra_headers, use_chat_completions_api). Credentials do not belong here:
	// keys that name one are rejected; use credentialsSecretRef.
	// +optional
	Params map[string]apiextensionsv1.JSON `json:"params,omitempty"`

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

	// Key is the key within the secret containing the value
	// +kubebuilder:default="api-key"
	// +optional
	Key string `json:"key,omitempty"`
}

// CredentialsSecretReference references a whole Secret.
type CredentialsSecretReference struct {
	// Name is the name of the secret
	// +kubebuilder:validation:Required
	Name string `json:"name"`
}

// EffectiveProvider is the LiteLLM-facing provider: Provider, or LiteLLMProvider
// when Provider is unset.
func (s *LanguageModelSpec) EffectiveProvider() string {
	if s.Provider != "" {
		return s.Provider
	}
	return s.LiteLLMProvider
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
