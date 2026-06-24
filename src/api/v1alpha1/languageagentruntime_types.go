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

	// Ports defines default ports for agents using this runtime.
	// Replace semantics: when the agent defines spec.ports, runtime ports are ignored entirely.
	// +optional
	// +listType=map
	// +listMapKey=name
	Ports []AgentPort `json:"ports,omitempty"`

	// Deployment provides default Kubernetes pod and container configuration.
	// Scalars (args, command, resources, probes, etc.) are used when the agent has none set.
	// Lists (initContainers, env, volumes, volumeMounts, envFrom) are runtime-first, agent-appended.
	// +optional
	Deployment DeploymentSpec `json:"deployment,omitempty"`

	// Credentials declares environment variables backed by Secret values that the
	// operator resolves and injects into agents using this runtime. Each entry is
	// auto-generated, set inline, or sourced from an existing Secret. Merged into the
	// agent's effective spec runtime-first; agent entries override by name.
	// +optional
	// +listType=map
	// +listMapKey=name
	Credentials []CredentialSpec `json:"credentials,omitempty"`

	// Auth gates whether agents using this runtime sit behind the cluster OIDC proxy.
	// Effective only when the cluster has auth enabled (which provisions the OIDC
	// infrastructure). The OIDC connection itself is configured cluster-wide.
	// +optional
	Auth *RuntimeAuthSpec `json:"auth,omitempty"`
}

// RuntimeAuthSpec gates OIDC authentication for agents using a runtime.
type RuntimeAuthSpec struct {
	// Enabled puts agents using this runtime behind the cluster OIDC proxy.
	// Has no effect unless the cluster has auth enabled.
	// +optional
	Enabled *bool `json:"enabled,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:resource:scope=Cluster,shortName=laruntime
// +kubebuilder:printcolumn:name="Age",type=date,JSONPath=`.metadata.creationTimestamp`

// LanguageAgentRuntime is the Schema for the languageagentruntimes API.
// It defines a reusable preset for LanguageAgent deployments, analogous to an IngressClass.
// Admins create runtimes; users reference them via spec.runtime on a LanguageAgent.
type LanguageAgentRuntime struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec LanguageAgentRuntimeSpec `json:"spec,omitempty"`
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
