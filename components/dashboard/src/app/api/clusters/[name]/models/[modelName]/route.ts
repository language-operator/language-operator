import { NextRequest, NextResponse } from 'next/server'
import { getServerSession } from 'next-auth'
import { authOptions } from '@/lib/auth'
import { k8sClient } from '@/lib/k8s-client'
import { requirePermission } from '@/lib/permissions'
import { getUserOrganization } from '@/lib/organization-context'
import { validateClusterExists } from '@/lib/cluster-validation'
import { createErrorResponse, createSuccessResponse, handleKubernetesOperation, validateClusterNameFormat, createAuthenticationRequiredError, createPermissionDeniedError } from '@/lib/api-error-handler'
import { z } from 'zod'

// Validation schema for model updates
const updateModelSchema = z.object({
  name: z.string().min(1).optional(),
  provider: z.string().min(1).optional(),
  model: z.string().min(1).optional(),
  endpoint: z.string().url().optional(),
  apiKey: z.string().optional(),
  description: z.string().optional(),
  spec: z.object({
    provider: z.string().optional(),
    model: z.string().optional(),
    endpoint: z.string().optional(),
    apiKey: z.string().optional(),
    timeout: z.string().optional(),
    parameters: z.object({
      maxTokens: z.number().optional(),
      temperature: z.number().optional(),
      topP: z.number().optional(),
      frequencyPenalty: z.number().optional(),
      presencePenalty: z.number().optional(),
      stopSequences: z.array(z.string()).optional(),
      additionalParameters: z.record(z.string(), z.any()).optional(),
    }).optional(),
    contextWindow: z.number().optional(),
    costTracking: z.object({
      inputTokenCost: z.number().optional(),
      outputTokenCost: z.number().optional(),
      currency: z.string().optional(),
      enabled: z.boolean().optional()
    }).optional(),
    enabled: z.boolean().optional(),
    requireApproval: z.boolean().optional(),
    caching: z.object({
      enabled: z.boolean().optional(),
      backend: z.string().optional(),
      ttl: z.string().optional(),
      maxSize: z.number().optional()
    }).optional(),
    loadBalancing: z.object({
      strategy: z.string().optional(),
      endpoints: z.array(z.string()).optional(),
      healthCheck: z.object({
        interval: z.string().optional(),
        timeout: z.string().optional(),
        healthyThreshold: z.number().optional(),
        unhealthyThreshold: z.number().optional()
      }).optional()
    }).optional(),
    observability: z.object({
      logging: z.object({
        level: z.string().optional(),
        logRequests: z.boolean().optional(),
        logResponses: z.boolean().optional()
      }).optional(),
      metrics: z.boolean().optional(),
      tracing: z.boolean().optional()
    }).optional(),
    rateLimiting: z.object({
      requestsPerMinute: z.number().optional(),
      tokensPerMinute: z.number().optional(),
      concurrentRequests: z.number().optional()
    }).optional(),
    retryPolicy: z.object({
      maxAttempts: z.number().optional(),
      initialBackoff: z.string().optional(),
      maxBackoff: z.string().optional(),
      backoffMultiplier: z.number().optional(),
      retryableStatusCodes: z.array(z.number()).optional()
    }).optional(),
    egress: z.array(z.object({
      description: z.string(),
      to: z.object({
        dns: z.array(z.string()).optional(),
        cidr: z.string().optional(),
      }),
      ports: z.array(z.object({
        port: z.number().int().min(1).max(65535),
        protocol: z.enum(['TCP', 'UDP', 'SCTP']).optional().default('TCP'),
      })).optional(),
    })).optional(),
    regions: z.array(z.string()).optional(),
    fallbacks: z.array(z.string()).optional()
  }).optional()
})

// GET /api/clusters/[name]/models/[modelName] - Get individual model details
export async function GET(
  request: NextRequest,
  { params }: { params: Promise<{ name: string; modelName: string }> }
) {
  try {
    // Get user's selected organization
    const { user, organization, userRole } = await getUserOrganization(request)
    
    if (!user?.id) {
      throw createAuthenticationRequiredError()
    }

    // Check permissions
    const hasPermission = await requirePermission(user.id, organization.id, 'view')
    if (!hasPermission) {
      throw createPermissionDeniedError('view model', 'cluster-scoped models', userRole)
    }

    const { name: clusterName, modelName } = await params
    
    // Validate cluster name format
    validateClusterNameFormat(clusterName)
    
    // Validate cluster exists and user has access
    await validateClusterExists(organization.namespace, clusterName, { validateAccess: true })

    console.log(`Fetching model ${modelName} for cluster ${clusterName} from namespace:`, organization.namespace)
    
    // Fetch the specific model from Kubernetes
    const response = await handleKubernetesOperation(
      'get model',
      k8sClient.getLanguageModel(organization.namespace, modelName)
    )
    
    // Extract model data from response
    let model = null
    if (response.body && typeof response.body === 'object') {
      model = response.body
    } else if (response.data && typeof response.data === 'object') {
      model = response.data
    } else {
      model = response
    }

    if (!model) {
      return createErrorResponse(
        new Error(`Model '${modelName}' not found`),
        'Model not found'
      )
    }

    // Verify the model belongs to this cluster (check clusterRef)
    if (model.spec?.clusterRef !== clusterName) {
      return createErrorResponse(
        new Error(`Model '${modelName}' does not belong to cluster '${clusterName}'`),
        'Model not found in cluster'
      )
    }

    console.log(`Successfully fetched model ${modelName} for cluster ${clusterName}`)

    return createSuccessResponse(model, undefined, {
      cluster: clusterName,
    })

  } catch (error) {
    console.error('Error fetching cluster model:', error)
    return createErrorResponse(error, 'Failed to fetch cluster model')
  }
}

// PUT /api/clusters/[name]/models/[modelName] - Update model
export async function PUT(
  request: NextRequest,
  { params }: { params: Promise<{ name: string; modelName: string }> }
) {
  try {
    // Get user's selected organization
    const { user, organization, userRole } = await getUserOrganization(request)
    
    if (!user?.id) {
      throw createAuthenticationRequiredError()
    }

    // Check permissions
    const hasPermission = await requirePermission(user.id, organization.id, 'edit')
    if (!hasPermission) {
      throw createPermissionDeniedError('edit model', 'cluster-scoped models', userRole)
    }

    const { name: clusterName, modelName } = await params
    
    // Validate cluster name format
    validateClusterNameFormat(clusterName)
    
    // Validate cluster exists and user has access
    await validateClusterExists(organization.namespace, clusterName, { validateAccess: true })

    const body = await request.json()
    const validatedData = updateModelSchema.parse(body)

    console.log(`Updating model ${modelName} for cluster ${clusterName} in namespace:`, organization.namespace)
    
    // First, get the existing model to merge with updates
    const existingResponse = await handleKubernetesOperation(
      'get model for update',
      k8sClient.getLanguageModel(organization.namespace, modelName)
    )
    
    let existingModel = null
    if (existingResponse.body && typeof existingResponse.body === 'object') {
      existingModel = existingResponse.body
    } else if (existingResponse.data && typeof existingResponse.data === 'object') {
      existingModel = existingResponse.data
    } else {
      existingModel = existingResponse
    }

    if (!existingModel) {
      return createErrorResponse(
        new Error(`Model '${modelName}' not found`),
        'Model not found'
      )
    }

    // Verify the model belongs to this cluster
    if (existingModel.spec?.clusterRef !== clusterName) {
      return createErrorResponse(
        new Error(`Model '${modelName}' does not belong to cluster '${clusterName}'`),
        'Model not found in cluster'
      )
    }

    // Merge the updates with existing model spec
    const updatedModel = {
      ...existingModel,
      spec: {
        ...existingModel.spec,
        ...validatedData.spec,
        // Ensure clusterRef is preserved
        clusterRef: clusterName,
      }
    }

    // Remove metadata fields that shouldn't be updated (keep resourceVersion for updates)
    delete updatedModel.metadata.generation
    delete updatedModel.metadata.creationTimestamp
    delete updatedModel.metadata.uid
    delete updatedModel.status

    // Update the model in Kubernetes using replace (patch format issues with CRDs)
    const response = await handleKubernetesOperation(
      'update model',
      k8sClient.replaceLanguageModel(organization.namespace, modelName, updatedModel)
    )
    
    console.log(`Successfully updated model ${modelName} for cluster ${clusterName}`)

    return createSuccessResponse(response, 'Model updated successfully')

  } catch (error) {
    console.error('Error updating cluster model:', error)
    return createErrorResponse(error, 'Failed to update cluster model')
  }
}

// DELETE /api/clusters/[name]/models/[modelName] - Delete model
export async function DELETE(
  request: NextRequest,
  { params }: { params: Promise<{ name: string; modelName: string }> }
) {
  try {
    // Get user's selected organization
    const { user, organization, userRole } = await getUserOrganization(request)
    
    if (!user?.id) {
      throw createAuthenticationRequiredError()
    }

    // Check permissions
    const hasPermission = await requirePermission(user.id, organization.id, 'delete')
    if (!hasPermission) {
      throw createPermissionDeniedError('delete model', 'cluster-scoped models', userRole)
    }

    const { name: clusterName, modelName } = await params
    
    // Validate cluster name format
    validateClusterNameFormat(clusterName)
    
    // Validate cluster exists and user has access
    await validateClusterExists(organization.namespace, clusterName, { validateAccess: true })

    console.log(`Deleting model ${modelName} for cluster ${clusterName} from namespace:`, organization.namespace)
    
    // First verify the model belongs to this cluster
    const existingResponse = await handleKubernetesOperation(
      'get model for deletion',
      k8sClient.getLanguageModel(organization.namespace, modelName)
    )
    
    let existingModel = null
    if (existingResponse.body && typeof existingResponse.body === 'object') {
      existingModel = existingResponse.body
    } else if (existingResponse.data && typeof existingResponse.data === 'object') {
      existingModel = existingResponse.data
    } else {
      existingModel = existingResponse
    }

    if (!existingModel) {
      return createErrorResponse(
        new Error(`Model '${modelName}' not found`),
        'Model not found'
      )
    }

    // Verify the model belongs to this cluster
    if (existingModel.spec?.clusterRef !== clusterName) {
      return createErrorResponse(
        new Error(`Model '${modelName}' does not belong to cluster '${clusterName}'`),
        'Model not found in cluster'
      )
    }

    // Delete the model from Kubernetes
    const response = await handleKubernetesOperation(
      'delete model',
      k8sClient.deleteLanguageModel(organization.namespace, modelName)
    )
    
    console.log(`Successfully deleted model ${modelName} for cluster ${clusterName}`)

    return createSuccessResponse(response, 'Model deleted successfully')

  } catch (error) {
    console.error('Error deleting cluster model:', error)
    return createErrorResponse(error, 'Failed to delete cluster model')
  }
}