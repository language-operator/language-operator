import { NextRequest, NextResponse } from 'next/server'
import { createClickHouseClient } from '@/lib/clickhouse-config'

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

// ClickHouse client
const clickhouse = createClickHouseClient()

export async function GET(
  request: NextRequest,
  { params }: { params: Promise<{ name: string; agentName: string; executionId: string }> }
) {
  try {
    const { name: clusterName, agentName, executionId } = await params
    
    // Extract trace ID from execution ID (reverse of exec_{traceId.substring(0,8)})
    const traceIdPrefix = executionId.startsWith('exec_') ? executionId.substring(5) : executionId

    // Query ClickHouse for all spans in this trace
    const sql = `
      SELECT 
        TraceId as traceId,
        SpanId as spanId,
        ParentSpanId as parentSpanId,
        SpanName as spanName,
        Timestamp as startTime,
        addNanoseconds(Timestamp, Duration) as endTime,
        Duration / 1000000 as duration,
        StatusCode as statusCode,
        SpanAttributes as attributes,
        Events.Timestamp as eventTimestamps,
        Events.Name as eventNames,
        Events.Attributes as eventAttributes
      FROM langop.otel_traces
      WHERE TraceId LIKE {traceIdPattern:String}
      ORDER BY Timestamp ASC
    `

    const resultSet = await clickhouse.query({
      query: sql,
      query_params: {
        traceIdPattern: `${traceIdPrefix}%` // Match trace IDs that start with our pattern
      },
    })

    const rows = await resultSet.json()
    
    if (rows.data.length === 0) {
      return NextResponse.json({
        success: true,
        data: {
          traceId: traceIdPrefix,
          executionId,
          spans: []
        }
      })
    }

    const spans: TraceSpan[] = rows.data.map((row: any) => {
      // Parse events arrays
      const events: SpanEvent[] = []
      if (row.eventTimestamps && row.eventNames) {
        for (let i = 0; i < row.eventTimestamps.length; i++) {
          events.push({
            time: new Date(row.eventTimestamps[i]),
            name: row.eventNames[i] || '',
            attributes: row.eventAttributes?.[i] || {}
          })
        }
      }

      return {
        spanId: row.spanId,
        parentSpanId: row.parentSpanId || undefined,
        spanName: row.spanName,
        startTime: new Date(row.startTime),
        endTime: new Date(row.endTime),
        duration: Math.round(row.duration),
        status: row.statusCode === 'STATUS_CODE_OK' ? 'success' : 'error',
        attributes: row.attributes || {},
        events
      }
    })

    return NextResponse.json({
      success: true,
      data: {
        traceId: rows.data.length > 0 ? (rows.data[0] as any).traceId : traceIdPrefix,
        executionId,
        spans
      }
    })

  } catch (error) {
    console.error('Error querying ClickHouse for trace data:', error)
    
    return NextResponse.json(
      { success: false, error: 'Failed to fetch trace data' },
      { status: 500 }
    )
  }
}