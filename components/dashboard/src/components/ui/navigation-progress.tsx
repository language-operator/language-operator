'use client'

import { useEffect, useState } from 'react'
import { useNavigationLoading } from '@/contexts/navigation-loading-context'
import { cn } from '@/lib/utils'

interface NavigationProgressProps {
  className?: string
  height?: number
  speed?: number
}

export function NavigationProgress({ 
  className,
  height = 5, // 5px tall
  speed = 300 // Make it slower
}: NavigationProgressProps) {
  const { isNavigating } = useNavigationLoading()
  const [progress, setProgress] = useState(0)
  const [isVisible, setIsVisible] = useState(false)
  
  // Debug logging
  useEffect(() => {
    console.log('%cNavigationProgress: isNavigating =', 'color: #a855f7;', isNavigating, 'isVisible =', isVisible, 'progress =', progress)
  }, [isNavigating, isVisible, progress])

  useEffect(() => {
    let progressTimer: NodeJS.Timeout | undefined

    if (isNavigating) {
      setIsVisible(true)
      setProgress(0)

      // Animate progress until navigation completes
      let currentProgress = 0
      progressTimer = setInterval(() => {
        currentProgress += Math.random() * 15

        // Slow down as we approach completion
        if (currentProgress > 70) {
          currentProgress += Math.random() * 5
        }
        
        // Don't complete until navigation finishes
        if (currentProgress >= 90) {
          currentProgress = 90
        }
        
        setProgress(currentProgress)
      }, speed)
    } else if (isVisible) {
      // Complete the progress bar when navigation finishes
      if (progressTimer) {
        clearInterval(progressTimer)
      }
      
      setProgress(100)
      
      // TEMPORARY: Add manual sleep to make progress bar visible for testing
      setTimeout(() => {
        setIsVisible(false)
        setProgress(0)
      }, 1000) // 1 second to see completed bar
    }

    return () => {
      if (progressTimer) {
        clearInterval(progressTimer)
      }
    }
  }, [isNavigating, speed, isVisible])

  if (!isVisible) {
    return null
  }

  return (
    <div
      className={cn(
        "fixed top-0 left-0 z-50 transition-all duration-200 ease-out",
        className
      )}
      style={{
        height: `${height}px`,
        width: `${progress}%`,
      }}
    >
      <div 
        className="h-full bg-green-500" 
        style={{ 
          boxShadow: progress > 0 ? '0 0 10px rgba(34, 197, 94, 0.6), 0 0 5px rgba(34, 197, 94, 0.4)' : 'none' 
        }} 
      />
    </div>
  )
}

export function usePageReady() {
  const { setReady } = useNavigationLoading()
  return setReady
}