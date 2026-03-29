/*
Copyright 2025 Langop Team.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package v1alpha1

import (
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// ClusterCapacitySpec declares hard limits enforced via a ResourceQuota in the cluster's namespace.
type ClusterCapacitySpec struct {
	// MaxAgents is the maximum number of LanguageAgent objects allowed.
	// +optional
	MaxAgents *int32 `json:"maxAgents,omitempty"`
	// MaxModels is the maximum number of LanguageModel objects allowed.
	// +optional
	MaxModels *int32 `json:"maxModels,omitempty"`
	// MaxTools is the maximum number of LanguageTool objects allowed.
	// +optional
	MaxTools *int32 `json:"maxTools,omitempty"`
	// MaxPersonas is the maximum number of LanguagePersona objects allowed.
	// +optional
	MaxPersonas *int32 `json:"maxPersonas,omitempty"`

	// MaxCPU is the aggregate CPU limit for all pods in the cluster namespace.
	// Maps to limits.cpu in the namespace ResourceQuota.
	// Example: "4", "2500m"
	// +optional
	MaxCPU *resource.Quantity `json:"maxCPU,omitempty"`
	// MaxMemory is the aggregate memory limit for all pods in the cluster namespace.
	// Maps to limits.memory in the namespace ResourceQuota.
	// Example: "8Gi", "512Mi"
	// +optional
	MaxMemory *resource.Quantity `json:"maxMemory,omitempty"`
}

// ClusterCapacityStatus reports observed resource usage in the cluster's namespace.
type ClusterCapacityStatus struct {
	// AgentCount is the number of LanguageAgent objects in the cluster namespace.
	AgentCount int32 `json:"agentCount"`
	// ModelCount is the number of LanguageModel objects in the cluster namespace.
	ModelCount int32 `json:"modelCount"`
	// ToolCount is the number of LanguageTool objects in the cluster namespace.
	ToolCount int32 `json:"toolCount"`
	// PersonaCount is the number of LanguagePersona objects in the cluster namespace.
	PersonaCount int32 `json:"personaCount"`

	// TotalCPULimits is the sum of limits.cpu across all agent pod specs.
	// +optional
	TotalCPULimits resource.Quantity `json:"totalCPULimits,omitempty"`
	// TotalMemoryLimits is the sum of limits.memory across all agent pod specs.
	// +optional
	TotalMemoryLimits resource.Quantity `json:"totalMemoryLimits,omitempty"`
}

// LanguageClusterSpec defines the desired state of LanguageCluster
type LanguageClusterSpec struct {
	// LanguageCluster is now a namespaced resource.
	// All resources referencing this cluster must be in the same namespace.
	// No configuration needed - cluster provides logical grouping for resources.

	// Domain is the base domain for the cluster and agent webhook routing
	// The domain itself serves as the cluster dashboard/UI endpoint
	// Agent webhooks will be accessible at <uuid>.<domain>
	// Example: "ai.theryans.io" results in webhooks like "abc123.ai.theryans.io"
	// +optional
	Domain string `json:"domain,omitempty"`

	// Ingress defines ingress configuration for the cluster
	// +optional
	Ingress *IngressConfig `json:"ingress,omitempty"`

	// NetworkPolicies defines egress network policies for agents in this cluster
	// +optional
	NetworkPolicies []NetworkRule `json:"networkPolicies,omitempty"`

	// Gateway configures the shared LiteLLM gateway deployed per cluster
	// +optional
	Gateway *GatewaySpec `json:"gateway,omitempty"`

	// Capacity declares hard limits enforced via a ResourceQuota in the cluster's namespace.
	// When set, the controller creates a ResourceQuota named "langop-quota".
	// When unset, any existing "langop-quota" is deleted.
	// +optional
	Capacity *ClusterCapacitySpec `json:"capacity,omitempty"`
}

// DeploymentSpec groups Kubernetes deployment configuration that is common across
// LanguageAgent, LanguageTool, and LanguageCluster gateway deployments.
// All fields are optional; controllers only read the fields relevant to their resource.
type DeploymentSpec struct {
	// Replicas is the number of pod replicas to run.
	// +kubebuilder:validation:Minimum=0
	// +optional
	Replicas *int32 `json:"replicas,omitempty"`

	// ImagePullPolicy defines when to pull the container image.
	// +kubebuilder:validation:Enum=Always;Never;IfNotPresent
	// +optional
	ImagePullPolicy corev1.PullPolicy `json:"imagePullPolicy,omitempty"`

	// ImagePullSecrets is a list of references to secrets for pulling images.
	// +optional
	ImagePullSecrets []corev1.LocalObjectReference `json:"imagePullSecrets,omitempty"`

	// Env contains environment variables for the container.
	// +optional
	Env []corev1.EnvVar `json:"env,omitempty"`

	// EnvFrom sources to populate environment variables.
	// +optional
	EnvFrom []corev1.EnvFromSource `json:"envFrom,omitempty"`

	// Resources defines compute resource requirements.
	// +optional
	Resources corev1.ResourceRequirements `json:"resources,omitempty"`

	// NodeSelector is a selector which must match a node's labels.
	// +optional
	NodeSelector map[string]string `json:"nodeSelector,omitempty"`

	// Affinity defines pod affinity and anti-affinity rules.
	// +optional
	Affinity *corev1.Affinity `json:"affinity,omitempty"`

	// Tolerations allow pods to schedule onto nodes with matching taints.
	// +optional
	Tolerations []corev1.Toleration `json:"tolerations,omitempty"`

	// TopologySpreadConstraints describes how pods should spread across topology domains.
	// +optional
	TopologySpreadConstraints []corev1.TopologySpreadConstraint `json:"topologySpreadConstraints,omitempty"`

	// ServiceAccountName is the name of the ServiceAccount to use.
	// +optional
	ServiceAccountName string `json:"serviceAccountName,omitempty"`

	// SecurityContext holds pod-level security attributes.
	// +optional
	SecurityContext *corev1.PodSecurityContext `json:"securityContext,omitempty"`

	// VolumeMounts to mount into the container.
	// +optional
	VolumeMounts []corev1.VolumeMount `json:"volumeMounts,omitempty"`

	// Volumes to attach to the pod.
	// +optional
	Volumes []corev1.Volume `json:"volumes,omitempty"`

	// PodAnnotations are annotations to add to the Pods.
	// +optional
	PodAnnotations map[string]string `json:"podAnnotations,omitempty"`

	// PodLabels are additional labels to add to the Pods.
	// +optional
	PodLabels map[string]string `json:"podLabels,omitempty"`

	// InitContainers are additional init containers injected before the main container starts.
	// +optional
	InitContainers []corev1.Container `json:"initContainers,omitempty"`

	// LivenessProbe defines the liveness probe for the container.
	// +optional
	LivenessProbe *corev1.Probe `json:"livenessProbe,omitempty"`

	// ReadinessProbe defines the readiness probe for the container.
	// +optional
	ReadinessProbe *corev1.Probe `json:"readinessProbe,omitempty"`

	// StartupProbe defines the startup probe for the container.
	// +optional
	StartupProbe *corev1.Probe `json:"startupProbe,omitempty"`

	// ServiceType specifies the type of Service to create (ClusterIP, NodePort, LoadBalancer).
	// +kubebuilder:validation:Enum=ClusterIP;NodePort;LoadBalancer
	// +optional
	ServiceType corev1.ServiceType `json:"serviceType,omitempty"`

	// ServiceAnnotations are annotations to add to the Service.
	// +optional
	ServiceAnnotations map[string]string `json:"serviceAnnotations,omitempty"`
}

// GatewaySpec configures the shared LiteLLM gateway deployed per LanguageCluster.
type GatewaySpec struct {
	// Enabled controls whether an Ingress is created for the gateway.
	// Defaults to true when cluster.spec.domain is set.
	// +optional
	Enabled *bool `json:"enabled,omitempty"`

	// Deployment configures the Kubernetes deployment for the gateway pod.
	// +optional
	Deployment DeploymentSpec `json:"deployment,omitempty"`
}

// IngressConfig defines ingress configuration
type IngressConfig struct {
	// TLS configuration for agent webhooks
	// +optional
	TLS *IngressTLSConfig `json:"tls,omitempty"`

	// ClassName specifies the IngressClass to use (maps to spec.ingressClassName on the Ingress object).
	// +optional
	ClassName string `json:"className,omitempty"`
}

// IngressTLSConfig defines TLS configuration
type IngressTLSConfig struct {
	// Enabled controls whether TLS is enabled for webhooks
	// +kubebuilder:default=true
	// +optional
	Enabled *bool `json:"enabled,omitempty"`

	// SecretName is the name of the TLS secret (for manual cert management)
	// If empty, cert-manager will be used if available
	// +optional
	SecretName string `json:"secretName,omitempty"`

	// IssuerRef references a cert-manager Issuer or ClusterIssuer
	// +optional
	IssuerRef *CertIssuerReference `json:"issuerRef,omitempty"`
}

// CertIssuerReference references a cert-manager issuer
type CertIssuerReference struct {
	// Name of the Issuer or ClusterIssuer
	// +kubebuilder:validation:Required
	Name string `json:"name"`

	// Kind is either "Issuer" or "ClusterIssuer"
	// +kubebuilder:validation:Enum=Issuer;ClusterIssuer
	// +kubebuilder:default=ClusterIssuer
	// +optional
	Kind string `json:"kind,omitempty"`

	// Group is the API group of the issuer
	// +kubebuilder:default=cert-manager.io
	// +optional
	Group string `json:"group,omitempty"`
}

// NetworkRule defines a single network policy rule
type NetworkRule struct {
	// Description of this rule
	// +optional
	Description string `json:"description,omitempty"`

	// From selector for ingress rules
	// +optional
	From *NetworkPeer `json:"from,omitempty"`

	// To selector for egress rules
	// +optional
	To *NetworkPeer `json:"to,omitempty"`

	// Ports allowed by this rule
	// +optional
	Ports []NetworkPort `json:"ports,omitempty"`
}

// NetworkPeer defines the source/destination of network traffic
type NetworkPeer struct {
	// Group selects pods with matching langop.io/group label
	// Used to allow communication with specific labeled resources
	// +optional
	Group string `json:"group,omitempty"`

	// CIDR block
	// +optional
	CIDR string `json:"cidr,omitempty"`

	// DNS names (supports wildcards with *)
	// Examples: "api.openai.com", "*.googleapis.com"
	// +optional
	DNS []string `json:"dns,omitempty"`

	// Kubernetes service reference
	// +optional
	Service *ServiceReference `json:"service,omitempty"`

	// Namespace selector (for cross-namespace rules)
	// +optional
	NamespaceSelector *metav1.LabelSelector `json:"namespaceSelector,omitempty"`

	// Pod selector (within namespace)
	// +optional
	PodSelector *metav1.LabelSelector `json:"podSelector,omitempty"`
}

// ServiceReference identifies a Kubernetes Service
type ServiceReference struct {
	// Service name
	// +kubebuilder:validation:Required
	Name string `json:"name"`

	// Service namespace (defaults to same namespace if omitted)
	// +optional
	Namespace string `json:"namespace,omitempty"`
}

// NetworkPort defines a port and protocol
type NetworkPort struct {
	// Protocol (TCP, UDP, SCTP)
	// +kubebuilder:validation:Enum=TCP;UDP;SCTP
	// +kubebuilder:default=TCP
	// +optional
	Protocol string `json:"protocol,omitempty"`

	// Port number
	// +kubebuilder:validation:Minimum=1
	// +kubebuilder:validation:Maximum=65535
	Port int32 `json:"port"`
}

// LanguageClusterStatus defines the observed state
type LanguageClusterStatus struct {
	// Phase of the cluster
	// +kubebuilder:validation:Enum=Ready
	Phase string `json:"phase,omitempty"`

	// Conditions
	Conditions []metav1.Condition `json:"conditions,omitempty"`

	// GatewayEndpoint is the in-cluster URL for the shared LiteLLM gateway
	// +optional
	GatewayEndpoint string `json:"gatewayEndpoint,omitempty"`

	// GatewayReady indicates whether the shared gateway Deployment is available
	// +optional
	GatewayReady bool `json:"gatewayReady,omitempty"`

	// Capacity reports observed resource usage in this cluster's namespace.
	// +optional
	Capacity *ClusterCapacityStatus `json:"capacity,omitempty"`
}

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status
//+kubebuilder:resource:scope=Cluster
//+kubebuilder:printcolumn:name="Phase",type=string,JSONPath=`.status.phase`
//+kubebuilder:printcolumn:name="Age",type=date,JSONPath=`.metadata.creationTimestamp`

// LanguageCluster is the Schema for the languageclusters API
type LanguageCluster struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   LanguageClusterSpec   `json:"spec,omitempty"`
	Status LanguageClusterStatus `json:"status,omitempty"`
}

//+kubebuilder:object:root=true

// LanguageClusterList contains a list of LanguageCluster
type LanguageClusterList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []LanguageCluster `json:"items"`
}

func init() {
	SchemeBuilder.Register(&LanguageCluster{}, &LanguageClusterList{})
}
