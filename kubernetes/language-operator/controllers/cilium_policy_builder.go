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
	"strings"

	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/types"

	langopv1alpha1 "github.com/based/language-operator/api/v1alpha1"
)

func (r *LanguageClusterReconciler) reconcileCiliumPolicies(ctx context.Context,
	cluster *langopv1alpha1.LanguageCluster,
	namespace string,
	membership map[string]langopv1alpha1.GroupMembershipInfo) ([]string, error) {

	var createdPolicies []string

	// Only create Cilium policies for rules that need DNS/L7 features
	for _, group := range cluster.Spec.Groups {
		if membership[group.Name].Count == 0 {
			continue
		}

		// Check if any egress rules use DNS
		hasDNSRules := false
		for _, rule := range group.Egress {
			if rule.To != nil && len(rule.To.DNS) > 0 {
				hasDNSRules = true
				break
			}
		}

		if hasDNSRules {
			dnsPolicy := buildCiliumDNSPolicy(cluster, group, namespace)
			if err := r.createOrUpdateCiliumPolicy(ctx, dnsPolicy); err != nil {
				return nil, err
			}
			createdPolicies = append(createdPolicies, dnsPolicy.GetName())
		}
	}

	return createdPolicies, nil
}

func (r *LanguageClusterReconciler) createOrUpdateCiliumPolicy(ctx context.Context, policy *unstructured.Unstructured) error {
	existing := &unstructured.Unstructured{}
	existing.SetGroupVersionKind(policy.GroupVersionKind())

	err := r.Get(ctx, types.NamespacedName{
		Name:      policy.GetName(),
		Namespace: policy.GetNamespace(),
	}, existing)

	if errors.IsNotFound(err) {
		return r.Create(ctx, policy)
	} else if err != nil {
		return err
	}

	// Update existing policy
	policy.SetResourceVersion(existing.GetResourceVersion())
	return r.Update(ctx, policy)
}

func buildCiliumDNSPolicy(cluster *langopv1alpha1.LanguageCluster,
	group langopv1alpha1.SecurityGroup, namespace string) *unstructured.Unstructured {

	policy := &unstructured.Unstructured{
		Object: map[string]interface{}{
			"apiVersion": "cilium.io/v2",
			"kind":       "CiliumNetworkPolicy",
			"metadata": map[string]interface{}{
				"name":      fmt.Sprintf("%s-%s-dns-egress", cluster.Name, group.Name),
				"namespace": namespace,
				"labels": map[string]interface{}{
					"langop.io/cluster": cluster.Name,
					"langop.io/group":   group.Name,
					"langop.io/type":    "cilium-dns",
				},
			},
			"spec": map[string]interface{}{
				"endpointSelector": map[string]interface{}{
					"matchLabels": map[string]interface{}{
						"langop.io/cluster": cluster.Name,
						"langop.io/group":   group.Name,
					},
				},
				"egress": []interface{}{},
			},
		},
	}

	egressRules := []interface{}{}

	for _, rule := range group.Egress {
		if rule.To != nil && len(rule.To.DNS) > 0 {
			ciliumRule := buildCiliumDNSEgressRule(rule)
			egressRules = append(egressRules, ciliumRule)
		}
	}

	policy.Object["spec"].(map[string]interface{})["egress"] = egressRules

	return policy
}

func buildCiliumDNSEgressRule(rule langopv1alpha1.NetworkRule) map[string]interface{} {
	ciliumRule := map[string]interface{}{
		"toFQDNs": []interface{}{},
		"toPorts": []interface{}{},
	}

	// Add DNS patterns
	fqdns := []interface{}{}
	for _, dns := range rule.To.DNS {
		if strings.Contains(dns, "*") {
			// Wildcard pattern
			fqdns = append(fqdns, map[string]interface{}{
				"matchPattern": dns,
			})
		} else {
			// Exact match
			fqdns = append(fqdns, map[string]interface{}{
				"matchName": dns,
			})
		}
	}
	ciliumRule["toFQDNs"] = fqdns

	// Add ports
	ports := []interface{}{}
	for _, port := range rule.Ports {
		ports = append(ports, map[string]interface{}{
			"ports": []interface{}{
				map[string]interface{}{
					"port":     fmt.Sprintf("%d", port.Port),
					"protocol": string(port.Protocol),
				},
			},
		})
	}
	ciliumRule["toPorts"] = ports

	return ciliumRule
}
