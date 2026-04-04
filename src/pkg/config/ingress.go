package config

import (
	"context"
	"strings"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

// DetectIngressControllerNamespace attempts to find the namespace the ingress
// controller runs in, given the ingress class name. Returns empty string if
// detection fails (no NetworkPolicy rule will be added).
//
// Detection order:
//  1. meta.helm.sh/release-namespace annotation on the IngressClass
//  2. Pod label scan: app.kubernetes.io/name=<controller-shortname>
func DetectIngressControllerNamespace(ctx context.Context, clientset kubernetes.Interface, ingressClassName string) string {
	if ingressClassName == "" {
		return ""
	}

	ic, err := clientset.NetworkingV1().IngressClasses().Get(ctx, ingressClassName, metav1.GetOptions{})
	if err != nil {
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

	pods, err := clientset.CoreV1().Pods("").List(ctx, metav1.ListOptions{
		LabelSelector: "app.kubernetes.io/name=" + controllerShortName,
	})
	if err != nil || len(pods.Items) == 0 {
		return ""
	}

	return pods.Items[0].Namespace
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
	// Strip "-controller" suffix so "traefik-controller" → "traefik"
	if name, _, found := strings.Cut(controller, "-controller"); found {
		return name
	}
	return controller
}
