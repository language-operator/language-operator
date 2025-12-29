import { NextRequest, NextResponse } from 'next/server'
import { createClickHouseClient } from '@/lib/clickhouse-config'

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
const clickhouse = createClickHouseClient()

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

    // For kubectl-proxy, use direct HTTP request instead of the ClickHouse client
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
              // Clean up common JSON formatting issues
              let cleanStr = str.trim()
              
              // Handle ClickHouse NULL values
              if (cleanStr === '\\N' || cleanStr === 'NULL') return defaultValue
              
              // Try to parse as-is first
              try {
                return JSON.parse(cleanStr)
              } catch (firstError) {
                // If that fails, try to fix common issues
                // Remove trailing commas, fix single quotes, etc.
                cleanStr = cleanStr
                  .replace(/,(\s*[}\]])/g, '$1')  // Remove trailing commas
                  .replace(/'/g, '"')             // Replace single quotes with double quotes
                  .replace(/([{,]\s*)(\w+):/g, '$1"$2":')  // Quote unquoted keys
                
                return JSON.parse(cleanStr)
              }
            } catch (e) {
              // Only log warnings in development
              if (process.env.NODE_ENV === 'development') {
                console.warn('Failed to parse JSON field after cleanup:', str.substring(0, 200), e)
              }
              return defaultValue
            }
          }
          
          const attributes = safeJsonParse(fields[10], {})
          
          return {
            traceId: fields[0] || '',
            spanId: fields[1] || '',
            parentSpanId: fields[2] || null,
            spanName: fields[3] || '',
            startTime: fields[4] || '',
            endTime: fields[5] || '',
            duration: fields[6] ? parseInt(fields[6]) : 0,
            statusCode: fields[7] || 'STATUS_CODE_UNSET',
            statusMessage: fields[8] || '',
            spanKind: fields[9] ? parseInt(fields[9]) : 1,
            attributes: attributes,
            eventTimestamps: safeJsonParse(fields[11], []),
            eventNames: safeJsonParse(fields[12], []),
            eventAttributes: safeJsonParse(fields[13], [])
          }
        })
        rows = { data }
      }
    } else {
      const resultSet = await clickhouse.query({
        query: sql,
        query_params: {
          traceIdPattern: `${traceIdPrefix}%`
        },
      })
      rows = await resultSet.json()
    }
    
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