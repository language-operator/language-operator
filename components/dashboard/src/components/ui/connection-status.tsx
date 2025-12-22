'use client'

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
  connectionError,
  className
}: ConnectionStatusProps) {
  const getDotColor = () => {
    if (connectionError) {
      return 'bg-red-500'
    }

    if (isConnected) {
      return 'bg-green-500'
    }

    return 'bg-yellow-500'
  }

  return (
    <div className={cn('h-2 w-2 rounded-full', getDotColor(), className)} />
  )
}