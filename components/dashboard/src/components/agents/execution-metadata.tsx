'use client'

import { Clock, Calendar, Activity, Hash } from 'lucide-react'
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

interface ExecutionMetadataProps {
  execution: AgentExecution
}

export function ExecutionMetadata({ execution }: ExecutionMetadataProps) {
  const formatDateTime = (date: Date) => {
    return new Date(date).toLocaleString('en-US', {
      year: 'numeric',
      month: 'short',
      day: 'numeric',
      hour: '2-digit',
      minute: '2-digit',
      second: '2-digit',
      hour12: true
    })
  }

  const formatDuration = (ms: number) => {
    const seconds = Math.floor(ms / 1000)
    const minutes = Math.floor(seconds / 60)
    const hours = Math.floor(minutes / 60)

    if (hours > 0) {
      const remainingMinutes = minutes % 60
      const remainingSeconds = seconds % 60
      return `${hours}h ${remainingMinutes}m ${remainingSeconds}s`
    } else if (minutes > 0) {
      const remainingSeconds = seconds % 60
      return `${minutes}m ${remainingSeconds}s`
    } else {
      return `${seconds}s`
    }
  }

  const formatExecutionId = (executionId: string) => {
    const match = executionId.match(/\d+/)
    return match ? `#${match[0]}` : executionId
  }

  return (
    <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-4 gap-6">
      {/* Execution ID */}
      <div className="flex items-center gap-3">
        <div className="flex-shrink-0">
          <Hash className="h-5 w-5 text-muted-foreground" />
        </div>
        <div>
          <p className="text-sm font-medium text-muted-foreground">Execution ID</p>
          <p className="text-lg font-semibold text-foreground">
            {formatExecutionId(execution.executionId)}
          </p>
        </div>
      </div>

      {/* Duration */}
      <div className="flex items-center gap-3">
        <div className="flex-shrink-0">
          <Clock className="h-5 w-5 text-muted-foreground" />
        </div>
        <div>
          <p className="text-sm font-medium text-muted-foreground">Total Duration</p>
          <p className="text-lg font-semibold text-foreground">
            {formatDuration(execution.duration)}
          </p>
        </div>
      </div>

      {/* Span Count */}
      <div className="flex items-center gap-3">
        <div className="flex-shrink-0">
          <Activity className="h-5 w-5 text-muted-foreground" />
        </div>
        <div>
          <p className="text-sm font-medium text-muted-foreground">Number of Spans</p>
          <p className="text-lg font-semibold text-foreground">
            {execution.spanCount}
          </p>
        </div>
      </div>

      {/* Root Operation */}
      <div className="flex items-center gap-3">
        <div className="flex-shrink-0">
          <Calendar className="h-5 w-5 text-muted-foreground" />
        </div>
        <div>
          <p className="text-sm font-medium text-muted-foreground">Root Operation</p>
          <p className="text-lg font-semibold text-foreground truncate">
            {execution.rootSpanName}
          </p>
        </div>
      </div>

      {/* Start Time */}
      <div className="md:col-span-2">
        <p className="text-sm font-medium text-muted-foreground mb-1">Start Time</p>
        <p className="text-base text-foreground">
          {formatDateTime(execution.startTime)}
        </p>
      </div>

      {/* End Time */}
      <div className="md:col-span-2">
        <p className="text-sm font-medium text-muted-foreground mb-1">End Time</p>
        <p className="text-base text-foreground">
          {formatDateTime(execution.endTime)}
        </p>
      </div>

      {/* Trace ID */}
      <div className="md:col-span-2 lg:col-span-4">
        <p className="text-sm font-medium text-muted-foreground mb-1">Trace ID</p>
        <p className="text-base font-mono text-foreground bg-muted px-3 py-2 rounded border">
          {execution.traceId}
        </p>
      </div>
    </div>
  )
}