# API Reference

## Packages
- [langop.io/v1alpha1](#langopiov1alpha1)


## langop.io/v1alpha1

Package v1alpha1 contains API Schema definitions for the language v1alpha1 API group

### Resource Types
- [LanguageAgent](#languageagent)
- [LanguageAgentRuntime](#languageagentruntime)
- [LanguageCluster](#languagecluster)
- [LanguageModel](#languagemodel)
- [LanguagePersona](#languagepersona)
- [LanguageTool](#languagetool)



#### AgentNetworkPolicies



AgentNetworkPolicies defines user-supplied ingress and egress rules for an agent workload.
The shape mirrors the native Kubernetes NetworkPolicySpec so that rules can be copied
verbatim from real NetworkPolicy manifests.



_Appears in:_
- [LanguageAgentSpec](#languageagentspec)
- [LanguageClusterSpec](#languageclusterspec)
- [LanguageToolSpec](#languagetoolspec)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `ingress` _[NetworkIngressRule](#networkingressrule) array_ | Ingress rules — each entry allows traffic into the workload from the listed sources. |  |  |
| `egress` _[NetworkEgressRule](#networkegressrule) array_ | Egress rules — each entry allows traffic from the workload to the listed destinations. |  |  |


#### AgentPort



AgentPort describes a single network port that an agent container exposes.



_Appears in:_
- [LanguageAgentRuntimeSpec](#languageagentruntimespec)
- [LanguageAgentSpec](#languageagentspec)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `name` _string_ | Name uniquely identifies this port within the agent (e.g., "web", "ws").<br />Used as the Service port name; must conform to Kubernetes port-name rules. |  | MaxLength: 15 <br />Pattern: `^[a-z][a-z0-9-]*$` <br />Required: \{\} <br /> |
| `port` _integer_ | Port is the port number the container listens on. |  | Maximum: 65535 <br />Minimum: 1 <br />Required: \{\} <br /> |
| `protocol` _[Protocol](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.28/#protocol-v1-core)_ | Protocol is the transport protocol. Defaults to TCP. | TCP | Enum: [TCP UDP SCTP] <br /> |
| `expose` _boolean_ | Expose controls whether ingress/HTTPRoute routes to this port.<br />At most one port should have expose: true; if none, the first port is used. |  |  |


#### CertIssuerReference



CertIssuerReference references a cert-manager issuer



_Appears in:_
- [IngressTLSConfig](#ingresstlsconfig)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `name` _string_ | Name of the Issuer or ClusterIssuer |  | Required: \{\} <br /> |
| `kind` _string_ | Kind is either "Issuer" or "ClusterIssuer" | ClusterIssuer | Enum: [Issuer ClusterIssuer] <br /> |


#### ClusterCapacitySpec



ClusterCapacitySpec declares hard limits enforced via a ResourceQuota in the cluster's namespace.



_Appears in:_
- [LanguageClusterSpec](#languageclusterspec)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `maxAgents` _integer_ | MaxAgents is the maximum number of LanguageAgent objects allowed. |  |  |
| `maxModels` _integer_ | MaxModels is the maximum number of LanguageModel objects allowed. |  |  |
| `maxTools` _integer_ | MaxTools is the maximum number of LanguageTool objects allowed. |  |  |
| `maxPersonas` _integer_ | MaxPersonas is the maximum number of LanguagePersona objects allowed. |  |  |
| `maxCPU` _[Quantity](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.28/#quantity-resource-api)_ | MaxCPU is the aggregate CPU limit for all pods in the cluster namespace.<br />Maps to limits.cpu in the namespace ResourceQuota.<br />Example: "4", "2500m" |  |  |
| `maxMemory` _[Quantity](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.28/#quantity-resource-api)_ | MaxMemory is the aggregate memory limit for all pods in the cluster namespace.<br />Maps to limits.memory in the namespace ResourceQuota.<br />Example: "8Gi", "512Mi" |  |  |


#### ClusterCapacityStatus



ClusterCapacityStatus reports observed resource usage in the cluster's namespace.



_Appears in:_
- [LanguageClusterStatus](#languageclusterstatus)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `agentCount` _integer_ | AgentCount is the number of LanguageAgent objects in the cluster namespace. |  |  |
| `modelCount` _integer_ | ModelCount is the number of LanguageModel objects in the cluster namespace. |  |  |
| `toolCount` _integer_ | ToolCount is the number of LanguageTool objects in the cluster namespace. |  |  |
| `personaCount` _integer_ | PersonaCount is the number of LanguagePersona objects in the cluster namespace. |  |  |
| `totalCPULimits` _[Quantity](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.28/#quantity-resource-api)_ | TotalCPULimits is the sum of limits.cpu across all agent pod specs. |  |  |
| `totalMemoryLimits` _[Quantity](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.28/#quantity-resource-api)_ | TotalMemoryLimits is the sum of limits.memory across all agent pod specs. |  |  |


#### DeploymentSpec



DeploymentSpec groups Kubernetes deployment configuration that is common across
LanguageAgent, LanguageTool, and LanguageCluster gateway deployments.
All fields are optional; controllers only read the fields relevant to their resource.



_Appears in:_
- [GatewaySpec](#gatewayspec)
- [LanguageAgentRuntimeSpec](#languageagentruntimespec)
- [LanguageAgentSpec](#languageagentspec)
- [LanguageToolSpec](#languagetoolspec)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `replicas` _integer_ | Replicas is the number of pod replicas to run. |  | Minimum: 0 <br /> |
| `imagePullPolicy` _[PullPolicy](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.28/#pullpolicy-v1-core)_ | ImagePullPolicy defines when to pull the container image. |  | Enum: [Always Never IfNotPresent] <br /> |
| `imagePullSecrets` _[LocalObjectReference](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.28/#localobjectreference-v1-core) array_ | ImagePullSecrets is a list of references to secrets for pulling images. |  |  |
| `env` _[EnvVar](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.28/#envvar-v1-core) array_ | Env contains environment variables for the container. |  |  |
| `envFrom` _[EnvFromSource](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.28/#envfromsource-v1-core) array_ | EnvFrom sources to populate environment variables. |  |  |
| `resources` _[ResourceRequirements](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.28/#resourcerequirements-v1-core)_ | Resources defines compute resource requirements. |  |  |
| `nodeSelector` _object (keys:string, values:string)_ | NodeSelector is a selector which must match a node's labels. |  |  |
| `affinity` _[Affinity](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.28/#affinity-v1-core)_ | Affinity defines pod affinity and anti-affinity rules. |  |  |
| `tolerations` _[Toleration](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.28/#toleration-v1-core) array_ | Tolerations allow pods to schedule onto nodes with matching taints. |  |  |
| `topologySpreadConstraints` _[TopologySpreadConstraint](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.28/#topologyspreadconstraint-v1-core) array_ | TopologySpreadConstraints describes how pods should spread across topology domains. |  |  |
| `serviceAccountName` _string_ | ServiceAccountName is the name of the ServiceAccount to use. |  |  |
| `securityContext` _[PodSecurityContext](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.28/#podsecuritycontext-v1-core)_ | SecurityContext holds pod-level security attributes. |  |  |
| `volumeMounts` _[VolumeMount](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.28/#volumemount-v1-core) array_ | VolumeMounts to mount into the container. |  |  |
| `volumes` _[Volume](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.28/#volume-v1-core) array_ | Volumes to attach to the pod. |  |  |
| `podAnnotations` _object (keys:string, values:string)_ | PodAnnotations are annotations to add to the Pods. |  |  |
| `podLabels` _object (keys:string, values:string)_ | PodLabels are additional labels to add to the Pods. |  |  |
| `initContainers` _[Container](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.28/#container-v1-core) array_ | InitContainers are additional init containers injected before the main container starts. |  |  |
| `livenessProbe` _[Probe](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.28/#probe-v1-core)_ | LivenessProbe defines the liveness probe for the container. |  |  |
| `readinessProbe` _[Probe](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.28/#probe-v1-core)_ | ReadinessProbe defines the readiness probe for the container. |  |  |
| `startupProbe` _[Probe](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.28/#probe-v1-core)_ | StartupProbe defines the startup probe for the container. |  |  |
| `command` _string array_ | Command overrides the container entrypoint. |  |  |
| `args` _string array_ | Args overrides the container command arguments. |  |  |
| `serviceType` _[ServiceType](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.28/#servicetype-v1-core)_ | ServiceType specifies the type of Service to create (ClusterIP, NodePort, LoadBalancer). |  | Enum: [ClusterIP NodePort LoadBalancer] <br /> |
| `serviceAnnotations` _object (keys:string, values:string)_ | ServiceAnnotations are annotations to add to the Service. |  |  |


#### GatewaySpec



GatewaySpec configures the shared LiteLLM gateway deployed per LanguageCluster.



_Appears in:_
- [LanguageClusterSpec](#languageclusterspec)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `deployment` _[DeploymentSpec](#deploymentspec)_ | Deployment configures the Kubernetes deployment for the gateway pod. |  |  |


#### IngressConfig



IngressConfig defines ingress configuration



_Appears in:_
- [LanguageClusterSpec](#languageclusterspec)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `enabled` _boolean_ | Enabled controls whether an Ingress is created for the gateway.<br />Defaults to true when cluster.spec.domain is set. |  |  |
| `tls` _[IngressTLSConfig](#ingresstlsconfig)_ | TLS configuration for agent webhooks |  |  |
| `className` _string_ | ClassName specifies the IngressClass to use (maps to spec.ingressClassName on the Ingress object). |  |  |


#### IngressTLSConfig



IngressTLSConfig defines TLS configuration



_Appears in:_
- [IngressConfig](#ingressconfig)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `enabled` _boolean_ | Enabled controls whether TLS is enabled for webhooks | true |  |
| `secretName` _string_ | SecretName is the name of the TLS secret (for manual cert management)<br />If empty, cert-manager will be used if available |  |  |
| `issuerRef` _[CertIssuerReference](#certissuerreference)_ | IssuerRef references a cert-manager Issuer or ClusterIssuer |  |  |


#### LanguageAgent



LanguageAgent is the Schema for the languageagents API





| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `apiVersion` _string_ | `langop.io/v1alpha1` | | |
| `kind` _string_ | `LanguageAgent` | | |
| `metadata` _[ObjectMeta](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.28/#objectmeta-v1-meta)_ | Refer to Kubernetes API documentation for fields of `metadata`. |  |  |
| `spec` _[LanguageAgentSpec](#languageagentspec)_ |  |  |  |
| `status` _[LanguageAgentStatus](#languageagentstatus)_ |  |  |  |


#### LanguageAgentRuntime



LanguageAgentRuntime is the Schema for the languageagentruntimes API.
It defines a reusable preset for LanguageAgent deployments, analogous to an IngressClass.
Admins create runtimes; users reference them via spec.runtime on a LanguageAgent.





| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `apiVersion` _string_ | `langop.io/v1alpha1` | | |
| `kind` _string_ | `LanguageAgentRuntime` | | |
| `metadata` _[ObjectMeta](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.28/#objectmeta-v1-meta)_ | Refer to Kubernetes API documentation for fields of `metadata`. |  |  |
| `spec` _[LanguageAgentRuntimeSpec](#languageagentruntimespec)_ |  |  |  |


#### LanguageAgentRuntimeSpec



LanguageAgentRuntimeSpec defines a preset configuration for LanguageAgent deployments.
All fields are optional; unset fields leave the agent's own spec in effect.
When a LanguageAgent references a runtime, the runtime's fields are merged as defaults:
scalars fill in zeros/nils (agent wins if set), lists are runtime-first then agent-appended.



_Appears in:_
- [LanguageAgentRuntime](#languageagentruntime)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `image` _string_ | Image is the default container image for agents using this runtime.<br />Agents may override this. When a runtime is referenced, spec.image on the agent is optional. |  |  |
| `ports` _[AgentPort](#agentport) array_ | Ports defines default ports for agents using this runtime.<br />Replace semantics: when the agent defines spec.ports, runtime ports are ignored entirely. |  |  |
| `workspace` _[WorkspaceSpec](#workspacespec)_ | Workspace provides default size, storageClass, and mountPath for the agent's workspace.<br />Workspace storage is always provisioned; this presets its parameters.<br />Agents may override individual workspace fields. |  |  |
| `deployment` _[DeploymentSpec](#deploymentspec)_ | Deployment provides default Kubernetes pod and container configuration.<br />Scalars (args, command, resources, probes, etc.) are used when the agent has none set.<br />Lists (initContainers, env, volumes, volumeMounts, envFrom) are runtime-first, agent-appended. |  |  |
| `openclaw` _[OpenclawConfig](#openclawconfig)_ | Openclaw provides default openclaw credential configuration for agents using this runtime.<br />When set, the operator auto-generates OPENCLAW_GATEWAY_TOKEN per agent unless overridden. |  |  |
| `opencode` _[OpencodeConfig](#opencodeconfig)_ | Opencode provides default opencode credential configuration for agents using this runtime.<br />When set, the operator auto-generates OPENCODE_SERVER_PASSWORD per agent unless overridden. |  |  |


#### LanguageAgentSpec



LanguageAgentSpec defines the desired state of LanguageAgent



_Appears in:_
- [LanguageAgent](#languageagent)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `runtime` _string_ | Runtime is the name of a LanguageAgentRuntime that provides preset configuration<br />(image, port, init containers, env vars, probes, etc.).<br />When set, spec.image is optional; the runtime provides a default. |  |  |
| `image` _string_ | Image is the container image to run for this agent.<br />Required unless spec.runtime is set (the runtime provides a default image). |  | Pattern: `^([a-z0-9]+([._-][a-z0-9]+)*\/)*[a-z0-9]+([._-][a-z0-9]+)*(:[a-z0-9]+([._-][a-z0-9]+)*)?$` <br /> |
| `models` _[ModelReference](#modelreference) array_ | Models is a list of LanguageModel references this agent can use |  |  |
| `tools` _[ToolReference](#toolreference) array_ | Tools is a list of LanguageTool references available to this agent |  |  |
| `persona` _string_ | Persona is the name of a LanguagePersona this agent uses |  |  |
| `instructions` _string_ | Instructions provides system instructions for the agent.<br />Delivered as the top-level "instructions" field in /etc/agent/config.yaml. |  |  |
| `workspace` _[WorkspaceSpec](#workspacespec)_ | Workspace defines persistent storage for the agent |  |  |
| `networkPolicies` _[AgentNetworkPolicies](#agentnetworkpolicies)_ | NetworkPolicies defines ingress and egress rules for this agent.<br />Rules mirror the native Kubernetes NetworkPolicy shape. |  |  |
| `ports` _[AgentPort](#agentport) array_ | Ports defines all network ports this agent exposes.<br />At most one entry should have expose: true (the ingress target);<br />if none are marked, the first port is used for ingress routing.<br />Defaults to a single HTTP port on 8080 when not set. |  |  |
| `deployment` _[DeploymentSpec](#deploymentspec)_ | Deployment groups Kubernetes-specific pod and container configuration. |  |  |
| `opencode` _[OpencodeConfig](#opencodeconfig)_ | Opencode holds configuration specific to the opencode runtime.<br />Only effective when spec.runtime is "opencode". |  |  |
| `openclaw` _[OpenclawConfig](#openclawconfig)_ | Openclaw holds configuration specific to the openclaw runtime.<br />Only effective when spec.runtime is "openclaw". |  |  |


#### LanguageAgentStatus



LanguageAgentStatus defines the observed state of LanguageAgent



_Appears in:_
- [LanguageAgent](#languageagent)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `phase` _string_ | Phase represents the current phase of the agent |  | Enum: [Pending Running Failed] <br /> |
| `conditions` _[Condition](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.28/#condition-v1-meta) array_ | Conditions represent the latest available observations of the agent's state |  |  |
| `activeReplicas` _integer_ | ActiveReplicas is the number of agent pods currently running |  |  |
| `readyReplicas` _integer_ | ReadyReplicas is the number of agent pods ready |  |  |
| `uuid` _string_ | UUID is a unique identifier for this agent instance<br />Not used for webhook routing; webhooks are routed via agent name (e.g., <agent-name>.domain.com) |  |  |
| `webhookURLs` _string array_ | WebhookURLs contains the URLs where this agent can receive webhooks |  |  |
| `observedGeneration` _integer_ | ObservedGeneration is the most recent generation observed by the controller.<br />It corresponds to the metadata.generation of the LanguageAgent at the time<br />the controller last processed it. Watchers can use this to detect when the<br />status reflects a stale version of the spec. |  |  |




#### LanguageCluster



LanguageCluster is the Schema for the languageclusters API





| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `apiVersion` _string_ | `langop.io/v1alpha1` | | |
| `kind` _string_ | `LanguageCluster` | | |
| `metadata` _[ObjectMeta](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.28/#objectmeta-v1-meta)_ | Refer to Kubernetes API documentation for fields of `metadata`. |  |  |
| `spec` _[LanguageClusterSpec](#languageclusterspec)_ |  |  |  |
| `status` _[LanguageClusterStatus](#languageclusterstatus)_ |  |  |  |


#### LanguageClusterSpec



LanguageClusterSpec defines the desired state of LanguageCluster



_Appears in:_
- [LanguageCluster](#languagecluster)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `domain` _string_ | Domain is the base domain for the cluster and agent webhook routing<br />The domain itself serves as the cluster dashboard/UI endpoint<br />Agent webhooks will be accessible at <agent-name>.<domain><br />Example: "ai.theryans.io" results in webhooks like "my-agent.ai.theryans.io" |  |  |
| `ingress` _[IngressConfig](#ingressconfig)_ | Ingress defines ingress configuration for the cluster |  |  |
| `networkPolicies` _[AgentNetworkPolicies](#agentnetworkpolicies)_ | NetworkPolicies defines ingress and egress rules for agents in this cluster.<br />Rules mirror the native Kubernetes NetworkPolicy shape. |  |  |
| `gateway` _[GatewaySpec](#gatewayspec)_ | Gateway configures the shared LiteLLM gateway deployed per cluster |  |  |
| `capacity` _[ClusterCapacitySpec](#clustercapacityspec)_ | Capacity declares hard limits enforced via a ResourceQuota in the cluster's namespace.<br />When set, the controller creates a ResourceQuota named "langop-quota".<br />When unset, any existing "langop-quota" is deleted. |  |  |


#### LanguageClusterStatus



LanguageClusterStatus defines the observed state



_Appears in:_
- [LanguageCluster](#languagecluster)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `phase` _string_ | Phase of the cluster |  | Enum: [Pending Ready Failed] <br /> |
| `conditions` _[Condition](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.28/#condition-v1-meta) array_ |  |  |  |
| `gatewayEndpoint` _string_ | GatewayEndpoint is the in-cluster URL for the shared LiteLLM gateway |  |  |
| `gatewayReady` _boolean_ | GatewayReady indicates whether the shared gateway Deployment is available.<br />Pointer distinguishes "not yet reconciled" (nil) from "known not ready" (false). |  |  |
| `capacity` _[ClusterCapacityStatus](#clustercapacitystatus)_ | Capacity reports observed resource usage in this cluster's namespace. |  |  |
| `observedGeneration` _integer_ | ObservedGeneration is the most recent generation observed by the controller.<br />It corresponds to the metadata.generation of the LanguageCluster at the time<br />the controller last processed it. Watchers can use this to detect when the<br />status reflects a stale version of the spec. |  |  |


#### LanguageModel



LanguageModel is the Schema for the languagemodels API





| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `apiVersion` _string_ | `langop.io/v1alpha1` | | |
| `kind` _string_ | `LanguageModel` | | |
| `metadata` _[ObjectMeta](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.28/#objectmeta-v1-meta)_ | Refer to Kubernetes API documentation for fields of `metadata`. |  |  |
| `spec` _[LanguageModelSpec](#languagemodelspec)_ |  |  |  |
| `status` _[LanguageModelStatus](#languagemodelstatus)_ |  |  |  |


#### LanguageModelSpec



LanguageModelSpec defines the desired state of LanguageModel



_Appears in:_
- [LanguageModel](#languagemodel)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `provider` _string_ | Provider specifies the LLM provider type |  | Enum: [openai anthropic openai-compatible azure bedrock vertex custom] <br />Required: \{\} <br /> |
| `modelName` _string_ | ModelName is the specific model identifier (e.g., "gpt-4", "claude-3-opus") |  | MinLength: 1 <br />Required: \{\} <br /> |
| `endpoint` _string_ | Endpoint is the API endpoint URL (required for openai-compatible, azure, custom) |  |  |
| `apiKeySecretRef` _[SecretReference](#secretreference)_ | APIKeySecretRef references a secret containing the API key |  |  |
| `rateLimits` _[RateLimitSpec](#ratelimitspec)_ | RateLimits defines rate limiting configuration |  |  |
| `timeout` _string_ | Timeout specifies request timeout duration (e.g., "5m", "30s") | 5m | Pattern: `^[0-9]+(ns\|us\|µs\|ms\|s\|m\|h)$` <br /> |


#### LanguageModelStatus



LanguageModelStatus defines the observed state of LanguageModel



_Appears in:_
- [LanguageModel](#languagemodel)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `observedGeneration` _integer_ | ObservedGeneration reflects the generation of the most recently observed LanguageModel |  |  |
| `phase` _string_ | Phase represents the current phase of the model |  | Enum: [Ready] <br /> |
| `conditions` _[Condition](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.28/#condition-v1-meta) array_ | Conditions represent the latest available observations of the model's state |  |  |
| `message` _string_ | Message provides human-readable details about the current state |  |  |




#### LanguagePersona



LanguagePersona is the Schema for the languagepersonas API





| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `apiVersion` _string_ | `langop.io/v1alpha1` | | |
| `kind` _string_ | `LanguagePersona` | | |
| `metadata` _[ObjectMeta](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.28/#objectmeta-v1-meta)_ | Refer to Kubernetes API documentation for fields of `metadata`. |  |  |
| `spec` _[LanguagePersonaSpec](#languagepersonaspec)_ |  |  |  |
| `status` _[LanguagePersonaStatus](#languagepersonastatus)_ |  |  |  |


#### LanguagePersonaSpec



LanguagePersonaSpec defines the desired state of LanguagePersona



_Appears in:_
- [LanguagePersona](#languagepersona)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `tone` _string_ | Tone describes the agent's communication style<br />e.g. "professional", "concise and direct", "warm and encouraging" |  |  |
| `personality` _string_ | Personality describes the agent's character and behavioural traits<br />e.g. "curious and methodical, always explains reasoning step by step" |  |  |
| `expertise` _string_ | Expertise describes the agent's domain knowledge and skills<br />e.g. "senior software engineer specialising in distributed systems and Go" |  |  |


#### LanguagePersonaStatus



LanguagePersonaStatus defines the observed state of LanguagePersona



_Appears in:_
- [LanguagePersona](#languagepersona)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `observedGeneration` _integer_ | ObservedGeneration reflects the generation of the most recently observed LanguagePersona |  |  |
| `phase` _string_ | Phase represents the current phase |  | Enum: [Ready] <br /> |
| `conditions` _[Condition](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.28/#condition-v1-meta) array_ | Conditions represent the latest available observations of the persona's state |  |  |




#### LanguageTool



LanguageTool is the Schema for the languagetools API





| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `apiVersion` _string_ | `langop.io/v1alpha1` | | |
| `kind` _string_ | `LanguageTool` | | |
| `metadata` _[ObjectMeta](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.28/#objectmeta-v1-meta)_ | Refer to Kubernetes API documentation for fields of `metadata`. |  |  |
| `spec` _[LanguageToolSpec](#languagetoolspec)_ |  |  |  |
| `status` _[LanguageToolStatus](#languagetoolstatus)_ |  |  |  |


#### LanguageToolSpec



LanguageToolSpec defines the desired state of LanguageTool



_Appears in:_
- [LanguageTool](#languagetool)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `image` _string_ | Image is the container image to run for this tool |  | MinLength: 1 <br />Required: \{\} <br /> |
| `type` _string_ | Type specifies the tool protocol type. Only "mcp" is currently implemented. | mcp | Enum: [mcp] <br /> |
| `deploymentMode` _string_ | DeploymentMode specifies how this tool should be deployed<br />- "service": Deployed as a standalone Deployment+Service (default, shared across agents)<br />- "sidecar": Deployed as a sidecar container in each agent pod (dedicated, with workspace access) | service | Enum: [service sidecar] <br /> |
| `port` _integer_ | Port is the port the tool listens on | 8080 | Maximum: 65535 <br />Minimum: 1 <br /> |
| `deployment` _[DeploymentSpec](#deploymentspec)_ | Deployment groups Kubernetes-specific pod and container configuration. |  |  |
| `networkPolicies` _[AgentNetworkPolicies](#agentnetworkpolicies)_ | NetworkPolicies defines ingress and egress rules for this tool.<br />Rules mirror the native Kubernetes NetworkPolicy shape. |  |  |


#### LanguageToolStatus



LanguageToolStatus defines the observed state of LanguageTool



_Appears in:_
- [LanguageTool](#languagetool)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `observedGeneration` _integer_ | ObservedGeneration reflects the generation of the most recently observed LanguageTool |  |  |
| `phase` _string_ | Phase represents the current phase of the tool (Pending, Running, Failed, Updating) |  | Enum: [Pending Running Failed Updating] <br /> |
| `conditions` _[Condition](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.28/#condition-v1-meta) array_ | Conditions represent the latest available observations of the tool's state |  |  |
| `endpoint` _string_ | Endpoint is the service endpoint where the tool is accessible |  |  |
| `toolSchemas` _[ToolSchema](#toolschema) array_ | ToolSchemas contains the complete MCP tool schemas discovered from this service |  |  |
| `readyReplicas` _integer_ | ReadyReplicas is the number of pods ready and passing health checks |  |  |
| `availableReplicas` _integer_ | AvailableReplicas is the number of pods targeted by this LanguageTool with at least one available condition |  |  |
| `updatedReplicas` _integer_ | UpdatedReplicas is the number of pods targeted by this LanguageTool that have the desired spec |  |  |
| `unavailableReplicas` _integer_ | UnavailableReplicas is the number of pods targeted by this LanguageTool that are unavailable |  |  |




#### ModelReference



ModelReference references a LanguageModel



_Appears in:_
- [LanguageAgentSpec](#languageagentspec)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `name` _string_ | Name is the name of the LanguageModel |  | MaxLength: 63 <br />Pattern: `^[a-z0-9]([-a-z0-9]*[a-z0-9])?$` <br />Required: \{\} <br /> |
| `role` _string_ | Role defines the purpose of this model — a hint for the agent runtime for model selection<br />(e.g. prefer role=primary for general calls, role=reasoning for chain-of-thought).<br />The operator does not enforce routing by role; it is surfaced in the agent config (agent.json). | primary | Enum: [primary fallback reasoning tool-calling summarization] <br /> |
| `priority` _integer_ | Priority for model selection — a hint for the agent runtime (lower value = higher priority).<br />The operator does not enforce priority; it is surfaced in the agent config (agent.json). |  |  |


#### NetworkEgressRule



NetworkEgressRule is one egress rule: zero or more destination peers, zero or more ports.
Mirrors networkingv1.NetworkPolicyEgressRule.



_Appears in:_
- [AgentNetworkPolicies](#agentnetworkpolicies)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `to` _[NetworkPeer](#networkpeer) array_ | To lists the destinations this workload is allowed to reach. |  |  |
| `ports` _[NetworkPort](#networkport) array_ | Ports lists the destination ports to allow. |  |  |


#### NetworkIngressRule



NetworkIngressRule is one ingress rule: zero or more source peers, zero or more ports.
Mirrors networkingv1.NetworkPolicyIngressRule.



_Appears in:_
- [AgentNetworkPolicies](#agentnetworkpolicies)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `from` _[NetworkPeer](#networkpeer) array_ | From lists the sources allowed to send traffic to this workload. |  |  |
| `ports` _[NetworkPort](#networkport) array_ | Ports lists the ports on which to allow incoming traffic. |  |  |


#### NetworkPeer



NetworkPeer defines the source/destination of network traffic



_Appears in:_
- [NetworkEgressRule](#networkegressrule)
- [NetworkIngressRule](#networkingressrule)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `group` _string_ | Group selects pods with matching langop.io/group label<br />Used to allow communication with specific labeled resources |  |  |
| `cidr` _string_ | CIDR block |  |  |
| `dns` _string array_ | DNS names (supports wildcards with *)<br />Examples: "api.openai.com", "*.googleapis.com" |  |  |
| `service` _[ServiceReference](#servicereference)_ | Kubernetes service reference |  |  |
| `namespaceSelector` _[LabelSelector](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.28/#labelselector-v1-meta)_ | Namespace selector (for cross-namespace rules) |  |  |
| `podSelector` _[LabelSelector](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.28/#labelselector-v1-meta)_ | Pod selector (within namespace) |  |  |


#### NetworkPort



NetworkPort defines a port and protocol



_Appears in:_
- [NetworkEgressRule](#networkegressrule)
- [NetworkIngressRule](#networkingressrule)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `protocol` _string_ | Protocol (TCP, UDP, SCTP) | TCP | Enum: [TCP UDP SCTP] <br /> |
| `port` _integer_ | Port number |  | Maximum: 65535 <br />Minimum: 1 <br /> |


#### OpenclawConfig



OpenclawConfig holds configuration specific to the openclaw runtime.
Effective only when spec.runtime is "openclaw".



_Appears in:_
- [LanguageAgentRuntimeSpec](#languageagentruntimespec)
- [LanguageAgentSpec](#languageagentspec)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `enabled` _boolean_ | Enabled activates openclaw credential management for this agent.<br />Set to true in a LanguageAgentRuntime to trigger auto-generation of OPENCLAW_GATEWAY_TOKEN<br />without requiring any explicit config on the LanguageAgent. |  |  |
| `token` _string_ | Token is the gateway authentication token (inline).<br />The operator creates a managed Secret and injects it via envFrom.<br />Mutually exclusive with TokenRef. |  |  |
| `tokenRef` _[RuntimeSecretRef](#runtimesecretref)_ | TokenRef references a Secret whose keys are injected via envFrom.<br />The Secret must contain OPENCLAW_GATEWAY_TOKEN.<br />Mutually exclusive with Token. |  |  |


#### OpencodeConfig



OpencodeConfig holds configuration specific to the opencode runtime.
Effective only when spec.runtime is "opencode".



_Appears in:_
- [LanguageAgentRuntimeSpec](#languageagentruntimespec)
- [LanguageAgentSpec](#languageagentspec)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `enabled` _boolean_ | Enabled activates opencode credential management for this agent.<br />Set to true in a LanguageAgentRuntime to trigger auto-generation of credentials<br />without requiring any explicit config on the LanguageAgent. |  |  |
| `username` _string_ | Username for HTTP Basic Auth. Defaults to "opencode" if not set.<br />Sets OPENCODE_SERVER_USERNAME in the agent container. |  |  |
| `password` _string_ | Password is the HTTP Basic Auth password (inline).<br />The operator creates a managed Secret and injects it via envFrom.<br />Mutually exclusive with PasswordRef. |  |  |
| `passwordRef` _[RuntimeSecretRef](#runtimesecretref)_ | PasswordRef references a Secret whose keys are injected via envFrom.<br />The Secret must contain OPENCODE_SERVER_PASSWORD (and optionally OPENCODE_SERVER_USERNAME).<br />Mutually exclusive with Password. |  |  |


#### RateLimitSpec



RateLimitSpec defines rate limiting configuration



_Appears in:_
- [LanguageModelSpec](#languagemodelspec)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `requestsPerMinute` _integer_ | RequestsPerMinute limits requests per minute |  |  |
| `tokensPerMinute` _integer_ | TokensPerMinute limits tokens per minute |  |  |


#### RuntimeSecretRef



RuntimeSecretRef references a Secret in the same namespace.
All keys in the Secret are injected as env vars via envFrom.



_Appears in:_
- [OpenclawConfig](#openclawconfig)
- [OpencodeConfig](#opencodeconfig)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `name` _string_ | Name is the name of the Secret. |  | Required: \{\} <br /> |


#### SecretReference



SecretReference references a Kubernetes Secret



_Appears in:_
- [LanguageModelSpec](#languagemodelspec)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `name` _string_ | Name is the name of the secret |  | Required: \{\} <br /> |
| `namespace` _string_ | Namespace is the namespace of the secret (defaults to same namespace as LanguageModel) |  |  |
| `key` _string_ | Key is the key within the secret containing the value | api-key |  |


#### ServiceReference



ServiceReference identifies a Kubernetes Service



_Appears in:_
- [NetworkPeer](#networkpeer)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `name` _string_ | Service name |  | Required: \{\} <br /> |
| `namespace` _string_ | Service namespace (defaults to same namespace if omitted) |  |  |


#### ToolProperty



ToolProperty defines an individual parameter or return field



_Appears in:_
- [ToolSchemaDefinition](#toolschemadefinition)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `type` _string_ | Type is the JSON schema type (string, integer, boolean, etc.) |  |  |
| `description` _string_ | Description explains what this property represents |  |  |
| `example` _string_ | Example provides an example value as a JSON string |  |  |


#### ToolReference



ToolReference references a LanguageTool



_Appears in:_
- [LanguageAgentSpec](#languageagentspec)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `name` _string_ | Name is the name of the LanguageTool |  | MaxLength: 63 <br />Pattern: `^[a-z0-9]([-a-z0-9]*[a-z0-9])?$` <br />Required: \{\} <br /> |
| `enabled` _boolean_ | Enabled indicates if this tool is available to the agent.<br />Defaults to true. Set to false to explicitly disable the tool without removing it. | true |  |


#### ToolSchema



ToolSchema represents the complete schema of an MCP tool



_Appears in:_
- [LanguageToolStatus](#languagetoolstatus)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `name` _string_ | Name is the tool identifier |  |  |
| `description` _string_ | Description is a human-readable description of the tool |  |  |
| `inputSchema` _[ToolSchemaDefinition](#toolschemadefinition)_ | InputSchema defines the parameters this tool accepts |  |  |


#### ToolSchemaDefinition



ToolSchemaDefinition defines parameter or return value structure



_Appears in:_
- [ToolSchema](#toolschema)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `type` _string_ | Type is the JSON schema type (object, array, string, etc.) |  |  |
| `properties` _object (keys:string, values:[ToolProperty](#toolproperty))_ | Properties defines object properties (for type: object) |  |  |
| `required` _string array_ | Required lists required property names (for type: object) |  |  |


#### WorkspaceSpec



WorkspaceSpec defines persistent workspace storage for an agent



_Appears in:_
- [LanguageAgentRuntimeSpec](#languageagentruntimespec)
- [LanguageAgentSpec](#languageagentspec)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `enabled` _boolean_ | Enabled controls whether to create a workspace volume.<br />Defaults to true. Set to false to explicitly disable without removing the workspace config. | true |  |
| `size` _string_ | Size is the requested storage size (e.g., "10Gi", "1.5Ti", "500Mi")<br />Supports integer and decimal quantities with standard Kubernetes suffixes | 10Gi | MinLength: 1 <br />Pattern: `^([0-9]*\.?[0-9]+)(Ei\|Pi\|Ti\|Gi\|Mi\|Ki\|E\|P\|T\|G\|M\|K\|m)?$` <br /> |
| `storageClassName` _string_ | StorageClassName specifies the StorageClass for the PVC<br />If not specified, uses the cluster default |  |  |
| `accessMode` _string_ | AccessMode defines the volume access mode | ReadWriteOnce | Enum: [ReadWriteOnce ReadWriteMany] <br /> |
| `mountPath` _string_ | MountPath is where the workspace is mounted in containers | /workspace |  |


