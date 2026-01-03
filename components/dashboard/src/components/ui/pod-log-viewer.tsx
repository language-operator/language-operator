'use client'

import { Card, CardContent, CardHeader } from '@/components/ui/card'
import { Button } from '@/components/ui/button'
import { AlertCircle, ChevronDown } from 'lucide-react'
import { convertAnsiToHtml } from '../agents/utils'
import type { UseLogViewerReturn } from '@/hooks/useLogViewer'

interface PodLogViewerProps {
  logs: UseLogViewerReturn
  onRefresh?: () => void
  refreshDisabled?: boolean
  className?: string
  children?: React.ReactNode
  // Streaming controls
  isStreaming?: boolean
  onStartStreaming?: () => void
  onStopStreaming?: () => void
  streamingDisabled?: boolean
}

export function PodLogViewer({ 
  logs, 
  onRefresh, 
  refreshDisabled = false,
  className = "",
  children,
  isStreaming = false,
  onStartStreaming,
  onStopStreaming,
  streamingDisabled = false
}: PodLogViewerProps) {
  return (
    <div className={`space-y-6 ${className}`}>
      {/* Log Controls */}
      <Card className="flex-shrink-0">
        <CardContent className="px-4">
          <div className="flex items-center gap-2">
            {!logs.isAtBottom && (
              <Button
                variant="outline"
                size="sm"
                onClick={logs.scrollToBottom}
                className="animate-pulse"
              >
                <ChevronDown className="h-4 w-4 mr-1" />
                Scroll to Bottom
              </Button>
            )}
            <Button
              variant="outline"
              size="sm"
              onClick={logs.clearLogs}
              disabled={logs.logs.length === 0}
            >
              Clear
            </Button>
            {onRefresh && (
              <Button
                variant="outline"
                size="sm"
                onClick={onRefresh}
                disabled={refreshDisabled}
              >
                Refresh
              </Button>
            )}
            {/* Streaming Controls */}
            {(onStartStreaming || onStopStreaming) && (
              <>
                {isStreaming ? (
                  <Button
                    variant="destructive"
                    size="sm"
                    onClick={onStopStreaming}
                    disabled={!onStopStreaming}
                  >
                    Stop Streaming
                  </Button>
                ) : (
                  <Button
                    variant="default"
                    size="sm"
                    onClick={onStartStreaming}
                    disabled={streamingDisabled || !onStartStreaming}
                  >
                    Start Streaming
                  </Button>
                )}
              </>
            )}
            {children}
          </div>
        </CardContent>
      </Card>

      {/* Error Display */}
      {logs.error && (
        <Card className="flex-shrink-0">
          <CardContent className="p-4">
            <div className="flex items-center gap-2 text-red-600">
              <AlertCircle className="h-4 w-4" />
              <span className="text-sm">{logs.error}</span>
            </div>
          </CardContent>
        </Card>
      )}

      {/* Log Output */}
      <Card>
        <CardContent className="p-0">
          <div
            ref={logs.logsContainerRef}
            className="bg-black text-white font-mono text-sm max-h-[60vh] overflow-y-auto p-4"
            onScroll={logs.handleScroll}
          >
            {logs.loading ? (
              <div className="flex items-center justify-center h-32 text-gray-500">
                <div className="animate-spin rounded-full h-8 w-8 border-b-2 border-gray-500 mr-2"></div>
                Loading logs...
              </div>
            ) : logs.logs.length === 0 ? (
              <div className="flex items-center justify-center h-32 text-gray-500">
                {logs.error ? 'Failed to load logs' : 'No logs available'}
              </div>
            ) : (
              <div>
                {logs.logs.map((log, index) => (
                  <div
                    key={index}
                    className="whitespace-pre-wrap break-words"
                    dangerouslySetInnerHTML={{
                      __html: convertAnsiToHtml(log)
                    }}
                  />
                ))}
                <div ref={logs.logsEndRef} />
              </div>
            )}
          </div>
        </CardContent>
      </Card>
    </div>
  )
}