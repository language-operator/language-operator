import { z } from 'zod'

// Common Kubernetes metadata schema
const KubernetesMetadataSchema = z.object({
  name: z.string().min(1).max(253).regex(/^[a-z0-9]([a-z0-9-]*[a-z0-9])?$/, {
    message: "Name must be lowercase alphanumeric with hyphens, starting and ending with alphanumeric"
  }),
  namespace: z.string().min(1).max(63).regex(/^[a-z0-9]([a-z0-9-]*[a-z0-9])?$/, {
    message: "Namespace must be lowercase alphanumeric with hyphens"
  }),
  labels: z.record(z.string(), z.string()).optional(),
  annotations: z.record(z.string(), z.string()).optional(),
  uid: z.string().optional(),
  resourceVersion: z.string().optional(),
  generation: z.number().optional(),
  creationTimestamp: z.string().optional(),
  deletionTimestamp: z.string().optional(),
  finalizers: z.array(z.string()).optional(),
})

// LanguageAgent validation
export const LanguageAgentSpecSchema = z.object({
  model: z.string().min(1, "Model reference is required"),
  persona: z.string().optional(),
  tools: z.array(z.string()).optional(),
  cluster: z.string().optional(),
  config: z.object({
    maxTokens: z.number().int().min(1).max(100000).optional(),
    temperature: z.number().min(0).max(2).optional(),
    systemPrompt: z.string().optional(),
    enableStreaming: z.boolean().optional(),
    timeout: z.number().int().min(1).max(3600).optional(),
  }).optional(),
  replicas: z.number().int().min(0).max(100).default(1),
  resources: z.object({
    requests: z.object({
      cpu: z.string().optional(),
      memory: z.string().optional(),
    }).optional(),
    limits: z.object({
      cpu: z.string().optional(),
      memory: z.string().optional(),
    }).optional(),
  }).optional(),
})

export const LanguageAgentStatusSchema = z.object({
  phase: z.enum(['Pending', 'Ready', 'Failed', 'Updating']).optional(),
  replicas: z.object({
    ready: z.number().int().min(0).optional(),
    total: z.number().int().min(0).optional(),
    available: z.number().int().min(0).optional(),
  }).optional(),
  conditions: z.array(z.object({
    type: z.string(),
    status: z.enum(['True', 'False', 'Unknown']),
    reason: z.string().optional(),
    message: z.string().optional(),
    lastTransitionTime: z.string().optional(),
  })).optional(),
  lastUpdateTime: z.string().optional(),
  message: z.string().optional(),
  endpoint: z.string().url().optional(),
})

export const LanguageAgentSchema = z.object({
  apiVersion: z.string().default('language.operator.io/v1'),
  kind: z.string().default('LanguageAgent'),
  metadata: KubernetesMetadataSchema,
  spec: LanguageAgentSpecSchema,
  status: LanguageAgentStatusSchema.optional(),
})

// LanguageModel validation
export const LanguageModelSpecSchema = z.object({
  provider: z.enum(['openai', 'anthropic', 'huggingface', 'azure', 'google', 'local'], {
    errorMap: () => ({ message: "Provider must be one of: openai, anthropic, huggingface, azure, google, local" })
  }),
  modelName: z.string().min(1, "Model name is required"),
  apiKey: z.string().min(1, "API key is required"),
  endpoint: z.string().url().optional(),
  config: z.object({
    maxTokens: z.number().int().min(1).max(100000).optional(),
    temperature: z.number().min(0).max(2).optional(),
    topP: z.number().min(0).max(1).optional(),
    presencePenalty: z.number().min(-2).max(2).optional(),
    frequencyPenalty: z.number().min(-2).max(2).optional(),
    timeout: z.number().int().min(1).max(3600).optional(),
  }).optional(),
  rateLimit: z.object({
    requestsPerMinute: z.number().int().min(1).max(10000).optional(),
    tokensPerMinute: z.number().int().min(1).max(1000000).optional(),
  }).optional(),
})

export const LanguageModelStatusSchema = z.object({
  phase: z.enum(['Pending', 'Available', 'Failed', 'Testing']).optional(),
  lastTested: z.string().optional(),
  testResult: z.object({
    success: z.boolean(),
    latency: z.number().optional(),
    error: z.string().optional(),
    timestamp: z.string().optional(),
  }).optional(),
  usage: z.object({
    totalRequests: z.number().int().min(0).optional(),
    totalTokens: z.number().int().min(0).optional(),
    lastUsed: z.string().optional(),
  }).optional(),
  conditions: z.array(z.object({
    type: z.string(),
    status: z.enum(['True', 'False', 'Unknown']),
    reason: z.string().optional(),
    message: z.string().optional(),
    lastTransitionTime: z.string().optional(),
  })).optional(),
  message: z.string().optional(),
})

export const LanguageModelSchema = z.object({
  apiVersion: z.string().default('language.operator.io/v1'),
  kind: z.string().default('LanguageModel'),
  metadata: KubernetesMetadataSchema,
  spec: LanguageModelSpecSchema,
  status: LanguageModelStatusSchema.optional(),
})

// LanguageTool validation
export const LanguageToolSpecSchema = z.object({
  type: z.enum(['function', 'api', 'webhook', 'script'], {
    errorMap: () => ({ message: "Tool type must be one of: function, api, webhook, script" })
  }),
  name: z.string().min(1, "Tool name is required"),
  description: z.string().min(1, "Tool description is required"),
  parameters: z.object({
    type: z.literal('object'),
    properties: z.record(z.string(), z.any()),
    required: z.array(z.string()).optional(),
    additionalProperties: z.boolean().optional(),
  }),
  implementation: z.object({
    code: z.string().optional(),
    endpoint: z.string().url().optional(),
    method: z.enum(['GET', 'POST', 'PUT', 'DELETE', 'PATCH']).optional(),
    headers: z.record(z.string(), z.string()).optional(),
    timeout: z.number().int().min(1).max(300).optional(),
    retries: z.number().int().min(0).max(5).optional(),
  }),
  security: z.object({
    requiresAuth: z.boolean().optional(),
    allowedOrigins: z.array(z.string()).optional(),
    rateLimiting: z.object({
      requestsPerMinute: z.number().int().min(1).max(1000).optional(),
    }).optional(),
  }).optional(),
})

export const LanguageToolStatusSchema = z.object({
  phase: z.enum(['Pending', 'Available', 'Failed', 'Testing']).optional(),
  lastTested: z.string().optional(),
  testResult: z.object({
    success: z.boolean(),
    responseTime: z.number().optional(),
    error: z.string().optional(),
    timestamp: z.string().optional(),
  }).optional(),
  usage: z.object({
    totalCalls: z.number().int().min(0).optional(),
    successfulCalls: z.number().int().min(0).optional(),
    lastUsed: z.string().optional(),
  }).optional(),
  agentReferences: z.array(z.object({
    name: z.string(),
    namespace: z.string(),
  })).optional(),
  conditions: z.array(z.object({
    type: z.string(),
    status: z.enum(['True', 'False', 'Unknown']),
    reason: z.string().optional(),
    message: z.string().optional(),
    lastTransitionTime: z.string().optional(),
  })).optional(),
  message: z.string().optional(),
})

export const LanguageToolSchema = z.object({
  apiVersion: z.string().default('language.operator.io/v1'),
  kind: z.string().default('LanguageTool'),
  metadata: KubernetesMetadataSchema,
  spec: LanguageToolSpecSchema,
  status: LanguageToolStatusSchema.optional(),
})

// LanguagePersona validation
export const LanguagePersonaSpecSchema = z.object({
  tone: z.string().min(1, "Tone is required"),
  description: z.string().min(1, "Description is required"),
  systemPrompt: z.string().min(1, "System prompt is required"),
  examples: z.array(z.object({
    input: z.string().min(1, "Example input is required"),
    output: z.string().min(1, "Example output is required"),
    context: z.string().optional(),
  })).optional(),
  constraints: z.array(z.string()).optional(),
  vocabulary: z.object({
    preferred: z.array(z.string()).optional(),
    forbidden: z.array(z.string()).optional(),
  }).optional(),
  responseFormat: z.object({
    structure: z.enum(['freeform', 'structured', 'json', 'markdown']).optional(),
    maxLength: z.number().int().min(1).max(10000).optional(),
    includeMetadata: z.boolean().optional(),
  }).optional(),
})

export const LanguagePersonaStatusSchema = z.object({
  phase: z.enum(['Pending', 'Available', 'Failed']).optional(),
  agentReferences: z.array(z.object({
    name: z.string(),
    namespace: z.string(),
  })).optional(),
  usage: z.object({
    totalAgents: z.number().int().min(0).optional(),
    lastUsed: z.string().optional(),
  }).optional(),
  conditions: z.array(z.object({
    type: z.string(),
    status: z.enum(['True', 'False', 'Unknown']),
    reason: z.string().optional(),
    message: z.string().optional(),
    lastTransitionTime: z.string().optional(),
  })).optional(),
  message: z.string().optional(),
})

export const LanguagePersonaSchema = z.object({
  apiVersion: z.string().default('language.operator.io/v1'),
  kind: z.string().default('LanguagePersona'),
  metadata: KubernetesMetadataSchema,
  spec: LanguagePersonaSpecSchema,
  status: LanguagePersonaStatusSchema.optional(),
})

// LanguageCluster validation
export const LanguageClusterSpecSchema = z.object({
  domain: z.string().min(1, "Domain is required").refine((val) => {
    // Basic domain validation
    return /^[a-zA-Z0-9]([a-zA-Z0-9-]*[a-zA-Z0-9])?(\.[a-zA-Z0-9]([a-zA-Z0-9-]*[a-zA-Z0-9])?)*$/.test(val)
  }, "Invalid domain format"),
  subdomain: z.string().optional(),
  tls: z.object({
    enabled: z.boolean().default(true),
    autoProvision: z.boolean().default(true),
    secretName: z.string().optional(),
    issuer: z.string().optional(),
  }).optional(),
  gateway: z.object({
    className: z.string().optional(),
    loadBalancerType: z.enum(['standard', 'internal', 'external']).optional(),
    annotations: z.record(z.string(), z.string()).optional(),
    allowedOrigins: z.array(z.string()).optional(),
  }).optional(),
  scaling: z.object({
    minReplicas: z.number().int().min(0).max(100).default(1),
    maxReplicas: z.number().int().min(1).max(100).default(10),
    targetCPUUtilization: z.number().int().min(1).max(100).default(70),
    targetMemoryUtilization: z.number().int().min(1).max(100).default(80),
  }).optional(),
})

export const LanguageClusterStatusSchema = z.object({
  phase: z.enum(['Pending', 'Ready', 'Failed', 'Scaling']).optional(),
  agentCount: z.number().int().min(0).optional(),
  agents: z.array(z.object({
    name: z.string(),
    namespace: z.string(),
    status: z.enum(['Ready', 'Pending', 'Failed']),
  })).optional(),
  ingress: z.object({
    ready: z.boolean().optional(),
    endpoint: z.string().url().optional(),
    dnsRecords: z.array(z.object({
      type: z.string(),
      name: z.string(),
      value: z.string(),
    })).optional(),
  }).optional(),
  tls: z.object({
    ready: z.boolean().optional(),
    certificateExpiry: z.string().optional(),
    issuer: z.string().optional(),
  }).optional(),
  gateway: z.object({
    ready: z.boolean().optional(),
    externalIP: z.string().optional(),
    loadBalancerStatus: z.string().optional(),
  }).optional(),
  conditions: z.array(z.object({
    type: z.string(),
    status: z.enum(['True', 'False', 'Unknown']),
    reason: z.string().optional(),
    message: z.string().optional(),
    lastTransitionTime: z.string().optional(),
  })).optional(),
  lastUpdateTime: z.string().optional(),
  message: z.string().optional(),
})

export const LanguageClusterSchema = z.object({
  apiVersion: z.string().default('language.operator.io/v1'),
  kind: z.string().default('LanguageCluster'),
  metadata: KubernetesMetadataSchema,
  spec: LanguageClusterSpecSchema,
  status: LanguageClusterStatusSchema.optional(),
})

// Export all schemas for easy access
export const CRDSchemas = {
  LanguageAgent: LanguageAgentSchema,
  LanguageModel: LanguageModelSchema,
  LanguageTool: LanguageToolSchema,
  LanguagePersona: LanguagePersonaSchema,
  LanguageCluster: LanguageClusterSchema,
}

// Type inference from schemas
export type LanguageAgent = z.infer<typeof LanguageAgentSchema>
export type LanguageModel = z.infer<typeof LanguageModelSchema>
export type LanguageTool = z.infer<typeof LanguageToolSchema>
export type LanguagePersona = z.infer<typeof LanguagePersonaSchema>
export type LanguageCluster = z.infer<typeof LanguageClusterSchema>

// Validation helper functions
export function validateLanguageAgent(data: unknown): LanguageAgent {
  return LanguageAgentSchema.parse(data)
}

export function validateLanguageModel(data: unknown): LanguageModel {
  return LanguageModelSchema.parse(data)
}

export function validateLanguageTool(data: unknown): LanguageTool {
  return LanguageToolSchema.parse(data)
}

export function validateLanguagePersona(data: unknown): LanguagePersona {
  return LanguagePersonaSchema.parse(data)
}

export function validateLanguageCluster(data: unknown): LanguageCluster {
  return LanguageClusterSchema.parse(data)
}

// Safe validation that returns errors instead of throwing
export function safeValidateLanguageAgent(data: unknown): { success: true; data: LanguageAgent } | { success: false; error: z.ZodError } {
  const result = LanguageAgentSchema.safeParse(data)
  return result.success ? { success: true, data: result.data } : { success: false, error: result.error }
}

export function safeValidateLanguageModel(data: unknown): { success: true; data: LanguageModel } | { success: false; error: z.ZodError } {
  const result = LanguageModelSchema.safeParse(data)
  return result.success ? { success: true, data: result.data } : { success: false, error: result.error }
}

export function safeValidateLanguageTool(data: unknown): { success: true; data: LanguageTool } | { success: false; error: z.ZodError } {
  const result = LanguageToolSchema.safeParse(data)
  return result.success ? { success: true, data: result.data } : { success: false, error: result.error }
}

export function safeValidateLanguagePersona(data: unknown): { success: true; data: LanguagePersona } | { success: false; error: z.ZodError } {
  const result = LanguagePersonaSchema.safeParse(data)
  return result.success ? { success: true, data: result.data } : { success: false, error: result.error }
}

export function safeValidateLanguageCluster(data: unknown): { success: true; data: LanguageCluster } | { success: false; error: z.ZodError } {
  const result = LanguageClusterSchema.safeParse(data)
  return result.success ? { success: true, data: result.data } : { success: false, error: result.error }
}