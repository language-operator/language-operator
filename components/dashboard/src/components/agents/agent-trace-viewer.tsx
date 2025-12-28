'use client'

import { useState, useEffect, useMemo } from 'react'
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card'
import { Badge } from '@/components/ui/badge'
import { Button } from '@/components/ui/button'
import { Activity, Clock, AlertTriangle, ChevronDown, ChevronRight, Share, Link } from 'lucide-react'

// Enhanced span interface with computed properties
interface EnhancedSpan {
  id: string
  name: string
  parentId?: string
  startTime: number
  endTime: number
  duration: number
  status: 'ok' | 'error' | 'unset'
  attributes: Record<string, any>
  events: Array<{
    timestamp: number
    name: string
    attributes: Record<string, any>
  }>
  // Computed properties
  children: EnhancedSpan[]
  depth: number
  relativeStartTime: number
  criticalPath: boolean
  errorInSubtree: boolean
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

interface AgentTraceViewerProps {
  execution: AgentExecution
  clusterName: string
  agentName: string
}

// Utility functions
function buildSpanHierarchy(spans: any[]): EnhancedSpan[] {
  if (!spans.length) return []

  const spanMap = new Map<string, EnhancedSpan>()
  const rootSpans: EnhancedSpan[] = []
  
  // First pass: create enhanced spans
  const traceStartTime = Math.min(...spans.map(s => new Date(s.startTime).getTime()))
  
  for (const span of spans) {
    const startTime = new Date(span.startTime).getTime()
    const endTime = new Date(span.endTime).getTime()
    
    const enhancedSpan: EnhancedSpan = {
      id: span.spanId,
      name: span.spanName,
      parentId: span.parentSpanId,
      startTime,
      endTime,
      duration: endTime - startTime,
      status: span.statusCode === 'STATUS_CODE_ERROR' ? 'error' : 
              span.statusCode === 'STATUS_CODE_OK' ? 'ok' : 'unset',
      attributes: span.attributes || {},
      events: span.eventTimestamps?.map((timestamp: string, idx: number) => ({
        timestamp: new Date(timestamp).getTime(),
        name: span.eventNames?.[idx] || 'Unknown Event',
        attributes: span.eventAttributes?.[idx] || {}
      })) || [],
      children: [],
      depth: 0,
      relativeStartTime: startTime - traceStartTime,
      criticalPath: false,
      errorInSubtree: false
    }
    
    spanMap.set(span.spanId, enhancedSpan)
  }

  // Second pass: build hierarchy and compute properties
  for (const span of spanMap.values()) {
    if (span.parentId && spanMap.has(span.parentId)) {
      const parent = spanMap.get(span.parentId)!
      parent.children.push(span)
      span.depth = parent.depth + 1
    } else {
      rootSpans.push(span)
    }
  }

  // Third pass: compute error propagation and critical path
  function computeProperties(span: EnhancedSpan): boolean {
    let hasError = span.status === 'error'
    let maxChildDuration = 0
    let criticalChild: EnhancedSpan | null = null

    for (const child of span.children) {
      const childHasError = computeProperties(child)
      if (childHasError) hasError = true
      
      if (child.duration > maxChildDuration) {
        maxChildDuration = child.duration
        criticalChild = child
      }
    }

    span.errorInSubtree = hasError
    
    // Mark critical path (longest duration chain)
    if (criticalChild) {
      criticalChild.criticalPath = true
    }
    
    return hasError
  }

  rootSpans.forEach(computeProperties)
  
  // Sort by start time
  function sortSpans(spans: EnhancedSpan[]) {
    spans.sort((a, b) => a.startTime - b.startTime)
    spans.forEach(span => sortSpans(span.children))
  }
  sortSpans(rootSpans)

  return rootSpans
}

function formatDuration(ms: number): string {
  if (ms < 1) return `${(ms * 1000).toFixed(0)}μs`
  if (ms < 1000) return `${ms.toFixed(1)}ms`
  if (ms < 60000) return `${(ms / 1000).toFixed(2)}s`
  return `${(ms / 60000).toFixed(1)}m`
}

function getSpanColor(span: EnhancedSpan): string {
  if (span.status === 'error') return 'bg-destructive'
  if (span.criticalPath) return 'bg-orange-500 dark:bg-orange-600'
  if (span.errorInSubtree) return 'bg-yellow-500 dark:bg-yellow-600'
  return 'bg-primary'
}

function SpanTimeline({ 
  spans, 
  totalDuration, 
  onSpanSelect,
  selectedSpanId,
  expandedSpanIds,
  onToggleExpand
}: {
  spans: EnhancedSpan[]
  totalDuration: number
  onSpanSelect: (span: EnhancedSpan) => void
  selectedSpanId?: string
  expandedSpanIds: Set<string>
  onToggleExpand: (spanId: string) => void
}) {

  const renderSpan = (span: EnhancedSpan) => {
    const widthPercent = (span.duration / totalDuration) * 100
    const leftPercent = (span.relativeStartTime / totalDuration) * 100
    const hasChildren = span.children.length > 0
    const isExpanded = expandedSpanIds.has(span.id)
    const isSelected = selectedSpanId === span.id

    return (
      <div key={span.id} className="space-y-1">
        <div 
          className={`relative flex items-center p-2 rounded-lg cursor-pointer transition-all group
            ${isSelected ? 'bg-primary/10' : 'hover:bg-muted/50'}
          `}
          style={{ marginLeft: `${span.depth * 24}px` }}
          onClick={() => onSpanSelect(span)}
        >
          {/* Expand/collapse button */}
          {hasChildren && (
            <Button
              variant="ghost"
              size="sm"
              className="h-6 w-6 p-0 mr-2"
              onClick={(e) => {
                e.stopPropagation()
                onToggleExpand(span.id)
              }}
            >
              {isExpanded ? <ChevronDown className="h-4 w-4" /> : <ChevronRight className="h-4 w-4" />}
            </Button>
          )}
          {!hasChildren && <div className="w-8" />}

          {/* Span info */}
          <div className="flex-1 min-w-0">
            <div className="flex items-center justify-between">
              <div className="flex items-center space-x-2 min-w-0">
                <span className="font-medium text-sm truncate">{span.name}</span>
                {span.status === 'error' && (
                  <AlertTriangle className="h-4 w-4 text-red-500 flex-shrink-0" />
                )}
                {span.criticalPath && (
                  <Badge variant="outline" className="text-xs">Critical Path</Badge>
                )}
              </div>
              <div className="flex items-center space-x-2 text-xs text-muted-foreground flex-shrink-0">
                <Clock className="h-3 w-3" />
                <span>{formatDuration(span.duration)}</span>
              </div>
            </div>

            {/* Timeline bar */}
            <div className="mt-2 h-4 bg-muted rounded relative overflow-hidden">
              <div
                className={`h-full rounded transition-all ${getSpanColor(span)}`}
                style={{
                  width: `${Math.max(widthPercent, 0.5)}%`,
                  marginLeft: `${leftPercent}%`
                }}
              />
              {span.events.map((event, idx) => {
                const eventPercent = ((event.timestamp - span.startTime + span.relativeStartTime) / totalDuration) * 100
                return (
                  <div
                    key={idx}
                    className="absolute top-0 bottom-0 w-0.5 bg-purple-500 dark:bg-purple-400"
                    style={{ left: `${eventPercent}%` }}
                    title={event.name}
                  />
                )
              })}
            </div>
          </div>
        </div>

        {/* Children */}
        {hasChildren && isExpanded && (
          <div className="space-y-1">
            {span.children.map(renderSpan)}
          </div>
        )}
      </div>
    )
  }

  return (
    <div className="space-y-1">
      {spans.map(renderSpan)}
    </div>
  )
}

function SpanDetailsPanel({ span }: { span: EnhancedSpan }) {

  return (
    <div className="space-y-6 max-h-[600px] overflow-y-auto">
      {/* Basic Info */}
      <div>
        <h3 className="font-semibold text-lg mb-3">{span.name}</h3>
        <div className="grid grid-cols-2 gap-3 text-sm">
          <div>
            <span className="text-muted-foreground">Duration:</span>
            <span className="ml-2 font-mono">{formatDuration(span.duration)}</span>
          </div>
          <div>
            <span className="text-muted-foreground">Status:</span>
            <Badge 
              className="ml-2" 
              variant={span.status === 'error' ? 'destructive' : 'default'}
            >
              {span.status}
            </Badge>
          </div>
          <div className="col-span-2">
            <span className="text-muted-foreground">Span ID:</span>
            <span className="ml-2 font-mono text-xs">{span.id}</span>
          </div>
          {span.parentId && (
            <div className="col-span-2">
              <span className="text-muted-foreground">Parent ID:</span>
              <span className="ml-2 font-mono text-xs">{span.parentId}</span>
            </div>
          )}
        </div>
      </div>

      {/* Special Attributes - Input/Output */}
      {(span.attributes['gen_ai.prompt'] || span.attributes['gen_ai.completion'] || 
        span.attributes['gen_ai.system'] || span.attributes['tool.input'] || 
        span.attributes['tool.output']) && (
        <div>
          <h4 className="font-medium mb-3">Input/Output</h4>
          <div className="space-y-4">
            {span.attributes['gen_ai.system'] && (
              <div>
                <h5 className="text-sm font-medium text-muted-foreground mb-2">System Prompt</h5>
                <div className="text-sm p-3 bg-muted rounded font-mono whitespace-pre-wrap max-h-32 overflow-y-auto">
                  {String(span.attributes['gen_ai.system'])}
                </div>
              </div>
            )}
            {span.attributes['gen_ai.prompt'] && (
              <div>
                <h5 className="text-sm font-medium text-muted-foreground mb-2">User Prompt</h5>
                <div className="text-sm p-3 bg-muted rounded font-mono whitespace-pre-wrap max-h-32 overflow-y-auto">
                  {String(span.attributes['gen_ai.prompt'])}
                </div>
              </div>
            )}
            {span.attributes['tool.input'] && (
              <div>
                <h5 className="text-sm font-medium text-muted-foreground mb-2">Tool Input</h5>
                <div className="text-sm p-3 bg-muted rounded font-mono whitespace-pre-wrap max-h-32 overflow-y-auto">
                  {typeof span.attributes['tool.input'] === 'string' ? 
                    span.attributes['tool.input'] : 
                    JSON.stringify(span.attributes['tool.input'], null, 2)}
                </div>
              </div>
            )}
            {span.attributes['gen_ai.completion'] && (
              <div>
                <h5 className="text-sm font-medium text-muted-foreground mb-2">AI Response</h5>
                <div className="text-sm p-3 bg-muted rounded font-mono whitespace-pre-wrap max-h-32 overflow-y-auto">
                  {String(span.attributes['gen_ai.completion'])}
                </div>
              </div>
            )}
            {span.attributes['tool.output'] && (
              <div>
                <h5 className="text-sm font-medium text-muted-foreground mb-2">Tool Output</h5>
                <div className="text-sm p-3 bg-muted rounded font-mono whitespace-pre-wrap max-h-32 overflow-y-auto">
                  {typeof span.attributes['tool.output'] === 'string' ? 
                    span.attributes['tool.output'] : 
                    JSON.stringify(span.attributes['tool.output'], null, 2)}
                </div>
              </div>
            )}
          </div>
        </div>
      )}

      {/* All Attributes - Debug Version */}
      {true && (
        <div>
          <h4 className="font-medium mb-3">All Attributes ({Object.keys(span.attributes).length})</h4>
          <div className="space-y-3 max-h-64 overflow-y-auto">
            {Object.entries(span.attributes)
              .sort(([a], [b]) => a.localeCompare(b)) // Sort alphabetically
              .map(([key, value]) => {
                const isLongValue = String(value).length > 100
                const isJsonValue = typeof value === 'object' || (typeof value === 'string' && 
                  (value.startsWith('{') || value.startsWith('[')))
                
                return (
                  <div key={key} className="border-b border-muted pb-2">
                    <div className="flex items-start gap-2">
                      <span className="text-muted-foreground font-mono text-xs min-w-0 flex-shrink-0 font-medium">
                        {key}:
                      </span>
                      <div className="min-w-0 flex-1">
                        {isLongValue ? (
                          <div className="text-xs bg-muted rounded p-2 font-mono whitespace-pre-wrap max-h-24 overflow-y-auto">
                            {isJsonValue && typeof value === 'string' ? 
                              (() => {
                                try {
                                  return JSON.stringify(JSON.parse(value), null, 2)
                                } catch {
                                  return value
                                }
                              })() :
                              typeof value === 'object' ?
                                JSON.stringify(value, null, 2) :
                                String(value)
                            }
                          </div>
                        ) : (
                          <span className="text-xs font-mono break-all">
                            {typeof value === 'object' ? JSON.stringify(value) : String(value)}
                          </span>
                        )}
                      </div>
                    </div>
                  </div>
                )
              })}
          </div>
        </div>
      )}

      {/* Events */}
      {span.events.length > 0 && (
        <div>
          <h4 className="font-medium mb-3">Events ({span.events.length})</h4>
          <div className="space-y-3 max-h-48 overflow-y-auto">
            {span.events.map((event, idx) => (
              <div key={idx} className="border rounded p-3">
                <div className="flex justify-between items-center mb-2">
                  <span className="font-medium text-sm">{event.name}</span>
                  <span className="text-xs text-muted-foreground">
                    +{formatDuration(event.timestamp - span.startTime)}
                  </span>
                </div>
                {Object.keys(event.attributes).length > 0 && (
                  <div className="space-y-1">
                    {Object.entries(event.attributes).map(([key, value]) => (
                      <div key={key} className="flex text-xs">
                        <span className="text-muted-foreground">{key}:</span>
                        <span className="ml-1">{String(value)}</span>
                      </div>
                    ))}
                  </div>
                )}
              </div>
            ))}
          </div>
        </div>
      )}

      {/* Children Summary */}
      {span.children.length > 0 && (
        <div>
          <h4 className="font-medium mb-3">Child Spans ({span.children.length})</h4>
          <div className="space-y-2">
            {span.children.map(child => (
              <div key={child.id} className="flex justify-between items-center text-sm">
                <span className="truncate">{child.name}</span>
                <span className="text-muted-foreground">{formatDuration(child.duration)}</span>
              </div>
            ))}
          </div>
        </div>
      )}
    </div>
  )
}

export function AgentTraceViewer({ execution, clusterName, agentName }: AgentTraceViewerProps) {
  const [rawData, setRawData] = useState<any>(null)
  const [spans, setSpans] = useState<EnhancedSpan[]>([])
  const [selectedSpan, setSelectedSpan] = useState<EnhancedSpan | null>(null)
  const [expandedSpanIds, setExpandedSpanIds] = useState<Set<string>>(new Set())
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState<string | null>(null)

  useEffect(() => {
    async function fetchTraceData() {
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

        const otlpData = result.data
        setRawData(otlpData)

        // Extract spans from OTLP format
        const flatSpans = []
        for (const resourceSpan of otlpData.resourceSpans || []) {
          for (const scopeSpan of resourceSpan.scopeSpans || []) {
            for (const span of scopeSpan.spans || []) {
              flatSpans.push({
                spanId: span.spanId,
                parentSpanId: span.parentSpanId,
                spanName: span.name,
                startTime: new Date(parseInt(span.startTimeUnixNano) / 1000000),
                endTime: new Date(parseInt(span.endTimeUnixNano) / 1000000),
                statusCode: span.status?.code || 'STATUS_CODE_UNSET',
                attributes: span.attributes?.reduce((acc: any, attr: any) => {
                  acc[attr.key] = attr.value?.stringValue || attr.value?.intValue || attr.value?.boolValue
                  return acc
                }, {}) || {},
                eventTimestamps: span.events?.map((e: any) => new Date(parseInt(e.timeUnixNano) / 1000000)),
                eventNames: span.events?.map((e: any) => e.name),
                eventAttributes: span.events?.map((e: any) => 
                  e.attributes?.reduce((acc: any, attr: any) => {
                    acc[attr.key] = attr.value?.stringValue || attr.value?.intValue || attr.value?.boolValue
                    return acc
                  }, {})
                )
              })
            }
          }
        }

        const hierarchicalSpans = buildSpanHierarchy(flatSpans)
        setSpans(hierarchicalSpans)

        // Auto-expand all spans and don't select any span initially
        const allSpanIds = new Set<string>()
        function collectAllIds(spanList: EnhancedSpan[]) {
          for (const span of spanList) {
            allSpanIds.add(span.id)
            collectAllIds(span.children)
          }
        }
        collectAllIds(hierarchicalSpans)
        setExpandedSpanIds(allSpanIds)
        setSelectedSpan(null)

        setError(null)
      } catch (err) {
        setError(err instanceof Error ? err.message : 'Unknown error')
        setSpans([])
      } finally {
        setLoading(false)
      }
    }

    fetchTraceData()
  }, [execution.executionId, clusterName, agentName])

  const totalDuration = useMemo(() => {
    if (!spans.length) return 0
    const allSpans: EnhancedSpan[] = []
    function collectSpans(spanList: EnhancedSpan[]) {
      for (const span of spanList) {
        allSpans.push(span)
        collectSpans(span.children)
      }
    }
    collectSpans(spans)
    
    const minStart = Math.min(...allSpans.map(s => s.relativeStartTime))
    const maxEnd = Math.max(...allSpans.map(s => s.relativeStartTime + s.duration))
    return maxEnd - minStart
  }, [spans])

  const handleToggleExpand = (spanId: string) => {
    setExpandedSpanIds(prev => {
      const newSet = new Set(prev)
      if (newSet.has(spanId)) {
        newSet.delete(spanId)
      } else {
        newSet.add(spanId)
      }
      return newSet
    })
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
            <AlertTriangle className="h-12 w-12 text-red-600 mx-auto mb-4" />
            <p className="text-muted-foreground">Failed to load trace data</p>
            <p className="text-sm text-muted-foreground mt-2">{error}</p>
          </div>
        </CardContent>
      </Card>
    )
  }

  return (
    <div className="space-y-6">

      {/* Timeline */}
      <Card>
        <CardHeader>
          <div className="flex items-center justify-between">
            <CardTitle className="flex items-center gap-2">
              <Activity className="h-5 w-5" />
              Timeline ({(() => {
                const seconds = Math.floor(execution.duration / 1000)
                if (seconds < 60) return `${seconds}s`
                const minutes = Math.floor(seconds / 60)
                const remainingSeconds = seconds % 60
                return `${minutes}m ${remainingSeconds}s`
              })()})
            </CardTitle>
            <Button
              variant="outline"
              size="sm"
              className="gap-2"
              onClick={() => {
                const url = `/clusters/${clusterName}/agents/${agentName}/traces?traceId=${execution.traceId}`
                const fullUrl = window.location.origin + url
                navigator.clipboard.writeText(fullUrl)
                // Could add a toast notification here
              }}
            >
              <Link className="h-4 w-4" />
              Share
            </Button>
          </div>
        </CardHeader>
        <CardContent>
          
          <div className={`grid gap-6 min-h-[700px] transition-all ${selectedSpan ? 'grid-cols-1 lg:grid-cols-3' : 'grid-cols-1'}`}>
            {/* Timeline */}
            <div className={`space-y-4 ${selectedSpan ? 'lg:col-span-2' : ''}`}>
              <div className="flex items-center justify-between">
                <h3 className="text-sm font-medium text-muted-foreground">
                  Spans ({(() => {
                    let totalSpans = 0
                    function countSpans(spanList: EnhancedSpan[]) {
                      totalSpans += spanList.length
                      spanList.forEach(span => countSpans(span.children))
                    }
                    countSpans(spans)
                    return totalSpans
                  })()})
                </h3>
                <div className="flex items-center gap-1 text-xs text-muted-foreground">
                  <button
                    className="hover:text-foreground transition-colors cursor-pointer"
                    onClick={() => {
                      const allSpanIds = new Set<string>()
                      function collectIds(spanList: EnhancedSpan[]) {
                        for (const span of spanList) {
                          allSpanIds.add(span.id)
                          collectIds(span.children)
                        }
                      }
                      collectIds(spans)
                      setExpandedSpanIds(allSpanIds)
                    }}
                  >
                    Expand All
                  </button>
                  <span>|</span>
                  <button
                    className="hover:text-foreground transition-colors cursor-pointer"
                    onClick={() => setExpandedSpanIds(new Set())}
                  >
                    Collapse All
                  </button>
                </div>
              </div>
              <div className="border rounded-lg p-4 bg-background max-h-[700px] overflow-y-auto">
                {/* Time Ruler */}
                <div className="mb-4 pb-2 border-b border-muted">
                  <div className="flex justify-between text-xs text-muted-foreground mb-1">
                    <span>0ms</span>
                    <span>{formatDuration(totalDuration / 4)}</span>
                    <span>{formatDuration(totalDuration / 2)}</span>
                    <span>{formatDuration((totalDuration * 3) / 4)}</span>
                    <span>{formatDuration(totalDuration)}</span>
                  </div>
                  <div className="h-2 bg-muted rounded relative">
                    <div className="absolute top-0 left-0 w-0.5 h-full bg-border"></div>
                    <div className="absolute top-0 left-1/4 w-0.5 h-full bg-border"></div>
                    <div className="absolute top-0 left-1/2 w-0.5 h-full bg-border"></div>
                    <div className="absolute top-0 left-3/4 w-0.5 h-full bg-border"></div>
                    <div className="absolute top-0 right-0 w-0.5 h-full bg-border"></div>
                  </div>
                </div>
                
                <SpanTimeline
                  spans={spans}
                  totalDuration={totalDuration}
                  onSpanSelect={setSelectedSpan}
                  selectedSpanId={selectedSpan?.id}
                  expandedSpanIds={expandedSpanIds}
                  onToggleExpand={handleToggleExpand}
                />
              </div>
            </div>

            {/* Details Panel - only show when span is selected */}
            {selectedSpan && (
              <div className="space-y-4">
                <div className="flex items-center justify-between">
                  <h3 className="text-sm font-medium text-muted-foreground">
                    Span Details
                  </h3>
                  <button
                    onClick={() => setSelectedSpan(null)}
                    className="text-xs text-muted-foreground hover:text-foreground transition-colors cursor-pointer"
                  >
                    ✕
                  </button>
                </div>
                <div className="border rounded-lg p-4 bg-background min-h-[600px]">
                  <SpanDetailsPanel span={selectedSpan} />
                </div>
              </div>
            )}
          </div>
        </CardContent>
      </Card>
    </div>
  )
}