'use client'

import { useState } from 'react'
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card'
import { Badge } from '@/components/ui/badge'
import { ExecutionDropdown } from './execution-dropdown'
import { FlameChartTimeline } from './flame-chart-timeline' 
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
  spans: any[]
}

interface AgentTelemetryPanelProps {
  agent: any
  clusterName: string
}

export function AgentTelemetryPanel({ agent, clusterName }: AgentTelemetryPanelProps) {
  const [selectedExecution, setSelectedExecution] = useState<AgentExecution | null>(null)
  
  const { 
    data: executions, 
    isLoading, 
    error 
  } = useAgentExecutions(agent.metadata.name, clusterName)

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
            Recent Executions
          </CardTitle>
        </CardHeader>
        <CardContent>
          <ExecutionDropdown
            executions={executionList}
            selectedExecution={selectedExecution}
            onExecutionSelect={setSelectedExecution}
          />
        </CardContent>
      </Card>

      {selectedExecution && (
        <>
          {/* Execution Metadata */}
          <Card>
            <CardHeader>
              <CardTitle className="flex items-center justify-between">
                <div className="flex items-center gap-2">
                  <Clock className="h-5 w-5" />
                  Execution Summary
                </div>
                <div className="flex items-center gap-2">
                  {getStatusIcon(selectedExecution.status)}
                  <Badge className={getStatusColor(selectedExecution.status)}>
                    {selectedExecution.status}
                  </Badge>
                </div>
              </CardTitle>
            </CardHeader>
            <CardContent>
              <ExecutionMetadata execution={selectedExecution} />
            </CardContent>
          </Card>

          {/* Flame Chart */}
          <Card>
            <CardHeader>
              <CardTitle className="flex items-center gap-2">
                <Activity className="h-5 w-5" />
                Execution Timeline
              </CardTitle>
            </CardHeader>
            <CardContent>
              <FlameChartTimeline
                execution={selectedExecution}
                clusterName={clusterName}
                agentName={agent.metadata.name}
              />
            </CardContent>
          </Card>
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