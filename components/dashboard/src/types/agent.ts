// Generated TypeScript types for LanguageAgent CRD

import { V1ObjectMeta, V1Affinity, V1ResourceRequirements, V1Toleration, V1PodSecurityContext, V1SecurityContext } from '@kubernetes/client-node'

// LanguageAgent CRD Types
export interface LanguageAgent {
  apiVersion: 'langop.io/v1alpha1'
  kind: 'LanguageAgent'
  metadata: V1ObjectMeta
  spec: LanguageAgentSpec
  status?: LanguageAgentStatus
}

export interface LanguageAgentList {
  apiVersion: 'langop.io/v1alpha1'
  kind: 'LanguageAgentList'
  metadata: V1ObjectMeta
  items: LanguageAgent[]
}

export interface LanguageAgentSpec {
  // Execution configuration
  executionMode: 'worker' | 'server' | 'hybrid'
  replicas?: number
  
  // Model configuration
  model: LanguageModelConfig
  
  // Persona configuration
  persona?: LanguagePersonaConfig
  
  // Tools configuration
  tools?: LanguageToolConfig[]
  
  // Resource management
  resources?: V1ResourceRequirements
  
  // Kubernetes scheduling
  affinity?: V1Affinity
  tolerations?: V1Toleration[]
  nodeSelector?: Record<string, string>
  
  // Security
  securityContext?: V1SecurityContext
  podSecurityContext?: V1PodSecurityContext
  
  // Advanced configuration
  scaling?: ScalingConfig
  monitoring?: MonitoringConfig
  networking?: NetworkingConfig
}

export interface LanguageModelConfig {
  name: string
  provider?: string
  endpoint?: string
  parameters?: Record<string, any>
  credentials?: CredentialReference
}

export interface LanguagePersonaConfig {
  name: string
  tone?: string
  instructions?: string
  examples?: string[]
}

export interface LanguageToolConfig {
  name: string
  type?: string
  endpoint?: string
  schema?: Record<string, any>
  timeout?: string
}

export interface ScalingConfig {
  minReplicas?: number
  maxReplicas?: number
  targetCPUUtilization?: number
  targetMemoryUtilization?: number
  scaleDownDelay?: string
  scaleUpDelay?: string
}

export interface MonitoringConfig {
  enabled?: boolean
  metricsPort?: number
  healthCheckPath?: string
  readinessCheckPath?: string
  livenessCheckPath?: string
}

export interface NetworkingConfig {
  port?: number
  protocol?: string
  ingress?: IngressConfig
  service?: ServiceConfig
}

export interface IngressConfig {
  enabled?: boolean
  host?: string
  path?: string
  tls?: boolean
  annotations?: Record<string, string>
}

export interface ServiceConfig {
  type?: 'ClusterIP' | 'NodePort' | 'LoadBalancer'
  port?: number
  annotations?: Record<string, string>
}

export interface CredentialReference {
  secretName: string
  secretKey: string
}

// Status types
export interface LanguageAgentStatus {
  phase: 'Pending' | 'Running' | 'Succeeded' | 'Failed' | 'Unknown'
  conditions?: LanguageAgentCondition[]
  activeReplicas?: number
  readyReplicas?: number
  executionCount?: number
  lastExecution?: string
  metrics?: LanguageAgentMetrics
  observedGeneration?: number
}

export interface LanguageAgentCondition {
  type: string
  status: 'True' | 'False' | 'Unknown'
  lastTransitionTime?: string
  lastUpdateTime?: string
  reason?: string
  message?: string
}

export interface LanguageAgentMetrics {
  successRate?: string
  averageLatency?: string
  totalRequests?: number
  errorRate?: string
  costMetrics?: CostMetrics
}

export interface CostMetrics {
  totalCost?: string
  costPerExecution?: string
  currency?: string
  billingPeriod?: string
}

// Frontend-specific types
export interface LanguageAgentFormData {
  name: string
  namespace: string
  executionMode: 'worker' | 'server' | 'hybrid'
  replicas: number
  
  // Model selection
  modelName: string
  modelProvider?: string
  modelEndpoint?: string
  modelParameters?: Record<string, any>
  
  // Persona selection
  personaName?: string
  personaTone?: string
  personaInstructions?: string
  
  // Tools selection
  selectedTools: string[]
  
  // Resources
  cpuRequest?: string
  memoryRequest?: string
  cpuLimit?: string
  memoryLimit?: string
  
  // Scaling
  minReplicas?: number
  maxReplicas?: number
  targetCPUUtilization?: number
  
  // Advanced
  nodeSelector?: Record<string, string>
  tolerations?: V1Toleration[]
  
  // Networking
  enableIngress?: boolean
  ingressHost?: string
  ingressPath?: string
  enableTLS?: boolean
}

export interface LanguageAgentListItem {
  name: string
  namespace: string
  mode: string
  phase: string
  replicas?: number
  executions?: number
  successRate?: string
  age: string
  creationTimestamp: string
}

// API response types
export interface LanguageAgentResponse {
  success: boolean
  data?: LanguageAgent
  error?: string
}

export interface LanguageAgentListResponse {
  success: boolean
  data?: LanguageAgent[]
  error?: string
  total?: number
  page?: number
  limit?: number
}

// Query parameters for listing agents
export interface LanguageAgentListParams {
  namespace?: string
  labelSelector?: string
  fieldSelector?: string
  page?: number
  limit?: number
  sortBy?: 'name' | 'namespace' | 'phase' | 'age' | 'executions' | 'successRate'
  sortOrder?: 'asc' | 'desc'
  search?: string
  phase?: string[]
  executionMode?: string[]
}