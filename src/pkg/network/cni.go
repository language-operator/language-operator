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

package network

import (
	"context"

	"k8s.io/apimachinery/pkg/runtime/schema"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// CNIProvider represents the detected CNI implementation
type CNIProvider string

const (
	// CNIProviderCilium indicates Cilium CNI is being used
	CNIProviderCilium CNIProvider = "cilium"

	// CNIProviderCalico indicates Calico CNI is being used
	CNIProviderCalico CNIProvider = "calico"

	// CNIProviderGeneric indicates a standard CNI or unknown CNI
	CNIProviderGeneric CNIProvider = "generic"
)

// DetectCNI detects the CNI implementation by checking for CNI-specific CRDs
func DetectCNI(ctx context.Context, client client.Client) (CNIProvider, error) {
	// Check for Cilium CRDs first (most specific)
	if hasCiliumCRDs(ctx, client) {
		return CNIProviderCilium, nil
	}

	// Check for Calico CRDs
	if hasCalicoCRDs(ctx, client) {
		return CNIProviderCalico, nil
	}

	// Default to generic CNI for compatibility
	return CNIProviderGeneric, nil
}

// hasCiliumCRDs checks if Cilium CRDs are available in the cluster
func hasCiliumCRDs(ctx context.Context, client client.Client) bool {
	// Check for CiliumNetworkPolicy CRD
	gvk := schema.GroupVersionKind{
		Group:   "cilium.io",
		Version: "v2",
		Kind:    "CiliumNetworkPolicy",
	}

	return hasCRD(ctx, client, gvk)
}

// hasCalicoCRDs checks if Calico CRDs are available in the cluster
func hasCalicoCRDs(ctx context.Context, client client.Client) bool {
	// Check for Calico NetworkPolicy CRD
	gvk := schema.GroupVersionKind{
		Group:   "crd.projectcalico.org",
		Version: "v1",
		Kind:    "NetworkPolicy",
	}

	return hasCRD(ctx, client, gvk)
}

// hasCRD checks if a specific CRD exists by attempting to list resources of that type
func hasCRD(ctx context.Context, client client.Client, gvk schema.GroupVersionKind) bool {
	// Try to discover the resource
	restMapper := client.RESTMapper()
	if restMapper == nil {
		return false
	}

	mapping, err := restMapper.RESTMapping(gvk.GroupKind(), gvk.Version)
	if err != nil {
		return false
	}

	// If we can get the mapping, the CRD exists
	return mapping != nil
}
