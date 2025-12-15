'use client'

import { useEffect, useRef } from 'react'
import { toast } from 'sonner'
import { CheckCircle, AlertCircle, XCircle, Plus, Trash2, Edit } from 'lucide-react'
import { WatchEvent } from '@/hooks/use-watch'

export interface ResourceNotificationProps {
  onEvent: (event: WatchEvent) => void
  enabled?: boolean
  showAllEvents?: boolean // If false, only shows important events
}

export function useResourceNotifications({ 
  onEvent, 
  enabled = true, 
  showAllEvents = false 
}: ResourceNotificationProps) {
  // Track if we're in the initial sync period to avoid showing notifications for existing resources
  const initialSyncRef = useRef(true)
  const seenResourcesRef = useRef(new Set<string>())
  
  // Allow notifications for new resources after a delay (initial sync typically completes within 2-3 seconds)
  useEffect(() => {
    const timer = setTimeout(() => {
      initialSyncRef.current = false
    }, 3000) // Wait 3 seconds before showing ADDED notifications
    
    return () => clearTimeout(timer)
  }, [])
  
  const handleEvent = (event: WatchEvent) => {
    if (!enabled) return
    
    try {
      onEvent(event)
      
      const resourceName = event.data?.metadata?.name || 'Unknown'
      const resourceType = event.resource || 'Resource'
      const resourceKey = `${resourceType}-${resourceName}`
      
      // Skip ADDED events during initial sync period to avoid showing notifications for existing resources
      if (event.type === 'ADDED' && initialSyncRef.current) {
        seenResourcesRef.current.add(resourceKey)
        return
      }
      
      // For ADDED events after initial sync, only show if we haven't seen this resource before
      if (event.type === 'ADDED' && seenResourcesRef.current.has(resourceKey)) {
        return
      }
      
      // Track new resources
      if (event.type === 'ADDED') {
        seenResourcesRef.current.add(resourceKey)
      }
      
      // Remove from tracking when deleted
      if (event.type === 'DELETED') {
        seenResourcesRef.current.delete(resourceKey)
      }
      
      // Determine if we should show this event
      const shouldShow = showAllEvents || 
                        event.type === 'ADDED' || 
                        (event.type === 'MODIFIED' && isImportantStatusChange(event)) ||
                        event.type === 'DELETED' ||
                        event.type === 'ERROR'

      if (!shouldShow) return

      // Get notification config based on event type
      const getNotificationConfig = () => {
        switch (event.type) {
          case 'ADDED':
            return {
              title: `${resourceType.charAt(0).toUpperCase() + resourceType.slice(1)} Created`,
              description: `${resourceName} has been created`,
              icon: Plus,
              duration: 4000,
              className: 'border-green-200 bg-green-50'
            }
          case 'DELETED':
            return {
              title: `${resourceType.charAt(0).toUpperCase() + resourceType.slice(1)} Deleted`,
              description: `${resourceName} has been deleted`,
              icon: Trash2,
              duration: 4000,
              className: 'border-red-200 bg-red-50'
            }
          case 'MODIFIED':
            const status = event.data?.status?.phase
            const isStatusChange = isImportantStatusChange(event)
            
            if (isStatusChange && status) {
              const statusConfig = getStatusNotificationConfig(status)
              return {
                title: `${resourceType.charAt(0).toUpperCase() + resourceType.slice(1)} ${status}`,
                description: `${resourceName} is now ${status.toLowerCase()}`,
                icon: statusConfig.icon,
                duration: statusConfig.duration,
                className: statusConfig.className
              }
            }
            
            return {
              title: `${resourceType.charAt(0).toUpperCase() + resourceType.slice(1)} Updated`,
              description: `${resourceName} has been updated`,
              icon: Edit,
              duration: 3000,
              className: 'border-blue-200 bg-blue-50'
            }
          case 'ERROR':
            return {
              title: 'Resource Error',
              description: event.data?.error || `Error with ${resourceName}`,
              icon: AlertCircle,
              duration: 6000,
              className: 'border-red-200 bg-red-50'
            }
          default:
            return {
              title: 'Resource Event',
              description: `${resourceName} - ${event.type}`,
              icon: AlertCircle,
              duration: 3000,
              className: ''
            }
        }
      }

      const config = getNotificationConfig()
      const Icon = config.icon

      toast(config.title, {
        description: config.description,
        duration: config.duration,
        icon: <Icon className="h-4 w-4" />,
        className: config.className,
      })
    } catch (error) {
      console.error('Error in resource notification handler:', error)
    }
  }

  return handleEvent
}

function isImportantStatusChange(event: WatchEvent): boolean {
  const currentPhase = event.data?.status?.phase
  const previousPhase = event.data?.metadata?.annotations?.['langop.io/previous-phase']
  
  // Consider it important if phase changed to/from these states
  const importantPhases = ['Ready', 'Failed', 'Error', 'Pending']
  
  return importantPhases.includes(currentPhase) || 
         (previousPhase && previousPhase !== currentPhase)
}

function getStatusNotificationConfig(status: string) {
  switch (status.toLowerCase()) {
    case 'ready':
      return {
        icon: CheckCircle,
        duration: 4000,
        className: 'border-green-200 bg-green-50'
      }
    case 'failed':
    case 'error':
      return {
        icon: XCircle,
        duration: 6000,
        className: 'border-red-200 bg-red-50'
      }
    case 'pending':
      return {
        icon: AlertCircle,
        duration: 3000,
        className: 'border-yellow-200 bg-yellow-50'
      }
    default:
      return {
        icon: Edit,
        duration: 3000,
        className: 'border-blue-200 bg-blue-50'
      }
  }
}