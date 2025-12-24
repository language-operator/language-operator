'use client'

import React from 'react'
import Link from 'next/link'
import { Button } from '@/components/ui/button'
import { ArrowLeft, LucideIcon } from 'lucide-react'
import { cn } from '@/lib/utils'

interface ResourceHeaderProps {
  /** Optional back navigation */
  backHref?: string
  backLabel?: string
  
  /** Icon configuration */
  icon: LucideIcon
  iconBgColor?: string
  
  /** Title and subtitle */
  title: string | React.ReactNode
  subtitle?: string | React.ReactNode
  
  /** Right-side actions */
  actions?: React.ReactNode
  
  /** Additional styling */
  className?: string
}

export function ResourceHeader({
  backHref,
  backLabel = 'Back',
  icon: Icon,
  iconBgColor,
  title,
  subtitle,
  actions,
  className
}: ResourceHeaderProps) {
  return (
    <div className={cn('flex items-center justify-between', className)}>
      <div className="flex items-center gap-3">
        {/* Optional Back Button */}
        {backHref && (
          <Button variant="ghost" size="sm" asChild>
            <Link href={backHref}>
              <ArrowLeft className="h-4 w-4" />
              <span className="sr-only">{backLabel}</span>
            </Link>
          </Button>
        )}
        
        {/* Resource Icon */}
        {iconBgColor ? (
          <div className={cn('flex items-center justify-center w-12 h-12 pl-4 pr-2', iconBgColor)}>
            <Icon className="h-6 w-6 text-amber-700 dark:text-amber-400" />
          </div>
        ) : (
          <div className="pl-4 pr-2">
            <Icon className="h-8 w-8 text-amber-700 dark:text-amber-400" />
          </div>
        )}
        
        {/* Title and Subtitle */}
        <div>
          <h1 className="text-3xl font-bold font-mono">{title}</h1>
          {subtitle && (
            <div className="text-stone-600 dark:text-stone-400">{subtitle}</div>
          )}
        </div>
      </div>
      
      {/* Actions */}
      {actions && (
        <div className="flex items-center gap-3">
          {actions}
        </div>
      )}
    </div>
  )
}