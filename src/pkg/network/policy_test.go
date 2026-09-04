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
	"testing"

	corev1 "k8s.io/api/core/v1"
	networkingv1 "k8s.io/api/networking/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
)

func TestCreateSecureAPIServerEgressRules(t *testing.T) {
	// Ensure networkingv1 import is used
	var _ []networkingv1.NetworkPolicyEgressRule

	tests := []struct {
		name          string
		provider      CNIProvider
		expectedRules int
		expectCIDR    bool
		expectNS      bool
	}{
		{
			name:          "cilium provider",
			provider:      CNIProviderCilium,
			expectedRules: 3, // namespace + CIDR rules, plus the unrestricted-peer API server rule
			expectCIDR:    true,
			expectNS:      true,
		},
		{
			name:          "calico provider",
			provider:      CNIProviderCalico,
			expectedRules: 2, // CIDR rules, plus the unrestricted-peer API server rule
			expectCIDR:    true,
			expectNS:      false,
		},
		{
			name:          "generic provider",
			provider:      CNIProviderGeneric,
			expectedRules: 2, // namespace rules, plus the unrestricted-peer API server rule
			expectCIDR:    false,
			expectNS:      true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rules := CreateSecureAPIServerEgressRules(tt.provider, DefaultAPIServerPorts)

			if len(rules) != tt.expectedRules {
				t.Errorf("CreateSecureAPIServerEgressRules() returned %d rules, want %d", len(rules), tt.expectedRules)
			}

			// Check for expected rule types
			hasCIDR := false
			hasNamespace := false

			for _, rule := range rules {
				// Check ports
				if len(rule.Ports) != 2 {
					t.Errorf("Expected 2 ports per rule, got %d", len(rule.Ports))
				}

				// Verify ports are 443 and 6443
				ports := make([]int32, len(rule.Ports))
				for i, port := range rule.Ports {
					if port.Port.Type != intstr.Int {
						t.Error("Port should be integer type")
					}
					if port.Protocol == nil || *port.Protocol != corev1.ProtocolTCP {
						t.Error("Port should be TCP")
					}
					ports[i] = port.Port.IntVal
				}

				if !(contains(ports, 443) && contains(ports, 6443)) {
					t.Error("Rules should contain ports 443 and 6443")
				}

				// Check peer types
				for _, peer := range rule.To {
					if peer.IPBlock != nil {
						hasCIDR = true
						// Validate CIDR format
						cidr := peer.IPBlock.CIDR
						if cidr != "192.168.0.0/16" && cidr != "10.0.0.0/8" && cidr != "172.16.0.0/12" {
							t.Errorf("Unexpected CIDR: %s", cidr)
						}
					}
					if peer.NamespaceSelector != nil {
						hasNamespace = true
						// Validate namespace selector
						labels := peer.NamespaceSelector.MatchLabels
						name, ok := labels["kubernetes.io/metadata.name"]
						if !ok {
							t.Error("Namespace selector should have metadata.name label")
						}
						if name != "default" && name != "kube-system" {
							t.Errorf("Unexpected namespace: %s", name)
						}
					}
				}
			}

			if hasCIDR != tt.expectCIDR {
				t.Errorf("Expected CIDR rules: %v, got: %v", tt.expectCIDR, hasCIDR)
			}
			if hasNamespace != tt.expectNS {
				t.Errorf("Expected namespace rules: %v, got: %v", tt.expectNS, hasNamespace)
			}
		})
	}
}

func TestCiliumCompatibleAPIEgress(t *testing.T) {
	rules := createCiliumCompatibleAPIEgress()

	if len(rules) != 2 {
		t.Errorf("Expected 2 rules for Cilium, got %d", len(rules))
	}

	// First rule should be namespace-based
	nsRule := rules[0]
	if len(nsRule.To) != 1 || nsRule.To[0].NamespaceSelector == nil {
		t.Error("First rule should have namespace selector")
	}

	// Second rule should be CIDR-based
	cidrRule := rules[1]
	if len(cidrRule.To) != 3 {
		t.Error("Second rule should have 3 CIDR blocks")
	}
	for _, peer := range cidrRule.To {
		if peer.IPBlock == nil {
			t.Error("CIDR rule should have IPBlock")
		}
	}
}

func TestCalicoCompatibleAPIEgress(t *testing.T) {
	rules := createCalicoCompatibleAPIEgress()

	if len(rules) != 1 {
		t.Errorf("Expected 1 rule for Calico, got %d", len(rules))
	}

	// Should be CIDR-based only
	rule := rules[0]
	if len(rule.To) != 3 {
		t.Error("Rule should have 3 CIDR blocks")
	}
	for _, peer := range rule.To {
		if peer.IPBlock == nil {
			t.Error("All peers should have IPBlock for Calico")
		}
	}
}

func TestGenericAPIEgress(t *testing.T) {
	rules := createGenericAPIEgress()

	if len(rules) != 1 {
		t.Errorf("Expected 1 rule for generic CNI, got %d", len(rules))
	}

	// Should be namespace-based only
	rule := rules[0]
	if len(rule.To) != 2 {
		t.Error("Rule should target 2 namespaces")
	}
	for _, peer := range rule.To {
		if peer.NamespaceSelector == nil {
			t.Error("All peers should have NamespaceSelector for generic CNI")
		}
	}
}

// contains checks if a slice contains an element
func contains(slice []int32, item int32) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}

// TestCreateSecureAPIServerEgressRules_UnrestrictedPeerRule pins the rule that actually
// lets an agent reach the API server.
//
// Every peer-scoped rule fails to match it: Cilium classifies the API server as a reserved
// entity that neither ipBlock nor namespaceSelector matches, and it is not a pod, so no
// pod selector can match it under any CNI. Only a rule with no `to` reaches it. Without
// this rule the Argo executor cannot post a WorkflowTaskResult, so every task run hangs
// until its deadline and reports Failed even though the agent's work succeeded.
func TestCreateSecureAPIServerEgressRules_UnrestrictedPeerRule(t *testing.T) {
	for _, provider := range []CNIProvider{CNIProviderCilium, CNIProviderCalico, CNIProviderGeneric} {
		t.Run(string(provider), func(t *testing.T) {
			rules := CreateSecureAPIServerEgressRules(provider, []int32{6443})

			var found *networkingv1.NetworkPolicyEgressRule
			for i := range rules {
				if len(rules[i].To) == 0 {
					found = &rules[i]
					break
				}
			}
			if found == nil {
				t.Fatal("no rule with an unrestricted peer; agents cannot reach the API server")
			}
			if len(found.Ports) != 1 || found.Ports[0].Port.IntVal != 6443 {
				t.Errorf("rule should be scoped to the discovered API server port, got %+v", found.Ports)
			}
		})
	}
}

// TestCreateSecureAPIServerEgressRules_UsesDiscoveredPort verifies the discovered port is
// what ends up in the policy. NetworkPolicy is evaluated after DNAT, so naming the Service
// port (443) rather than the endpoint port (6443) silently fails to match.
func TestCreateSecureAPIServerEgressRules_UsesDiscoveredPort(t *testing.T) {
	rules := CreateSecureAPIServerEgressRules(CNIProviderCilium, []int32{9443})

	for _, rule := range rules {
		if len(rule.To) == 0 {
			if len(rule.Ports) != 1 || rule.Ports[0].Port.IntVal != 9443 {
				t.Errorf("expected the discovered port 9443, got %+v", rule.Ports)
			}
			return
		}
	}
	t.Fatal("no unrestricted-peer rule found")
}

// TestCreateSecureAPIServerEgressRules_EmptyPortsFallBack ensures a failed discovery
// degrades to a slightly broader rule rather than locking agents out entirely.
func TestCreateSecureAPIServerEgressRules_EmptyPortsFallBack(t *testing.T) {
	rules := CreateSecureAPIServerEgressRules(CNIProviderCilium, nil)
	for _, rule := range rules {
		if len(rule.To) == 0 {
			if len(rule.Ports) != len(DefaultAPIServerPorts) {
				t.Errorf("expected fallback to %v, got %+v", DefaultAPIServerPorts, rule.Ports)
			}
			return
		}
	}
	t.Fatal("no unrestricted-peer rule found")
}
