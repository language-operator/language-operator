package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// ToolSchema represents the complete schema of an MCP tool
type ToolSchema struct {
	// Name is the tool identifier
	Name string `json:"name"`

	// Description is a human-readable description of the tool
	// +optional
	Description string `json:"description,omitempty"`

	// InputSchema defines the parameters this tool accepts
	// +optional
	InputSchema *ToolSchemaDefinition `json:"inputSchema,omitempty"`
}

// ToolSchemaDefinition defines parameter or return value structure
type ToolSchemaDefinition struct {
	// Type is the JSON schema type (object, array, string, etc.)
	// +optional
	Type string `json:"type,omitempty"`

	// Properties defines object properties (for type: object)
	// +optional
	Properties map[string]ToolProperty `json:"properties,omitempty"`

	// Required lists required property names (for type: object)
	// +optional
	Required []string `json:"required,omitempty"`
}

// ToolProperty defines an individual parameter or return field
type ToolProperty struct {
	// Type is the JSON schema type (string, integer, boolean, etc.)
	Type string `json:"type"`

	// Description explains what this property represents
	// +optional
	Description string `json:"description,omitempty"`

	// Example provides an example value as a JSON string
	// +optional
	Example string `json:"example,omitempty"`
}

// DefaultMCPBridgeImage is the stdio→Streamable-HTTP bridge image the operator injects for
// transport=stdio tools. It is the first-party mcp-bridge image (built from
// components/mcp-bridge/) which bundles supergateway plus Node and Python+uv, so both npx and
// uvx stdio servers work. The operator's --mcp-bridge-image flag overrides what is actually
// deployed; this constant is also used by the defaulting webhook to satisfy the required Image
// field when a stdio tool omits it (the controller ignores spec.image for stdio tools).
const DefaultMCPBridgeImage = "ghcr.io/language-operator/mcp-bridge:latest"

// LanguageToolSpec defines the desired state of LanguageTool
type LanguageToolSpec struct {
	// Image is the container image to run for this tool. For transport=stdio it is ignored —
	// the operator injects the MCP bridge image instead — and may be omitted (the defaulting
	// webhook fills it).
	// +kubebuilder:validation:Required
	// +kubebuilder:validation:MinLength=1
	Image string `json:"image"`

	// Type specifies the tool protocol type. Only "mcp" is currently implemented.
	// +kubebuilder:validation:Enum=mcp
	// +kubebuilder:default=mcp
	Type string `json:"type,omitempty"`

	// Transport selects how the operator exposes this tool's MCP endpoint.
	// - "streamable-http" (default): spec.image already serves Streamable HTTP at /mcp.
	// - "sse": spec.image already serves the (legacy) MCP HTTP+SSE transport.
	// - "stdio": the user supplies a stdio MCP command in spec.stdio; the operator injects a
	//   pinned, persistent stdio→Streamable-HTTP bridge that serves /mcp and /health on spec.port.
	// +kubebuilder:validation:Enum=streamable-http;sse;stdio
	// +kubebuilder:default=streamable-http
	// +optional
	Transport string `json:"transport,omitempty"`

	// Stdio configures the stdio MCP server when transport=stdio. Required for that transport,
	// ignored otherwise.
	// +optional
	Stdio *StdioServerSpec `json:"stdio,omitempty"`

	// DeploymentMode specifies how this tool should be deployed
	// - "service": Deployed as a standalone Deployment+Service (default, shared across agents)
	// - "sidecar": Deployed as a sidecar container in each agent pod (dedicated, with workspace access)
	// +kubebuilder:validation:Enum=service;sidecar
	// +kubebuilder:default=service
	// +optional
	DeploymentMode string `json:"deploymentMode,omitempty"`

	// Port is the port the tool listens on
	// +kubebuilder:validation:Minimum=1
	// +kubebuilder:validation:Maximum=65535
	// +kubebuilder:default=8080
	Port int32 `json:"port,omitempty"`

	// Deployment groups Kubernetes-specific pod and container configuration.
	// +optional
	Deployment DeploymentSpec `json:"deployment,omitempty"`

	// NetworkPolicies defines ingress and egress rules for this tool.
	// Rules mirror the native Kubernetes NetworkPolicy shape.
	// +optional
	NetworkPolicies *AgentNetworkPolicies `json:"networkPolicies,omitempty"`
}

// StdioServerSpec describes a stdio-based MCP server the operator wraps with a persistent
// stdio→Streamable-HTTP bridge.
type StdioServerSpec struct {
	// Command is the full argv of the stdio MCP server, e.g.
	// ["npx","-y","@upstash/context7-mcp"] or ["uvx","mcp-server-git","--repository","/workspace"].
	// The operator passes it to the bridge as a single stdio command. Environment for the
	// child comes from spec.deployment.env / spec.deployment.envFrom.
	// +kubebuilder:validation:Required
	// +kubebuilder:validation:MinItems=1
	Command []string `json:"command"`
}

// LanguageToolStatus defines the observed state of LanguageTool
type LanguageToolStatus struct {
	// ObservedGeneration reflects the generation of the most recently observed LanguageTool
	// +optional
	ObservedGeneration int64 `json:"observedGeneration,omitempty"`

	// Phase represents the current phase of the tool (Pending, Running, Failed, Updating, Degraded)
	// +kubebuilder:validation:Enum=Pending;Running;Failed;Updating;Degraded
	// +optional
	Phase string `json:"phase,omitempty"`

	// Conditions represent the latest available observations of the tool's state
	// +optional
	// +listType=map
	// +listMapKey=type
	// +patchMergeKey=type
	// +patchStrategy=merge
	Conditions []metav1.Condition `json:"conditions,omitempty" patchStrategy:"merge" patchMergeKey:"type"`

	// Endpoint is the service endpoint where the tool is accessible
	// +optional
	Endpoint string `json:"endpoint,omitempty"`

	// ToolSchemas contains the complete MCP tool schemas discovered from this service
	// +optional
	ToolSchemas []ToolSchema `json:"toolSchemas,omitempty"`

	// ReadyReplicas is the number of pods ready and passing health checks
	// +optional
	ReadyReplicas int32 `json:"readyReplicas,omitempty"`

	// AvailableReplicas is the number of pods targeted by this LanguageTool with at least one available condition
	// +optional
	AvailableReplicas int32 `json:"availableReplicas,omitempty"`

	// UpdatedReplicas is the number of pods targeted by this LanguageTool that have the desired spec
	// +optional
	UpdatedReplicas int32 `json:"updatedReplicas,omitempty"`

	// UnavailableReplicas is the number of pods targeted by this LanguageTool that are unavailable
	// +optional
	UnavailableReplicas int32 `json:"unavailableReplicas,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:resource:scope=Namespaced,shortName=ltool
// +kubebuilder:printcolumn:name="Type",type=string,JSONPath=`.spec.type`
// +kubebuilder:printcolumn:name="Image",type=string,JSONPath=`.spec.image`
// +kubebuilder:printcolumn:name="Phase",type=string,JSONPath=`.status.phase`
// +kubebuilder:printcolumn:name="Endpoint",type=string,JSONPath=`.status.endpoint`
// +kubebuilder:printcolumn:name="Age",type=date,JSONPath=`.metadata.creationTimestamp`

// LanguageTool is the Schema for the languagetools API
type LanguageTool struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   LanguageToolSpec   `json:"spec,omitempty"`
	Status LanguageToolStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// LanguageToolList contains a list of LanguageTool
type LanguageToolList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []LanguageTool `json:"items"`
}

func init() {
	SchemeBuilder.Register(&LanguageTool{}, &LanguageToolList{})
}
