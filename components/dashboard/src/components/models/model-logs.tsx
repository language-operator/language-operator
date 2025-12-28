'use client'

import { useState, useEffect, useRef, useCallback } from 'react'
import { fetchWithOrganization } from '@/lib/api-client'
import { Card, CardContent, CardHeader } from '@/components/ui/card'
import { Button } from '@/components/ui/button'
import { AlertCircle, ChevronDown } from 'lucide-react'
import { LanguageModel } from '@/types/model'
import { convertAnsiToHtml } from '../agents/utils'

interface ModelLogsProps {
  model: LanguageModel
  clusterName: string
}

export function ModelLogs({ model, clusterName }: ModelLogsProps) {
  const [logs, setLogs] = useState<string[]>([])
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState<string | null>(null)
  const [isAtBottom, setIsAtBottom] = useState(true)
  const [userScrolledUp, setUserScrolledUp] = useState(false)
  const logsEndRef = useRef<HTMLDivElement>(null)
  const logsContainerRef = useRef<HTMLDivElement>(null)


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

  const fetchLogs = useCallback(async () => {
    try {
      setLoading(true)
      setError(null)

      const url = `/api/clusters/${clusterName}/models/${model.metadata.name}/logs`

      const response = await fetchWithOrganization(url)
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
      console.error('Error fetching model logs:', err)
      setError(err instanceof Error ? err.message : 'Failed to load logs')
    } finally {
      setLoading(false)
    }
  }, [model.metadata.name, clusterName])

  useEffect(() => {
    fetchLogs()
  }, [fetchLogs])

  const clearLogs = () => {
    setLogs([])
    setIsAtBottom(true)
    setUserScrolledUp(false)
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
              <h4 className="text-sm font-medium">LiteLLM Proxy Logs</h4>
              <p className="text-xs text-muted-foreground">
                Viewing logs for model &ldquo;{model.metadata.name}&rdquo;
              </p>
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
                onClick={fetchLogs}
              >
                Refresh
              </Button>
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
                {error ? 'Failed to load logs' : 'No logs available'}
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
    </div>
  )
}