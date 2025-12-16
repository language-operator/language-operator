'use client'

import { Badge } from '@/components/ui/badge'
import { cn } from '@/lib/utils'
import { CheckCircle, AlertCircle, Clock, XCircle } from 'lucide-react'

export interface AnimatedStatusProps {
  status: 'Ready' | 'Pending' | 'Failed' | 'Error' | 'Unknown' | string
  size?: 'sm' | 'md' | 'lg'
  showIcon?: boolean
  className?: string
}

export function AnimatedStatus({ 
  status, 
  size = 'md', 
  showIcon = true, 
  className 
}: AnimatedStatusProps) {
  const getStatusConfig = () => {
    const normalizedStatus = status.toLowerCase()
    
    switch (normalizedStatus) {
      case 'ready':
      case 'running':
        return {
          variant: 'default' as const,
          icon: CheckCircle,
          className: 'bg-green-100 text-green-800 dark:bg-green-900/20 dark:text-green-300',
          iconClass: 'text-green-500',
          animation: ''
        }
      case 'pending':
      case 'creating':
      case 'updating':
        return {
          variant: 'secondary' as const,
          icon: Clock,
          className: 'bg-yellow-100 text-yellow-800 dark:bg-yellow-900/20 dark:text-yellow-300',
          iconClass: 'text-yellow-500',
          animation: 'animate-pulse'
        }
      case 'failed':
      case 'error':
        return {
          variant: 'destructive' as const,
          icon: XCircle,
          className: 'bg-red-100 text-red-800 dark:bg-red-900/20 dark:text-red-300',
          iconClass: 'text-red-500',
          animation: ''
        }
      default:
        return {
          variant: 'outline' as const,
          icon: AlertCircle,
          className: 'bg-gray-100 text-gray-800 dark:bg-gray-800 dark:text-gray-300',
          iconClass: 'text-gray-500',
          animation: ''
        }
    }
  }

  const config = getStatusConfig()
  const StatusIcon = config.icon

  const sizeClasses = {
    sm: 'h-4 px-1.5 text-xs',
    md: 'h-5 px-2 text-xs', 
    lg: 'h-6 px-2.5 text-sm'
  }

  const iconSizes = {
    sm: 'h-2.5 w-2.5',
    md: 'h-3 w-3',
    lg: 'h-3.5 w-3.5'
  }

  return (
    <Badge
      variant={config.variant}
      className={cn(
        sizeClasses[size],
        config.className,
        config.animation,
        'inline-flex items-center gap-1 font-medium transition-all duration-200',
        className
      )}
    >
      {showIcon && (
        <StatusIcon 
          className={cn(
            iconSizes[size],
            config.iconClass,
            config.animation
          )} 
        />
      )}
      {status}
    </Badge>
  )
}

// Predefined status components for common use cases
export function ReadyStatus({ size, className }: Omit<AnimatedStatusProps, 'status'>) {
  return <AnimatedStatus status="Ready" size={size} className={className} />
}

export function PendingStatus({ size, className }: Omit<AnimatedStatusProps, 'status'>) {
  return <AnimatedStatus status="Pending" size={size} className={className} />
}

export function FailedStatus({ size, className }: Omit<AnimatedStatusProps, 'status'>) {
  return <AnimatedStatus status="Failed" size={size} className={className} />
}

export function ErrorStatus({ size, className }: Omit<AnimatedStatusProps, 'status'>) {
  return <AnimatedStatus status="Error" size={size} className={className} />
}