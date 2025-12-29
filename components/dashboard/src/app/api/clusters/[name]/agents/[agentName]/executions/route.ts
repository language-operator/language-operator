import { NextRequest, NextResponse } from 'next/server'
import { z } from 'zod'
import { createClickHouseClient } from '@/lib/clickhouse-config'

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
const clickhouse = createClickHouseClient()

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
        countIf(StatusCode = 'STATUS_CODE_ERROR') > 0 ? 'error' : 'success' as status
      FROM langop.otel_traces
      WHERE Timestamp >= {startTime:DateTime64(9)}
        AND Timestamp <= {endTime:DateTime64(9)}
        AND SpanAttributes['agent.name'] = {agentName:String}
        AND SpanName != 'agent.reconcile'
      GROUP BY TraceId
      ORDER BY startTime DESC
      LIMIT {limit:UInt32}
    `


    // For kubectl-proxy, use direct HTTP request instead of the ClickHouse client
    // because the client library doesn't work well with the proxy format
    let rows: any
    if (process.env.KUBERNETES_SERVER_URL?.includes('kubectl-proxy') && process.env.CLICKHOUSE_DIRECT_ACCESS !== 'true') {
      const baseUrl = `${process.env.KUBERNETES_SERVER_URL}/api/v1/namespaces/language-operator/services/language-operator-clickhouse:8123/proxy`
      
      // Build the SQL query with parameters substituted directly
      const sqlWithParams = sql
        .replace('{startTime:DateTime64(9)}', `fromUnixTimestamp64Nano(${startTime.getTime() * 1000000})`)
        .replace('{endTime:DateTime64(9)}', `fromUnixTimestamp64Nano(${endTime.getTime() * 1000000})`)
        .replace('{agentName:String}', `'${agentName}'`)
        .replace('{limit:UInt32}', query.limit.toString())

      const response = await fetch(`${baseUrl}?database=${process.env.CLICKHOUSE_DATABASE || 'langop'}&query=${encodeURIComponent(sqlWithParams)}`, {
        headers: {
          'Authorization': `Basic ${Buffer.from(`${process.env.CLICKHOUSE_USERNAME || 'langop'}:${process.env.CLICKHOUSE_PASSWORD || 'langop'}`).toString('base64')}`
        }
      })
      if (!response.ok) {
        throw new Error(`ClickHouse query failed: ${response.status} ${response.statusText}`)
      }
      
      const text = await response.text()
      
      // Parse tab-separated values response
      const lines = text.trim().split('\n')
      const data = lines.map(line => {
        const fields = line.split('\t')
        return {
          traceId: fields[0],
          startTime: new Date(fields[1]),
          endTime: new Date(fields[2]),
          duration: parseFloat(fields[3]),
          spanCount: parseInt(fields[4]),
          rootSpanName: fields[5],
          status: fields[6]
        }
      })
      
      rows = { data }
    } else {
      const resultSet = await clickhouse.query({
        query: sql,
        query_params: {
          startTime: startTime.getTime() * 1000000, // Convert to nanoseconds
          endTime: endTime.getTime() * 1000000, // Convert to nanoseconds
          agentName,
          limit: query.limit,
        },
      })
      rows = await resultSet.json()
    }

    
    // Handle different response formats from ClickHouse
    const data = rows.data || rows
    if (!Array.isArray(data)) {
      return NextResponse.json({ data: [] })
    }
    
    const executions: AgentExecution[] = data.map((row: any) => ({
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
    console.error('Error stack:', error instanceof Error ? error.stack : 'No stack')
    
    return NextResponse.json(
      { 
        success: false, 
        error: 'Failed to fetch agent executions',
        details: error instanceof Error ? error.message : 'Unknown error'
      },
      { status: 500 }
    )
  }
}