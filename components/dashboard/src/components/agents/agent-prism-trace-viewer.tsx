'use client'

import { useState, useEffect } from 'react'
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card'
import { Badge } from '@/components/ui/badge'
import { Button } from '@/components/ui/button'
import { Activity, AlertCircle, ChevronDown, ChevronRight, Clock, Database } from 'lucide-react'
// TODO: Fix AgentPrism package imports
// import { openTelemetrySpanAdapter } from '@evilmartians/agent-prism-data'
// import type { TraceRecord, TraceSpan } from '@evilmartians/agent-prism-types'

// Temporary local types until AgentPrism is working
interface TraceRecord {
  id: string
  name: string
  spansCount: number
  durationMs: number
  agentDescription: string
}

interface TraceSpan {
  id: string
  title: string
  type?: string
  duration: number
  status?: string
  children?: TraceSpan[]
  attributes?: Array<{
    key: string
    value: {
      stringValue?: string
      intValue?: number
      boolValue?: boolean
    }
  }>
  input?: string
  output?: string
}

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

// Temporary OTLP to TraceSpan conversion until AgentPrism is working
function convertOtlpToSpans(otlpData: any): TraceSpan[] {
  if (!otlpData || !otlpData.resourceSpans) {
    return []
  }

  const spans: TraceSpan[] = []
  
  for (const resourceSpan of otlpData.resourceSpans) {
    for (const scopeSpan of resourceSpan.scopeSpans || []) {
      for (const span of scopeSpan.spans || []) {
        const convertedSpan: TraceSpan = {
          id: span.spanId || Math.random().toString(36),
          title: span.name || 'Unknown Span',
          type: span.kind ? `Kind ${span.kind}` : undefined,
          duration: span.endTimeUnixNano && span.startTimeUnixNano 
            ? Math.round((parseInt(span.endTimeUnixNano) - parseInt(span.startTimeUnixNano)) / 1000000)
            : 0,
          status: span.status?.code === 2 ? 'error' : 'success',
          attributes: (span.attributes || []).map((attr: any) => ({
            key: attr.key,
            value: attr.value
          })),
          children: [] // TODO: Build span hierarchy
        }
        spans.push(convertedSpan)
      }
    }
  }
  
  return spans
}

interface AgentPrismTraceViewerProps {
  execution: AgentExecution
  clusterName: string
  agentName: string
}

// Simple SpanCard component
function SpanCard({ 
  span, 
  isSelected, 
  isExpanded,
  onSelect, 
  onToggleExpand,
  level = 0 
}: { 
  span: TraceSpan
  isSelected: boolean
  isExpanded: boolean
  onSelect: () => void
  onToggleExpand: () => void
  level?: number
}) {
  const hasChildren = span.children && span.children.length > 0
  const durationMs = span.duration
  
  return (
    <div className="space-y-1">
      <div 
        className={`flex items-center gap-2 p-2 rounded cursor-pointer border transition-colors
          ${isSelected ? 'border-blue-500 bg-blue-50' : 'border-gray-200 hover:bg-gray-50'}
        `}
        style={{ marginLeft: `${level * 20}px` }}
        onClick={onSelect}
      >
        {hasChildren && (
          <Button
            variant="ghost" 
            size="sm"
            className="h-4 w-4 p-0"
            onClick={(e) => {
              e.stopPropagation()
              onToggleExpand()
            }}
          >
            {isExpanded ? <ChevronDown className="h-3 w-3" /> : <ChevronRight className="h-3 w-3" />}
          </Button>
        )}
        {!hasChildren && <div className="w-4" />}
        
        <div className="flex-1 flex items-center justify-between">
          <div className="flex items-center gap-2">
            <span className="font-medium text-sm">{span.title}</span>
            {span.type && (
              <Badge variant="outline" className="text-xs">
                {span.type}
              </Badge>
            )}
          </div>
          
          <div className="flex items-center gap-2 text-xs text-gray-500">
            <Clock className="h-3 w-3" />
            <span>{durationMs}ms</span>
            {span.status && (
              <Badge 
                variant={span.status === 'success' ? 'default' : 'destructive'}
                className="text-xs"
              >
                {span.status}
              </Badge>
            )}
          </div>
        </div>
      </div>
      
      {hasChildren && isExpanded && span.children && (
        <div>
          {span.children.map((child) => (
            <SpanCard
              key={child.id}
              span={child}
              isSelected={false}
              isExpanded={false}
              onSelect={() => {}}
              onToggleExpand={() => {}}
              level={level + 1}
            />
          ))}
        </div>
      )}
    </div>
  )
}

// Simple span details component
function SpanDetails({ span }: { span: TraceSpan }) {
  return (
    <div className="space-y-4">
      <div>
        <h4 className="font-medium mb-2">Span Information</h4>
        <div className="space-y-2 text-sm">
          <div className="flex justify-between">
            <span className="text-gray-500">Name:</span>
            <span className="font-mono">{span.title}</span>
          </div>
          <div className="flex justify-between">
            <span className="text-gray-500">ID:</span>
            <span className="font-mono text-xs">{span.id}</span>
          </div>
          <div className="flex justify-between">
            <span className="text-gray-500">Type:</span>
            <span className="font-mono">{span.type}</span>
          </div>
          <div className="flex justify-between">
            <span className="text-gray-500">Duration:</span>
            <span>{span.duration}ms</span>
          </div>
          <div className="flex justify-between">
            <span className="text-gray-500">Status:</span>
            <Badge variant={span.status === 'success' ? 'default' : 'destructive'}>
              {span.status || 'unknown'}
            </Badge>
          </div>
        </div>
      </div>

      {span.attributes && span.attributes.length > 0 && (
        <div>
          <h4 className="font-medium mb-2">Attributes</h4>
          <div className="space-y-1 text-sm max-h-60 overflow-y-auto">
            {span.attributes.map((attr, idx) => (
              <div key={idx} className="flex justify-between gap-2">
                <span className="text-gray-500 font-mono text-xs min-w-0 flex-1">{attr.key}:</span>
                <span className="font-mono text-xs break-all">
                  {attr.value.stringValue || attr.value.intValue || attr.value.boolValue?.toString()}
                </span>
              </div>
            ))}
          </div>
        </div>
      )}

      {span.input && (
        <div>
          <h4 className="font-medium mb-2">Input</h4>
          <div className="text-sm max-h-40 overflow-y-auto bg-gray-50 p-2 rounded font-mono">
            {span.input}
          </div>
        </div>
      )}

      {span.output && (
        <div>
          <h4 className="font-medium mb-2">Output</h4>
          <div className="text-sm max-h-40 overflow-y-auto bg-gray-50 p-2 rounded font-mono">
            {span.output}
          </div>
        </div>
      )}
    </div>
  )
}

export function AgentPrismTraceViewer({ 
  execution, 
  clusterName, 
  agentName 
}: AgentPrismTraceViewerProps) {
  const [traceData, setTraceData] = useState<any>(null)
  const [spans, setSpans] = useState<TraceSpan[]>([])
  const [selectedSpan, setSelectedSpan] = useState<TraceSpan | undefined>(undefined)
  const [expandedSpansIds, setExpandedSpansIds] = useState<string[]>([])
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState<string | null>(null)

  useEffect(() => {
    async function fetchOtlpData() {
      try {
        setLoading(true)
        const response = await fetch(
          `/api/clusters/${clusterName}/agents/${agentName}/executions/${execution.executionId}/otlp`
        )
        
        if (!response.ok) {
          throw new Error('Failed to fetch trace data')
        }

        const result = await response.json()
        if (!result.success) {
          throw new Error(result.error || 'Failed to fetch trace data')
        }

        const otlpDocument = result.data
        setTraceData(otlpDocument)

        // TODO: Replace with proper AgentPrism conversion
        // const convertedSpans = openTelemetrySpanAdapter.convertRawDocumentsToSpans(otlpDocument)
        
        // Temporary simple conversion for OTLP data
        const convertedSpans = convertOtlpToSpans(otlpDocument)
        setSpans(convertedSpans)
        
        setError(null)
      } catch (err) {
        setError(err instanceof Error ? err.message : 'Unknown error occurred')
        setSpans([])
        setTraceData(null)
      } finally {
        setLoading(false)
      }
    }

    fetchOtlpData()
  }, [execution.executionId, clusterName, agentName])

  const traceRecord: TraceRecord = {
    id: execution.traceId,
    name: `${execution.rootSpanName} (${execution.executionId})`,
    spansCount: execution.spanCount,
    durationMs: execution.duration,
    agentDescription: `Agent: ${agentName}`
  }

  if (loading) {
    return (
      <Card>
        <CardContent className="flex items-center justify-center py-16">
          <div className="text-center">
            <div className="animate-spin rounded-full h-12 w-12 border-b-2 border-foreground mx-auto mb-4"></div>
            <p className="text-muted-foreground">Loading trace visualization...</p>
          </div>
        </CardContent>
      </Card>
    )
  }

  if (error) {
    return (
      <Card>
        <CardContent className="flex items-center justify-center py-16">
          <div className="text-center">
            <AlertCircle className="h-12 w-12 text-red-600 mx-auto mb-4" />
            <p className="text-muted-foreground">Failed to load trace data</p>
            <p className="text-sm text-muted-foreground mt-2">{error}</p>
          </div>
        </CardContent>
      </Card>
    )
  }

  return (
    <div className="space-y-6">
      {/* Execution Overview */}
      <Card>
        <CardHeader>
          <CardTitle className="flex items-center justify-between">
            <div className="flex items-center gap-2">
              <Activity className="h-5 w-5" />
              Trace: {execution.executionId}
            </div>
            <Badge variant="outline">
              {spans.length} spans
            </Badge>
          </CardTitle>
        </CardHeader>
        <CardContent>
          <div className="grid grid-cols-2 md:grid-cols-4 gap-4 text-sm">
            <div>
              <p className="text-muted-foreground">Duration</p>
              <p className="font-medium">{execution.duration}ms</p>
            </div>
            <div>
              <p className="text-muted-foreground">Root Span</p>
              <p className="font-medium">{execution.rootSpanName}</p>
            </div>
            <div>
              <p className="text-muted-foreground">Start Time</p>
              <p className="font-medium">{new Date(execution.startTime).toLocaleTimeString()}</p>
            </div>
            <div>
              <p className="text-muted-foreground">Status</p>
              <Badge className={
                execution.status === 'success' ? 'bg-green-100 text-green-800' :
                execution.status === 'error' ? 'bg-red-100 text-red-800' :
                'bg-yellow-100 text-yellow-800'
              }>
                {execution.status}
              </Badge>
            </div>
          </div>
        </CardContent>
      </Card>

      {/* AgentPrism Trace Visualization */}
      <Card>
        <CardHeader>
          <CardTitle className="flex items-center gap-2">
            <Activity className="h-5 w-5" />
            Agent Execution Trace
          </CardTitle>
        </CardHeader>
        <CardContent>
          <div className="grid grid-cols-1 lg:grid-cols-3 gap-6 min-h-[600px]">
            {/* Trace Tree View */}
            <div className="lg:col-span-2 space-y-4">
              <h3 className="text-sm font-medium text-muted-foreground">Execution Timeline</h3>
              <div className="border rounded-lg p-4 bg-background max-h-[600px] overflow-y-auto">
                {spans.map((span) => (
                  <SpanCard
                    key={span.id}
                    span={span}
                    isSelected={selectedSpan?.id === span.id}
                    isExpanded={expandedSpansIds.includes(span.id)}
                    onSelect={() => setSelectedSpan(span)}
                    onToggleExpand={() => {
                      setExpandedSpansIds(prev => 
                        prev.includes(span.id)
                          ? prev.filter(id => id !== span.id)
                          : [...prev, span.id]
                      )
                    }}
                  />
                ))}
              </div>
            </div>

            {/* Span Details Panel */}
            <div className="space-y-4">
              <h3 className="text-sm font-medium text-muted-foreground">
                {selectedSpan ? 'Span Details' : 'Select a span'}
              </h3>
              <div className="border rounded-lg p-4 bg-background min-h-[400px] max-h-[600px] overflow-y-auto">
                {selectedSpan ? (
                  <SpanDetails span={selectedSpan} />
                ) : (
                  <div className="flex items-center justify-center h-full text-muted-foreground">
                    Click on a span in the timeline to view details
                  </div>
                )}
              </div>
            </div>
          </div>
        </CardContent>
      </Card>
    </div>
  )
}