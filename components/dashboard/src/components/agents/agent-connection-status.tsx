'use client'

import { AgentChatStatus } from '@/types/chat'
import { Bot, Wifi, WifiOff, Loader2, AlertTriangle, CheckCircle } from 'lucide-react'
import { cn } from '@/lib/utils'

interface AgentConnectionStatusProps {
  status: AgentChatStatus | null
  isLoading?: boolean
  className?: string
  showLabel?: boolean
}

export function AgentConnectionStatus({ 
  status, 
  isLoading = false,
  className,
  showLabel = true 
}: AgentConnectionStatusProps) {
  if (isLoading) {
    return (
      <div className={cn("flex items-center gap-2", className)}>
        <div className="relative">
          <Bot className="h-4 w-4 text-stone-400" />
          <Loader2 className="absolute -top-1 -right-1 h-3 w-3 animate-spin text-amber-600" />
        </div>
        {showLabel && (
          <span className="text-xs text-stone-600 dark:text-stone-400">
            Connecting...
          </span>
        )}
      </div>
    )
  }

  if (!status) {
    return (
      <div className={cn("flex items-center gap-2", className)}>
        <div className="relative">
          <Bot className="h-4 w-4 text-stone-400" />
          <WifiOff className="absolute -top-1 -right-1 h-3 w-3 text-stone-500" />
        </div>
        {showLabel && (
          <span className="text-xs text-stone-600 dark:text-stone-400">
            Unavailable
          </span>
        )}
      </div>
    )
  }

  const getStatusInfo = () => {
    if (!status.chatAvailable) {
      switch (status.status) {
        case 'Pending':
          return {
            icon: <Loader2 className="h-3 w-3 animate-spin text-amber-600" />,
            label: 'Starting up',
            color: 'text-amber-600'
          }
        case 'Failed':
          return {
            icon: <AlertTriangle className="h-3 w-3 text-red-500" />,
            label: 'Failed',
            color: 'text-red-500'
          }
        default:
          return {
            icon: <WifiOff className="h-3 w-3 text-stone-500" />,
            label: 'Offline',
            color: 'text-stone-500'
          }
      }
    }

    return {
      icon: <CheckCircle className="h-3 w-3 text-green-500" />,
      label: 'Connected',
      color: 'text-green-600'
    }
  }

  const statusInfo = getStatusInfo()

  return (
    <div className={cn("flex items-center gap-2", className)}>
      <div className="relative">
        <Bot className="h-4 w-4 text-blue-600 dark:text-blue-400" />
        <div className="absolute -top-1 -right-1">
          {statusInfo.icon}
        </div>
      </div>
      {showLabel && (
        <div className="flex flex-col">
          <span className={cn("text-xs font-medium", statusInfo.color)}>
            {statusInfo.label}
          </span>
          {status.agentName && (
            <span className="text-xs text-stone-500 dark:text-stone-400">
              {status.agentName}
            </span>
          )}
        </div>
      )}
      
      {/* Connection pulse animation when connected */}
      {status.chatAvailable && (
        <div className="absolute -top-1 -right-1">
          <div className="h-2 w-2 bg-green-500 rounded-full">
            <div className="h-2 w-2 bg-green-500 rounded-full animate-ping" />
          </div>
        </div>
      )}
    </div>
  )
}

// Compact version for use in tables and lists
export function AgentConnectionIndicator({ 
  status, 
  isLoading = false,
  className 
}: Omit<AgentConnectionStatusProps, 'showLabel'>) {
  return (
    <AgentConnectionStatus 
      status={status}
      isLoading={isLoading}
      showLabel={false}
      className={className}
    />
  )
}