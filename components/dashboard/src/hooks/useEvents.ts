import { useQuery } from '@tanstack/react-query'
import { useWatchEvents } from './use-watch'
import { useState, useEffect, useMemo } from 'react'

export interface K8sEvent {
  id: string
  type: 'agent' | 'model' | 'tool' | 'persona' | 'cluster' | 'pod' | 'service' | 'deployment'
  action: 'created' | 'updated' | 'scaled' | 'failed' | 'ready' | 'warning' | 'normal' | string
  resourceName: string
  namespace: string
  message: string
  timestamp: string
  reason: string
  source: string
  eventType: 'Normal' | 'Warning'
  count?: number
  firstTimestamp?: string
  lastTimestamp?: string
  involvedObject: {
    kind: string
    name: string
    namespace: string
    uid?: string
  }
}

export interface EventsResponse {
  success: boolean
  data: K8sEvent[]
  total: number
  namespace?: string
  clusterName?: string
}

export interface UseEventsOptions {
  clusterName?: string     // Scope to specific cluster
  namespace?: string       // Scope to specific namespace
  resourceType?: string    // Scope to specific resource type (e.g., 'agent', 'model')
  resourceName?: string    // Scope to specific resource instance
  limit?: number           // Number of events to fetch
  enabled?: boolean        // Whether to enable the query
  refetchInterval?: number // Auto-refresh interval in ms
}

export function useEvents(options: UseEventsOptions = {}) {
  const { 
    clusterName,
    namespace,
    resourceType,
    resourceName,
    limit = 20, 
    enabled = true,
    refetchInterval = 30000 // 30 seconds - used as fallback only
  } = options

  // State for managing real-time events
  const [realtimeEvents, setRealtimeEvents] = useState<K8sEvent[]>([])
  const [lastEventTime, setLastEventTime] = useState<string | null>(null)

  // Use SSE watch for real-time updates
  const { isConnected, lastEvent, connectionError } = useWatchEvents({
    enabled,
    clusterName,
    resourceType,
    resourceName,
    namespace,
    onEvent: (event) => {
      if (event.resource === 'event' && event.data) {
        const k8sEvent = event.data as K8sEvent
        
        // Add or update the event in our real-time list
        setRealtimeEvents(prev => {
          const filtered = prev.filter(e => e.id !== k8sEvent.id)
          const updated = [k8sEvent, ...filtered]
          return updated.slice(0, limit) // Keep only the most recent events
        })
        
        setLastEventTime(new Date().toISOString())
      }
    }
  })

  // Fallback to polling when SSE is not connected
  const queryResult = useQuery<EventsResponse>({
    queryKey: ['events', clusterName, namespace, resourceType, resourceName, limit],
    queryFn: async () => {
      const searchParams = new URLSearchParams({
        limit: limit.toString(),
      })

      // Add optional filters
      if (clusterName) searchParams.append('clusterName', clusterName)
      if (namespace) searchParams.append('namespace', namespace)
      if (resourceType) searchParams.append('resourceType', resourceType)
      if (resourceName) searchParams.append('resourceName', resourceName)

      const response = await fetch(`/api/events?${searchParams}`)
      if (!response.ok) {
        throw new Error('Failed to fetch events')
      }
      return response.json()
    },
    enabled: enabled && !isConnected, // Only poll when SSE is not connected
    staleTime: 10 * 1000, // Consider data stale after 10 seconds
    gcTime: 5 * 60 * 1000, // Keep in cache for 5 minutes
    refetchInterval: (enabled && !isConnected) ? refetchInterval : false,
    refetchIntervalInBackground: false,
  })

  // Determine which data source to use
  const data = useMemo(() => {
    if (isConnected && realtimeEvents.length > 0) {
      // Use real-time events when SSE is connected and we have events
      return {
        success: true,
        data: realtimeEvents,
        total: realtimeEvents.length,
        clusterName,
        namespace,
      } as EventsResponse
    } else if (queryResult.data) {
      // Fallback to query result
      return queryResult.data
    } else {
      // Return empty state
      return {
        success: false,
        data: [],
        total: 0,
        clusterName,
        namespace,
      } as EventsResponse
    }
  }, [isConnected, realtimeEvents, queryResult.data, clusterName, namespace])

  // Determine loading state
  const isLoading = !isConnected && queryResult.isLoading

  // Determine error state
  const error = connectionError || queryResult.error

  return {
    data,
    isLoading,
    error,
    refetch: queryResult.refetch,
    isConnected,
    connectionError,
    isRealtime: isConnected && realtimeEvents.length > 0,
  }
}

// Specialized hook for cluster-scoped events
export function useClusterEvents(clusterName: string, options: Omit<UseEventsOptions, 'clusterName'> = {}) {
  return useEvents({ ...options, clusterName })
}

// Specialized hook for resource-scoped events  
export function useResourceEvents(
  resourceType: string, 
  resourceName: string, 
  namespace: string,
  options: Omit<UseEventsOptions, 'resourceType' | 'resourceName' | 'namespace'> = {}
) {
  return useEvents({ 
    ...options, 
    resourceType, 
    resourceName, 
    namespace 
  })
}