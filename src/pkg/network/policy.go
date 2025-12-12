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
	corev1 "k8s.io/api/core/v1"
	networkingv1 "k8s.io/api/networking/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
)

// CreateSecureAPIServerEgressRules creates egress rules for Kubernetes API server access
// based on the detected CNI implementation
func CreateSecureAPIServerEgressRules(cniProvider CNIProvider) []networkingv1.NetworkPolicyEgressRule {
	switch cniProvider {
	case CNIProviderCilium:
		return createCiliumCompatibleAPIEgress()
	case CNIProviderCalico:
		return createCalicoCompatibleAPIEgress()
	default:
		return createGenericAPIEgress()
	}
}

// createCiliumCompatibleAPIEgress creates Cilium-compatible API server egress rules
func createCiliumCompatibleAPIEgress() []networkingv1.NetworkPolicyEgressRule {
	return []networkingv1.NetworkPolicyEgressRule{
		{
			// Allow access to kubernetes service in default namespace
			// This works around Cilium's CIDR resolution issues
			To: []networkingv1.NetworkPolicyPeer{
				{
					NamespaceSelector: &metav1.LabelSelector{
						MatchLabels: map[string]string{
							"kubernetes.io/metadata.name": "default",
						},
					},
				},
			},
			Ports: []networkingv1.NetworkPolicyPort{
				{
					Protocol: protocolPtr(corev1.ProtocolTCP),
					Port:     &intstr.IntOrString{Type: intstr.Int, IntVal: 443},
				},
				{
					Protocol: protocolPtr(corev1.ProtocolTCP),
					Port:     &intstr.IntOrString{Type: intstr.Int, IntVal: 6443},
				},
			},
		},
		{
			// Allow access to internal cluster networks via CIDR
			// These work reliably with Cilium for internal communication
			To: []networkingv1.NetworkPolicyPeer{
				{
					IPBlock: &networkingv1.IPBlock{
						CIDR: "192.168.0.0/16",
					},
				},
				{
					IPBlock: &networkingv1.IPBlock{
						CIDR: "10.0.0.0/8",
					},
				},
				{
					IPBlock: &networkingv1.IPBlock{
						CIDR: "172.16.0.0/12",
					},
				},
			},
			Ports: []networkingv1.NetworkPolicyPort{
				{
					Protocol: protocolPtr(corev1.ProtocolTCP),
					Port:     &intstr.IntOrString{Type: intstr.Int, IntVal: 443},
				},
				{
					Protocol: protocolPtr(corev1.ProtocolTCP),
					Port:     &intstr.IntOrString{Type: intstr.Int, IntVal: 6443},
				},
			},
		},
	}
}

// createCalicoCompatibleAPIEgress creates Calico-compatible API server egress rules
func createCalicoCompatibleAPIEgress() []networkingv1.NetworkPolicyEgressRule {
	return []networkingv1.NetworkPolicyEgressRule{
		{
			// Calico handles CIDR blocks reliably
			To: []networkingv1.NetworkPolicyPeer{
				{
					IPBlock: &networkingv1.IPBlock{
						CIDR: "192.168.0.0/16",
					},
				},
				{
					IPBlock: &networkingv1.IPBlock{
						CIDR: "10.0.0.0/8",
					},
				},
				{
					IPBlock: &networkingv1.IPBlock{
						CIDR: "172.16.0.0/12",
					},
				},
			},
			Ports: []networkingv1.NetworkPolicyPort{
				{
					Protocol: protocolPtr(corev1.ProtocolTCP),
					Port:     &intstr.IntOrString{Type: intstr.Int, IntVal: 443},
				},
				{
					Protocol: protocolPtr(corev1.ProtocolTCP),
					Port:     &intstr.IntOrString{Type: intstr.Int, IntVal: 6443},
				},
			},
		},
	}
}

// createGenericAPIEgress creates generic API server egress rules for standard CNI
func createGenericAPIEgress() []networkingv1.NetworkPolicyEgressRule {
	return []networkingv1.NetworkPolicyEgressRule{
		{
			// Use namespace selectors for maximum compatibility
			To: []networkingv1.NetworkPolicyPeer{
				{
					NamespaceSelector: &metav1.LabelSelector{
						MatchLabels: map[string]string{
							"kubernetes.io/metadata.name": "default",
						},
					},
				},
				{
					NamespaceSelector: &metav1.LabelSelector{
						MatchLabels: map[string]string{
							"kubernetes.io/metadata.name": "kube-system",
						},
					},
				},
			},
			Ports: []networkingv1.NetworkPolicyPort{
				{
					Protocol: protocolPtr(corev1.ProtocolTCP),
					Port:     &intstr.IntOrString{Type: intstr.Int, IntVal: 443},
				},
				{
					Protocol: protocolPtr(corev1.ProtocolTCP),
					Port:     &intstr.IntOrString{Type: intstr.Int, IntVal: 6443},
				},
			},
		},
	}
}

// protocolPtr returns a pointer to the given protocol
func protocolPtr(protocol corev1.Protocol) *corev1.Protocol {
	return &protocol
}
