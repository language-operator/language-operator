package controllers

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
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
	"k8s.io/utils/ptr"
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
		if cfg.Tools == nil {
			cfg.Tools = make(map[string]toolConfigYAML)
		}
		// Full Streamable HTTP MCP URL (includes /mcp) so MCP clients can use it directly.
		cfg.Tools[tool.Name] = toolConfigYAML{Endpoint: mcpToolEndpoint(tool, agent.Namespace), Protocol: "mcp"}
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

// repositoryImage is the git client image used by the repository init container.
const repositoryImage = "alpine/git:latest"

// workspaceMountPath returns the path the workspace PVC is mounted at, defaulting
// to /workspace when the workspace is enabled without an explicit mountPath.
func workspaceMountPath(agent *langopv1alpha1.LanguageAgent) string {
	if agent.Spec.Workspace != nil && agent.Spec.Workspace.MountPath != "" {
		return agent.Spec.Workspace.MountPath
	}
	return "/workspace"
}

// agentHasRepository reports whether a git repository clone is configured.
func agentHasRepository(agent *langopv1alpha1.LanguageAgent) bool {
	return agent.Spec.Repository != nil && strings.TrimSpace(agent.Spec.Repository.URL) != ""
}

// deriveRepoName extracts a repository directory name from a git URL, stripping a
// trailing slash and ".git" suffix. Handles both URL forms (https://host/foo/bar.git)
// and the scp-like SSH form (git@host:foo/bar.git), yielding "bar" in both cases.
func deriveRepoName(rawURL string) string {
	s := strings.TrimSuffix(strings.TrimRight(rawURL, "/"), ".git")
	if i := strings.LastIndexAny(s, "/:"); i >= 0 {
		s = s[i+1:]
	}
	if s == "" {
		return "repository"
	}
	return s
}

// repositoryDir returns the absolute path the repository is cloned into:
// <workspace mountPath>/<spec.repository.path or derived repo name>. Empty when
// no repository is configured.
func repositoryDir(agent *langopv1alpha1.LanguageAgent) string {
	if !agentHasRepository(agent) {
		return ""
	}
	sub := agent.Spec.Repository.Path
	if sub == "" {
		sub = deriveRepoName(agent.Spec.Repository.URL)
	}
	return filepath.Join(workspaceMountPath(agent), sub)
}

// repositoryCredentialsVolume is the pod volume holding spec.repository.secretRef.
const repositoryCredentialsVolume = "repository-credentials"

// repositoryCredentialsMountPath is where the credentials Secret is mounted, read-only,
// in the repository init container and the agent container.
const repositoryCredentialsMountPath = "/var/run/secrets/langop.io/git"

// gitIdentityEmailDomain is the domain of the default commit identity
// (<agent>@<namespace>.<domain>); a control plane overrides it through GIT_AUTHOR_* /
// GIT_COMMITTER_* in spec.deployment.env.
const gitIdentityEmailDomain = "langop.io"

// repositoryUsesSSH reports whether the repository URL is fetched over SSH (an ssh://
// URL or the scp-like git@host:path form) rather than HTTP(S).
func repositoryUsesSSH(rawURL string) bool {
	rawURL = strings.TrimSpace(rawURL)
	if strings.HasPrefix(rawURL, "ssh://") {
		return true
	}
	return !strings.Contains(rawURL, "://") && strings.Contains(rawURL, "@") && strings.Contains(rawURL, ":")
}

// repositoryVendor returns spec.repository.vendor, falling back to the host mapping for
// objects admitted before the webhook defaulted it.
func repositoryVendor(agent *langopv1alpha1.LanguageAgent) string {
	if !agentHasRepository(agent) {
		return ""
	}
	if v := agent.Spec.Repository.Vendor; v != "" {
		return v
	}
	return langopv1alpha1.DefaultRepositoryVendor(agent.Spec.Repository.URL)
}

// buildRepositoryVolumes returns the pod-level volume holding git credentials when
// spec.repository.secretRef is set. The Secret is mounted read-only into the
// repository init container and the agent container; the operator never reads its
// contents.
func buildRepositoryVolumes(agent *langopv1alpha1.LanguageAgent) []corev1.Volume {
	if !agentHasRepository(agent) || agent.Spec.Repository.SecretRef == nil {
		return nil
	}
	mode := int32(0o400)
	return []corev1.Volume{
		{
			Name: repositoryCredentialsVolume,
			VolumeSource: corev1.VolumeSource{
				Secret: &corev1.SecretVolumeSource{
					SecretName:  agent.Spec.Repository.SecretRef.Name,
					DefaultMode: &mode,
				},
			},
		},
	}
}

// buildRepositoryCredentialMount returns the read-only mount of the credentials
// Secret, or nil when no secretRef is set.
func buildRepositoryCredentialMount(agent *langopv1alpha1.LanguageAgent) *corev1.VolumeMount {
	if !agentHasRepository(agent) || agent.Spec.Repository.SecretRef == nil {
		return nil
	}
	return &corev1.VolumeMount{
		Name:      repositoryCredentialsVolume,
		MountPath: repositoryCredentialsMountPath,
		ReadOnly:  true,
	}
}

// gitCredentialHelper is an inline git credential helper that answers "get" from the
// mounted Secret: username from the `username` key (else the x-access-token
// convention), password from `password` or `token`. Reading the files on every call
// means a rotated Secret is picked up without a pod restart; nothing is ever written.
const gitCredentialHelper = `!f() { d=` + repositoryCredentialsMountPath + `; [ "$1" = get ] || exit 0; u=x-access-token; [ -f "$d/username" ] && u=$(cat "$d/username"); if [ -f "$d/password" ]; then p=$(cat "$d/password"); elif [ -f "$d/token" ]; then p=$(cat "$d/token"); else exit 0; fi; printf 'username=%s\npassword=%s\n' "$u" "$p"; }; f`

// buildGitEnv returns the git configuration for the repository init container and the
// agent container, expressed through GIT_CONFIG_COUNT/GIT_CONFIG_KEY_n/GIT_CONFIG_VALUE_n
// so no file has to be written into a read-only image: a default commit identity
// (<agent>@<namespace>.langop.io, overridable with GIT_AUTHOR_*/GIT_COMMITTER_* in
// spec.deployment.env), a host-scoped credential helper for HTTPS remotes, or
// GIT_SSH_COMMAND pointing at the mounted key for SSH remotes. Nil without a repository.
func buildGitEnv(agent *langopv1alpha1.LanguageAgent) []corev1.EnvVar {
	if !agentHasRepository(agent) {
		return nil
	}
	repo := agent.Spec.Repository
	entries := [][2]string{
		{"user.name", agent.Name},
		{"user.email", fmt.Sprintf("%s@%s.%s", agent.Name, agent.Namespace, gitIdentityEmailDomain)},
	}
	var env []corev1.EnvVar
	if repo.SecretRef != nil {
		if repositoryUsesSSH(repo.URL) {
			env = append(env, corev1.EnvVar{
				Name: "GIT_SSH_COMMAND",
				Value: fmt.Sprintf("ssh -i %s/ssh-privatekey -o IdentitiesOnly=yes -o StrictHostKeyChecking=no -o UserKnownHostsFile=/dev/null",
					repositoryCredentialsMountPath),
			})
		} else if host := langopv1alpha1.RepositoryHost(repo.URL); host != "" {
			entries = append(entries, [2]string{"credential.https://" + host + ".helper", gitCredentialHelper})
		}
		env = append(env, corev1.EnvVar{Name: "GIT_TERMINAL_PROMPT", Value: "0"})
	}
	env = append(env, corev1.EnvVar{Name: "GIT_CONFIG_COUNT", Value: strconv.Itoa(len(entries))})
	for i, kv := range entries {
		env = append(env,
			corev1.EnvVar{Name: fmt.Sprintf("GIT_CONFIG_KEY_%d", i), Value: kv[0]},
			corev1.EnvVar{Name: fmt.Sprintf("GIT_CONFIG_VALUE_%d", i), Value: kv[1]},
		)
	}
	return env
}

// buildVendorEnv returns the credential the repository vendor's CLI reads, taken from
// the `token` key of spec.repository.secretRef: GH_TOKEN (and GH_HOST for GitHub
// Enterprise) for github, GITLAB_TOKEN (and GITLAB_HOST for self-hosted) for gitlab,
// nothing for git. The reference is optional so a Secret without a `token` key (SSH
// key, username and password) still starts. Agent container only.
func buildVendorEnv(agent *langopv1alpha1.LanguageAgent) []corev1.EnvVar {
	if !agentHasRepository(agent) || agent.Spec.Repository.SecretRef == nil {
		return nil
	}
	token := &corev1.EnvVarSource{
		SecretKeyRef: &corev1.SecretKeySelector{
			LocalObjectReference: corev1.LocalObjectReference{Name: agent.Spec.Repository.SecretRef.Name},
			Key:                  "token",
			Optional:             ptr.To(true),
		},
	}
	host := langopv1alpha1.RepositoryHost(agent.Spec.Repository.URL)
	switch repositoryVendor(agent) {
	case langopv1alpha1.RepositoryVendorGitHub:
		env := []corev1.EnvVar{{Name: "GH_TOKEN", ValueFrom: token}}
		if host != "" && host != "github.com" {
			env = append(env, corev1.EnvVar{Name: "GH_HOST", Value: host})
		}
		return env
	case langopv1alpha1.RepositoryVendorGitLab:
		env := []corev1.EnvVar{{Name: "GITLAB_TOKEN", ValueFrom: token}}
		if host != "" && host != "gitlab.com" {
			env = append(env, corev1.EnvVar{Name: "GITLAB_HOST", Value: host})
		}
		return env
	default:
		return nil
	}
}

// buildRepositoryInitContainer returns the "repository" init container that clones
// spec.repository into the workspace, or nil when no repository is configured. The
// clone uses clone-once semantics (skipped when the target already contains a .git
// directory), so agent edits and commits survive pod restarts — mirroring the
// workspace-seeder's seed-once behavior.
//
// Authentication comes from the same GIT_CONFIG_* / GIT_SSH_COMMAND environment the
// agent container gets (buildGitEnv), reading the mounted Secret; the operator never
// reads the Secret and the script never embeds a credential in the checkout.
func buildRepositoryInitContainer(agent *langopv1alpha1.LanguageAgent) *corev1.Container {
	if !agentHasRepository(agent) {
		return nil
	}

	repo := agent.Spec.Repository
	target := repositoryDir(agent)
	mountPath := workspaceMountPath(agent)

	depthFlag := ""
	if repo.Depth > 0 {
		depthFlag = fmt.Sprintf("--depth %d", repo.Depth)
	}

	script := fmt.Sprintf(`set -e
TARGET=%q
URL=%q
REF=%q
if [ -d "$TARGET/.git" ]; then
  echo "repository already present at $TARGET, skipping clone"
  exit 0
fi
if [ -n "$REF" ]; then
  git clone %s --branch "$REF" "$URL" "$TARGET" 2>/dev/null || {
    rm -rf "$TARGET"
    git clone "$URL" "$TARGET"
    git -C "$TARGET" checkout "$REF"
  }
else
  git clone %s "$URL" "$TARGET"
fi
echo "cloned $URL into $TARGET"`, target, repo.URL, repo.Ref, depthFlag, depthFlag)

	mounts := []corev1.VolumeMount{
		{
			Name:      "workspace",
			MountPath: mountPath,
		},
	}
	if credMount := buildRepositoryCredentialMount(agent); credMount != nil {
		mounts = append(mounts, *credMount)
	}

	return &corev1.Container{
		Name:         "repository",
		Image:        repositoryImage,
		Command:      []string{"/bin/sh", "-c", script},
		VolumeMounts: mounts,
		Env: append([]corev1.EnvVar{
			{Name: "HOME", Value: "/tmp"},
		}, buildGitEnv(agent)...),
	}
}

// buildAgentEnv returns the operator-managed environment for a container. `extra`
// entries (the agent container's git and vendor CLI credentials) are appended after
// the contract variables and before spec.deployment.env, so user env still wins.
func (r *LanguageAgentReconciler) buildAgentEnv(ctx context.Context, agent *langopv1alpha1.LanguageAgent, cluster *langopv1alpha1.LanguageCluster, modelURLs []string, modelNames []string, toolURLs []string, extra ...corev1.EnvVar) []corev1.EnvVar {
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

	// AGENT_REPO_DIR points at the cloned repository so the runtime (and any init
	// containers) can open inside it. Injected into every container per spec/agents.md.
	if dir := repositoryDir(agent); dir != "" {
		env = append(env, corev1.EnvVar{
			Name:  "AGENT_REPO_DIR",
			Value: dir,
		})
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

	env = append(env, extra...)

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
