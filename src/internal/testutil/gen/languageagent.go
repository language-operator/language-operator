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

// SetAgentReplicas sets spec.deployment.replicas.
func SetAgentReplicas(replicas int32) LanguageAgentModifier {
	return func(a *langopv1alpha1.LanguageAgent) {
		a.Spec.Deployment.Replicas = &replicas
	}
}

// SetAgentAutoscaling sets spec.deployment.autoscaling with the given min/max replica bounds.
func SetAgentAutoscaling(minReplicas, maxReplicas int32) LanguageAgentModifier {
	return func(a *langopv1alpha1.LanguageAgent) {
		a.Spec.Deployment.Autoscaling = &langopv1alpha1.AutoscalingSpec{
			MinReplicas: &minReplicas,
			MaxReplicas: maxReplicas,
		}
	}
}

// SetAgentImage sets spec.image.
func SetAgentImage(image string) LanguageAgentModifier {
	return func(a *langopv1alpha1.LanguageAgent) {
		a.Spec.Image = image
	}
}

// SetAgentModel appends a ModelReference with the given name.
func SetAgentModel(name string) LanguageAgentModifier {
	return func(a *langopv1alpha1.LanguageAgent) {
		a.Spec.Models = append(a.Spec.Models, langopv1alpha1.ModelReference{Name: name})
	}
}

// SetAgentPorts sets spec.ports.
func SetAgentPorts(ports []langopv1alpha1.AgentPort) LanguageAgentModifier {
	return func(a *langopv1alpha1.LanguageAgent) {
		a.Spec.Ports = ports
	}
}

// SetAgentWorkspace sets spec.workspace.
func SetAgentWorkspace(size string) LanguageAgentModifier {
	return func(a *langopv1alpha1.LanguageAgent) {
		enabled := true
		a.Spec.Workspace = &langopv1alpha1.WorkspaceSpec{
			Enabled: &enabled,
			Size:    size,
		}
	}
}

// SetAgentResources sets spec.deployment.resources.
func SetAgentResources(cpuRequest, memRequest, cpuLimit, memLimit string) LanguageAgentModifier {
	return func(a *langopv1alpha1.LanguageAgent) {
		a.Spec.Deployment.Resources = corev1.ResourceRequirements{
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
		a.Spec.Deployment.Env = append(a.Spec.Deployment.Env, corev1.EnvVar{Name: name, Value: value})
	}
}

// SetAgentInstructions sets spec.instructions.
func SetAgentInstructions(instructions string) LanguageAgentModifier {
	return func(a *langopv1alpha1.LanguageAgent) {
		a.Spec.Instructions = instructions
	}
}

// SetAgentServiceType sets spec.deployment.serviceType.
func SetAgentServiceType(svcType corev1.ServiceType) LanguageAgentModifier {
	return func(a *langopv1alpha1.LanguageAgent) {
		a.Spec.Deployment.ServiceType = svcType
	}
}

// SetAgentServiceAnnotations sets spec.deployment.serviceAnnotations.
func SetAgentServiceAnnotations(annotations map[string]string) LanguageAgentModifier {
	return func(a *langopv1alpha1.LanguageAgent) {
		a.Spec.Deployment.ServiceAnnotations = annotations
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

// SetAgentPersona sets spec.persona.
func SetAgentPersona(name string) LanguageAgentModifier {
	return func(a *langopv1alpha1.LanguageAgent) {
		a.Spec.Persona = name
	}
}

// SetAgentTool appends a ToolReference to spec.tools.
func SetAgentTool(name string, enabled *bool) LanguageAgentModifier {
	return func(a *langopv1alpha1.LanguageAgent) {
		a.Spec.Tools = append(a.Spec.Tools, langopv1alpha1.ToolReference{Name: name, Enabled: enabled})
	}
}

// SetAgentNetworkPolicies sets spec.networkPolicies.
func SetAgentNetworkPolicies(policies *langopv1alpha1.AgentNetworkPolicies) LanguageAgentModifier {
	return func(a *langopv1alpha1.LanguageAgent) {
		a.Spec.NetworkPolicies = policies
	}
}

// SetAgentWorkspaceStorageClass sets spec.workspace.storageClassName, initialising the workspace if needed.
func SetAgentWorkspaceStorageClass(className string) LanguageAgentModifier {
	return func(a *langopv1alpha1.LanguageAgent) {
		if a.Spec.Workspace == nil {
			enabled := true
			a.Spec.Workspace = &langopv1alpha1.WorkspaceSpec{Enabled: &enabled, Size: "10Gi"}
		}
		a.Spec.Workspace.StorageClassName = &className
	}
}

// SetAgentWorkspaceMountPath sets spec.workspace.mountPath, initialising the workspace if needed.
func SetAgentWorkspaceMountPath(path string) LanguageAgentModifier {
	return func(a *langopv1alpha1.LanguageAgent) {
		if a.Spec.Workspace == nil {
			enabled := true
			a.Spec.Workspace = &langopv1alpha1.WorkspaceSpec{Enabled: &enabled, Size: "10Gi"}
		}
		a.Spec.Workspace.MountPath = path
	}
}

// SetAgentWorkspaceAccessMode sets spec.workspace.accessMode, initialising the workspace if needed.
func SetAgentWorkspaceAccessMode(mode corev1.PersistentVolumeAccessMode) LanguageAgentModifier {
	return func(a *langopv1alpha1.LanguageAgent) {
		if a.Spec.Workspace == nil {
			enabled := true
			a.Spec.Workspace = &langopv1alpha1.WorkspaceSpec{Enabled: &enabled, Size: "10Gi"}
		}
		a.Spec.Workspace.AccessMode = string(mode)
	}
}

// SetAgentWorkspaceRetain sets spec.workspace.retain, initialising the workspace if needed.
func SetAgentWorkspaceRetain(retain bool) LanguageAgentModifier {
	return func(a *langopv1alpha1.LanguageAgent) {
		if a.Spec.Workspace == nil {
			enabled := true
			a.Spec.Workspace = &langopv1alpha1.WorkspaceSpec{Enabled: &enabled, Size: "10Gi"}
		}
		a.Spec.Workspace.Retain = &retain
	}
}

// SetAgentWorkspaceInitialFiles sets spec.workspace.initialFiles, initialising the workspace if needed.
func SetAgentWorkspaceInitialFiles(files map[string]string) LanguageAgentModifier {
	return func(a *langopv1alpha1.LanguageAgent) {
		if a.Spec.Workspace == nil {
			enabled := true
			a.Spec.Workspace = &langopv1alpha1.WorkspaceSpec{Enabled: &enabled, Size: "10Gi"}
		}
		a.Spec.Workspace.InitialFiles = files
	}
}

// SetAgentWorkspaceSeedConfigMapRef sets spec.workspace.seedConfigMapRef, initialising the workspace if needed.
func SetAgentWorkspaceSeedConfigMapRef(name string) LanguageAgentModifier {
	return func(a *langopv1alpha1.LanguageAgent) {
		if a.Spec.Workspace == nil {
			enabled := true
			a.Spec.Workspace = &langopv1alpha1.WorkspaceSpec{Enabled: &enabled, Size: "10Gi"}
		}
		a.Spec.Workspace.SeedConfigMapRef = &corev1.LocalObjectReference{Name: name}
	}
}

// SetAgentServiceAccountName sets spec.deployment.serviceAccountName.
func SetAgentServiceAccountName(name string) LanguageAgentModifier {
	return func(a *langopv1alpha1.LanguageAgent) {
		a.Spec.Deployment.ServiceAccountName = name
	}
}

// SetAgentRepository sets spec.repository. Empty ref/path/secretRef are left unset.
func SetAgentRepository(url, ref, path, secretRef string) LanguageAgentModifier {
	return func(a *langopv1alpha1.LanguageAgent) {
		repo := &langopv1alpha1.RepositorySpec{URL: url, Ref: ref, Path: path}
		if secretRef != "" {
			repo.SecretRef = &corev1.LocalObjectReference{Name: secretRef}
		}
		a.Spec.Repository = repo
	}
}

// SetAgentRepositoryDepth sets spec.repository.depth, initialising the repository if needed.
func SetAgentRepositoryDepth(depth int) LanguageAgentModifier {
	return func(a *langopv1alpha1.LanguageAgent) {
		if a.Spec.Repository == nil {
			a.Spec.Repository = &langopv1alpha1.RepositorySpec{}
		}
		a.Spec.Repository.Depth = depth
	}
}
