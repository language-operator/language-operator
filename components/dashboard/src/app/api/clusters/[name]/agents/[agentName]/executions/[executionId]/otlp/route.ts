import { NextRequest, NextResponse } from 'next/server'
import { createClient } from '@clickhouse/client'

// OpenTelemetry OTLP format interfaces
interface OtlpAttribute {
  key: string
  value: {
    stringValue?: string
    intValue?: string
    doubleValue?: number
    boolValue?: boolean
    arrayValue?: { values: OtlpAttributeValue[] }
    kvlistValue?: { values: OtlpAttribute[] }
  }
}

interface OtlpAttributeValue {
  stringValue?: string
  intValue?: string
  doubleValue?: number
  boolValue?: boolean
}

interface OtlpSpan {
  traceId: string
  spanId: string
  parentSpanId?: string
  name: string
  kind: number
  startTimeUnixNano: string
  endTimeUnixNano: string
  attributes: OtlpAttribute[]
  events: OtlpEvent[]
  status: {
    code: string
    message?: string
  }
}

interface OtlpEvent {
  timeUnixNano: string
  name: string
  attributes: OtlpAttribute[]
}

interface OtlpDocument {
  resourceSpans: {
    resource: {
      attributes: OtlpAttribute[]
    }
    scopeSpans: {
      scope: {
        name: string
        version?: string
      }
      spans: OtlpSpan[]
    }[]
  }[]
}

// ClickHouse client
const clickhouse = createClient({
  host: process.env.CLICKHOUSE_URL || 'http://localhost:8123',
  database: 'langop',
})

function convertAttributesToOtlp(attributes: Record<string, any>): OtlpAttribute[] {
  return Object.entries(attributes).map(([key, value]) => ({
    key,
    value: typeof value === 'string' ? { stringValue: value } :
           typeof value === 'number' ? { intValue: value.toString() } :
           typeof value === 'boolean' ? { boolValue: value } :
           { stringValue: String(value) }
  }))
}

function getStatusCode(statusCode: string): { code: string; message?: string } {
  // AgentPrism expects string status codes, not numeric
  switch (statusCode) {
    case 'STATUS_CODE_UNSET': return { code: 'STATUS_CODE_UNSET' }
    case 'STATUS_CODE_OK': return { code: 'STATUS_CODE_OK' }
    case 'STATUS_CODE_ERROR': return { code: 'STATUS_CODE_ERROR' }
    default: return { code: 'STATUS_CODE_UNSET' }
  }
}

export async function GET(
  request: NextRequest,
  { params }: { params: Promise<{ name: string; agentName: string; executionId: string }> }
) {
  try {
    const { name: clusterName, agentName, executionId } = await params
    
    // Extract trace ID from execution ID
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
        Duration as duration,
        StatusCode as statusCode,
        StatusMessage as statusMessage,
        SpanKind as spanKind,
        SpanAttributes as attributes,
        Events.Timestamp as eventTimestamps,
        Events.Name as eventNames,
        Events.Attributes as eventAttributes
      FROM langop.otel_traces
      WHERE TraceId LIKE {traceIdPattern:String}
        AND SpanName != 'agent.reconcile'
      ORDER BY Timestamp ASC
    `

    const resultSet = await clickhouse.query({
      query: sql,
      query_params: {
        traceIdPattern: `${traceIdPrefix}%`
      },
    })

    const rows = await resultSet.json()
    
    if (rows.data.length === 0) {
      return NextResponse.json({
        success: true,
        data: {
          resourceSpans: []
        }
      })
    }

    const spans: OtlpSpan[] = rows.data.map((row: any) => {
      // Convert events
      const events: OtlpEvent[] = []
      if (row.eventTimestamps && row.eventNames) {
        for (let i = 0; i < row.eventTimestamps.length; i++) {
          const eventTime = new Date(row.eventTimestamps[i])
          events.push({
            timeUnixNano: (eventTime.getTime() * 1000000).toString(),
            name: row.eventNames[i] || '',
            attributes: convertAttributesToOtlp(row.eventAttributes?.[i] || {})
          })
        }
      }

      const startTime = new Date(row.startTime)
      const endTime = new Date(row.endTime)

      return {
        traceId: row.traceId,
        spanId: row.spanId,
        parentSpanId: row.parentSpanId || undefined,
        name: row.spanName,
        kind: row.spanKind || 1, // SPAN_KIND_INTERNAL
        startTimeUnixNano: (startTime.getTime() * 1000000).toString(),
        endTimeUnixNano: (endTime.getTime() * 1000000).toString(),
        attributes: convertAttributesToOtlp(row.attributes || {}),
        events,
        status: getStatusCode(row.statusCode)
      }
    })

    // Create OTLP document structure
    const otlpDocument: OtlpDocument = {
      resourceSpans: [
        {
          resource: {
            attributes: [
              {
                key: 'service.name',
                value: { stringValue: agentName }
              },
              {
                key: 'service.namespace',
                value: { stringValue: clusterName }
              }
            ]
          },
          scopeSpans: [
            {
              scope: {
                name: 'language-operator',
                version: '1.0.0'
              },
              spans
            }
          ]
        }
      ]
    }

    return NextResponse.json({
      success: true,
      data: otlpDocument
    })

  } catch (error) {
    console.error('Error querying ClickHouse for OTLP trace data:', error)
    
    return NextResponse.json(
      { success: false, error: 'Failed to fetch OTLP trace data' },
      { status: 500 }
    )
  }
}