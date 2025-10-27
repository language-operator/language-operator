package v1alpha1

import (
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// LanguageToolSpec defines the desired state of LanguageTool
type LanguageToolSpec struct {
	// Image is the container image to run for this tool
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

	// Type specifies the tool protocol type (e.g., "mcp", "openapi")
	// +kubebuilder:validation:Enum=mcp;openapi
	// +kubebuilder:default=mcp
	Type string `json:"type,omitempty"`

	// Port is the port the tool listens on
	// +kubebuilder:validation:Minimum=1
	// +kubebuilder:validation:Maximum=65535
	// +kubebuilder:default=8080
	Port int32 `json:"port,omitempty"`

	// Replicas is the number of pod replicas to run
	// +kubebuilder:validation:Minimum=0
	// +kubebuilder:default=1
	Replicas *int32 `json:"replicas,omitempty"`

	// Env contains environment variables for the tool container
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

	// ServiceAccountName is the name of the ServiceAccount to use for this tool
	// +optional
	ServiceAccountName string `json:"serviceAccountName,omitempty"`

	// SecurityContext holds pod-level security attributes
	// +optional
	SecurityContext *corev1.PodSecurityContext `json:"securityContext,omitempty"`

	// VolumeMounts to mount into the tool container
	// +optional
	VolumeMounts []corev1.VolumeMount `json:"volumeMounts,omitempty"`

	// Volumes to attach to the pod
	// +optional
	Volumes []corev1.Volume `json:"volumes,omitempty"`

	// LivenessProbe defines the liveness probe for the tool container
	// +optional
	LivenessProbe *corev1.Probe `json:"livenessProbe,omitempty"`

	// ReadinessProbe defines the readiness probe for the tool container
	// +optional
	ReadinessProbe *corev1.Probe `json:"readinessProbe,omitempty"`

	// StartupProbe defines the startup probe for the tool container
	// +optional
	StartupProbe *corev1.Probe `json:"startupProbe,omitempty"`

	// ServiceType specifies the type of Service to create (ClusterIP, NodePort, LoadBalancer)
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

	// PodDisruptionBudget defines the PDB for this tool
	// +optional
	PodDisruptionBudget *PodDisruptionBudgetSpec `json:"podDisruptionBudget,omitempty"`

	// UpdateStrategy defines the update strategy for the Deployment
	// +optional
	UpdateStrategy *UpdateStrategySpec `json:"updateStrategy,omitempty"`
}

// PodDisruptionBudgetSpec defines PDB configuration
type PodDisruptionBudgetSpec struct {
	// MinAvailable specifies the minimum number of pods that must be available
	// +optional
	MinAvailable *int32 `json:"minAvailable,omitempty"`

	// MaxUnavailable specifies the maximum number of pods that can be unavailable
	// +optional
	MaxUnavailable *int32 `json:"maxUnavailable,omitempty"`
}

// UpdateStrategySpec defines deployment update strategy
type UpdateStrategySpec struct {
	// Type of deployment update strategy (RollingUpdate or Recreate)
	// +kubebuilder:validation:Enum=RollingUpdate;Recreate
	// +kubebuilder:default=RollingUpdate
	Type string `json:"type,omitempty"`

	// RollingUpdate configuration (only used if Type is RollingUpdate)
	// +optional
	RollingUpdate *RollingUpdateSpec `json:"rollingUpdate,omitempty"`
}

// RollingUpdateSpec defines rolling update parameters
type RollingUpdateSpec struct {
	// MaxUnavailable is the maximum number of pods that can be unavailable during update
	// +optional
	MaxUnavailable *int32 `json:"maxUnavailable,omitempty"`

	// MaxSurge is the maximum number of pods that can be created above desired replicas
	// +optional
	MaxSurge *int32 `json:"maxSurge,omitempty"`
}

// LanguageToolStatus defines the observed state of LanguageTool
type LanguageToolStatus struct {
	// ObservedGeneration reflects the generation of the most recently observed LanguageTool
	// +optional
	ObservedGeneration int64 `json:"observedGeneration,omitempty"`

	// Phase represents the current phase of the tool (Pending, Running, Failed, Unknown)
	// +kubebuilder:validation:Enum=Pending;Running;Failed;Unknown;Updating
	// +optional
	Phase string `json:"phase,omitempty"`

	// Conditions represent the latest available observations of the tool's state
	// +optional
	// +listType=map
	// +listMapKey=type
	// +patchMergeKey=type
	// +patchStrategy=merge
	Conditions []metav1.Condition `json:"conditions,omitempty" patchStrategy:"merge" patchMergeKey:"type"`

	// Endpoint is the service endpoint where the tool is accessible
	// +optional
	Endpoint string `json:"endpoint,omitempty"`

	// AvailableTools lists the tools discovered from this service
	// +optional
	AvailableTools []string `json:"availableTools,omitempty"`

	// ReadyReplicas is the number of pods ready and passing health checks
	// +optional
	ReadyReplicas int32 `json:"readyReplicas,omitempty"`

	// AvailableReplicas is the number of pods targeted by this LanguageTool with at least one available condition
	// +optional
	AvailableReplicas int32 `json:"availableReplicas,omitempty"`

	// UpdatedReplicas is the number of pods targeted by this LanguageTool that have the desired spec
	// +optional
	UpdatedReplicas int32 `json:"updatedReplicas,omitempty"`

	// UnavailableReplicas is the number of pods targeted by this LanguageTool that are unavailable
	// +optional
	UnavailableReplicas int32 `json:"unavailableReplicas,omitempty"`

	// LastUpdateTime is the last time the status was updated
	// +optional
	LastUpdateTime *metav1.Time `json:"lastUpdateTime,omitempty"`

	// LastTransitionTime is the last time the phase transitioned
	// +optional
	LastTransitionTime *metav1.Time `json:"lastTransitionTime,omitempty"`

	// Message provides human-readable details about the current state
	// +optional
	Message string `json:"message,omitempty"`

	// Reason provides a machine-readable reason for the current state
	// +optional
	Reason string `json:"reason,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:resource:scope=Namespaced,shortName=ltool
// +kubebuilder:printcolumn:name="Type",type=string,JSONPath=`.spec.type`
// +kubebuilder:printcolumn:name="Image",type=string,JSONPath=`.spec.image`
// +kubebuilder:printcolumn:name="Phase",type=string,JSONPath=`.status.phase`
// +kubebuilder:printcolumn:name="Endpoint",type=string,JSONPath=`.status.endpoint`
// +kubebuilder:printcolumn:name="Age",type=date,JSONPath=`.metadata.creationTimestamp`

// LanguageTool is the Schema for the languagetools API
type LanguageTool struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   LanguageToolSpec   `json:"spec,omitempty"`
	Status LanguageToolStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// LanguageToolList contains a list of LanguageTool
type LanguageToolList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []LanguageTool `json:"items"`
}

func init() {
	SchemeBuilder.Register(&LanguageTool{}, &LanguageToolList{})
}
