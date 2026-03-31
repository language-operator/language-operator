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

package controllers

import (
	"context"
	"testing"

	langopv1alpha1 "github.com/language-operator/language-operator/api/v1alpha1"
	"github.com/language-operator/language-operator/controllers/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	networkingv1 "k8s.io/api/networking/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

// newFakeClient returns a minimal fake client sufficient for BuildEgressNetworkPolicy
// (used by DetectCNI to list CRDs; returns empty lists which causes CNI to fall back to standard rules).
func newFakeClient(t *testing.T) *fake.ClientBuilder {
	t.Helper()
	return fake.NewClientBuilder().WithScheme(testutil.SetupTestScheme(t))
}

func TestBuildEgressNetworkPolicy_GroupSelector(t *testing.T) {
	c := newFakeClient(t).Build()
	policy := BuildEgressNetworkPolicy(
		context.Background(), c,
		"test-policy", "default",
		map[string]string{"app": "test"},
		"",
		[]langopv1alpha1.NetworkRule{
			{To: &langopv1alpha1.NetworkPeer{Group: "my-group"}},
		},
	)
	require.NotNil(t, policy)

	// Find the user-defined egress rule (skip the built-in cluster/DNS rules)
	var found bool
	for _, rule := range policy.Spec.Egress {
		for _, peer := range rule.To {
			if peer.PodSelector != nil {
				v, ok := peer.PodSelector.MatchLabels[LabelKeyLangopGroup]
				if ok && v == "my-group" {
					found = true
				}
			}
		}
	}
	assert.True(t, found, "expected egress peer with PodSelector langop.io/group=my-group")
}

func TestBuildEgressNetworkPolicy_ServiceSelector_ExplicitNamespace(t *testing.T) {
	c := newFakeClient(t).Build()
	policy := BuildEgressNetworkPolicy(
		context.Background(), c,
		"test-policy", "default",
		map[string]string{"app": "test"},
		"",
		[]langopv1alpha1.NetworkRule{
			{To: &langopv1alpha1.NetworkPeer{Service: &langopv1alpha1.ServiceReference{
				Name:      "my-svc",
				Namespace: "other-ns",
			}}},
		},
	)
	require.NotNil(t, policy)

	var found bool
	for _, rule := range policy.Spec.Egress {
		for _, peer := range rule.To {
			if peer.NamespaceSelector != nil {
				v, ok := peer.NamespaceSelector.MatchLabels[LabelKeyMetadataName]
				if ok && v == "other-ns" {
					found = true
				}
			}
		}
	}
	assert.True(t, found, "expected egress peer with NamespaceSelector kubernetes.io/metadata.name=other-ns")
}

func TestBuildEgressNetworkPolicy_ServiceSelector_DefaultsToCurrentNamespace(t *testing.T) {
	c := newFakeClient(t).Build()
	policy := BuildEgressNetworkPolicy(
		context.Background(), c,
		"test-policy", "my-namespace",
		map[string]string{"app": "test"},
		"",
		[]langopv1alpha1.NetworkRule{
			{To: &langopv1alpha1.NetworkPeer{Service: &langopv1alpha1.ServiceReference{
				Name: "my-svc",
				// Namespace intentionally omitted
			}}},
		},
	)
	require.NotNil(t, policy)

	var found bool
	for _, rule := range policy.Spec.Egress {
		for _, peer := range rule.To {
			if peer.NamespaceSelector != nil {
				v, ok := peer.NamespaceSelector.MatchLabels[LabelKeyMetadataName]
				if ok && v == "my-namespace" {
					found = true
				}
			}
		}
	}
	assert.True(t, found, "expected egress peer defaulting namespace to my-namespace")
}

func TestBuildEgressNetworkPolicy_NamespaceSelector(t *testing.T) {
	c := newFakeClient(t).Build()
	policy := BuildEgressNetworkPolicy(
		context.Background(), c,
		"test-policy", "default",
		map[string]string{"app": "test"},
		"",
		[]langopv1alpha1.NetworkRule{
			{To: &langopv1alpha1.NetworkPeer{
				NamespaceSelector: &metav1.LabelSelector{
					MatchLabels: map[string]string{"env": "prod"},
				},
			}},
		},
	)
	require.NotNil(t, policy)

	var found bool
	for _, rule := range policy.Spec.Egress {
		for _, peer := range rule.To {
			if peer.NamespaceSelector != nil && peer.PodSelector == nil {
				v, ok := peer.NamespaceSelector.MatchLabels["env"]
				if ok && v == "prod" {
					found = true
				}
			}
		}
	}
	assert.True(t, found, "expected egress peer with NamespaceSelector env=prod")
}

func TestBuildEgressNetworkPolicy_PodSelector(t *testing.T) {
	c := newFakeClient(t).Build()
	policy := BuildEgressNetworkPolicy(
		context.Background(), c,
		"test-policy", "default",
		map[string]string{"app": "test"},
		"",
		[]langopv1alpha1.NetworkRule{
			{To: &langopv1alpha1.NetworkPeer{
				PodSelector: &metav1.LabelSelector{
					MatchLabels: map[string]string{"role": "backend"},
				},
			}},
		},
	)
	require.NotNil(t, policy)

	var found bool
	for _, rule := range policy.Spec.Egress {
		for _, peer := range rule.To {
			if peer.PodSelector != nil && peer.NamespaceSelector == nil {
				v, ok := peer.PodSelector.MatchLabels["role"]
				if ok && v == "backend" {
					found = true
				}
			}
		}
	}
	assert.True(t, found, "expected egress peer with PodSelector role=backend")
}

func TestBuildEgressNetworkPolicy_NamespaceAndPodSelectorCombined(t *testing.T) {
	nsSel := &metav1.LabelSelector{MatchLabels: map[string]string{"env": "staging"}}
	podSel := &metav1.LabelSelector{MatchLabels: map[string]string{"app": "worker"}}

	c := newFakeClient(t).Build()
	policy := BuildEgressNetworkPolicy(
		context.Background(), c,
		"test-policy", "default",
		map[string]string{"app": "test"},
		"",
		[]langopv1alpha1.NetworkRule{
			{To: &langopv1alpha1.NetworkPeer{
				NamespaceSelector: nsSel,
				PodSelector:       podSel,
			}},
		},
	)
	require.NotNil(t, policy)

	// Both selectors must appear on the SAME peer (not separate peers)
	var found bool
	for _, rule := range policy.Spec.Egress {
		for _, peer := range rule.To {
			if peer.NamespaceSelector != nil && peer.PodSelector != nil {
				nsOk := peer.NamespaceSelector.MatchLabels["env"] == "staging"
				podOk := peer.PodSelector.MatchLabels["app"] == "worker"
				if nsOk && podOk {
					found = true
				}
			}
		}
	}
	assert.True(t, found, "expected a single egress peer combining NamespaceSelector and PodSelector")
}

func TestBuildEgressNetworkPolicy_FromRule_AddsIngressAndPolicyType(t *testing.T) {
	c := newFakeClient(t).Build()
	policy := BuildEgressNetworkPolicy(
		context.Background(), c,
		"test-policy", "default",
		map[string]string{"app": "test"},
		"",
		[]langopv1alpha1.NetworkRule{
			{
				From: &langopv1alpha1.NetworkPeer{
					Group: "external-readers",
				},
				Ports: []langopv1alpha1.NetworkPort{{Port: 8080}},
			},
		},
	)
	require.NotNil(t, policy)

	// PolicyTypeIngress must be present
	hasIngress := false
	for _, pt := range policy.Spec.PolicyTypes {
		if pt == networkingv1.PolicyTypeIngress {
			hasIngress = true
		}
	}
	assert.True(t, hasIngress, "expected PolicyTypeIngress when From rules are present")

	require.NotEmpty(t, policy.Spec.Ingress, "expected at least one ingress rule")
	peer := policy.Spec.Ingress[0].From[0]
	require.NotNil(t, peer.PodSelector)
	assert.Equal(t, "external-readers", peer.PodSelector.MatchLabels[LabelKeyLangopGroup])

	require.NotEmpty(t, policy.Spec.Ingress[0].Ports)
	assert.Equal(t, int32(8080), policy.Spec.Ingress[0].Ports[0].Port.IntVal)
}

func TestBuildEgressNetworkPolicy_NoFromRule_NoPolicyTypeIngress(t *testing.T) {
	c := newFakeClient(t).Build()
	policy := BuildEgressNetworkPolicy(
		context.Background(), c,
		"test-policy", "default",
		map[string]string{"app": "test"},
		"",
		[]langopv1alpha1.NetworkRule{
			{To: &langopv1alpha1.NetworkPeer{CIDR: "10.0.0.0/8"}},
		},
	)
	require.NotNil(t, policy)

	for _, pt := range policy.Spec.PolicyTypes {
		assert.NotEqual(t, networkingv1.PolicyTypeIngress, pt, "PolicyTypeIngress must not appear when no From rules present")
	}
	assert.Empty(t, policy.Spec.Ingress)
}
