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
	"math"
	"math/rand"
	"net"
	"net/url"
	"strings"
	"time"

	corev1 "k8s.io/api/core/v1"
	networkingv1 "k8s.io/api/networking/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/utils/ptr"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/log"

	"github.com/go-logr/logr"
	langopv1alpha1 "github.com/language-operator/language-operator/api/v1alpha1"
	langoplabels "github.com/language-operator/language-operator/pkg/labels"
	"github.com/language-operator/language-operator/pkg/network"
)

// CreateOrUpdateNetworkPolicy creates or updates a NetworkPolicy with owner reference.
// Includes timeout and exponential backoff retry logic for slow CNI plugins.
func CreateOrUpdateNetworkPolicy(
	ctx context.Context,
	c client.Client,
	scheme *runtime.Scheme,
	owner client.Object,
	networkPolicy *networkingv1.NetworkPolicy,
) error {
	return CreateOrUpdateNetworkPolicyWithTimeout(ctx, c, scheme, owner, networkPolicy, 30*time.Second, 3)
}

// CreateOrUpdateNetworkPolicyWithTimeout creates or updates a NetworkPolicy with configurable timeout and retries.
func CreateOrUpdateNetworkPolicyWithTimeout(
	ctx context.Context,
	c client.Client,
	scheme *runtime.Scheme,
	owner client.Object,
	networkPolicy *networkingv1.NetworkPolicy,
	timeout time.Duration,
	maxRetries int,
) error {
	logger := log.FromContext(ctx).WithValues(
		"networkPolicy", networkPolicy.Name,
		"namespace", networkPolicy.Namespace,
		"timeout", timeout,
		"maxRetries", maxRetries,
	)

	// Set owner reference so NetworkPolicy is cleaned up with owner
	if err := controllerutil.SetControllerReference(owner, networkPolicy, scheme); err != nil {
		return fmt.Errorf("failed to set owner reference: %w", err)
	}

	// Retry logic with exponential backoff
	var lastErr error
	for attempt := 0; attempt <= maxRetries; attempt++ {
		// Create timeout context for this attempt and cancel it immediately after
		// use — not via defer, which is function-scoped and would leak the timer
		// goroutine across all remaining loop iterations.
		timeoutCtx, cancel := context.WithTimeout(ctx, timeout)

		start := time.Now()
		err := tryCreateOrUpdateNetworkPolicy(timeoutCtx, c, networkPolicy, logger)
		cancel()
		duration := time.Since(start)

		if err == nil {
			if attempt > 0 {
				logger.Info("NetworkPolicy operation succeeded after retry",
					"attempt", attempt+1,
					"duration", duration)
			} else {
				logger.V(1).Info("NetworkPolicy operation succeeded",
					"duration", duration)
			}
			return nil
		}

		lastErr = err

		// Don't retry on the last attempt
		if attempt == maxRetries {
			break
		}

		// Calculate exponential backoff delay with jitter
		baseDelay := time.Duration(math.Pow(2, float64(attempt))) * time.Second
		jitter := time.Duration(rand.Float64() * float64(baseDelay/2))
		delay := baseDelay + jitter

		logger.Info("NetworkPolicy operation failed, retrying",
			"attempt", attempt+1,
			"duration", duration,
			"error", err.Error(),
			"retryDelay", delay)

		// Wait before retry with context cancellation check
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(delay):
			// Continue to next attempt
		}
	}

	logger.Error(lastErr, "NetworkPolicy operation failed after all retries",
		"maxRetries", maxRetries,
		"totalAttempts", maxRetries+1)

	return fmt.Errorf("NetworkPolicy operation failed after %d attempts: %w", maxRetries+1, lastErr)
}

// tryCreateOrUpdateNetworkPolicy performs a single attempt to create or update a NetworkPolicy.
func tryCreateOrUpdateNetworkPolicy(
	ctx context.Context,
	c client.Client,
	networkPolicy *networkingv1.NetworkPolicy,
	logger logr.Logger,
) error {
	// Try to get existing NetworkPolicy
	existingPolicy := &networkingv1.NetworkPolicy{}
	err := c.Get(ctx, client.ObjectKeyFromObject(networkPolicy), existingPolicy)

	if err != nil {
		if apierrors.IsNotFound(err) {
			// Create new NetworkPolicy
			logger.V(1).Info("Creating new NetworkPolicy")
			if err := c.Create(ctx, networkPolicy); err != nil {
				return fmt.Errorf("failed to create NetworkPolicy: %w", err)
			}
			return nil
		}
		return fmt.Errorf("failed to get NetworkPolicy: %w", err)
	}

	// Update existing NetworkPolicy
	logger.V(1).Info("Updating existing NetworkPolicy")
	existingPolicy.Spec = networkPolicy.Spec
	existingPolicy.Labels = networkPolicy.Labels
	if err := c.Update(ctx, existingPolicy); err != nil {
		return fmt.Errorf("failed to update NetworkPolicy: %w", err)
	}

	return nil
}

// BuildEgressNetworkPolicy creates a NetworkPolicy for egress rules.
// Default policy: deny all external egress, allow internal cluster + DNS.
// DNS-based rules are resolved to IP addresses at policy creation time.
// If otelEndpoint is set, an egress rule is automatically generated for the OpenTelemetry collector.
func BuildEgressNetworkPolicy(
	ctx context.Context,
	client client.Client,
	name, namespace string,
	labels map[string]string,
	otelEndpoint string,
	policies *langopv1alpha1.AgentNetworkPolicies,
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
							langoplabels.LabelKeyMetadataName: "kube-system",
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
					Protocol: ptr.To(corev1.ProtocolUDP),
					Port:     &intstr.IntOrString{Type: intstr.Int, IntVal: network.DNSPort},
				},
				{
					Protocol: ptr.To(corev1.ProtocolTCP),
					Port:     &intstr.IntOrString{Type: intstr.Int, IntVal: network.DNSPort},
				},
			},
		},
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
										langoplabels.LabelKeyMetadataName: otelNamespace,
									},
								},
							},
						},
						Ports: []networkingv1.NetworkPolicyPort{
							{
								Protocol: ptr.To(corev1.ProtocolTCP),
								Port:     &intstr.IntOrString{Type: intstr.Int, IntVal: network.OTELGRPCPort},
							},
							{
								Protocol: ptr.To(corev1.ProtocolTCP),
								Port:     &intstr.IntOrString{Type: intstr.Int, IntVal: network.OTELHTTPPort},
							},
						},
					})
				}
			}
		}
	}

	// Kubernetes API server access rules are handled by CNI-aware function below
	// This was causing duplicate rules that interfered with proper network policy enforcement

	// Add CNI-aware rules for Kubernetes API server access
	// Use CNI detection to generate appropriate egress rules for different CNI implementations
	cniProvider, err := network.DetectCNI(ctx, client)
	if err != nil {
		log.FromContext(ctx).Error(err, "Failed to detect CNI provider, falling back to generic")
		cniProvider = network.CNIProviderGeneric
	}

	// Generate secure API server egress rules based on detected CNI
	apiServerRules := network.CreateSecureAPIServerEgressRules(cniProvider)
	egress = append(egress, apiServerRules...)

	// Add user-defined egress rules.
	// Each NetworkEgressRule has a To slice (multiple peers) and a Ports slice.
	if policies != nil {
		for _, rule := range policies.Egress {
			policyRule := networkingv1.NetworkPolicyEgressRule{}

			for _, peer := range rule.To {
				peer := peer // capture loop variable

				// Handle CIDR-based egress
				if peer.CIDR != "" {
					policyRule.To = append(policyRule.To, networkingv1.NetworkPolicyPeer{
						IPBlock: &networkingv1.IPBlock{
							CIDR: peer.CIDR,
						},
					})
				}

				// Handle DNS-based egress by resolving to IPs at policy creation time.
				// If resolution fails the peer contributes nothing (fail-closed for security).
				if len(peer.DNS) > 0 {
					resolvedCIDRs, err := resolveDNSToCIDRs(ctx, peer.DNS)
					if err == nil {
						for _, cidr := range resolvedCIDRs {
							policyRule.To = append(policyRule.To, networkingv1.NetworkPolicyPeer{
								IPBlock: &networkingv1.IPBlock{CIDR: cidr},
							})
						}
					}
				}

				// Handle Group-based egress (pods with langop.io/group label)
				if peer.Group != "" {
					policyRule.To = append(policyRule.To, networkingv1.NetworkPolicyPeer{
						PodSelector: &metav1.LabelSelector{
							MatchLabels: map[string]string{langoplabels.LabelKeyLangopGroup: peer.Group},
						},
					})
				}

				// Handle Service-based egress (select pods in the service's namespace)
				if peer.Service != nil {
					ns := peer.Service.Namespace
					if ns == "" {
						ns = namespace
					}
					policyRule.To = append(policyRule.To, networkingv1.NetworkPolicyPeer{
						NamespaceSelector: &metav1.LabelSelector{
							MatchLabels: map[string]string{langoplabels.LabelKeyMetadataName: ns},
						},
					})
				}

				// Handle NamespaceSelector and/or PodSelector — combined into one peer so
				// both constraints apply simultaneously.
				if peer.NamespaceSelector != nil || peer.PodSelector != nil {
					policyRule.To = append(policyRule.To, networkingv1.NetworkPolicyPeer{
						NamespaceSelector: peer.NamespaceSelector,
						PodSelector:       peer.PodSelector,
					})
				}
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

			if len(policyRule.To) > 0 || len(policyRule.Ports) > 0 {
				egress = append(egress, policyRule)
			}
		}
	}

	// Ingress rules are assembled per-controller and appended after this function returns.
	var ingress []networkingv1.NetworkPolicyIngressRule

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
			Ingress:     ingress,
		},
	}
}

// buildIngressPeerFromNetworkPeer converts a NetworkPeer to a k8s NetworkPolicyPeer for use in
// ingress rules. DNS is not applicable for ingress sources and is ignored.
func buildIngressPeerFromNetworkPeer(peer *langopv1alpha1.NetworkPeer, namespace string) networkingv1.NetworkPolicyPeer {
	p := networkingv1.NetworkPolicyPeer{}
	if peer.CIDR != "" {
		p.IPBlock = &networkingv1.IPBlock{CIDR: peer.CIDR}
	}
	if peer.Group != "" {
		p.PodSelector = &metav1.LabelSelector{
			MatchLabels: map[string]string{langoplabels.LabelKeyLangopGroup: peer.Group},
		}
	}
	if peer.Service != nil {
		ns := peer.Service.Namespace
		if ns == "" {
			ns = namespace
		}
		p.NamespaceSelector = &metav1.LabelSelector{
			MatchLabels: map[string]string{langoplabels.LabelKeyMetadataName: ns},
		}
	}
	if peer.PodSelector != nil {
		p.PodSelector = peer.PodSelector
	}
	if peer.NamespaceSelector != nil {
		p.NamespaceSelector = peer.NamespaceSelector
	}
	return p
}

// serviceURL returns the in-cluster HTTP URL for a Kubernetes service.
func serviceURL(name, namespace string, port int32) string {
	return fmt.Sprintf("http://%s.%s.svc.cluster.local:%d", name, namespace, port)
}

// mcpToolEndpoint returns the Streamable HTTP MCP URL an agent uses to reach a tool, including
// the /mcp path so the value is directly usable by an MCP client (no path-appending required).
// Sidecar-mode tools are reached over localhost; service-mode tools via the in-cluster Service.
func mcpToolEndpoint(tool *langopv1alpha1.LanguageTool, namespace string) string {
	port := tool.Spec.Port
	if port == 0 {
		port = 8080 // Default MCP port
	}
	if tool.Spec.DeploymentMode == "sidecar" {
		return fmt.Sprintf("http://localhost:%d/mcp", port)
	}
	return serviceURL(tool.Name, namespace, port) + "/mcp"
}

// buildNetworkPolicyIngressRules converts user-defined ingress policies into
// networkingv1.NetworkPolicyIngressRule slices, normalising missing protocol
// values to TCP.
func buildNetworkPolicyIngressRules(
	policies []langopv1alpha1.NetworkIngressRule,
	namespace string,
) []networkingv1.NetworkPolicyIngressRule {
	var rules []networkingv1.NetworkPolicyIngressRule
	for _, rule := range policies {
		ingressRule := networkingv1.NetworkPolicyIngressRule{}
		for _, peer := range rule.From {
			peer := peer
			ingressRule.From = append(ingressRule.From, buildIngressPeerFromNetworkPeer(&peer, namespace))
		}
		for _, p := range rule.Ports {
			protocol := corev1.Protocol(p.Protocol)
			if protocol == "" {
				protocol = corev1.ProtocolTCP
			}
			ingressRule.Ports = append(ingressRule.Ports, networkingv1.NetworkPolicyPort{
				Protocol: ptr.To(protocol),
				Port:     &intstr.IntOrString{Type: intstr.Int, IntVal: p.Port},
			})
		}
		rules = append(rules, ingressRule)
	}
	return rules
}

// resolveDNSToCIDRs resolves DNS hostnames to IP addresses and returns CIDR blocks.
// Supports wildcards: *.example.com will resolve example.com and cache the result.
// Special case: "*" means allow all destinations (0.0.0.0/0).
// Each lookup is bounded by a 3-second per-host timeout derived from ctx to avoid stalling
// the reconcile worker goroutine.
func resolveDNSToCIDRs(ctx context.Context, dnsNames []string) ([]string, error) {
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

		// Resolve the hostname to IP addresses with a per-lookup timeout so a
		// slow or unreachable DNS server cannot stall the reconcile worker.
		lookupCtx, cancel := context.WithTimeout(ctx, 3*time.Second)
		addrs, err := (&net.Resolver{}).LookupIPAddr(lookupCtx, resolveHostname)
		cancel()
		if err != nil {
			// Don't fail the entire policy if one DNS lookup fails
			// Log and continue
			continue
		}

		// Convert IPs to /32 (IPv4) or /128 (IPv6) CIDR blocks
		for _, addr := range addrs {
			ip := addr.IP
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
