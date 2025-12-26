import { NextRequest, NextResponse } from 'next/server'
import { z } from 'zod'
import { createClient } from '@clickhouse/client'

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

// ClickHouse client
const clickhouse = createClient({
  host: process.env.CLICKHOUSE_URL || 'http://localhost:8123',
  database: 'langop',
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

    const startTime = new Date(Date.now() - query.timeRange)
    const endTime = new Date()

    // Query ClickHouse for agent executions (grouped by TraceId)
    const sql = `
      SELECT 
        TraceId as traceId,
        min(Timestamp) as startTime,
        max(addNanoseconds(Timestamp, Duration)) as endTime,
        max(Duration) / 1000000 as duration,
        count() as spanCount,
        any(SpanName) as rootSpanName,
        countIf(StatusCode != 'STATUS_CODE_OK') > 0 ? 'error' : 'success' as status
      FROM langop.otel_traces
      WHERE Timestamp >= {startTime:DateTime64(9)}
        AND Timestamp <= {endTime:DateTime64(9)}
        AND SpanAttributes['agent.name'] = {agentName:String}
      GROUP BY TraceId
      ORDER BY startTime DESC
      LIMIT {limit:UInt32}
    `

    const resultSet = await clickhouse.query({
      query: sql,
      query_params: {
        startTime: startTime.getTime() * 1000000, // Convert to nanoseconds
        endTime: endTime.getTime() * 1000000, // Convert to nanoseconds
        agentName,
        limit: query.limit,
      },
    })

    const rows = await resultSet.json()
    
    const executions: AgentExecution[] = rows.data.map((row: any) => ({
      traceId: row.traceId,
      executionId: `exec_${row.traceId.substring(0, 8)}`,
      startTime: new Date(row.startTime),
      endTime: new Date(row.endTime), 
      duration: Math.round(row.duration),
      status: row.status,
      rootSpanName: row.rootSpanName,
      spanCount: row.spanCount,
    }))

    return NextResponse.json({
      success: true,
      data: executions
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

    console.error('Error querying ClickHouse:', error)
    
    return NextResponse.json(
      { success: false, error: 'Failed to fetch agent executions' },
      { status: 500 }
    )
  }
}