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
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// LanguageAgentRuntimeSpec defines a preset configuration for LanguageAgent deployments.
// All fields are optional; unset fields leave the agent's own spec in effect.
// When a LanguageAgent references a runtime, the runtime's fields are merged as defaults:
// scalars fill in zeros/nils (agent wins if set), lists are runtime-first then agent-appended.
type LanguageAgentRuntimeSpec struct {
	// Image is the default container image for agents using this runtime.
	// Agents may override this. When a runtime is referenced, spec.image on the agent is optional.
	// +optional
	Image string `json:"image,omitempty"`

	// Port is the default port the agent container listens on.
	// +kubebuilder:validation:Minimum=1
	// +kubebuilder:validation:Maximum=65535
	// +optional
	Port *int32 `json:"port,omitempty"`

	// ExecutionMode is the default execution mode for agents using this runtime.
	// +kubebuilder:validation:Enum=autonomous;interactive;scheduled;event-driven
	// +optional
	ExecutionMode string `json:"executionMode,omitempty"`

	// Workspace provides default size, storageClass, and mountPath for the agent's workspace.
	// Workspace storage is always provisioned; this presets its parameters.
	// Agents may override individual workspace fields.
	// +optional
	Workspace *WorkspaceSpec `json:"workspace,omitempty"`

	// Deployment provides default Kubernetes pod and container configuration.
	// Scalars (args, command, resources, probes, etc.) are used when the agent has none set.
	// Lists (initContainers, env, volumes, volumeMounts, envFrom) are runtime-first, agent-appended.
	// +optional
	Deployment DeploymentSpec `json:"deployment,omitempty"`
}

// LanguageAgentRuntimeStatus defines the observed state of LanguageAgentRuntime
type LanguageAgentRuntimeStatus struct {
	// ObservedGeneration is the most recent generation processed by the controller.
	// +optional
	ObservedGeneration int64 `json:"observedGeneration,omitempty"`

	// Conditions represent the latest available observations of the runtime's state
	// +optional
	// +listType=map
	// +listMapKey=type
	// +patchMergeKey=type
	// +patchStrategy=merge
	Conditions []metav1.Condition `json:"conditions,omitempty" patchStrategy:"merge" patchMergeKey:"type"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:resource:scope=Cluster,shortName=laruntime
// +kubebuilder:printcolumn:name="Ready",type=string,JSONPath=`.status.conditions[?(@.type=="Ready")].status`
// +kubebuilder:printcolumn:name="Age",type=date,JSONPath=`.metadata.creationTimestamp`

// LanguageAgentRuntime is the Schema for the languageagentruntimes API.
// It defines a reusable preset for LanguageAgent deployments, analogous to an IngressClass.
// Admins create runtimes; users reference them via spec.runtime on a LanguageAgent.
type LanguageAgentRuntime struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   LanguageAgentRuntimeSpec   `json:"spec,omitempty"`
	Status LanguageAgentRuntimeStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// LanguageAgentRuntimeList contains a list of LanguageAgentRuntime
type LanguageAgentRuntimeList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []LanguageAgentRuntime `json:"items"`
}

func init() {
	SchemeBuilder.Register(&LanguageAgentRuntime{}, &LanguageAgentRuntimeList{})
}
