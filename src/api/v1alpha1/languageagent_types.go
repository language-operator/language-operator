package v1alpha1

import (
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// AgentPort describes a single network port that an agent container exposes.
type AgentPort struct {
	// Name uniquely identifies this port within the agent (e.g., "web", "ws").
	// Used as the Service port name; must conform to Kubernetes port-name rules.
	// +kubebuilder:validation:Required
	// +kubebuilder:validation:Pattern=`^[a-z][a-z0-9-]*$`
	// +kubebuilder:validation:MaxLength=15
	Name string `json:"name"`

	// Port is the port number the container listens on.
	// +kubebuilder:validation:Required
	// +kubebuilder:validation:Minimum=1
	// +kubebuilder:validation:Maximum=65535
	Port int32 `json:"port"`

	// Protocol is the transport protocol. Defaults to TCP.
	// +kubebuilder:validation:Enum=TCP;UDP;SCTP
	// +kubebuilder:default=TCP
	// +optional
	Protocol corev1.Protocol `json:"protocol,omitempty"`

	// Expose controls whether ingress/HTTPRoute routes to this port.
	// At most one port should have expose: true; if none, the first port is used.
	// +kubebuilder:default=false
	// +optional
	Expose *bool `json:"expose,omitempty"`
}

// LanguageAgentSpec defines the desired state of LanguageAgent
type LanguageAgentSpec struct {
	// Runtime is the name of a LanguageAgentRuntime that provides preset configuration
	// (image, port, init containers, env vars, probes, etc.).
	// When set, spec.image is optional; the runtime provides a default.
	// +optional
	Runtime string `json:"runtime,omitempty"`

	// Image is the container image to run for this agent.
	// Required unless spec.runtime is set (the runtime provides a default image).
	// +kubebuilder:validation:Pattern=`^([a-z0-9]+([._-][a-z0-9]+)*\/)*[a-z0-9]+([._-][a-z0-9]+)*(:[a-z0-9]+([._-][a-z0-9]+)*)?$`
	// +optional
	Image string `json:"image,omitempty"`

	// Models is a list of LanguageModel references this agent can use
	// +optional
	Models []ModelReference `json:"models,omitempty"`

	// Tools is a list of LanguageTool references available to this agent
	// +optional
	Tools []ToolReference `json:"tools,omitempty"`

	// Persona is the name of a LanguagePersona this agent uses
	// +optional
	Persona string `json:"persona,omitempty"`

	// Instructions provides system instructions for the agent.
	// Delivered as the top-level "instructions" field in /etc/agent/config.yaml.
	// +optional
	Instructions string `json:"instructions,omitempty"`

	// Workspace defines persistent storage for the agent
	// +optional
	Workspace *WorkspaceSpec `json:"workspace,omitempty"`

	// NetworkPolicies defines ingress and egress rules for this agent.
	// Rules mirror the native Kubernetes NetworkPolicy shape.
	// +optional
	NetworkPolicies *AgentNetworkPolicies `json:"networkPolicies,omitempty"`

	// Ports defines all network ports this agent exposes.
	// At most one entry should have expose: true (the ingress target);
	// if none are marked, the first port is used for ingress routing.
	// Defaults to a single HTTP port on 8080 when not set.
	// +optional
	// +listType=map
	// +listMapKey=name
	Ports []AgentPort `json:"ports,omitempty"`

	// Deployment groups Kubernetes-specific pod and container configuration.
	// +optional
	Deployment DeploymentSpec `json:"deployment,omitempty"`

	// Opencode holds configuration specific to the opencode runtime.
	// Only effective when spec.runtime is "opencode".
	// +optional
	Opencode *OpencodeConfig `json:"opencode,omitempty"`

	// Openclaw holds configuration specific to the openclaw runtime.
	// Only effective when spec.runtime is "openclaw".
	// +optional
	Openclaw *OpenclawConfig `json:"openclaw,omitempty"`

	// ClaudeCode holds configuration specific to the claude-code runtime.
	// Only effective when spec.runtime is "claude-code".
	// +optional
	ClaudeCode *ClaudeCodeConfig `json:"claudeCode,omitempty"`

	// SelfConfigure controls whether this agent may submit LanguageAgentSelfConfig
	// requests to modify its own spec at runtime. When enabled, the operator grants
	// the agent's ServiceAccount permission to create LanguageAgentSelfConfig resources.
	// +optional
	SelfConfigure *SelfConfigureSpec `json:"selfConfigure,omitempty"`

	// Monitoring configures Prometheus Operator integration for this agent.
	// When set, the operator creates a ServiceMonitor and/or PrometheusRule resource.
	// Requires prometheus-operator to be installed in the cluster; silently skipped otherwise.
	// +optional
	Monitoring *AgentMonitoringSpec `json:"monitoring,omitempty"`

	// Auth controls whether OIDC authentication is applied to this agent's ingress route.
	// When nil, the agent inherits the cluster-level auth setting from its LanguageCluster.
	// +optional
	Auth *AgentAuthSpec `json:"auth,omitempty"`
}

// AgentAuthSpec controls OIDC authentication for a LanguageAgent's ingress route.
type AgentAuthSpec struct {
	// Enabled explicitly enables or disables OIDC authentication for this agent.
	// When nil, the agent inherits the cluster-level auth.enabled setting.
	// +optional
	Enabled *bool `json:"enabled,omitempty"`
}

// AgentMonitoringSpec defines Prometheus Operator integration for a LanguageAgent.
type AgentMonitoringSpec struct {
	// ServiceMonitor configures a ServiceMonitor resource for this agent.
	// +optional
	ServiceMonitor *AgentServiceMonitorSpec `json:"serviceMonitor,omitempty"`

	// Rules defines PrometheusRule groups for this agent.
	// When non-empty, the operator creates a PrometheusRule resource.
	// +optional
	Rules []PrometheusRuleGroup `json:"rules,omitempty"`
}

// AgentServiceMonitorSpec configures a Prometheus Operator ServiceMonitor for an agent.
type AgentServiceMonitorSpec struct {
	// Enabled controls whether a ServiceMonitor is created for this agent.
	Enabled bool `json:"enabled"`

	// Port is the name of the service port to scrape for metrics.
	// Defaults to the name of the first port in spec.ports, or "http" if no ports are defined.
	// +optional
	Port string `json:"port,omitempty"`

	// Path is the HTTP path to scrape for metrics. Defaults to /metrics.
	// +optional
	Path string `json:"path,omitempty"`

	// Interval is the scrape interval (e.g. "30s"). Uses the Prometheus default when omitted.
	// +optional
	Interval string `json:"interval,omitempty"`

	// ScrapeTimeout is the per-scrape timeout. Uses the Prometheus default when omitted.
	// +optional
	ScrapeTimeout string `json:"scrapeTimeout,omitempty"`

	// Labels are additional labels added to the ServiceMonitor metadata.
	// +optional
	Labels map[string]string `json:"labels,omitempty"`
}

// PrometheusRuleGroup defines a group of Prometheus alerting or recording rules.
type PrometheusRuleGroup struct {
	// Name is the name of the rule group.
	// +kubebuilder:validation:Required
	Name string `json:"name"`

	// Interval is the evaluation interval for this group. Uses the Prometheus default when omitted.
	// +optional
	Interval string `json:"interval,omitempty"`

	// Rules is the list of alerting or recording rules in this group.
	// +kubebuilder:validation:MinItems=1
	Rules []PrometheusAlertingRule `json:"rules"`
}

// PrometheusAlertingRule defines a single Prometheus alerting or recording rule.
type PrometheusAlertingRule struct {
	// Alert is the alert name. Leave empty for recording rules.
	// +optional
	Alert string `json:"alert,omitempty"`

	// Record is the output metric name for recording rules. Leave empty for alerting rules.
	// +optional
	Record string `json:"record,omitempty"`

	// Expr is the PromQL expression evaluated at each evaluation cycle.
	// +kubebuilder:validation:Required
	Expr string `json:"expr"`

	// For is the duration the condition must be true before the alert fires.
	// Only valid for alerting rules.
	// +optional
	For string `json:"for,omitempty"`

	// Labels are labels attached to the alert or recording rule.
	// +optional
	Labels map[string]string `json:"labels,omitempty"`

	// Annotations are annotations attached to the alert. Only valid for alerting rules.
	// +optional
	Annotations map[string]string `json:"annotations,omitempty"`
}

// ModelReference references a LanguageModel
type ModelReference struct {
	// Name is the name of the LanguageModel
	// +kubebuilder:validation:Required
	// +kubebuilder:validation:Pattern=`^[a-z0-9]([-a-z0-9]*[a-z0-9])?$`
	// +kubebuilder:validation:MaxLength=63
	Name string `json:"name"`

	// Role defines the purpose of this model — a hint for the agent runtime for model selection
	// (e.g. prefer role=primary for general calls, role=reasoning for chain-of-thought).
	// The operator does not enforce routing by role; it is surfaced in the agent config (agent.json).
	// +kubebuilder:validation:Enum=primary;fallback;reasoning;tool-calling;summarization
	// +kubebuilder:default=primary
	// +optional
	Role string `json:"role,omitempty"`

	// Priority for model selection — a hint for the agent runtime (lower value = higher priority).
	// The operator does not enforce priority; it is surfaced in the agent config (agent.json).
	// +optional
	Priority *int32 `json:"priority,omitempty"`
}

// ToolReference references a LanguageTool
type ToolReference struct {
	// Name is the name of the LanguageTool
	// +kubebuilder:validation:Required
	// +kubebuilder:validation:Pattern=`^[a-z0-9]([-a-z0-9]*[a-z0-9])?$`
	// +kubebuilder:validation:MaxLength=63
	Name string `json:"name"`

	// Enabled indicates if this tool is available to the agent.
	// Defaults to true. Set to false to explicitly disable the tool without removing it.
	// +kubebuilder:default=true
	// +optional
	Enabled *bool `json:"enabled,omitempty"`
}

// WorkspaceSpec defines persistent workspace storage for an agent
type WorkspaceSpec struct {
	// Enabled controls whether to create a workspace volume.
	// Defaults to true. Set to false to explicitly disable without removing the workspace config.
	// +kubebuilder:default=true
	// +optional
	Enabled *bool `json:"enabled,omitempty"`

	// Size is the requested storage size (e.g., "10Gi", "1.5Ti", "500Mi")
	// Supports integer and decimal quantities with standard Kubernetes suffixes
	// +kubebuilder:validation:Pattern=`^([0-9]*\.?[0-9]+)(Ei|Pi|Ti|Gi|Mi|Ki|E|P|T|G|M|K|m)?$`
	// +kubebuilder:validation:MinLength=1
	// +kubebuilder:default="10Gi"
	// +optional
	Size string `json:"size,omitempty"`

	// StorageClassName specifies the StorageClass for the PVC
	// If not specified, uses the cluster default
	// +optional
	StorageClassName *string `json:"storageClassName,omitempty"`

	// AccessMode defines the volume access mode
	// +kubebuilder:validation:Enum=ReadWriteOnce;ReadWriteMany
	// +kubebuilder:default=ReadWriteOnce
	// +optional
	AccessMode string `json:"accessMode,omitempty"`

	// MountPath is where the workspace is mounted in containers
	// +kubebuilder:default="/workspace"
	// +optional
	MountPath string `json:"mountPath,omitempty"`

	// Retain prevents the workspace PVC from being deleted when the agent is deleted.
	// When true, the PVC's ownerReference is removed during cleanup so Kubernetes GC
	// does not collect it. The orphaned PVC name is surfaced in status.workspacePVCName.
	// Defaults to false.
	// +kubebuilder:default=false
	// +optional
	Retain *bool `json:"retain,omitempty"`

	// InitialFiles are seeded into the workspace PVC on first boot only.
	// Keys are filenames (must be valid ConfigMap keys: alphanumeric, '.', '-', '_').
	// Files are not overwritten if they already exist on the PVC.
	// +optional
	InitialFiles map[string]string `json:"initialFiles,omitempty"`

	// SeedConfigMapRef references an external ConfigMap whose keys are filenames
	// and values are file contents. Merged with InitialFiles; InitialFiles wins on collision.
	// +optional
	SeedConfigMapRef *corev1.LocalObjectReference `json:"seedConfigMapRef,omitempty"`
}

// OpencodeConfig holds configuration specific to the opencode runtime.
// Effective only when spec.runtime is "opencode".
type OpencodeConfig struct {
	// Enabled activates opencode credential management for this agent.
	// Set to true in a LanguageAgentRuntime to trigger auto-generation of credentials
	// without requiring any explicit config on the LanguageAgent.
	// +kubebuilder:default=false
	// +optional
	Enabled *bool `json:"enabled,omitempty"`

	// Username for HTTP Basic Auth. Defaults to "opencode" if not set.
	// Sets OPENCODE_SERVER_USERNAME in the agent container.
	// +optional
	Username string `json:"username,omitempty"`

	// Password is the HTTP Basic Auth password (inline).
	// The operator creates a managed Secret and injects it via envFrom.
	// Mutually exclusive with PasswordRef.
	// +optional
	Password string `json:"password,omitempty"`

	// PasswordRef references a Secret whose keys are injected via envFrom.
	// The Secret must contain OPENCODE_SERVER_PASSWORD (and optionally OPENCODE_SERVER_USERNAME).
	// Mutually exclusive with Password.
	// +optional
	PasswordRef *RuntimeSecretRef `json:"passwordRef,omitempty"`
}

// OpenclawConfig holds configuration specific to the openclaw runtime.
// Effective only when spec.runtime is "openclaw".
type OpenclawConfig struct {
	// Enabled activates openclaw credential management for this agent.
	// Set to true in a LanguageAgentRuntime to trigger auto-generation of OPENCLAW_GATEWAY_TOKEN
	// without requiring any explicit config on the LanguageAgent.
	// +kubebuilder:default=false
	// +optional
	Enabled *bool `json:"enabled,omitempty"`

	// Token is the gateway authentication token (inline).
	// The operator creates a managed Secret and injects it via envFrom.
	// Mutually exclusive with TokenRef.
	// +optional
	Token string `json:"token,omitempty"`

	// TokenRef references a Secret whose keys are injected via envFrom.
	// The Secret must contain OPENCLAW_GATEWAY_TOKEN.
	// Mutually exclusive with Token.
	// +optional
	TokenRef *RuntimeSecretRef `json:"tokenRef,omitempty"`
}

// ClaudeCodeConfig holds configuration specific to the claude-code runtime.
// Effective only when spec.runtime is "claude-code". Authentication is interactive
// (run `/login` inside the agent terminal); credentials persist on the workspace PVC.
type ClaudeCodeConfig struct {
	// MaxTurns limits the number of agentic turns per request.
	// Sets CLAUDE_CODE_MAX_TURNS in the agent container.
	// +optional
	MaxTurns *int32 `json:"maxTurns,omitempty"`
}

// RuntimeSecretRef references a Secret in the same namespace.
// All keys in the Secret are injected as env vars via envFrom.
type RuntimeSecretRef struct {
	// Name is the name of the Secret.
	// +kubebuilder:validation:Required
	Name string `json:"name"`
}

// LanguageAgentStatus defines the observed state of LanguageAgent
type LanguageAgentStatus struct {
	// Phase represents the current phase of the agent
	// +kubebuilder:validation:Enum=Pending;Running;Failed;Updating;Degraded
	// +optional
	Phase string `json:"phase,omitempty"`

	// Conditions represent the latest available observations of the agent's state
	// +optional
	// +listType=map
	// +listMapKey=type
	// +patchMergeKey=type
	// +patchStrategy=merge
	Conditions []metav1.Condition `json:"conditions,omitempty" patchStrategy:"merge" patchMergeKey:"type"`

	// ActiveReplicas is the number of agent pods currently running
	// +optional
	ActiveReplicas int32 `json:"activeReplicas,omitempty"`

	// ReadyReplicas is the number of agent pods ready
	// +optional
	ReadyReplicas int32 `json:"readyReplicas,omitempty"`

	// UUID is a unique identifier for this agent instance
	// Not used for webhook routing; webhooks are routed via agent name (e.g., <agent-name>.domain.com)
	// +optional
	UUID string `json:"uuid,omitempty"`

	// WebhookURLs contains the URLs where this agent can receive webhooks
	// +optional
	WebhookURLs []string `json:"webhookURLs,omitempty"`

	// ObservedGeneration is the most recent generation observed by the controller.
	// It corresponds to the metadata.generation of the LanguageAgent at the time
	// the controller last processed it. Watchers can use this to detect when the
	// status reflects a stale version of the spec.
	// +optional
	ObservedGeneration int64 `json:"observedGeneration,omitempty"`

	// WorkspacePVCName is the name of the retained workspace PVC after agent deletion.
	// Only set when spec.workspace.retain is true.
	// +optional
	WorkspacePVCName string `json:"workspacePVCName,omitempty"`

	// ManagedResources is the inventory of Kubernetes resources created and owned
	// by this controller on behalf of this LanguageAgent.
	// The list is replaced atomically on every successful reconcile.
	// +optional
	// +listType=atomic
	ManagedResources []ManagedResource `json:"managedResources,omitempty"`
}

// +kubebuilder:resource:scope=Namespaced,shortName=lagent
// +kubebuilder:printcolumn:name="Phase",type=string,JSONPath=`.status.phase`
// +kubebuilder:printcolumn:name="Replicas",type=integer,JSONPath=`.status.activeReplicas`
// +kubebuilder:printcolumn:name="Ready",type=integer,JSONPath=`.status.readyReplicas`
// +kubebuilder:printcolumn:name="UUID",type=string,JSONPath=`.status.uuid`
// +kubebuilder:printcolumn:name="Age",type=date,JSONPath=`.metadata.creationTimestamp`

// LanguageAgent is the Schema for the languageagents API
// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
type LanguageAgent struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   LanguageAgentSpec   `json:"spec,omitempty"`
	Status LanguageAgentStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// LanguageAgentList contains a list of LanguageAgent
type LanguageAgentList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []LanguageAgent `json:"items"`
}

func init() {
	SchemeBuilder.Register(&LanguageAgent{}, &LanguageAgentList{})
}
