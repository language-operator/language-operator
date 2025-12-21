// Generated TypeScript types for LanguageModel CRD

import { V1ObjectMeta, V1LabelSelector } from '@kubernetes/client-node'

// LanguageModel CRD Types
export interface LanguageModel {
  apiVersion: 'langop.io/v1alpha1'
  kind: 'LanguageModel'
  metadata: V1ObjectMeta
  spec: LanguageModelSpec
  status?: LanguageModelStatus
}

export interface LanguageModelList {
  apiVersion: 'langop.io/v1alpha1'
  kind: 'LanguageModelList'
  metadata: V1ObjectMeta
  items: LanguageModel[]
}

export interface LanguageModelSpec {
  // Core configuration
  provider: 'openai' | 'anthropic' | 'openai-compatible' | 'azure' | 'bedrock' | 'vertex' | 'custom'
  modelName: string
  endpoint?: string
  
  // Cluster reference for scoping
  clusterRef?: string
  
  // API Key configuration
  apiKeySecretRef?: SecretReference
  
  // Model configuration
  configuration?: ModelConfiguration
  
  // Cost tracking
  costTracking?: CostTracking
  
  // Caching configuration
  caching?: CachingConfiguration
  
  // Rate limiting
  rateLimits?: RateLimits
  
  // Retry policy
  retryPolicy?: RetryPolicy
  
  // Load balancing
  loadBalancing?: LoadBalancing
  
  // Regions
  regions?: RegionSpec[]
  
  // Fallbacks
  fallbacks?: ModelFallbackSpec[]
  
  // Observability
  observability?: ObservabilityConfiguration
  
  // Network egress rules
  egress?: NetworkRule[]
}

export interface SecretReference {
  name: string
  namespace?: string
  key?: string
}

export interface ModelConfiguration {
  temperature?: number
  maxTokens?: number
  topP?: number
  frequencyPenalty?: number
  presencePenalty?: number
  stopSequences?: string[]
  timeoutMs?: number
  additionalParameters?: Record<string, string>
}

export interface CostTracking {
  enabled?: boolean
  currency?: string
  inputTokenCost?: number
  outputTokenCost?: number
}

export interface CachingConfiguration {
  enabled?: boolean
  backend?: 'memory' | 'redis' | 'memcached'
  ttl?: string
  maxSize?: number
}

export interface RateLimits {
  requestsPerMinute?: number
  tokensPerMinute?: number
  concurrentRequests?: number
}

export interface RetryPolicy {
  enabled?: boolean
  maxRetries?: number
  initialInterval?: string
  maxInterval?: string
  backoffMultiplier?: number
  retryOn?: string[]
}

export interface LoadBalancing {
  strategy?: 'round-robin' | 'least-connections' | 'random' | 'weighted' | 'latency-based'
  endpoints?: EndpointSpec[]
  healthCheck?: HealthCheckConfiguration
}

export interface EndpointSpec {
  url: string
  weight?: number
  priority?: number
  region?: string
}

export interface HealthCheckConfiguration {
  enabled?: boolean
  interval?: string
  timeout?: string
  healthyThreshold?: number
  unhealthyThreshold?: number
}

export interface RegionSpec {
  name: string
  enabled?: boolean
  endpoint?: string
  priority?: number
}

export interface ModelFallbackSpec {
  modelRef: string
  conditions?: string[]
}

export interface ObservabilityConfiguration {
  metrics?: boolean
  tracing?: boolean
  logging?: LoggingConfiguration
}

export interface LoggingConfiguration {
  level?: 'debug' | 'info' | 'warn' | 'error'
  logRequests?: boolean
  logResponses?: boolean
}

export interface NetworkRule {
  description?: string
  ports?: NetworkPort[]
  to?: NetworkSelector
  from?: NetworkSelector
}

export interface NetworkPort {
  port: number
  protocol?: 'TCP' | 'UDP' | 'SCTP'
}

export interface NetworkSelector {
  cidr?: string
  dns?: string[]
  group?: string
  podSelector?: V1LabelSelector
  namespaceSelector?: V1LabelSelector
  service?: ServiceReference
}

export interface ServiceReference {
  name: string
  namespace?: string
}

// Status types
export interface LanguageModelStatus {
  phase?: 'Pending' | 'Ready' | 'Failed' | 'Unknown'
  healthy?: boolean
  conditions?: LanguageModelCondition[]
  endpoints?: EndpointStatus[]
  metrics?: LanguageModelMetrics
  lastHealthCheck?: string
  observedGeneration?: number
}

export interface LanguageModelCondition {
  type: string
  status: 'True' | 'False' | 'Unknown'
  lastTransitionTime?: string
  lastUpdateTime?: string
  reason?: string
  message?: string
}

export interface EndpointStatus {
  url: string
  healthy: boolean
  latency?: string
  lastChecked?: string
  region?: string
  error?: string
}

export interface LanguageModelMetrics {
  totalRequests?: number
  successfulRequests?: number
  failedRequests?: number
  successRate?: string
  errorRate?: string
  averageLatency?: number
  p95Latency?: number
  p99Latency?: number
  requestsPerMinute?: number
  tokensPerMinute?: number
  costMetrics?: CostMetrics
  healthScore?: number
  regionHealth?: any[]
}

export interface CostMetrics {
  totalCost?: string
  totalInputTokens?: number
  totalOutputTokens?: number
  averageCostPerRequest?: string
  currency?: string
  billingPeriod?: string
}

// Frontend-specific types
export interface LanguageModelFormData {
  name: string
  namespace: string
  provider: 'openai' | 'anthropic' | 'openai-compatible' | 'azure' | 'bedrock' | 'vertex' | 'custom'
  modelName: string
  endpoint?: string
  
  // API Key
  apiKeySecretName?: string
  apiKeySecretKey?: string
  
  // Configuration
  temperature?: number
  maxTokens?: number
  topP?: number
  
  // Cost tracking
  enableCostTracking?: boolean
  inputTokenCost?: number
  outputTokenCost?: number
  currency?: string
  
  // Caching
  enableCaching?: boolean
  cacheTtl?: string
  cacheBackend?: 'memory' | 'redis' | 'memcached'
  
  // Rate limiting
  requestsPerMinute?: number
  tokensPerMinute?: number
  concurrentRequests?: number
  
  // Retry policy
  enableRetry?: boolean
  maxRetries?: number
  
  // Load balancing
  enableLoadBalancing?: boolean
  loadBalancingStrategy?: 'round-robin' | 'least-connections' | 'random' | 'weighted'
  additionalEndpoints?: string[]
  
  // Health checking
  enableHealthCheck?: boolean
  healthCheckInterval?: string
  
  // Observability
  enableMetrics?: boolean
  enableTracing?: boolean
  logLevel?: 'debug' | 'info' | 'warn' | 'error'
  
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
  
  // Ingress policy fields - models accept connections from agents
  ingressRules?: Array<{
    description?: string
    from: 'agents' | 'tools' | 'models' | 'cluster' | 'external' | 'gateway'
    ports?: Array<{
      port: number
      protocol: 'TCP' | 'UDP'
    }>
  }>
}

export interface LanguageModelListItem {
  name: string
  namespace: string
  provider: string
  modelName: string
  phase?: string
  healthy?: boolean
  requests?: number
  latency?: string
  age: string
  creationTimestamp: string
}

// API response types
export interface LanguageModelResponse {
  success: boolean
  data?: LanguageModel
  error?: string
}

export interface LanguageModelListResponse {
  success: boolean
  data?: LanguageModel[]
  error?: string
  total?: number
  page?: number
  limit?: number
}

// Query parameters for listing models
export interface LanguageModelListParams {
  namespace?: string
  labelSelector?: string
  fieldSelector?: string
  page?: number
  limit?: number
  sortBy?: 'name' | 'provider' | 'phase' | 'healthy' | 'requests' | 'age' | 'latency' | 'health' | 'cost'
  sortOrder?: 'asc' | 'desc'
  search?: string
  provider?: string[]
  phase?: string[]
  healthy?: boolean
}