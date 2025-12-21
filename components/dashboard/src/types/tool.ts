// Generated TypeScript types for LanguageTool CRD

import { V1ObjectMeta, V1Affinity, V1ResourceRequirements, V1Toleration, V1PodSecurityContext, V1SecurityContext } from '@kubernetes/client-node'

// LanguageTool CRD Types
export interface LanguageTool {
  apiVersion: 'langop.io/v1alpha1'
  kind: 'LanguageTool'
  metadata: V1ObjectMeta
  spec: LanguageToolSpec
  status?: LanguageToolStatus
}

export interface LanguageToolList {
  apiVersion: 'langop.io/v1alpha1'
  kind: 'LanguageToolList'
  metadata: V1ObjectMeta
  items: LanguageTool[]
}

export interface LanguageToolSpec {
  // Tool type and deployment
  type: 'container' | 'function' | 'webhook' | 'builtin'
  image?: string
  description?: string
  
  // Cluster reference for scoping
  clusterRef?: string
  
  // Tool definition
  schema?: ToolSchema
  webhook?: any
  container?: any
  function?: any
  security?: any
  
  // Container configuration (for container type)
  command?: string[]
  args?: string[]
  env?: EnvironmentVariable[]
  
  // Resource management
  resources?: V1ResourceRequirements
  
  // Kubernetes scheduling
  affinity?: V1Affinity
  tolerations?: V1Toleration[]
  nodeSelector?: Record<string, string>
  
  // Security
  securityContext?: V1SecurityContext
  podSecurityContext?: V1PodSecurityContext
  
  // Service configuration
  service?: ServiceConfiguration
  
  // Health check configuration
  healthCheck?: HealthCheckConfiguration
  
  // Networking
  networking?: NetworkingConfiguration
  
  // Authentication
  authentication?: AuthenticationConfiguration
  
  // Scaling
  scaling?: ScalingConfiguration
  
  // External dependencies
  dependencies?: DependencyConfiguration[]
  
  // Configuration
  configuration?: Record<string, any>
}

export interface ToolSchema {
  name: string
  description?: string
  version?: string
  
  // Function schema
  parameters?: ToolParameter[]
  returns?: ToolReturnType
  inputSchema?: any
  outputSchema?: any
  
  // Examples
  examples?: ToolExample[]
  
  // Metadata
  tags?: string[]
  category?: string
  author?: string
  documentation?: string
}

export interface ToolParameter {
  name: string
  type: 'string' | 'number' | 'boolean' | 'object' | 'array'
  description?: string
  required?: boolean
  default?: any
  enum?: string[]
  pattern?: string
  minimum?: number
  maximum?: number
  items?: ToolParameter
  properties?: Record<string, ToolParameter>
}

export interface ToolReturnType {
  type: 'string' | 'number' | 'boolean' | 'object' | 'array'
  description?: string
  properties?: Record<string, ToolParameter>
  items?: ToolParameter
}

export interface ToolExample {
  name: string
  description?: string
  input: Record<string, any>
  output: any
}

export interface EnvironmentVariable {
  name: string
  value?: string
  valueFrom?: EnvironmentVariableSource
}

export interface EnvironmentVariableSource {
  secretKeyRef?: SecretKeyRef
  configMapKeyRef?: ConfigMapKeyRef
  fieldRef?: FieldRef
}

export interface SecretKeyRef {
  name: string
  key: string
}

export interface ConfigMapKeyRef {
  name: string
  key: string
}

export interface FieldRef {
  fieldPath: string
}

export interface ServiceConfiguration {
  type?: 'ClusterIP' | 'NodePort' | 'LoadBalancer'
  port?: number
  targetPort?: number
  protocol?: 'TCP' | 'UDP'
  annotations?: Record<string, string>
}

export interface HealthCheckConfiguration {
  enabled?: boolean
  path?: string
  port?: number
  protocol?: 'HTTP' | 'HTTPS' | 'TCP'
  initialDelaySeconds?: number
  periodSeconds?: number
  timeoutSeconds?: number
  failureThreshold?: number
  successThreshold?: number
}

export interface NetworkingConfiguration {
  ingress?: IngressConfiguration
  networkPolicy?: NetworkPolicyConfiguration
}

export interface IngressConfiguration {
  enabled?: boolean
  host?: string
  path?: string
  tls?: boolean
  annotations?: Record<string, string>
}

export interface NetworkPolicyConfiguration {
  enabled?: boolean
  ingress?: NetworkPolicyRule[]
  egress?: NetworkPolicyRule[]
}

export interface NetworkPolicyRule {
  from?: NetworkPolicyPeer[]
  to?: NetworkPolicyPeer[]
  ports?: NetworkPolicyPort[]
}

export interface NetworkPolicyPeer {
  podSelector?: Record<string, string>
  namespaceSelector?: Record<string, string>
  ipBlock?: IPBlock
}

export interface IPBlock {
  cidr: string
  except?: string[]
}

export interface NetworkPolicyPort {
  protocol?: 'TCP' | 'UDP' | 'SCTP'
  port?: number | string
  endPort?: number
}

export interface AuthenticationConfiguration {
  enabled?: boolean
  type?: 'bearer' | 'basic' | 'apikey' | 'oauth2'
  secretRef?: SecretReference
}

export interface SecretReference {
  name: string
  key?: string
}

export interface ScalingConfiguration {
  replicas?: number
  minReplicas?: number
  maxReplicas?: number
  targetCPUUtilization?: number
  targetMemoryUtilization?: number
}

export interface DependencyConfiguration {
  name: string
  type: 'service' | 'database' | 'api'
  endpoint?: string
  version?: string
  required?: boolean
}

// Status types
export interface LanguageToolStatus {
  phase?: 'Pending' | 'Running' | 'Failed' | 'Unknown'
  conditions?: LanguageToolCondition[]
  endpoint?: string
  replicas?: number
  readyReplicas?: number
  metrics?: LanguageToolMetrics
  lastHealthCheck?: string
  observedGeneration?: number
  endpointStatus?: any
}

export interface LanguageToolCondition {
  type: string
  status: 'True' | 'False' | 'Unknown'
  lastTransitionTime?: string
  lastUpdateTime?: string
  reason?: string
  message?: string
}

export interface LanguageToolMetrics {
  invocationCount?: number
  successfulInvocations?: number
  failedInvocations?: number
  averageLatency?: string
  averageDuration?: number
  successRate?: string
  errorRate?: string
  lastInvocation?: string
}

// Frontend-specific types
export interface LanguageToolFormData {
  name: string
  namespace: string
  type: 'container' | 'function' | 'webhook' | 'builtin'
  image?: string
  
  // Tool definition
  toolName: string
  description?: string
  version?: string
  category?: string
  
  // Container configuration
  command?: string[]
  args?: string[]
  envVars?: { name: string; value: string }[]
  
  // Resources
  cpuRequest?: string
  memoryRequest?: string
  cpuLimit?: string
  memoryLimit?: string
  
  // Service
  enableService?: boolean
  servicePort?: number
  serviceType?: 'ClusterIP' | 'NodePort' | 'LoadBalancer'
  
  // Health checks
  enableHealthCheck?: boolean
  healthCheckPath?: string
  healthCheckPort?: number
  
  // Scaling
  replicas?: number
  minReplicas?: number
  maxReplicas?: number
  targetCPUUtilization?: number
  
  // Authentication
  enableAuth?: boolean
  authType?: 'bearer' | 'basic' | 'apikey' | 'oauth2'
  authSecretName?: string
  
  // Networking
  enableIngress?: boolean
  ingressHost?: string
  ingressPath?: string
  enableTLS?: boolean
  
  // Schema
  parameters?: ToolParameterFormData[]
  returnType?: string
  examples?: ToolExampleFormData[]
  
  // Network policy fields
  egressRules?: Array<{
    description?: string
    dns?: string[]
    cidr?: string
    ports?: Array<{
      port: number
      protocol: 'TCP' | 'UDP'
    }>
  }>
  
  // Ingress policy fields - tools accept connections from agents
  ingressRules?: Array<{
    description?: string
    from: 'agents' | 'tools' | 'models' | 'cluster' | 'external' | 'gateway'
    ports?: Array<{
      port: number
      protocol: 'TCP' | 'UDP'
    }>
  }>
}

export interface ToolParameterFormData {
  name: string
  type: 'string' | 'number' | 'boolean' | 'object' | 'array'
  description?: string
  required?: boolean
  default?: any
}

export interface ToolExampleFormData {
  name: string
  description?: string
  input: string // JSON string
  output: string // JSON string
}

export interface LanguageToolListItem {
  name: string
  namespace: string
  type: string
  image?: string
  phase?: string
  endpoint?: string
  invocations?: number
  successRate?: string
  age: string
  creationTimestamp: string
}

// API response types
export interface LanguageToolResponse {
  success: boolean
  data?: LanguageTool
  error?: string
}

export interface LanguageToolListResponse {
  success: boolean
  data?: LanguageTool[]
  error?: string
  total?: number
  page?: number
  limit?: number
}

// Query parameters for listing tools
export interface LanguageToolListParams {
  namespace?: string
  labelSelector?: string
  fieldSelector?: string
  page?: number
  limit?: number
  sortBy?: 'name' | 'type' | 'phase' | 'invocations' | 'age'
  sortOrder?: 'asc' | 'desc'
  search?: string
  type?: string[]
  phase?: string[]
  category?: string[]
}