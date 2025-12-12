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
	"testing"

	"k8s.io/apimachinery/pkg/api/meta"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

// mockRESTMapper implements RESTMapper for testing
type mockRESTMapper struct {
	supportedGVKs map[schema.GroupVersionKind]bool
}

func (m *mockRESTMapper) KindFor(resource schema.GroupVersionResource) (schema.GroupVersionKind, error) {
	return schema.GroupVersionKind{}, nil
}

func (m *mockRESTMapper) KindsFor(resource schema.GroupVersionResource) ([]schema.GroupVersionKind, error) {
	return nil, nil
}

func (m *mockRESTMapper) ResourceFor(input schema.GroupVersionResource) (schema.GroupVersionResource, error) {
	return schema.GroupVersionResource{}, nil
}

func (m *mockRESTMapper) ResourcesFor(input schema.GroupVersionResource) ([]schema.GroupVersionResource, error) {
	return nil, nil
}

func (m *mockRESTMapper) RESTMapping(gk schema.GroupKind, versions ...string) (*meta.RESTMapping, error) {
	gvk := schema.GroupVersionKind{
		Group:   gk.Group,
		Kind:    gk.Kind,
		Version: "v1",
	}
	if len(versions) > 0 {
		gvk.Version = versions[0]
	}

	if m.supportedGVKs[gvk] {
		return &meta.RESTMapping{
			Resource:         schema.GroupVersionResource{Group: gvk.Group, Version: gvk.Version, Resource: "test"},
			GroupVersionKind: gvk,
		}, nil
	}

	return nil, &meta.NoKindMatchError{}
}

func (m *mockRESTMapper) RESTMappings(gk schema.GroupKind, versions ...string) ([]*meta.RESTMapping, error) {
	return nil, nil
}

func (m *mockRESTMapper) ResourceSingularizer(resource string) (singular string, err error) {
	return resource, nil
}

func TestDetectCNI(t *testing.T) {
	tests := []struct {
		name          string
		supportedGVKs map[schema.GroupVersionKind]bool
		expected      CNIProvider
	}{
		{
			name: "cilium detected",
			supportedGVKs: map[schema.GroupVersionKind]bool{
				{Group: "cilium.io", Version: "v2", Kind: "CiliumNetworkPolicy"}: true,
			},
			expected: CNIProviderCilium,
		},
		{
			name: "calico detected",
			supportedGVKs: map[schema.GroupVersionKind]bool{
				{Group: "crd.projectcalico.org", Version: "v1", Kind: "NetworkPolicy"}: true,
			},
			expected: CNIProviderCalico,
		},
		{
			name:          "generic CNI",
			supportedGVKs: map[schema.GroupVersionKind]bool{},
			expected:      CNIProviderGeneric,
		},
		{
			name: "cilium preferred over calico",
			supportedGVKs: map[schema.GroupVersionKind]bool{
				{Group: "cilium.io", Version: "v2", Kind: "CiliumNetworkPolicy"}:       true,
				{Group: "crd.projectcalico.org", Version: "v1", Kind: "NetworkPolicy"}: true,
			},
			expected: CNIProviderCilium,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create fake client with custom REST mapper
			scheme := runtime.NewScheme()
			client := fake.NewClientBuilder().
				WithScheme(scheme).
				WithRESTMapper(&mockRESTMapper{supportedGVKs: tt.supportedGVKs}).
				Build()

			result, err := DetectCNI(context.Background(), client)
			if err != nil {
				t.Errorf("DetectCNI() error = %v", err)
				return
			}

			if result != tt.expected {
				t.Errorf("DetectCNI() = %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestCNIProviderString(t *testing.T) {
	tests := []struct {
		provider CNIProvider
		expected string
	}{
		{CNIProviderCilium, "cilium"},
		{CNIProviderCalico, "calico"},
		{CNIProviderGeneric, "generic"},
	}

	for _, tt := range tests {
		t.Run(string(tt.provider), func(t *testing.T) {
			if string(tt.provider) != tt.expected {
				t.Errorf("CNIProvider string = %v, want %v", string(tt.provider), tt.expected)
			}
		})
	}
}
