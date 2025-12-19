import { NextRequest, NextResponse } from 'next/server'
import { getServerSession } from 'next-auth'
import { authOptions } from '@/lib/auth'
import { requirePermission } from '@/lib/permissions'
import { getUserOrganization } from '@/lib/organization-context'
import { validateClusterExists } from '@/lib/cluster-validation'
import { validateClusterNameFormat } from '@/lib/api-error-handler'

// POST /api/clusters/[name]/models/discover - Discover available models from an endpoint
export async function POST(
  request: NextRequest,
  { params }: { params: Promise<{ name: string }> }
) {
  try {
    const { name: clusterName } = await params
    
    // Validate cluster name format
    validateClusterNameFormat(clusterName)
    
    // Get user's selected organization
    const { user, organization, userRole } = await getUserOrganization(request)
    
    if (!user?.id) {
      return NextResponse.json({ error: 'Unauthorized' }, { status: 401 })
    }

    // Check permissions
    const hasPermission = await requirePermission(user.id, organization.id, 'create')
    if (!hasPermission) {
      return NextResponse.json({ error: 'Insufficient permissions' }, { status: 403 })
    }
    
    // Validate cluster exists and user has access
    await validateClusterExists(organization.namespace, clusterName, { validateAccess: true })

    const body = await request.json()
    const { endpoint, provider, apiKey } = body

    if (!endpoint) {
      return NextResponse.json(
        { error: 'Endpoint is required' }, 
        { status: 400 }
      )
    }

    // Validate endpoint URL
    try {
      new URL(endpoint)
    } catch {
      return NextResponse.json(
        { error: 'Invalid endpoint URL' }, 
        { status: 400 }
      )
    }

    let models: string[] = []

    try {
      // Try to fetch models from the endpoint
      if (provider === 'openai-compatible' || provider === 'openai') {
        // Try OpenAI-compatible /v1/models endpoint
        const modelsUrl = endpoint.endsWith('/v1') ? `${endpoint}/models` : `${endpoint}/v1/models`
        
        console.log(`🔍 Discovering models for cluster ${clusterName} from: ${modelsUrl}`)
        
        const headers: Record<string, string> = {
          'Content-Type': 'application/json',
        }
        
        if (apiKey) {
          headers['Authorization'] = `Bearer ${apiKey}`
        }

        const response = await fetch(modelsUrl, {
          method: 'GET',
          headers,
          signal: AbortSignal.timeout(10000) // 10 second timeout
        })

        console.log(`📡 Response status: ${response.status} for ${modelsUrl}`)

        if (response.ok) {
          const data = await response.json()
          console.log(`📊 Response data:`, data)
          
          // OpenAI-compatible format: { data: [{ id: "model-name", ... }] }
          if (data.data && Array.isArray(data.data)) {
            models = data.data.map((model: any) => model.id || model.name).filter(Boolean)
          }
          // Some providers return different formats
          else if (data.models && Array.isArray(data.models)) {
            models = data.models.map((model: any) => 
              typeof model === 'string' ? model : model.id || model.name
            ).filter(Boolean)
          }
          // Direct array format
          else if (Array.isArray(data)) {
            models = data.map((model: any) => 
              typeof model === 'string' ? model : model.id || model.name
            ).filter(Boolean)
          }
        } else {
          console.error(`❌ Failed to fetch models: ${response.status} ${response.statusText}`)
        }
      } else if (provider === 'anthropic') {
        // Anthropic doesn't have a models discovery endpoint
        // Return common Claude models
        models = [
          'claude-3-5-sonnet-20241022',
          'claude-3-opus-20240229', 
          'claude-3-sonnet-20240229',
          'claude-3-haiku-20240307'
        ]
      }
    } catch (error) {
      console.error('❌ Failed to fetch models from endpoint:', error)
    }

    console.log(`✅ Discovered ${models.length} models for cluster ${clusterName}:`, models)

    return NextResponse.json({
      success: true,
      models: models.slice(0, 50), // Limit to 50 models for performance
      endpoint,
      provider,
      cluster: clusterName,
    })

  } catch (error) {
    console.error('Error discovering models:', error)
    return NextResponse.json(
      { error: 'Failed to discover models' },
      { status: 500 }
    )
  }
}