package gen

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	langopv1alpha1 "github.com/language-operator/language-operator/api/v1alpha1"
)

// LanguageToolModifier is a function that modifies a LanguageTool.
type LanguageToolModifier func(*langopv1alpha1.LanguageTool)

// LanguageTool constructs a LanguageTool with the given name, namespace, and modifiers.
func LanguageTool(name, namespace string, mods ...LanguageToolModifier) *langopv1alpha1.LanguageTool {
	t := &langopv1alpha1.LanguageTool{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
		},
		Spec: langopv1alpha1.LanguageToolSpec{
			Image: "test-tool:latest",
			Type:  "mcp",
		},
	}
	for _, mod := range mods {
		mod(t)
	}
	return t
}

// LanguageToolFrom clones an existing LanguageTool and applies modifiers.
func LanguageToolFrom(t *langopv1alpha1.LanguageTool, mods ...LanguageToolModifier) *langopv1alpha1.LanguageTool {
	clone := t.DeepCopy()
	for _, mod := range mods {
		mod(clone)
	}
	return clone
}

// SetToolImage sets spec.image.
func SetToolImage(image string) LanguageToolModifier {
	return func(t *langopv1alpha1.LanguageTool) {
		t.Spec.Image = image
	}
}

// SetToolClusterRef sets spec.clusterRef.
func SetToolClusterRef(ref string) LanguageToolModifier {
	return func(t *langopv1alpha1.LanguageTool) {
		t.Spec.ClusterRef = ref
	}
}

// SetToolPort sets spec.port.
func SetToolPort(port int32) LanguageToolModifier {
	return func(t *langopv1alpha1.LanguageTool) {
		t.Spec.Port = port
	}
}

// SetToolType sets spec.type.
func SetToolType(toolType string) LanguageToolModifier {
	return func(t *langopv1alpha1.LanguageTool) {
		t.Spec.Type = toolType
	}
}

// SetToolDeploymentMode sets spec.deploymentMode.
func SetToolDeploymentMode(mode string) LanguageToolModifier {
	return func(t *langopv1alpha1.LanguageTool) {
		t.Spec.DeploymentMode = mode
	}
}
