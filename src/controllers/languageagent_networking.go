package controllers

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"os"

	corev1 "k8s.io/api/core/v1"
	networkingv1 "k8s.io/api/networking/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/utils/ptr"
	"sigs.k8s.io/controller-runtime/pkg/log"

	langopv1alpha1 "github.com/language-operator/language-operator/api/v1alpha1"
	langoplabels "github.com/language-operator/language-operator/pkg/labels"
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
							langoplabels.LabelKeyLangopComponent: "trigger",
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
							langoplabels.LabelKeyLangopKind: "LanguageAgent",
						},
					},
				},
			},
			Ports: npPorts,
		},
	}

	// Allow ingress controller pods to reach agent ports (needed when an Ingress routes external
	// traffic to the agent). Only added when the ingress controller namespace is configured.
	// When auth is enabled the ingress routes to the oauth2-proxy sidecar (4180), not the agent
	// port directly, so we use OAuth2ProxyPort in that case.
	if r.IngressControllerNamespace != "" {
		cluster := &langopv1alpha1.LanguageCluster{}
		clusterFetched := r.Get(ctx, types.NamespacedName{Name: agent.Namespace}, cluster) == nil

		ingressPorts := npPorts
		authEnabled := false
		if clusterFetched {
			var err error
			authEnabled, err = agentAuthEnabled(ctx, r.Client, agent, cluster)
			if err != nil {
				return err
			}
		}
		if authEnabled {
			oauthPort := intstr.FromInt32(OAuth2ProxyPort)
			ingressPorts = []networkingv1.NetworkPolicyPort{
				{Protocol: ptr.To(corev1.ProtocolTCP), Port: &oauthPort},
			}
		}

		networkPolicy.Spec.Ingress = append(networkPolicy.Spec.Ingress, networkingv1.NetworkPolicyIngressRule{
			From: []networkingv1.NetworkPolicyPeer{
				{
					NamespaceSelector: &metav1.LabelSelector{
						MatchLabels: map[string]string{
							langoplabels.LabelKeyMetadataName: r.IngressControllerNamespace,
						},
					},
				},
			},
			Ports: ingressPorts,
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
	labels[langoplabels.LabelKeyLangopComponent] = "agent" // Only route to agent pods, not trigger pods

	// Try to fetch the cluster to determine if auth (and the oauth2-proxy sidecar) is enabled.
	cluster := &langopv1alpha1.LanguageCluster{}
	clusterFetched := r.Get(ctx, types.NamespacedName{Name: agent.Namespace}, cluster) == nil
	authEnabled := false
	if clusterFetched {
		var err error
		authEnabled, err = agentAuthEnabled(ctx, r.Client, agent, cluster)
		if err != nil {
			return err
		}
	}

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
		// Expose the oauth2-proxy sidecar port so the Ingress can route to 4180.
		if authEnabled {
			svcPorts = append(svcPorts, corev1.ServicePort{
				Name:       "oauth2-proxy",
				Port:       OAuth2ProxyPort,
				TargetPort: intstr.FromInt32(OAuth2ProxyPort),
				Protocol:   corev1.ProtocolTCP,
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

	// Reconcile the oauth2-proxy sidecar when auth is enabled for this agent.
	if err := r.reconcileOAuthProxy(ctx, agent, cluster); err != nil {
		return fmt.Errorf("failed to reconcile oauth2-proxy: %w", err)
	}

	log.Info("Creating Ingress for webhook", "hostname", hostname)
	if err := r.reconcileIngress(ctx, agent, cluster, hostname); err != nil {
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

// reconcileIngress creates or updates an Ingress for the agent.
// When auth is enabled, the backend is the agent's oauth2-proxy Service rather than
// the agent Service directly.
func (r *LanguageAgentReconciler) reconcileIngress(ctx context.Context, agent *langopv1alpha1.LanguageAgent, cluster *langopv1alpha1.LanguageCluster, hostname string) error {
	labels := GetCommonLabels(agent.Name, "LanguageAgent")

	authEnabled, err := agentAuthEnabled(ctx, r.Client, agent, cluster)
	if err != nil {
		return err
	}

	ingress := &networkingv1.Ingress{
		ObjectMeta: metav1.ObjectMeta{
			Name:      agent.Name,
			Namespace: agent.Namespace,
			Labels:    labels,
		},
	}

	err = CreateOrUpdateOwned(ctx, r.Client, r.Scheme, agent, ingress, func() error {
		pathType := networkingv1.PathTypePrefix

		// Route to the oauth2-proxy sidecar port when auth is enabled; directly to the agent port otherwise.
		// Both use the same Service (the sidecar port 4180 is added when auth is active).
		var backendPort int32
		if authEnabled {
			backendPort = OAuth2ProxyPort
		} else {
			backendPort = agentIngressPort(agent)
		}

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
												Number: backendPort,
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

		// Apply TLS and IngressClass from cluster config with operator-level defaults.
		ingressClass := r.DefaultIngressClassName
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
				{Hosts: []string{hostname}, SecretName: secretName},
			}
		}
		if cluster.Spec.Ingress != nil && cluster.Spec.Ingress.ClassName != "" {
			ingressClass = cluster.Spec.Ingress.ClassName
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

// reconcileOAuthProxy ensures the per-agent cookie secret exists when auth is enabled.
// The oauth2-proxy itself runs as a sidecar in the agent pod (see buildOAuthProxySidecar).
func (r *LanguageAgentReconciler) reconcileOAuthProxy(ctx context.Context, agent *langopv1alpha1.LanguageAgent, cluster *langopv1alpha1.LanguageCluster) error {
	enabled, err := agentAuthEnabled(ctx, r.Client, agent, cluster)
	if err != nil {
		return err
	}
	if !enabled {
		return nil
	}
	_, err = r.reconcileOAuthCookieSecret(ctx, agent)
	return err
}

// buildOAuthProxySidecar returns an oauth2-proxy container to inject into the agent pod when
// auth is enabled, or nil if auth is not active. The sidecar upstream is localhost so no extra
// Service, NetworkPolicy rule, or cross-pod network hop is required.
func (r *LanguageAgentReconciler) buildOAuthProxySidecar(ctx context.Context, agent *langopv1alpha1.LanguageAgent) (*corev1.Container, error) {
	cluster := &langopv1alpha1.LanguageCluster{}
	if err := r.Get(ctx, types.NamespacedName{Name: agent.Namespace}, cluster); err != nil {
		if apierrors.IsNotFound(err) {
			return nil, nil
		}
		return nil, err
	}
	if enabled, err := agentAuthEnabled(ctx, r.Client, agent, cluster); err != nil {
		return nil, err
	} else if !enabled {
		return nil, nil
	}

	issuerURL := dexIssuerURL(cluster)
	if issuerURL == "" {
		return nil, fmt.Errorf("cannot build oauth2-proxy sidecar: auth is enabled but no OIDC issuer URL could be determined (set spec.domain or spec.auth.oidc.externalIssuerURL)")
	}
	clientID := oauth2ProxyClientID(cluster, agent.Name)

	clientSecretVal, err := r.readOAuthClientSecret(ctx, cluster)
	if err != nil {
		return nil, err
	}

	cookieSecret, err := r.reconcileOAuthCookieSecret(ctx, agent)
	if err != nil {
		return nil, err
	}

	// Upstream is localhost — no network hop, no NetworkPolicy needed.
	upstream := fmt.Sprintf("http://localhost:%d", agentIngressPort(agent))
	hostname := fmt.Sprintf("%s.%s", agent.Name, cluster.Spec.Domain)
	redirectURL := fmt.Sprintf("https://%s/oauth2/callback", hostname)

	emailDomain := "*"
	if cluster.Spec.Auth.OIDC != nil && cluster.Spec.Auth.OIDC.EmailDomain != "" {
		emailDomain = cluster.Spec.Auth.OIDC.EmailDomain
	}

	return &corev1.Container{
		Name:  "oauth2-proxy",
		Image: r.oauth2ProxyImage(),
		Args: []string{
			"--provider=oidc",
			// Use the public issuer URL for iss-claim validation and browser redirects,
			// but skip OIDC discovery and wire internal endpoints directly so
			// oauth2-proxy never needs to reach the public ingress from inside the cluster.
			"--oidc-issuer-url=" + issuerURL,
			"--skip-oidc-discovery=true",
			"--oidc-jwks-url=" + fmt.Sprintf("http://%s.%s.svc.cluster.local:%d/keys", DexResourceName, cluster.Name, DexPort),
			"--login-url=" + issuerURL + "/auth",
			"--redeem-url=" + fmt.Sprintf("http://%s.%s.svc.cluster.local:%d/token", DexResourceName, cluster.Name, DexPort),
			"--client-id=" + clientID,
			"--client-secret=" + clientSecretVal,
			"--cookie-secret=" + cookieSecret,
			"--upstream=" + upstream,
			"--redirect-url=" + redirectURL,
			"--cookie-domain=." + cluster.Spec.Domain,
			"--email-domain=" + emailDomain,
			fmt.Sprintf("--http-address=0.0.0.0:%d", OAuth2ProxyPort),
			"--skip-provider-button=true",
			"--skip-auth-regex=^/oauth2/",
		},
		Ports: []corev1.ContainerPort{
			{Name: "http", ContainerPort: OAuth2ProxyPort, Protocol: corev1.ProtocolTCP},
		},
		Resources: corev1.ResourceRequirements{
			Requests: corev1.ResourceList{
				corev1.ResourceCPU:    resource.MustParse("25m"),
				corev1.ResourceMemory: resource.MustParse("32Mi"),
			},
			Limits: corev1.ResourceList{
				corev1.ResourceCPU:    resource.MustParse("100m"),
				corev1.ResourceMemory: resource.MustParse("128Mi"),
			},
		},
		ReadinessProbe: &corev1.Probe{
			ProbeHandler: corev1.ProbeHandler{
				HTTPGet: &corev1.HTTPGetAction{
					Path: "/ping",
					Port: intstr.FromInt32(OAuth2ProxyPort),
				},
			},
			InitialDelaySeconds: 5,
			PeriodSeconds:       10,
		},
	}, nil
}

// readOAuthClientSecret reads the shared oauth2-proxy client secret from the cluster namespace.
func (r *LanguageAgentReconciler) readOAuthClientSecret(ctx context.Context, cluster *langopv1alpha1.LanguageCluster) (string, error) {
	// For external OIDC, the client secret comes from spec.auth.oidc.clientSecretRef.
	if cluster.Spec.Auth.OIDC != nil && cluster.Spec.Auth.OIDC.ClientSecretRef != nil {
		ref := cluster.Spec.Auth.OIDC.ClientSecretRef
		secret := &corev1.Secret{}
		if err := r.Get(ctx, types.NamespacedName{Name: ref.Name, Namespace: cluster.Name}, secret); err != nil {
			return "", fmt.Errorf("failed to read external client secret %s: %w", ref.Name, err)
		}
		key := ref.Key
		if key == "" {
			key = "api-key"
		}
		return string(secret.Data[key]), nil
	}

	// For embedded Dex, use the auto-generated auth-client-secret.
	secret := &corev1.Secret{}
	if err := r.Get(ctx, types.NamespacedName{Name: DexClientSecretName, Namespace: cluster.Name}, secret); err != nil {
		return "", fmt.Errorf("failed to read dex client secret: %w", err)
	}
	return string(secret.Data[DexClientSecretKey]), nil
}

// reconcileOAuthCookieSecret ensures a per-agent cookie encryption secret exists and returns its value.
func (r *LanguageAgentReconciler) reconcileOAuthCookieSecret(ctx context.Context, agent *langopv1alpha1.LanguageAgent) (string, error) {
	name := agent.Name + OAuth2ProxySuffix + "-cookie"
	key := types.NamespacedName{Name: name, Namespace: agent.Namespace}
	existing := &corev1.Secret{}
	if err := r.Get(ctx, key, existing); err == nil {
		return string(existing.Data[OAuth2ProxyCookieSecretKey]), nil
	} else if !apierrors.IsNotFound(err) {
		return "", fmt.Errorf("failed to get cookie secret: %w", err)
	}

	// Generate a random 32-byte cookie secret (oauth2-proxy requires 16, 24, or 32 bytes).
	raw := make([]byte, 32)
	if _, err := rand.Read(raw); err != nil {
		return "", fmt.Errorf("failed to generate cookie secret: %w", err)
	}
	cookieSecret := base64.URLEncoding.EncodeToString(raw)

	obj := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{Name: name, Namespace: agent.Namespace},
		StringData: map[string]string{OAuth2ProxyCookieSecretKey: cookieSecret},
	}
	if err := CreateOrUpdateOwned(ctx, r.Client, r.Scheme, agent, obj, func() error { return nil }); err != nil {
		return "", fmt.Errorf("failed to create cookie secret: %w", err)
	}
	return cookieSecret, nil
}

// oauth2ProxyImage returns the oauth2-proxy container image to use.
func (r *LanguageAgentReconciler) oauth2ProxyImage() string {
	if r.OAuth2ProxyImage != "" {
		return r.OAuth2ProxyImage
	}
	return "quay.io/oauth2-proxy/oauth2-proxy:v7.6.0"
}
