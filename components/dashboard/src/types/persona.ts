// Generated TypeScript types for LanguagePersona CRD

import { V1ObjectMeta, V1Condition } from '@kubernetes/client-node'

// LanguagePersona CRD Types
export interface LanguagePersona {
  apiVersion: 'langop.io/v1alpha1'
  kind: 'LanguagePersona'
  metadata: V1ObjectMeta
  spec: LanguagePersonaSpec
  status?: LanguagePersonaStatus
}

export interface LanguagePersonaList {
  apiVersion: 'langop.io/v1alpha1'
  kind: 'LanguagePersonaList'
  metadata: V1ObjectMeta
  items: LanguagePersona[]
}

export interface LanguagePersonaSpec {
  // Core persona definition
  displayName?: string
  description?: string
  systemPrompt?: string
  personality?: string
  
  // Cluster reference for scoping
  clusterRef?: string
  
  // Communication style
  tone?: 'professional' | 'casual' | 'friendly' | 'formal' | 'technical' | 'empathetic' | 'concise' | 'detailed'
  language?: string
  
  // Version tracking
  version?: string
  
  // Inheritance
  parentPersona?: PersonaReference
  
  // Behavioral configuration
  capabilities?: string[]
  limitations?: string[]
  instructions?: string[]
  
  // Examples for few-shot learning
  examples?: PersonaExample[]
  
  // Conditional behaviors
  rules?: PersonaRule[]
  
  // Response format preferences
  responseFormat?: ResponseFormat
  
  // Tool usage preferences
  toolPreferences?: ToolPreferences
  
  // Knowledge sources
  knowledgeSources?: KnowledgeSourceSpec[]
  
  // Constraints
  constraints?: PersonaConstraints
  
  // Additional metadata
  metadata?: Record<string, string>
}

export interface PersonaReference {
  name: string
  namespace?: string
}

export interface PersonaExample {
  input: string
  output: string
  context?: string
  tags?: string[]
}

export interface PersonaRule {
  name: string
  condition: string
  action: string
  description?: string
  enabled?: boolean
  priority?: number
}

export interface ResponseFormat {
  type?: 'text' | 'markdown' | 'json' | 'structured' | 'list' | 'table'
  template?: string
  schema?: string
  maxLength?: number
  includeConfidence?: boolean
  includeSources?: boolean
}

export interface ToolPreferences {
  strategy?: 'conservative' | 'balanced' | 'aggressive' | 'minimal'
  preferredTools?: string[]
  avoidTools?: string[]
  explainToolUse?: boolean
  alwaysConfirm?: boolean
}

export interface KnowledgeSourceSpec {
  name: string
  type: 'url' | 'document' | 'database' | 'api' | 'vector-store'
  url?: string
  query?: string
  priority?: number
  enabled?: boolean
  secretRef?: SecretReference
}

export interface SecretReference {
  name: string
  namespace?: string
  key?: string
}

export interface PersonaConstraints {
  maxResponseTokens?: number
  maxToolCalls?: number
  maxKnowledgeQueries?: number
  responseTimeout?: string
  allowedDomains?: string[]
  blockedTopics?: string[]
  requireDocumentation?: boolean
}

// Status types
export interface LanguagePersonaStatus {
  phase?: 'Ready' | 'NotReady' | 'Validating' | 'Error'
  conditions?: V1Condition[]
  usageCount?: number
  activeAgents?: string[]
  agentReferences?: string[]
  validationResult?: ValidationResult
  metrics?: PersonaMetrics
  lastUpdateTime?: string
  message?: string
  reason?: string
  observedGeneration?: number
}

export interface ValidationResult {
  valid: boolean
  score?: number
  errors?: string[]
  warnings?: string[]
  validationTime?: string
}

export interface PersonaMetrics {
  totalInteractions?: number
  averageResponseLength?: number
  averageToolCalls?: number
  userSatisfaction?: number
  successRate?: string
  lastUsed?: string
  averageQuality?: string
  recentFeedback?: any[]
  topTopics?: TopicFrequency[]
  topTools?: ToolFrequency[]
  ruleActivations?: Record<string, number>
  recentActivity?: any[]
}

export interface TopicFrequency {
  topic: string
  count: number
  percentage?: number
}

export interface ToolFrequency {
  toolName: string
  count: number
  percentage?: number
}

// Frontend-specific types
export interface LanguagePersonaFormData {
  name: string
  namespace: string
  displayName: string
  description: string
  systemPrompt: string
  
  // Basic configuration
  tone?: 'professional' | 'casual' | 'friendly' | 'formal' | 'technical' | 'empathetic' | 'concise' | 'detailed'
  language?: string
  version?: string
  
  // Parent persona
  parentPersonaName?: string
  parentPersonaNamespace?: string
  
  // Behavioral
  capabilities?: string[]
  limitations?: string[]
  instructions?: string[]
  
  // Examples
  examples?: PersonaExampleFormData[]
  
  // Rules
  rules?: PersonaRuleFormData[]
  
  // Response format
  responseType?: 'text' | 'markdown' | 'json' | 'structured' | 'list' | 'table'
  responseTemplate?: string
  responseSchema?: string
  maxResponseLength?: number
  includeConfidence?: boolean
  includeSources?: boolean
  
  // Tool preferences
  toolStrategy?: 'conservative' | 'balanced' | 'aggressive' | 'minimal'
  preferredTools?: string[]
  avoidTools?: string[]
  explainToolUse?: boolean
  alwaysConfirm?: boolean
  
  // Knowledge sources
  knowledgeSources?: KnowledgeSourceFormData[]
  
  // Constraints
  maxResponseTokens?: number
  maxToolCalls?: number
  maxKnowledgeQueries?: number
  responseTimeout?: string
  allowedDomains?: string[]
  blockedTopics?: string[]
  requireDocumentation?: boolean
}

export interface PersonaExampleFormData {
  input: string
  output: string
  context?: string
  tags?: string[]
}

export interface PersonaRuleFormData {
  name: string
  condition: string
  action: string
  description?: string
  enabled?: boolean
  priority?: number
}

export interface KnowledgeSourceFormData {
  name: string
  type: 'url' | 'document' | 'database' | 'api' | 'vector-store'
  url?: string
  query?: string
  priority?: number
  enabled?: boolean
  secretName?: string
  secretKey?: string
}

export interface LanguagePersonaListItem {
  name: string
  namespace: string
  displayName: string
  tone?: string
  phase?: string
  usageCount?: number
  valid?: boolean
  interactions?: number
  age: string
  creationTimestamp: string
}

// API response types
export interface LanguagePersonaResponse {
  success: boolean
  data?: LanguagePersona
  error?: string
}

export interface LanguagePersonaListResponse {
  success: boolean
  data?: LanguagePersona[]
  error?: string
  total?: number
  page?: number
  limit?: number
}

// Query parameters for listing personas
export interface LanguagePersonaListParams {
  namespace?: string
  labelSelector?: string
  fieldSelector?: string
  page?: number
  limit?: number
  sortBy?: 'name' | 'displayName' | 'tone' | 'phase' | 'usageCount' | 'age' | 'usage'
  sortOrder?: 'asc' | 'desc'
  search?: string
  tone?: string[]
  phase?: string[]
  language?: string[]
  valid?: boolean
}