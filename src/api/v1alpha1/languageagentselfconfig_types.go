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
	rbacv1 "k8s.io/api/rbac/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// SelfConfigAction is a category of self-modification an agent may request.
// +kubebuilder:validation:Enum=tools;models;envVars;instructions;roleRules
type SelfConfigAction string

const (
	// SelfConfigActionTools allows the agent to add or remove tool references.
	SelfConfigActionTools SelfConfigAction = "tools"
	// SelfConfigActionModels allows the agent to add or remove model references.
	SelfConfigActionModels SelfConfigAction = "models"
	// SelfConfigActionEnvVars allows the agent to inject plain-value environment variables.
	SelfConfigActionEnvVars SelfConfigAction = "envVars"
	// SelfConfigActionInstructions allows the agent to replace its system instructions.
	SelfConfigActionInstructions SelfConfigAction = "instructions"
	// SelfConfigActionRoleRules allows the agent to append RBAC policy rules to its Role.
	SelfConfigActionRoleRules SelfConfigAction = "roleRules"
)

// SelfConfigureSpec controls whether a LanguageAgent may submit LanguageAgentSelfConfig
// requests to modify its own spec at runtime.
type SelfConfigureSpec struct {
	// Enabled gates all self-configuration. When false, any LanguageAgentSelfConfig
	// targeting this agent is immediately Denied.
	// +kubebuilder:default=false
	// +optional
	Enabled *bool `json:"enabled,omitempty"`

	// AllowedActions is the allowlist of self-config categories the agent may request.
	// If empty while Enabled=true, all actions are denied.
	// +optional
	// +listType=set
	AllowedActions []SelfConfigAction `json:"allowedActions,omitempty"`
}

// SelfConfigEnvVar is a plain-value environment variable injected by a self-config request.
// SecretKeyRef and other value sources are intentionally not supported.
type SelfConfigEnvVar struct {
	// Name of the environment variable.
	// +kubebuilder:validation:Required
	// +kubebuilder:validation:MinLength=1
	Name string `json:"name"`

	// Value is the literal string value of the variable.
	// +kubebuilder:validation:Required
	Value string `json:"value"`
}

// SelfConfigPhase reflects the current processing state of a LanguageAgentSelfConfig.
// +kubebuilder:validation:Enum=Pending;Applied;Failed;Denied
type SelfConfigPhase string

const (
	// SelfConfigPhasePending means the request has not yet been processed.
	SelfConfigPhasePending SelfConfigPhase = "Pending"
	// SelfConfigPhaseApplied means the requested changes were patched onto the parent LanguageAgent.
	SelfConfigPhaseApplied SelfConfigPhase = "Applied"
	// SelfConfigPhaseFailed means the controller encountered an error while processing the request.
	SelfConfigPhaseFailed SelfConfigPhase = "Failed"
	// SelfConfigPhaseDenied means the request was rejected due to policy (not enabled, or action not allowed).
	SelfConfigPhaseDenied SelfConfigPhase = "Denied"
)

// LanguageAgentSelfConfigSpec defines the desired self-modification.
type LanguageAgentSelfConfigSpec struct {
	// InstanceRef is the name of the LanguageAgent to modify. Must be in the same namespace.
	// +kubebuilder:validation:Required
	// +kubebuilder:validation:MinLength=1
	InstanceRef string `json:"instanceRef"`

	// AddTools lists LanguageTool names to append to spec.tools on the parent agent.
	// +optional
	AddTools []string `json:"addTools,omitempty"`

	// RemoveTools lists LanguageTool names to remove from spec.tools on the parent agent.
	// +optional
	RemoveTools []string `json:"removeTools,omitempty"`

	// AddModels lists LanguageModel names to append to spec.models on the parent agent.
	// +optional
	AddModels []string `json:"addModels,omitempty"`

	// RemoveModels lists LanguageModel names to remove from spec.models on the parent agent.
	// +optional
	RemoveModels []string `json:"removeModels,omitempty"`

	// AddEnvVars lists plain-value environment variables to inject into the parent agent's
	// spec.deployment.env. Existing vars with the same name are overwritten.
	// SecretKeyRef and other value sources are not supported.
	// +optional
	AddEnvVars []SelfConfigEnvVar `json:"addEnvVars,omitempty"`

	// UpdateInstructions, when non-empty, replaces spec.instructions on the parent agent.
	// +optional
	UpdateInstructions string `json:"updateInstructions,omitempty"`

	// AddRoleRules lists RBAC policy rules to append to spec.deployment.roleRules on the
	// parent agent. Duplicate rules (identical APIGroups+Resources+Verbs) are de-duplicated.
	// +optional
	AddRoleRules []rbacv1.PolicyRule `json:"addRoleRules,omitempty"`
}

// LanguageAgentSelfConfigStatus reflects the observed state of a self-config request.
type LanguageAgentSelfConfigStatus struct {
	// Phase is the current processing state.
	// +optional
	Phase SelfConfigPhase `json:"phase,omitempty"`

	// Message provides a human-readable explanation of the current phase.
	// +optional
	Message string `json:"message,omitempty"`

	// CompletionTime is set when the request reaches a terminal phase (Applied, Failed, Denied).
	// The CR is automatically deleted 1 hour after this time.
	// +optional
	CompletionTime *metav1.Time `json:"completionTime,omitempty"`
}

// LanguageAgentSelfConfig is submitted by an agent pod to request runtime modifications
// to its own LanguageAgent spec. The controller validates the request against the parent's
// spec.selfConfigure allowlist before patching.
//
// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:resource:scope=Namespaced,shortName=lasc
// +kubebuilder:printcolumn:name="Agent",type=string,JSONPath=`.spec.instanceRef`
// +kubebuilder:printcolumn:name="Phase",type=string,JSONPath=`.status.phase`
// +kubebuilder:printcolumn:name="Age",type=date,JSONPath=`.metadata.creationTimestamp`
type LanguageAgentSelfConfig struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   LanguageAgentSelfConfigSpec   `json:"spec,omitempty"`
	Status LanguageAgentSelfConfigStatus `json:"status,omitempty"`
}

// LanguageAgentSelfConfigList contains a list of LanguageAgentSelfConfig.
//
// +kubebuilder:object:root=true
type LanguageAgentSelfConfigList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []LanguageAgentSelfConfig `json:"items"`
}

func init() {
	SchemeBuilder.Register(&LanguageAgentSelfConfig{}, &LanguageAgentSelfConfigList{})
}
