package controllers

import (
	"context"
	"fmt"
	"os"

	corev1 "k8s.io/api/core/v1"
	networkingv1 "k8s.io/api/networking/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/utils/ptr"
	"sigs.k8s.io/controller-runtime/pkg/log"

	langopv1alpha1 "github.com/language-operator/language-operator/api/v1alpha1"
	"github.com/language-operator/language-operator/pkg/validation"
)

// agentPorts returns the effective port list for the agent.
// Falls back to a single http:8080 port when spec.ports is not set.
func agentPorts(agent *langopv1alpha1.LanguageAgent) []langopv1alpha1.AgentPort {
	if len(agent.Spec.Ports) > 0 {
		return agent.Spec.Ports
	}
	return []langopv1alpha1.AgentPort{
		{Name: "http", Port: 8080, Protocol: corev1.ProtocolTCP, Expose: ptr.To(true)},
	}
}

// agentIngressPort returns the port that ingress/HTTPRoute should route to.
// Uses the first port with Expose: true; falls back to the first port; falls back to 8080.
func agentIngressPort(agent *langopv1alpha1.LanguageAgent) int32 {
	ports := agentPorts(agent)
	for _, p := range ports {
		if p.Expose != nil && *p.Expose {
			return p.Port
		}
	}
	if len(ports) > 0 {
		return ports[0].Port
	}
	return 8080
}

// buildAgentContainerPorts converts AgentPorts to ContainerPorts for visibility in the Deployment.
// Kubernetes does not enforce declared container ports; this is informational only.
func buildAgentContainerPorts(ports []langopv1alpha1.AgentPort) []corev1.ContainerPort {
	out := make([]corev1.ContainerPort, 0, len(ports))
	for _, p := range ports {
		proto := p.Protocol
		if proto == "" {
			proto = corev1.ProtocolTCP
		}
		out = append(out, corev1.ContainerPort{
			Name:          p.Name,
			ContainerPort: p.Port,
			Protocol:      proto,
		})
	}
	return out
}

func (r *LanguageAgentReconciler) reconcileNetworkPolicy(ctx context.Context, agent *langopv1alpha1.LanguageAgent) error {
	labels := GetCommonLabels(agent.Name, "LanguageAgent")

	// Get OTEL endpoint from operator environment
	// This ensures agents can send traces to the collector
	otelEndpoint := os.Getenv("OTEL_EXPORTER_OTLP_ENDPOINT")

	// Build NetworkPolicy using helper from utils.go
	networkPolicy := BuildEgressNetworkPolicy(
		ctx,
		r.Client,
		agent.Name,
		agent.Namespace,
		labels,
		otelEndpoint,
		agent.Spec.NetworkPolicies,
	)

	// Add ingress rules to allow trigger pods to connect to agent.
	// Build NetworkPolicy port list from all agent ports.
	var npPorts []networkingv1.NetworkPolicyPort
	for _, ap := range agentPorts(agent) {
		proto := ap.Protocol
		if proto == "" {
			proto = corev1.ProtocolTCP
		}
		portCopy := intstr.FromInt32(ap.Port) // copy per iteration to avoid pointer aliasing
		npPorts = append(npPorts, networkingv1.NetworkPolicyPort{
			Protocol: ptr.To(proto),
			Port:     &portCopy,
		})
	}
	networkPolicy.Spec.PolicyTypes = append(networkPolicy.Spec.PolicyTypes, networkingv1.PolicyTypeIngress)
	networkPolicy.Spec.Ingress = []networkingv1.NetworkPolicyIngressRule{
		{
			// Allow trigger pods in same namespace to connect to agent
			From: []networkingv1.NetworkPolicyPeer{
				{
					PodSelector: &metav1.LabelSelector{
						MatchLabels: map[string]string{
							LabelKeyLangopComponent: "trigger",
						},
					},
				},
			},
			Ports: npPorts,
		},
		{
			// Allow other agent pods to connect (agent-to-agent traffic)
			From: []networkingv1.NetworkPolicyPeer{
				{
					PodSelector: &metav1.LabelSelector{
						MatchLabels: map[string]string{
							LabelKeyLangopKind: "LanguageAgent",
						},
					},
				},
			},
			Ports: npPorts,
		},
	}

	// Allow ingress controller pods to reach agent ports (needed when an Ingress routes external
	// traffic to the agent). Only added when the ingress controller namespace is configured.
	if r.IngressControllerNamespace != "" {
		networkPolicy.Spec.Ingress = append(networkPolicy.Spec.Ingress, networkingv1.NetworkPolicyIngressRule{
			From: []networkingv1.NetworkPolicyPeer{
				{
					NamespaceSelector: &metav1.LabelSelector{
						MatchLabels: map[string]string{
							LabelKeyMetadataName: r.IngressControllerNamespace,
						},
					},
				},
			},
			Ports: npPorts,
		})
	}

	// Append user-defined ingress rules from spec.networkPolicies.ingress
	if agent.Spec.NetworkPolicies != nil {
		networkPolicy.Spec.Ingress = append(
			networkPolicy.Spec.Ingress,
			buildNetworkPolicyIngressRules(agent.Spec.NetworkPolicies.Ingress, agent.Namespace)...,
		)
	}

	// Create or update the NetworkPolicy with owner reference and configured timeout/retries
	return CreateOrUpdateNetworkPolicyWithTimeout(ctx, r.Client, r.Scheme, agent, networkPolicy, r.NetworkPolicyTimeout, r.NetworkPolicyRetries)
}

// reconcileService creates a Service for the agent
func (r *LanguageAgentReconciler) reconcileService(ctx context.Context, agent *langopv1alpha1.LanguageAgent) error {
	labels := GetCommonLabels(agent.Name, "LanguageAgent")
	labels[LabelKeyLangopComponent] = "agent" // Only route to agent pods, not trigger pods

	service := &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      agent.Name,
			Namespace: agent.Namespace,
			Labels:    labels,
		},
	}

	err := CreateOrUpdateOwned(ctx, r.Client, r.Scheme, agent, service, func() error {
		svcType := agent.Spec.Deployment.ServiceType
		if svcType == "" {
			svcType = corev1.ServiceTypeClusterIP
		}
		svcPorts := make([]corev1.ServicePort, 0)
		for _, ap := range agentPorts(agent) {
			proto := ap.Protocol
			if proto == "" {
				proto = corev1.ProtocolTCP
			}
			svcPorts = append(svcPorts, corev1.ServicePort{
				Name:       ap.Name,
				Port:       ap.Port,
				TargetPort: intstr.FromInt32(ap.Port),
				Protocol:   proto,
			})
		}
		service.Spec = corev1.ServiceSpec{
			Selector: labels,
			Ports:    svcPorts,
			Type:     svcType,
		}
		if len(agent.Spec.Deployment.ServiceAnnotations) > 0 {
			if service.Annotations == nil {
				service.Annotations = make(map[string]string)
			}
			for k, v := range agent.Spec.Deployment.ServiceAnnotations {
				service.Annotations[k] = v
			}
		}

		return nil
	})

	return err
}

// reconcileWebhooks creates an Ingress for webhook access
func (r *LanguageAgentReconciler) reconcileWebhooks(ctx context.Context, agent *langopv1alpha1.LanguageAgent) error {
	log := log.FromContext(ctx)

	// Get the cluster to check for domain configuration
	cluster := &langopv1alpha1.LanguageCluster{}
	if err := r.Get(ctx, types.NamespacedName{Name: agent.Namespace}, cluster); err != nil {
		if apierrors.IsNotFound(err) {
			log.Info("Cluster not found, skipping webhook reconciliation", "cluster", agent.Namespace)
			return nil
		}
		return err
	}

	// Skip webhook reconciliation if no domain is configured
	if cluster.Spec.Domain == "" {
		log.Info("No domain configured, skipping webhook reconciliation")
		return nil
	}

	// Build agent hostname: <agent-name>.<domain>
	hostname := fmt.Sprintf("%s.%s", agent.Name, cluster.Spec.Domain)

	log.Info("Creating Ingress for webhook", "hostname", hostname)
	if err := r.reconcileIngress(ctx, agent, hostname); err != nil {
		SetCondition(&agent.Status.Conditions, langopv1alpha1.ConditionWebhookRouteCreated, metav1.ConditionFalse, langopv1alpha1.ReasonIngressCreationFailed, err.Error(), agent.Generation)
		return fmt.Errorf("failed to reconcile Ingress: %w", err)
	}
	SetCondition(&agent.Status.Conditions, langopv1alpha1.ConditionWebhookRouteCreated, metav1.ConditionTrue, langopv1alpha1.ReasonIngressCreated, "Ingress created successfully", agent.Generation)

	var routeReady bool
	var routeReadyMsg string
	{
		ready, msg, err := r.checkIngressReadiness(ctx, agent.Name, agent.Namespace)
		if err != nil {
			log.Error(err, "Failed to check Ingress readiness")
			routeReadyMsg = fmt.Sprintf("Failed to check readiness: %v", err)
		} else {
			routeReady = ready
			routeReadyMsg = msg
		}
	}

	// Set WebhookRouteReady condition based on readiness check
	if routeReady {
		SetCondition(&agent.Status.Conditions, langopv1alpha1.ConditionWebhookRouteReady, metav1.ConditionTrue, langopv1alpha1.ReasonWebhookRouteReady, routeReadyMsg, agent.Generation)

		// Only populate WebhookURLs when route is ready
		webhookURL := fmt.Sprintf("https://%s", hostname)
		if agent.Status.WebhookURLs == nil || len(agent.Status.WebhookURLs) == 0 || agent.Status.WebhookURLs[0] != webhookURL {
			agent.Status.WebhookURLs = []string{webhookURL}
			log.Info("Updated webhook URL in status", "url", webhookURL)
		}
	} else {
		SetCondition(&agent.Status.Conditions, langopv1alpha1.ConditionWebhookRouteReady, metav1.ConditionFalse, langopv1alpha1.ReasonWebhookRouteNotReady, routeReadyMsg, agent.Generation)

		// Clear webhook URLs when route is not ready
		if len(agent.Status.WebhookURLs) > 0 {
			agent.Status.WebhookURLs = nil
			log.Info("Cleared webhook URLs from status - route not ready")
		}
	}

	return nil
}

// reconcileIngress creates or updates an Ingress for the agent
func (r *LanguageAgentReconciler) reconcileIngress(ctx context.Context, agent *langopv1alpha1.LanguageAgent, hostname string) error {
	labels := GetCommonLabels(agent.Name, "LanguageAgent")

	ingress := &networkingv1.Ingress{
		ObjectMeta: metav1.ObjectMeta{
			Name:      agent.Name,
			Namespace: agent.Namespace,
			Labels:    labels,
		},
	}

	err := CreateOrUpdateOwned(ctx, r.Client, r.Scheme, agent, ingress, func() error {
		pathType := networkingv1.PathTypePrefix
		ingress.Spec = networkingv1.IngressSpec{
			Rules: []networkingv1.IngressRule{
				{
					Host: hostname,
					IngressRuleValue: networkingv1.IngressRuleValue{
						HTTP: &networkingv1.HTTPIngressRuleValue{
							Paths: []networkingv1.HTTPIngressPath{
								{
									Path:     "/",
									PathType: &pathType,
									Backend: networkingv1.IngressBackend{
										Service: &networkingv1.IngressServiceBackend{
											Name: agent.Name,
											Port: networkingv1.ServiceBackendPort{
												Number: agentIngressPort(agent),
											},
										},
									},
								},
							},
						},
					},
				},
			},
		}

		// Add TLS and ClassName from cluster config (with operator-level defaults)
		ingressClass := r.DefaultIngressClassName
		{
			cluster := &langopv1alpha1.LanguageCluster{}
			if err := r.Get(ctx, types.NamespacedName{Name: agent.Namespace}, cluster); err == nil {
				tlsEnabled := r.DefaultTLSIssuerName != ""
				if cluster.Spec.Ingress != nil && cluster.Spec.Ingress.TLS != nil {
					if cluster.Spec.Ingress.TLS.Enabled != nil {
						tlsEnabled = *cluster.Spec.Ingress.TLS.Enabled
					} else {
						tlsEnabled = true
					}
				}
				if tlsEnabled {
					secretName := ""
					if cluster.Spec.Ingress != nil && cluster.Spec.Ingress.TLS != nil {
						secretName = cluster.Spec.Ingress.TLS.SecretName
					}
					if secretName == "" {
						if r.DefaultTLSIssuerName != "" {
							if ingress.Annotations == nil {
								ingress.Annotations = make(map[string]string)
							}
							kind := r.DefaultTLSIssuerKind
							if kind == "" {
								kind = "ClusterIssuer"
							}
							annotationKey := "cert-manager.io/" + certManagerIssuerAnnotationSuffix(kind)
							ingress.Annotations[annotationKey] = r.DefaultTLSIssuerName
						}
						secretName = GenerateTLSSecretName(agent.Name)
					}
					ingress.Spec.TLS = []networkingv1.IngressTLS{
						{
							Hosts:      []string{hostname},
							SecretName: secretName,
						},
					}
				}

				// Cluster-level ClassName overrides operator default
				if cluster.Spec.Ingress != nil && cluster.Spec.Ingress.ClassName != "" {
					ingressClass = cluster.Spec.Ingress.ClassName
				}
			}
		}
		if ingressClass != "" {
			ingress.Spec.IngressClassName = &ingressClass
		}

		return nil
	})

	return err
}

// validateImageRegistry validates that the agent's container image registry is in the whitelist
func (r *LanguageAgentReconciler) validateImageRegistry(agent *langopv1alpha1.LanguageAgent) error {
	// Skip validation if no whitelist configured
	allowedRegistries := r.RegistryManager.GetRegistries()
	if len(allowedRegistries) == 0 {
		return nil
	}

	return validation.ValidateImageRegistry(agent.Spec.Image, allowedRegistries)
}

// checkIngressReadiness checks if an Ingress is ready to serve traffic
// Returns (isReady, statusMessage, error)
func (r *LanguageAgentReconciler) checkIngressReadiness(ctx context.Context, name, namespace string) (bool, string, error) {
	ingress := &networkingv1.Ingress{}
	err := r.Get(ctx, types.NamespacedName{Name: name, Namespace: namespace}, ingress)
	if err != nil {
		if apierrors.IsNotFound(err) {
			return false, "Ingress not found", nil
		}
		return false, "", fmt.Errorf("failed to get Ingress: %w", err)
	}

	// Check if load balancer is ready
	if len(ingress.Status.LoadBalancer.Ingress) == 0 {
		return false, "Ingress load balancer not ready - no ingress points assigned", nil
	}

	// Check if any ingress point has an IP or hostname
	for _, lbIngress := range ingress.Status.LoadBalancer.Ingress {
		if lbIngress.IP != "" || lbIngress.Hostname != "" {
			return true, "Ingress is ready with load balancer", nil
		}
	}

	return false, "Ingress load balancer assigned but no IP or hostname available", nil
}
