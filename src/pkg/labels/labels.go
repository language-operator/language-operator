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

// Package labels defines standard label key constants used across the operator.
package labels

const (
	// Langop-specific label keys.
	LabelKeyLangopKind       = "langop.io/kind"
	LabelKeyLangopGroup      = "langop.io/group"
	LabelKeyLangopCluster    = "langop.io/cluster"
	LabelKeyLangopComponent  = "langop.io/component"
	LabelKeyLangopConfigHash = "langop.io/config-hash"

	// LabelKeyMetadataName is the well-known Kubernetes metadata name label.
	LabelKeyMetadataName = "kubernetes.io/metadata.name"

	// Kubernetes recommended label keys (app.kubernetes.io/*).
	LabelKeyK8sName      = "app.kubernetes.io/name"
	LabelKeyK8sComponent = "app.kubernetes.io/component"
	LabelKeyK8sManagedBy = "app.kubernetes.io/managed-by"
	LabelKeyK8sPartOf    = "app.kubernetes.io/part-of"
)
