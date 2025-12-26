import { NextRequest, NextResponse } from 'next/server'
import { z } from 'zod'

// Types based on the issue requirements
export interface AgentExecution {
  traceId: string
  executionId: string
  startTime: Date
  endTime: Date
  duration: number
  status: 'success' | 'error' | 'running'
  rootSpanName: string
  spans: TraceSpan[]
}

export interface TraceSpan {
  spanId: string
  parentSpanId?: string
  spanName: string
  startTime: Date
  endTime: Date
  duration: number
  status: string
  attributes: Record<string, any>
  events: SpanEvent[]
}

export interface SpanEvent {
  time: Date
  name: string
  attributes: Record<string, any>
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

    // For now, return mock data since telemetry integration is complex
    // TODO: Integrate with ClickHouse telemetry adapter
    const mockExecutions: AgentExecution[] = [
      {
        traceId: 'trace_001',
        executionId: 'exec_001',
        startTime: new Date(Date.now() - 2 * 60 * 60 * 1000), // 2 hours ago
        endTime: new Date(Date.now() - 2 * 60 * 60 * 1000 + 45000), // 45s duration
        duration: 45000,
        status: 'success',
        rootSpanName: 'agent_execution',
        spans: []
      },
      {
        traceId: 'trace_002', 
        executionId: 'exec_002',
        startTime: new Date(Date.now() - 4 * 60 * 60 * 1000), // 4 hours ago
        endTime: new Date(Date.now() - 4 * 60 * 60 * 1000 + 32000), // 32s duration
        duration: 32000,
        status: 'success',
        rootSpanName: 'agent_execution',
        spans: []
      },
      {
        traceId: 'trace_003',
        executionId: 'exec_003', 
        startTime: new Date(Date.now() - 6 * 60 * 60 * 1000), // 6 hours ago
        endTime: new Date(Date.now() - 6 * 60 * 60 * 1000 + 12000), // 12s duration
        duration: 12000,
        status: 'error',
        rootSpanName: 'agent_execution',
        spans: []
      }
    ]

    return NextResponse.json({
      success: true,
      data: mockExecutions.slice(0, query.limit)
    })

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