'use client'

import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card'
import { Badge } from '@/components/ui/badge'
import { 
  Activity,
  Bot, 
  Cpu, 
  Wrench, 
  Users, 
  Boxes,
  AlertCircle,
  CheckCircle,
  Clock,
  Loader2,
  Server,
  Layers,
  Zap
} from 'lucide-react'
import { useEvents, K8sEvent } from '@/hooks/useEvents'
import { cn } from '@/lib/utils'

interface EventsActivityProps {
  title?: string
  description?: string
  clusterName?: string     // Scope to specific cluster
  namespace?: string       // Scope to specific namespace
  resourceType?: string    // Scope to specific resource type
  resourceName?: string    // Scope to specific resource
  limit?: number           // Number of events to show (default: 10)
  className?: string       // Custom styling
  showNamespace?: boolean  // Whether to show namespace in event display
  showConnectionStatus?: boolean // Whether to show real-time connection status (default: true)
}

export function EventsActivity({
  title = "Events",
  description = "Recent Kubernetes events",
  clusterName,
  namespace,
  resourceType,
  resourceName,
  limit = 10,
  className,
  showNamespace = true,
  showConnectionStatus = true,
}: EventsActivityProps) {
  const { 
    data: eventsData, 
    isLoading, 
    error,
    refetch,
    isConnected,
    isRealtime 
  } = useEvents({ 
    clusterName,
    namespace,
    resourceType,
    resourceName,
    limit,
    enabled: true
  })

  const getEventIcon = (event: K8sEvent) => {
    const iconClass = "h-4 w-4 mt-0.5 flex-shrink-0"
    
    // First check event type for warnings/errors
    if (event.eventType === 'Warning' || event.action === 'failed') {
      return <AlertCircle className={`${iconClass} text-red-500`} />
    }

    // Then check by resource type
    switch (event.type) {
      case 'agent':
        return <Bot className={`${iconClass} text-blue-500`} />
      case 'model':
        return <Cpu className={`${iconClass} text-green-500`} />
      case 'tool':
        return <Wrench className={`${iconClass} text-purple-500`} />
      case 'persona':
        return <Users className={`${iconClass} text-indigo-500`} />
      case 'cluster':
        return <Boxes className={`${iconClass} text-orange-500`} />
      case 'pod':
        return <Server className={`${iconClass} text-cyan-500`} />
      case 'service':
        return <Layers className={`${iconClass} text-teal-500`} />
      case 'deployment':
        return <Zap className={`${iconClass} text-yellow-500`} />
      default:
        // Default based on event type
        if (event.eventType === 'Normal') {
          return <CheckCircle className={`${iconClass} text-stone-600 dark:text-stone-400`} />
        }
        return <Activity className={`${iconClass} text-stone-600 dark:text-stone-400`} />
    }
  }

  const getEventTypeBadge = (event: K8sEvent) => {
    if (event.eventType === 'Warning') {
      return <Badge variant="destructive" className="text-xs">Warning</Badge>
    }
    if (event.action === 'failed') {
      return <Badge variant="destructive" className="text-xs">Failed</Badge>
    }
    if (event.action === 'ready') {
      return <Badge variant="default" className="text-xs bg-green-100 text-green-800">Ready</Badge>
    }
    if (event.action === 'created') {
      return <Badge variant="default" className="text-xs bg-blue-100 text-blue-800">Created</Badge>
    }
    return null
  }

  const formatTimestamp = (timestamp: string) => {
    const date = new Date(timestamp)
    const now = new Date()
    const diffInMinutes = Math.floor((now.getTime() - date.getTime()) / (1000 * 60))
    
    if (diffInMinutes < 60) {
      return `${diffInMinutes}m ago`
    } else if (diffInMinutes < 1440) { // 24 hours
      const hours = Math.floor(diffInMinutes / 60)
      return `${hours}h ago`
    } else {
      const days = Math.floor(diffInMinutes / 1440)
      return `${days}d ago`
    }
  }

  const formatEventMessage = (event: K8sEvent) => {
    // If we have a formatted message from the API, use it
    if (event.message) {
      return event.message
    }
    
    // Otherwise, construct a message from the event data
    const resourceType = event.type.charAt(0).toUpperCase() + event.type.slice(1)
    return `${resourceType} "${event.resourceName}" ${event.action || event.reason}`
  }

  return (
    <Card className={cn("", className)}>
      <CardHeader className="pb-3">
        <div className="flex items-center justify-between">
          <div>
            <CardTitle className="flex items-center gap-2">
              {title}
              {/* Real-time connection indicator */}
              {showConnectionStatus && (
                <div className="flex items-center gap-1">
                  {isConnected ? (
                    <div className="flex items-center gap-1">
                      <div className="h-2 w-2 bg-green-500 rounded-full animate-pulse" />
                      {isRealtime && <span className="text-xs text-green-600 font-medium">Live</span>}
                    </div>
                  ) : (
                    <div className="h-2 w-2 bg-stone-400 dark:bg-stone-600 rounded-full" />
                  )}
                </div>
              )}
            </CardTitle>
            <CardDescription className="text-sm">
              {description}
              {showConnectionStatus && !isConnected && (
                <span className="text-amber-600"> • Using cached data</span>
              )}
            </CardDescription>
          </div>
          {/* Show scope information if filtered */}
          <div className="text-xs text-stone-600 dark:text-stone-400 flex flex-col items-end gap-1">
            {clusterName && (
              <div className="flex items-center gap-1">
                <Boxes className="h-3 w-3" />
                {clusterName}
              </div>
            )}
            {resourceName && (
              <div className="flex items-center gap-1">
                {resourceType && getEventIcon({ type: resourceType } as K8sEvent)}
                {resourceName}
              </div>
            )}
          </div>
        </div>
      </CardHeader>
      <CardContent className="pt-0">
        {isLoading ? (
          <div className="space-y-3">
            {[1, 2, 3].map((i) => (
              <div key={i} className="flex items-start space-x-3">
                <div className="h-4 w-4 bg-stone-200 dark:bg-stone-700 mt-0.5 animate-pulse flex-shrink-0" />
                <div className="flex-1 min-w-0">
                  <div className="h-4 bg-stone-200 dark:bg-stone-700 animate-pulse mb-1" />
                  <div className="h-3 bg-stone-100 dark:bg-stone-800 animate-pulse w-3/4" />
                </div>
              </div>
            ))}
          </div>
        ) : error ? (
          <div className="text-center py-6">
            <AlertCircle className="h-8 w-8 text-stone-500 dark:text-stone-400 mx-auto mb-2" />
            <p className="text-sm text-stone-600 dark:text-stone-400">Failed to load events</p>
            <button 
              onClick={() => refetch()}
              className="text-xs text-blue-600 hover:text-blue-800 mt-1"
            >
              Try again
            </button>
          </div>
        ) : !eventsData?.data || eventsData.data.length === 0 ? (
          <div className="text-center py-6">
            <Clock className="h-8 w-8 text-stone-500 dark:text-stone-400 mx-auto mb-2" />
            <p className="text-sm text-stone-600 dark:text-stone-400">No recent events</p>
            <p className="text-xs text-stone-500 dark:text-stone-500 mt-1">
              Events will appear here when resources are created or updated
            </p>
          </div>
        ) : (
          <div className="space-y-3">
            {eventsData.data.map((event) => (
              <div key={event.id} className="flex items-start space-x-3">
                {getEventIcon(event)}
                <div className="flex-1 min-w-0">
                  <div className="flex items-start justify-between">
                    <div className="flex-1 min-w-0">
                      <p className="text-sm font-medium text-stone-900 dark:text-stone-300 leading-tight">
                        {formatEventMessage(event)}
                      </p>
                      <div className="flex items-center gap-2 mt-1 text-xs text-stone-600 dark:text-stone-400">
                        <span>{formatTimestamp(event.timestamp)}</span>
                        {showNamespace && event.namespace && event.namespace !== 'default' && (
                          <>
                            <span>•</span>
                            <span>{event.namespace}</span>
                          </>
                        )}
                        {event.count && event.count > 1 && (
                          <>
                            <span>•</span>
                            <span>{event.count}x</span>
                          </>
                        )}
                      </div>
                    </div>
                    <div className="flex-shrink-0 ml-2">
                      {getEventTypeBadge(event)}
                    </div>
                  </div>
                </div>
              </div>
            ))}
            
            {/* Show total count if there are more events */}
            {eventsData.total > eventsData.data.length && (
              <div className="pt-2 border-t border-stone-200 dark:border-stone-700">
                <p className="text-xs text-stone-600 dark:text-stone-400 text-center">
                  Showing {eventsData.data.length} of {eventsData.total} events
                </p>
              </div>
            )}
          </div>
        )}
      </CardContent>
    </Card>
  )
}

// Convenience components for common use cases
export function ClusterEventsActivity({ 
  clusterName, 
  ...props 
}: Omit<EventsActivityProps, 'clusterName'> & { clusterName: string }) {
  return (
    <EventsActivity
      {...props}
      clusterName={clusterName}
      title="Cluster Events"
      description={`Real-time events for ${clusterName}`}
    />
  )
}

export function ResourceEventsActivity({ 
  resourceType,
  resourceName,
  namespace,
  ...props 
}: Omit<EventsActivityProps, 'resourceType' | 'resourceName' | 'namespace'> & {
  resourceType: string
  resourceName: string
  namespace: string
}) {
  const resourceDisplayName = resourceType.charAt(0).toUpperCase() + resourceType.slice(1)
  
  return (
    <EventsActivity
      {...props}
      resourceType={resourceType}
      resourceName={resourceName}
      namespace={namespace}
      title={`${resourceDisplayName} Events`}
      description={`Real-time events for ${resourceName}`}
      showNamespace={false} // Don't show namespace for resource-scoped events
    />
  )
}