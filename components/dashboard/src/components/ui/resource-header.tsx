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
  iconColor?: string
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
  iconColor = 'text-blue-600',
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
          <Button variant="outline" size="sm" asChild>
            <Link href={backHref}>
              <ArrowLeft className="h-4 w-4" />
              <span className="sr-only">{backLabel}</span>
            </Link>
          </Button>
        )}
        
        {/* Resource Icon */}
        {iconBgColor ? (
          <div className={cn('flex items-center justify-center w-12 h-12 rounded-lg pl-4 pr-2', iconBgColor)}>
            <Icon className={cn('h-6 w-6', iconColor)} />
          </div>
        ) : (
          <div className="pl-4 pr-2">
            <Icon className={cn('h-8 w-8', iconColor)} />
          </div>
        )}
        
        {/* Title and Subtitle */}
        <div>
          <h1 className="text-3xl font-bold font-mono">{title}</h1>
          {subtitle && (
            <div className="text-gray-600">{subtitle}</div>
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