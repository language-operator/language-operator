package controllers

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"os"
	"strings"

	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/yaml"

	langopv1alpha1 "github.com/language-operator/language-operator/api/v1alpha1"
	"github.com/language-operator/language-operator/pkg/events"
	"github.com/language-operator/language-operator/pkg/network"
)

// agentConfigYAML is the structure marshaled into /etc/agent/config.yaml.
// sigs.k8s.io/yaml marshals via JSON, so json tags control the output key names.
type agentConfigYAML struct {
	Agent        agentIdentityYAML          `json:"agent"`
	Instructions string                     `json:"instructions,omitempty"`
	Personas     []personaConfigYAML        `json:"personas,omitempty"`
	Tools        map[string]toolConfigYAML  `json:"tools,omitempty"`
	Models       map[string]modelConfigYAML `json:"models,omitempty"`
}

type agentIdentityYAML struct {
	Name      string `json:"name"`
	Namespace string `json:"namespace"`
}

type personaConfigYAML struct {
	Name        string `json:"name"`
	Tone        string `json:"tone,omitempty"`
	Personality string `json:"personality,omitempty"`
	Expertise   string `json:"expertise,omitempty"`
}

type toolConfigYAML struct {
	Endpoint string `json:"endpoint"`
	Protocol string `json:"protocol"`
}

type modelConfigYAML struct {
	Role     string `json:"role,omitempty"`
	Provider string `json:"provider"`
	Model    string `json:"model"`
	Endpoint string `json:"endpoint"`
	Priority *int32 `json:"priority,omitempty"`
}

func (r *LanguageAgentReconciler) reconcileConfigMap(ctx context.Context, agent *langopv1alpha1.LanguageAgent) (string, error) {
	l := log.FromContext(ctx)

	cfg := agentConfigYAML{
		Agent: agentIdentityYAML{
			Name:      agent.Name,
			Namespace: agent.Namespace,
		},
		Instructions: agent.Spec.Instructions,
	}

	// Persona
	persona, err := r.fetchPersona(ctx, agent)
	if err != nil {
		l.Error(err, "Failed to fetch persona, continuing without it")
	}
	if persona != nil {
		cfg.Personas = []personaConfigYAML{{
			Name:        persona.Name,
			Tone:        persona.Spec.Tone,
			Personality: persona.Spec.Personality,
			Expertise:   persona.Spec.Expertise,
		}}
	}

	// Tools
	for _, toolRef := range agent.Spec.Tools {
		if toolRef.Enabled != nil && !*toolRef.Enabled {
			continue
		}
		tool := &langopv1alpha1.LanguageTool{}
		if err := r.Get(ctx, types.NamespacedName{Name: toolRef.Name, Namespace: agent.Namespace}, tool); err != nil {
			l.Error(err, "Failed to get tool for config.yaml, skipping", "tool", toolRef.Name)
			continue
		}
		port := tool.Spec.Port
		if port == 0 {
			port = 8080
		}
		endpoint := serviceURL(tool.Name, agent.Namespace, port)
		if tool.Spec.DeploymentMode == "sidecar" {
			endpoint = fmt.Sprintf("http://localhost:%d", port)
		}
		if cfg.Tools == nil {
			cfg.Tools = make(map[string]toolConfigYAML)
		}
		cfg.Tools[tool.Name] = toolConfigYAML{Endpoint: endpoint, Protocol: "mcp"}
	}

	// Models — all served via the shared namespace gateway
	gatewayURL := serviceURL("gateway", agent.Namespace, network.GatewayServicePort)
	for _, modelRef := range agent.Spec.Models {
		model := &langopv1alpha1.LanguageModel{}
		if err := r.Get(ctx, types.NamespacedName{Name: modelRef.Name, Namespace: agent.Namespace}, model); err != nil {
			l.Error(err, "Failed to get model for config.yaml, skipping", "model", modelRef.Name)
			continue
		}
		if cfg.Models == nil {
			cfg.Models = make(map[string]modelConfigYAML)
		}
		cfg.Models[modelRef.Name] = modelConfigYAML{
			Role:     modelRef.Role,
			Provider: model.Spec.Provider,
			Model:    model.Spec.ModelName,
			Endpoint: gatewayURL,
			Priority: modelRef.Priority,
		}
	}

	configYAMLBytes, err := yaml.Marshal(cfg)
	if err != nil {
		return "", fmt.Errorf("failed to marshal config.yaml: %w", err)
	}

	configHash := hashString(string(configYAMLBytes))[:16]

	data := map[string]string{
		"config.yaml": string(configYAMLBytes),
	}

	configMapName := GenerateConfigMapName(agent.Name, "agent")
	return configHash, CreateOrUpdateConfigMap(ctx, r.Client, r.Scheme, agent, configMapName, agent.Namespace, data)
}

// getToolNames extracts tool names from agent's tools
func (r *LanguageAgentReconciler) getToolNames(agent *langopv1alpha1.LanguageAgent) []string {
	var names []string
	for _, ref := range agent.Spec.Tools {
		names = append(names, ref.Name)
	}
	return names
}

// getModelNames extracts model names from agent's models
func (r *LanguageAgentReconciler) getModelNames(agent *langopv1alpha1.LanguageAgent) []string {
	var names []string
	for _, ref := range agent.Spec.Models {
		names = append(names, ref.Name)
	}
	return names
}

// getPersonaNames returns the persona name for the agent, if set
func (r *LanguageAgentReconciler) getPersonaNames(agent *langopv1alpha1.LanguageAgent) []string {
	if agent.Spec.Persona == "" {
		return nil
	}
	return []string{agent.Spec.Persona}
}

// hashString creates a SHA256 hash of a string for change detection
func hashString(s string) string {
	h := sha256.New()
	h.Write([]byte(s))
	return fmt.Sprintf("%x", h.Sum(nil))
}

// reconcileWorkspaceSeedConfigMap creates or deletes the workspace-seed ConfigMap
// that holds the contents of spec.workspace.initialFiles.
// When InitialFiles is empty (or workspace is nil/disabled), the ConfigMap is deleted
// so stale seed data is not left behind.
func (r *LanguageAgentReconciler) reconcileWorkspaceSeedConfigMap(ctx context.Context, agent *langopv1alpha1.LanguageAgent) error {
	cmName := GenerateConfigMapName(agent.Name, "workspace-seed")

	wsEnabled := agent.Spec.Workspace != nil &&
		(agent.Spec.Workspace.Enabled == nil || *agent.Spec.Workspace.Enabled)

	if !wsEnabled || len(agent.Spec.Workspace.InitialFiles) == 0 {
		// No InitialFiles — remove any previously-created seed ConfigMap.
		return DeleteConfigMap(ctx, r.Client, cmName, agent.Namespace)
	}

	return CreateOrUpdateConfigMap(ctx, r.Client, r.Scheme, agent, cmName, agent.Namespace, agent.Spec.Workspace.InitialFiles)
}

// workspaceSeedEnabled reports whether workspace seeding is configured on the agent.
func workspaceSeedEnabled(agent *langopv1alpha1.LanguageAgent) bool {
	if agent.Spec.Workspace == nil {
		return false
	}
	if agent.Spec.Workspace.Enabled != nil && !*agent.Spec.Workspace.Enabled {
		return false
	}
	return len(agent.Spec.Workspace.InitialFiles) > 0 || agent.Spec.Workspace.SeedConfigMapRef != nil
}

// buildWorkspaceSeedVolumes returns pod-level volumes required by the workspace-seeder
// init container. These are not mounted in the main agent container.
func buildWorkspaceSeedVolumes(agent *langopv1alpha1.LanguageAgent) []corev1.Volume {
	if !workspaceSeedEnabled(agent) {
		return nil
	}
	var vols []corev1.Volume
	if len(agent.Spec.Workspace.InitialFiles) > 0 {
		vols = append(vols, corev1.Volume{
			Name: "workspace-seed-init",
			VolumeSource: corev1.VolumeSource{
				ConfigMap: &corev1.ConfigMapVolumeSource{
					LocalObjectReference: corev1.LocalObjectReference{
						Name: GenerateConfigMapName(agent.Name, "workspace-seed"),
					},
				},
			},
		})
	}
	if agent.Spec.Workspace.SeedConfigMapRef != nil {
		vols = append(vols, corev1.Volume{
			Name: "workspace-seed-ref",
			VolumeSource: corev1.VolumeSource{
				ConfigMap: &corev1.ConfigMapVolumeSource{
					LocalObjectReference: *agent.Spec.Workspace.SeedConfigMapRef,
				},
			},
		})
	}
	return vols
}

// buildWorkspaceSeedInitContainer returns the workspace-seeder init container when seeding
// is configured, or nil otherwise. The container uses seed-once semantics: files are only
// copied if they do not already exist at the destination path, preserving any agent edits.
// InitialFiles are processed first (higher priority), SeedConfigMapRef second.
func buildWorkspaceSeedInitContainer(agent *langopv1alpha1.LanguageAgent) *corev1.Container {
	if !workspaceSeedEnabled(agent) {
		return nil
	}

	mountPath := agent.Spec.Workspace.MountPath
	if mountPath == "" {
		mountPath = "/workspace"
	}

	// Build the shell script. Both loops use seed-once semantics (test -f).
	script := fmt.Sprintf(`set -e
WORKSPACE=%s
if [ -d /seed-init ]; then
  for f in /seed-init/*; do
    [ -f "$f" ] || continue
    dest="$WORKSPACE/$(basename "$f")"
    [ -f "$dest" ] || cp "$f" "$dest"
  done
fi
if [ -d /seed-ref ]; then
  for f in /seed-ref/*; do
    [ -f "$f" ] || continue
    dest="$WORKSPACE/$(basename "$f")"
    [ -f "$dest" ] || cp "$f" "$dest"
  done
fi`, mountPath)

	mounts := []corev1.VolumeMount{
		{
			Name:      "workspace",
			MountPath: mountPath,
		},
	}
	if len(agent.Spec.Workspace.InitialFiles) > 0 {
		mounts = append(mounts, corev1.VolumeMount{
			Name:      "workspace-seed-init",
			MountPath: "/seed-init",
			ReadOnly:  true,
		})
	}
	if agent.Spec.Workspace.SeedConfigMapRef != nil {
		mounts = append(mounts, corev1.VolumeMount{
			Name:      "workspace-seed-ref",
			MountPath: "/seed-ref",
			ReadOnly:  true,
		})
	}

	return &corev1.Container{
		Name:         "workspace-seeder",
		Image:        "busybox:latest",
		Command:      []string{"/bin/sh", "-c", script},
		VolumeMounts: mounts,
	}
}

func (r *LanguageAgentReconciler) buildAgentEnv(ctx context.Context, agent *langopv1alpha1.LanguageAgent, cluster *langopv1alpha1.LanguageCluster, modelURLs []string, modelNames []string, toolURLs []string) []corev1.EnvVar {
	env := []corev1.EnvVar{
		{
			Name:  "AGENT_NAME",
			Value: agent.Name,
		},
		{
			Name:  "AGENT_NAMESPACE",
			Value: agent.Namespace,
		},
		{
			Name:  "AGENT_UUID",
			Value: agent.Status.UUID,
		},
		{
			Name:  "AGENT_CLUSTER_NAME",
			Value: cluster.Name,
		},
		{
			Name:  "AGENT_CLUSTER_UUID",
			Value: string(cluster.UID),
		},
	}

	// Pass through OpenTelemetry collector endpoint from operator environment.
	// Agents are responsible for configuring their own OTEL SDK.
	if endpoint := os.Getenv("OTEL_EXPORTER_OTLP_ENDPOINT"); endpoint != "" {
		env = append(env, corev1.EnvVar{
			Name:  "OTEL_EXPORTER_OTLP_ENDPOINT",
			Value: endpoint,
		})
		env = append(env, corev1.EnvVar{
			Name:  "OTEL_SERVICE_NAME",
			Value: fmt.Sprintf("agent-%s", agent.Name),
		})

		if resourceAttrs := os.Getenv("OTEL_RESOURCE_ATTRIBUTES"); resourceAttrs != "" {
			env = append(env, corev1.EnvVar{
				Name:  "OTEL_RESOURCE_ATTRIBUTES",
				Value: resourceAttrs,
			})
		}
		if sampler := os.Getenv("OTEL_TRACES_SAMPLER"); sampler != "" {
			env = append(env, corev1.EnvVar{
				Name:  "OTEL_TRACES_SAMPLER",
				Value: sampler,
			})
		}
		if samplerArg := os.Getenv("OTEL_TRACES_SAMPLER_ARG"); samplerArg != "" {
			env = append(env, corev1.EnvVar{
				Name:  "OTEL_TRACES_SAMPLER_ARG",
				Value: samplerArg,
			})
		}
	}

	if agent.Spec.Instructions != "" {
		env = append(env, corev1.EnvVar{
			Name:  "AGENT_INSTRUCTIONS",
			Value: agent.Spec.Instructions,
		})
	}

	// AGENT_PERSONA is the role context — the runtime launcher passes it to the
	// agent CLI via --append-system-prompt. Looked up here (not in the adapter)
	// so the operator stays the single source of truth for env injection.
	if persona, err := r.fetchPersona(ctx, agent); err == nil && persona != nil {
		if text := formatPersona(persona); text != "" {
			env = append(env, corev1.EnvVar{
				Name:  "AGENT_PERSONA",
				Value: text,
			})
		}
	}

	// Model gateway URLs and names (comma-separated)
	if len(modelURLs) > 0 {
		env = append(env, corev1.EnvVar{
			Name:  "MODEL_ENDPOINT",
			Value: strings.Join(modelURLs, ","),
		})
	}
	if len(modelNames) > 0 {
		env = append(env, corev1.EnvVar{
			Name:  "LLM_MODEL",
			Value: strings.Join(modelNames, ","),
		})
	}

	// MCP tool server URLs (comma-separated)
	if len(toolURLs) > 0 {
		env = append(env, corev1.EnvVar{
			Name:  "MCP_SERVERS",
			Value: strings.Join(toolURLs, ","),
		})
	}

	// User-specified env vars (may override any of the above)
	env = append(env, agent.Spec.Deployment.Env...)

	return env
}

func (r *LanguageAgentReconciler) fetchPersona(ctx context.Context, agent *langopv1alpha1.LanguageAgent) (*langopv1alpha1.LanguagePersona, error) {
	if agent.Spec.Persona == "" {
		return nil, nil
	}

	persona := &langopv1alpha1.LanguagePersona{}
	if err := r.Get(ctx, types.NamespacedName{Name: agent.Spec.Persona, Namespace: agent.Namespace}, persona); err != nil {
		if apierrors.IsNotFound(err) {
			return nil, fmt.Errorf("persona %s/%s not found", agent.Namespace, agent.Spec.Persona)
		}
		return nil, fmt.Errorf("failed to get persona %s/%s: %w", agent.Namespace, agent.Spec.Persona, err)
	}

	if persona.Status.Phase != events.PhaseStatusReady {
		return nil, fmt.Errorf("persona %s/%s is not ready (phase: %s)", agent.Namespace, agent.Spec.Persona, persona.Status.Phase)
	}

	return persona, nil
}

// formatPersona renders a LanguagePersona's tone/personality/expertise into a
// plain-text paragraph suitable for use as a system-prompt append. Empty fields
// are skipped; if all fields are empty, returns "".
func formatPersona(persona *langopv1alpha1.LanguagePersona) string {
	if persona == nil {
		return ""
	}
	var lines []string
	if t := strings.TrimSpace(persona.Spec.Tone); t != "" {
		lines = append(lines, "Tone: "+t+".")
	}
	if p := strings.TrimSpace(persona.Spec.Personality); p != "" {
		lines = append(lines, "Personality: "+p+".")
	}
	if e := strings.TrimSpace(persona.Spec.Expertise); e != "" {
		lines = append(lines, "Expertise: "+e+".")
	}
	return strings.Join(lines, "\n")
}

// generateCredential returns a cryptographically random 32-byte hex string.
func generateCredential() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}

// reconcileRuntimeSecret resolves workingAgent's declared credentials into env vars.
// Each CredentialSpec injects an env var named after the credential:
//   - ValueFrom: the referenced Secret's keys are injected via envFrom directly.
//   - Value: the literal is stored in an operator-managed Secret, GC'd on agent deletion.
//   - neither: a value is auto-generated once and preserved on subsequent reconciles.
//
// Credentials typically originate from the referenced LanguageAgentRuntime and are
// merged into workingAgent.Spec by ApplyRuntimeDefaults before this runs.
func (r *LanguageAgentReconciler) reconcileRuntimeSecret(
	ctx context.Context,
	agent *langopv1alpha1.LanguageAgent,
	workingAgent *langopv1alpha1.LanguageAgent,
) error {
	secretName := agent.Name + "-runtime"
	secretData := map[string][]byte{}
	var refEnvFrom []corev1.EnvFromSource

	// Load existing secret so auto-generated values can be preserved across reconciles.
	existing := &corev1.Secret{}
	existingData := map[string][]byte{}
	if err := r.Get(ctx, types.NamespacedName{Name: secretName, Namespace: agent.Namespace}, existing); err == nil {
		existingData = existing.Data
	}

	for _, c := range workingAgent.Spec.Credentials {
		switch {
		case c.ValueFrom != nil:
			// Inject the referenced Secret's keys directly.
			refEnvFrom = append(refEnvFrom, corev1.EnvFromSource{
				SecretRef: &corev1.SecretEnvSource{
					LocalObjectReference: corev1.LocalObjectReference{Name: c.ValueFrom.Name},
				},
			})
		case c.Value != "":
			secretData[c.Name] = []byte(c.Value)
		default:
			// Auto-generate: preserve existing value if present, otherwise generate new.
			value := string(existingData[c.Name])
			if value == "" {
				var err error
				value, err = generateCredential()
				if err != nil {
					return fmt.Errorf("generating credential %q: %w", c.Name, err)
				}
			}
			secretData[c.Name] = []byte(value)
		}
	}

	if len(secretData) > 0 {
		secret := &corev1.Secret{
			ObjectMeta: metav1.ObjectMeta{
				Name:      secretName,
				Namespace: agent.Namespace,
			},
		}
		if err := CreateOrUpdateOwned(ctx, r.Client, r.Scheme, agent, secret, func() error {
			secret.Data = secretData
			return nil
		}); err != nil {
			return fmt.Errorf("reconciling runtime secret %s/%s: %w", agent.Namespace, secretName, err)
		}
		// Prepend managed secret to envFrom so it takes precedence
		workingAgent.Spec.Deployment.EnvFrom = append(
			[]corev1.EnvFromSource{{SecretRef: &corev1.SecretEnvSource{
				LocalObjectReference: corev1.LocalObjectReference{Name: secretName},
			}}},
			workingAgent.Spec.Deployment.EnvFrom...,
		)
	} else {
		// No inline credentials — delete managed secret if it exists
		secret := &corev1.Secret{}
		err := r.Get(ctx, types.NamespacedName{Name: secretName, Namespace: agent.Namespace}, secret)
		if err == nil {
			if err := r.Delete(ctx, secret); err != nil && !apierrors.IsNotFound(err) {
				return fmt.Errorf("deleting stale runtime secret %s/%s: %w", agent.Namespace, secretName, err)
			}
		}
	}

	// Append ref-based envFrom entries
	workingAgent.Spec.Deployment.EnvFrom = append(workingAgent.Spec.Deployment.EnvFrom, refEnvFrom...)

	return nil
}
