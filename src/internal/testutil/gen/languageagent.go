// Package gen provides fluent builder functions for constructing test fixtures.
// Inspired by the cert-manager gen package pattern.
package gen

import (
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	langopv1alpha1 "github.com/language-operator/language-operator/api/v1alpha1"
)

// LanguageAgentModifier is a function that modifies a LanguageAgent.
type LanguageAgentModifier func(*langopv1alpha1.LanguageAgent)

// LanguageAgent constructs a LanguageAgent with the given name, namespace, and modifiers.
func LanguageAgent(name, namespace string, mods ...LanguageAgentModifier) *langopv1alpha1.LanguageAgent {
	a := &langopv1alpha1.LanguageAgent{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
		},
		Spec: langopv1alpha1.LanguageAgentSpec{
			Image: "test-agent:latest",
		},
	}
	for _, mod := range mods {
		mod(a)
	}
	return a
}

// LanguageAgentFrom clones an existing LanguageAgent and applies modifiers.
func LanguageAgentFrom(a *langopv1alpha1.LanguageAgent, mods ...LanguageAgentModifier) *langopv1alpha1.LanguageAgent {
	clone := a.DeepCopy()
	for _, mod := range mods {
		mod(clone)
	}
	return clone
}

// SetAgentImage sets spec.image.
func SetAgentImage(image string) LanguageAgentModifier {
	return func(a *langopv1alpha1.LanguageAgent) {
		a.Spec.Image = image
	}
}

// SetAgentExecutionMode sets spec.executionMode.
func SetAgentExecutionMode(mode string) LanguageAgentModifier {
	return func(a *langopv1alpha1.LanguageAgent) {
		a.Spec.ExecutionMode = mode
	}
}

// SetAgentModel appends a ModelReference with the given name.
func SetAgentModel(name string) LanguageAgentModifier {
	return func(a *langopv1alpha1.LanguageAgent) {
		a.Spec.Models = append(a.Spec.Models, langopv1alpha1.ModelReference{Name: name})
	}
}

// SetAgentPort sets spec.port.
func SetAgentPort(port int32) LanguageAgentModifier {
	return func(a *langopv1alpha1.LanguageAgent) {
		a.Spec.Port = &port
	}
}

// SetAgentWorkspace sets spec.workspace.
func SetAgentWorkspace(size string) LanguageAgentModifier {
	return func(a *langopv1alpha1.LanguageAgent) {
		a.Spec.Workspace = &langopv1alpha1.WorkspaceSpec{
			Enabled: true,
			Size:    size,
		}
	}
}

// SetAgentResources sets spec.resources.
func SetAgentResources(cpuRequest, memRequest, cpuLimit, memLimit string) LanguageAgentModifier {
	return func(a *langopv1alpha1.LanguageAgent) {
		a.Spec.Resources = corev1.ResourceRequirements{
			Requests: corev1.ResourceList{
				corev1.ResourceCPU:    resource.MustParse(cpuRequest),
				corev1.ResourceMemory: resource.MustParse(memRequest),
			},
			Limits: corev1.ResourceList{
				corev1.ResourceCPU:    resource.MustParse(cpuLimit),
				corev1.ResourceMemory: resource.MustParse(memLimit),
			},
		}
	}
}

// SetAgentEnv appends environment variables.
func SetAgentEnv(name, value string) LanguageAgentModifier {
	return func(a *langopv1alpha1.LanguageAgent) {
		a.Spec.Env = append(a.Spec.Env, corev1.EnvVar{Name: name, Value: value})
	}
}

// SetAgentInstructions sets spec.instructions.
func SetAgentInstructions(instructions string) LanguageAgentModifier {
	return func(a *langopv1alpha1.LanguageAgent) {
		a.Spec.Instructions = instructions
	}
}

// SetAgentLabel sets a label on the agent.
func SetAgentLabel(key, value string) LanguageAgentModifier {
	return func(a *langopv1alpha1.LanguageAgent) {
		if a.Labels == nil {
			a.Labels = make(map[string]string)
		}
		a.Labels[key] = value
	}
}
