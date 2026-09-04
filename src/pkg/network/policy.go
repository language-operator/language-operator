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

	corev1 "k8s.io/api/core/v1"
	networkingv1 "k8s.io/api/networking/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/utils/ptr"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"

	langoplabels "github.com/language-operator/language-operator/pkg/labels"
)

// DefaultAPIServerPorts is the fallback used when the API server's real listening
// port cannot be discovered: 6443 is the k3s/kubeadm default, 443 the usual port
// for a managed control plane behind a load balancer.
var DefaultAPIServerPorts = []int32{443, 6443}

// CreateSecureAPIServerEgressRules creates egress rules for Kubernetes API server access
// based on the detected CNI implementation.
//
// apiServerPorts are the ports the API server actually listens on, as discovered from
// the kubernetes Endpoints — see DiscoverAPIServerPorts. This matters because policy is
// evaluated *after* DNAT: a connection to the ClusterIP `10.43.0.1:443` reaches the
// kubelet as `<node-ip>:6443`, and a rule naming only 443 does not match it.
//
// Agent pods must be able to reach the API server: they run as Argo Workflow pods, and
// the executor sidecar reports each node's outcome by creating a WorkflowTaskResult. A
// pod that cannot reach the API server hangs after its work completes instead of
// finishing, so every run fails.
func CreateSecureAPIServerEgressRules(cniProvider CNIProvider, apiServerPorts []int32) []networkingv1.NetworkPolicyEgressRule {
	if len(apiServerPorts) == 0 {
		apiServerPorts = DefaultAPIServerPorts
	}

	var rules []networkingv1.NetworkPolicyEgressRule
	switch cniProvider {
	case CNIProviderCilium:
		rules = createCiliumCompatibleAPIEgress()
	case CNIProviderCalico:
		rules = createCalicoCompatibleAPIEgress()
	default:
		rules = createGenericAPIEgress()
	}

	// The peer-scoped rules above cannot be relied on to match the API server.
	//
	// Cilium classifies it as a reserved entity (`host`/`kube-apiserver`) rather than a
	// CIDR or a pod, and reserved entities are matched by neither `ipBlock` nor
	// `namespaceSelector` peers — verified empirically: with a deny-all-egress baseline,
	// ipBlock rules for the node's /32 and even 0.0.0.0/0 were both blocked, while a rule
	// with no peer restriction on the API server's real port was permitted. The generic
	// namespaceSelector rules have the same problem for a different reason: the API server
	// is not a pod, so no pod selector can ever match it.
	//
	// A rule with no `to` matches any destination, which is the only construction that
	// reliably includes the API server. It is scoped to the API server's own port to keep
	// the exposure narrow — on a typical cluster that is 6443, not a general-purpose port.
	rules = append(rules, networkingv1.NetworkPolicyEgressRule{
		Ports: apiServerPortRules(apiServerPorts),
	})

	return rules
}

// apiServerPortRules converts discovered ports into TCP NetworkPolicy port rules.
func apiServerPortRules(ports []int32) []networkingv1.NetworkPolicyPort {
	out := make([]networkingv1.NetworkPolicyPort, 0, len(ports))
	for _, p := range ports {
		port := intstr.FromInt32(p)
		out = append(out, networkingv1.NetworkPolicyPort{
			Protocol: ptr.To(corev1.ProtocolTCP),
			Port:     &port,
		})
	}
	return out
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
							langoplabels.LabelKeyMetadataName: "default",
						},
					},
				},
			},
			Ports: []networkingv1.NetworkPolicyPort{
				{
					Protocol: ptr.To(corev1.ProtocolTCP),
					Port:     &intstr.IntOrString{Type: intstr.Int, IntVal: 443},
				},
				{
					Protocol: ptr.To(corev1.ProtocolTCP),
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
					Protocol: ptr.To(corev1.ProtocolTCP),
					Port:     &intstr.IntOrString{Type: intstr.Int, IntVal: 443},
				},
				{
					Protocol: ptr.To(corev1.ProtocolTCP),
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
					Protocol: ptr.To(corev1.ProtocolTCP),
					Port:     &intstr.IntOrString{Type: intstr.Int, IntVal: 443},
				},
				{
					Protocol: ptr.To(corev1.ProtocolTCP),
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
							langoplabels.LabelKeyMetadataName: "default",
						},
					},
				},
				{
					NamespaceSelector: &metav1.LabelSelector{
						MatchLabels: map[string]string{
							langoplabels.LabelKeyMetadataName: "kube-system",
						},
					},
				},
			},
			Ports: []networkingv1.NetworkPolicyPort{
				{
					Protocol: ptr.To(corev1.ProtocolTCP),
					Port:     &intstr.IntOrString{Type: intstr.Int, IntVal: 443},
				},
				{
					Protocol: ptr.To(corev1.ProtocolTCP),
					Port:     &intstr.IntOrString{Type: intstr.Int, IntVal: 6443},
				},
			},
		},
	}
}

// DiscoverAPIServerPorts returns the port(s) the Kubernetes API server actually listens
// on, read from the `kubernetes` Endpoints in the default namespace.
//
// This is the endpoint port, not the Service port, and the two commonly differ: on k3s
// and kubeadm the Service is 443 while the endpoint — and therefore the destination
// NetworkPolicy is evaluated against, since policy is applied after DNAT — is 6443.
//
// Returns DefaultAPIServerPorts if the Endpoints cannot be read, so a missing RBAC
// permission or an unusual cluster degrades to a slightly broader rule rather than
// locking every agent out of the API server.
func DiscoverAPIServerPorts(ctx context.Context, c client.Client) []int32 {
	eps := &corev1.Endpoints{}
	if err := c.Get(ctx, types.NamespacedName{Name: "kubernetes", Namespace: "default"}, eps); err != nil {
		log.FromContext(ctx).V(1).Info("Could not read kubernetes Endpoints; using default API server ports",
			"error", err, "ports", DefaultAPIServerPorts)
		return DefaultAPIServerPorts
	}

	seen := map[int32]bool{}
	var ports []int32
	for _, subset := range eps.Subsets {
		for _, p := range subset.Ports {
			if p.Port != 0 && !seen[p.Port] {
				seen[p.Port] = true
				ports = append(ports, p.Port)
			}
		}
	}
	if len(ports) == 0 {
		return DefaultAPIServerPorts
	}
	return ports
}
