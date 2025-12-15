'use client'

import { useEffect, useState } from 'react'
import { cn } from '@/lib/utils'

export interface LiveCounterProps {
  value: number
  previousValue?: number
  className?: string
  duration?: number // Animation duration in milliseconds
}

export function LiveCounter({ 
  value, 
  previousValue, 
  className, 
  duration = 300 
}: LiveCounterProps) {
  const [displayValue, setDisplayValue] = useState(value)
  const [isAnimating, setIsAnimating] = useState(false)

  useEffect(() => {
    if (previousValue !== undefined && previousValue !== value) {
      setIsAnimating(true)
      
      // Animate the counter change
      const startTime = Date.now()
      const startValue = previousValue
      const endValue = value
      const difference = endValue - startValue

      const animate = () => {
        const elapsed = Date.now() - startTime
        const progress = Math.min(elapsed / duration, 1)
        
        // Easing function for smooth animation
        const easeOutQuart = 1 - Math.pow(1 - progress, 4)
        const currentValue = Math.round(startValue + (difference * easeOutQuart))
        
        setDisplayValue(currentValue)
        
        if (progress < 1) {
          requestAnimationFrame(animate)
        } else {
          setIsAnimating(false)
        }
      }
      
      requestAnimationFrame(animate)
    } else {
      setDisplayValue(value)
    }
  }, [value, previousValue, duration])

  const hasIncreased = previousValue !== undefined && value > previousValue
  const hasDecreased = previousValue !== undefined && value < previousValue

  return (
    <span 
      className={cn(
        'inline-block transition-all duration-200 font-bold text-2xl',
        isAnimating && 'scale-110',
        hasIncreased && 'text-green-600 dark:text-green-400',
        hasDecreased && 'text-red-600 dark:text-red-400',
        className
      )}
    >
      {displayValue}
      {hasIncreased && isAnimating && (
        <span className="ml-1 text-xs text-green-500 animate-bounce">↗</span>
      )}
      {hasDecreased && isAnimating && (
        <span className="ml-1 text-xs text-red-500 animate-bounce">↘</span>
      )}
    </span>
  )
}

export interface LiveCounterCardProps extends LiveCounterProps {
  title: string
  description?: string
  icon?: React.ComponentType<{ className?: string }>
}

export function LiveCounterCard({
  title,
  description,
  icon: Icon,
  value,
  previousValue,
  className
}: LiveCounterCardProps) {
  return (
    <div className={cn("p-4 bg-white dark:bg-gray-900 rounded-lg border", className)}>
      <div className="flex items-center justify-between">
        <div>
          <p className="text-sm font-medium text-gray-600 dark:text-gray-300">{title}</p>
          <LiveCounter 
            value={value} 
            previousValue={previousValue} 
            className="text-2xl" 
          />
          {description && (
            <p className="text-xs text-gray-500 dark:text-gray-400 mt-1">
              {description}
            </p>
          )}
        </div>
        {Icon && (
          <Icon className="h-4 w-4 text-gray-400 dark:text-gray-500" />
        )}
      </div>
    </div>
  )
}