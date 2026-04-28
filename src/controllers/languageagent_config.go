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

// generateCredential returns a cryptographically random 32-byte hex string.
func generateCredential() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}

// reconcileRuntimeSecret creates or updates a managed Secret containing credentials for
// runtime-specific configuration (opencode, openclaw). When inline values are provided,
// the operator owns the Secret and GC's it on agent deletion. When *Ref variants are used,
// the referenced Secret is injected into workingAgent's envFrom directly. When neither is
// set, credentials are auto-generated once and preserved on subsequent reconciles.
func (r *LanguageAgentReconciler) reconcileRuntimeSecret(
	ctx context.Context,
	agent *langopv1alpha1.LanguageAgent,
	workingAgent *langopv1alpha1.LanguageAgent,
) error {
	secretName := agent.Name + "-runtime"
	secretData := map[string][]byte{}
	var extraEnv []corev1.EnvVar
	var refEnvFrom []corev1.EnvFromSource

	// Load existing secret so auto-generated values can be preserved across reconciles.
	existing := &corev1.Secret{}
	existingData := map[string][]byte{}
	if err := r.Get(ctx, types.NamespacedName{Name: secretName, Namespace: agent.Namespace}, existing); err == nil {
		existingData = existing.Data
	}

	// opencode inline credentials → managed secret
	// Use workingAgent so runtime-provided config (e.g. spec.opencode from a LanguageAgentRuntime) is respected.
	if workingAgent.Spec.Opencode != nil && workingAgent.Spec.Opencode.Enabled != nil && *workingAgent.Spec.Opencode.Enabled {
		oc := workingAgent.Spec.Opencode
		if oc.Password != "" {
			username := oc.Username
			if username == "" {
				username = "opencode"
			}
			secretData["OPENCODE_SERVER_USERNAME"] = []byte(username)
			secretData["OPENCODE_SERVER_PASSWORD"] = []byte(oc.Password)
		} else if oc.PasswordRef != nil {
			// Username as literal env var if specified alongside a ref
			if oc.Username != "" {
				extraEnv = append(extraEnv, corev1.EnvVar{
					Name:  "OPENCODE_SERVER_USERNAME",
					Value: oc.Username,
				})
			}
			refEnvFrom = append(refEnvFrom, corev1.EnvFromSource{
				SecretRef: &corev1.SecretEnvSource{
					LocalObjectReference: corev1.LocalObjectReference{Name: oc.PasswordRef.Name},
				},
			})
		} else {
			// Auto-generate: preserve existing value if present, otherwise generate new.
			password := string(existingData["OPENCODE_SERVER_PASSWORD"])
			if password == "" {
				var err error
				password, err = generateCredential()
				if err != nil {
					return fmt.Errorf("generating opencode password: %w", err)
				}
			}
			username := oc.Username
			if username == "" {
				username = "opencode"
			}
			secretData["OPENCODE_SERVER_USERNAME"] = []byte(username)
			secretData["OPENCODE_SERVER_PASSWORD"] = []byte(password)
		}
	}

	// openclaw inline credentials → managed secret
	if workingAgent.Spec.Openclaw != nil && workingAgent.Spec.Openclaw.Enabled != nil && *workingAgent.Spec.Openclaw.Enabled {
		oc := workingAgent.Spec.Openclaw
		if oc.Token != "" {
			secretData["OPENCLAW_GATEWAY_TOKEN"] = []byte(oc.Token)
		} else if oc.TokenRef != nil {
			refEnvFrom = append(refEnvFrom, corev1.EnvFromSource{
				SecretRef: &corev1.SecretEnvSource{
					LocalObjectReference: corev1.LocalObjectReference{Name: oc.TokenRef.Name},
				},
			})
		} else {
			// Auto-generate: preserve existing value if present, otherwise generate new.
			token := string(existingData["OPENCLAW_GATEWAY_TOKEN"])
			if token == "" {
				var err error
				token, err = generateCredential()
				if err != nil {
					return fmt.Errorf("generating openclaw token: %w", err)
				}
			}
			secretData["OPENCLAW_GATEWAY_TOKEN"] = []byte(token)
		}
	}

	// claude-code credentials → managed secret, ref envFrom, or gateway-routed placeholder
	if workingAgent.Spec.ClaudeCode != nil && workingAgent.Spec.ClaudeCode.Enabled != nil && *workingAgent.Spec.ClaudeCode.Enabled {
		cc := workingAgent.Spec.ClaudeCode
		if cc.APIKey != "" {
			secretData["ANTHROPIC_API_KEY"] = []byte(cc.APIKey)
		} else if cc.APIKeyRef != nil {
			key := cc.APIKeyRef.Key
			if key == "" {
				key = "api-key"
			}
			extraEnv = append(extraEnv, corev1.EnvVar{
				Name: "ANTHROPIC_API_KEY",
				ValueFrom: &corev1.EnvVarSource{
					SecretKeyRef: &corev1.SecretKeySelector{
						LocalObjectReference: corev1.LocalObjectReference{Name: cc.APIKeyRef.Name},
						Key:                  key,
					},
				},
			})
		} else {
			// Gateway-routed mode: inject placeholder key and route SDK traffic to the
			// LiteLLM gateway via ANTHROPIC_BASE_URL so calls never reach api.anthropic.com.
			extraEnv = append(extraEnv, corev1.EnvVar{
				Name:  "ANTHROPIC_API_KEY",
				Value: "sk-langop-proxy",
			})
			extraEnv = append(extraEnv, corev1.EnvVar{
				Name:  "ANTHROPIC_BASE_URL",
				Value: serviceURL("gateway", agent.Namespace, network.GatewayServicePort),
			})
		}
		if cc.MaxTurns != nil {
			extraEnv = append(extraEnv, corev1.EnvVar{
				Name:  "CLAUDE_CODE_MAX_TURNS",
				Value: fmt.Sprintf("%d", *cc.MaxTurns),
			})
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
	// Prepend any literal env vars (e.g. username alongside a passwordRef)
	workingAgent.Spec.Deployment.Env = append(extraEnv, workingAgent.Spec.Deployment.Env...)

	return nil
}
