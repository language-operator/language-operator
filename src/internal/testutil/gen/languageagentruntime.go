package gen

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	langopv1alpha1 "github.com/language-operator/language-operator/api/v1alpha1"
)

// LanguageAgentRuntimeModifier is a function that modifies a LanguageAgentRuntime.
type LanguageAgentRuntimeModifier func(*langopv1alpha1.LanguageAgentRuntime)

// LanguageAgentRuntime constructs a cluster-scoped LanguageAgentRuntime with the given name and modifiers.
func LanguageAgentRuntime(name string, mods ...LanguageAgentRuntimeModifier) *langopv1alpha1.LanguageAgentRuntime {
	rt := &langopv1alpha1.LanguageAgentRuntime{
		ObjectMeta: metav1.ObjectMeta{
			Name: name,
		},
	}
	for _, mod := range mods {
		mod(rt)
	}
	return rt
}

// LanguageAgentRuntimeFrom clones an existing LanguageAgentRuntime and applies modifiers.
func LanguageAgentRuntimeFrom(rt *langopv1alpha1.LanguageAgentRuntime, mods ...LanguageAgentRuntimeModifier) *langopv1alpha1.LanguageAgentRuntime {
	clone := rt.DeepCopy()
	for _, mod := range mods {
		mod(clone)
	}
	return clone
}

// SetRuntimeImage sets spec.image.
func SetRuntimeImage(image string) LanguageAgentRuntimeModifier {
	return func(rt *langopv1alpha1.LanguageAgentRuntime) {
		rt.Spec.Image = image
	}
}
