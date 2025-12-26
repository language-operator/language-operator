'use client'

import { useState, useEffect } from 'react'
import { Card } from '@/components/ui/card'
import { Badge } from '@/components/ui/badge'
import { Activity, Clock, Info } from 'lucide-react'
// Local type definitions
interface AgentExecution {
  traceId: string
  executionId: string
  startTime: Date
  endTime: Date
  duration: number
  status: 'success' | 'error' | 'running'
  rootSpanName: string
  spanCount: number
}

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
import { fetchWithOrganization } from '@/lib/api-client'

interface FlameChartTimelineProps {
  execution: AgentExecution
  clusterName: string
  agentName: string
}

interface TraceData {
  traceId: string
  executionId: string
  spans: TraceSpan[]
}

export function FlameChartTimeline({ execution, clusterName, agentName }: FlameChartTimelineProps) {
  const [traceData, setTraceData] = useState<TraceData | null>(null)
  const [isLoading, setIsLoading] = useState(true)
  const [error, setError] = useState<string | null>(null)
  const [activeSpan, setActiveSpan] = useState<TraceSpan | null>(null)

  useEffect(() => {
    const fetchTraceData = async () => {
      try {
        setIsLoading(true)
        const response = await fetchWithOrganization(
          `/api/clusters/${clusterName}/agents/${agentName}/executions/${execution.executionId}/traces`
        )
        
        if (!response.ok) {
          throw new Error('Failed to fetch trace data')
        }
        
        const result = await response.json()
        setTraceData(result.data)
      } catch (err) {
        console.error('Error fetching trace data:', err)
        setError(err instanceof Error ? err.message : 'Unknown error')
      } finally {
        setIsLoading(false)
      }
    }

    fetchTraceData()
  }, [execution.executionId, clusterName, agentName])

  const formatDuration = (ms: number) => {
    if (ms < 1000) {
      return `${ms}ms`
    }
    const seconds = Math.floor(ms / 1000)
    if (seconds < 60) {
      return `${seconds}s`
    }
    const minutes = Math.floor(seconds / 60)
    const remainingSeconds = seconds % 60
    return `${minutes}m ${remainingSeconds}s`
  }

  const getSpanDepth = (span: TraceSpan, spans: TraceSpan[]): number => {
    if (!span.parentSpanId) return 0
    const parent = spans.find(s => s.spanId === span.parentSpanId)
    return parent ? getSpanDepth(parent, spans) + 1 : 0
  }

  const calculateSpanPosition = (span: TraceSpan, totalDuration: number) => {
    const startTime = new Date(span.startTime).getTime()
    const executionStartTime = new Date(execution.startTime).getTime()
    const relativeStart = startTime - executionStartTime
    const left = (relativeStart / totalDuration) * 100
    const width = (span.duration / totalDuration) * 100
    
    return { left: Math.max(0, left), width: Math.min(100 - Math.max(0, left), width) }
  }

  const getStatusColor = (status: string) => {
    switch (status) {
      case 'success':
        return 'bg-green-500'
      case 'error':
        return 'bg-red-500'
      default:
        return 'bg-blue-500'
    }
  }

  if (isLoading) {
    return (
      <div className="flex items-center justify-center py-12">
        <div className="text-center">
          <div className="animate-spin rounded-full h-8 w-8 border-b-2 border-foreground mx-auto mb-4"></div>
          <p className="text-muted-foreground">Loading trace timeline...</p>
        </div>
      </div>
    )
  }

  if (error) {
    return (
      <div className="flex items-center justify-center py-12">
        <div className="text-center">
          <Activity className="h-8 w-8 text-red-500 mx-auto mb-4" />
          <p className="text-foreground">Failed to load trace data</p>
          <p className="text-sm text-muted-foreground mt-2">{error}</p>
        </div>
      </div>
    )
  }

  if (!traceData || traceData.spans.length === 0) {
    return (
      <div className="flex items-center justify-center py-12">
        <div className="text-center">
          <Activity className="h-8 w-8 text-muted-foreground mx-auto mb-4" />
          <p className="text-foreground">No span data available</p>
        </div>
      </div>
    )
  }

  const totalDuration = execution.duration
  const maxDepth = Math.max(...traceData.spans.map(span => getSpanDepth(span, traceData.spans)))

  return (
    <div className="space-y-4">
      {/* Timeline Header */}
      <div className="flex justify-between items-center text-sm text-gray-500">
        <span>0s</span>
        <span className="flex items-center gap-2">
          <Clock className="h-4 w-4" />
          Total: {formatDuration(totalDuration)}
        </span>
      </div>

      {/* Timeline Container */}
      <div className="relative">
        {/* Time axis */}
        <div className="h-8 border-b border-gray-200 relative">
          {[0, 25, 50, 75, 100].map(percent => (
            <div
              key={percent}
              className="absolute top-0 h-full border-l border-gray-300"
              style={{ left: `${percent}%` }}
            >
              <div className="absolute top-full mt-1 text-xs text-gray-500 transform -translate-x-1/2">
                {formatDuration((percent / 100) * totalDuration)}
              </div>
            </div>
          ))}
        </div>

        {/* Spans */}
        <div 
          className="relative mt-8"
          style={{ height: (maxDepth + 1) * 48 + 16 }}
        >
          {traceData.spans.map((span) => {
            const depth = getSpanDepth(span, traceData.spans)
            const position = calculateSpanPosition(span, totalDuration)
            
            const isActive = activeSpan?.spanId === span.spanId
            
            return (
              <div
                key={span.spanId}
                className={`absolute h-8 rounded-md cursor-pointer transition-all duration-200 ${getStatusColor(span.status)} ${
                  isActive 
                    ? 'ring-2 ring-offset-2 ring-blue-500 opacity-100 shadow-lg' 
                    : 'hover:opacity-80'
                } flex items-center px-2`}
                style={{
                  top: depth * 48 + 8,
                  left: `${position.left}%`,
                  width: `${position.width}%`,
                  minWidth: '2px'
                }}
                onClick={() => setActiveSpan(activeSpan?.spanId === span.spanId ? null : span)}
              >
                <span 
                  className="text-white text-xs font-medium truncate"
                  style={{ fontSize: position.width > 10 ? '0.75rem' : '0' }}
                >
                  {span.spanName}
                </span>
              </div>
            )
          })}
        </div>
      </div>

      {/* Span Details Panel */}
      {activeSpan && (
        <Card className="p-4 bg-muted">
          <div className="space-y-3">
            <div className="flex items-center justify-between">
              <h4 className="font-medium text-foreground">{activeSpan.spanName}</h4>
              <Badge className={
                activeSpan.status === 'success' 
                  ? 'bg-green-100 text-green-800' 
                  : 'bg-red-100 text-red-800'
              }>
                {activeSpan.status}
              </Badge>
            </div>
            
            <div className="grid grid-cols-2 gap-4 text-sm">
              <div>
                <p className="text-muted-foreground">Duration</p>
                <p className="font-medium text-foreground">{formatDuration(activeSpan.duration)}</p>
              </div>
              <div>
                <p className="text-muted-foreground">Span ID</p>
                <p className="font-mono text-xs text-foreground">{activeSpan.spanId}</p>
              </div>
            </div>

            {Object.keys(activeSpan.attributes).length > 0 && (
              <div>
                <p className="text-muted-foreground text-sm mb-2">Attributes</p>
                <div className="space-y-1">
                  {Object.entries(activeSpan.attributes).slice(0, 3).map(([key, value]) => (
                    <div key={key} className="flex justify-between text-xs">
                      <span className="text-muted-foreground">{key}:</span>
                      <span className="font-mono text-foreground">{value}</span>
                    </div>
                  ))}
                </div>
              </div>
            )}

            {activeSpan.events.length > 0 && (
              <div>
                <p className="text-muted-foreground text-sm mb-2">Events</p>
                <div className="space-y-1">
                  {activeSpan.events.slice(0, 2).map((event, index) => (
                    <div key={index} className="text-xs">
                      <span className="text-foreground">{event.name}</span>
                    </div>
                  ))}
                </div>
              </div>
            )}
          </div>
        </Card>
      )}

      {/* Legend */}
      <div className="flex items-center gap-6 text-sm">
        <div className="flex items-center gap-2">
          <div className="w-4 h-4 bg-green-500 rounded"></div>
          <span>Success</span>
        </div>
        <div className="flex items-center gap-2">
          <div className="w-4 h-4 bg-red-500 rounded"></div>
          <span>Error</span>
        </div>
        <div className="flex items-center gap-2">
          <div className="w-4 h-4 bg-blue-500 rounded"></div>
          <span>Other</span>
        </div>
        <div className="flex items-center gap-2 text-muted-foreground">
          <Info className="h-4 w-4" />
          <span>Click spans for details</span>
        </div>
      </div>
    </div>
  )
}