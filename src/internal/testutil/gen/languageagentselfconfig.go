// Package gen provides fluent builder functions for constructing test fixtures.
package gen

import (
	rbacv1 "k8s.io/api/rbac/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	langopv1alpha1 "github.com/language-operator/language-operator/api/v1alpha1"
)

// LanguageAgentSelfConfigModifier is a function that modifies a LanguageAgentSelfConfig.
type LanguageAgentSelfConfigModifier func(*langopv1alpha1.LanguageAgentSelfConfig)

// LanguageAgentSelfConfig constructs a LanguageAgentSelfConfig targeting instanceRef.
func LanguageAgentSelfConfig(name, namespace, instanceRef string, mods ...LanguageAgentSelfConfigModifier) *langopv1alpha1.LanguageAgentSelfConfig {
	sc := &langopv1alpha1.LanguageAgentSelfConfig{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
		},
		Spec: langopv1alpha1.LanguageAgentSelfConfigSpec{
			InstanceRef: instanceRef,
		},
	}
	for _, mod := range mods {
		mod(sc)
	}
	return sc
}

// LanguageAgentSelfConfigFrom clones an existing LanguageAgentSelfConfig and applies modifiers.
func LanguageAgentSelfConfigFrom(sc *langopv1alpha1.LanguageAgentSelfConfig, mods ...LanguageAgentSelfConfigModifier) *langopv1alpha1.LanguageAgentSelfConfig {
	clone := sc.DeepCopy()
	for _, mod := range mods {
		mod(clone)
	}
	return clone
}

// SetSelfConfigAddTools sets spec.addTools.
func SetSelfConfigAddTools(tools ...string) LanguageAgentSelfConfigModifier {
	return func(sc *langopv1alpha1.LanguageAgentSelfConfig) {
		sc.Spec.AddTools = tools
	}
}

// SetSelfConfigRemoveTools sets spec.removeTools.
func SetSelfConfigRemoveTools(tools ...string) LanguageAgentSelfConfigModifier {
	return func(sc *langopv1alpha1.LanguageAgentSelfConfig) {
		sc.Spec.RemoveTools = tools
	}
}

// SetSelfConfigAddModels sets spec.addModels.
func SetSelfConfigAddModels(models ...string) LanguageAgentSelfConfigModifier {
	return func(sc *langopv1alpha1.LanguageAgentSelfConfig) {
		sc.Spec.AddModels = models
	}
}

// SetSelfConfigRemoveModels sets spec.removeModels.
func SetSelfConfigRemoveModels(models ...string) LanguageAgentSelfConfigModifier {
	return func(sc *langopv1alpha1.LanguageAgentSelfConfig) {
		sc.Spec.RemoveModels = models
	}
}

// SetSelfConfigAddEnvVars sets spec.addEnvVars.
func SetSelfConfigAddEnvVars(vars ...langopv1alpha1.SelfConfigEnvVar) LanguageAgentSelfConfigModifier {
	return func(sc *langopv1alpha1.LanguageAgentSelfConfig) {
		sc.Spec.AddEnvVars = vars
	}
}

// SetSelfConfigUpdateInstructions sets spec.updateInstructions.
func SetSelfConfigUpdateInstructions(instructions string) LanguageAgentSelfConfigModifier {
	return func(sc *langopv1alpha1.LanguageAgentSelfConfig) {
		sc.Spec.UpdateInstructions = instructions
	}
}

// SetSelfConfigAddRoleRules sets spec.addRoleRules.
func SetSelfConfigAddRoleRules(rules ...rbacv1.PolicyRule) LanguageAgentSelfConfigModifier {
	return func(sc *langopv1alpha1.LanguageAgentSelfConfig) {
		sc.Spec.AddRoleRules = rules
	}
}

// SetSelfConfigPhase sets status.phase and completionTime (for testing TTL logic).
func SetSelfConfigPhase(phase langopv1alpha1.SelfConfigPhase) LanguageAgentSelfConfigModifier {
	return func(sc *langopv1alpha1.LanguageAgentSelfConfig) {
		now := metav1.Now()
		sc.Status.Phase = phase
		sc.Status.CompletionTime = &now
	}
}
