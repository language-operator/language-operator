/*
Copyright 2025 Based Team.

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

	corev1 "k8s.io/api/core/v1"
	networkingv1 "k8s.io/api/networking/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/intstr"

	langopv1alpha1 "github.com/based/language-operator/api/v1alpha1"
)

func (r *LanguageClusterReconciler) reconcileNetworkPolicies(ctx context.Context,
	cluster *langopv1alpha1.LanguageCluster,
	namespace string,
	membership map[string]langopv1alpha1.GroupMembershipInfo) ([]string, error) {

	var createdPolicies []string

	// For each security group, create ingress and egress NetworkPolicies
	for _, group := range cluster.Spec.Groups {
		// Skip groups with no members
		if membership[group.Name].Count == 0 {
			continue
		}

		// Create ingress policy
		if len(group.Ingress) > 0 {
			ingressPolicy := buildIngressNetworkPolicy(cluster, group, namespace)
			if err := r.createOrUpdateNetworkPolicy(ctx, ingressPolicy); err != nil {
				return nil, err
			}
			createdPolicies = append(createdPolicies, ingressPolicy.Name)
		}

		// Create egress policy (filter out DNS rules - handled by Cilium)
		egressRules := filterNonDNSRules(group.Egress)
		if len(egressRules) > 0 {
			egressPolicy := buildEgressNetworkPolicy(cluster, group, namespace, egressRules)
			if err := r.createOrUpdateNetworkPolicy(ctx, egressPolicy); err != nil {
				return nil, err
			}
			createdPolicies = append(createdPolicies, egressPolicy.Name)
		}
	}

	return createdPolicies, nil
}

func (r *LanguageClusterReconciler) createOrUpdateNetworkPolicy(ctx context.Context, policy *networkingv1.NetworkPolicy) error {
	existing := &networkingv1.NetworkPolicy{}
	err := r.Get(ctx, types.NamespacedName{Name: policy.Name, Namespace: policy.Namespace}, existing)

	if errors.IsNotFound(err) {
		return r.Create(ctx, policy)
	} else if err != nil {
		return err
	}

	// Update existing policy
	existing.Spec = policy.Spec
	existing.Labels = policy.Labels
	return r.Update(ctx, existing)
}

func buildDefaultDenyPolicy(cluster *langopv1alpha1.LanguageCluster, namespace string) *networkingv1.NetworkPolicy {
	return &networkingv1.NetworkPolicy{
		ObjectMeta: metav1.ObjectMeta{
			Name:      fmt.Sprintf("%s-default-deny", cluster.Name),
			Namespace: namespace,
			Labels: map[string]string{
				"langop.io/cluster": cluster.Name,
				"langop.io/type":    "default-deny",
			},
		},
		Spec: networkingv1.NetworkPolicySpec{
			PodSelector: metav1.LabelSelector{}, // Applies to all pods
			PolicyTypes: []networkingv1.PolicyType{
				networkingv1.PolicyTypeIngress,
				networkingv1.PolicyTypeEgress,
			},
			Ingress: []networkingv1.NetworkPolicyIngressRule{}, // Deny all
			Egress:  []networkingv1.NetworkPolicyEgressRule{},  // Deny all
		},
	}
}

func buildIngressNetworkPolicy(cluster *langopv1alpha1.LanguageCluster,
	group langopv1alpha1.SecurityGroup, namespace string) *networkingv1.NetworkPolicy {

	policy := &networkingv1.NetworkPolicy{
		ObjectMeta: metav1.ObjectMeta{
			Name:      fmt.Sprintf("%s-%s-ingress", cluster.Name, group.Name),
			Namespace: namespace,
			Labels: map[string]string{
				"langop.io/cluster": cluster.Name,
				"langop.io/group":   group.Name,
				"langop.io/type":    "ingress",
			},
		},
		Spec: networkingv1.NetworkPolicySpec{
			PodSelector: metav1.LabelSelector{
				MatchLabels: map[string]string{
					"langop.io/cluster": cluster.Name,
					"langop.io/group":   group.Name,
				},
			},
			PolicyTypes: []networkingv1.PolicyType{networkingv1.PolicyTypeIngress},
			Ingress:     []networkingv1.NetworkPolicyIngressRule{},
		},
	}

	// Convert each rule
	for _, rule := range group.Ingress {
		netpolRule := convertToIngressRule(rule, cluster, namespace)
		policy.Spec.Ingress = append(policy.Spec.Ingress, netpolRule)
	}

	return policy
}

func buildEgressNetworkPolicy(cluster *langopv1alpha1.LanguageCluster,
	group langopv1alpha1.SecurityGroup, namespace string,
	egressRules []langopv1alpha1.NetworkRule) *networkingv1.NetworkPolicy {

	policy := &networkingv1.NetworkPolicy{
		ObjectMeta: metav1.ObjectMeta{
			Name:      fmt.Sprintf("%s-%s-egress", cluster.Name, group.Name),
			Namespace: namespace,
			Labels: map[string]string{
				"langop.io/cluster": cluster.Name,
				"langop.io/group":   group.Name,
				"langop.io/type":    "egress",
			},
		},
		Spec: networkingv1.NetworkPolicySpec{
			PodSelector: metav1.LabelSelector{
				MatchLabels: map[string]string{
					"langop.io/cluster": cluster.Name,
					"langop.io/group":   group.Name,
				},
			},
			PolicyTypes: []networkingv1.PolicyType{networkingv1.PolicyTypeEgress},
			Egress:      []networkingv1.NetworkPolicyEgressRule{},
		},
	}

	// Convert each rule
	for _, rule := range egressRules {
		netpolRule := convertToEgressRule(rule, cluster, namespace)
		policy.Spec.Egress = append(policy.Spec.Egress, netpolRule)
	}

	return policy
}

func convertToIngressRule(rule langopv1alpha1.NetworkRule,
	cluster *langopv1alpha1.LanguageCluster, namespace string) networkingv1.NetworkPolicyIngressRule {

	netpolRule := networkingv1.NetworkPolicyIngressRule{
		Ports: []networkingv1.NetworkPolicyPort{},
		From:  []networkingv1.NetworkPolicyPeer{},
	}

	// Convert ports
	for _, port := range rule.Ports {
		portCopy := port.Port
		protocol := corev1.Protocol(port.Protocol)
		if protocol == "" {
			protocol = corev1.ProtocolTCP
		}
		netpolRule.Ports = append(netpolRule.Ports, networkingv1.NetworkPolicyPort{
			Protocol: &protocol,
			Port:     &intstr.IntOrString{Type: intstr.Int, IntVal: portCopy},
		})
	}

	// Convert peer
	if rule.From != nil {
		peer := convertToPeer(rule.From, cluster, namespace)
		netpolRule.From = append(netpolRule.From, peer)
	}

	return netpolRule
}

func convertToEgressRule(rule langopv1alpha1.NetworkRule,
	cluster *langopv1alpha1.LanguageCluster, namespace string) networkingv1.NetworkPolicyEgressRule {

	netpolRule := networkingv1.NetworkPolicyEgressRule{
		Ports: []networkingv1.NetworkPolicyPort{},
		To:    []networkingv1.NetworkPolicyPeer{},
	}

	// Convert ports
	for _, port := range rule.Ports {
		portCopy := port.Port
		protocol := corev1.Protocol(port.Protocol)
		if protocol == "" {
			protocol = corev1.ProtocolTCP
		}
		netpolRule.Ports = append(netpolRule.Ports, networkingv1.NetworkPolicyPort{
			Protocol: &protocol,
			Port:     &intstr.IntOrString{Type: intstr.Int, IntVal: portCopy},
		})
	}

	// Convert peer
	if rule.To != nil {
		peer := convertToPeer(rule.To, cluster, namespace)
		netpolRule.To = append(netpolRule.To, peer)
	}

	return netpolRule
}

func convertToPeer(peer *langopv1alpha1.NetworkPeer,
	cluster *langopv1alpha1.LanguageCluster, namespace string) networkingv1.NetworkPolicyPeer {

	netpolPeer := networkingv1.NetworkPolicyPeer{}

	// Handle group reference
	if peer.Group != "" {
		netpolPeer.PodSelector = &metav1.LabelSelector{
			MatchLabels: map[string]string{
				"langop.io/cluster": cluster.Name,
				"langop.io/group":   peer.Group,
			},
		}
	}

	// Handle CIDR
	if peer.CIDR != "" {
		netpolPeer.IPBlock = &networkingv1.IPBlock{
			CIDR: peer.CIDR,
		}
	}

	// Handle namespace/pod selectors
	if peer.NamespaceSelector != nil {
		netpolPeer.NamespaceSelector = peer.NamespaceSelector
	}
	if peer.PodSelector != nil {
		netpolPeer.PodSelector = peer.PodSelector
	}

	// Handle service reference
	if peer.Service != nil {
		svcNamespace := peer.Service.Namespace
		if svcNamespace == "" {
			svcNamespace = namespace
		}

		netpolPeer.NamespaceSelector = &metav1.LabelSelector{
			MatchLabels: map[string]string{
				"kubernetes.io/metadata.name": svcNamespace,
			},
		}
	}

	return netpolPeer
}

func filterNonDNSRules(rules []langopv1alpha1.NetworkRule) []langopv1alpha1.NetworkRule {
	var filtered []langopv1alpha1.NetworkRule
	for _, rule := range rules {
		// Skip rules that only have DNS (handled by Cilium)
		if rule.To != nil && len(rule.To.DNS) > 0 && rule.To.Group == "" &&
			rule.To.CIDR == "" && rule.To.Service == nil {
			continue
		}
		filtered = append(filtered, rule)
	}
	return filtered
}
