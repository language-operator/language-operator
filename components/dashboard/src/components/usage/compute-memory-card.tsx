'use client'

import { cn } from '@/lib/utils'
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card'
import { USAGE_STATUS_COLORS, type ComputeMemoryCardData } from '@/types/usage'
import { ComputeIcon, MemoryIcon } from '@/components/ui/icons'

interface ComputeMemoryCardProps {
  data: ComputeMemoryCardData
  className?: string
}

function UsageBar({ 
  label, 
  current, 
  limit, 
  percentage, 
  status 
}: {
  label: string
  current: string | number
  limit: string | number
  percentage: number
  status: 'healthy' | 'warning' | 'critical' | 'limit' | 'exceeded'
}) {
  const statusColor = USAGE_STATUS_COLORS[status]

  return (
    <div className="space-y-2">
      {/* Label and values */}
      <div className="flex items-center justify-between">
        <span className="text-xs font-light text-stone-600 dark:text-stone-400 uppercase tracking-wider">
          {label}
        </span>
        <div className="flex items-center gap-1 text-xs font-light text-stone-600 dark:text-stone-400">
          <span className="text-stone-900 dark:text-stone-300">{current}</span>
          <span>/</span>
          <span>{limit}</span>
        </div>
      </div>
      
      {/* Progress bar */}
      <div className="w-full bg-stone-200 dark:bg-stone-800 h-1">
        <div 
          className={cn(
            "h-full transition-all duration-500",
            statusColor
          )}
          style={{ 
            width: `${Math.min(percentage, 100)}%`,
            minWidth: percentage > 0 ? '2px' : '0px'
          }}
        />
      </div>
      
      {/* Percentage */}
      <div className="text-right">
        <span className="text-xs font-light text-stone-500 dark:text-stone-500">
          {percentage.toFixed(1)}%
        </span>
      </div>
    </div>
  )
}

export function ComputeMemoryCard({ data, className }: ComputeMemoryCardProps) {
  const { resource, requests, limits } = data

  const getResourceIcon = () => {
    if (resource === 'CPU') {
      return <ComputeIcon className="h-4 w-4" />
    } else if (resource === 'MEMORY') {
      return <MemoryIcon className="h-4 w-4" />
    }
    return null
  }

  return (
    <Card className={cn("gap-6 py-6", className)}>
      <CardHeader className="px-8 space-y-4">
        <div className="flex items-center justify-between">
          <CardTitle className="text-sm font-light text-stone-600 dark:text-stone-400">
            {resource}
          </CardTitle>
          <div className="text-stone-400 dark:text-stone-500">
            {getResourceIcon()}
          </div>
        </div>
      </CardHeader>
      
      <CardContent className="px-8">
        <div className="space-y-6">
          {/* Requests bar */}
          <UsageBar
            label="Requests"
            current={requests.current}
            limit={requests.limit}
            percentage={requests.percentage}
            status={requests.status}
          />
          
          {/* Limits bar */}
          <UsageBar
            label="Limits"
            current={limits.current}
            limit={limits.limit}
            percentage={limits.percentage}
            status={limits.status}
          />
        </div>
      </CardContent>
    </Card>
  )
}