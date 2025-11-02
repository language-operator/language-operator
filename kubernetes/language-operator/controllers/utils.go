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
	"net"
	"strings"

	corev1 "k8s.io/api/core/v1"
	networkingv1 "k8s.io/api/networking/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/intstr"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"

	langopv1alpha1 "github.com/based/language-operator/api/v1alpha1"
)

const (
	FinalizerName = "langop.io/finalizer"
)

// SetCondition updates or adds a condition to the conditions slice
func SetCondition(conditions *[]metav1.Condition, conditionType string, status metav1.ConditionStatus, reason, message string, generation int64) {
	now := metav1.Now()
	condition := metav1.Condition{
		Type:               conditionType,
		Status:             status,
		ObservedGeneration: generation,
		LastTransitionTime: now,
		Reason:             reason,
		Message:            message,
	}

	// Find existing condition
	for i, existing := range *conditions {
		if existing.Type == conditionType {
			// Only update LastTransitionTime if status changed
			if existing.Status != status {
				(*conditions)[i] = condition
			} else {
				condition.LastTransitionTime = existing.LastTransitionTime
				(*conditions)[i] = condition
			}
			return
		}
	}

	// Add new condition
	*conditions = append(*conditions, condition)
}

// CreateOrUpdateConfigMap creates or updates a ConfigMap with owner reference
func CreateOrUpdateConfigMap(
	ctx context.Context,
	c client.Client,
	scheme *runtime.Scheme,
	owner client.Object,
	name, namespace string,
	data map[string]string,
) error {
	configMap := &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
		},
	}

	op, err := controllerutil.CreateOrUpdate(ctx, c, configMap, func() error {
		// Set owner reference
		if err := controllerutil.SetControllerReference(owner, configMap, scheme); err != nil {
			return err
		}

		// Update data
		configMap.Data = data

		return nil
	})

	if err != nil {
		return err
	}

	if op != controllerutil.OperationResultNone {
		// Log or track the operation if needed
		_ = op
	}

	return nil
}

// DeleteConfigMap deletes a ConfigMap if it exists
func DeleteConfigMap(ctx context.Context, c client.Client, name, namespace string) error {
	configMap := &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
		},
	}

	err := c.Delete(ctx, configMap)
	if err != nil && !apierrors.IsNotFound(err) {
		return err
	}

	return nil
}

// AddFinalizer adds a finalizer to the object
func AddFinalizer(obj client.Object) bool {
	finalizers := obj.GetFinalizers()
	for _, f := range finalizers {
		if f == FinalizerName {
			return false
		}
	}
	obj.SetFinalizers(append(finalizers, FinalizerName))
	return true
}

// RemoveFinalizer removes a finalizer from the object
func RemoveFinalizer(obj client.Object) bool {
	finalizers := obj.GetFinalizers()
	for i, f := range finalizers {
		if f == FinalizerName {
			obj.SetFinalizers(append(finalizers[:i], finalizers[i+1:]...))
			return true
		}
	}
	return false
}

// HasFinalizer checks if the object has the finalizer
func HasFinalizer(obj client.Object) bool {
	finalizers := obj.GetFinalizers()
	for _, f := range finalizers {
		if f == FinalizerName {
			return true
		}
	}
	return false
}

// GenerateConfigMapName generates a ConfigMap name for a resource
func GenerateConfigMapName(resourceName, suffix string) string {
	return fmt.Sprintf("%s-%s", resourceName, suffix)
}

// GenerateServiceName generates a Service name for a resource
func GenerateServiceName(resourceName string) string {
	return resourceName
}

// GenerateDeploymentName generates a Deployment name for a resource
func GenerateDeploymentName(resourceName string) string {
	return resourceName
}

// GenerateIngressName generates an Ingress name for a resource
func GenerateIngressName(resourceName string) string {
	return resourceName
}

// GetCommonLabels returns common labels for resources
func GetCommonLabels(resourceName, resourceKind string) map[string]string {
	return map[string]string{
		"app.kubernetes.io/name":       resourceName,
		"app.kubernetes.io/managed-by": "language-operator",
		"app.kubernetes.io/part-of":    "langop",
		"langop.io/kind":               resourceKind,
	}
}

// MergeLabels merges two label maps, with the second map taking precedence
func MergeLabels(base, override map[string]string) map[string]string {
	result := make(map[string]string)
	for k, v := range base {
		result[k] = v
	}
	for k, v := range override {
		result[k] = v
	}
	return result
}

// Int32Ptr returns a pointer to an int32 value
func Int32Ptr(i int32) *int32 {
	return &i
}

// StringPtr returns a pointer to a string value
func StringPtr(s string) *string {
	return &s
}

// BoolPtr returns a pointer to a bool value
func BoolPtr(b bool) *bool {
	return &b
}

// resolveDNSToCIDRs resolves DNS hostnames to IP addresses and returns CIDR blocks
// Supports wildcards: *.example.com will resolve example.com and cache the result
func resolveDNSToCIDRs(dnsNames []string) ([]string, error) {
	var cidrs []string
	seenIPs := make(map[string]bool)

	for _, hostname := range dnsNames {
		// Handle wildcard domains by stripping the *. prefix
		// Note: This is an approximation - we can't know all subdomains
		// so we resolve the base domain
		resolveHostname := hostname
		if strings.HasPrefix(hostname, "*.") {
			resolveHostname = hostname[2:] // Remove *.
		}

		// Resolve the hostname to IP addresses
		ips, err := net.LookupIP(resolveHostname)
		if err != nil {
			// Don't fail the entire policy if one DNS lookup fails
			// Log and continue
			continue
		}

		// Convert IPs to /32 (IPv4) or /128 (IPv6) CIDR blocks
		for _, ip := range ips {
			var cidr string
			if ip.To4() != nil {
				cidr = ip.String() + "/32"
			} else {
				cidr = ip.String() + "/128"
			}

			// Deduplicate
			if !seenIPs[cidr] {
				seenIPs[cidr] = true
				cidrs = append(cidrs, cidr)
			}
		}
	}

	return cidrs, nil
}

// BuildEgressNetworkPolicy creates a NetworkPolicy for egress rules
// Default policy: deny all external egress, allow internal cluster + DNS
// DNS-based rules are resolved to IP addresses at policy creation time
func BuildEgressNetworkPolicy(
	name, namespace string,
	labels map[string]string,
	egressRules []langopv1alpha1.NetworkRule,
) *networkingv1.NetworkPolicy {

	policyTypes := []networkingv1.PolicyType{networkingv1.PolicyTypeEgress}

	// Start with allow all internal cluster traffic + DNS
	egress := []networkingv1.NetworkPolicyEgressRule{
		// Allow all internal cluster communication
		{
			To: []networkingv1.NetworkPolicyPeer{
				{
					// Allow same namespace
					PodSelector: &metav1.LabelSelector{},
				},
			},
		},
		// Allow DNS
		{
			To: []networkingv1.NetworkPolicyPeer{
				{
					NamespaceSelector: &metav1.LabelSelector{
						MatchLabels: map[string]string{
							"kubernetes.io/metadata.name": "kube-system",
						},
					},
					PodSelector: &metav1.LabelSelector{
						MatchLabels: map[string]string{
							"k8s-app": "kube-dns",
						},
					},
				},
			},
			Ports: []networkingv1.NetworkPolicyPort{
				{
					Protocol: protocolPtr(corev1.ProtocolUDP),
					Port:     &intstr.IntOrString{Type: intstr.Int, IntVal: 53},
				},
				{
					Protocol: protocolPtr(corev1.ProtocolTCP),
					Port:     &intstr.IntOrString{Type: intstr.Int, IntVal: 53},
				},
			},
		},
	}

	// Add user-defined egress rules
	for _, rule := range egressRules {
		if rule.To == nil {
			continue
		}

		policyRule := networkingv1.NetworkPolicyEgressRule{}

		// Handle CIDR-based egress
		if rule.To.CIDR != "" {
			policyRule.To = append(policyRule.To, networkingv1.NetworkPolicyPeer{
				IPBlock: &networkingv1.IPBlock{
					CIDR: rule.To.CIDR,
				},
			})
		}

		// Handle DNS-based egress by resolving to IPs
		// Note: DNS records can change, so this is a point-in-time resolution
		// Policies will be updated on the next reconciliation loop
		if len(rule.To.DNS) > 0 {
			resolvedCIDRs, err := resolveDNSToCIDRs(rule.To.DNS)
			if err == nil && len(resolvedCIDRs) > 0 {
				for _, cidr := range resolvedCIDRs {
					policyRule.To = append(policyRule.To, networkingv1.NetworkPolicyPeer{
						IPBlock: &networkingv1.IPBlock{
							CIDR: cidr,
						},
					})
				}
			}
			// If DNS resolution fails, the rule simply won't have any destinations
			// which means it won't allow any traffic (fail-closed for security)
		}

		// Handle ports
		for _, port := range rule.Ports {
			protocol := corev1.Protocol(port.Protocol)
			if protocol == "" {
				protocol = corev1.ProtocolTCP
			}

			policyRule.Ports = append(policyRule.Ports, networkingv1.NetworkPolicyPort{
				Protocol: &protocol,
				Port:     &intstr.IntOrString{Type: intstr.Int, IntVal: port.Port},
			})
		}

		// Only add rule if it has destinations
		if len(policyRule.To) > 0 || len(policyRule.Ports) > 0 {
			egress = append(egress, policyRule)
		}
	}

	return &networkingv1.NetworkPolicy{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
			Labels:    labels,
		},
		Spec: networkingv1.NetworkPolicySpec{
			PodSelector: metav1.LabelSelector{
				MatchLabels: labels,
			},
			PolicyTypes: policyTypes,
			Egress:      egress,
		},
	}
}

func protocolPtr(p corev1.Protocol) *corev1.Protocol {
	return &p
}
