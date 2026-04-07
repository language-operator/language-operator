package controllers

import (
	"context"
	"fmt"
	"maps"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/intstr"

	langopv1alpha1 "github.com/language-operator/language-operator/api/v1alpha1"
)

func (r *LanguageAgentReconciler) reconcileDeployment(ctx context.Context, agent *langopv1alpha1.LanguageAgent, configHash string) error {
	// Resolve model URLs and names
	modelURLs, modelNames, err := r.resolveModels(ctx, agent)
	if err != nil {
		return fmt.Errorf("failed to resolve models: %w", err)
	}

	// Resolve tool URLs
	toolURLs, err := r.resolveTools(ctx, agent)
	if err != nil {
		return fmt.Errorf("failed to resolve tools: %w", err)
	}

	// Resolve sidecar tools
	sidecarContainers, err := r.resolveSidecarTools(ctx, agent)
	if err != nil {
		return fmt.Errorf("failed to resolve sidecar tools: %w", err)
	}

	// Determine target namespace and labels
	targetNamespace := agent.Namespace
	labels := GetCommonLabels(agent.Name, "LanguageAgent")
	labels[LabelKeyLangopComponent] = "agent" // Distinguish from trigger pods

	if err := ValidateClusterReference(ctx, r.Client, agent.Namespace); err != nil {
		return err
	}

	cluster := &langopv1alpha1.LanguageCluster{}
	if err := r.Get(ctx, types.NamespacedName{Name: agent.Namespace}, cluster); err != nil {
		return fmt.Errorf("failed to get cluster %s: %w", agent.Namespace, err)
	}

	labels[LabelKeyLangopCluster] = agent.Namespace

	deployment := &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      agent.Name,
			Namespace: targetNamespace,
			Labels:    labels,
		},
	}

	err = CreateOrUpdateOwned(ctx, r.Client, r.Scheme, agent, deployment, func() error {
		replicas := int32(1)
		if agent.Spec.Deployment.Replicas != nil {
			replicas = *agent.Spec.Deployment.Replicas
		}
		// When HPA is active, preserve the current replica count rather than overwriting it.
		// The HPA controller owns spec.replicas; if we reset it every reconcile the
		// Deployment will never scale.
		deploymentReplicas := &replicas
		if agent.Spec.Deployment.Autoscaling != nil && deployment.Spec.Replicas != nil {
			deploymentReplicas = deployment.Spec.Replicas
		}

		// Build container list starting with the agent
		containers := []corev1.Container{
			{
				Name:            "agent",
				Image:           agent.Spec.Image,
				ImagePullPolicy: agent.Spec.Deployment.ImagePullPolicy,
				Command:         agent.Spec.Deployment.Command,
				Args:            agent.Spec.Deployment.Args,
				Env:             r.buildAgentEnv(ctx, agent, cluster, modelURLs, modelNames, toolURLs),
				EnvFrom:         agent.Spec.Deployment.EnvFrom,
				Resources:       agent.Spec.Deployment.Resources,
				LivenessProbe:   agent.Spec.Deployment.LivenessProbe,
				ReadinessProbe:  agent.Spec.Deployment.ReadinessProbe,
				StartupProbe:    agent.Spec.Deployment.StartupProbe,
				Ports:           buildAgentContainerPorts(agentPorts(agent)),
			},
		}

		// Inject operator-managed env vars and volume mounts into user-specified init containers.
		// Per spec/agents.md, all contracted env vars must be present in every
		// container including init containers. The agent-config volume mount is also
		// injected so init containers (e.g. runtime adapters) can read /etc/agent/config.yaml.
		userInitContainers := make([]corev1.Container, len(agent.Spec.Deployment.InitContainers))
		copy(userInitContainers, agent.Spec.Deployment.InitContainers)
		agentEnv := r.buildAgentEnv(ctx, agent, cluster, modelURLs, modelNames, toolURLs)
		agentConfigMount := corev1.VolumeMount{
			Name:      "agent-config",
			MountPath: "/etc/agent",
			ReadOnly:  true,
		}
		for i := range userInitContainers {
			userInitContainers[i].Env = append(agentEnv, userInitContainers[i].Env...)
			userInitContainers[i].VolumeMounts = append([]corev1.VolumeMount{agentConfigMount}, userInitContainers[i].VolumeMounts...)
		}

		// Prepend the workspace-seeder init container so the workspace is populated
		// before any user init containers or sidecar tools run.
		allInitContainers := userInitContainers
		if seedContainer := buildWorkspaceSeedInitContainer(agent); seedContainer != nil {
			allInitContainers = append([]corev1.Container{*seedContainer}, allInitContainers...)
		}
		allInitContainers = append(allInitContainers, sidecarContainers...)

		// Merge user pod labels; operator-managed labels take precedence to protect selector stability.
		podLabels := make(map[string]string, len(labels)+len(agent.Spec.Deployment.PodLabels))
		for k, v := range agent.Spec.Deployment.PodLabels {
			podLabels[k] = v
		}
		for k, v := range labels {
			podLabels[k] = v
		}

		// Use user-supplied pod security context if set, otherwise apply operator defaults.
		podSecCtx := buildPodSecurityContext()
		if agent.Spec.Deployment.SecurityContext != nil {
			podSecCtx = agent.Spec.Deployment.SecurityContext
		}

		// Seed pod annotations with the operator-managed config-hash, then overlay user annotations.
		podAnnotations := map[string]string{LabelKeyLangopConfigHash: configHash}
		maps.Copy(podAnnotations, agent.Spec.Deployment.PodAnnotations)

		deployment.Spec = appsv1.DeploymentSpec{
			Replicas: deploymentReplicas,
			Selector: &metav1.LabelSelector{
				MatchLabels: labels,
			},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels:      podLabels,
					Annotations: podAnnotations,
				},
				Spec: corev1.PodSpec{
					ServiceAccountName:        r.getServiceAccountName(agent),
					ShareProcessNamespace:     &[]bool{len(allInitContainers) > 0}[0],
					InitContainers:            allInitContainers,
					Containers:                containers,
					SecurityContext:           podSecCtx,
					ImagePullSecrets:          agent.Spec.Deployment.ImagePullSecrets,
					NodeSelector:              agent.Spec.Deployment.NodeSelector,
					Tolerations:               agent.Spec.Deployment.Tolerations,
					TopologySpreadConstraints: agent.Spec.Deployment.TopologySpreadConstraints,
					Affinity:                  agent.Spec.Deployment.Affinity,
				},
			},
		}

		// Add container security context for agent container
		deployment.Spec.Template.Spec.Containers[0].SecurityContext = buildContainerSecurityContext()

		// Build operator-managed volumes and volume mounts, then append user-supplied ones.
		volumes, volumeMounts := r.buildVolumes(ctx, agent)
		// Append seed ConfigMap volumes (not mounted in main container; used by workspace-seeder init container).
		volumes = append(volumes, buildWorkspaceSeedVolumes(agent)...)
		volumes = append(volumes, agent.Spec.Deployment.Volumes...)
		volumeMounts = append(volumeMounts, agent.Spec.Deployment.VolumeMounts...)
		if len(volumes) > 0 {
			deployment.Spec.Template.Spec.Volumes = volumes
		}
		if len(volumeMounts) > 0 {
			deployment.Spec.Template.Spec.Containers[0].VolumeMounts = volumeMounts
		}

		return nil
	})

	return err
}

func (r *LanguageAgentReconciler) resolveModels(ctx context.Context, agent *langopv1alpha1.LanguageAgent) ([]string, []string, error) {
	var modelURLs []string
	var modelNames []string

	for _, modelRef := range agent.Spec.Models {
		// Fetch the LanguageModel (always in the agent's namespace)
		model := &langopv1alpha1.LanguageModel{}
		if err := r.Get(ctx, types.NamespacedName{Name: modelRef.Name, Namespace: agent.Namespace}, model); err != nil {
			return nil, nil, fmt.Errorf("failed to get model %s/%s: %w", agent.Namespace, modelRef.Name, err)
		}

		// All models in a cluster are served by the shared gateway
		// in the cluster namespace. Deduplicate: only add the gateway URL once.
		gatewayURL := serviceURL("gateway", agent.Namespace, GatewayServicePort)
		alreadyAdded := false
		for _, u := range modelURLs {
			if u == gatewayURL {
				alreadyAdded = true
				break
			}
		}
		if !alreadyAdded {
			modelURLs = append(modelURLs, gatewayURL)
		}

		// Collect model name from spec
		if model.Spec.ModelName != "" {
			modelNames = append(modelNames, model.Spec.ModelName)
		}
	}

	return modelURLs, modelNames, nil
}

func (r *LanguageAgentReconciler) resolveSidecarTools(ctx context.Context, agent *langopv1alpha1.LanguageAgent) ([]corev1.Container, error) {
	var sidecarContainers []corev1.Container

	for _, toolRef := range agent.Spec.Tools {
		// Skip tools explicitly disabled by the user
		if toolRef.Enabled != nil && !*toolRef.Enabled {
			continue
		}

		// Fetch the LanguageTool (always in the agent's namespace)
		tool := &langopv1alpha1.LanguageTool{}
		if err := r.Get(ctx, types.NamespacedName{Name: toolRef.Name, Namespace: agent.Namespace}, tool); err != nil {
			return nil, fmt.Errorf("failed to get tool %s/%s: %w", agent.Namespace, toolRef.Name, err)
		}

		// Only process sidecar tools
		if tool.Spec.DeploymentMode != "sidecar" {
			continue
		}

		// Build sidecar container spec
		port := tool.Spec.Port
		if port == 0 {
			port = 8080 // Default MCP port
		}

		// Use native sidecar support (Kubernetes 1.28+)
		// Sidecars with restartPolicy: Always will terminate automatically
		// when the main container completes
		restartPolicy := corev1.ContainerRestartPolicyAlways
		container := corev1.Container{
			Name:          fmt.Sprintf("tool-%s", tool.Name),
			Image:         tool.Spec.Image,
			RestartPolicy: &restartPolicy,
			Ports: []corev1.ContainerPort{
				{
					Name:          "mcp",
					ContainerPort: port,
					Protocol:      corev1.ProtocolTCP,
				},
			},
			Env: tool.Spec.Deployment.Env,
			ReadinessProbe: &corev1.Probe{
				ProbeHandler: corev1.ProbeHandler{
					TCPSocket: &corev1.TCPSocketAction{
						Port: intstr.FromInt(int(port)),
					},
				},
				InitialDelaySeconds: 2,
				PeriodSeconds:       2,
				TimeoutSeconds:      1,
				SuccessThreshold:    1,
				FailureThreshold:    3,
			},
		}

		// Add resource requirements if specified, otherwise use sensible defaults for tool containers
		if tool.Spec.Deployment.Resources.Requests == nil && tool.Spec.Deployment.Resources.Limits == nil {
			container.Resources = corev1.ResourceRequirements{
				Requests: corev1.ResourceList{
					corev1.ResourceCPU:    resource.MustParse("50m"),
					corev1.ResourceMemory: resource.MustParse("128Mi"),
				},
				Limits: corev1.ResourceList{
					corev1.ResourceCPU:    resource.MustParse("500m"),
					corev1.ResourceMemory: resource.MustParse("1Gi"),
				},
			}
		} else {
			container.Resources = tool.Spec.Deployment.Resources
		}

		// Mount workspace if agent has workspace enabled
		if agent.Spec.Workspace != nil && (agent.Spec.Workspace.Enabled == nil || *agent.Spec.Workspace.Enabled) {
			mountPath := agent.Spec.Workspace.MountPath
			if mountPath == "" {
				mountPath = "/workspace"
			}

			container.VolumeMounts = []corev1.VolumeMount{
				{
					Name:      "workspace",
					MountPath: mountPath,
				},
			}
		}

		sidecarContainers = append(sidecarContainers, container)
	}

	return sidecarContainers, nil
}

func (r *LanguageAgentReconciler) resolveTools(ctx context.Context, agent *langopv1alpha1.LanguageAgent) ([]string, error) {
	var toolURLs []string

	for _, toolRef := range agent.Spec.Tools {
		// Skip tools explicitly disabled by the user
		if toolRef.Enabled != nil && !*toolRef.Enabled {
			continue
		}

		// Fetch the LanguageTool (always in the agent's namespace)
		tool := &langopv1alpha1.LanguageTool{}
		if err := r.Get(ctx, types.NamespacedName{Name: toolRef.Name, Namespace: agent.Namespace}, tool); err != nil {
			return nil, fmt.Errorf("failed to get tool %s/%s: %w", agent.Namespace, toolRef.Name, err)
		}

		port := tool.Spec.Port
		if port == 0 {
			port = 8080 // Default MCP port
		}

		// Sidecar tools use localhost URLs
		if tool.Spec.DeploymentMode == "sidecar" {
			// Format: http://localhost:<port>
			localhostURL := fmt.Sprintf("http://localhost:%d", port)
			toolURLs = append(toolURLs, localhostURL)
			continue
		}

		// Build MCP server URL (service mode)
		mcpURL := serviceURL(tool.Name, agent.Namespace, port)
		toolURLs = append(toolURLs, mcpURL)
	}

	return toolURLs, nil
}

// buildVolumes creates the volumes and volume mounts for agent pods
func (r *LanguageAgentReconciler) buildVolumes(ctx context.Context, agent *langopv1alpha1.LanguageAgent) ([]corev1.Volume, []corev1.VolumeMount) {
	volumes := []corev1.Volume{}
	volumeMounts := []corev1.VolumeMount{}

	// Add tmpfs volumes for read-only root filesystem
	// /tmp - general temporary files
	volumes = append(volumes, corev1.Volume{
		Name: "tmp",
		VolumeSource: corev1.VolumeSource{
			EmptyDir: &corev1.EmptyDirVolumeSource{
				Medium: corev1.StorageMediumMemory, // Use tmpfs
			},
		},
	})
	volumeMounts = append(volumeMounts, corev1.VolumeMount{
		Name:      "tmp",
		MountPath: "/tmp",
	})

	// Mount agent config ConfigMap at /etc/agent/ (provides config.yaml)
	agentConfigMapName := GenerateConfigMapName(agent.Name, "agent")
	volumes = append(volumes, corev1.Volume{
		Name: "agent-config",
		VolumeSource: corev1.VolumeSource{
			ConfigMap: &corev1.ConfigMapVolumeSource{
				LocalObjectReference: corev1.LocalObjectReference{
					Name: agentConfigMapName,
				},
			},
		},
	})
	volumeMounts = append(volumeMounts, corev1.VolumeMount{
		Name:      "agent-config",
		MountPath: "/etc/agent",
		ReadOnly:  true,
	})

	// Add workspace volume if enabled
	if agent.Spec.Workspace != nil && (agent.Spec.Workspace.Enabled == nil || *agent.Spec.Workspace.Enabled) {
		mountPath := agent.Spec.Workspace.MountPath
		if mountPath == "" {
			mountPath = "/workspace"
		}

		volumes = append(volumes, corev1.Volume{
			Name: "workspace",
			VolumeSource: corev1.VolumeSource{
				PersistentVolumeClaim: &corev1.PersistentVolumeClaimVolumeSource{
					ClaimName: GeneratePVCName(agent.Name),
				},
			},
		})

		volumeMounts = append(volumeMounts, corev1.VolumeMount{
			Name:      "workspace",
			MountPath: mountPath,
		})
	}

	return volumes, volumeMounts
}
