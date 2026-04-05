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

// ManagedResource describes a single Kubernetes resource created and owned
// by this operator on behalf of a LanguageAgent or LanguageCluster.
// The combination of Group+Kind+Namespace+Name uniquely identifies the resource.
type ManagedResource struct {
	// Group is the API group of the resource.
	// Empty string means the core API group (e.g. ConfigMap, Service, PersistentVolumeClaim).
	// +optional
	Group string `json:"group,omitempty"`

	// Kind is the Kubernetes resource kind (e.g. "Deployment", "ConfigMap").
	Kind string `json:"kind"`

	// Name is the name of the resource.
	Name string `json:"name"`

	// Namespace is the namespace of the resource.
	// Empty for cluster-scoped resources (e.g. Namespace).
	// +optional
	Namespace string `json:"namespace,omitempty"`
}
