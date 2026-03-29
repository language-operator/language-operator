package gen

import (
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	langopv1alpha1 "github.com/language-operator/language-operator/api/v1alpha1"
)

// LanguageClusterModifier is a function that modifies a LanguageCluster.
type LanguageClusterModifier func(*langopv1alpha1.LanguageCluster)

// LanguageCluster constructs a LanguageCluster with the given name and modifiers.
// LanguageCluster is cluster-scoped so no namespace is needed.
func LanguageCluster(name string, mods ...LanguageClusterModifier) *langopv1alpha1.LanguageCluster {
	c := &langopv1alpha1.LanguageCluster{
		ObjectMeta: metav1.ObjectMeta{
			Name: name,
		},
	}
	for _, mod := range mods {
		mod(c)
	}
	return c
}

// LanguageClusterFrom clones an existing LanguageCluster and applies modifiers.
func LanguageClusterFrom(c *langopv1alpha1.LanguageCluster, mods ...LanguageClusterModifier) *langopv1alpha1.LanguageCluster {
	clone := c.DeepCopy()
	for _, mod := range mods {
		mod(clone)
	}
	return clone
}

// SetClusterDomain sets spec.domain.
func SetClusterDomain(domain string) LanguageClusterModifier {
	return func(c *langopv1alpha1.LanguageCluster) {
		c.Spec.Domain = domain
	}
}

// SetClusterIngressClassName sets spec.ingress.className.
func SetClusterIngressClassName(className string) LanguageClusterModifier {
	return func(c *langopv1alpha1.LanguageCluster) {
		if c.Spec.Ingress == nil {
			c.Spec.Ingress = &langopv1alpha1.IngressConfig{}
		}
		c.Spec.Ingress.ClassName = className
	}
}

// ReadyCluster constructs a LanguageCluster with Status.Phase already set to "Ready".
// Use this in unit tests that need a reconcilable namespace without setting up a full cluster reconciliation.
func ReadyCluster(name string, mods ...LanguageClusterModifier) *langopv1alpha1.LanguageCluster {
	c := LanguageCluster(name, mods...)
	c.Status.Phase = "Ready"
	return c
}

// SetClusterCapacity sets spec.capacity.
func SetClusterCapacity(cap *langopv1alpha1.ClusterCapacitySpec) LanguageClusterModifier {
	return func(c *langopv1alpha1.LanguageCluster) {
		c.Spec.Capacity = cap
	}
}

// SetClusterGatewayServiceType sets spec.gateway.deployment.serviceType.
func SetClusterGatewayServiceType(svcType corev1.ServiceType) LanguageClusterModifier {
	return func(c *langopv1alpha1.LanguageCluster) {
		if c.Spec.Gateway == nil {
			c.Spec.Gateway = &langopv1alpha1.GatewaySpec{}
		}
		c.Spec.Gateway.Deployment.ServiceType = svcType
	}
}

// SetClusterGatewayServiceAnnotations sets spec.gateway.deployment.serviceAnnotations.
func SetClusterGatewayServiceAnnotations(annotations map[string]string) LanguageClusterModifier {
	return func(c *langopv1alpha1.LanguageCluster) {
		if c.Spec.Gateway == nil {
			c.Spec.Gateway = &langopv1alpha1.GatewaySpec{}
		}
		c.Spec.Gateway.Deployment.ServiceAnnotations = annotations
	}
}
