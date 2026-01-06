'use client'

import { useState, useRef, useEffect } from 'react'
import { ChevronDown, Clock, CheckCircle, XCircle, AlertCircle, ExternalLink } from 'lucide-react'
import Link from 'next/link'
import { Button } from '@/components/ui/button'
import { Badge } from '@/components/ui/badge'
import { useOrganization } from '@/components/organization-provider'
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuTrigger,
} from '@/components/ui/dropdown-menu'
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

interface ExecutionDropdownProps {
  executions: AgentExecution[]
  selectedExecution: AgentExecution | null
  onExecutionSelect: (execution: AgentExecution) => void
  clusterName: string
  agentName: string
}

export function ExecutionDropdown({
  executions,
  selectedExecution,
  onExecutionSelect,
  clusterName,
  agentName
}: ExecutionDropdownProps) {
  const { getOrgUrl } = useOrganization()
  const triggerRef = useRef<HTMLButtonElement>(null)
  const [triggerWidth, setTriggerWidth] = useState<number>(0)

  useEffect(() => {
    if (triggerRef.current) {
      const updateWidth = () => {
        setTriggerWidth(triggerRef.current?.offsetWidth || 0)
      }
      updateWidth()
      window.addEventListener('resize', updateWidth)
      return () => window.removeEventListener('resize', updateWidth)
    }
  }, [])
  const getStatusIcon = (status: string) => {
    switch (status) {
      case 'success':
        return <CheckCircle className="h-4 w-4 text-green-600" />
      case 'error':
        return <XCircle className="h-4 w-4 text-red-600" />
      case 'running':
        return <AlertCircle className="h-4 w-4 text-yellow-600" />
      default:
        return <Clock className="h-4 w-4 text-gray-600" />
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

  const formatTimestamp = (date: Date) => {
    const now = new Date()
    const diffMs = now.getTime() - new Date(date).getTime()
    const diffMins = Math.floor(diffMs / (1000 * 60))
    const diffHours = Math.floor(diffMins / 60)
    const diffDays = Math.floor(diffHours / 24)

    if (diffMins < 1) {
      return 'Just now'
    } else if (diffMins < 60) {
      return `${diffMins}m ago`
    } else if (diffHours < 24) {
      return `${diffHours}h ago`
    } else {
      return `${diffDays}d ago`
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

  const formatExecutionId = (executionId: string) => {
    // Extract number from execution ID for display
    const match = executionId.match(/\d+/)
    return match ? `#${match[0]}` : executionId
  }

  return (
    <div className="w-full">
      <DropdownMenu>
        <DropdownMenuTrigger asChild>
          <Button
            ref={triggerRef}
            variant="outline"
            className="w-full justify-between h-auto p-4"
          >
            {selectedExecution ? (
              <div className="flex items-center justify-between w-full">
                <div className="flex items-center gap-3">
                  {getStatusIcon(selectedExecution.status)}
                  <div className="text-left">
                    <div className="font-mono text-sm">
                      {selectedExecution.traceId}
                    </div>
                    <div className="text-sm text-muted-foreground" style={{ textTransform: 'none' }}>
                      {formatTimestamp(selectedExecution.startTime)} • {formatDuration(selectedExecution.duration)}
                    </div>
                  </div>
                </div>
                <div className="flex items-center gap-2">
                  <Badge className={getStatusColor(selectedExecution.status)}>
                    {selectedExecution.status}
                  </Badge>
                  <ChevronDown className="h-4 w-4" />
                </div>
              </div>
            ) : (
              <div className="flex items-center justify-between w-full">
                <span className="text-muted-foreground">Select an execution...</span>
                <ChevronDown className="h-4 w-4" />
              </div>
            )}
          </Button>
        </DropdownMenuTrigger>
        <DropdownMenuContent 
          className="max-h-[300px] overflow-y-auto"
          align="start"
          sideOffset={4}
          style={{ 
            width: triggerWidth > 0 ? `${triggerWidth}px` : 'auto'
          }}
        >
          {executions.map((execution) => (
            <DropdownMenuItem
              key={execution.executionId}
              onClick={() => onExecutionSelect(execution)}
              className="p-4 cursor-pointer"
            >
              <div className="flex items-center justify-between w-full">
                <div className="flex items-center gap-3">
                  {getStatusIcon(execution.status)}
                  <div>
                    <div className="font-mono text-sm">
                      {execution.traceId}
                    </div>
                    <div className="text-sm text-muted-foreground" style={{ textTransform: 'none' }}>
                      {formatTimestamp(execution.startTime)} • {formatDuration(execution.duration)}
                    </div>
                  </div>
                </div>
                <div className="flex items-center gap-2">
                  <Link
                    href={getOrgUrl(`/clusters/${clusterName}/agents/${agentName}/traces/trace/${execution.traceId}`)}
                    className="p-1 hover:bg-accent rounded"
                    onClick={(e) => e.stopPropagation()}
                  >
                    <ExternalLink className="h-3 w-3" />
                  </Link>
                  <Badge className={getStatusColor(execution.status)}>
                    {execution.status}
                  </Badge>
                </div>
              </div>
            </DropdownMenuItem>
          ))}
        </DropdownMenuContent>
      </DropdownMenu>
    </div>
  )
}