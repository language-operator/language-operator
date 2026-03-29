package gen

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	langopv1alpha1 "github.com/language-operator/language-operator/api/v1alpha1"
)

// LanguageModelModifier is a function that modifies a LanguageModel.
type LanguageModelModifier func(*langopv1alpha1.LanguageModel)

// LanguageModel constructs a LanguageModel with the given name, namespace, and modifiers.
func LanguageModel(name, namespace string, mods ...LanguageModelModifier) *langopv1alpha1.LanguageModel {
	m := &langopv1alpha1.LanguageModel{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
		},
		Spec: langopv1alpha1.LanguageModelSpec{
			Provider:  "anthropic",
			ModelName: "claude-sonnet-4-5",
		},
	}
	for _, mod := range mods {
		mod(m)
	}
	return m
}

// LanguageModelFrom clones an existing LanguageModel and applies modifiers.
func LanguageModelFrom(m *langopv1alpha1.LanguageModel, mods ...LanguageModelModifier) *langopv1alpha1.LanguageModel {
	clone := m.DeepCopy()
	for _, mod := range mods {
		mod(clone)
	}
	return clone
}

// SetModelProvider sets spec.provider.
func SetModelProvider(provider string) LanguageModelModifier {
	return func(m *langopv1alpha1.LanguageModel) {
		m.Spec.Provider = provider
	}
}

// SetModelName sets spec.modelName.
func SetModelName(name string) LanguageModelModifier {
	return func(m *langopv1alpha1.LanguageModel) {
		m.Spec.ModelName = name
	}
}

// SetModelAPIKeySecretRef sets spec.apiKeySecretRef.
func SetModelAPIKeySecretRef(secretName, key string) LanguageModelModifier {
	return func(m *langopv1alpha1.LanguageModel) {
		m.Spec.APIKeySecretRef = &langopv1alpha1.SecretReference{
			Name: secretName,
			Key:  key,
		}
	}
}

// SetModelEndpoint sets spec.endpoint.
func SetModelEndpoint(endpoint string) LanguageModelModifier {
	return func(m *langopv1alpha1.LanguageModel) {
		m.Spec.Endpoint = endpoint
	}
}
