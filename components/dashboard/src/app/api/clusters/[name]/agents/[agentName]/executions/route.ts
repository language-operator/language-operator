import { NextRequest, NextResponse } from 'next/server'
import { z } from 'zod'

// Types for API responses
export interface AgentExecution {
  traceId: string
  executionId: string
  startTime: Date
  endTime: Date
  duration: number
  status: 'success' | 'error' | 'running'
  rootSpanName: string
  spanCount: number
}

// Query parameters schema
const QuerySchema = z.object({
  limit: z.coerce.number().int().positive().max(1000).default(50),
  timeRange: z.coerce.number().int().positive().default(24 * 60 * 60 * 1000), // 24h in ms
})

export async function GET(
  request: NextRequest,
  { params }: { params: Promise<{ name: string; agentName: string }> }
) {
  try {
    const { name: clusterName, agentName } = await params
    const { searchParams } = new URL(request.url)
    
    // Validate query parameters
    const query = QuerySchema.parse({
      limit: searchParams.get('limit'),
      timeRange: searchParams.get('timeRange'),
    })

    // Proxy request to Go telemetry service
    const telemetryServiceUrl = process.env.TELEMETRY_SERVICE_URL || 'http://localhost:8080'
    const proxyUrl = `${telemetryServiceUrl}/api/clusters/${encodeURIComponent(clusterName)}/agents/${encodeURIComponent(agentName)}/executions?limit=${query.limit}&timeRange=${query.timeRange}`
    
    const response = await fetch(proxyUrl, {
      method: 'GET',
      headers: {
        'Content-Type': 'application/json',
      },
    })

    if (!response.ok) {
      console.error(`Telemetry service error: ${response.status} ${response.statusText}`)
      return NextResponse.json(
        { success: false, error: 'Failed to fetch executions from telemetry service' },
        { status: response.status }
      )
    }

    const data = await response.json()
    
    // Transform dates from string to Date objects for frontend compatibility
    if (data.success && Array.isArray(data.data)) {
      data.data = data.data.map((execution: any) => ({
        ...execution,
        startTime: new Date(execution.startTime),
        endTime: new Date(execution.endTime),
      }))
    }

    return NextResponse.json(data)

  } catch (error) {
    if (error instanceof z.ZodError) {
      return NextResponse.json(
        { 
          success: false, 
          error: 'Invalid query parameters',
          details: error.issues 
        },
        { status: 400 }
      )
    }

    console.error('Error fetching agent executions:', error)
    
    return NextResponse.json(
      { success: false, error: 'Failed to fetch agent executions' },
      { status: 500 }
    )
  }
}