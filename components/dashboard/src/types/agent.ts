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
  // Required: Container image for the agent
  image?: string
  imagePullPolicy?: 'Always' | 'IfNotPresent' | 'Never'
  
  // Primary instructions for agent behavior
  instructions?: string
  
  // Execution configuration - matches CRD enum
  executionMode?: 'autonomous' | 'interactive' | 'scheduled' | 'event-driven'
  replicas?: number
  
  // Model references - matches CRD structure (optional for backward compatibility)
  modelRefs?: ModelReference[]
  
  // Tool references - matches CRD structure
  toolRefs?: ToolReference[]
  
  // Persona references - matches CRD structure  
  personaRefs?: PersonaReference[]
  
  // Required fields from CRD
  clusterRef?: string
  backoffLimit?: number
  maxIterations?: number
  timeout?: string
  restartPolicy?: 'OnFailure' | 'Always' | 'Never'
  
  // Agent version reference
  agentVersionRef?: AgentVersionReference
  
  // Workspace configuration
  workspace?: WorkspaceConfig
  
  // Legacy fields for backward compatibility
  model?: LanguageModelConfig
  models?: LanguageModelConfig[]
  persona?: LanguagePersonaConfig
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

// CRD-matching reference types
export interface ModelReference {
  name: string
  namespace?: string
  role?: 'primary' | 'fallback' | 'reasoning' | 'tool-calling' | 'summarization'
  priority?: number
}

export interface ToolReference {
  name: string
  namespace?: string
  enabled?: boolean
  requireApproval?: boolean
}

export interface PersonaReference {
  name: string
  namespace?: string
}

export interface AgentVersionReference {
  name: string
  namespace?: string
  lock?: boolean
}

export interface WorkspaceConfig {
  enabled?: boolean
  size?: string
  accessMode?: 'ReadWriteOnce' | 'ReadOnlyMany' | 'ReadWriteMany'
  mountPath?: string
  storageClassName?: string
}

// Legacy model config for backward compatibility
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
  synthesisInfo?: SynthesisInfo
  runsPendingLearning?: number
  learningRequestPending?: boolean
}

export interface SynthesisInfo {
  lastSynthesisTime?: string
  synthesisModel?: string
  synthesisDuration?: number
  synthesisAttempts?: number
  codeHash?: string
  instructionsHash?: string
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

// Frontend-specific types - simplified for new form structure
export interface LanguageAgentFormData {
  // Primary fields matching simplified form
  instructions?: string
  name: string
  namespace: string
  selectedModels?: string[]
  selectedTools?: string[]
  selectedPersona?: string
  
  // Legacy/backward compatibility fields (optional)
  executionMode?: 'autonomous' | 'interactive' | 'scheduled' | 'event-driven'
  replicas?: number
  modelName?: string
  modelProvider?: string
  modelEndpoint?: string
  modelParameters?: Record<string, any>
  personaName?: string
  personaTone?: string
  personaInstructions?: string
  
  // Resource fields for legacy forms
  cpuRequest?: string
  memoryRequest?: string
  cpuLimit?: string
  memoryLimit?: string
  minReplicas?: number
  maxReplicas?: number
  targetCPUUtilization?: number
  nodeSelector?: Record<string, string>
  tolerations?: V1Toleration[]
  enableIngress?: boolean
  ingressHost?: string
  ingressPath?: string
  enableTLS?: boolean
  clusterRef?: string
  
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
  
  // Ingress policy fields
  ingressRules?: Array<{
    description?: string
    from: 'agents' | 'tools' | 'models' | 'cluster' | 'external' | 'gateway'
    ports?: Array<{
      port: number
      protocol: 'TCP' | 'UDP'
    }>
  }>
  
  // Cluster-specific ingress rules for advanced use cases
  clusterIngressRules?: Array<{
    description?: string
    from: 'agents' | 'tools' | 'models' | 'cluster'
    clusterName?: string
    ports?: Array<{
      port: number
      protocol: 'TCP' | 'UDP'
    }>
  }>
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