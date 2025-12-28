import { NextRequest, NextResponse } from 'next/server'
import { getServerSession } from 'next-auth'
import { authOptions } from '@/lib/auth'
import { k8sClient } from '@/lib/k8s-client'
import { db } from '@/lib/db'
import { requirePermission } from '@/lib/permissions'
import { getUserOrganization } from '@/lib/organization-context'
import { filterByClusterRef } from '@/lib/cluster-utils'
import { validateClusterExists, validateResourceBelongsToCluster } from '@/lib/cluster-validation'
import { createErrorResponse, createSuccessResponse, handleKubernetesOperation, validateClusterNameFormat, createAuthenticationRequiredError, createPermissionDeniedError, KubernetesError } from '@/lib/api-error-handler'
import { LanguageModel, LanguageModelListParams } from '@/types/model'

// GET /api/clusters/[name]/models - List models for a specific cluster
export async function GET(
  request: NextRequest,
  { params }: { params: Promise<{ name: string }> }
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
      throw createPermissionDeniedError('view models', 'cluster-scoped models', userRole)
    }

    const { name: clusterName } = await params
    
    // Validate cluster name format
    validateClusterNameFormat(clusterName)
    
    // Validate cluster exists and user has access
    await validateClusterExists(organization.namespace, clusterName, { validateAccess: true })

    // Parse query parameters
    const url = new URL(request.url)
    const listParams: LanguageModelListParams = {
      namespace: organization.namespace,
      page: parseInt(url.searchParams.get('page') || '1'),
      limit: parseInt(url.searchParams.get('limit') || '50'),
      sortBy: (url.searchParams.get('sortBy') as any) || 'name',
      sortOrder: (url.searchParams.get('sortOrder') as any) || 'asc',
      search: url.searchParams.get('search') || undefined,
      provider: url.searchParams.getAll('provider') || undefined,
      phase: url.searchParams.getAll('phase') || undefined,
      healthy: url.searchParams.get('healthy') === 'true' ? true : url.searchParams.get('healthy') === 'false' ? false : undefined,
    }

    // Fetch models from Kubernetes namespace with proper error handling
    console.log(`Fetching models for cluster ${clusterName} from namespace:`, organization.namespace)
    
    const response = await handleKubernetesOperation(
      'list models',
      k8sClient.listLanguageModels(organization.namespace)
    )
    
    // Handle different response structures
    let rawItems = null
    if (response.body && typeof response.body === 'object') {
      rawItems = (response.body as any)?.items
    } else if (response.data && typeof response.data === 'object') {
      rawItems = (response.data as any)?.items
    } else {
      if (Array.isArray(response)) {
        rawItems = response
      } else if ((response as any)?.items) {
        rawItems = (response as any).items
      }
    }
    
    const allModels = rawItems || []
    
    // Filter models that belong to this specific cluster
    // Uses validation to handle orphaned resources gracefully
    const models = validateResourceBelongsToCluster(
      allModels, 
      clusterName, 
      { allowOrphanedResources: true }
    ) as LanguageModel[]
    
    console.log(`Found ${models.length} models for cluster ${clusterName}`)

    // Apply client-side filtering
    let filteredModels = models.filter((model: LanguageModel) => {
      // Search filter
      if (listParams.search) {
        const searchLower = listParams.search.toLowerCase()
        const nameMatch = model.metadata.name?.toLowerCase().includes(searchLower)
        const providerMatch = model.spec.provider?.toLowerCase().includes(searchLower)
        const modelNameMatch = model.spec.modelName?.toLowerCase().includes(searchLower)
        if (!nameMatch && !providerMatch && !modelNameMatch) {
          return false
        }
      }

      // Provider filter
      if (listParams.provider && listParams.provider.length > 0) {
        if (!listParams.provider.includes(model.spec.provider)) {
          return false
        }
      }

      // Phase filter
      if (listParams.phase && listParams.phase.length > 0) {
        if (!listParams.phase.includes(model.status?.phase || '')) {
          return false
        }
      }

      // Healthy filter
      if (listParams.healthy !== undefined) {
        if (model.status?.healthy !== listParams.healthy) {
          return false
        }
      }

      return true
    })

    // Sort models
    filteredModels.sort((a: LanguageModel, b: LanguageModel) => {
      let aValue: any, bValue: any
      
      switch (listParams.sortBy) {
        case 'name':
          aValue = a.metadata.name || ''
          bValue = b.metadata.name || ''
          break
        case 'provider':
          aValue = a.spec.provider || ''
          bValue = b.spec.provider || ''
          break
        case 'phase':
          aValue = a.status?.phase || ''
          bValue = b.status?.phase || ''
          break
        case 'healthy':
          aValue = a.status?.healthy ? 1 : 0
          bValue = b.status?.healthy ? 1 : 0
          break
        case 'requests':
          aValue = a.status?.metrics?.totalRequests || 0
          bValue = b.status?.metrics?.totalRequests || 0
          break
        case 'age':
          aValue = new Date(a.metadata.creationTimestamp || 0).getTime()
          bValue = new Date(b.metadata.creationTimestamp || 0).getTime()
          break
        default:
          aValue = a.metadata.name || ''
          bValue = b.metadata.name || ''
      }

      if (listParams.sortOrder === 'desc') {
        return aValue < bValue ? 1 : aValue > bValue ? -1 : 0
      }
      return aValue > bValue ? 1 : aValue < bValue ? -1 : 0
    })

    // Pagination
    const startIndex = ((listParams.page || 1) - 1) * (listParams.limit || 50)
    const endIndex = startIndex + (listParams.limit || 50)
    const paginatedModels = filteredModels.slice(startIndex, endIndex)

    return createSuccessResponse(paginatedModels, undefined, {
      total: filteredModels.length,
      page: listParams.page || 1,
      limit: listParams.limit || 50,
      cluster: clusterName,
    })

  } catch (error) {
    console.error('Error fetching cluster models:', error)
    return createErrorResponse(error, 'Failed to fetch cluster models')
  }
}

// POST /api/clusters/[name]/models - Create a new model for a specific cluster
export async function POST(
  request: NextRequest,
  { params }: { params: Promise<{ name: string }> }
) {
  try {
    // Get user's selected organization
    const { user, organization, userRole } = await getUserOrganization(request)
    
    if (!user?.id) {
      throw createAuthenticationRequiredError()
    }

    // Check permissions
    const hasPermission = await requirePermission(user.id, organization.id, 'create')
    if (!hasPermission) {
      throw createPermissionDeniedError('create models', 'cluster-scoped models', userRole)
    }

    const { name: clusterName } = await params
    
    // Validate cluster name format
    validateClusterNameFormat(clusterName)
    
    // Validate cluster exists and user has access
    await validateClusterExists(organization.namespace, clusterName, { validateAccess: true })

    const body = await request.json()
    
    // Validate required fields
    if (!body.name) {
      return createErrorResponse(
        new Error('Model name is required'),
        'Invalid request data'
      )
    }
    
    if (!body.provider) {
      return createErrorResponse(
        new Error('Provider is required'),
        'Invalid request data'
      )
    }
    
    if (!body.modelName) {
      return createErrorResponse(
        new Error('Model name is required'),
        'Invalid request data'
      )
    }

    // Create model spec with clusterRef automatically set
    const modelSpec = {
      apiVersion: 'langop.io/v1alpha1',
      kind: 'LanguageModel',
      metadata: {
        name: body.name,
        namespace: organization.namespace,
        labels: {
          'langop.io/organization-id': organization.id,
          'langop.io/cluster': clusterName,
          'langop.io/managed-by': 'language-operator-dashboard'
        }
      },
      spec: {
        clusterRef: clusterName, // Automatically set clusterRef to the cluster name
        provider: body.provider,
        modelName: body.modelName,
        endpoint: body.endpoint,
        ...(body.apiKeySecretName && { 
          apiKeySecretName: body.apiKeySecretName,
          apiKeySecretKey: body.apiKeySecretKey || 'api-key'
        }),
        ...(body.temperature !== undefined && { temperature: body.temperature }),
        ...(body.maxTokens !== undefined && { maxTokens: body.maxTokens }),
        ...(body.topP !== undefined && { topP: body.topP }),
      }
    }

    console.log(`Creating model ${body.name} for cluster ${clusterName} in namespace:`, organization.namespace)
    console.log('Model spec being sent to Kubernetes:', JSON.stringify(modelSpec, null, 2))
    
    // Create the model in Kubernetes
    const response = await handleKubernetesOperation(
      'create model',
      k8sClient.createLanguageModel(organization.namespace, modelSpec)
    )
    
    console.log(`Model ${body.name} created successfully for cluster ${clusterName}`)

    return createSuccessResponse(response, 'Model created successfully')

  } catch (error) {
    console.error('Error creating cluster model:', error)
    return createErrorResponse(error, 'Failed to create cluster model')
  }
}