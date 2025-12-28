'use client'

import { useState, useEffect } from 'react'
import { useRouter, usePathname, useSearchParams } from 'next/navigation'
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card'
import { Badge } from '@/components/ui/badge'
import { ExecutionDropdown } from './execution-dropdown'
import { AgentTraceViewer } from './agent-trace-viewer' 
import { ExecutionMetadata } from './execution-metadata'
import { useAgentExecutions } from '@/hooks/use-agent-executions'
import { Activity, Clock, CheckCircle, XCircle, AlertCircle } from 'lucide-react'
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

interface AgentTelemetryPanelProps {
  agent: any
  clusterName: string
}

export function AgentTelemetryPanel({ agent, clusterName }: AgentTelemetryPanelProps) {
  const [selectedExecution, setSelectedExecution] = useState<AgentExecution | null>(null)
  const router = useRouter()
  const pathname = usePathname()
  const searchParams = useSearchParams()
  
  const { 
    data: executions, 
    isLoading, 
    error 
  } = useAgentExecutions(agent.metadata.name, clusterName)

  // Handle URL-based trace selection and auto-select most recent execution
  useEffect(() => {
    if (!executions?.data || executions.data.length === 0) return

    const urlTraceId = searchParams.get('traceId')
    
    if (urlTraceId) {
      // If there's a trace ID in the URL, try to find and select that execution
      const foundExecution = executions.data.find(exec => exec.traceId === urlTraceId)
      if (foundExecution && foundExecution !== selectedExecution) {
        setSelectedExecution(foundExecution)
      }
    } else if (!selectedExecution) {
      // No trace ID in URL and no execution selected, auto-select most recent
      const mostRecent = executions.data[0]
      setSelectedExecution(mostRecent)
      // Update URL to reflect the selected trace
      const newUrl = `${window.location.pathname}?traceId=${mostRecent.traceId}`
      window.history.pushState(null, '', newUrl)
    }
  }, [executions?.data, selectedExecution, searchParams])

  // Handle execution selection with URL update (but stay on same page)
  const handleExecutionSelect = (execution: AgentExecution) => {
    setSelectedExecution(execution)
    // Use shallow routing to update the URL without navigating to a new page
    const newUrl = `${window.location.pathname}?traceId=${execution.traceId}`
    window.history.pushState(null, '', newUrl)
  }

  const getStatusIcon = (status: string) => {
    switch (status) {
      case 'success':
        return <CheckCircle className="h-4 w-4 text-green-600" />
      case 'error':
        return <XCircle className="h-4 w-4 text-red-600" />
      case 'running':
        return <AlertCircle className="h-4 w-4 text-yellow-600" />
      default:
        return <Activity className="h-4 w-4 text-gray-600" />
    }
  }

  const getStatusColor = (status: string) => {
    switch (status) {
      case 'success':
        return 'bg-green-100 text-green-800 border-green-200'
      case 'error':
        return 'bg-red-100 text-red-800 border-red-200'
      case 'running':
        return 'bg-yellow-100 text-yellow-800 border-yellow-200'
      default:
        return 'bg-gray-100 text-gray-800 border-gray-200'
    }
  }

  const formatDuration = (ms: number) => {
    const seconds = Math.floor(ms / 1000)
    if (seconds < 60) {
      return `${seconds}s`
    }
    const minutes = Math.floor(seconds / 60)
    const remainingSeconds = seconds % 60
    return `${minutes}m ${remainingSeconds}s`
  }

  if (isLoading) {
    return (
      <div className="space-y-6">
        <Card>
          <CardContent className="flex items-center justify-center py-16">
            <div className="text-center">
              <div className="animate-spin rounded-full h-12 w-12 border-b-2 border-foreground mx-auto mb-4"></div>
              <p className="text-muted-foreground">Loading execution data...</p>
            </div>
          </CardContent>
        </Card>
      </div>
    )
  }

  if (error) {
    return (
      <Card>
        <CardContent className="flex items-center justify-center py-16">
          <div className="text-center">
            <XCircle className="h-12 w-12 text-red-600 mx-auto mb-4" />
            <p className="text-muted-foreground">Failed to load telemetry data</p>
            <p className="text-sm text-muted-foreground mt-2">{error.message}</p>
          </div>
        </CardContent>
      </Card>
    )
  }

  const executionList = executions?.data || []

  if (executionList.length === 0) {
    return (
      <Card>
        <CardContent className="flex items-center justify-center py-16">
          <div className="text-center">
            <Activity className="h-12 w-12 text-muted-foreground mx-auto mb-4" />
            <p className="text-foreground">No execution traces found</p>
            <p className="text-sm text-muted-foreground mt-2">
              Executions will appear here once the agent runs
            </p>
          </div>
        </CardContent>
      </Card>
    )
  }

  return (
    <div className="space-y-6">
      {/* Execution Selection */}
      <Card>
        <CardHeader>
          <CardTitle className="flex items-center gap-2">
            <Activity className="h-5 w-5" />
            Executions
          </CardTitle>
        </CardHeader>
        <CardContent>
          <ExecutionDropdown
            executions={executionList}
            selectedExecution={selectedExecution}
            onExecutionSelect={handleExecutionSelect}
            clusterName={clusterName}
            agentName={agent.metadata.name}
          />
        </CardContent>
      </Card>

      {selectedExecution && (
        <>
          {/* Agent Trace Viewer */}
          <AgentTraceViewer
            execution={selectedExecution}
            clusterName={clusterName}
            agentName={agent.metadata.name}
          />
        </>
      )}

      {!selectedExecution && executionList.length > 0 && (
        <Card>
          <CardContent className="flex items-center justify-center py-16">
            <div className="text-center">
              <Activity className="h-12 w-12 text-muted-foreground mx-auto mb-4" />
              <p className="text-foreground">Select an execution to view trace details</p>
              <p className="text-sm text-muted-foreground mt-2">
                Choose from {executionList.length} recent executions above
              </p>
            </div>
          </CardContent>
        </Card>
      )}
    </div>
  )
}