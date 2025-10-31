/*
Copyright 2025 Based Team.

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
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// LanguageClusterSpec defines the desired state of LanguageCluster
type LanguageClusterSpec struct {
	// Namespace to create/use for this cluster
	// If empty, auto-generates: <cluster-name>-ns
	// +optional
	Namespace string `json:"namespace,omitempty"`

	// Network configuration
	// +optional
	Network NetworkConfig `json:"network,omitempty"`

	// Security groups defining network boundaries (advanced use)
	// By default, all resources in the cluster can communicate with each other,
	// and external access is controlled by egress rules on individual resources
	// +optional
	Groups []SecurityGroup `json:"groups,omitempty"`
}

// NetworkConfig defines network-level settings
type NetworkConfig struct {
	// PodCIDR for documentation/validation purposes
	// +optional
	PodCIDR string `json:"podCIDR,omitempty"`

	// DefaultPolicy: deny (default) or allow
	// +kubebuilder:validation:Enum=deny;allow
	// +kubebuilder:default=deny
	DefaultPolicy string `json:"defaultPolicy,omitempty"`
}

// SecurityGroup defines a network isolation boundary
type SecurityGroup struct {
	// Name of the security group
	// +kubebuilder:validation:Required
	// +kubebuilder:validation:Pattern=^[a-z0-9]([-a-z0-9]*[a-z0-9])?$
	Name string `json:"name"`

	// Description of this group's purpose
	// +optional
	Description string `json:"description,omitempty"`

	// Ingress rules (who can connect TO this group)
	// +optional
	Ingress []NetworkRule `json:"ingress,omitempty"`

	// Egress rules (where this group can connect TO)
	// +optional
	Egress []NetworkRule `json:"egress,omitempty"`
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
	// Group within the same LanguageCluster
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

	// Namespace created/used by this cluster
	Namespace string `json:"namespace,omitempty"`

	// Conditions
	Conditions []metav1.Condition `json:"conditions,omitempty"`

	// Group membership tracking
	GroupMembership map[string]GroupMembershipInfo `json:"groupMembership,omitempty"`

	// NetworkPolicies created
	NetworkPolicies []string `json:"networkPolicies,omitempty"`
}

// GroupMembershipInfo tracks resources in a security group
type GroupMembershipInfo struct {
	// Count of resources in this group
	Count int `json:"count"`

	// List of resource names
	Resources []string `json:"resources,omitempty"`
}

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status
//+kubebuilder:resource:scope=Cluster
//+kubebuilder:printcolumn:name="Phase",type=string,JSONPath=`.status.phase`
//+kubebuilder:printcolumn:name="Namespace",type=string,JSONPath=`.status.namespace`
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
