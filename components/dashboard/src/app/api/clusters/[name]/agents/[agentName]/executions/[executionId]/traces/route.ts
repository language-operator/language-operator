import { NextRequest, NextResponse } from 'next/server'

// Types for trace data
interface TraceSpan {
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

interface SpanEvent {
  time: Date
  name: string
  attributes: Record<string, any>
}

interface TraceData {
  traceId: string
  executionId: string
  spans: TraceSpan[]
}

export async function GET(
  request: NextRequest,
  { params }: { params: Promise<{ name: string; agentName: string; executionId: string }> }
) {
  try {
    const { name: clusterName, agentName, executionId } = await params

    // Proxy request to Go telemetry service
    const telemetryServiceUrl = process.env.TELEMETRY_SERVICE_URL || 'http://localhost:8080'
    const proxyUrl = `${telemetryServiceUrl}/api/clusters/${encodeURIComponent(clusterName)}/agents/${encodeURIComponent(agentName)}/executions/${encodeURIComponent(executionId)}/traces`
    
    const response = await fetch(proxyUrl, {
      method: 'GET',
      headers: {
        'Content-Type': 'application/json',
      },
    })

    if (!response.ok) {
      console.error(`Telemetry service error: ${response.status} ${response.statusText}`)
      return NextResponse.json(
        { success: false, error: 'Failed to fetch traces from telemetry service' },
        { status: response.status }
      )
    }

    const data = await response.json()
    
    // Transform dates from string to Date objects for frontend compatibility
    if (data.success && data.data && Array.isArray(data.data.spans)) {
      data.data.spans = data.data.spans.map((span: any) => ({
        ...span,
        startTime: new Date(span.startTime),
        endTime: new Date(span.endTime),
        events: span.events?.map((event: any) => ({
          ...event,
          time: new Date(event.time),
        })) || [],
      }))
    }

    return NextResponse.json(data)

  } catch (error) {
    console.error('Error fetching trace data:', error)
    
    return NextResponse.json(
      { success: false, error: 'Failed to fetch trace data' },
      { status: 500 }
    )
  }
}