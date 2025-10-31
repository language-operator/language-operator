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

	// Namespace created/used by this cluster
	Namespace string `json:"namespace,omitempty"`

	// Conditions
	Conditions []metav1.Condition `json:"conditions,omitempty"`
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
