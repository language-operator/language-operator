import { NextRequest, NextResponse } from 'next/server'
import { getServerSession } from 'next-auth'
import { authOptions } from '@/lib/auth'

// POST /api/models/discover - Discover available models from an endpoint
export async function POST(request: NextRequest) {
  try {
    const session = await getServerSession(authOptions)
    if (!session?.user?.email) {
      return NextResponse.json({ error: 'Unauthorized' }, { status: 401 })
    }

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

        if (response.ok) {
          const data = await response.json()
          
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
        }
      }
    } catch (error) {
      console.warn('Failed to fetch models from endpoint:', error)
    }

    // If we couldn't fetch models, return an error response instead of fake data
    if (models.length === 0) {
      return NextResponse.json(
        { error: 'Failed to discover models from the API endpoint. Please verify the endpoint URL and credentials are correct.' },
        { status: 400 }
      )
    }

    return NextResponse.json({
      success: true,
      models: models.slice(0, 50), // Limit to 50 models for performance
      endpoint,
      provider,
    })

  } catch (error) {
    console.error('Error discovering models:', error)
    return NextResponse.json(
      { error: 'Failed to discover models' },
      { status: 500 }
    )
  }
}