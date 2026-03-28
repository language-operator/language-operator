// Package gen provides fluent builder functions for constructing test fixtures.
package gen

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	langopv1alpha1 "github.com/language-operator/language-operator/api/v1alpha1"
)

// LanguagePersonaModifier is a function that modifies a LanguagePersona.
type LanguagePersonaModifier func(*langopv1alpha1.LanguagePersona)

// LanguagePersona constructs a LanguagePersona with the given name, namespace, and modifiers.
func LanguagePersona(name, namespace string, mods ...LanguagePersonaModifier) *langopv1alpha1.LanguagePersona {
	p := &langopv1alpha1.LanguagePersona{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
		},
		Spec: langopv1alpha1.LanguagePersonaSpec{
			SystemPrompt: "You are a test assistant.",
		},
	}
	for _, mod := range mods {
		mod(p)
	}
	return p
}

// LanguagePersonaFrom clones an existing LanguagePersona and applies modifiers.
func LanguagePersonaFrom(p *langopv1alpha1.LanguagePersona, mods ...LanguagePersonaModifier) *langopv1alpha1.LanguagePersona {
	clone := p.DeepCopy()
	for _, mod := range mods {
		mod(clone)
	}
	return clone
}

// SetPersonaSystemPrompt sets spec.systemPrompt.
func SetPersonaSystemPrompt(prompt string) LanguagePersonaModifier {
	return func(p *langopv1alpha1.LanguagePersona) {
		p.Spec.SystemPrompt = prompt
	}
}
