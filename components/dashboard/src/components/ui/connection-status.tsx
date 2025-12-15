'use client'

import { useState, useEffect } from 'react'
import { Badge } from '@/components/ui/badge'
import { Button } from '@/components/ui/button'
// Simple tooltip using CSS hover instead of Radix
import { 
  Wifi, WifiOff, RefreshCw, AlertCircle, CheckCircle2, 
  Activity, Clock
} from 'lucide-react'
import { cn } from '@/lib/utils'

export interface ConnectionStatusProps {
  isConnected: boolean
  lastEvent?: { timestamp: string } | null
  connectionError?: string | null
  reconnectCount?: number
  onReconnect?: () => void
  className?: string
}

export function ConnectionStatus({
  isConnected,
  lastEvent,
  connectionError,
  reconnectCount = 0,
  onReconnect,
  className
}: ConnectionStatusProps) {
  const [timeSinceLastEvent, setTimeSinceLastEvent] = useState<string>('')

  // Update time since last event
  useEffect(() => {
    if (!lastEvent?.timestamp) {
      setTimeSinceLastEvent('')
      return
    }

    const updateTime = () => {
      const now = new Date()
      const eventTime = new Date(lastEvent.timestamp)
      const diffMs = now.getTime() - eventTime.getTime()
      
      if (diffMs < 60000) { // Less than 1 minute
        const seconds = Math.floor(diffMs / 1000)
        setTimeSinceLastEvent(`${seconds}s ago`)
      } else if (diffMs < 3600000) { // Less than 1 hour
        const minutes = Math.floor(diffMs / 60000)
        setTimeSinceLastEvent(`${minutes}m ago`)
      } else {
        const hours = Math.floor(diffMs / 3600000)
        setTimeSinceLastEvent(`${hours}h ago`)
      }
    }

    // Update immediately
    updateTime()

    // Update every 10 seconds
    const interval = setInterval(updateTime, 10000)
    return () => clearInterval(interval)
  }, [lastEvent?.timestamp])

  const getStatusInfo = () => {
    if (connectionError) {
      return {
        icon: AlertCircle,
        label: 'Connection Error',
        variant: 'destructive' as const,
        detail: connectionError,
        showReconnect: true
      }
    }
    
    if (!isConnected) {
      return {
        icon: WifiOff,
        label: reconnectCount > 0 ? 'Reconnecting...' : 'Disconnected',
        variant: 'secondary' as const,
        detail: reconnectCount > 0 ? `Attempt ${reconnectCount}` : 'No real-time updates',
        showReconnect: true
      }
    }

    return {
      icon: CheckCircle2,
      label: 'Connected',
      variant: 'default' as const,
      detail: timeSinceLastEvent ? `Last update ${timeSinceLastEvent}` : 'Real-time updates active',
      showReconnect: false
    }
  }

  const status = getStatusInfo()
  const StatusIcon = status.icon

  return (
    <div 
      className={cn(
        'relative flex items-center gap-2 px-2 py-1 rounded-md transition-colors group',
        'hover:bg-gray-50 dark:hover:bg-gray-800',
        className
      )}
      title={`${status.label} - ${status.detail}${reconnectCount > 0 ? ` (${reconnectCount} attempts)` : ''}`}
    >
      <StatusIcon className={cn(
        'h-3 w-3',
        status.variant === 'default' && 'text-green-500',
        status.variant === 'secondary' && 'text-gray-500',
        status.variant === 'destructive' && 'text-red-500',
        reconnectCount > 0 && 'animate-spin'
      )} />
      
      <Badge 
        variant={status.variant}
        className="text-xs px-1.5 py-0.5 h-5"
      >
        {isConnected && (
          <Activity className="h-2 w-2 mr-1 animate-pulse" />
        )}
        {status.label}
      </Badge>

      {status.showReconnect && onReconnect && (
        <Button
          size="sm"
          variant="ghost"
          className="h-5 px-1.5 text-xs"
          onClick={onReconnect}
        >
          <RefreshCw className="h-3 w-3" />
        </Button>
      )}

      {/* Simple CSS tooltip */}
      <div className="absolute bottom-full mb-2 left-1/2 transform -translate-x-1/2 px-2 py-1 bg-black text-white text-xs rounded opacity-0 group-hover:opacity-100 transition-opacity pointer-events-none whitespace-nowrap z-50">
        <div className="font-medium">{status.label}</div>
        <div>{status.detail}</div>
        {reconnectCount > 0 && (
          <div>Reconnect attempts: {reconnectCount}</div>
        )}
        {isConnected && (
          <div className="flex items-center gap-1 mt-1">
            <Wifi className="h-3 w-3" />
            Real-time updates enabled
          </div>
        )}
      </div>
    </div>
  )
}