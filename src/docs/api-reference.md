# API Reference

## Packages
- [langop.io/v1alpha1](#langopiov1alpha1)


## langop.io/v1alpha1

Package v1alpha1 contains API Schema definitions for the language v1alpha1 API group

### Resource Types
- [LanguageAgent](#languageagent)
- [LanguageCluster](#languagecluster)
- [LanguageModel](#languagemodel)
- [LanguagePersona](#languagepersona)
- [LanguageTool](#languagetool)



#### AgentCostMetrics



AgentCostMetrics contains agent cost tracking



_Appears in:_
- [LanguageAgentStatus](#languageagentstatus)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `totalCost` _float_ | TotalCost is the total cost incurred by this agent |  |  |
| `modelCosts` _[ModelCostSpec](#modelcostspec) array_ | ModelCosts breaks down cost by model |  |  |
| `currency` _string_ | Currency is the currency for cost metrics |  |  |
| `lastReset` _[Time](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.28/#time-v1-meta)_ | LastReset is when cost metrics were last reset |  |  |


#### AgentMetrics



AgentMetrics contains agent execution metrics



_Appears in:_
- [LanguageAgentStatus](#languageagentstatus)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `averageIterations` _float_ | AverageIterations is the average number of iterations per execution |  |  |
| `averageExecutionTime` _float_ | AverageExecutionTime is the average execution time in seconds |  |  |
| `totalTokens` _integer_ | TotalTokens is the total number of tokens consumed |  |  |
| `totalToolCalls` _integer_ | TotalToolCalls is the total number of tool invocations |  |  |
| `successRate` _float_ | SuccessRate is the percentage of successful executions |  |  |


#### CachingSpec



CachingSpec defines response caching configuration



_Appears in:_
- [LanguageModelSpec](#languagemodelspec)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `enabled` _boolean_ | Enabled enables response caching | false |  |
| `ttl` _string_ | TTL is the cache time-to-live (e.g., "5m") |  | Pattern: `^[0-9]+(ns\|us\|µs\|ms\|s\|m\|h)$` <br /> |
| `maxSize` _integer_ | MaxSize is the maximum cache size in MB |  |  |
| `backend` _string_ | Backend specifies the caching backend (memory, redis, etc.) | memory | Enum: [memory redis memcached] <br /> |


#### CertIssuerReference



CertIssuerReference references a cert-manager issuer



_Appears in:_
- [IngressTLSConfig](#ingresstlsconfig)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `name` _string_ | Name of the Issuer or ClusterIssuer |  | Required: \{\} <br /> |
| `kind` _string_ | Kind is either "Issuer" or "ClusterIssuer" | ClusterIssuer | Enum: [Issuer ClusterIssuer] <br /> |
| `group` _string_ | Group is the API group of the issuer | cert-manager.io |  |


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


#### CostMetrics



CostMetrics contains cost tracking data



_Appears in:_
- [LanguageModelStatus](#languagemodelstatus)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `totalCost` _float_ | TotalCost is the total cost incurred |  |  |
| `inputTokenCost` _float_ | InputTokenCost is the cost for input tokens |  |  |
| `outputTokenCost` _float_ | OutputTokenCost is the cost for output tokens |  |  |
| `currency` _string_ | Currency is the currency for cost metrics |  |  |
| `lastReset` _[Time](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.28/#time-v1-meta)_ | LastReset is when cost metrics were last reset |  |  |


#### CostTrackingSpec



CostTrackingSpec defines cost tracking configuration



_Appears in:_
- [LanguageModelSpec](#languagemodelspec)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `enabled` _boolean_ | Enabled enables cost tracking | false |  |
| `currency` _string_ | Currency is the currency for cost tracking (e.g., "USD") | USD |  |
| `inputTokenCost` _float_ | InputTokenCost is the cost per 1000 input tokens |  |  |
| `outputTokenCost` _float_ | OutputTokenCost is the cost per 1000 output tokens |  |  |


#### EndpointSpec



EndpointSpec defines an endpoint for load balancing



_Appears in:_
- [LoadBalancingSpec](#loadbalancingspec)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `url` _string_ | URL is the endpoint URL |  | Required: \{\} <br /> |
| `weight` _integer_ | Weight for weighted load balancing | 100 | Minimum: 0 <br /> |
| `region` _string_ | Region specifies the region for this endpoint |  |  |
| `priority` _integer_ | Priority for priority-based routing (lower is higher priority) |  |  |


#### EndpointStatusSpec



EndpointStatusSpec shows the status of a load-balanced endpoint



_Appears in:_
- [LanguageModelStatus](#languagemodelstatus)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `url` _string_ | URL is the endpoint URL |  |  |
| `healthy` _boolean_ | Healthy indicates if the endpoint is healthy |  |  |
| `lastCheck` _[Time](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.28/#time-v1-meta)_ | LastCheck is the timestamp of the last health check |  |  |
| `failureCount` _integer_ | FailureCount is the number of consecutive failures |  |  |
| `latency` _integer_ | Latency is the average latency in milliseconds |  |  |


#### EventTriggerSpec



EventTriggerSpec defines an event trigger



_Appears in:_
- [LanguageAgentSpec](#languageagentspec)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `type` _string_ | Type is the event type (webhook, kubernetes-event, message-queue) |  | Enum: [webhook kubernetes-event message-queue schedule] <br />Required: \{\} <br /> |
| `source` _string_ | Source identifies the event source |  |  |
| `filter` _object (keys:string, values:string)_ | Filter defines filtering criteria for events |  |  |


#### HealthCheckSpec



HealthCheckSpec defines health checking configuration



_Appears in:_
- [LoadBalancingSpec](#loadbalancingspec)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `enabled` _boolean_ | Enabled enables health checking | true |  |
| `interval` _string_ | Interval is the health check interval (e.g., "30s") | 30s | Pattern: `^[0-9]+(ns\|us\|µs\|ms\|s\|m\|h)$` <br /> |
| `timeout` _string_ | Timeout is the health check timeout (e.g., "5s") | 5s | Pattern: `^[0-9]+(ns\|us\|µs\|ms\|s\|m\|h)$` <br /> |
| `unhealthyThreshold` _integer_ | UnhealthyThreshold is the number of failures before marking unhealthy | 3 | Minimum: 1 <br /> |
| `healthyThreshold` _integer_ | HealthyThreshold is the number of successes before marking healthy | 2 | Minimum: 1 <br /> |


#### IngressConfig



IngressConfig defines ingress/gateway configuration



_Appears in:_
- [LanguageClusterSpec](#languageclusterspec)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `tls` _[IngressTLSConfig](#ingresstlsconfig)_ | TLS configuration for agent webhooks |  |  |
| `gatewayName` _string_ | GatewayName specifies the Gateway resource name to use<br />If empty, will attempt auto-detection or fall back to Ingress |  |  |
| `gatewayNamespace` _string_ | GatewayNamespace specifies the namespace of the Gateway resource<br />If empty, defaults to the same namespace as the LanguageCluster |  |  |
| `gatewayClassName` _string_ | Deprecated: Use GatewayName instead. This field actually refers to a Gateway resource name, not a GatewayClass.<br />GatewayClassName specifies the Gateway API GatewayClass to use<br />If empty, will attempt auto-detection or fall back to Ingress |  |  |
| `ingressClassName` _string_ | IngressClassName specifies the Ingress class to use for fallback<br />Only used when Gateway API is not available |  |  |


#### IngressTLSConfig



IngressTLSConfig defines TLS configuration



_Appears in:_
- [IngressConfig](#ingressconfig)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `enabled` _boolean_ | Enabled controls whether TLS is enabled for webhooks | true |  |
| `secretName` _string_ | SecretName is the name of the TLS secret (for manual cert management)<br />If empty, cert-manager will be used if available |  |  |
| `issuerRef` _[CertIssuerReference](#certissuerreference)_ | IssuerRef references a cert-manager Issuer or ClusterIssuer |  |  |


#### KnowledgeSourceSpec



KnowledgeSourceSpec references an external knowledge base



_Appears in:_
- [LanguagePersonaSpec](#languagepersonaspec)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `name` _string_ | Name is the knowledge source identifier |  | Required: \{\} <br /> |
| `type` _string_ | Type specifies the knowledge source type |  | Enum: [url document database api vector-store] <br />Required: \{\} <br /> |
| `url` _string_ | URL is the knowledge source URL (for url, api types) |  |  |
| `secretRef` _[SecretReference](#secretreference)_ | SecretRef references credentials for accessing the knowledge source |  |  |
| `query` _string_ | Query defines how to query this knowledge source |  |  |
| `priority` _integer_ | Priority determines knowledge source precedence |  |  |
| `enabled` _boolean_ | Enabled indicates if this knowledge source is active | true |  |


#### LanguageAgent



LanguageAgent is the Schema for the languageagents API





| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `apiVersion` _string_ | `langop.io/v1alpha1` | | |
| `kind` _string_ | `LanguageAgent` | | |
| `metadata` _[ObjectMeta](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.28/#objectmeta-v1-meta)_ | Refer to Kubernetes API documentation for fields of `metadata`. |  |  |
| `spec` _[LanguageAgentSpec](#languageagentspec)_ |  |  |  |
| `status` _[LanguageAgentStatus](#languageagentstatus)_ |  |  |  |


#### LanguageAgentSpec



LanguageAgentSpec defines the desired state of LanguageAgent



_Appears in:_
- [LanguageAgent](#languageagent)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `image` _string_ | Image is the container image to run for this agent |  | MinLength: 1 <br />Pattern: `^([a-z0-9]+([._-][a-z0-9]+)*\/)*[a-z0-9]+([._-][a-z0-9]+)*(:[a-z0-9]+([._-][a-z0-9]+)*)?$` <br />Required: \{\} <br /> |
| `imagePullPolicy` _[PullPolicy](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.28/#pullpolicy-v1-core)_ | ImagePullPolicy defines when to pull the container image | IfNotPresent | Enum: [Always Never IfNotPresent] <br /> |
| `imagePullSecrets` _[LocalObjectReference](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.28/#localobjectreference-v1-core) array_ | ImagePullSecrets is a list of references to secrets for pulling images |  |  |
| `models` _[ModelReference](#modelreference) array_ | Models is a list of LanguageModel references this agent can use |  |  |
| `tools` _[ToolReference](#toolreference) array_ | Tools is a list of LanguageTool references available to this agent |  |  |
| `personas` _[PersonaReference](#personareference) array_ | Personas is a list of LanguagePersona references that compose in order of importance<br />Personas are merged with later personas taking precedence over earlier ones |  |  |
| `instructions` _string_ | Instructions provides system instructions for the agent.<br />Mounted at /etc/agent/instructions.txt if set. |  |  |
| `executionMode` _string_ | ExecutionMode defines how the agent operates | autonomous | Enum: [autonomous interactive scheduled event-driven] <br /> |
| `schedule` _string_ | Schedule defines when the agent runs (cron format, for scheduled mode)<br />Must be a valid cron expression (5 fields: minute hour day month weekday) or special syntax (@hourly, @daily, etc.) |  | MaxLength: 100 <br />Pattern: `^(@(annually\|yearly\|monthly\|weekly\|daily\|hourly\|every_minute))\|(@every\s+((\d+(\.\d+)?)(ns\|us\|µs\|ms\|s\|m\|h))+)\|(((\*\|[0-9]\|[1-5][0-9]\|\*\/[0-9]+)\s+)\{4\}(\*\|[0-7]\|[1-7]\|\*\/[0-9]+))$` <br /> |
| `eventTriggers` _[EventTriggerSpec](#eventtriggerspec) array_ | EventTriggers defines events that trigger the agent (for event-driven mode) |  |  |
| `timeout` _string_ | Timeout is the maximum execution time (e.g., "10m", "1h") | 10m | Pattern: `^[0-9]+(ns\|us\|µs\|ms\|s\|m\|h)$` <br /> |
| `replicas` _integer_ | Replicas is the number of agent instances to run | 1 | Minimum: 0 <br /> |
| `env` _[EnvVar](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.28/#envvar-v1-core) array_ | Env contains environment variables for the agent container |  |  |
| `envFrom` _[EnvFromSource](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.28/#envfromsource-v1-core) array_ | EnvFrom sources to populate environment variables |  |  |
| `resources` _[ResourceRequirements](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.28/#resourcerequirements-v1-core)_ | Resources defines compute resource requirements |  |  |
| `nodeSelector` _object (keys:string, values:string)_ | NodeSelector is a selector which must match a node's labels |  |  |
| `affinity` _[Affinity](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.28/#affinity-v1-core)_ | Affinity defines pod affinity and anti-affinity rules |  |  |
| `tolerations` _[Toleration](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.28/#toleration-v1-core) array_ | Tolerations allow pods to schedule onto nodes with matching taints |  |  |
| `serviceAccountName` _string_ | ServiceAccountName is the name of the ServiceAccount to use |  |  |
| `securityContext` _[PodSecurityContext](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.28/#podsecuritycontext-v1-core)_ | SecurityContext holds pod-level security attributes |  |  |
| `volumeMounts` _[VolumeMount](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.28/#volumemount-v1-core) array_ | VolumeMounts to mount into the agent container |  |  |
| `volumes` _[Volume](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.28/#volume-v1-core) array_ | Volumes to attach to the pod |  |  |
| `podAnnotations` _object (keys:string, values:string)_ | PodAnnotations are annotations to add to the Pods |  |  |
| `podLabels` _object (keys:string, values:string)_ | PodLabels are additional labels to add to the Pods |  |  |
| `restartPolicy` _[RestartPolicy](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.28/#restartpolicy-v1-core)_ | RestartPolicy defines when to restart the agent | OnFailure | Enum: [Always OnFailure Never] <br /> |
| `backoffLimit` _integer_ | BackoffLimit specifies the number of retries before marking as Failed | 3 | Minimum: 0 <br /> |
| `workspace` _[WorkspaceSpec](#workspacespec)_ | Workspace defines persistent storage for the agent |  |  |
| `egress` _[NetworkRule](#networkrule) array_ | Egress defines external network access rules for this agent<br />By default, agents can access all resources within the cluster but no external endpoints |  |  |
| `port` _integer_ | Port is the port the agent container listens on.<br />Used for the ClusterIP Service and NetworkPolicy ingress rules.<br />Defaults to 8080. |  | Maximum: 65535 <br />Minimum: 1 <br /> |
| `livenessProbe` _[Probe](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.28/#probe-v1-core)_ | LivenessProbe defines the liveness probe for the agent container.<br />If not set, no liveness probe is configured. |  |  |
| `readinessProbe` _[Probe](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.28/#probe-v1-core)_ | ReadinessProbe defines the readiness probe for the agent container.<br />If not set, no readiness probe is configured. |  |  |
| `initContainers` _[Container](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.28/#container-v1-core) array_ | InitContainers are additional init containers injected before the agent container starts.<br />Useful for seeding config, migrating workspace data, or other pre-start setup. |  |  |


#### LanguageAgentStatus



LanguageAgentStatus defines the observed state of LanguageAgent



_Appears in:_
- [LanguageAgent](#languageagent)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `observedGeneration` _integer_ | ObservedGeneration reflects the generation of the most recently observed LanguageAgent |  |  |
| `phase` _string_ | Phase represents the current phase (Pending, Running, Succeeded, Failed, Unknown) |  | Enum: [Pending Running Succeeded Failed Unknown Suspended] <br /> |
| `conditions` _[Condition](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.28/#condition-v1-meta) array_ | Conditions represent the latest available observations of the agent's state |  |  |
| `activeReplicas` _integer_ | ActiveReplicas is the number of agent pods currently running |  |  |
| `readyReplicas` _integer_ | ReadyReplicas is the number of agent pods ready |  |  |
| `executionCount` _integer_ | ExecutionCount is the total number of executions |  |  |
| `successfulExecutions` _integer_ | SuccessfulExecutions is the number of successful executions |  |  |
| `failedExecutions` _integer_ | FailedExecutions is the number of failed executions |  |  |
| `lastExecutionTime` _[Time](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.28/#time-v1-meta)_ | LastExecutionTime is the timestamp of the last execution |  |  |
| `lastExecutionResult` _string_ | LastExecutionResult describes the result of the last execution |  |  |
| `currentGoal` _string_ | CurrentGoal is the current goal being pursued (for autonomous agents) |  |  |
| `iterationCount` _integer_ | IterationCount is the current iteration in the reasoning loop |  |  |
| `metrics` _[AgentMetrics](#agentmetrics)_ | Metrics contains execution metrics |  |  |
| `activeConversations` _integer_ | ActiveConversations is the number of active conversations |  |  |
| `toolUsage` _[ToolUsageSpec](#toolusagespec) array_ | ToolUsage tracks tool invocation statistics |  |  |
| `modelUsage` _[ModelUsageSpec](#modelusagespec) array_ | ModelUsage tracks model usage statistics |  |  |
| `costMetrics` _[AgentCostMetrics](#agentcostmetrics)_ | CostMetrics contains cost tracking data |  |  |
| `lastUpdateTime` _[Time](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.28/#time-v1-meta)_ | LastUpdateTime is the last time the status was updated |  |  |
| `message` _string_ | Message provides human-readable details about the current state |  |  |
| `reason` _string_ | Reason provides a machine-readable reason for the current state |  |  |
| `uuid` _string_ | UUID is a unique identifier for this agent instance<br />Used for webhook routing (e.g., <uuid>.domain.com) |  |  |
| `webhookURLs` _string array_ | WebhookURLs contains the URLs where this agent can receive webhooks |  |  |




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
| `domain` _string_ | Domain is the base domain for the cluster and agent webhook routing<br />The domain itself serves as the cluster dashboard/UI endpoint<br />Agent webhooks will be accessible at <uuid>.<domain><br />Example: "ai.theryans.io" results in webhooks like "abc123.ai.theryans.io" |  |  |
| `ingressConfig` _[IngressConfig](#ingressconfig)_ | IngressConfig defines ingress/gateway configuration for the cluster |  |  |
| `networkPolicies` _[NetworkRule](#networkrule) array_ | NetworkPolicies defines egress network policies for agents in this cluster |  |  |
| `proxy` _[ProxyConfig](#proxyconfig)_ | Proxy configures the shared LiteLLM proxy deployed per cluster |  |  |
| `capacity` _[ClusterCapacitySpec](#clustercapacityspec)_ | Capacity declares hard limits enforced via a ResourceQuota in the cluster's namespace.<br />When set, the controller creates a ResourceQuota named "langop-quota".<br />When unset, any existing "langop-quota" is deleted. |  |  |


#### LanguageClusterStatus



LanguageClusterStatus defines the observed state



_Appears in:_
- [LanguageCluster](#languagecluster)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `phase` _string_ | Phase of the cluster (Pending, Ready, Failed) |  |  |
| `conditions` _[Condition](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.28/#condition-v1-meta) array_ | Conditions |  |  |
| `proxyEndpoint` _string_ | ProxyEndpoint is the in-cluster URL for the shared LiteLLM proxy |  |  |
| `proxyReady` _boolean_ | ProxyReady indicates whether the shared proxy Deployment is available |  |  |
| `capacity` _[ClusterCapacityStatus](#clustercapacitystatus)_ | Capacity reports observed resource usage in this cluster's namespace. |  |  |


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
| `configuration` _[ProviderConfiguration](#providerconfiguration)_ | Configuration contains provider-specific configuration |  |  |
| `rateLimits` _[RateLimitSpec](#ratelimitspec)_ | RateLimits defines rate limiting configuration |  |  |
| `timeout` _string_ | Timeout specifies request timeout duration (e.g., "5m", "30s") | 5m | Pattern: `^[0-9]+(ns\|us\|µs\|ms\|s\|m\|h)$` <br /> |
| `retryPolicy` _[RetryPolicySpec](#retrypolicyspec)_ | RetryPolicy defines retry behavior for failed requests |  |  |
| `fallbacks` _[ModelFallbackSpec](#modelfallbackspec) array_ | Fallbacks is an ordered list of fallback models to use if this model fails |  |  |
| `loadBalancing` _[LoadBalancingSpec](#loadbalancingspec)_ | LoadBalancing defines load balancing strategy across multiple endpoints |  |  |
| `caching` _[CachingSpec](#cachingspec)_ | Caching defines response caching configuration |  |  |
| `observability` _[ObservabilitySpec](#observabilityspec)_ | Observability defines monitoring and tracing configuration |  |  |
| `costTracking` _[CostTrackingSpec](#costtrackingspec)_ | CostTracking enables cost tracking for this model |  |  |
| `regions` _[RegionSpec](#regionspec) array_ | Regions specifies which regions this model is deployed in (for multi-region) |  |  |
| `egress` _[NetworkRule](#networkrule) array_ | Egress defines external network access rules for this model proxy.<br />By default, model proxies can access all resources within the cluster but no external endpoints.<br />Common patterns:<br /># Azure OpenAI Service<br />egress:<br />- description: "Allow Azure OpenAI API"<br />  to:<br />    dns: ["*.openai.azure.com"]<br />  ports:<br />  - port: 443<br />    protocol: TCP<br /># AWS Bedrock<br />egress:<br />- description: "Allow AWS Bedrock API"<br />  to:<br />    dns: ["bedrock-runtime.*.amazonaws.com"]<br />  ports:<br />  - port: 443<br />    protocol: TCP<br /># Corporate proxy<br />egress:<br />- description: "Allow corporate proxy"<br />  to:<br />    cidr: "10.0.0.0/8"<br />  ports:<br />  - port: 8080<br />    protocol: TCP<br /># Local LM Studio<br />egress:<br />- description: "Allow LM Studio API"<br />  to:<br />    cidr: "192.168.1.0/24"<br />  ports:<br />  - port: 1234<br />    protocol: TCP |  |  |
| `resources` _[ResourceRequirements](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.28/#resourcerequirements-v1-core)_ | Resources defines CPU and memory resource requirements for the model proxy.<br />If not specified, defaults to:<br />  requests: cpu=100m, memory=128Mi<br />  limits: cpu=1000m, memory=512Mi |  |  |


#### LanguageModelStatus



LanguageModelStatus defines the observed state of LanguageModel



_Appears in:_
- [LanguageModel](#languagemodel)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `observedGeneration` _integer_ | ObservedGeneration reflects the generation of the most recently observed LanguageModel |  |  |
| `phase` _string_ | Phase represents the current phase (Ready, NotReady, Error, Configuring) |  | Enum: [Ready NotReady Error Configuring Degraded] <br /> |
| `conditions` _[Condition](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.28/#condition-v1-meta) array_ | Conditions represent the latest available observations of the model's state |  |  |
| `healthy` _boolean_ | Healthy indicates if the model is healthy and available |  |  |
| `lastHealthCheck` _[Time](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.28/#time-v1-meta)_ | LastHealthCheck is the timestamp of the last health check |  |  |
| `endpointStatus` _[EndpointStatusSpec](#endpointstatusspec) array_ | EndpointStatus shows status of each load-balanced endpoint |  |  |
| `regionStatus` _[RegionStatusSpec](#regionstatusspec) array_ | RegionStatus shows status of each region |  |  |
| `metrics` _[ModelMetrics](#modelmetrics)_ | Metrics contains usage metrics |  |  |
| `costMetrics` _[CostMetrics](#costmetrics)_ | CostMetrics contains cost tracking data |  |  |
| `lastUpdateTime` _[Time](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.28/#time-v1-meta)_ | LastUpdateTime is the last time the status was updated |  |  |
| `message` _string_ | Message provides human-readable details about the current state |  |  |
| `reason` _string_ | Reason provides a machine-readable reason for the current state |  |  |




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
| `displayName` _string_ | DisplayName is the human-readable name for this persona |  |  |
| `description` _string_ | Description describes the persona's role and behavior |  |  |
| `systemPrompt` _string_ | SystemPrompt is the base system instruction for this persona |  |  |
| `instructions` _string array_ | Instructions provides additional behavioral guidelines |  |  |
| `rules` _[PersonaRule](#personarule) array_ | Rules define conditional behaviors and policies |  |  |
| `examples` _[PersonaExample](#personaexample) array_ | Examples provide few-shot learning examples |  |  |
| `capabilities` _string array_ | Capabilities lists what this persona can do |  |  |
| `limitations` _string array_ | Limitations lists what this persona should not do |  |  |
| `tone` _string_ | Tone defines the communication style | professional | Enum: [professional casual friendly formal technical empathetic concise detailed] <br /> |
| `language` _string_ | Language specifies the primary language for responses | en |  |
| `responseFormat` _[ResponseFormatSpec](#responseformatspec)_ | ResponseFormat defines preferred response structure |  |  |
| `toolPreferences` _[ToolPreferencesSpec](#toolpreferencesspec)_ | ToolPreferences defines how this persona uses tools |  |  |
| `knowledgeSources` _[KnowledgeSourceSpec](#knowledgesourcespec) array_ | KnowledgeSources references external knowledge bases |  |  |
| `constraints` _[PersonaConstraints](#personaconstraints)_ | Constraints define operational constraints |  |  |
| `metadata` _object (keys:string, values:string)_ | Refer to Kubernetes API documentation for fields of `metadata`. |  |  |
| `version` _string_ | Version tracks the persona version |  |  |
| `parentPersona` _[PersonaReference](#personareference)_ | ParentPersona references a parent persona to inherit from |  |  |


#### LanguagePersonaStatus



LanguagePersonaStatus defines the observed state of LanguagePersona



_Appears in:_
- [LanguagePersona](#languagepersona)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `observedGeneration` _integer_ | ObservedGeneration reflects the generation of the most recently observed LanguagePersona |  |  |
| `phase` _string_ | Phase represents the current phase (Ready, NotReady) |  | Enum: [Ready NotReady] <br /> |
| `conditions` _[Condition](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.28/#condition-v1-meta) array_ | Conditions represent the latest available observations of the persona's state |  |  |
| `lastUpdateTime` _[Time](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.28/#time-v1-meta)_ | LastUpdateTime is the last time the status was updated |  |  |
| `message` _string_ | Message provides human-readable details about the current state |  |  |




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
| `imagePullPolicy` _[PullPolicy](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.28/#pullpolicy-v1-core)_ | ImagePullPolicy defines when to pull the container image | IfNotPresent | Enum: [Always Never IfNotPresent] <br /> |
| `imagePullSecrets` _[LocalObjectReference](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.28/#localobjectreference-v1-core) array_ | ImagePullSecrets is a list of references to secrets for pulling images |  |  |
| `type` _string_ | Type specifies the tool protocol type (e.g., "mcp", "openapi") | mcp | Enum: [mcp openapi] <br /> |
| `deploymentMode` _string_ | DeploymentMode specifies how this tool should be deployed<br />- "service": Deployed as a standalone Deployment+Service (default, shared across agents)<br />- "sidecar": Deployed as a sidecar container in each agent pod (dedicated, with workspace access) | service | Enum: [service sidecar] <br /> |
| `port` _integer_ | Port is the port the tool listens on | 8080 | Maximum: 65535 <br />Minimum: 1 <br /> |
| `replicas` _integer_ | Replicas is the number of pod replicas to run | 1 | Minimum: 0 <br /> |
| `env` _[EnvVar](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.28/#envvar-v1-core) array_ | Env contains environment variables for the tool container |  |  |
| `envFrom` _[EnvFromSource](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.28/#envfromsource-v1-core) array_ | EnvFrom sources to populate environment variables |  |  |
| `resources` _[ResourceRequirements](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.28/#resourcerequirements-v1-core)_ | Resources defines compute resource requirements |  |  |
| `nodeSelector` _object (keys:string, values:string)_ | NodeSelector is a selector which must match a node's labels for the pod to be scheduled |  |  |
| `affinity` _[Affinity](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.28/#affinity-v1-core)_ | Affinity defines pod affinity and anti-affinity rules |  |  |
| `tolerations` _[Toleration](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.28/#toleration-v1-core) array_ | Tolerations allow pods to schedule onto nodes with matching taints |  |  |
| `topologySpreadConstraints` _[TopologySpreadConstraint](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.28/#topologyspreadconstraint-v1-core) array_ | TopologySpreadConstraints describes how pods should spread across topology domains |  |  |
| `serviceAccountName` _string_ | ServiceAccountName is the name of the ServiceAccount to use for this tool |  |  |
| `securityContext` _[PodSecurityContext](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.28/#podsecuritycontext-v1-core)_ | SecurityContext holds pod-level security attributes |  |  |
| `volumeMounts` _[VolumeMount](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.28/#volumemount-v1-core) array_ | VolumeMounts to mount into the tool container |  |  |
| `volumes` _[Volume](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.28/#volume-v1-core) array_ | Volumes to attach to the pod |  |  |
| `livenessProbe` _[Probe](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.28/#probe-v1-core)_ | LivenessProbe defines the liveness probe for the tool container |  |  |
| `readinessProbe` _[Probe](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.28/#probe-v1-core)_ | ReadinessProbe defines the readiness probe for the tool container |  |  |
| `startupProbe` _[Probe](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.28/#probe-v1-core)_ | StartupProbe defines the startup probe for the tool container |  |  |
| `serviceType` _[ServiceType](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.28/#servicetype-v1-core)_ | ServiceType specifies the type of Service to create (ClusterIP, NodePort, LoadBalancer) | ClusterIP | Enum: [ClusterIP NodePort LoadBalancer] <br /> |
| `serviceAnnotations` _object (keys:string, values:string)_ | ServiceAnnotations are annotations to add to the Service |  |  |
| `podAnnotations` _object (keys:string, values:string)_ | PodAnnotations are annotations to add to the Pods |  |  |
| `podLabels` _object (keys:string, values:string)_ | PodLabels are additional labels to add to the Pods |  |  |
| `podDisruptionBudget` _[PodDisruptionBudgetSpec](#poddisruptionbudgetspec)_ | PodDisruptionBudget defines the PDB for this tool |  |  |
| `updateStrategy` _[UpdateStrategySpec](#updatestrategyspec)_ | UpdateStrategy defines the update strategy for the Deployment |  |  |
| `egress` _[NetworkRule](#networkrule) array_ | Egress defines external network access rules for this tool<br />By default, tools can access all resources within the cluster but no external endpoints |  |  |


#### LanguageToolStatus



LanguageToolStatus defines the observed state of LanguageTool



_Appears in:_
- [LanguageTool](#languagetool)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `observedGeneration` _integer_ | ObservedGeneration reflects the generation of the most recently observed LanguageTool |  |  |
| `phase` _string_ | Phase represents the current phase of the tool (Pending, Running, Failed, Unknown) |  | Enum: [Pending Running Failed Unknown Updating] <br /> |
| `conditions` _[Condition](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.28/#condition-v1-meta) array_ | Conditions represent the latest available observations of the tool's state |  |  |
| `endpoint` _string_ | Endpoint is the service endpoint where the tool is accessible |  |  |
| `availableTools` _string array_ | AvailableTools lists the tools discovered from this service |  |  |
| `toolSchemas` _[ToolSchema](#toolschema) array_ | ToolSchemas contains the complete MCP tool schemas discovered from this service |  |  |
| `readyReplicas` _integer_ | ReadyReplicas is the number of pods ready and passing health checks |  |  |
| `availableReplicas` _integer_ | AvailableReplicas is the number of pods targeted by this LanguageTool with at least one available condition |  |  |
| `updatedReplicas` _integer_ | UpdatedReplicas is the number of pods targeted by this LanguageTool that have the desired spec |  |  |
| `unavailableReplicas` _integer_ | UnavailableReplicas is the number of pods targeted by this LanguageTool that are unavailable |  |  |
| `lastUpdateTime` _[Time](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.28/#time-v1-meta)_ | LastUpdateTime is the last time the status was updated |  |  |
| `lastTransitionTime` _[Time](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.28/#time-v1-meta)_ | LastTransitionTime is the last time the phase transitioned |  |  |
| `message` _string_ | Message provides human-readable details about the current state |  |  |
| `reason` _string_ | Reason provides a machine-readable reason for the current state |  |  |




#### LoadBalancingSpec



LoadBalancingSpec defines load balancing configuration



_Appears in:_
- [LanguageModelSpec](#languagemodelspec)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `strategy` _string_ | Strategy specifies the load balancing strategy | round-robin | Enum: [round-robin least-connections random weighted latency-based] <br /> |
| `endpoints` _[EndpointSpec](#endpointspec) array_ | Endpoints is a list of endpoint configurations for load balancing |  |  |
| `healthCheck` _[HealthCheckSpec](#healthcheckspec)_ | HealthCheck defines health checking for endpoints |  |  |


#### ModelCostSpec



ModelCostSpec tracks cost per model



_Appears in:_
- [AgentCostMetrics](#agentcostmetrics)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `modelName` _string_ | ModelName is the name of the model |  |  |
| `cost` _float_ | Cost is the total cost for this model |  |  |


#### ModelFallbackSpec



ModelFallbackSpec defines a fallback model



_Appears in:_
- [LanguageModelSpec](#languagemodelspec)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `modelRef` _string_ | ModelRef is a reference to another LanguageModel |  | Required: \{\} <br /> |
| `conditions` _string array_ | Conditions specifies when to use this fallback |  |  |


#### ModelLoggingSpec



ModelLoggingSpec defines logging configuration



_Appears in:_
- [ObservabilitySpec](#observabilityspec)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `level` _string_ | Level is the log level (debug, info, warn, error) | info | Enum: [debug info warn error] <br /> |
| `logRequests` _boolean_ | LogRequests enables request logging | true |  |
| `logResponses` _boolean_ | LogResponses enables response logging | false |  |


#### ModelMetrics



ModelMetrics contains usage metrics



_Appears in:_
- [LanguageModelStatus](#languagemodelstatus)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `totalRequests` _integer_ | TotalRequests is the total number of requests |  |  |
| `successfulRequests` _integer_ | SuccessfulRequests is the number of successful requests |  |  |
| `failedRequests` _integer_ | FailedRequests is the number of failed requests |  |  |
| `totalTokens` _integer_ | TotalTokens is the total number of tokens processed |  |  |
| `inputTokens` _integer_ | InputTokens is the total number of input tokens |  |  |
| `outputTokens` _integer_ | OutputTokens is the total number of output tokens |  |  |
| `averageLatency` _integer_ | AverageLatency is the average request latency in milliseconds |  |  |
| `p95Latency` _integer_ | P95Latency is the 95th percentile latency in milliseconds |  |  |
| `p99Latency` _integer_ | P99Latency is the 99th percentile latency in milliseconds |  |  |


#### ModelReference



ModelReference references a LanguageModel



_Appears in:_
- [LanguageAgentSpec](#languageagentspec)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `name` _string_ | Name is the name of the LanguageModel |  | MaxLength: 63 <br />Pattern: `^[a-z0-9]([-a-z0-9]*[a-z0-9])?$` <br />Required: \{\} <br /> |
| `role` _string_ | Role defines the purpose of this model (primary, fallback, specialized) | primary | Enum: [primary fallback reasoning tool-calling summarization] <br /> |
| `priority` _integer_ | Priority for model selection (lower is higher priority) |  |  |


#### ModelUsageSpec



ModelUsageSpec tracks model usage



_Appears in:_
- [LanguageAgentStatus](#languageagentstatus)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `modelName` _string_ | ModelName is the name of the model |  |  |
| `requestCount` _integer_ | RequestCount is the number of requests to this model |  |  |
| `totalTokens` _integer_ | TotalTokens is the total tokens consumed by this model |  |  |
| `inputTokens` _integer_ | InputTokens is the total input tokens |  |  |
| `outputTokens` _integer_ | OutputTokens is the total output tokens |  |  |


#### NetworkPeer



NetworkPeer defines the source/destination of network traffic



_Appears in:_
- [NetworkRule](#networkrule)

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
- [NetworkRule](#networkrule)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `protocol` _string_ | Protocol (TCP, UDP, SCTP) | TCP | Enum: [TCP UDP SCTP] <br /> |
| `port` _integer_ | Port number |  | Maximum: 65535 <br />Minimum: 1 <br /> |


#### NetworkRule



NetworkRule defines a single network policy rule



_Appears in:_
- [LanguageAgentSpec](#languageagentspec)
- [LanguageClusterSpec](#languageclusterspec)
- [LanguageModelSpec](#languagemodelspec)
- [LanguageToolSpec](#languagetoolspec)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `description` _string_ | Description of this rule |  |  |
| `from` _[NetworkPeer](#networkpeer)_ | From selector for ingress rules |  |  |
| `to` _[NetworkPeer](#networkpeer)_ | To selector for egress rules |  |  |
| `ports` _[NetworkPort](#networkport) array_ | Ports allowed by this rule |  |  |


#### ObservabilitySpec



ObservabilitySpec defines monitoring and tracing



_Appears in:_
- [LanguageModelSpec](#languagemodelspec)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `metrics` _boolean_ | Metrics enables metrics collection | true |  |
| `tracing` _boolean_ | Tracing enables distributed tracing | false |  |
| `logging` _[ModelLoggingSpec](#modelloggingspec)_ | Logging defines logging configuration |  |  |


#### PersonaConstraints



PersonaConstraints defines operational constraints



_Appears in:_
- [LanguagePersonaSpec](#languagepersonaspec)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `maxResponseTokens` _integer_ | MaxResponseTokens limits response length in tokens |  |  |
| `maxToolCalls` _integer_ | MaxToolCalls limits tool invocations per interaction |  |  |
| `maxKnowledgeQueries` _integer_ | MaxKnowledgeQueries limits knowledge base queries per interaction |  |  |
| `responseTimeout` _string_ | ResponseTimeout limits response generation time |  | Pattern: `^[0-9]+(ns\|us\|µs\|ms\|s\|m\|h)$` <br /> |
| `requireDocumentation` _boolean_ | RequireDocumentation requires citing sources for claims | false |  |
| `blockedTopics` _string array_ | BlockedTopics lists topics this persona should refuse to discuss |  |  |
| `allowedDomains` _string array_ | AllowedDomains restricts knowledge sources to specific domains |  |  |


#### PersonaExample



PersonaExample provides a few-shot learning example



_Appears in:_
- [LanguagePersonaSpec](#languagepersonaspec)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `input` _string_ | Input is the example user input |  | Required: \{\} <br /> |
| `output` _string_ | Output is the expected persona response |  | Required: \{\} <br /> |
| `context` _string_ | Context provides additional context for this example |  |  |
| `tags` _string array_ | Tags categorize this example |  |  |


#### PersonaReference



PersonaReference references a LanguagePersona



_Appears in:_
- [LanguageAgentSpec](#languageagentspec)
- [LanguagePersonaSpec](#languagepersonaspec)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `name` _string_ | Name is the name of the LanguagePersona |  | MaxLength: 63 <br />Pattern: `^[a-z0-9]([-a-z0-9]*[a-z0-9])?$` <br />Required: \{\} <br /> |


#### PersonaRule



PersonaRule defines a conditional behavior rule



_Appears in:_
- [LanguagePersonaSpec](#languagepersonaspec)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `name` _string_ | Name is a unique identifier for this rule |  | Required: \{\} <br /> |
| `description` _string_ | Description explains what this rule does |  |  |
| `condition` _string_ | Condition defines when this rule applies (e.g., "when asked about X") |  | Required: \{\} <br /> |
| `action` _string_ | Action defines what to do when condition matches |  | Required: \{\} <br /> |
| `priority` _integer_ | Priority determines rule evaluation order (lower is higher priority) | 100 |  |
| `enabled` _boolean_ | Enabled indicates if this rule is active | true |  |


#### PodDisruptionBudgetSpec



PodDisruptionBudgetSpec defines PDB configuration



_Appears in:_
- [LanguageToolSpec](#languagetoolspec)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `minAvailable` _integer_ | MinAvailable specifies the minimum number of pods that must be available |  |  |
| `maxUnavailable` _integer_ | MaxUnavailable specifies the maximum number of pods that can be unavailable |  |  |


#### ProviderConfiguration



ProviderConfiguration contains provider-specific settings



_Appears in:_
- [LanguageModelSpec](#languagemodelspec)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `maxTokens` _integer_ | MaxTokens is the maximum tokens for responses |  |  |
| `temperature` _float_ | Temperature controls randomness (0.0 to 2.0) |  |  |
| `topP` _float_ | TopP controls nucleus sampling |  |  |
| `frequencyPenalty` _float_ | FrequencyPenalty penalizes frequent tokens (-2.0 to 2.0) |  |  |
| `presencePenalty` _float_ | PresencePenalty penalizes tokens based on presence (-2.0 to 2.0) |  |  |
| `stopSequences` _string array_ | StopSequences are sequences that stop generation |  |  |
| `additionalParameters` _object (keys:string, values:string)_ | AdditionalParameters for provider-specific options |  |  |


#### ProxyConfig



ProxyConfig configures the shared LiteLLM proxy for a LanguageCluster



_Appears in:_
- [LanguageClusterSpec](#languageclusterspec)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `ingressEnabled` _boolean_ | IngressEnabled controls whether an Ingress/HTTPRoute is created for the proxy.<br />Defaults to true when cluster.spec.domain is set. |  |  |
| `resources` _[ResourceRequirements](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.28/#resourcerequirements-v1-core)_ | Resources sets CPU/memory requests and limits for the proxy Deployment. |  |  |
| `replicas` _integer_ | Replicas sets the number of proxy pod replicas. | 1 |  |


#### RateLimitSpec



RateLimitSpec defines rate limiting configuration



_Appears in:_
- [LanguageModelSpec](#languagemodelspec)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `requestsPerMinute` _integer_ | RequestsPerMinute limits requests per minute |  |  |
| `tokensPerMinute` _integer_ | TokensPerMinute limits tokens per minute |  |  |
| `concurrentRequests` _integer_ | ConcurrentRequests limits concurrent requests |  |  |


#### RegionSpec



RegionSpec defines a region configuration



_Appears in:_
- [LanguageModelSpec](#languagemodelspec)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `name` _string_ | Name is the region name (e.g., "us-east-1", "eu-west-1") |  | Required: \{\} <br /> |
| `endpoint` _string_ | Endpoint is the region-specific endpoint URL |  |  |
| `priority` _integer_ | Priority for region routing (lower is higher priority) |  |  |
| `enabled` _boolean_ | Enabled indicates if this region is enabled | true |  |


#### RegionStatusSpec



RegionStatusSpec shows the status of a region



_Appears in:_
- [LanguageModelStatus](#languagemodelstatus)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `name` _string_ | Name is the region name |  |  |
| `available` _boolean_ | Available indicates if the region is available |  |  |
| `latency` _integer_ | Latency is the average latency to this region in milliseconds |  |  |
| `lastCheck` _[Time](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.28/#time-v1-meta)_ | LastCheck is the timestamp of the last check |  |  |


#### ResponseFormatSpec



ResponseFormatSpec defines response structure preferences



_Appears in:_
- [LanguagePersonaSpec](#languagepersonaspec)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `type` _string_ | Type specifies the response format | text | Enum: [text markdown json structured list table] <br /> |
| `template` _string_ | Template defines a response template |  |  |
| `schema` _string_ | Schema defines JSON schema for structured responses |  |  |
| `maxLength` _integer_ | MaxLength limits response length in characters |  |  |
| `includeSources` _boolean_ | IncludeSources indicates whether to cite sources | false |  |
| `includeConfidence` _boolean_ | IncludeConfidence indicates whether to include confidence scores | false |  |


#### RetryPolicySpec



RetryPolicySpec defines retry behavior



_Appears in:_
- [LanguageModelSpec](#languagemodelspec)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `maxAttempts` _integer_ | MaxAttempts is the maximum number of retry attempts | 3 | Maximum: 10 <br />Minimum: 0 <br /> |
| `initialBackoff` _string_ | InitialBackoff is the initial backoff duration (e.g., "1s") | 1s | Pattern: `^[0-9]+(ns\|us\|µs\|ms\|s\|m\|h)$` <br /> |
| `maxBackoff` _string_ | MaxBackoff is the maximum backoff duration (e.g., "30s") | 30s | Pattern: `^[0-9]+(ns\|us\|µs\|ms\|s\|m\|h)$` <br /> |
| `backoffMultiplier` _float_ | BackoffMultiplier is the multiplier for exponential backoff | 2 |  |
| `retryableStatusCodes` _integer array_ | RetryableStatusCodes are HTTP status codes that trigger retry |  |  |


#### RollingUpdateSpec



RollingUpdateSpec defines rolling update parameters



_Appears in:_
- [UpdateStrategySpec](#updatestrategyspec)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `maxUnavailable` _integer_ | MaxUnavailable is the maximum number of pods that can be unavailable during update |  |  |
| `maxSurge` _integer_ | MaxSurge is the maximum number of pods that can be created above desired replicas |  |  |


#### SecretReference



SecretReference references a Kubernetes Secret



_Appears in:_
- [KnowledgeSourceSpec](#knowledgesourcespec)
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


#### ToolPreferencesSpec



ToolPreferencesSpec defines tool usage preferences



_Appears in:_
- [LanguagePersonaSpec](#languagepersonaspec)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `preferredTools` _string array_ | PreferredTools lists tools to prefer using |  |  |
| `avoidTools` _string array_ | AvoidTools lists tools to avoid using |  |  |
| `strategy` _string_ | ToolUsageStrategy defines how aggressively to use tools | balanced | Enum: [conservative balanced aggressive minimal] <br /> |
| `alwaysConfirm` _boolean_ | AlwaysConfirm requires confirmation before tool use | false |  |
| `explainToolUse` _boolean_ | ExplainToolUse explains tool usage to users | true |  |


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
| `enabled` _boolean_ | Enabled indicates if this tool is available to the agent | true |  |
| `requireApproval` _boolean_ | RequireApproval requires human approval before tool execution | false |  |


#### ToolSchema



ToolSchema represents the complete schema of an MCP tool



_Appears in:_
- [LanguageToolStatus](#languagetoolstatus)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `name` _string_ | Name is the tool identifier |  |  |
| `description` _string_ | Description is a human-readable description of the tool |  |  |
| `inputSchema` _[ToolSchemaDefinition](#toolschemadefinition)_ | InputSchema defines the parameters this tool accepts |  |  |
| `outputSchema` _[ToolSchemaDefinition](#toolschemadefinition)_ | OutputSchema defines the structure this tool returns |  |  |


#### ToolSchemaDefinition



ToolSchemaDefinition defines parameter or return value structure



_Appears in:_
- [ToolSchema](#toolschema)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `type` _string_ | Type is the JSON schema type (object, array, string, etc.) |  |  |
| `properties` _object (keys:string, values:[ToolProperty](#toolproperty))_ | Properties defines object properties (for type: object) |  |  |
| `required` _string array_ | Required lists required property names (for type: object) |  |  |


#### ToolUsageSpec



ToolUsageSpec tracks tool usage



_Appears in:_
- [LanguageAgentStatus](#languageagentstatus)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `toolName` _string_ | ToolName is the name of the tool |  |  |
| `invocationCount` _integer_ | InvocationCount is the number of times this tool was invoked |  |  |
| `successCount` _integer_ | SuccessCount is the number of successful invocations |  |  |
| `failureCount` _integer_ | FailureCount is the number of failed invocations |  |  |
| `averageLatency` _integer_ | AverageLatency is the average latency in milliseconds |  |  |


#### UpdateStrategySpec



UpdateStrategySpec defines deployment update strategy



_Appears in:_
- [LanguageToolSpec](#languagetoolspec)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `type` _string_ | Type of deployment update strategy (RollingUpdate or Recreate) | RollingUpdate | Enum: [RollingUpdate Recreate] <br /> |
| `rollingUpdate` _[RollingUpdateSpec](#rollingupdatespec)_ | RollingUpdate configuration (only used if Type is RollingUpdate) |  |  |


#### WorkspaceSpec



WorkspaceSpec defines persistent workspace storage for an agent



_Appears in:_
- [LanguageAgentSpec](#languageagentspec)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `enabled` _boolean_ | Enabled controls whether to create a workspace volume | true |  |
| `size` _string_ | Size is the requested storage size (e.g., "10Gi", "1.5Ti", "500Mi")<br />Supports integer and decimal quantities with standard Kubernetes suffixes | 10Gi | MinLength: 1 <br />Pattern: `^([0-9]*\.?[0-9]+)(Ei\|Pi\|Ti\|Gi\|Mi\|Ki\|E\|P\|T\|G\|M\|K\|m)?$` <br /> |
| `storageClassName` _string_ | StorageClassName specifies the StorageClass for the PVC<br />If not specified, uses the cluster default |  |  |
| `accessMode` _string_ | AccessMode defines the volume access mode | ReadWriteOnce | Enum: [ReadWriteOnce ReadWriteMany] <br /> |
| `mountPath` _string_ | MountPath is where the workspace is mounted in containers | /workspace |  |


