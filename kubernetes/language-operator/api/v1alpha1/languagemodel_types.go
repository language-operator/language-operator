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

	// Configuration contains provider-specific configuration
	// +optional
	Configuration *ProviderConfiguration `json:"configuration,omitempty"`

	// RateLimits defines rate limiting configuration
	// +optional
	RateLimits *RateLimitSpec `json:"rateLimits,omitempty"`

	// Timeout specifies request timeout duration (e.g., "5m", "30s")
	// +kubebuilder:validation:Pattern=`^[0-9]+(ns|us|µs|ms|s|m|h)$`
	// +kubebuilder:default="5m"
	// +optional
	Timeout string `json:"timeout,omitempty"`

	// RetryPolicy defines retry behavior for failed requests
	// +optional
	RetryPolicy *RetryPolicySpec `json:"retryPolicy,omitempty"`

	// Fallbacks is an ordered list of fallback models to use if this model fails
	// +optional
	Fallbacks []ModelFallbackSpec `json:"fallbacks,omitempty"`

	// LoadBalancing defines load balancing strategy across multiple endpoints
	// +optional
	LoadBalancing *LoadBalancingSpec `json:"loadBalancing,omitempty"`

	// Caching defines response caching configuration
	// +optional
	Caching *CachingSpec `json:"caching,omitempty"`

	// Observability defines monitoring and tracing configuration
	// +optional
	Observability *ObservabilitySpec `json:"observability,omitempty"`

	// CostTracking enables cost tracking for this model
	// +optional
	CostTracking *CostTrackingSpec `json:"costTracking,omitempty"`

	// Regions specifies which regions this model is deployed in (for multi-region)
	// +optional
	Regions []RegionSpec `json:"regions,omitempty"`
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

// ProviderConfiguration contains provider-specific settings
type ProviderConfiguration struct {
	// MaxTokens is the maximum tokens for responses
	// +optional
	MaxTokens *int32 `json:"maxTokens,omitempty"`

	// Temperature controls randomness (0.0 to 2.0)
	// +optional
	Temperature *float64 `json:"temperature,omitempty"`

	// TopP controls nucleus sampling
	// +optional
	TopP *float64 `json:"topP,omitempty"`

	// FrequencyPenalty penalizes frequent tokens (-2.0 to 2.0)
	// +optional
	FrequencyPenalty *float64 `json:"frequencyPenalty,omitempty"`

	// PresencePenalty penalizes tokens based on presence (-2.0 to 2.0)
	// +optional
	PresencePenalty *float64 `json:"presencePenalty,omitempty"`

	// StopSequences are sequences that stop generation
	// +optional
	StopSequences []string `json:"stopSequences,omitempty"`

	// AdditionalParameters for provider-specific options
	// +optional
	AdditionalParameters map[string]string `json:"additionalParameters,omitempty"`
}

// RateLimitSpec defines rate limiting configuration
type RateLimitSpec struct {
	// RequestsPerMinute limits requests per minute
	// +optional
	RequestsPerMinute *int32 `json:"requestsPerMinute,omitempty"`

	// TokensPerMinute limits tokens per minute
	// +optional
	TokensPerMinute *int32 `json:"tokensPerMinute,omitempty"`

	// ConcurrentRequests limits concurrent requests
	// +optional
	ConcurrentRequests *int32 `json:"concurrentRequests,omitempty"`
}

// RetryPolicySpec defines retry behavior
type RetryPolicySpec struct {
	// MaxAttempts is the maximum number of retry attempts
	// +kubebuilder:validation:Minimum=0
	// +kubebuilder:validation:Maximum=10
	// +kubebuilder:default=3
	// +optional
	MaxAttempts *int32 `json:"maxAttempts,omitempty"`

	// InitialBackoff is the initial backoff duration (e.g., "1s")
	// +kubebuilder:validation:Pattern=`^[0-9]+(ns|us|µs|ms|s|m|h)$`
	// +kubebuilder:default="1s"
	// +optional
	InitialBackoff string `json:"initialBackoff,omitempty"`

	// MaxBackoff is the maximum backoff duration (e.g., "30s")
	// +kubebuilder:validation:Pattern=`^[0-9]+(ns|us|µs|ms|s|m|h)$`
	// +kubebuilder:default="30s"
	// +optional
	MaxBackoff string `json:"maxBackoff,omitempty"`

	// BackoffMultiplier is the multiplier for exponential backoff
	// +kubebuilder:default=2.0
	// +optional
	BackoffMultiplier *float64 `json:"backoffMultiplier,omitempty"`

	// RetryableStatusCodes are HTTP status codes that trigger retry
	// +optional
	RetryableStatusCodes []int32 `json:"retryableStatusCodes,omitempty"`
}

// ModelFallbackSpec defines a fallback model
type ModelFallbackSpec struct {
	// ModelRef is a reference to another LanguageModel
	// +kubebuilder:validation:Required
	ModelRef string `json:"modelRef"`

	// Conditions specifies when to use this fallback
	// +optional
	Conditions []string `json:"conditions,omitempty"`
}

// LoadBalancingSpec defines load balancing configuration
type LoadBalancingSpec struct {
	// Strategy specifies the load balancing strategy
	// +kubebuilder:validation:Enum=round-robin;least-connections;random;weighted;latency-based
	// +kubebuilder:default=round-robin
	Strategy string `json:"strategy,omitempty"`

	// Endpoints is a list of endpoint configurations for load balancing
	// +optional
	Endpoints []EndpointSpec `json:"endpoints,omitempty"`

	// HealthCheck defines health checking for endpoints
	// +optional
	HealthCheck *HealthCheckSpec `json:"healthCheck,omitempty"`
}

// EndpointSpec defines an endpoint for load balancing
type EndpointSpec struct {
	// URL is the endpoint URL
	// +kubebuilder:validation:Required
	URL string `json:"url"`

	// Weight for weighted load balancing
	// +kubebuilder:validation:Minimum=0
	// +kubebuilder:default=100
	// +optional
	Weight *int32 `json:"weight,omitempty"`

	// Region specifies the region for this endpoint
	// +optional
	Region string `json:"region,omitempty"`

	// Priority for priority-based routing (lower is higher priority)
	// +optional
	Priority *int32 `json:"priority,omitempty"`
}

// HealthCheckSpec defines health checking configuration
type HealthCheckSpec struct {
	// Enabled enables health checking
	// +kubebuilder:default=true
	Enabled bool `json:"enabled,omitempty"`

	// Interval is the health check interval (e.g., "30s")
	// +kubebuilder:validation:Pattern=`^[0-9]+(ns|us|µs|ms|s|m|h)$`
	// +kubebuilder:default="30s"
	// +optional
	Interval string `json:"interval,omitempty"`

	// Timeout is the health check timeout (e.g., "5s")
	// +kubebuilder:validation:Pattern=`^[0-9]+(ns|us|µs|ms|s|m|h)$`
	// +kubebuilder:default="5s"
	// +optional
	Timeout string `json:"timeout,omitempty"`

	// UnhealthyThreshold is the number of failures before marking unhealthy
	// +kubebuilder:validation:Minimum=1
	// +kubebuilder:default=3
	// +optional
	UnhealthyThreshold *int32 `json:"unhealthyThreshold,omitempty"`

	// HealthyThreshold is the number of successes before marking healthy
	// +kubebuilder:validation:Minimum=1
	// +kubebuilder:default=2
	// +optional
	HealthyThreshold *int32 `json:"healthyThreshold,omitempty"`
}

// CachingSpec defines response caching configuration
type CachingSpec struct {
	// Enabled enables response caching
	// +kubebuilder:default=false
	Enabled bool `json:"enabled,omitempty"`

	// TTL is the cache time-to-live (e.g., "5m")
	// +kubebuilder:validation:Pattern=`^[0-9]+(ns|us|µs|ms|s|m|h)$`
	// +optional
	TTL string `json:"ttl,omitempty"`

	// MaxSize is the maximum cache size in MB
	// +optional
	MaxSize *int32 `json:"maxSize,omitempty"`

	// Backend specifies the caching backend (memory, redis, etc.)
	// +kubebuilder:validation:Enum=memory;redis;memcached
	// +kubebuilder:default=memory
	// +optional
	Backend string `json:"backend,omitempty"`
}

// ObservabilitySpec defines monitoring and tracing
type ObservabilitySpec struct {
	// Metrics enables metrics collection
	// +kubebuilder:default=true
	Metrics bool `json:"metrics,omitempty"`

	// Tracing enables distributed tracing
	// +kubebuilder:default=false
	Tracing bool `json:"tracing,omitempty"`

	// Logging defines logging configuration
	// +optional
	Logging *ModelLoggingSpec `json:"logging,omitempty"`
}

// ModelLoggingSpec defines logging configuration
type ModelLoggingSpec struct {
	// Level is the log level (debug, info, warn, error)
	// +kubebuilder:validation:Enum=debug;info;warn;error
	// +kubebuilder:default=info
	Level string `json:"level,omitempty"`

	// LogRequests enables request logging
	// +kubebuilder:default=true
	LogRequests bool `json:"logRequests,omitempty"`

	// LogResponses enables response logging
	// +kubebuilder:default=false
	LogResponses bool `json:"logResponses,omitempty"`
}

// CostTrackingSpec defines cost tracking configuration
type CostTrackingSpec struct {
	// Enabled enables cost tracking
	// +kubebuilder:default=false
	Enabled bool `json:"enabled,omitempty"`

	// Currency is the currency for cost tracking (e.g., "USD")
	// +kubebuilder:default="USD"
	// +optional
	Currency string `json:"currency,omitempty"`

	// InputTokenCost is the cost per 1000 input tokens
	// +optional
	InputTokenCost *float64 `json:"inputTokenCost,omitempty"`

	// OutputTokenCost is the cost per 1000 output tokens
	// +optional
	OutputTokenCost *float64 `json:"outputTokenCost,omitempty"`
}

// RegionSpec defines a region configuration
type RegionSpec struct {
	// Name is the region name (e.g., "us-east-1", "eu-west-1")
	// +kubebuilder:validation:Required
	Name string `json:"name"`

	// Endpoint is the region-specific endpoint URL
	// +optional
	Endpoint string `json:"endpoint,omitempty"`

	// Priority for region routing (lower is higher priority)
	// +optional
	Priority *int32 `json:"priority,omitempty"`

	// Enabled indicates if this region is enabled
	// +kubebuilder:default=true
	Enabled bool `json:"enabled,omitempty"`
}

// LanguageModelStatus defines the observed state of LanguageModel
type LanguageModelStatus struct {
	// ObservedGeneration reflects the generation of the most recently observed LanguageModel
	// +optional
	ObservedGeneration int64 `json:"observedGeneration,omitempty"`

	// Phase represents the current phase (Ready, NotReady, Error, Configuring)
	// +kubebuilder:validation:Enum=Ready;NotReady;Error;Configuring;Degraded
	// +optional
	Phase string `json:"phase,omitempty"`

	// Conditions represent the latest available observations of the model's state
	// +optional
	// +listType=map
	// +listMapKey=type
	// +patchMergeKey=type
	// +patchStrategy=merge
	Conditions []metav1.Condition `json:"conditions,omitempty" patchStrategy:"merge" patchMergeKey:"type"`

	// Healthy indicates if the model is healthy and available
	// +optional
	Healthy bool `json:"healthy,omitempty"`

	// LastHealthCheck is the timestamp of the last health check
	// +optional
	LastHealthCheck *metav1.Time `json:"lastHealthCheck,omitempty"`

	// EndpointStatus shows status of each load-balanced endpoint
	// +optional
	EndpointStatus []EndpointStatusSpec `json:"endpointStatus,omitempty"`

	// RegionStatus shows status of each region
	// +optional
	RegionStatus []RegionStatusSpec `json:"regionStatus,omitempty"`

	// Metrics contains usage metrics
	// +optional
	Metrics *ModelMetrics `json:"metrics,omitempty"`

	// CostMetrics contains cost tracking data
	// +optional
	CostMetrics *CostMetrics `json:"costMetrics,omitempty"`

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

// EndpointStatusSpec shows the status of a load-balanced endpoint
type EndpointStatusSpec struct {
	// URL is the endpoint URL
	URL string `json:"url"`

	// Healthy indicates if the endpoint is healthy
	Healthy bool `json:"healthy"`

	// LastCheck is the timestamp of the last health check
	// +optional
	LastCheck *metav1.Time `json:"lastCheck,omitempty"`

	// FailureCount is the number of consecutive failures
	// +optional
	FailureCount int32 `json:"failureCount,omitempty"`

	// Latency is the average latency in milliseconds
	// +optional
	Latency *int32 `json:"latency,omitempty"`
}

// RegionStatusSpec shows the status of a region
type RegionStatusSpec struct {
	// Name is the region name
	Name string `json:"name"`

	// Available indicates if the region is available
	Available bool `json:"available"`

	// Latency is the average latency to this region in milliseconds
	// +optional
	Latency *int32 `json:"latency,omitempty"`

	// LastCheck is the timestamp of the last check
	// +optional
	LastCheck *metav1.Time `json:"lastCheck,omitempty"`
}

// ModelMetrics contains usage metrics
type ModelMetrics struct {
	// TotalRequests is the total number of requests
	// +optional
	TotalRequests int64 `json:"totalRequests,omitempty"`

	// SuccessfulRequests is the number of successful requests
	// +optional
	SuccessfulRequests int64 `json:"successfulRequests,omitempty"`

	// FailedRequests is the number of failed requests
	// +optional
	FailedRequests int64 `json:"failedRequests,omitempty"`

	// TotalTokens is the total number of tokens processed
	// +optional
	TotalTokens int64 `json:"totalTokens,omitempty"`

	// InputTokens is the total number of input tokens
	// +optional
	InputTokens int64 `json:"inputTokens,omitempty"`

	// OutputTokens is the total number of output tokens
	// +optional
	OutputTokens int64 `json:"outputTokens,omitempty"`

	// AverageLatency is the average request latency in milliseconds
	// +optional
	AverageLatency *int32 `json:"averageLatency,omitempty"`

	// P95Latency is the 95th percentile latency in milliseconds
	// +optional
	P95Latency *int32 `json:"p95Latency,omitempty"`

	// P99Latency is the 99th percentile latency in milliseconds
	// +optional
	P99Latency *int32 `json:"p99Latency,omitempty"`
}

// CostMetrics contains cost tracking data
type CostMetrics struct {
	// TotalCost is the total cost incurred
	// +optional
	TotalCost *float64 `json:"totalCost,omitempty"`

	// InputTokenCost is the cost for input tokens
	// +optional
	InputTokenCost *float64 `json:"inputTokenCost,omitempty"`

	// OutputTokenCost is the cost for output tokens
	// +optional
	OutputTokenCost *float64 `json:"outputTokenCost,omitempty"`

	// Currency is the currency for cost metrics
	// +optional
	Currency string `json:"currency,omitempty"`

	// LastReset is when cost metrics were last reset
	// +optional
	LastReset *metav1.Time `json:"lastReset,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:resource:scope=Namespaced,shortName=lmodel
// +kubebuilder:printcolumn:name="Provider",type=string,JSONPath=`.spec.provider`
// +kubebuilder:printcolumn:name="Model",type=string,JSONPath=`.spec.modelName`
// +kubebuilder:printcolumn:name="Phase",type=string,JSONPath=`.status.phase`
// +kubebuilder:printcolumn:name="Healthy",type=boolean,JSONPath=`.status.healthy`
// +kubebuilder:printcolumn:name="Requests",type=integer,JSONPath=`.status.metrics.totalRequests`
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
