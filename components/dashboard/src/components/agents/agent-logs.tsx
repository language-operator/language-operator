'use client'

import { useState, useEffect, useRef, useMemo } from 'react'
import { fetchWithOrganization } from '@/lib/api-client'
import { Card, CardContent, CardHeader } from '@/components/ui/card'
import { Badge } from '@/components/ui/badge'
import { Button } from '@/components/ui/button'
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from '@/components/ui/select'
import { AlertCircle, ChevronDown } from 'lucide-react'
import { LanguageAgent } from '@/types/agent'
import { convertAnsiToHtml } from './utils'

interface AgentLogsProps {
  agent: LanguageAgent
  clusterName: string
}

export function AgentLogs({ agent, clusterName }: AgentLogsProps) {
  const [logs, setLogs] = useState<string[]>([])
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState<string | null>(null)
  const [isStreaming, setIsStreaming] = useState(false)
  const [pods, setPods] = useState<any[]>([])
  const [selectedPod, setSelectedPod] = useState<string>('')
  const [podsLoading, setPodsLoading] = useState(false)
  const [isAtBottom, setIsAtBottom] = useState(true)
  const [userScrolledUp, setUserScrolledUp] = useState(false)
  const logsEndRef = useRef<HTMLDivElement>(null)
  const logsContainerRef = useRef<HTMLDivElement>(null)
  const eventSourceRef = useRef<EventSource | null>(null)

  // Memoize ANSI converter for performance
  const convertLogsToHtml = useMemo(() => {
    return (logLines: string[]): string[] => {
      return logLines.map(line => convertAnsiToHtml(line))
    }
  }, [])

  const scrollToBottom = () => {
    logsEndRef.current?.scrollIntoView({ behavior: 'smooth' })
    setIsAtBottom(true)
    setUserScrolledUp(false)
  }

  const checkScrollPosition = () => {
    if (logsContainerRef.current) {
      const { scrollTop, scrollHeight, clientHeight } = logsContainerRef.current
      const isNearBottom = scrollTop + clientHeight >= scrollHeight - 10
      setIsAtBottom(isNearBottom)
    }
  }

  const handleScroll = () => {
    checkScrollPosition()
    if (logsContainerRef.current) {
      const { scrollTop, scrollHeight, clientHeight } = logsContainerRef.current
      const isNearBottom = scrollTop + clientHeight >= scrollHeight - 10
      if (!isNearBottom) {
        setUserScrolledUp(true)
      }
    }
  }

  // Only auto-scroll if user is at bottom and hasn't scrolled up
  useEffect(() => {
    if (isAtBottom && !userScrolledUp) {
      scrollToBottom()
    }
  }, [logs, isAtBottom, userScrolledUp])

  useEffect(() => {
    fetchPods()
    return () => {
      // Cleanup on unmount
      if (eventSourceRef.current) {
        eventSourceRef.current.close()
      }
    }
  }, [agent.metadata.name, clusterName])

  useEffect(() => {
    // Fetch logs when selected pod changes
    if (selectedPod) {
      // Reset scroll state when switching pods
      setIsAtBottom(true)
      setUserScrolledUp(false)
      fetchInitialLogs()
    }
  }, [selectedPod])

  const fetchPods = async () => {
    try {
      setPodsLoading(true)
      setError(null)

      const response = await fetchWithOrganization(`/api/clusters/${clusterName}/agents/${agent.metadata.name}/pods`)
      if (!response.ok) {
        throw new Error(`Failed to fetch pods: ${response.status} ${response.statusText}`)
      }

      const data = await response.json()
      setPods(data.data || [])

      // Auto-select the recommended pod
      if (data.recommendedPod && data.data.length > 0) {
        setSelectedPod(data.recommendedPod)
      }
    } catch (err) {
      console.error('Error fetching pods:', err)
      setError(err instanceof Error ? err.message : 'Failed to load pods')
    } finally {
      setPodsLoading(false)
    }
  }

  const fetchInitialLogs = async () => {
    try {
      setLoading(true)
      setError(null)

      const url = selectedPod
        ? `/api/clusters/${clusterName}/agents/${agent.metadata.name}/logs?podName=${selectedPod}`
        : `/api/clusters/${clusterName}/agents/${agent.metadata.name}/logs`

      const response = await fetch(url)
      if (!response.ok) {
        throw new Error(`Failed to fetch logs: ${response.status} ${response.statusText}`)
      }

      const data = await response.json()
      const logLines = data.logs ? data.logs.split('\n').filter((line: string) => line.trim()) : []
      setLogs(logLines)

      // Check scroll position after logs are loaded
      setTimeout(() => {
        checkScrollPosition()
      }, 100)
    } catch (err) {
      console.error('Error fetching initial logs:', err)
      setError(err instanceof Error ? err.message : 'Failed to load logs')
    } finally {
      setLoading(false)
    }
  }

  const startStreaming = () => {
    if (eventSourceRef.current) {
      eventSourceRef.current.close()
    }

    setIsStreaming(true)
    setError(null)

    const url = selectedPod
      ? `/api/clusters/${clusterName}/agents/${agent.metadata.name}/logs/stream?podName=${selectedPod}`
      : `/api/clusters/${clusterName}/agents/${agent.metadata.name}/logs/stream`

    const eventSource = new EventSource(url)
    eventSourceRef.current = eventSource

    eventSource.onmessage = (event) => {
      const newLog = event.data
      if (newLog && newLog.trim()) {
        setLogs(prev => [...prev, newLog])
      }
    }

    eventSource.onerror = (error) => {
      console.error('EventSource error:', error)
      setError('Connection lost. Click "Start Streaming" to reconnect.')
      setIsStreaming(false)
      eventSource.close()
    }
  }

  const stopStreaming = () => {
    if (eventSourceRef.current) {
      eventSourceRef.current.close()
      eventSourceRef.current = null
    }
    setIsStreaming(false)
  }

  const clearLogs = () => {
    setLogs([])
    setIsAtBottom(true)
    setUserScrolledUp(false)
  }

  // Helper functions for pod display
  const formatTimeAgoCondensed = (timestamp: string) => {
    const date = new Date(timestamp)
    const now = new Date()
    const diff = now.getTime() - date.getTime()
    const minutes = Math.floor(diff / 60000)
    const hours = Math.floor(diff / 3600000)
    const days = Math.floor(diff / 86400000)

    if (days > 0) return `${days}d ago`
    if (hours > 0) return `${hours}h ago`
    if (minutes > 0) return `${minutes}m ago`
    return 'Just now'
  }

  const getStatusColor = (status: string) => {
    switch (status) {
      case 'Running': return 'text-green-600'
      case 'Succeeded': return 'text-blue-600'
      case 'Failed': return 'text-red-600'
      case 'Pending': return 'text-yellow-600'
      default: return 'text-gray-600'
    }
  }

  if (loading) {
    return (
      <div className="space-y-6">
        <Card>
          <CardContent className="flex items-center justify-center py-16">
            <div className="text-center">
              <div className="animate-spin rounded-full h-12 w-12 border-b-2 border-gray-900 mx-auto mb-4"></div>
              <p className="text-gray-600">Loading logs...</p>
            </div>
          </CardContent>
        </Card>
      </div>
    )
  }

  return (
    <div className="space-y-6">
      {/* Log Controls */}
      <Card className="flex-shrink-0">
        <CardHeader>
          <div className="flex items-center justify-between">
            <div className="flex-1">
              {/* Pod Selection */}
              <div>
                <Select
                  value={selectedPod}
                  onValueChange={(value) => {
                    setSelectedPod(value)
                    // Stop streaming when switching pods
                    if (isStreaming) {
                      stopStreaming()
                    }
                    // Clear existing logs
                    setLogs([])
                  }}
                  disabled={podsLoading || pods.length === 0}
                >
                  <SelectTrigger className="min-w-96">
                    <SelectValue placeholder={podsLoading ? "Loading pods..." : "Select a pod"} />
                  </SelectTrigger>
                  <SelectContent className="min-w-96">
                    {pods.map((pod) => (
                      <SelectItem key={pod.name} value={pod.name}>
                        <div className="flex items-center justify-between w-full">
                          <span className="font-mono text-sm">{pod.name}</span>
                          <div className="flex items-center gap-2 ml-4">
                            <Badge variant="outline" className={`${getStatusColor(pod.status)} text-xs`}>
                              {pod.status}
                            </Badge>
                            <span className="text-xs text-muted-foreground">
                              {formatTimeAgoCondensed(pod.creationTimestamp)}
                            </span>
                          </div>
                        </div>
                      </SelectItem>
                    ))}
                  </SelectContent>
                </Select>

                {pods.length === 0 && !podsLoading && (
                  <span className="text-sm text-muted-foreground">No pods found</span>
                )}
              </div>
            </div>
            <div className="flex items-center gap-2">
              {!isAtBottom && (
                <Button
                  variant="outline"
                  size="sm"
                  onClick={scrollToBottom}
                  className="animate-pulse"
                >
                  <ChevronDown className="h-4 w-4 mr-1" />
                  Scroll to Bottom
                </Button>
              )}
              <Button
                variant="outline"
                size="sm"
                onClick={clearLogs}
                disabled={logs.length === 0}
              >
                Clear
              </Button>
              <Button
                variant="outline"
                size="sm"
                onClick={fetchInitialLogs}
              >
                Refresh
              </Button>
              {isStreaming ? (
                <Button
                  variant="destructive"
                  size="sm"
                  onClick={stopStreaming}
                >
                  Stop Streaming
                </Button>
              ) : (
                <Button
                  variant="default"
                  size="sm"
                  onClick={startStreaming}
                >
                  Start Streaming
                </Button>
              )}
            </div>
          </div>
        </CardHeader>
      </Card>

      {/* Error Display */}
      {error && (
        <Card className="flex-shrink-0">
          <CardContent className="p-4">
            <div className="flex items-center gap-2 text-red-600">
              <AlertCircle className="h-4 w-4" />
              <span className="text-sm">{error}</span>
            </div>
          </CardContent>
        </Card>
      )}

      {/* Log Output */}
      <Card>
        <CardContent className="p-0">
          <div
            ref={logsContainerRef}
            className="bg-black text-white font-mono text-sm max-h-[60vh] overflow-y-auto p-4"
            onScroll={handleScroll}
          >
            {logs.length === 0 ? (
              <div className="flex items-center justify-center h-32 text-gray-500">
                No logs available
              </div>
            ) : (
              <div>
                {logs.map((log, index) => (
                  <div
                    key={index}
                    className="whitespace-pre-wrap break-words"
                    dangerouslySetInnerHTML={{
                      __html: convertAnsiToHtml(log)
                    }}
                  />
                ))}
                <div ref={logsEndRef} />
              </div>
            )}
          </div>
        </CardContent>
      </Card>

      {/* Streaming Status */}
      {isStreaming && (
        <Card>
          <CardContent className="p-4">
            <div className="flex items-center gap-2 text-green-600">
              <div className="animate-pulse w-2 h-2 bg-green-500 rounded-full" />
              <span className="text-sm">Streaming logs in real-time</span>
            </div>
          </CardContent>
        </Card>
      )}
    </div>
  )
}
