import { NextRequest, NextResponse } from 'next/server'
import { getServerSession } from 'next-auth'
import { authOptions } from '@/lib/auth'
import { k8sClient } from '@/lib/k8s-client'
import { db } from '@/lib/db'
import { requirePermission } from '@/lib/permissions'
import { LanguageModel, LanguageModelListParams, LanguageModelFormData } from '@/types/model'
import { safeValidateLanguageModel } from '@/lib/validation'

// GET /api/models - List all models for user's organization
export async function GET(request: NextRequest) {
  try {
    const session = await getServerSession(authOptions)
    if (!session?.user?.email) {
      return NextResponse.json({ error: 'Unauthorized' }, { status: 401 })
    }

    // Get user's current organization
    const user = await db.user.findUnique({
      where: { email: session.user.email },
      include: { 
        memberships: { 
          include: { organization: true } 
        } 
      },
    })

    if (!user || user.memberships.length === 0) {
      return NextResponse.json({ error: 'No organization found' }, { status: 404 })
    }

    const organization = user.memberships[0].organization
    
    // Check permissions
    const hasPermission = await requirePermission(user.id, organization.id, 'view')
    if (!hasPermission) {
      return NextResponse.json({ error: 'Insufficient permissions' }, { status: 403 })
    }

    // Parse query parameters
    const url = new URL(request.url)
    const params: LanguageModelListParams = {
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

    // Fetch models from Kubernetes
    const response = await k8sClient.listLanguageModels(organization.namespace)
    const models = (response.data as any)?.items || []

    // Apply client-side filtering
    let filteredModels = models.filter((model: LanguageModel) => {
      // Search filter
      if (params.search) {
        const searchLower = params.search.toLowerCase()
        const nameMatch = model.metadata.name?.toLowerCase().includes(searchLower)
        const providerMatch = model.spec.provider?.toLowerCase().includes(searchLower)
        const modelNameMatch = model.spec.modelName?.toLowerCase().includes(searchLower)
        if (!nameMatch && !providerMatch && !modelNameMatch) return false
      }

      // Provider filter
      if (params.provider && params.provider.length > 0) {
        if (!params.provider.includes(model.spec.provider)) return false
      }

      // Phase filter
      if (params.phase && params.phase.length > 0) {
        if (!params.phase.includes(model.status?.phase || '')) return false
      }

      // Healthy filter
      if (params.healthy !== undefined) {
        if (model.status?.healthy !== params.healthy) return false
      }

      return true
    })

    // Sort models
    filteredModels.sort((a: LanguageModel, b: LanguageModel) => {
      let aValue: any, bValue: any
      
      switch (params.sortBy) {
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

      if (params.sortOrder === 'desc') {
        return aValue < bValue ? 1 : aValue > bValue ? -1 : 0
      }
      return aValue > bValue ? 1 : aValue < bValue ? -1 : 0
    })

    // Pagination
    const startIndex = ((params.page || 1) - 1) * (params.limit || 50)
    const endIndex = startIndex + (params.limit || 50)
    const paginatedModels = filteredModels.slice(startIndex, endIndex)

    return NextResponse.json({
      success: true,
      data: paginatedModels,
      total: filteredModels.length,
      page: params.page || 1,
      limit: params.limit || 50,
    })

  } catch (error) {
    console.error('Error fetching models:', error)
    return NextResponse.json(
      { error: 'Failed to fetch models' },
      { status: 500 }
    )
  }
}

// POST /api/models - Create a new model
export async function POST(request: NextRequest) {
  try {
    const session = await getServerSession(authOptions)
    if (!session?.user?.email) {
      return NextResponse.json({ error: 'Unauthorized' }, { status: 401 })
    }

    // Get user's current organization
    const user = await db.user.findUnique({
      where: { email: session.user.email },
      include: { 
        memberships: { 
          include: { organization: true } 
        } 
      },
    })

    if (!user || user.memberships.length === 0) {
      return NextResponse.json({ error: 'No organization found' }, { status: 404 })
    }

    const organization = user.memberships[0].organization
    
    // Check permissions
    const hasPermission = await requirePermission(user.id, organization.id, 'create')
    if (!hasPermission) {
      return NextResponse.json({ error: 'Insufficient permissions' }, { status: 403 })
    }

    // Parse and validate request body
    const body = await request.json()
    
    // Basic validation
    if (!body.name || !body.provider || !body.modelName) {
      return NextResponse.json(
        { error: 'Missing required fields: name, provider, modelName' },
        { status: 400 }
      )
    }

    const formData: LanguageModelFormData = body

    // Convert form data to LanguageModel CRD
    const modelData: LanguageModel = {
      apiVersion: 'langop.io/v1alpha1',
      kind: 'LanguageModel',
      metadata: {
        name: formData.name,
        namespace: organization.namespace,
        labels: {
          'langop.io/organization': organization.id,
          'langop.io/created-by': user.id,
        },
        annotations: {
          'langop.io/created-by-email': user.email!,
          'langop.io/created-at': new Date().toISOString(),
        },
      },
      spec: {
        provider: formData.provider,
        modelName: formData.modelName,
        ...(formData.endpoint && { endpoint: formData.endpoint }),
        ...(formData.apiKeySecretName && {
          apiKeySecretRef: {
            name: formData.apiKeySecretName,
            key: formData.apiKeySecretKey || 'api-key',
          },
        }),
        ...(formData.temperature || formData.maxTokens || formData.topP) && {
          configuration: {
            ...(formData.temperature && { temperature: formData.temperature }),
            ...(formData.maxTokens && { maxTokens: formData.maxTokens }),
            ...(formData.topP && { topP: formData.topP }),
          },
        },
        ...(formData.enableCostTracking) && {
          costTracking: {
            enabled: true,
            currency: formData.currency || 'USD',
            ...(formData.inputTokenCost && { inputTokenCost: formData.inputTokenCost }),
            ...(formData.outputTokenCost && { outputTokenCost: formData.outputTokenCost }),
          },
        },
        ...(formData.enableCaching) && {
          caching: {
            enabled: true,
            backend: formData.cacheBackend || 'memory',
            ...(formData.cacheTtl && { ttl: formData.cacheTtl }),
          },
        },
        ...(formData.requestsPerMinute || formData.tokensPerMinute || formData.concurrentRequests) && {
          rateLimits: {
            ...(formData.requestsPerMinute && { requestsPerMinute: formData.requestsPerMinute }),
            ...(formData.tokensPerMinute && { tokensPerMinute: formData.tokensPerMinute }),
            ...(formData.concurrentRequests && { concurrentRequests: formData.concurrentRequests }),
          },
        },
        ...(formData.enableRetry) && {
          retryPolicy: {
            enabled: true,
            ...(formData.maxRetries && { maxRetries: formData.maxRetries }),
          },
        },
        observability: {
          metrics: formData.enableMetrics !== false,
          tracing: formData.enableTracing || false,
          logging: {
            level: formData.logLevel || 'info',
          },
        },
      },
    }

    // Validate the model CRD structure
    const validationResult = safeValidateLanguageModel(modelData)
    if (!validationResult.success) {
      return NextResponse.json(
        { 
          error: 'Invalid model configuration',
          details: validationResult.error.issues.map(err => ({
            path: err.path.join('.'),
            message: err.message
          }))
        },
        { status: 400 }
      )
    }

    const model = validationResult.data

    // Create model in Kubernetes
    const response = await k8sClient.createLanguageModel(organization.namespace, model)
    const createdModel = response.data as LanguageModel

    // Audit log
    console.log(`User ${user.email} created LanguageModel ${formData.name} in organization ${organization.name}`)

    return NextResponse.json({
      success: true,
      data: createdModel,
    })

  } catch (error) {
    console.error('Error creating model:', error)
    return NextResponse.json(
      { error: 'Failed to create model' },
      { status: 500 }
    )
  }
}