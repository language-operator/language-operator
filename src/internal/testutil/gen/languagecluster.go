package gen

import (
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

// SetClusterIngressClassName sets spec.ingressConfig.ingressClassName.
func SetClusterIngressClassName(className string) LanguageClusterModifier {
	return func(c *langopv1alpha1.LanguageCluster) {
		if c.Spec.IngressConfig == nil {
			c.Spec.IngressConfig = &langopv1alpha1.IngressConfig{}
		}
		c.Spec.IngressConfig.IngressClassName = className
	}
}

// SetClusterGatewayName sets spec.ingressConfig.gatewayName.
func SetClusterGatewayName(name string) LanguageClusterModifier {
	return func(c *langopv1alpha1.LanguageCluster) {
		if c.Spec.IngressConfig == nil {
			c.Spec.IngressConfig = &langopv1alpha1.IngressConfig{}
		}
		c.Spec.IngressConfig.GatewayName = name
	}
}
