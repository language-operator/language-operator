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

    // For kubectl-proxy, use direct HTTP request instead of the ClickHouse client
    // because the client library doesn't work well with the proxy format
    let rows: any
    if (process.env.KUBERNETES_SERVER_URL?.includes('kubectl-proxy') && process.env.CLICKHOUSE_DIRECT_ACCESS !== 'true') {
      const baseUrl = `${process.env.KUBERNETES_SERVER_URL}/api/v1/namespaces/language-operator/services/language-operator-clickhouse:8123/proxy`
      
      // Build the SQL query with parameters substituted directly
      const sqlWithParams = sql.replace('{traceIdPattern:String}', `'${traceIdPrefix}%'`)

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
      if (lines.length === 0 || lines[0] === '') {
        rows = { data: [] }
      } else {
        const data = lines.map(line => {
          const fields = line.split('\t')
          
          // Helper function to safely parse JSON
          const safeJsonParse = (str: string, defaultValue: any = null) => {
            if (!str || str === 'null' || str === '' || str === '\\N') return defaultValue
            try {
              return JSON.parse(str)
            } catch (e) {
              return defaultValue
            }
          }
          
          return {
            traceId: fields[0] || '',
            spanId: fields[1] || '',
            parentSpanId: fields[2] || null,
            spanName: fields[3] || '',
            startTime: fields[4] || '',
            endTime: fields[5] || '',
            duration: fields[6] ? parseFloat(fields[6]) : 0,
            statusCode: fields[7] || 'STATUS_CODE_UNSET',
            attributes: safeJsonParse(fields[8], {}),
            eventTimestamps: safeJsonParse(fields[9], []),
            eventNames: safeJsonParse(fields[10], []),
            eventAttributes: safeJsonParse(fields[11], [])
          }
        })
        rows = { data }
      }
    } else {
      const resultSet = await clickhouse.query({
        query: sql,
        query_params: {
          traceIdPattern: `${traceIdPrefix}%` // Match trace IDs that start with our pattern
        },
      })
      rows = await resultSet.json()
    }
    
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