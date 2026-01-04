'use client'

import { cn } from '@/lib/utils'
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card'
import { USAGE_STATUS_COLORS, type UsageCardData } from '@/types/usage'

interface ResourceUsageCardProps {
  data: UsageCardData
  className?: string
}

function UsageStatusBadge({ percentage }: { percentage: number }) {
  if (percentage >= 96) {
    return (
      <span className="text-[10px] tracking-widest uppercase font-light text-red-600 dark:text-red-400">
        Exceeded
      </span>
    )
  } else if (percentage >= 86) {
    return (
      <span className="text-[10px] tracking-widest uppercase font-light text-orange-600 dark:text-orange-400">
        Critical
      </span>
    )
  } else if (percentage >= 71) {
    return (
      <span className="text-[10px] tracking-widest uppercase font-light text-amber-600 dark:text-amber-400">
        Warning
      </span>
    )
  } else {
    return (
      <span className="text-[10px] tracking-widest uppercase font-light text-stone-600 dark:text-stone-400">
        Healthy
      </span>
    )
  }
}

export function ResourceUsageCard({ data, className }: ResourceUsageCardProps) {
  const { resource, current, limit, percentage, status, description } = data
  const statusColor = USAGE_STATUS_COLORS[status]

  return (
    <Card className={cn("gap-6 py-6", className)}>
      <CardHeader className="px-8 space-y-4">
        <CardTitle className="text-sm font-light text-stone-600 dark:text-stone-400">
          {resource}
        </CardTitle>
      </CardHeader>
      
      <CardContent className="px-8">
        <div className="space-y-6">
          {/* Usage Numbers */}
          <div className="flex items-end gap-2">
            <span className="text-2xl font-light text-stone-900 dark:text-stone-300">
              {current}
            </span>
            <span className="text-sm font-light text-stone-600 dark:text-stone-400 mb-1">
              / {limit}
            </span>
          </div>
          
          {/* Marfa Progress Bar - Clean 4px horizontal bar with earth tones */}
          <div className="w-full">
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
          </div>
          
          {/* Status Information */}
          <div className="flex items-center justify-between text-xs font-light">
            <span className="text-stone-600 dark:text-stone-400">
              {percentage.toFixed(1)}% used
            </span>
            <UsageStatusBadge percentage={percentage} />
          </div>
          
          {/* Description */}
          {description && (
            <p className="text-xs font-light text-stone-500 dark:text-stone-500 leading-relaxed">
              {description}
            </p>
          )}
        </div>
      </CardContent>
    </Card>
  )
}