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
			expectedRules: 2, // namespace + CIDR rules
			expectCIDR:    true,
			expectNS:      true,
		},
		{
			name:          "calico provider",
			provider:      CNIProviderCalico,
			expectedRules: 1, // CIDR rules only
			expectCIDR:    true,
			expectNS:      false,
		},
		{
			name:          "generic provider",
			provider:      CNIProviderGeneric,
			expectedRules: 1, // namespace rules only
			expectCIDR:    false,
			expectNS:      true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rules := CreateSecureAPIServerEgressRules(tt.provider)

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
