package config

import (
	"context"
	"strings"

	corev1 "k8s.io/api/core/v1"
	networkingv1 "k8s.io/api/networking/v1"
	"k8s.io/apimachinery/pkg/labels"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// DetectIngressControllerNamespace attempts to find the namespace the ingress
// controller runs in, given the ingress class name. Returns empty string if
// detection fails (no NetworkPolicy rule will be added).
//
// Detection order:
//  1. meta.helm.sh/release-namespace annotation on the IngressClass
//  2. Pod label scan: app.kubernetes.io/name=<controller-shortname>
func DetectIngressControllerNamespace(ctx context.Context, c client.Client, ingressClassName string) string {
	if ingressClassName == "" {
		return ""
	}

	ic := &networkingv1.IngressClass{}
	if err := c.Get(ctx, client.ObjectKey{Name: ingressClassName}, ic); err != nil {
		return ""
	}

	// Step 1: Helm release namespace annotation
	if ns, ok := ic.Annotations["meta.helm.sh/release-namespace"]; ok && ns != "" {
		return ns
	}

	// Step 2: Pod label scan using short name from spec.controller
	controllerShortName := extractControllerShortName(ic.Spec.Controller)
	if controllerShortName == "" {
		return ""
	}

	podList := &corev1.PodList{}
	if err := c.List(ctx, podList, &client.ListOptions{
		LabelSelector: labels.SelectorFromSet(labels.Set{
			"app.kubernetes.io/name": controllerShortName,
		}),
	}); err != nil || len(podList.Items) == 0 {
		return ""
	}

	return podList.Items[0].Namespace
}

// extractControllerShortName derives a short name from an IngressClass controller string.
// e.g. "traefik.io/ingress-controller" → "traefik"
//
//	"k8s.io/ingress-nginx"          → "ingress-nginx"
func extractControllerShortName(controller string) string {
	// Strip domain prefix (everything up to and including the first "/")
	if i := strings.Index(controller, "/"); i >= 0 {
		controller = controller[i+1:]
	}
	// Strip "-controller" suffix so "ingress-controller" → "ingress" doesn't happen;
	// but "traefik-controller" → "traefik" does.
	if name, _, found := strings.Cut(controller, "-controller"); found {
		return name
	}
	return controller
}
