package v1alpha1

import (
	corev1 "k8s.io/api/core/v1"
	networkingv1 "k8s.io/api/networking/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// LanguageClientSpec defines the desired state of LanguageClient
type LanguageClientSpec struct {
	// ClusterRef references a LanguageCluster to deploy this client into
	// +optional
	ClusterRef string `json:"clusterRef,omitempty"`

	// Image is the container image to run for this client interface
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

	// Type specifies the client interface type
	// +kubebuilder:validation:Enum=web;api;cli;widget;slack;discord;teams
	// +kubebuilder:default=web
	Type string `json:"type,omitempty"`

	// Port is the port the client interface listens on
	// +kubebuilder:validation:Minimum=1
	// +kubebuilder:validation:Maximum=65535
	// +kubebuilder:default=8080
	Port int32 `json:"port,omitempty"`

	// Replicas is the number of pod replicas to run
	// +kubebuilder:validation:Minimum=0
	// +kubebuilder:default=2
	Replicas *int32 `json:"replicas,omitempty"`

	// ModelRefs references the LanguageModel resources this client can use
	// +kubebuilder:validation:MinItems=1
	ModelRefs []ModelReference `json:"modelRefs"`

	// ToolRefs references the LanguageTool resources available to this client
	// +optional
	ToolRefs []ToolReference `json:"toolRefs,omitempty"`

	// PersonaRef references a LanguagePersona to apply by default
	// +optional
	PersonaRef *PersonaReference `json:"personaRef,omitempty"`

	// AgentRefs references LanguageAgent resources this client can invoke
	// +optional
	AgentRefs []AgentReference `json:"agentRefs,omitempty"`

	// AllowModelSelection allows users to choose which model to use
	// +kubebuilder:default=true
	AllowModelSelection bool `json:"allowModelSelection,omitempty"`

	// AllowPersonaSelection allows users to choose which persona to use
	// +kubebuilder:default=false
	AllowPersonaSelection bool `json:"allowPersonaSelection,omitempty"`

	// SessionConfig defines session management configuration
	// +optional
	SessionConfig *SessionConfigSpec `json:"sessionConfig,omitempty"`

	// Authentication defines authentication and authorization configuration
	// +optional
	Authentication *AuthenticationSpec `json:"authentication,omitempty"`

	// RateLimiting defines rate limiting per user/session
	// +optional
	RateLimiting *ClientRateLimitSpec `json:"rateLimiting,omitempty"`

	// ContentModeration defines content filtering rules
	// +optional
	ContentModeration *ContentModerationSpec `json:"contentModeration,omitempty"`

	// UIConfig defines UI customization options
	// +optional
	UIConfig *UIConfigSpec `json:"uiConfig,omitempty"`

	// Ingress defines ingress configuration for external access
	// +optional
	Ingress *IngressSpec `json:"ingress,omitempty"`

	// TLS defines TLS configuration
	// +optional
	TLS *TLSSpec `json:"tls,omitempty"`

	// CORS defines Cross-Origin Resource Sharing configuration
	// +optional
	CORS *CORSSpec `json:"cors,omitempty"`

	// Monitoring defines monitoring and observability configuration
	// +optional
	Monitoring *MonitoringSpec `json:"monitoring,omitempty"`

	// Logging defines logging configuration
	// +optional
	Logging *LoggingSpec `json:"logging,omitempty"`

	// Env contains environment variables for the client container
	// +optional
	Env []corev1.EnvVar `json:"env,omitempty"`

	// EnvFrom sources to populate environment variables
	// +optional
	EnvFrom []corev1.EnvFromSource `json:"envFrom,omitempty"`

	// Resources defines compute resource requirements
	// +optional
	Resources corev1.ResourceRequirements `json:"resources,omitempty"`

	// NodeSelector is a selector which must match a node's labels for the pod to be scheduled
	// +optional
	NodeSelector map[string]string `json:"nodeSelector,omitempty"`

	// Affinity defines pod affinity and anti-affinity rules
	// +optional
	Affinity *corev1.Affinity `json:"affinity,omitempty"`

	// Tolerations allow pods to schedule onto nodes with matching taints
	// +optional
	Tolerations []corev1.Toleration `json:"tolerations,omitempty"`

	// TopologySpreadConstraints describes how pods should spread across topology domains
	// +optional
	TopologySpreadConstraints []corev1.TopologySpreadConstraint `json:"topologySpreadConstraints,omitempty"`

	// ServiceAccountName is the name of the ServiceAccount to use for this client
	// +optional
	ServiceAccountName string `json:"serviceAccountName,omitempty"`

	// SecurityContext holds pod-level security attributes
	// +optional
	SecurityContext *corev1.PodSecurityContext `json:"securityContext,omitempty"`

	// ContainerSecurityContext holds container-level security attributes
	// +optional
	ContainerSecurityContext *corev1.SecurityContext `json:"containerSecurityContext,omitempty"`

	// VolumeMounts to mount into the client container
	// +optional
	VolumeMounts []corev1.VolumeMount `json:"volumeMounts,omitempty"`

	// Volumes to attach to the pod
	// +optional
	Volumes []corev1.Volume `json:"volumes,omitempty"`

	// LivenessProbe defines the liveness probe for the client container
	// +optional
	LivenessProbe *corev1.Probe `json:"livenessProbe,omitempty"`

	// ReadinessProbe defines the readiness probe for the client container
	// +optional
	ReadinessProbe *corev1.Probe `json:"readinessProbe,omitempty"`

	// StartupProbe defines the startup probe for the client container
	// +optional
	StartupProbe *corev1.Probe `json:"startupProbe,omitempty"`

	// ServiceType specifies the type of Service to create
	// +kubebuilder:validation:Enum=ClusterIP;NodePort;LoadBalancer
	// +kubebuilder:default=ClusterIP
	// +optional
	ServiceType corev1.ServiceType `json:"serviceType,omitempty"`

	// ServiceAnnotations are annotations to add to the Service
	// +optional
	ServiceAnnotations map[string]string `json:"serviceAnnotations,omitempty"`

	// PodAnnotations are annotations to add to the Pods
	// +optional
	PodAnnotations map[string]string `json:"podAnnotations,omitempty"`

	// PodLabels are additional labels to add to the Pods
	// +optional
	PodLabels map[string]string `json:"podLabels,omitempty"`

	// PodDisruptionBudget defines the PDB for this client
	// +optional
	PodDisruptionBudget *PodDisruptionBudgetSpec `json:"podDisruptionBudget,omitempty"`

	// UpdateStrategy defines the update strategy for the Deployment
	// +optional
	UpdateStrategy *UpdateStrategySpec `json:"updateStrategy,omitempty"`

	// HorizontalPodAutoscaler defines HPA configuration
	// +optional
	HorizontalPodAutoscaler *HPASpec `json:"horizontalPodAutoscaler,omitempty"`

	// Regions defines multi-region deployment configuration
	// +optional
	Regions []ClientRegionSpec `json:"regions,omitempty"`
}

// SessionConfigSpec defines session management configuration
type SessionConfigSpec struct {
	// Backend specifies the session storage backend
	// +kubebuilder:validation:Enum=memory;redis;postgres;dynamodb
	// +kubebuilder:default=memory
	Backend string `json:"backend,omitempty"`

	// BackendConfig contains backend-specific configuration
	// +optional
	BackendConfig *BackendConfigSpec `json:"backendConfig,omitempty"`

	// TTL is the session time-to-live
	// +kubebuilder:validation:Pattern="^[0-9]+(ns|us|µs|ms|s|m|h)$"
	// +kubebuilder:default="24h"
	TTL string `json:"ttl,omitempty"`

	// MaxMessagesPerSession limits conversation history length
	// +kubebuilder:default=100
	// +optional
	MaxMessagesPerSession *int32 `json:"maxMessagesPerSession,omitempty"`

	// EnablePersistence enables saving conversations beyond session TTL
	// +kubebuilder:default=false
	EnablePersistence bool `json:"enablePersistence,omitempty"`

	// CookieConfig defines session cookie configuration
	// +optional
	CookieConfig *CookieConfigSpec `json:"cookieConfig,omitempty"`
}

// BackendConfigSpec contains session backend configuration
type BackendConfigSpec struct {
	// Endpoint is the backend service endpoint
	// +optional
	Endpoint string `json:"endpoint,omitempty"`

	// SecretRef references credentials for the backend
	// +optional
	SecretRef *SecretReference `json:"secretRef,omitempty"`

	// Database specifies the database name (for SQL backends)
	// +optional
	Database string `json:"database,omitempty"`

	// KeyPrefix is prepended to all session keys
	// +optional
	KeyPrefix string `json:"keyPrefix,omitempty"`
}

// CookieConfigSpec defines session cookie configuration
type CookieConfigSpec struct {
	// Name is the cookie name
	// +kubebuilder:default=session_id
	Name string `json:"name,omitempty"`

	// Secure requires HTTPS
	// +kubebuilder:default=true
	Secure bool `json:"secure,omitempty"`

	// HttpOnly prevents JavaScript access
	// +kubebuilder:default=true
	HttpOnly bool `json:"httpOnly,omitempty"`

	// SameSite controls cross-site cookie behavior
	// +kubebuilder:validation:Enum=Strict;Lax;None
	// +kubebuilder:default=Lax
	SameSite string `json:"sameSite,omitempty"`

	// Domain for the cookie
	// +optional
	Domain string `json:"domain,omitempty"`

	// Path for the cookie
	// +kubebuilder:default=/
	Path string `json:"path,omitempty"`
}

// AuthenticationSpec defines authentication configuration
type AuthenticationSpec struct {
	// Enabled enables authentication
	// +kubebuilder:default=false
	Enabled bool `json:"enabled,omitempty"`

	// Providers lists authentication providers
	// +optional
	Providers []AuthProviderSpec `json:"providers,omitempty"`

	// RequireAuthentication blocks unauthenticated access
	// +kubebuilder:default=false
	RequireAuthentication bool `json:"requireAuthentication,omitempty"`

	// AllowAnonymous allows anonymous usage
	// +kubebuilder:default=true
	AllowAnonymous bool `json:"allowAnonymous,omitempty"`

	// RBAC defines role-based access control
	// +optional
	RBAC *RBACSpec `json:"rbac,omitempty"`
}

// AuthProviderSpec defines an authentication provider
type AuthProviderSpec struct {
	// Name is the provider identifier
	// +kubebuilder:validation:Required
	Name string `json:"name"`

	// Type specifies the provider type
	// +kubebuilder:validation:Enum=oauth2;oidc;saml;ldap;basic;api-key
	// +kubebuilder:validation:Required
	Type string `json:"type"`

	// Config contains provider-specific configuration
	// +optional
	Config map[string]string `json:"config,omitempty"`

	// SecretRef references credentials for this provider
	// +optional
	SecretRef *SecretReference `json:"secretRef,omitempty"`

	// Enabled indicates if this provider is active
	// +kubebuilder:default=true
	Enabled bool `json:"enabled,omitempty"`
}

// RBACSpec defines role-based access control
type RBACSpec struct {
	// Enabled enables RBAC
	// +kubebuilder:default=false
	Enabled bool `json:"enabled,omitempty"`

	// Roles defines available roles and their permissions
	// +optional
	Roles []RoleSpec `json:"roles,omitempty"`

	// DefaultRole is assigned to new users
	// +kubebuilder:default=user
	DefaultRole string `json:"defaultRole,omitempty"`
}

// RoleSpec defines a role and its permissions
type RoleSpec struct {
	// Name is the role identifier
	// +kubebuilder:validation:Required
	Name string `json:"name"`

	// Permissions lists what this role can do
	// +optional
	Permissions []string `json:"permissions,omitempty"`

	// ModelAccess lists models this role can use
	// +optional
	ModelAccess []string `json:"modelAccess,omitempty"`

	// ToolAccess lists tools this role can use
	// +optional
	ToolAccess []string `json:"toolAccess,omitempty"`

	// PersonaAccess lists personas this role can use
	// +optional
	PersonaAccess []string `json:"personaAccess,omitempty"`

	// AgentAccess lists agents this role can invoke
	// +optional
	AgentAccess []string `json:"agentAccess,omitempty"`
}

// ClientRateLimitSpec defines client-side rate limiting
type ClientRateLimitSpec struct {
	// RequestsPerMinute limits requests per user per minute
	// +optional
	RequestsPerMinute *int32 `json:"requestsPerMinute,omitempty"`

	// RequestsPerHour limits requests per user per hour
	// +optional
	RequestsPerHour *int32 `json:"requestsPerHour,omitempty"`

	// RequestsPerDay limits requests per user per day
	// +optional
	RequestsPerDay *int32 `json:"requestsPerDay,omitempty"`

	// TokensPerMinute limits tokens per user per minute
	// +optional
	TokensPerMinute *int32 `json:"tokensPerMinute,omitempty"`

	// TokensPerDay limits tokens per user per day
	// +optional
	TokensPerDay *int32 `json:"tokensPerDay,omitempty"`

	// CostPerDay limits cost per user per day (in USD cents)
	// +optional
	CostPerDay *int32 `json:"costPerDay,omitempty"`

	// ConcurrentSessions limits concurrent sessions per user
	// +optional
	ConcurrentSessions *int32 `json:"concurrentSessions,omitempty"`

	// Strategy defines rate limiting strategy
	// +kubebuilder:validation:Enum=fixed-window;sliding-window;token-bucket
	// +kubebuilder:default=sliding-window
	Strategy string `json:"strategy,omitempty"`
}

// ContentModerationSpec defines content filtering
type ContentModerationSpec struct {
	// Enabled enables content moderation
	// +kubebuilder:default=false
	Enabled bool `json:"enabled,omitempty"`

	// FilterProfanity filters profane content
	// +kubebuilder:default=false
	FilterProfanity bool `json:"filterProfanity,omitempty"`

	// FilterPII filters personally identifiable information
	// +kubebuilder:default=false
	FilterPII bool `json:"filterPII,omitempty"`

	// FilterToxic filters toxic content
	// +kubebuilder:default=false
	FilterToxic bool `json:"filterToxic,omitempty"`

	// CustomFilters are custom content filters
	// +optional
	CustomFilters []ContentFilterSpec `json:"customFilters,omitempty"`

	// Action defines what to do when content is flagged
	// +kubebuilder:validation:Enum=block;warn;log
	// +kubebuilder:default=warn
	Action string `json:"action,omitempty"`
}

// ContentFilterSpec defines a custom content filter
type ContentFilterSpec struct {
	// Name is the filter identifier
	// +kubebuilder:validation:Required
	Name string `json:"name"`

	// Pattern is a regex pattern to match
	// +kubebuilder:validation:Required
	Pattern string `json:"pattern"`

	// Action overrides the default action for this filter
	// +kubebuilder:validation:Enum=block;warn;log
	// +optional
	Action string `json:"action,omitempty"`

	// Message is shown when content matches
	// +optional
	Message string `json:"message,omitempty"`
}

// UIConfigSpec defines UI customization
type UIConfigSpec struct {
	// Title is the application title
	// +optional
	Title string `json:"title,omitempty"`

	// Description is the application description
	// +optional
	Description string `json:"description,omitempty"`

	// Logo is a URL to the application logo
	// +optional
	Logo string `json:"logo,omitempty"`

	// Theme defines the UI theme
	// +kubebuilder:validation:Enum=light;dark;auto
	// +kubebuilder:default=auto
	Theme string `json:"theme,omitempty"`

	// PrimaryColor is the primary brand color
	// +optional
	PrimaryColor string `json:"primaryColor,omitempty"`

	// SecondaryColor is the secondary brand color
	// +optional
	SecondaryColor string `json:"secondaryColor,omitempty"`

	// CustomCSS is a URL to custom CSS
	// +optional
	CustomCSS string `json:"customCSS,omitempty"`

	// CustomJS is a URL to custom JavaScript
	// +optional
	CustomJS string `json:"customJS,omitempty"`

	// Features defines enabled UI features
	// +optional
	Features *UIFeaturesSpec `json:"features,omitempty"`

	// Footer defines footer content
	// +optional
	Footer string `json:"footer,omitempty"`

	// PrivacyPolicyURL is a link to the privacy policy
	// +optional
	PrivacyPolicyURL string `json:"privacyPolicyURL,omitempty"`

	// TermsOfServiceURL is a link to terms of service
	// +optional
	TermsOfServiceURL string `json:"termsOfServiceURL,omitempty"`
}

// UIFeaturesSpec defines enabled UI features
type UIFeaturesSpec struct {
	// ShowModelSelector shows model selection dropdown
	// +kubebuilder:default=true
	ShowModelSelector bool `json:"showModelSelector,omitempty"`

	// ShowPersonaSelector shows persona selection dropdown
	// +kubebuilder:default=false
	ShowPersonaSelector bool `json:"showPersonaSelector,omitempty"`

	// ShowToolUsage shows when tools are being used
	// +kubebuilder:default=true
	ShowToolUsage bool `json:"showToolUsage,omitempty"`

	// ShowThinkingProcess shows model reasoning
	// +kubebuilder:default=false
	ShowThinkingProcess bool `json:"showThinkingProcess,omitempty"`

	// EnableFileUpload enables file uploads
	// +kubebuilder:default=false
	EnableFileUpload bool `json:"enableFileUpload,omitempty"`

	// EnableVoiceInput enables voice input
	// +kubebuilder:default=false
	EnableVoiceInput bool `json:"enableVoiceInput,omitempty"`

	// EnableExport enables conversation export
	// +kubebuilder:default=true
	EnableExport bool `json:"enableExport,omitempty"`

	// EnableSharing enables conversation sharing
	// +kubebuilder:default=false
	EnableSharing bool `json:"enableSharing,omitempty"`

	// ShowTimestamps shows message timestamps
	// +kubebuilder:default=true
	ShowTimestamps bool `json:"showTimestamps,omitempty"`

	// ShowCosts shows token/cost usage
	// +kubebuilder:default=false
	ShowCosts bool `json:"showCosts,omitempty"`
}

// IngressSpec defines ingress configuration
type IngressSpec struct {
	// Enabled creates an Ingress resource
	// +kubebuilder:default=false
	Enabled bool `json:"enabled,omitempty"`

	// ClassName specifies the IngressClass to use
	// +optional
	ClassName *string `json:"className,omitempty"`

	// Hosts lists the hostnames for this ingress
	// +kubebuilder:validation:MinItems=1
	// +optional
	Hosts []string `json:"hosts,omitempty"`

	// Annotations are annotations to add to the Ingress
	// +optional
	Annotations map[string]string `json:"annotations,omitempty"`

	// TLS defines TLS configuration for ingress
	// +optional
	TLS []networkingv1.IngressTLS `json:"tls,omitempty"`

	// Path is the URL path for the ingress
	// +kubebuilder:default=/
	Path string `json:"path,omitempty"`

	// PathType specifies the path matching type
	// +kubebuilder:validation:Enum=Exact;Prefix;ImplementationSpecific
	// +kubebuilder:default=Prefix
	PathType string `json:"pathType,omitempty"`
}

// TLSSpec defines TLS configuration
type TLSSpec struct {
	// Enabled enables TLS
	// +kubebuilder:default=false
	Enabled bool `json:"enabled,omitempty"`

	// SecretRef references the TLS certificate secret
	// +optional
	SecretRef *SecretReference `json:"secretRef,omitempty"`

	// CertManager enables cert-manager integration
	// +kubebuilder:default=false
	CertManager bool `json:"certManager,omitempty"`

	// CertManagerIssuer specifies the cert-manager issuer
	// +optional
	CertManagerIssuer string `json:"certManagerIssuer,omitempty"`
}

// CORSSpec defines CORS configuration
type CORSSpec struct {
	// Enabled enables CORS
	// +kubebuilder:default=false
	Enabled bool `json:"enabled,omitempty"`

	// AllowedOrigins lists allowed origins
	// +optional
	AllowedOrigins []string `json:"allowedOrigins,omitempty"`

	// AllowedMethods lists allowed HTTP methods
	// +optional
	AllowedMethods []string `json:"allowedMethods,omitempty"`

	// AllowedHeaders lists allowed headers
	// +optional
	AllowedHeaders []string `json:"allowedHeaders,omitempty"`

	// AllowCredentials allows credentials
	// +kubebuilder:default=false
	AllowCredentials bool `json:"allowCredentials,omitempty"`

	// MaxAge is preflight cache duration in seconds
	// +optional
	MaxAge *int32 `json:"maxAge,omitempty"`
}

// MonitoringSpec defines monitoring configuration
type MonitoringSpec struct {
	// Enabled enables monitoring
	// +kubebuilder:default=true
	Enabled bool `json:"enabled,omitempty"`

	// Prometheus enables Prometheus metrics
	// +kubebuilder:default=true
	Prometheus bool `json:"prometheus,omitempty"`

	// MetricsPath is the metrics endpoint path
	// +kubebuilder:default=/metrics
	MetricsPath string `json:"metricsPath,omitempty"`

	// ServiceMonitor creates a ServiceMonitor resource
	// +kubebuilder:default=false
	ServiceMonitor bool `json:"serviceMonitor,omitempty"`

	// AdditionalLabels are labels for the ServiceMonitor
	// +optional
	AdditionalLabels map[string]string `json:"additionalLabels,omitempty"`
}

// LoggingSpec defines logging configuration
type LoggingSpec struct {
	// Level sets the log level
	// +kubebuilder:validation:Enum=debug;info;warn;error
	// +kubebuilder:default=info
	Level string `json:"level,omitempty"`

	// Format sets the log format
	// +kubebuilder:validation:Enum=json;text
	// +kubebuilder:default=json
	Format string `json:"format,omitempty"`

	// LogRequests enables request logging
	// +kubebuilder:default=true
	LogRequests bool `json:"logRequests,omitempty"`

	// LogResponses enables response logging
	// +kubebuilder:default=false
	LogResponses bool `json:"logResponses,omitempty"`

	// SanitizeSecrets removes secrets from logs
	// +kubebuilder:default=true
	SanitizeSecrets bool `json:"sanitizeSecrets,omitempty"`
}

// HPASpec defines HorizontalPodAutoscaler configuration
type HPASpec struct {
	// Enabled creates an HPA resource
	// +kubebuilder:default=false
	Enabled bool `json:"enabled,omitempty"`

	// MinReplicas is the minimum number of replicas
	// +kubebuilder:validation:Minimum=1
	// +kubebuilder:default=2
	MinReplicas *int32 `json:"minReplicas,omitempty"`

	// MaxReplicas is the maximum number of replicas
	// +kubebuilder:validation:Minimum=1
	// +kubebuilder:default=10
	MaxReplicas int32 `json:"maxReplicas"`

	// TargetCPUUtilizationPercentage is the target CPU utilization
	// +optional
	TargetCPUUtilizationPercentage *int32 `json:"targetCPUUtilizationPercentage,omitempty"`

	// TargetMemoryUtilizationPercentage is the target memory utilization
	// +optional
	TargetMemoryUtilizationPercentage *int32 `json:"targetMemoryUtilizationPercentage,omitempty"`

	// CustomMetrics are custom metrics for scaling
	// +optional
	CustomMetrics []CustomMetricSpec `json:"customMetrics,omitempty"`
}

// CustomMetricSpec defines a custom autoscaling metric
type CustomMetricSpec struct {
	// Name is the metric name
	// +kubebuilder:validation:Required
	Name string `json:"name"`

	// Type is the metric type
	// +kubebuilder:validation:Enum=Pods;Object;External
	// +kubebuilder:validation:Required
	Type string `json:"type"`

	// Target is the target value
	// +kubebuilder:validation:Required
	Target string `json:"target"`
}

// ClientRegionSpec defines region-specific configuration
type ClientRegionSpec struct {
	// Name is the region identifier
	// +kubebuilder:validation:Required
	Name string `json:"name"`

	// Enabled indicates if this region is active
	// +kubebuilder:default=true
	Enabled bool `json:"enabled,omitempty"`

	// Replicas overrides the default replica count for this region
	// +optional
	Replicas *int32 `json:"replicas,omitempty"`

	// ModelRefs overrides model references for this region
	// +optional
	ModelRefs []ModelReference `json:"modelRefs,omitempty"`

	// NodeSelector overrides node selector for this region
	// +optional
	NodeSelector map[string]string `json:"nodeSelector,omitempty"`

	// Affinity overrides affinity for this region
	// +optional
	Affinity *corev1.Affinity `json:"affinity,omitempty"`

	// Ingress overrides ingress configuration for this region
	// +optional
	Ingress *IngressSpec `json:"ingress,omitempty"`
}

// AgentReference references a LanguageAgent
type AgentReference struct {
	// Name is the LanguageAgent name
	// +kubebuilder:validation:Required
	Name string `json:"name"`

	// Namespace is the LanguageAgent namespace
	// +optional
	Namespace string `json:"namespace,omitempty"`

	// Enabled indicates if this agent reference is active
	// +kubebuilder:default=true
	Enabled bool `json:"enabled,omitempty"`

	// DisplayName is shown in the UI
	// +optional
	DisplayName string `json:"displayName,omitempty"`

	// Description is shown in the UI
	// +optional
	Description string `json:"description,omitempty"`

	// Icon is shown in the UI
	// +optional
	Icon string `json:"icon,omitempty"`
}

// LanguageClientStatus defines the observed state of LanguageClient
type LanguageClientStatus struct {
	// ObservedGeneration reflects the generation of the most recently observed LanguageClient
	// +optional
	ObservedGeneration int64 `json:"observedGeneration,omitempty"`

	// Phase represents the current phase (Pending, Running, Failed, Unknown)
	// +kubebuilder:validation:Enum=Pending;Running;Failed;Unknown;Updating
	// +optional
	Phase string `json:"phase,omitempty"`

	// Conditions represent the latest available observations of the client's state
	// +optional
	// +listType=map
	// +listMapKey=type
	// +patchMergeKey=type
	// +patchStrategy=merge
	Conditions []metav1.Condition `json:"conditions,omitempty" patchStrategy:"merge" patchMergeKey:"type"`

	// URL is the primary access URL for this client
	// +optional
	URL string `json:"url,omitempty"`

	// RegionalURLs lists URLs per region
	// +optional
	RegionalURLs map[string]string `json:"regionalURLs,omitempty"`

	// ReadyReplicas is the number of pods ready and passing health checks
	// +optional
	ReadyReplicas int32 `json:"readyReplicas,omitempty"`

	// AvailableReplicas is the number of pods with at least one available condition
	// +optional
	AvailableReplicas int32 `json:"availableReplicas,omitempty"`

	// UpdatedReplicas is the number of pods with the desired spec
	// +optional
	UpdatedReplicas int32 `json:"updatedReplicas,omitempty"`

	// UnavailableReplicas is the number of unavailable pods
	// +optional
	UnavailableReplicas int32 `json:"unavailableReplicas,omitempty"`

	// ActiveSessions is the current number of active sessions
	// +optional
	ActiveSessions int32 `json:"activeSessions,omitempty"`

	// TotalUsers is the total number of users who have used this client
	// +optional
	TotalUsers int64 `json:"totalUsers,omitempty"`

	// Metrics contains usage metrics
	// +optional
	Metrics *ClientMetrics `json:"metrics,omitempty"`

	// RegionStatus tracks status per region
	// +optional
	RegionStatus []RegionStatus `json:"regionStatus,omitempty"`

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

// ClientMetrics contains client usage metrics
type ClientMetrics struct {
	// TotalRequests is the total number of requests
	// +optional
	TotalRequests int64 `json:"totalRequests,omitempty"`

	// TotalSessions is the total number of sessions created
	// +optional
	TotalSessions int64 `json:"totalSessions,omitempty"`

	// TotalMessages is the total number of messages processed
	// +optional
	TotalMessages int64 `json:"totalMessages,omitempty"`

	// TotalTokens is the total number of tokens processed
	// +optional
	TotalTokens int64 `json:"totalTokens,omitempty"`

	// TotalCost is the total cost in USD cents
	// +optional
	TotalCost int64 `json:"totalCost,omitempty"`

	// AverageResponseTime is the average response time in milliseconds
	// +optional
	AverageResponseTime *int32 `json:"averageResponseTime,omitempty"`

	// P50ResponseTime is the 50th percentile response time in milliseconds
	// +optional
	P50ResponseTime *int32 `json:"p50ResponseTime,omitempty"`

	// P95ResponseTime is the 95th percentile response time in milliseconds
	// +optional
	P95ResponseTime *int32 `json:"p95ResponseTime,omitempty"`

	// P99ResponseTime is the 99th percentile response time in milliseconds
	// +optional
	P99ResponseTime *int32 `json:"p99ResponseTime,omitempty"`

	// ErrorRate is the percentage of failed requests
	// +optional
	ErrorRate *float64 `json:"errorRate,omitempty"`

	// TopModels lists most frequently used models
	// +optional
	TopModels []ModelUsageMetric `json:"topModels,omitempty"`

	// TopTools lists most frequently used tools
	// +optional
	TopTools []ToolFrequency `json:"topTools,omitempty"`

	// TopPersonas lists most frequently used personas
	// +optional
	TopPersonas []PersonaUsageMetric `json:"topPersonas,omitempty"`

	// UserRetention is the percentage of returning users
	// +optional
	UserRetention *float64 `json:"userRetention,omitempty"`

	// AverageSessionLength is the average session duration in seconds
	// +optional
	AverageSessionLength *int32 `json:"averageSessionLength,omitempty"`

	// RateLimitHits is the number of rate limit hits
	// +optional
	RateLimitHits int64 `json:"rateLimitHits,omitempty"`

	// ContentModerationFlags is the number of content moderation flags
	// +optional
	ContentModerationFlags int64 `json:"contentModerationFlags,omitempty"`
}

// ModelUsageMetric tracks model usage frequency
type ModelUsageMetric struct {
	// ModelName is the name of the model
	ModelName string `json:"modelName"`

	// Count is the number of times this model was used
	Count int64 `json:"count"`

	// Percentage is the percentage of total usage
	// +optional
	Percentage *float64 `json:"percentage,omitempty"`

	// TotalTokens is tokens used with this model
	// +optional
	TotalTokens int64 `json:"totalTokens,omitempty"`

	// TotalCost is cost incurred with this model (USD cents)
	// +optional
	TotalCost int64 `json:"totalCost,omitempty"`
}

// PersonaUsageMetric tracks persona usage frequency
type PersonaUsageMetric struct {
	// PersonaName is the name of the persona
	PersonaName string `json:"personaName"`

	// Count is the number of times this persona was used
	Count int64 `json:"count"`

	// Percentage is the percentage of total usage
	// +optional
	Percentage *float64 `json:"percentage,omitempty"`
}

// RegionStatus tracks status for a specific region
type RegionStatus struct {
	// Name is the region identifier
	Name string `json:"name"`

	// Phase is the current phase for this region
	Phase string `json:"phase,omitempty"`

	// ReadyReplicas is ready replicas in this region
	ReadyReplicas int32 `json:"readyReplicas,omitempty"`

	// URL is the access URL for this region
	// +optional
	URL string `json:"url,omitempty"`

	// LastUpdateTime is the last update for this region
	// +optional
	LastUpdateTime *metav1.Time `json:"lastUpdateTime,omitempty"`

	// Message provides region-specific status details
	// +optional
	Message string `json:"message,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:resource:scope=Namespaced,shortName=lclient
// +kubebuilder:printcolumn:name="Type",type=string,JSONPath=`.spec.type`
// +kubebuilder:printcolumn:name="Phase",type=string,JSONPath=`.status.phase`
// +kubebuilder:printcolumn:name="URL",type=string,JSONPath=`.status.url`
// +kubebuilder:printcolumn:name="Sessions",type=integer,JSONPath=`.status.activeSessions`
// +kubebuilder:printcolumn:name="Users",type=integer,JSONPath=`.status.totalUsers`
// +kubebuilder:printcolumn:name="Age",type=date,JSONPath=`.metadata.creationTimestamp`

// LanguageClient is the Schema for the languageclients API
type LanguageClient struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   LanguageClientSpec   `json:"spec,omitempty"`
	Status LanguageClientStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// LanguageClientList contains a list of LanguageClient
type LanguageClientList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []LanguageClient `json:"items"`
}

func init() {
	SchemeBuilder.Register(&LanguageClient{}, &LanguageClientList{})
}
