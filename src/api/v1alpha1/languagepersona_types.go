package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// LanguagePersonaSpec defines the desired state of LanguagePersona
type LanguagePersonaSpec struct {
	// DisplayName is the human-readable name for this persona
	// +kubebuilder:validation:Required
	// +kubebuilder:validation:MinLength=1
	DisplayName string `json:"displayName"`

	// Description describes the persona's role and behavior
	// +kubebuilder:validation:Required
	// +kubebuilder:validation:MinLength=1
	Description string `json:"description"`

	// SystemPrompt is the base system instruction for this persona
	// +kubebuilder:validation:Required
	// +kubebuilder:validation:MinLength=1
	SystemPrompt string `json:"systemPrompt"`

	// Instructions provides additional behavioral guidelines
	// +optional
	Instructions []string `json:"instructions,omitempty"`

	// Rules define conditional behaviors and policies
	// +optional
	Rules []PersonaRule `json:"rules,omitempty"`

	// Examples provide few-shot learning examples
	// +optional
	Examples []PersonaExample `json:"examples,omitempty"`

	// Capabilities lists what this persona can do
	// +optional
	Capabilities []string `json:"capabilities,omitempty"`

	// Limitations lists what this persona should not do
	// +optional
	Limitations []string `json:"limitations,omitempty"`

	// Tone defines the communication style
	// +kubebuilder:validation:Enum=professional;casual;friendly;formal;technical;empathetic;concise;detailed
	// +kubebuilder:default=professional
	// +optional
	Tone string `json:"tone,omitempty"`

	// Language specifies the primary language for responses
	// +kubebuilder:default=en
	// +optional
	Language string `json:"language,omitempty"`

	// ResponseFormat defines preferred response structure
	// +optional
	ResponseFormat *ResponseFormatSpec `json:"responseFormat,omitempty"`

	// ToolPreferences defines how this persona uses tools
	// +optional
	ToolPreferences *ToolPreferencesSpec `json:"toolPreferences,omitempty"`

	// KnowledgeSources references external knowledge bases
	// +optional
	KnowledgeSources []KnowledgeSourceSpec `json:"knowledgeSources,omitempty"`

	// Constraints define operational constraints
	// +optional
	Constraints *PersonaConstraints `json:"constraints,omitempty"`

	// Metadata contains additional persona metadata
	// +optional
	Metadata map[string]string `json:"metadata,omitempty"`

	// Version tracks the persona version
	// +optional
	Version string `json:"version,omitempty"`

	// ParentPersona references a parent persona to inherit from
	// +optional
	ParentPersona *PersonaReference `json:"parentPersona,omitempty"`
}

// PersonaRule defines a conditional behavior rule
type PersonaRule struct {
	// Name is a unique identifier for this rule
	// +kubebuilder:validation:Required
	Name string `json:"name"`

	// Description explains what this rule does
	// +optional
	Description string `json:"description,omitempty"`

	// Condition defines when this rule applies (e.g., "when asked about X")
	// +kubebuilder:validation:Required
	Condition string `json:"condition"`

	// Action defines what to do when condition matches
	// +kubebuilder:validation:Required
	Action string `json:"action"`

	// Priority determines rule evaluation order (lower is higher priority)
	// +kubebuilder:default=100
	// +optional
	Priority *int32 `json:"priority,omitempty"`

	// Enabled indicates if this rule is active
	// +kubebuilder:default=true
	Enabled bool `json:"enabled,omitempty"`
}

// PersonaExample provides a few-shot learning example
type PersonaExample struct {
	// Input is the example user input
	// +kubebuilder:validation:Required
	Input string `json:"input"`

	// Output is the expected persona response
	// +kubebuilder:validation:Required
	Output string `json:"output"`

	// Context provides additional context for this example
	// +optional
	Context string `json:"context,omitempty"`

	// Tags categorize this example
	// +optional
	Tags []string `json:"tags,omitempty"`
}

// ResponseFormatSpec defines response structure preferences
type ResponseFormatSpec struct {
	// Type specifies the response format
	// +kubebuilder:validation:Enum=text;markdown;json;structured;list;table
	// +kubebuilder:default=text
	Type string `json:"type,omitempty"`

	// Template defines a response template
	// +optional
	Template string `json:"template,omitempty"`

	// Schema defines JSON schema for structured responses
	// +optional
	Schema string `json:"schema,omitempty"`

	// MaxLength limits response length in characters
	// +optional
	MaxLength *int32 `json:"maxLength,omitempty"`

	// IncludeSources indicates whether to cite sources
	// +kubebuilder:default=false
	IncludeSources bool `json:"includeSources,omitempty"`

	// IncludeConfidence indicates whether to include confidence scores
	// +kubebuilder:default=false
	IncludeConfidence bool `json:"includeConfidence,omitempty"`
}

// ToolPreferencesSpec defines tool usage preferences
type ToolPreferencesSpec struct {
	// PreferredTools lists tools to prefer using
	// +optional
	PreferredTools []string `json:"preferredTools,omitempty"`

	// AvoidTools lists tools to avoid using
	// +optional
	AvoidTools []string `json:"avoidTools,omitempty"`

	// ToolUsageStrategy defines how aggressively to use tools
	// +kubebuilder:validation:Enum=conservative;balanced;aggressive;minimal
	// +kubebuilder:default=balanced
	Strategy string `json:"strategy,omitempty"`

	// AlwaysConfirm requires confirmation before tool use
	// +kubebuilder:default=false
	AlwaysConfirm bool `json:"alwaysConfirm,omitempty"`

	// ExplainToolUse explains tool usage to users
	// +kubebuilder:default=true
	ExplainToolUse bool `json:"explainToolUse,omitempty"`
}

// KnowledgeSourceSpec references an external knowledge base
type KnowledgeSourceSpec struct {
	// Name is the knowledge source identifier
	// +kubebuilder:validation:Required
	Name string `json:"name"`

	// Type specifies the knowledge source type
	// +kubebuilder:validation:Enum=url;document;database;api;vector-store
	// +kubebuilder:validation:Required
	Type string `json:"type"`

	// URL is the knowledge source URL (for url, api types)
	// +optional
	URL string `json:"url,omitempty"`

	// SecretRef references credentials for accessing the knowledge source
	// +optional
	SecretRef *SecretReference `json:"secretRef,omitempty"`

	// Query defines how to query this knowledge source
	// +optional
	Query string `json:"query,omitempty"`

	// Priority determines knowledge source precedence
	// +optional
	Priority *int32 `json:"priority,omitempty"`

	// Enabled indicates if this knowledge source is active
	// +kubebuilder:default=true
	Enabled bool `json:"enabled,omitempty"`
}

// PersonaConstraints defines operational constraints
type PersonaConstraints struct {
	// MaxResponseTokens limits response length in tokens
	// +optional
	MaxResponseTokens *int32 `json:"maxResponseTokens,omitempty"`

	// MaxToolCalls limits tool invocations per interaction
	// +optional
	MaxToolCalls *int32 `json:"maxToolCalls,omitempty"`

	// MaxKnowledgeQueries limits knowledge base queries per interaction
	// +optional
	MaxKnowledgeQueries *int32 `json:"maxKnowledgeQueries,omitempty"`

	// ResponseTimeout limits response generation time
	// +kubebuilder:validation:Pattern=`^[0-9]+(ns|us|µs|ms|s|m|h)$`
	// +optional
	ResponseTimeout string `json:"responseTimeout,omitempty"`

	// RequireDocumentation requires citing sources for claims
	// +kubebuilder:default=false
	RequireDocumentation bool `json:"requireDocumentation,omitempty"`

	// BlockedTopics lists topics this persona should refuse to discuss
	// +optional
	BlockedTopics []string `json:"blockedTopics,omitempty"`

	// AllowedDomains restricts knowledge sources to specific domains
	// +optional
	AllowedDomains []string `json:"allowedDomains,omitempty"`
}

// LanguagePersonaStatus defines the observed state of LanguagePersona
type LanguagePersonaStatus struct {
	// ObservedGeneration reflects the generation of the most recently observed LanguagePersona
	// +optional
	ObservedGeneration int64 `json:"observedGeneration,omitempty"`

	// Phase represents the current phase (Ready, NotReady, Validating)
	// +kubebuilder:validation:Enum=Ready;NotReady;Validating;Error
	// +optional
	Phase string `json:"phase,omitempty"`

	// Conditions represent the latest available observations of the persona's state
	// +optional
	// +listType=map
	// +listMapKey=type
	// +patchMergeKey=type
	// +patchStrategy=merge
	Conditions []metav1.Condition `json:"conditions,omitempty" patchStrategy:"merge" patchMergeKey:"type"`

	// UsageCount tracks how many agents use this persona
	// +optional
	UsageCount int32 `json:"usageCount,omitempty"`

	// ActiveAgents lists agents currently using this persona
	// +optional
	ActiveAgents []string `json:"activeAgents,omitempty"`

	// ValidationResult contains persona validation results
	// +optional
	ValidationResult *PersonaValidation `json:"validationResult,omitempty"`

	// Metrics contains usage metrics for this persona
	// +optional
	Metrics *PersonaMetrics `json:"metrics,omitempty"`

	// LastUpdateTime is the last time the status was updated
	// +optional
	LastUpdateTime *metav1.Time `json:"lastUpdateTime,omitempty"`

	// Message provides human-readable details about the current state
	// +optional
	Message string `json:"message,omitempty"`

	// Reason provides a machine-readable reason for the current state
	// +optional
	Reason string `json:"reason,omitempty"`
}

// PersonaValidation contains validation results
type PersonaValidation struct {
	// Valid indicates if the persona passed validation
	Valid bool `json:"valid"`

	// ValidationTime is when validation was performed
	// +optional
	ValidationTime *metav1.Time `json:"validationTime,omitempty"`

	// Errors lists validation errors
	// +optional
	Errors []string `json:"errors,omitempty"`

	// Warnings lists validation warnings
	// +optional
	Warnings []string `json:"warnings,omitempty"`

	// Score is an optional quality score (0-100)
	// +optional
	Score *int32 `json:"score,omitempty"`
}

// PersonaMetrics contains persona usage metrics
type PersonaMetrics struct {
	// TotalInteractions is the total number of interactions
	// +optional
	TotalInteractions int64 `json:"totalInteractions,omitempty"`

	// AverageResponseLength is the average response length in characters
	// +optional
	AverageResponseLength *int32 `json:"averageResponseLength,omitempty"`

	// AverageToolCalls is the average number of tool calls per interaction
	// +optional
	AverageToolCalls *float64 `json:"averageToolCalls,omitempty"`

	// RuleActivations tracks how often each rule triggers
	// +optional
	RuleActivations map[string]int64 `json:"ruleActivations,omitempty"`

	// TopTools lists most frequently used tools
	// +optional
	TopTools []ToolFrequency `json:"topTools,omitempty"`

	// TopTopics lists most frequently discussed topics
	// +optional
	TopTopics []TopicFrequency `json:"topTopics,omitempty"`

	// UserSatisfaction is an optional satisfaction score (0-100)
	// +optional
	UserSatisfaction *float64 `json:"userSatisfaction,omitempty"`
}

// ToolFrequency tracks tool usage frequency
type ToolFrequency struct {
	// ToolName is the name of the tool
	ToolName string `json:"toolName"`

	// Count is the number of times this tool was used
	Count int64 `json:"count"`

	// Percentage is the percentage of total tool usage
	// +optional
	Percentage *float64 `json:"percentage,omitempty"`
}

// TopicFrequency tracks topic discussion frequency
type TopicFrequency struct {
	// Topic is the topic name
	Topic string `json:"topic"`

	// Count is the number of times this topic was discussed
	Count int64 `json:"count"`

	// Percentage is the percentage of total interactions
	// +optional
	Percentage *float64 `json:"percentage,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:resource:scope=Namespaced,shortName=lpersona
// +kubebuilder:printcolumn:name="Display Name",type=string,JSONPath=`.spec.displayName`
// +kubebuilder:printcolumn:name="Tone",type=string,JSONPath=`.spec.tone`
// +kubebuilder:printcolumn:name="Phase",type=string,JSONPath=`.status.phase`
// +kubebuilder:printcolumn:name="Usage",type=integer,JSONPath=`.status.usageCount`
// +kubebuilder:printcolumn:name="Valid",type=boolean,JSONPath=`.status.validationResult.valid`
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
