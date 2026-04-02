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
	"fmt"
	"testing"
	"time"

	langopv1alpha1 "github.com/language-operator/language-operator/api/v1alpha1"
	"github.com/language-operator/language-operator/controllers/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	networkingv1 "k8s.io/api/networking/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/controller-runtime/pkg/client/interceptor"
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
		&langopv1alpha1.AgentNetworkPolicies{
			Egress: []langopv1alpha1.NetworkEgressRule{
				{To: []langopv1alpha1.NetworkPeer{{Group: "my-group"}}},
			},
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
		&langopv1alpha1.AgentNetworkPolicies{
			Egress: []langopv1alpha1.NetworkEgressRule{
				{To: []langopv1alpha1.NetworkPeer{{Service: &langopv1alpha1.ServiceReference{
					Name:      "my-svc",
					Namespace: "other-ns",
				}}}},
			},
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
		&langopv1alpha1.AgentNetworkPolicies{
			Egress: []langopv1alpha1.NetworkEgressRule{
				{To: []langopv1alpha1.NetworkPeer{{Service: &langopv1alpha1.ServiceReference{
					Name: "my-svc",
					// Namespace intentionally omitted
				}}}},
			},
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
		&langopv1alpha1.AgentNetworkPolicies{
			Egress: []langopv1alpha1.NetworkEgressRule{
				{To: []langopv1alpha1.NetworkPeer{{
					NamespaceSelector: &metav1.LabelSelector{
						MatchLabels: map[string]string{"env": "prod"},
					},
				}}},
			},
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
		&langopv1alpha1.AgentNetworkPolicies{
			Egress: []langopv1alpha1.NetworkEgressRule{
				{To: []langopv1alpha1.NetworkPeer{{
					PodSelector: &metav1.LabelSelector{
						MatchLabels: map[string]string{"role": "backend"},
					},
				}}},
			},
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
		&langopv1alpha1.AgentNetworkPolicies{
			Egress: []langopv1alpha1.NetworkEgressRule{
				{To: []langopv1alpha1.NetworkPeer{{
					NamespaceSelector: nsSel,
					PodSelector:       podSel,
				}}},
			},
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

// BuildEgressNetworkPolicy only processes egress rules; ingress rules are assembled by each
// controller after calling this function. Verify the builder does not populate Ingress.
func TestBuildEgressNetworkPolicy_IngressRulesNotSetByBuilder(t *testing.T) {
	c := newFakeClient(t).Build()
	policy := BuildEgressNetworkPolicy(
		context.Background(), c,
		"test-policy", "default",
		map[string]string{"app": "test"},
		"",
		&langopv1alpha1.AgentNetworkPolicies{
			Ingress: []langopv1alpha1.NetworkIngressRule{
				{
					From:  []langopv1alpha1.NetworkPeer{{Group: "external-readers"}},
					Ports: []langopv1alpha1.NetworkPort{{Port: 8080}},
				},
			},
		},
	)
	require.NotNil(t, policy)

	// Builder must not add PolicyTypeIngress or Ingress rules — controllers do that.
	for _, pt := range policy.Spec.PolicyTypes {
		assert.NotEqual(t, networkingv1.PolicyTypeIngress, pt, "builder must not set PolicyTypeIngress; controllers are responsible")
	}
	assert.Empty(t, policy.Spec.Ingress, "builder must not populate Ingress; controllers append ingress rules after calling BuildEgressNetworkPolicy")
}

func TestBuildEgressNetworkPolicy_NoIngressRules_NoPolicyTypeIngress(t *testing.T) {
	c := newFakeClient(t).Build()
	policy := BuildEgressNetworkPolicy(
		context.Background(), c,
		"test-policy", "default",
		map[string]string{"app": "test"},
		"",
		&langopv1alpha1.AgentNetworkPolicies{
			Egress: []langopv1alpha1.NetworkEgressRule{
				{To: []langopv1alpha1.NetworkPeer{{CIDR: "10.0.0.0/8"}}},
			},
		},
	)
	require.NotNil(t, policy)

	for _, pt := range policy.Spec.PolicyTypes {
		assert.NotEqual(t, networkingv1.PolicyTypeIngress, pt, "PolicyTypeIngress must not appear when no ingress rules present")
	}
	assert.Empty(t, policy.Spec.Ingress)
}

// TestCreateOrUpdateNetworkPolicyWithTimeout_CancelPerIteration verifies that
// context cancel functions are called after each iteration rather than deferred
// to function return, so leaked timer goroutines don't accumulate across retries.
// It exercises the retry path by injecting a client that always returns errors,
// then cancelling the outer context to verify the wait-for-retry select exits promptly.
func TestCreateOrUpdateNetworkPolicyWithTimeout_CancelPerIteration(t *testing.T) {
	scheme := testutil.SetupTestScheme(t)

	np := &networkingv1.NetworkPolicy{
		ObjectMeta: metav1.ObjectMeta{Name: "test-np", Namespace: "default"},
	}
	owner := &corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{Name: "default"},
	}

	// Use a short outer context to exercise the ctx.Done() branch in the retry wait.
	ctx, cancel := context.WithTimeout(context.Background(), 200*time.Millisecond)
	defer cancel()

	// Inject errors on every Create/Update so every attempt fails and triggers a retry.
	fakeClient := fake.NewClientBuilder().
		WithScheme(scheme).
		WithInterceptorFuncs(interceptor.Funcs{
			Create: func(_ context.Context, _ client.WithWatch, _ client.Object, _ ...client.CreateOption) error {
				return fmt.Errorf("injected create error")
			},
			Update: func(_ context.Context, _ client.WithWatch, _ client.Object, _ ...client.UpdateOption) error {
				return fmt.Errorf("injected update error")
			},
		}).
		Build()

	start := time.Now()
	err := CreateOrUpdateNetworkPolicyWithTimeout(ctx, fakeClient, scheme, owner, np, 50*time.Millisecond, 5)
	elapsed := time.Since(start)

	// Must return an error (context deadline or retry exhaustion) within a short
	// wall-clock window — verifies we don't block waiting for stale timer goroutines.
	require.Error(t, err)
	assert.Less(t, elapsed, 2*time.Second,
		"function must return promptly after context cancellation, not block on leaked timers")
}

// TestResolveDNSToCIDRs_ContextTimeout verifies that a cancelled context causes
// resolveDNSToCIDRs to return an empty slice immediately rather than blocking.
func TestResolveDNSToCIDRs_ContextTimeout(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // pre-cancel so every lookup sees a done context

	cidrs, err := resolveDNSToCIDRs(ctx, []string{"api.example.com", "*.example.org"})
	require.NoError(t, err)
	assert.Empty(t, cidrs, "cancelled context should produce no CIDRs")
}

// TestResolveDNSToCIDRs_WildcardAll verifies that "*" still returns 0.0.0.0/0
// even when the context is cancelled.
func TestResolveDNSToCIDRs_WildcardAll(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	cidrs, err := resolveDNSToCIDRs(ctx, []string{"*"})
	require.NoError(t, err)
	require.Len(t, cidrs, 1)
	assert.Equal(t, "0.0.0.0/0", cidrs[0])
}

func TestDeleteConfigMap_Exists(t *testing.T) {
	scheme := testutil.SetupTestScheme(t)
	cm := &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{Name: "my-cm", Namespace: "default"},
	}
	c := fake.NewClientBuilder().WithScheme(scheme).WithObjects(cm).Build()

	err := DeleteConfigMap(context.Background(), c, "my-cm", "default")
	require.NoError(t, err)

	// Verify deletion
	got := &corev1.ConfigMap{}
	getErr := c.Get(context.Background(), client.ObjectKey{Name: "my-cm", Namespace: "default"}, got)
	assert.True(t, client.IgnoreNotFound(getErr) == nil && getErr != nil, "expected not-found after deletion")
}

func TestDeleteConfigMap_AlreadyGone(t *testing.T) {
	scheme := testutil.SetupTestScheme(t)
	c := fake.NewClientBuilder().WithScheme(scheme).Build()

	// ConfigMap does not exist — must return nil, not an error
	err := DeleteConfigMap(context.Background(), c, "missing-cm", "default")
	require.NoError(t, err)
}

func TestSetCondition_AddNew(t *testing.T) {
	var conditions []metav1.Condition
	changed := SetCondition(&conditions, "Ready", metav1.ConditionTrue, "ReconcileSuccess", "all good", 1)
	assert.True(t, changed)
	require.Len(t, conditions, 1)
	assert.Equal(t, "Ready", conditions[0].Type)
	assert.Equal(t, metav1.ConditionTrue, conditions[0].Status)
	assert.Equal(t, "ReconcileSuccess", conditions[0].Reason)
}

func TestSetCondition_UpdateExistingStatusChange(t *testing.T) {
	var conditions []metav1.Condition
	SetCondition(&conditions, "Ready", metav1.ConditionTrue, "ReconcileSuccess", "all good", 1)
	originalTime := conditions[0].LastTransitionTime

	// Status changes from True to False → LastTransitionTime should be updated
	changed := SetCondition(&conditions, "Ready", metav1.ConditionFalse, "Error", "something failed", 2)
	assert.True(t, changed)
	require.Len(t, conditions, 1, "should update in-place, not append")
	assert.Equal(t, metav1.ConditionFalse, conditions[0].Status)
	assert.Equal(t, "Error", conditions[0].Reason)
	// LastTransitionTime must differ when status changes
	assert.NotEqual(t, originalTime, conditions[0].LastTransitionTime)
}

func TestSetCondition_UpdateNoStatusChange(t *testing.T) {
	var conditions []metav1.Condition
	SetCondition(&conditions, "Ready", metav1.ConditionTrue, "ReconcileSuccess", "all good", 1)
	originalTime := conditions[0].LastTransitionTime

	// Same status, different reason → updated but LastTransitionTime preserved
	changed := SetCondition(&conditions, "Ready", metav1.ConditionTrue, "NewReason", "updated message", 2)
	assert.True(t, changed)
	require.Len(t, conditions, 1)
	assert.Equal(t, "NewReason", conditions[0].Reason)
	assert.Equal(t, originalTime, conditions[0].LastTransitionTime, "LastTransitionTime must not change when status is unchanged")
}

func TestSetCondition_NoOpWhenUnchanged(t *testing.T) {
	var conditions []metav1.Condition
	SetCondition(&conditions, "Ready", metav1.ConditionTrue, "ReconcileSuccess", "all good", 1)

	// Identical call must return false (nothing changed)
	changed := SetCondition(&conditions, "Ready", metav1.ConditionTrue, "ReconcileSuccess", "all good", 1)
	assert.False(t, changed)
	require.Len(t, conditions, 1)
}

func TestBuildEgressNetworkPolicy_CIDRRule(t *testing.T) {
	c := newFakeClient(t).Build()
	policy := BuildEgressNetworkPolicy(
		context.Background(), c,
		"test-policy", "default",
		map[string]string{"app": "test"},
		"",
		&langopv1alpha1.AgentNetworkPolicies{
			Egress: []langopv1alpha1.NetworkEgressRule{
				{
					To: []langopv1alpha1.NetworkPeer{{CIDR: "192.168.0.0/24"}},
					Ports: []langopv1alpha1.NetworkPort{
						{Port: 443, Protocol: "TCP"},
						{Port: 8443, Protocol: "TCP"},
					},
				},
			},
		},
	)
	require.NotNil(t, policy)

	var found bool
	for _, rule := range policy.Spec.Egress {
		for _, peer := range rule.To {
			if peer.IPBlock != nil && peer.IPBlock.CIDR == "192.168.0.0/24" {
				found = true
				require.Len(t, rule.Ports, 2)
				assert.Equal(t, int32(443), rule.Ports[0].Port.IntVal)
				assert.Equal(t, int32(8443), rule.Ports[1].Port.IntVal)
			}
		}
	}
	assert.True(t, found, "expected egress rule with CIDR 192.168.0.0/24 and ports 443/8443")
}

func TestBuildEgressNetworkPolicy_OTELEndpointAddsEgressRule(t *testing.T) {
	c := newFakeClient(t).Build()
	policy := BuildEgressNetworkPolicy(
		context.Background(), c,
		"test-policy", "default",
		map[string]string{"app": "test"},
		"otel-collector.observability.svc.cluster.local:4317",
		nil,
	)
	require.NotNil(t, policy)

	var found bool
	for _, rule := range policy.Spec.Egress {
		for _, peer := range rule.To {
			if peer.NamespaceSelector != nil {
				v, ok := peer.NamespaceSelector.MatchLabels[LabelKeyMetadataName]
				if ok && v == "observability" {
					found = true
					// Rule should include OTEL gRPC and HTTP ports
					var portVals []int32
					for _, p := range rule.Ports {
						portVals = append(portVals, p.Port.IntVal)
					}
					assert.Contains(t, portVals, int32(OTELGRPCPort))
					assert.Contains(t, portVals, int32(OTELHTTPPort))
				}
			}
		}
	}
	assert.True(t, found, "expected egress rule targeting observability namespace for OTEL collector")
}

func TestBuildIngressPeerFromNetworkPeer_CIDR(t *testing.T) {
	peer := &langopv1alpha1.NetworkPeer{CIDR: "10.0.0.0/8"}
	result := buildIngressPeerFromNetworkPeer(peer, "default")
	require.NotNil(t, result.IPBlock)
	assert.Equal(t, "10.0.0.0/8", result.IPBlock.CIDR)
	assert.Nil(t, result.PodSelector)
	assert.Nil(t, result.NamespaceSelector)
}

func TestBuildIngressPeerFromNetworkPeer_Service(t *testing.T) {
	peer := &langopv1alpha1.NetworkPeer{
		Service: &langopv1alpha1.ServiceReference{Name: "my-svc", Namespace: "other-ns"},
	}
	result := buildIngressPeerFromNetworkPeer(peer, "default")
	require.NotNil(t, result.NamespaceSelector)
	assert.Equal(t, "other-ns", result.NamespaceSelector.MatchLabels[LabelKeyMetadataName])
}

func TestBuildIngressPeerFromNetworkPeer_ServiceDefaultsNamespace(t *testing.T) {
	peer := &langopv1alpha1.NetworkPeer{
		Service: &langopv1alpha1.ServiceReference{Name: "my-svc"},
	}
	result := buildIngressPeerFromNetworkPeer(peer, "my-ns")
	require.NotNil(t, result.NamespaceSelector)
	assert.Equal(t, "my-ns", result.NamespaceSelector.MatchLabels[LabelKeyMetadataName])
}

func TestBuildIngressPeerFromNetworkPeer_NamespaceAndPodSelector(t *testing.T) {
	nsSel := &metav1.LabelSelector{MatchLabels: map[string]string{"env": "prod"}}
	podSel := &metav1.LabelSelector{MatchLabels: map[string]string{"role": "api"}}
	peer := &langopv1alpha1.NetworkPeer{
		NamespaceSelector: nsSel,
		PodSelector:       podSel,
	}
	result := buildIngressPeerFromNetworkPeer(peer, "default")
	require.NotNil(t, result.NamespaceSelector)
	require.NotNil(t, result.PodSelector)
	assert.Equal(t, "prod", result.NamespaceSelector.MatchLabels["env"])
	assert.Equal(t, "api", result.PodSelector.MatchLabels["role"])
}
