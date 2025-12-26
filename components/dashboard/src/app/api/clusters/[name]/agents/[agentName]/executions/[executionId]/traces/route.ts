import { NextRequest, NextResponse } from 'next/server'

// Local type definitions to avoid import issues
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

export async function GET(
  request: NextRequest,
  { params }: { params: Promise<{ name: string; agentName: string; executionId: string }> }
) {
  try {
    const { name: clusterName, agentName, executionId } = await params

    // Mock trace data with hierarchical spans for flame chart visualization
    const mockSpans: TraceSpan[] = [
      {
        spanId: 'span_001',
        spanName: 'agent_execution',
        startTime: new Date(Date.now() - 2 * 60 * 60 * 1000),
        endTime: new Date(Date.now() - 2 * 60 * 60 * 1000 + 45000),
        duration: 45000,
        status: 'success',
        attributes: {
          'agent.name': agentName,
          'execution.id': executionId,
          'task.type': 'synthesis'
        },
        events: [
          {
            time: new Date(Date.now() - 2 * 60 * 60 * 1000),
            name: 'execution_started',
            attributes: {
              'execution.mode': 'scheduled'
            }
          }
        ]
      },
      {
        spanId: 'span_002',
        parentSpanId: 'span_001',
        spanName: 'task_planning',
        startTime: new Date(Date.now() - 2 * 60 * 60 * 1000 + 1000),
        endTime: new Date(Date.now() - 2 * 60 * 60 * 1000 + 8000),
        duration: 7000,
        status: 'success',
        attributes: {
          'task.name': 'analyze_user_request',
          'llm.model': 'gpt-4'
        },
        events: []
      },
      {
        spanId: 'span_003',
        parentSpanId: 'span_001', 
        spanName: 'tool_execution',
        startTime: new Date(Date.now() - 2 * 60 * 60 * 1000 + 8500),
        endTime: new Date(Date.now() - 2 * 60 * 60 * 1000 + 25000),
        duration: 16500,
        status: 'success',
        attributes: {
          'tool.name': 'web_search',
          'tool.method': 'search',
          'tool.input_size': '256'
        },
        events: [
          {
            time: new Date(Date.now() - 2 * 60 * 60 * 1000 + 9000),
            name: 'tool_call_started',
            attributes: {
              'tool.endpoint': 'https://api.search.com/v1/search'
            }
          },
          {
            time: new Date(Date.now() - 2 * 60 * 60 * 1000 + 24000),
            name: 'tool_response_received',
            attributes: {
              'tool.response_size': '1024'
            }
          }
        ]
      },
      {
        spanId: 'span_004',
        parentSpanId: 'span_003',
        spanName: 'api_request',
        startTime: new Date(Date.now() - 2 * 60 * 60 * 1000 + 10000),
        endTime: new Date(Date.now() - 2 * 60 * 60 * 1000 + 23000),
        duration: 13000,
        status: 'success',
        attributes: {
          'http.method': 'POST',
          'http.url': 'https://api.search.com/v1/search',
          'http.status_code': '200'
        },
        events: []
      },
      {
        spanId: 'span_005',
        parentSpanId: 'span_001',
        spanName: 'response_generation',
        startTime: new Date(Date.now() - 2 * 60 * 60 * 1000 + 25500),
        endTime: new Date(Date.now() - 2 * 60 * 60 * 1000 + 44000),
        duration: 18500,
        status: 'success',
        attributes: {
          'llm.model': 'gpt-4',
          'llm.tokens.input': '1500',
          'llm.tokens.output': '800',
          'llm.cost': '0.045'
        },
        events: [
          {
            time: new Date(Date.now() - 2 * 60 * 60 * 1000 + 26000),
            name: 'llm_request_sent',
            attributes: {
              'llm.provider': 'openai'
            }
          },
          {
            time: new Date(Date.now() - 2 * 60 * 60 * 1000 + 43000),
            name: 'llm_response_received',
            attributes: {
              'llm.completion_reason': 'stop'
            }
          }
        ]
      }
    ]

    // Filter spans for the requested execution
    const executionSpans = mockSpans.filter(span => 
      span.attributes['execution.id'] === executionId || 
      !span.attributes['execution.id'] // Include spans without execution ID for demo
    )

    return NextResponse.json({
      success: true,
      data: {
        traceId: `trace_${executionId.split('_')[1]}`,
        executionId,
        spans: executionSpans
      }
    })

  } catch (error) {
    console.error('Error fetching trace details:', error)
    
    return NextResponse.json(
      { success: false, error: 'Failed to fetch trace details' },
      { status: 500 }
    )
  }
}