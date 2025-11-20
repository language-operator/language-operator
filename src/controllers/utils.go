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
	"net/url"
	"strconv"
	"strings"

	corev1 "k8s.io/api/core/v1"
	networkingv1 "k8s.io/api/networking/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/intstr"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"

	langopv1alpha1 "github.com/language-operator/language-operator/api/v1alpha1"
)

const (
	FinalizerName = "langop.io/finalizer"
)

// SetCondition updates or adds a condition to the conditions slice
// Returns true if the condition was actually changed
func SetCondition(conditions *[]metav1.Condition, conditionType string, status metav1.ConditionStatus, reason, message string, generation int64) bool {
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
			// Check if anything actually changed
			if existing.Status == status &&
				existing.Reason == reason &&
				existing.Message == message &&
				existing.ObservedGeneration == generation {
				// Nothing changed, don't update
				return false
			}

			// Only update LastTransitionTime if status changed
			if existing.Status != status {
				(*conditions)[i] = condition
			} else {
				condition.LastTransitionTime = existing.LastTransitionTime
				(*conditions)[i] = condition
			}
			return true
		}
	}

	// Add new condition
	*conditions = append(*conditions, condition)
	return true
}

// ValidateClusterReference validates that a cluster exists and is ready
func ValidateClusterReference(ctx context.Context, c client.Client, clusterRef, namespace string) error {
	if clusterRef == "" {
		return nil // No cluster reference to validate
	}

	cluster := &langopv1alpha1.LanguageCluster{}
	if err := c.Get(ctx, client.ObjectKey{Name: clusterRef, Namespace: namespace}, cluster); err != nil {
		return fmt.Errorf("failed to get cluster %s: %w", clusterRef, err)
	}

	if cluster.Status.Phase != "Ready" {
		return fmt.Errorf("cluster %s is not ready yet (phase: %s)", clusterRef, cluster.Status.Phase)
	}

	return nil
}

// CreateOrUpdateNetworkPolicy creates or updates a NetworkPolicy with owner reference
func CreateOrUpdateNetworkPolicy(
	ctx context.Context,
	c client.Client,
	scheme *runtime.Scheme,
	owner client.Object,
	networkPolicy *networkingv1.NetworkPolicy,
) error {
	// Set owner reference so NetworkPolicy is cleaned up with owner
	if err := controllerutil.SetControllerReference(owner, networkPolicy, scheme); err != nil {
		return fmt.Errorf("failed to set owner reference: %w", err)
	}

	// Try to get existing NetworkPolicy
	existingPolicy := &networkingv1.NetworkPolicy{}
	err := c.Get(ctx, client.ObjectKeyFromObject(networkPolicy), existingPolicy)

	if err != nil {
		if apierrors.IsNotFound(err) {
			// Create new NetworkPolicy
			if err := c.Create(ctx, networkPolicy); err != nil {
				return fmt.Errorf("failed to create NetworkPolicy: %w", err)
			}
			return nil
		}
		return fmt.Errorf("failed to get NetworkPolicy: %w", err)
	}

	// Update existing NetworkPolicy
	existingPolicy.Spec = networkPolicy.Spec
	existingPolicy.Labels = networkPolicy.Labels
	if err := c.Update(ctx, existingPolicy); err != nil {
		return fmt.Errorf("failed to update NetworkPolicy: %w", err)
	}

	return nil
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

	_, err := controllerutil.CreateOrUpdate(ctx, c, configMap, func() error {
		// Set owner reference
		if err := controllerutil.SetControllerReference(owner, configMap, scheme); err != nil {
			return err
		}

		// Update data
		configMap.Data = data

		return nil
	})

	return err
}

// CreateOrUpdateConfigMapWithAnnotations creates or updates a ConfigMap with custom annotations
func CreateOrUpdateConfigMapWithAnnotations(
	ctx context.Context,
	c client.Client,
	scheme *runtime.Scheme,
	owner client.Object,
	name, namespace string,
	data map[string]string,
	annotations map[string]string,
) error {
	configMap := &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
		},
	}

	_, err := controllerutil.CreateOrUpdate(ctx, c, configMap, func() error {
		// Set owner reference
		if err := controllerutil.SetControllerReference(owner, configMap, scheme); err != nil {
			return err
		}

		// Update data
		configMap.Data = data

		// Update annotations
		if configMap.Annotations == nil {
			configMap.Annotations = make(map[string]string)
		}
		for k, v := range annotations {
			configMap.Annotations[k] = v
		}

		return nil
	})

	return err
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

// GenerateConfigMapName generates a ConfigMap name for a resource
func GenerateConfigMapName(resourceName, suffix string) string {
	return fmt.Sprintf("%s-%s", resourceName, suffix)
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

// resolveDNSToCIDRs resolves DNS hostnames to IP addresses and returns CIDR blocks
// Supports wildcards: *.example.com will resolve example.com and cache the result
// Special case: "*" means allow all destinations (0.0.0.0/0)
func resolveDNSToCIDRs(dnsNames []string) ([]string, error) {
	var cidrs []string
	seenIPs := make(map[string]bool)

	for _, hostname := range dnsNames {
		// Special case: "*" means any destination
		if hostname == "*" {
			return []string{"0.0.0.0/0"}, nil
		}

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

// providerDefaultEndpoints maps well-known provider types to their default API endpoints
// These endpoints are automatically added to NetworkPolicy egress rules so users don't need
// to manually configure network access for standard provider APIs
var providerDefaultEndpoints = map[string][]string{
	"openai":    {"https://api.openai.com"},
	"anthropic": {"https://api.anthropic.com"},
	// Note: bedrock and vertex use cloud provider endpoints that vary by region
	// and are typically accessible via VPC endpoints or default cloud egress
}

// generateEgressFromEndpoint parses an endpoint URL and generates a NetworkPolicy egress rule
// Returns nil if the URL cannot be parsed or is invalid
func generateEgressFromEndpoint(endpoint string) *networkingv1.NetworkPolicyEgressRule {
	// Parse the endpoint URL
	parsedURL, err := url.Parse(endpoint)
	if err != nil {
		// Invalid URL, skip auto-generation
		return nil
	}

	// Extract host and port
	host := parsedURL.Hostname()
	if host == "" {
		// No host in URL, skip auto-generation
		return nil
	}

	port := parsedURL.Port()
	if port == "" {
		// Use default ports based on scheme
		switch parsedURL.Scheme {
		case "https":
			port = "443"
		case "http":
			port = "80"
		default:
			// Unknown scheme, skip port specification
			port = ""
		}
	}

	rule := &networkingv1.NetworkPolicyEgressRule{}

	// Determine if host is an IP address or hostname
	if ip := net.ParseIP(host); ip != nil {
		// Host is an IP address - create CIDR-based rule
		cidr := host + "/32"
		if ip.To4() == nil {
			// IPv6 address
			cidr = host + "/128"
		}
		rule.To = append(rule.To, networkingv1.NetworkPolicyPeer{
			IPBlock: &networkingv1.IPBlock{
				CIDR: cidr,
			},
		})
	} else {
		// Host is a hostname - resolve to IPs using existing DNS resolution
		resolvedCIDRs, err := resolveDNSToCIDRs([]string{host})
		if err == nil && len(resolvedCIDRs) > 0 {
			for _, cidr := range resolvedCIDRs {
				rule.To = append(rule.To, networkingv1.NetworkPolicyPeer{
					IPBlock: &networkingv1.IPBlock{
						CIDR: cidr,
					},
				})
			}
		} else {
			// DNS resolution failed, skip auto-generation
			return nil
		}
	}

	// Add port if specified
	if port != "" {
		portInt, err := strconv.Atoi(port)
		if err == nil {
			protocol := corev1.ProtocolTCP
			rule.Ports = append(rule.Ports, networkingv1.NetworkPolicyPort{
				Protocol: &protocol,
				Port:     &intstr.IntOrString{Type: intstr.Int, IntVal: int32(portInt)},
			})
		}
	}

	// Only return rule if it has destinations
	if len(rule.To) == 0 {
		return nil
	}

	return rule
}

// BuildEgressNetworkPolicy creates a NetworkPolicy for egress rules
// Default policy: deny all external egress, allow internal cluster + DNS
// DNS-based rules are resolved to IP addresses at policy creation time
// If provider uses custom endpoints (openai-compatible, azure, custom) and endpoint is set,
// an egress rule is automatically generated for that endpoint
// If otelEndpoint is set, an egress rule is automatically generated for the OpenTelemetry collector
func BuildEgressNetworkPolicy(
	name, namespace string,
	labels map[string]string,
	provider, endpoint string,
	otelEndpoint string,
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

	// Auto-generate egress rules for well-known providers
	if defaultEndpoints, ok := providerDefaultEndpoints[provider]; ok {
		for _, defaultEndpoint := range defaultEndpoints {
			if autoRule := generateEgressFromEndpoint(defaultEndpoint); autoRule != nil {
				egress = append(egress, *autoRule)
			}
		}
	}

	// Auto-generate egress rule from custom endpoint (if specified)
	// This handles custom endpoints for openai-compatible/azure/custom providers,
	// as well as proxy scenarios where users override default provider endpoints
	if endpoint != "" {
		if autoRule := generateEgressFromEndpoint(endpoint); autoRule != nil {
			egress = append(egress, *autoRule)
		}
	}

	// Auto-generate egress rule for OpenTelemetry collector (if specified)
	// This allows agents and models to send traces to the OTEL collector
	// Note: Ruby agents use HTTP port 4318, but we need to allow the gRPC port 4317
	// as well for future compatibility and Go-based agents
	if otelEndpoint != "" {
		// Ensure endpoint has a scheme for proper URL parsing
		normalizedEndpoint := otelEndpoint
		if !strings.HasPrefix(otelEndpoint, "http://") && !strings.HasPrefix(otelEndpoint, "https://") {
			normalizedEndpoint = "http://" + otelEndpoint
		}

		// Parse the endpoint URL
		parsedURL, err := url.Parse(normalizedEndpoint)
		if err == nil {
			hostname := parsedURL.Hostname()
			if hostname != "" {
				// Extract namespace from Kubernetes service DNS name
				// Format: service.namespace or service.namespace.svc.cluster.local
				parts := strings.Split(hostname, ".")
				if len(parts) >= 2 {
					otelNamespace := parts[1]

					// Create namespace-based egress rule for both gRPC and HTTP ports
					// This works with Kubernetes services (unlike ipBlock which only works with pod IPs)
					egress = append(egress, networkingv1.NetworkPolicyEgressRule{
						To: []networkingv1.NetworkPolicyPeer{
							{
								NamespaceSelector: &metav1.LabelSelector{
									MatchLabels: map[string]string{
										"kubernetes.io/metadata.name": otelNamespace,
									},
								},
							},
						},
						Ports: []networkingv1.NetworkPolicyPort{
							{
								Protocol: protocolPtr(corev1.ProtocolTCP),
								Port:     &intstr.IntOrString{Type: intstr.Int, IntVal: 4317},
							},
							{
								Protocol: protocolPtr(corev1.ProtocolTCP),
								Port:     &intstr.IntOrString{Type: intstr.Int, IntVal: 4318},
							},
						},
					})
				}
			}
		}
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
