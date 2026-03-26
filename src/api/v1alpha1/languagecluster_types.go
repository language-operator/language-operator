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
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

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

	// IngressConfig defines ingress/gateway configuration for the cluster
	// +optional
	IngressConfig *IngressConfig `json:"ingressConfig,omitempty"`

	// NetworkPolicies defines egress network policies for agents in this cluster
	// +optional
	NetworkPolicies []NetworkRule `json:"networkPolicies,omitempty"`

	// Proxy configures the shared LiteLLM proxy deployed per cluster
	// +optional
	Proxy *ProxyConfig `json:"proxy,omitempty"`
}

// ProxyConfig configures the shared LiteLLM proxy for a LanguageCluster
type ProxyConfig struct {
	// IngressEnabled controls whether an Ingress/HTTPRoute is created for the proxy.
	// Defaults to true when cluster.spec.domain is set.
	// +optional
	IngressEnabled *bool `json:"ingressEnabled,omitempty"`

	// Resources sets CPU/memory requests and limits for the proxy Deployment.
	// +optional
	Resources corev1.ResourceRequirements `json:"resources,omitempty"`

	// Replicas sets the number of proxy pod replicas.
	// +kubebuilder:default=1
	// +optional
	Replicas *int32 `json:"replicas,omitempty"`
}

// IngressConfig defines ingress/gateway configuration
type IngressConfig struct {
	// TLS configuration for agent webhooks
	// +optional
	TLS *IngressTLSConfig `json:"tls,omitempty"`

	// GatewayName specifies the Gateway resource name to use
	// If empty, will attempt auto-detection or fall back to Ingress
	// +optional
	GatewayName string `json:"gatewayName,omitempty"`

	// GatewayNamespace specifies the namespace of the Gateway resource
	// If empty, defaults to the same namespace as the LanguageCluster
	// +optional
	GatewayNamespace string `json:"gatewayNamespace,omitempty"`

	// Deprecated: Use GatewayName instead. This field actually refers to a Gateway resource name, not a GatewayClass.
	// GatewayClassName specifies the Gateway API GatewayClass to use
	// If empty, will attempt auto-detection or fall back to Ingress
	// +optional
	GatewayClassName string `json:"gatewayClassName,omitempty"`

	// IngressClassName specifies the Ingress class to use for fallback
	// Only used when Gateway API is not available
	// +optional
	IngressClassName string `json:"ingressClassName,omitempty"`
}

// IngressTLSConfig defines TLS configuration
type IngressTLSConfig struct {
	// Enabled controls whether TLS is enabled for webhooks
	// +kubebuilder:default=true
	// +optional
	Enabled bool `json:"enabled,omitempty"`

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
	// Phase of the cluster (Pending, Ready, Failed)
	Phase string `json:"phase,omitempty"`

	// Conditions
	Conditions []metav1.Condition `json:"conditions,omitempty"`

	// ProxyEndpoint is the in-cluster URL for the shared LiteLLM proxy
	// +optional
	ProxyEndpoint string `json:"proxyEndpoint,omitempty"`

	// ProxyReady indicates whether the shared proxy Deployment is available
	// +optional
	ProxyReady bool `json:"proxyReady,omitempty"`
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
