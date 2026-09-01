# API Reference

## Packages
- [langop.io/v1alpha1](#langopiov1alpha1)


## langop.io/v1alpha1

Package v1alpha1 contains API Schema definitions for the language v1alpha1 API group

### Resource Types
- [LanguageAgent](#languageagent)
- [LanguageAgentRuntime](#languageagentruntime)
- [LanguageAgentSelfConfig](#languageagentselfconfig)
- [LanguageCluster](#languagecluster)
- [LanguageModel](#languagemodel)
- [LanguagePersona](#languagepersona)
- [LanguageTool](#languagetool)



#### AgentMonitoringSpec



AgentMonitoringSpec defines Prometheus Operator integration for a LanguageAgent.



_Appears in:_
- [LanguageAgentSpec](#languageagentspec)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `serviceMonitor` _[AgentServiceMonitorSpec](#agentservicemonitorspec)_ | ServiceMonitor configures a ServiceMonitor resource for this agent. |  | Optional: \{\} <br /> |
| `rules` _[PrometheusRuleGroup](#prometheusrulegroup) array_ | Rules defines PrometheusRule groups for this agent.<br />When non-empty, the operator creates a PrometheusRule resource. |  | Optional: \{\} <br /> |


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
| `ingress` _[NetworkIngressRule](#networkingressrule) array_ | Ingress rules — each entry allows traffic into the workload from the listed sources. |  | Optional: \{\} <br /> |
| `egress` _[NetworkEgressRule](#networkegressrule) array_ | Egress rules — each entry allows traffic from the workload to the listed destinations. |  | Optional: \{\} <br /> |


#### AgentPort



AgentPort describes a single network port that an agent container exposes.



_Appears in:_
- [LanguageAgentRuntimeSpec](#languageagentruntimespec)
- [LanguageAgentSpec](#languageagentspec)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `name` _string_ | Name uniquely identifies this port within the agent (e.g., "web", "ws").<br />Used as the Service port name; must conform to Kubernetes port-name rules. |  | MaxLength: 15 <br />Pattern: `^[a-z][a-z0-9-]*$` <br />Required: \{\} <br /> |
| `port` _integer_ | Port is the port number the container listens on. |  | Maximum: 65535 <br />Minimum: 1 <br />Required: \{\} <br /> |
| `protocol` _[Protocol](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.28/#protocol-v1-core)_ | Protocol is the transport protocol. Defaults to TCP. | TCP | Enum: [TCP UDP SCTP] <br />Optional: \{\} <br /> |
| `expose` _boolean_ | Expose controls whether ingress/HTTPRoute routes to this port.<br />At most one port should have expose: true; if none, the first port is used. | false | Optional: \{\} <br /> |


#### AgentServiceMonitorSpec



AgentServiceMonitorSpec configures a Prometheus Operator ServiceMonitor for an agent.



_Appears in:_
- [AgentMonitoringSpec](#agentmonitoringspec)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `enabled` _boolean_ | Enabled controls whether a ServiceMonitor is created for this agent. |  |  |
| `port` _string_ | Port is the name of the service port to scrape for metrics.<br />Defaults to the name of the first port in spec.ports, or "http" if no ports are defined. |  | Optional: \{\} <br /> |
| `path` _string_ | Path is the HTTP path to scrape for metrics. Defaults to /metrics. |  | Optional: \{\} <br /> |
| `interval` _string_ | Interval is the scrape interval (e.g. "30s"). Uses the Prometheus default when omitted. |  | Optional: \{\} <br /> |
| `scrapeTimeout` _string_ | ScrapeTimeout is the per-scrape timeout. Uses the Prometheus default when omitted. |  | Optional: \{\} <br /> |
| `labels` _object (keys:string, values:string)_ | Labels are additional labels added to the ServiceMonitor metadata. |  | Optional: \{\} <br /> |


#### AutoscalingSpec



AutoscalingSpec configures a HorizontalPodAutoscaler for the deployment.



_Appears in:_
- [DeploymentSpec](#deploymentspec)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `minReplicas` _integer_ | MinReplicas is the lower bound for replicas the HPA can scale down to.<br />Defaults to 1 if not specified. |  | Minimum: 1 <br />Optional: \{\} <br /> |
| `maxReplicas` _integer_ | MaxReplicas is the upper bound for replicas the HPA can scale up to. |  | Minimum: 1 <br /> |
| `metrics` _[MetricSpec](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.28/#metricspec-v2-autoscaling) array_ | Metrics specifies which metrics to use for scaling.<br />Defaults to 80% average CPU utilization if not specified. |  | Optional: \{\} <br /> |


#### ClusterAuthSpec



ClusterAuthSpec configures OIDC authentication for a LanguageCluster.



_Appears in:_
- [LanguageClusterSpec](#languageclusterspec)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `enabled` _boolean_ | Enabled controls whether OIDC authentication is active for this cluster. |  | Optional: \{\} <br /> |
| `oidc` _[ClusterOIDCSpec](#clusteroidcspec)_ | OIDC configures the OIDC provider (embedded Dex or external). |  | Optional: \{\} <br /> |


#### ClusterCapacitySpec



ClusterCapacitySpec declares hard limits enforced via a ResourceQuota in the cluster's namespace.



_Appears in:_
- [LanguageClusterSpec](#languageclusterspec)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `maxAgents` _integer_ | MaxAgents is the maximum number of LanguageAgent objects allowed. |  | Optional: \{\} <br /> |
| `maxModels` _integer_ | MaxModels is the maximum number of LanguageModel objects allowed. |  | Optional: \{\} <br /> |
| `maxTools` _integer_ | MaxTools is the maximum number of LanguageTool objects allowed. |  | Optional: \{\} <br /> |
| `maxPersonas` _integer_ | MaxPersonas is the maximum number of LanguagePersona objects allowed. |  | Optional: \{\} <br /> |
| `maxCPU` _[Quantity](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.28/#quantity-resource-api)_ | MaxCPU is the aggregate CPU limit for all pods in the cluster namespace.<br />Maps to limits.cpu in the namespace ResourceQuota.<br />Example: "4", "2500m" |  | Optional: \{\} <br /> |
| `maxMemory` _[Quantity](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.28/#quantity-resource-api)_ | MaxMemory is the aggregate memory limit for all pods in the cluster namespace.<br />Maps to limits.memory in the namespace ResourceQuota.<br />Example: "8Gi", "512Mi" |  | Optional: \{\} <br /> |


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
| `totalCPULimits` _[Quantity](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.28/#quantity-resource-api)_ | TotalCPULimits is the sum of limits.cpu across all agent pod specs. |  | Optional: \{\} <br /> |
| `totalMemoryLimits` _[Quantity](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.28/#quantity-resource-api)_ | TotalMemoryLimits is the sum of limits.memory across all agent pod specs. |  | Optional: \{\} <br /> |


#### ClusterOIDCSpec



ClusterOIDCSpec configures the OIDC provider for the cluster.



_Appears in:_
- [ClusterAuthSpec](#clusterauthspec)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `dex` _[DexSpec](#dexspec)_ | Dex configures the embedded Dex OIDC provider.<br />When set (and ExternalIssuerURL is not), the controller deploys Dex alongside the gateway. |  | Optional: \{\} <br /> |
| `externalIssuerURL` _string_ | ExternalIssuerURL skips deploying Dex and uses this issuer URL for oauth2-proxy.<br />Mutually exclusive with dex.<br />Example: "https://accounts.google.com" |  | Optional: \{\} <br /> |
| `clientID` _string_ | ClientID is the OAuth2 client ID when using an external OIDC provider.<br />Ignored when dex is configured (the operator manages the client ID). |  | Optional: \{\} <br /> |
| `clientSecretRef` _[SecretReference](#secretreference)_ | ClientSecretRef references a Secret containing the OAuth2 client secret.<br />Ignored when dex is configured (the operator manages the client secret). |  | Optional: \{\} <br /> |
| `emailDomain` _string_ | EmailDomain restricts login to users with this email domain.<br />Set to "*" to allow all email domains (default). |  | Optional: \{\} <br /> |


#### CredentialSpec



CredentialSpec declares an environment variable backed by a Secret value.
The operator resolves it once and injects it into the agent container.
Source priority: ValueFrom (existing Secret) > Value (inline) > auto-generate.



_Appears in:_
- [LanguageAgentRuntimeSpec](#languageagentruntimespec)
- [LanguageAgentSpec](#languageagentspec)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `name` _string_ | Name is the environment variable name and the key within the operator-managed<br />Secret (e.g. OPENCLAW_GATEWAY_TOKEN, OPENCODE_SERVER_PASSWORD). |  | Required: \{\} <br /> |
| `value` _string_ | Value, when set, is stored verbatim in the managed Secret instead of<br />generating a random credential. Mutually exclusive with ValueFrom. |  | Optional: \{\} <br /> |
| `valueFrom` _[RuntimeSecretRef](#runtimesecretref)_ | ValueFrom references an existing Secret whose keys are injected via envFrom.<br />When set, the operator does not create or manage a Secret for this entry.<br />Mutually exclusive with Value. |  | Optional: \{\} <br /> |


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
| `replicas` _integer_ | Replicas is the number of pod replicas to run. |  | Minimum: 0 <br />Optional: \{\} <br /> |
| `imagePullPolicy` _[PullPolicy](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.28/#pullpolicy-v1-core)_ | ImagePullPolicy defines when to pull the container image. |  | Enum: [Always Never IfNotPresent] <br />Optional: \{\} <br /> |
| `imagePullSecrets` _[LocalObjectReference](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.28/#localobjectreference-v1-core) array_ | ImagePullSecrets is a list of references to secrets for pulling images. |  | Optional: \{\} <br /> |
| `env` _[EnvVar](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.28/#envvar-v1-core) array_ | Env contains environment variables for the container. |  | Optional: \{\} <br /> |
| `envFrom` _[EnvFromSource](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.28/#envfromsource-v1-core) array_ | EnvFrom sources to populate environment variables. |  | Optional: \{\} <br /> |
| `resources` _[ResourceRequirements](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.28/#resourcerequirements-v1-core)_ | Resources defines compute resource requirements. |  | Optional: \{\} <br /> |
| `nodeSelector` _object (keys:string, values:string)_ | NodeSelector is a selector which must match a node's labels. |  | Optional: \{\} <br /> |
| `affinity` _[Affinity](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.28/#affinity-v1-core)_ | Affinity defines pod affinity and anti-affinity rules. |  | Optional: \{\} <br /> |
| `tolerations` _[Toleration](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.28/#toleration-v1-core) array_ | Tolerations allow pods to schedule onto nodes with matching taints. |  | Optional: \{\} <br /> |
| `topologySpreadConstraints` _[TopologySpreadConstraint](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.28/#topologyspreadconstraint-v1-core) array_ | TopologySpreadConstraints describes how pods should spread across topology domains. |  | Optional: \{\} <br /> |
| `serviceAccountName` _string_ | ServiceAccountName is the name of the ServiceAccount to use. |  | Optional: \{\} <br /> |
| `serviceAccountAnnotations` _object (keys:string, values:string)_ | ServiceAccountAnnotations are annotations to add to the operator-managed ServiceAccount.<br />Use this to attach cloud workload identity bindings, e.g. AWS IRSA, GCP WI, AKS WI.<br />Ignored when ServiceAccountName is set. |  | Optional: \{\} <br /> |
| `roleRules` _[PolicyRule](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.28/#policyrule-v1-rbac) array_ | RoleRules are additional RBAC policy rules appended to the operator-managed Role.<br />Use this to grant the agent extra in-cluster permissions beyond the defaults<br />(configmaps get/list, pods get/list/watch).<br />Ignored when ServiceAccountName is set. |  | Optional: \{\} <br /> |
| `securityContext` _[PodSecurityContext](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.28/#podsecuritycontext-v1-core)_ | SecurityContext holds pod-level security attributes. |  | Optional: \{\} <br /> |
| `volumeMounts` _[VolumeMount](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.28/#volumemount-v1-core) array_ | VolumeMounts to mount into the container. |  | Optional: \{\} <br /> |
| `volumes` _[Volume](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.28/#volume-v1-core) array_ | Volumes to attach to the pod. |  | Optional: \{\} <br /> |
| `podAnnotations` _object (keys:string, values:string)_ | PodAnnotations are annotations to add to the Pods. |  | Optional: \{\} <br /> |
| `podLabels` _object (keys:string, values:string)_ | PodLabels are additional labels to add to the Pods. |  | Optional: \{\} <br /> |
| `initContainers` _[Container](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.28/#container-v1-core) array_ | InitContainers are additional init containers injected before the main container starts. |  | Optional: \{\} <br /> |
| `livenessProbe` _[Probe](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.28/#probe-v1-core)_ | LivenessProbe defines the liveness probe for the container. |  | Optional: \{\} <br /> |
| `readinessProbe` _[Probe](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.28/#probe-v1-core)_ | ReadinessProbe defines the readiness probe for the container. |  | Optional: \{\} <br /> |
| `startupProbe` _[Probe](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.28/#probe-v1-core)_ | StartupProbe defines the startup probe for the container. |  | Optional: \{\} <br /> |
| `command` _string array_ | Command overrides the container entrypoint. |  | Optional: \{\} <br /> |
| `args` _string array_ | Args overrides the container command arguments. |  | Optional: \{\} <br /> |
| `serviceType` _[ServiceType](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.28/#servicetype-v1-core)_ | ServiceType specifies the type of Service to create (ClusterIP, NodePort, LoadBalancer). |  | Enum: [ClusterIP NodePort LoadBalancer] <br />Optional: \{\} <br /> |
| `serviceAnnotations` _object (keys:string, values:string)_ | ServiceAnnotations are annotations to add to the Service. |  | Optional: \{\} <br /> |
| `autoscaling` _[AutoscalingSpec](#autoscalingspec)_ | Autoscaling enables and configures a HorizontalPodAutoscaler for this deployment.<br />When set, the HPA manages the replica count; spec.deployment.replicas is used as<br />the initial desired count only and is no longer written on each reconcile. |  | Optional: \{\} <br /> |


#### DexConnector



DexConnector configures a Dex upstream identity provider connector.



_Appears in:_
- [DexSpec](#dexspec)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `type` _string_ | Type is the connector type: "github", "google", "oidc", "ldap", "microsoft", "saml", etc.<br />See https://dexidp.io/docs/connectors/ for the full list. |  | Required: \{\} <br /> |
| `id` _string_ | ID is the connector's unique identifier. |  | Required: \{\} <br /> |
| `name` _string_ | Name is the human-readable display name shown on the Dex login page. |  | Required: \{\} <br /> |
| `config` _object (keys:string, values:string)_ | Config contains connector-specific configuration key/value pairs. |  | Optional: \{\} <br /> |


#### DexSpec



DexSpec configures the embedded Dex OIDC provider.



_Appears in:_
- [ClusterOIDCSpec](#clusteroidcspec)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `connectors` _[DexConnector](#dexconnector) array_ | Connectors configures upstream identity providers (GitHub, Google, OIDC, etc.). |  | Optional: \{\} <br /> |
| `enablePasswordDB` _boolean_ | EnablePasswordDB enables Dex's built-in local password store.<br />When true, Dex presents a username/password login form backed by StaticPasswords.<br />This is independent of connectors — both can be active simultaneously. |  | Optional: \{\} <br /> |
| `staticPasswords` _[DexStaticPassword](#dexstaticpassword) array_ | StaticPasswords defines local user accounts for Dex's built-in password store.<br />Only used when EnablePasswordDB is true. |  | Optional: \{\} <br /> |
| `image` _string_ | Image overrides the Dex container image.<br />Defaults to the operator Helm chart's config.auth.dex.image value. |  | Optional: \{\} <br /> |


#### DexStaticPassword



DexStaticPassword defines a local user account in Dex's built-in password store.



_Appears in:_
- [DexSpec](#dexspec)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `email` _string_ | Email is the user's login email address. |  | Required: \{\} <br /> |
| `hash` _string_ | Hash is the bcrypt hash of the user's password.<br />Generate with: htpasswd -nbBC 10 "" <password> \| tr -d ':\n' \| sed 's/$2y/$2a/' |  | Required: \{\} <br /> |
| `username` _string_ | Username is the display name shown after login. |  | Optional: \{\} <br /> |
| `userID` _string_ | UserID is a stable unique identifier for this user. |  | Optional: \{\} <br /> |


#### ExecutionSpec



ExecutionSpec controls how an agent's workload is scheduled and run.

Agents run as Argo Workflows. The operator always renders a WorkflowTemplate
named after the agent; Mode decides what else is derived from it:

	service — a long-lived Workflow that never completes (the always-on agent).
	task    — a one-shot run, fired by a CronWorkflow when Schedule is set and/or
	          submitted manually against the WorkflowTemplate.



_Appears in:_
- [LanguageAgentRuntimeSpec](#languageagentruntimespec)
- [LanguageAgentSpec](#languageagentspec)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `mode` _string_ | Mode selects the execution model for this agent. | service | Enum: [service task] <br />Optional: \{\} <br /> |
| `schedule` _string_ | Schedule is a cron expression that fires a run. Only valid when mode is "task".<br />When unset, a task agent has no CronWorkflow and is invoked manually. |  | Optional: \{\} <br /> |
| `timezone` _string_ | Timezone is the IANA timezone the Schedule is evaluated in (e.g. "America/New_York").<br />Only valid alongside schedule. |  | Optional: \{\} <br /> |
| `activeDeadlineSeconds` _integer_ | ActiveDeadlineSeconds is the wall-clock limit for a single run.<br />Only valid when mode is "task" — a service agent is expected to run forever. |  | Minimum: 1 <br />Optional: \{\} <br /> |
| `ttlSecondsAfterFinished` _integer_ | TTLSecondsAfterFinished is how long a finished run is retained before Argo<br />garbage-collects it. Defaults to 86400 (24h). Only valid when mode is "task". |  | Minimum: 0 <br />Optional: \{\} <br /> |
| `concurrencyPolicy` _string_ | ConcurrencyPolicy decides what happens when a scheduled run is due while the<br />previous one is still going. Only valid alongside schedule. | Forbid | Enum: [Allow Forbid Replace] <br />Optional: \{\} <br /> |
| `suspend` _boolean_ | Suspend stops the agent from running: the CronWorkflow stops firing (task mode),<br />or the long-lived Workflow is torn down (service mode). The WorkflowTemplate is<br />left in place so the agent can still be invoked manually. |  | Optional: \{\} <br /> |
| `retryLimit` _integer_ | RetryLimit is the number of retries for a failed run. Only valid when mode is<br />"task"; a service agent always retries without limit so it stays up. |  | Minimum: 0 <br />Optional: \{\} <br /> |


#### GatewaySpec



GatewaySpec configures the shared LiteLLM gateway deployed per LanguageCluster.



_Appears in:_
- [LanguageClusterSpec](#languageclusterspec)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `deployment` _[DeploymentSpec](#deploymentspec)_ | Deployment configures the Kubernetes deployment for the gateway pod. |  | Optional: \{\} <br /> |


#### IngressConfig



IngressConfig defines ingress configuration



_Appears in:_
- [LanguageClusterSpec](#languageclusterspec)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `enabled` _boolean_ | Enabled controls whether an external Ingress is created for the shared gateway.<br />Defaults to false — the gateway is reachable in-cluster via its Service. Set to<br />true to expose it externally at gateway.<spec.domain>. |  | Optional: \{\} <br /> |
| `tls` _[IngressTLSConfig](#ingresstlsconfig)_ | TLS configuration for agent webhooks |  | Optional: \{\} <br /> |
| `className` _string_ | ClassName specifies the IngressClass to use (maps to spec.ingressClassName on the Ingress object). |  | Optional: \{\} <br /> |


#### IngressTLSConfig



IngressTLSConfig defines TLS configuration



_Appears in:_
- [IngressConfig](#ingressconfig)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `enabled` _boolean_ | Enabled controls whether TLS is enabled for webhooks.<br />Defaults to true; set to false to disable TLS. | true | Optional: \{\} <br /> |
| `secretName` _string_ | SecretName is the name of an existing TLS secret (bring-your-own certificate).<br />When set, cert-manager integration is skipped and this secret is used directly. |  | Optional: \{\} <br /> |


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
| `image` _string_ | Image is the default container image for agents using this runtime.<br />Agents may override this. When a runtime is referenced, spec.image on the agent is optional. |  | Optional: \{\} <br /> |
| `ports` _[AgentPort](#agentport) array_ | Ports defines default ports for agents using this runtime.<br />Replace semantics: when the agent defines spec.ports, runtime ports are ignored entirely. |  | Optional: \{\} <br /> |
| `deployment` _[DeploymentSpec](#deploymentspec)_ | Deployment provides default Kubernetes pod and container configuration.<br />Scalars (args, command, resources, probes, etc.) are used when the agent has none set.<br />Lists (initContainers, env, volumes, volumeMounts, envFrom) are runtime-first, agent-appended.<br />The name is historical: agents run as Argo Workflow pods, so replicas and<br />autoscaling have no effect for agents using this runtime. |  | Optional: \{\} <br /> |
| `execution` _[ExecutionSpec](#executionspec)_ | Execution provides the default execution model for agents using this runtime —<br />for example a runtime that only makes sense as a one-shot task can default<br />mode to "task". Agents override any field they set themselves. |  | Optional: \{\} <br /> |
| `credentials` _[CredentialSpec](#credentialspec) array_ | Credentials declares environment variables backed by Secret values that the<br />operator resolves and injects into agents using this runtime. Each entry is<br />auto-generated, set inline, or sourced from an existing Secret. Merged into the<br />agent's effective spec runtime-first; agent entries override by name. |  | Optional: \{\} <br /> |
| `auth` _[RuntimeAuthSpec](#runtimeauthspec)_ | Auth gates whether agents using this runtime sit behind the cluster OIDC proxy.<br />Effective only when the cluster has auth enabled (which provisions the OIDC<br />infrastructure). The OIDC connection itself is configured cluster-wide. |  | Optional: \{\} <br /> |




#### LanguageAgentSelfConfig



LanguageAgentSelfConfig is submitted by an agent pod to request runtime modifications
to its own LanguageAgent spec. The controller validates the request against the parent's
spec.selfConfigure allowlist before patching.





| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `apiVersion` _string_ | `langop.io/v1alpha1` | | |
| `kind` _string_ | `LanguageAgentSelfConfig` | | |
| `metadata` _[ObjectMeta](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.28/#objectmeta-v1-meta)_ | Refer to Kubernetes API documentation for fields of `metadata`. |  |  |
| `spec` _[LanguageAgentSelfConfigSpec](#languageagentselfconfigspec)_ |  |  |  |
| `status` _[LanguageAgentSelfConfigStatus](#languageagentselfconfigstatus)_ |  |  |  |


#### LanguageAgentSelfConfigSpec



LanguageAgentSelfConfigSpec defines the desired self-modification.



_Appears in:_
- [LanguageAgentSelfConfig](#languageagentselfconfig)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `instanceRef` _string_ | InstanceRef is the name of the LanguageAgent to modify. Must be in the same namespace. |  | MinLength: 1 <br />Required: \{\} <br /> |
| `addTools` _string array_ | AddTools lists LanguageTool names to append to spec.tools on the parent agent. |  | Optional: \{\} <br /> |
| `removeTools` _string array_ | RemoveTools lists LanguageTool names to remove from spec.tools on the parent agent. |  | Optional: \{\} <br /> |
| `addModels` _string array_ | AddModels lists LanguageModel names to append to spec.models on the parent agent. |  | Optional: \{\} <br /> |
| `removeModels` _string array_ | RemoveModels lists LanguageModel names to remove from spec.models on the parent agent. |  | Optional: \{\} <br /> |
| `addEnvVars` _[SelfConfigEnvVar](#selfconfigenvvar) array_ | AddEnvVars lists plain-value environment variables to inject into the parent agent's<br />spec.deployment.env. Existing vars with the same name are overwritten.<br />SecretKeyRef and other value sources are not supported. |  | Optional: \{\} <br /> |
| `updateInstructions` _string_ | UpdateInstructions, when non-empty, replaces spec.instructions on the parent agent. |  | Optional: \{\} <br /> |
| `addRoleRules` _[PolicyRule](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.28/#policyrule-v1-rbac) array_ | AddRoleRules lists RBAC policy rules to append to spec.deployment.roleRules on the<br />parent agent. Duplicate rules (identical APIGroups+Resources+Verbs) are de-duplicated. |  | Optional: \{\} <br /> |


#### LanguageAgentSelfConfigStatus



LanguageAgentSelfConfigStatus reflects the observed state of a self-config request.



_Appears in:_
- [LanguageAgentSelfConfig](#languageagentselfconfig)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `phase` _[SelfConfigPhase](#selfconfigphase)_ | Phase is the current processing state. |  | Enum: [Pending Applied Failed Denied] <br />Optional: \{\} <br /> |
| `message` _string_ | Message provides a human-readable explanation of the current phase. |  | Optional: \{\} <br /> |
| `completionTime` _[Time](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.28/#time-v1-meta)_ | CompletionTime is set when the request reaches a terminal phase (Applied, Failed, Denied).<br />The CR is automatically deleted 1 hour after this time. |  | Optional: \{\} <br /> |




#### LanguageAgentSpec



LanguageAgentSpec defines the desired state of LanguageAgent



_Appears in:_
- [LanguageAgent](#languageagent)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `runtime` _string_ | Runtime is the name of a LanguageAgentRuntime that provides preset configuration<br />(image, port, init containers, env vars, probes, etc.).<br />When set, spec.image is optional; the runtime provides a default. |  | Optional: \{\} <br /> |
| `image` _string_ | Image is the container image to run for this agent.<br />Required unless spec.runtime is set (the runtime provides a default image). |  | Pattern: `^([a-z0-9]+([._-][a-z0-9]+)*\/)*[a-z0-9]+([._-][a-z0-9]+)*(:[a-z0-9]+([._-][a-z0-9]+)*)?$` <br />Optional: \{\} <br /> |
| `models` _[ModelReference](#modelreference) array_ | Models is a list of LanguageModel references this agent can use |  | Optional: \{\} <br /> |
| `tools` _[ToolReference](#toolreference) array_ | Tools is a list of LanguageTool references available to this agent |  | Optional: \{\} <br /> |
| `persona` _string_ | Persona is the name of a LanguagePersona this agent uses |  | Optional: \{\} <br /> |
| `instructions` _string_ | Instructions provides system instructions for the agent.<br />Delivered as the top-level "instructions" field in /etc/agent/config.yaml. |  | Optional: \{\} <br /> |
| `workspace` _[WorkspaceSpec](#workspacespec)_ | Workspace defines persistent storage for the agent |  | Optional: \{\} <br /> |
| `repository` _[RepositorySpec](#repositoryspec)_ | Repository declares a git repository to clone into the agent's workspace.<br />When set, the operator ensures a workspace PVC is provisioned (defaulting it on<br />if not explicitly configured) so the clone has somewhere to land. |  | Optional: \{\} <br /> |
| `networkPolicies` _[AgentNetworkPolicies](#agentnetworkpolicies)_ | NetworkPolicies defines ingress and egress rules for this agent.<br />Rules mirror the native Kubernetes NetworkPolicy shape. |  | Optional: \{\} <br /> |
| `ports` _[AgentPort](#agentport) array_ | Ports defines all network ports this agent exposes.<br />At most one entry should have expose: true (the ingress target);<br />if none are marked, the first port is used for ingress routing.<br />Defaults to a single HTTP port on 8080 when not set. |  | Optional: \{\} <br /> |
| `deployment` _[DeploymentSpec](#deploymentspec)_ | Deployment groups Kubernetes-specific pod and container configuration.<br />The name is historical: agents run as Argo Workflow pods, not Deployments,<br />so spec.deployment.replicas and spec.deployment.autoscaling are rejected here. |  | Optional: \{\} <br /> |
| `execution` _[ExecutionSpec](#executionspec)_ | Execution controls how this agent's workload is scheduled and run —<br />always-on (mode: service) or invoked (mode: task). |  | Optional: \{\} <br /> |
| `credentials` _[CredentialSpec](#credentialspec) array_ | Credentials declares environment variables backed by Secret values that the<br />operator resolves and injects into the agent container. Typically supplied by<br />the referenced LanguageAgentRuntime; agents may add or override entries by name. |  | Optional: \{\} <br /> |
| `selfConfigure` _[SelfConfigureSpec](#selfconfigurespec)_ | SelfConfigure controls whether this agent may submit LanguageAgentSelfConfig<br />requests to modify its own spec at runtime. When enabled, the operator grants<br />the agent's ServiceAccount permission to create LanguageAgentSelfConfig resources. |  | Optional: \{\} <br /> |
| `monitoring` _[AgentMonitoringSpec](#agentmonitoringspec)_ | Monitoring configures Prometheus Operator integration for this agent.<br />When set, the operator creates a ServiceMonitor and/or PrometheusRule resource.<br />Requires prometheus-operator to be installed in the cluster; silently skipped otherwise. |  | Optional: \{\} <br /> |


#### LanguageAgentStatus



LanguageAgentStatus defines the observed state of LanguageAgent



_Appears in:_
- [LanguageAgent](#languageagent)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `phase` _string_ | Phase represents the current phase of the agent.<br />In service mode it tracks the long-lived Workflow; in task mode it mirrors<br />the most recent run. Suspended means spec.execution.suspend is set. |  | Enum: [Pending Running Succeeded Failed Suspended Degraded] <br />Optional: \{\} <br /> |
| `conditions` _[Condition](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.28/#condition-v1-meta) array_ | Conditions represent the latest available observations of the agent's state |  | Optional: \{\} <br /> |
| `workflowTemplateName` _string_ | WorkflowTemplateName is the Argo WorkflowTemplate rendered for this agent.<br />It is the unit submitted against for a manual run:<br />`argo submit --from workflowtemplate/<name> -n <namespace>`. |  | Optional: \{\} <br /> |
| `activeWorkflowName` _string_ | ActiveWorkflowName is the long-lived Workflow backing a service-mode agent.<br />Empty in task mode. |  | Optional: \{\} <br /> |
| `lastRunName` _string_ | LastRunName is the most recent Workflow run for this agent. |  | Optional: \{\} <br /> |
| `lastRunPhase` _string_ | LastRunPhase is the Argo phase of the most recent run<br />(Pending, Running, Succeeded, Failed, or Error). |  | Optional: \{\} <br /> |
| `lastRunStartedAt` _[Time](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.28/#time-v1-meta)_ | LastRunStartedAt is when the most recent run started. |  | Optional: \{\} <br /> |
| `lastRunFinishedAt` _[Time](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.28/#time-v1-meta)_ | LastRunFinishedAt is when the most recent run completed. Unset while it is running. |  | Optional: \{\} <br /> |
| `lastScheduledTime` _[Time](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.28/#time-v1-meta)_ | LastScheduledTime is when the CronWorkflow last fired a run. Task mode only. |  | Optional: \{\} <br /> |
| `uuid` _string_ | UUID is a unique identifier for this agent instance<br />Not used for webhook routing; webhooks are routed via agent name (e.g., <agent-name>.domain.com) |  | Optional: \{\} <br /> |
| `webhookURLs` _string array_ | WebhookURLs contains the URLs where this agent can receive webhooks |  | Optional: \{\} <br /> |
| `observedGeneration` _integer_ | ObservedGeneration is the most recent generation observed by the controller.<br />It corresponds to the metadata.generation of the LanguageAgent at the time<br />the controller last processed it. Watchers can use this to detect when the<br />status reflects a stale version of the spec. |  | Optional: \{\} <br /> |
| `workspacePVCName` _string_ | WorkspacePVCName is the name of the retained workspace PVC after agent deletion.<br />Only set when spec.workspace.retain is true. |  | Optional: \{\} <br /> |
| `managedResources` _[ManagedResource](#managedresource) array_ | ManagedResources is the inventory of Kubernetes resources created and owned<br />by this controller on behalf of this LanguageAgent.<br />The list is replaced atomically on every successful reconcile. |  | Optional: \{\} <br /> |




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
| `domain` _string_ | Domain is the base domain for the cluster and agent webhook routing.<br />Agent webhooks will be accessible at <agent-name>.<domain>.<br />Example: "ai.theryans.io" results in webhooks like "my-agent.ai.theryans.io" |  | Optional: \{\} <br /> |
| `ingress` _[IngressConfig](#ingressconfig)_ | Ingress defines ingress configuration for the cluster |  | Optional: \{\} <br /> |
| `networkPolicies` _[AgentNetworkPolicies](#agentnetworkpolicies)_ | NetworkPolicies defines ingress and egress rules for agents in this cluster.<br />Rules mirror the native Kubernetes NetworkPolicy shape. |  | Optional: \{\} <br /> |
| `gateway` _[GatewaySpec](#gatewayspec)_ | Gateway configures the shared LiteLLM gateway deployed per cluster |  | Optional: \{\} <br /> |
| `capacity` _[ClusterCapacitySpec](#clustercapacityspec)_ | Capacity declares hard limits enforced via a ResourceQuota in the cluster's namespace.<br />When set, the controller creates a ResourceQuota named "langop-quota".<br />When unset, any existing "langop-quota" is deleted. |  | Optional: \{\} <br /> |
| `auth` _[ClusterAuthSpec](#clusterauthspec)_ | Auth configures OIDC authentication for agent ingress routes in this cluster.<br />When enabled, a Dex OIDC provider is deployed alongside the gateway and each<br />LanguageAgent with auth enabled gets an oauth2-proxy in front of its ingress. |  | Optional: \{\} <br /> |


#### LanguageClusterStatus



LanguageClusterStatus defines the observed state



_Appears in:_
- [LanguageCluster](#languagecluster)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `phase` _string_ | Phase of the cluster |  | Enum: [Pending Ready Failed] <br /> |
| `conditions` _[Condition](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.28/#condition-v1-meta) array_ |  |  | Optional: \{\} <br /> |
| `gatewayEndpoint` _string_ | GatewayEndpoint is the in-cluster URL for the shared LiteLLM gateway |  | Optional: \{\} <br /> |
| `gatewayReady` _boolean_ | GatewayReady indicates whether the shared gateway Deployment is available.<br />Pointer distinguishes "not yet reconciled" (nil) from "known not ready" (false). |  | Optional: \{\} <br /> |
| `capacity` _[ClusterCapacityStatus](#clustercapacitystatus)_ | Capacity reports observed resource usage in this cluster's namespace. |  | Optional: \{\} <br /> |
| `observedGeneration` _integer_ | ObservedGeneration is the most recent generation observed by the controller.<br />It corresponds to the metadata.generation of the LanguageCluster at the time<br />the controller last processed it. Watchers can use this to detect when the<br />status reflects a stale version of the spec. |  | Optional: \{\} <br /> |
| `managedResources` _[ManagedResource](#managedresource) array_ | ManagedResources is the inventory of Kubernetes resources created and owned<br />by this controller on behalf of this LanguageCluster.<br />The list is replaced atomically on every successful reconcile. |  | Optional: \{\} <br /> |




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
| `endpoint` _string_ | Endpoint is the API endpoint URL (required for openai-compatible, azure, custom) |  | Optional: \{\} <br /> |
| `apiKeySecretRef` _[SecretReference](#secretreference)_ | APIKeySecretRef references a secret containing the API key |  | Optional: \{\} <br /> |
| `rateLimits` _[RateLimitSpec](#ratelimitspec)_ | RateLimits defines rate limiting configuration |  | Optional: \{\} <br /> |
| `timeout` _string_ | Timeout specifies request timeout duration (e.g., "5m", "30s") | 5m | Pattern: `^[0-9]+(ns\|us\|µs\|ms\|s\|m\|h)$` <br />Optional: \{\} <br /> |


#### LanguageModelStatus



LanguageModelStatus defines the observed state of LanguageModel



_Appears in:_
- [LanguageModel](#languagemodel)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `observedGeneration` _integer_ | ObservedGeneration reflects the generation of the most recently observed LanguageModel |  | Optional: \{\} <br /> |
| `phase` _string_ | Phase represents the current phase of the model (Pending, Ready, Failed) |  | Enum: [Pending Ready Failed] <br />Optional: \{\} <br /> |
| `conditions` _[Condition](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.28/#condition-v1-meta) array_ | Conditions represent the latest available observations of the model's state |  | Optional: \{\} <br /> |
| `message` _string_ | Message provides human-readable details about the current state |  | Optional: \{\} <br /> |




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
| `tone` _string_ | Tone describes the agent's communication style<br />e.g. "professional", "concise and direct", "warm and encouraging" |  | Optional: \{\} <br /> |
| `personality` _string_ | Personality describes the agent's character and behavioural traits<br />e.g. "curious and methodical, always explains reasoning step by step" |  | Optional: \{\} <br /> |
| `expertise` _string_ | Expertise describes the agent's domain knowledge and skills<br />e.g. "senior software engineer specialising in distributed systems and Go" |  | Optional: \{\} <br /> |


#### LanguagePersonaStatus



LanguagePersonaStatus defines the observed state of LanguagePersona



_Appears in:_
- [LanguagePersona](#languagepersona)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `observedGeneration` _integer_ | ObservedGeneration reflects the generation of the most recently observed LanguagePersona |  | Optional: \{\} <br /> |
| `phase` _string_ | Phase represents the current phase (Pending, Ready, Failed) |  | Enum: [Pending Ready Failed] <br />Optional: \{\} <br /> |
| `conditions` _[Condition](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.28/#condition-v1-meta) array_ | Conditions represent the latest available observations of the persona's state |  | Optional: \{\} <br /> |




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
| `image` _string_ | Image is the container image to run for this tool. For transport=stdio it is ignored —<br />the operator injects the MCP bridge image instead — and may be omitted (the defaulting<br />webhook fills it). |  | MinLength: 1 <br />Required: \{\} <br /> |
| `type` _string_ | Type specifies the tool protocol type. Only "mcp" is currently implemented. | mcp | Enum: [mcp] <br /> |
| `transport` _string_ | Transport selects how the operator exposes this tool's MCP endpoint.<br />- "streamable-http" (default): spec.image already serves Streamable HTTP at /mcp.<br />- "sse": spec.image already serves the (legacy) MCP HTTP+SSE transport.<br />- "stdio": the user supplies a stdio MCP command in spec.stdio; the operator injects a<br />  pinned, persistent stdio→Streamable-HTTP bridge that serves /mcp and /health on spec.port. | streamable-http | Enum: [streamable-http sse stdio] <br />Optional: \{\} <br /> |
| `stdio` _[StdioServerSpec](#stdioserverspec)_ | Stdio configures the stdio MCP server when transport=stdio. Required for that transport,<br />ignored otherwise. |  | Optional: \{\} <br /> |
| `deploymentMode` _string_ | DeploymentMode specifies how this tool should be deployed<br />- "service": Deployed as a standalone Deployment+Service (default, shared across agents)<br />- "sidecar": Deployed as a sidecar container in each agent pod (dedicated, with workspace access) | service | Enum: [service sidecar] <br />Optional: \{\} <br /> |
| `port` _integer_ | Port is the port the tool listens on | 8080 | Maximum: 65535 <br />Minimum: 1 <br /> |
| `deployment` _[DeploymentSpec](#deploymentspec)_ | Deployment groups Kubernetes-specific pod and container configuration. |  | Optional: \{\} <br /> |
| `networkPolicies` _[AgentNetworkPolicies](#agentnetworkpolicies)_ | NetworkPolicies defines ingress and egress rules for this tool.<br />Rules mirror the native Kubernetes NetworkPolicy shape. |  | Optional: \{\} <br /> |


#### LanguageToolStatus



LanguageToolStatus defines the observed state of LanguageTool



_Appears in:_
- [LanguageTool](#languagetool)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `observedGeneration` _integer_ | ObservedGeneration reflects the generation of the most recently observed LanguageTool |  | Optional: \{\} <br /> |
| `phase` _string_ | Phase represents the current phase of the tool (Pending, Running, Failed, Updating, Degraded) |  | Enum: [Pending Running Failed Updating Degraded] <br />Optional: \{\} <br /> |
| `conditions` _[Condition](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.28/#condition-v1-meta) array_ | Conditions represent the latest available observations of the tool's state |  | Optional: \{\} <br /> |
| `endpoint` _string_ | Endpoint is the service endpoint where the tool is accessible |  | Optional: \{\} <br /> |
| `toolSchemas` _[ToolSchema](#toolschema) array_ | ToolSchemas contains the complete MCP tool schemas discovered from this service |  | Optional: \{\} <br /> |
| `readyReplicas` _integer_ | ReadyReplicas is the number of pods ready and passing health checks |  | Optional: \{\} <br /> |
| `availableReplicas` _integer_ | AvailableReplicas is the number of pods targeted by this LanguageTool with at least one available condition |  | Optional: \{\} <br /> |
| `updatedReplicas` _integer_ | UpdatedReplicas is the number of pods targeted by this LanguageTool that have the desired spec |  | Optional: \{\} <br /> |
| `unavailableReplicas` _integer_ | UnavailableReplicas is the number of pods targeted by this LanguageTool that are unavailable |  | Optional: \{\} <br /> |




#### ManagedResource



ManagedResource describes a single Kubernetes resource created and owned
by this operator on behalf of a LanguageAgent or LanguageCluster.
The combination of Group+Kind+Namespace+Name uniquely identifies the resource.



_Appears in:_
- [LanguageAgentStatus](#languageagentstatus)
- [LanguageClusterStatus](#languageclusterstatus)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `group` _string_ | Group is the API group of the resource.<br />Empty string means the core API group (e.g. ConfigMap, Service, PersistentVolumeClaim). |  | Optional: \{\} <br /> |
| `kind` _string_ | Kind is the Kubernetes resource kind (e.g. "Deployment", "ConfigMap"). |  |  |
| `name` _string_ | Name is the name of the resource. |  |  |
| `namespace` _string_ | Namespace is the namespace of the resource.<br />Empty for cluster-scoped resources (e.g. Namespace). |  | Optional: \{\} <br /> |


#### ModelReference



ModelReference references a LanguageModel



_Appears in:_
- [LanguageAgentSpec](#languageagentspec)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `name` _string_ | Name is the name of the LanguageModel |  | MaxLength: 63 <br />Pattern: `^[a-z0-9]([-a-z0-9]*[a-z0-9])?$` <br />Required: \{\} <br /> |
| `role` _string_ | Role defines the purpose of this model — a hint for the agent runtime for model selection<br />(e.g. prefer role=primary for general calls, role=reasoning for chain-of-thought).<br />The operator does not enforce routing by role; it is surfaced in the agent config (agent.json). | primary | Enum: [primary fallback reasoning tool-calling summarization] <br />Optional: \{\} <br /> |
| `priority` _integer_ | Priority for model selection — a hint for the agent runtime (lower value = higher priority).<br />The operator does not enforce priority; it is surfaced in the agent config (agent.json). |  | Optional: \{\} <br /> |


#### NetworkEgressRule



NetworkEgressRule is one egress rule: zero or more destination peers, zero or more ports.
Mirrors networkingv1.NetworkPolicyEgressRule.



_Appears in:_
- [AgentNetworkPolicies](#agentnetworkpolicies)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `to` _[NetworkPeer](#networkpeer) array_ | To lists the destinations this workload is allowed to reach. |  | Optional: \{\} <br /> |
| `ports` _[NetworkPort](#networkport) array_ | Ports lists the destination ports to allow. |  | Optional: \{\} <br /> |


#### NetworkIngressRule



NetworkIngressRule is one ingress rule: zero or more source peers, zero or more ports.
Mirrors networkingv1.NetworkPolicyIngressRule.



_Appears in:_
- [AgentNetworkPolicies](#agentnetworkpolicies)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `from` _[NetworkPeer](#networkpeer) array_ | From lists the sources allowed to send traffic to this workload. |  | Optional: \{\} <br /> |
| `ports` _[NetworkPort](#networkport) array_ | Ports lists the ports on which to allow incoming traffic. |  | Optional: \{\} <br /> |


#### NetworkPeer



NetworkPeer defines the source/destination of network traffic



_Appears in:_
- [NetworkEgressRule](#networkegressrule)
- [NetworkIngressRule](#networkingressrule)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `group` _string_ | Group selects pods with matching langop.io/group label<br />Used to allow communication with specific labeled resources |  | Optional: \{\} <br /> |
| `cidr` _string_ | CIDR block |  | Optional: \{\} <br /> |
| `dns` _string array_ | DNS names (supports wildcards with *)<br />Examples: "api.openai.com", "*.googleapis.com" |  | Optional: \{\} <br /> |
| `service` _[ServiceReference](#servicereference)_ | Kubernetes service reference |  | Optional: \{\} <br /> |
| `namespaceSelector` _[LabelSelector](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.28/#labelselector-v1-meta)_ | Namespace selector (for cross-namespace rules) |  | Optional: \{\} <br /> |
| `podSelector` _[LabelSelector](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.28/#labelselector-v1-meta)_ | Pod selector (within namespace) |  | Optional: \{\} <br /> |


#### NetworkPort



NetworkPort defines a port and protocol



_Appears in:_
- [NetworkEgressRule](#networkegressrule)
- [NetworkIngressRule](#networkingressrule)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `protocol` _string_ | Protocol (TCP, UDP, SCTP) | TCP | Enum: [TCP UDP SCTP] <br />Optional: \{\} <br /> |
| `port` _integer_ | Port number |  | Maximum: 65535 <br />Minimum: 1 <br /> |


#### PrometheusAlertingRule



PrometheusAlertingRule defines a single Prometheus alerting or recording rule.



_Appears in:_
- [PrometheusRuleGroup](#prometheusrulegroup)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `alert` _string_ | Alert is the alert name. Leave empty for recording rules. |  | Optional: \{\} <br /> |
| `record` _string_ | Record is the output metric name for recording rules. Leave empty for alerting rules. |  | Optional: \{\} <br /> |
| `expr` _string_ | Expr is the PromQL expression evaluated at each evaluation cycle. |  | Required: \{\} <br /> |
| `for` _string_ | For is the duration the condition must be true before the alert fires.<br />Only valid for alerting rules. |  | Optional: \{\} <br /> |
| `labels` _object (keys:string, values:string)_ | Labels are labels attached to the alert or recording rule. |  | Optional: \{\} <br /> |
| `annotations` _object (keys:string, values:string)_ | Annotations are annotations attached to the alert. Only valid for alerting rules. |  | Optional: \{\} <br /> |


#### PrometheusRuleGroup



PrometheusRuleGroup defines a group of Prometheus alerting or recording rules.



_Appears in:_
- [AgentMonitoringSpec](#agentmonitoringspec)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `name` _string_ | Name is the name of the rule group. |  | Required: \{\} <br /> |
| `interval` _string_ | Interval is the evaluation interval for this group. Uses the Prometheus default when omitted. |  | Optional: \{\} <br /> |
| `rules` _[PrometheusAlertingRule](#prometheusalertingrule) array_ | Rules is the list of alerting or recording rules in this group. |  | MinItems: 1 <br /> |


#### RateLimitSpec



RateLimitSpec defines rate limiting configuration



_Appears in:_
- [LanguageModelSpec](#languagemodelspec)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `requestsPerMinute` _integer_ | RequestsPerMinute limits requests per minute |  | Optional: \{\} <br /> |
| `tokensPerMinute` _integer_ | TokensPerMinute limits tokens per minute |  | Optional: \{\} <br /> |


#### RepositorySpec



RepositorySpec declares a git repository to clone into the agent's workspace.
The clone is performed by the operator at pod startup; this type defines only the
desired source. Follows the WorkspaceSpec/CredentialSpec conventions.



_Appears in:_
- [LanguageAgentSpec](#languageagentspec)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `url` _string_ | URL is the git repository to clone (https://... or git@... SSH). |  | Required: \{\} <br /> |
| `ref` _string_ | Ref is the branch, tag, or commit SHA to check out. Defaults to the default branch. |  | Optional: \{\} <br /> |
| `path` _string_ | Path is the subdirectory under the workspace mountPath to clone into.<br />Defaults to the repository name derived from the URL. Must be a relative path<br />(no leading "/", no ".." segments). |  | Optional: \{\} <br /> |
| `depth` _integer_ | Depth, when > 0, performs a shallow clone with this history depth. |  | Minimum: 0 <br />Optional: \{\} <br /> |
| `secretRef` _[LocalObjectReference](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.28/#localobjectreference-v1-core)_ | SecretRef references a Secret with git credentials for private repos.<br />Recognized keys: `token` or `username`+`password` (HTTPS), `ssh-privatekey` (SSH). |  | Optional: \{\} <br /> |


#### RuntimeAuthSpec



RuntimeAuthSpec gates OIDC authentication for agents using a runtime.



_Appears in:_
- [LanguageAgentRuntimeSpec](#languageagentruntimespec)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `enabled` _boolean_ | Enabled puts agents using this runtime behind the cluster OIDC proxy.<br />Has no effect unless the cluster has auth enabled. |  | Optional: \{\} <br /> |


#### RuntimeSecretRef



RuntimeSecretRef references a Secret in the same namespace.
All keys in the Secret are injected as env vars via envFrom.



_Appears in:_
- [CredentialSpec](#credentialspec)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `name` _string_ | Name is the name of the Secret. |  | Required: \{\} <br /> |


#### SecretReference



SecretReference references a Kubernetes Secret



_Appears in:_
- [ClusterOIDCSpec](#clusteroidcspec)
- [LanguageModelSpec](#languagemodelspec)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `name` _string_ | Name is the name of the secret |  | Required: \{\} <br /> |
| `key` _string_ | Key is the key within the secret containing the value | api-key | Optional: \{\} <br /> |


#### SelfConfigAction

_Underlying type:_ _string_

SelfConfigAction is a category of self-modification an agent may request.

_Validation:_
- Enum: [tools models envVars instructions roleRules]

_Appears in:_
- [SelfConfigureSpec](#selfconfigurespec)

| Field | Description |
| --- | --- |
| `tools` | SelfConfigActionTools allows the agent to add or remove tool references.<br /> |
| `models` | SelfConfigActionModels allows the agent to add or remove model references.<br /> |
| `envVars` | SelfConfigActionEnvVars allows the agent to inject plain-value environment variables.<br /> |
| `instructions` | SelfConfigActionInstructions allows the agent to replace its system instructions.<br /> |
| `roleRules` | SelfConfigActionRoleRules allows the agent to append RBAC policy rules to its Role.<br /> |


#### SelfConfigEnvVar



SelfConfigEnvVar is a plain-value environment variable injected by a self-config request.
SecretKeyRef and other value sources are intentionally not supported.



_Appears in:_
- [LanguageAgentSelfConfigSpec](#languageagentselfconfigspec)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `name` _string_ | Name of the environment variable. |  | MinLength: 1 <br />Required: \{\} <br /> |
| `value` _string_ | Value is the literal string value of the variable. |  | Required: \{\} <br /> |


#### SelfConfigPhase

_Underlying type:_ _string_

SelfConfigPhase reflects the current processing state of a LanguageAgentSelfConfig.

_Validation:_
- Enum: [Pending Applied Failed Denied]

_Appears in:_
- [LanguageAgentSelfConfigStatus](#languageagentselfconfigstatus)

| Field | Description |
| --- | --- |
| `Pending` | SelfConfigPhasePending means the request has not yet been processed.<br /> |
| `Applied` | SelfConfigPhaseApplied means the requested changes were patched onto the parent LanguageAgent.<br /> |
| `Failed` | SelfConfigPhaseFailed means the controller encountered an error while processing the request.<br /> |
| `Denied` | SelfConfigPhaseDenied means the request was rejected due to policy (not enabled, or action not allowed).<br /> |


#### SelfConfigureSpec



SelfConfigureSpec controls whether a LanguageAgent may submit LanguageAgentSelfConfig
requests to modify its own spec at runtime.



_Appears in:_
- [LanguageAgentSpec](#languageagentspec)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `enabled` _boolean_ | Enabled gates all self-configuration. When false, any LanguageAgentSelfConfig<br />targeting this agent is immediately Denied. | false | Optional: \{\} <br /> |
| `allowedActions` _[SelfConfigAction](#selfconfigaction) array_ | AllowedActions is the allowlist of self-config categories the agent may request.<br />If empty while Enabled=true, all actions are denied. |  | Enum: [tools models envVars instructions roleRules] <br />Optional: \{\} <br /> |


#### ServiceReference



ServiceReference identifies a Kubernetes Service



_Appears in:_
- [NetworkPeer](#networkpeer)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `name` _string_ | Service name |  | Required: \{\} <br /> |
| `namespace` _string_ | Service namespace (defaults to same namespace if omitted) |  | Optional: \{\} <br /> |


#### StdioServerSpec



StdioServerSpec describes a stdio-based MCP server the operator wraps with a persistent
stdio→Streamable-HTTP bridge.



_Appears in:_
- [LanguageToolSpec](#languagetoolspec)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `command` _string array_ | Command is the full argv of the stdio MCP server, e.g.<br />["npx","-y","@upstash/context7-mcp"] or ["uvx","mcp-server-git","--repository","/workspace"].<br />The operator passes it to the bridge as a single stdio command. Environment for the<br />child comes from spec.deployment.env / spec.deployment.envFrom. |  | MinItems: 1 <br />Required: \{\} <br /> |


#### ToolProperty



ToolProperty defines an individual parameter or return field



_Appears in:_
- [ToolSchemaDefinition](#toolschemadefinition)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `type` _string_ | Type is the JSON schema type (string, integer, boolean, etc.) |  |  |
| `description` _string_ | Description explains what this property represents |  | Optional: \{\} <br /> |
| `example` _string_ | Example provides an example value as a JSON string |  | Optional: \{\} <br /> |


#### ToolReference



ToolReference references a LanguageTool



_Appears in:_
- [LanguageAgentSpec](#languageagentspec)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `name` _string_ | Name is the name of the LanguageTool |  | MaxLength: 63 <br />Pattern: `^[a-z0-9]([-a-z0-9]*[a-z0-9])?$` <br />Required: \{\} <br /> |
| `enabled` _boolean_ | Enabled indicates if this tool is available to the agent.<br />Defaults to true. Set to false to explicitly disable the tool without removing it. | true | Optional: \{\} <br /> |


#### ToolSchema



ToolSchema represents the complete schema of an MCP tool



_Appears in:_
- [LanguageToolStatus](#languagetoolstatus)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `name` _string_ | Name is the tool identifier |  |  |
| `description` _string_ | Description is a human-readable description of the tool |  | Optional: \{\} <br /> |
| `inputSchema` _[ToolSchemaDefinition](#toolschemadefinition)_ | InputSchema defines the parameters this tool accepts |  | Optional: \{\} <br /> |


#### ToolSchemaDefinition



ToolSchemaDefinition defines parameter or return value structure



_Appears in:_
- [ToolSchema](#toolschema)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `type` _string_ | Type is the JSON schema type (object, array, string, etc.) |  | Optional: \{\} <br /> |
| `properties` _object (keys:string, values:[ToolProperty](#toolproperty))_ | Properties defines object properties (for type: object) |  | Optional: \{\} <br /> |
| `required` _string array_ | Required lists required property names (for type: object) |  | Optional: \{\} <br /> |


#### WorkspaceSpec



WorkspaceSpec defines persistent workspace storage for an agent



_Appears in:_
- [LanguageAgentSpec](#languageagentspec)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `enabled` _boolean_ | Enabled controls whether to create a workspace volume.<br />Defaults to true. Set to false to explicitly disable without removing the workspace config. | true | Optional: \{\} <br /> |
| `size` _string_ | Size is the requested storage size (e.g., "10Gi", "1.5Ti", "500Mi")<br />Supports integer and decimal quantities with standard Kubernetes suffixes | 10Gi | MinLength: 1 <br />Pattern: `^([0-9]*\.?[0-9]+)(Ei\|Pi\|Ti\|Gi\|Mi\|Ki\|E\|P\|T\|G\|M\|K\|m)?$` <br />Optional: \{\} <br /> |
| `storageClassName` _string_ | StorageClassName specifies the StorageClass for the PVC<br />If not specified, uses the cluster default |  | Optional: \{\} <br /> |
| `accessMode` _string_ | AccessMode defines the volume access mode | ReadWriteOnce | Enum: [ReadWriteOnce ReadWriteMany] <br />Optional: \{\} <br /> |
| `mountPath` _string_ | MountPath is where the workspace is mounted in containers | /workspace | Optional: \{\} <br /> |
| `retain` _boolean_ | Retain prevents the workspace PVC from being deleted when the agent is deleted.<br />When true, the PVC's ownerReference is removed during cleanup so Kubernetes GC<br />does not collect it. The orphaned PVC name is surfaced in status.workspacePVCName.<br />Defaults to false. | false | Optional: \{\} <br /> |
| `initialFiles` _object (keys:string, values:string)_ | InitialFiles are seeded into the workspace PVC on first boot only.<br />Keys are filenames (must be valid ConfigMap keys: alphanumeric, '.', '-', '_').<br />Files are not overwritten if they already exist on the PVC. |  | Optional: \{\} <br /> |
| `seedConfigMapRef` _[LocalObjectReference](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.28/#localobjectreference-v1-core)_ | SeedConfigMapRef references an external ConfigMap whose keys are filenames<br />and values are file contents. Merged with InitialFiles; InitialFiles wins on collision. |  | Optional: \{\} <br /> |


