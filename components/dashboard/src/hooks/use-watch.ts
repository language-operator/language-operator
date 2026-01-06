import { useEffect, useRef, useState, useCallback, useMemo } from 'react'
import { useQueryClient } from '@tanstack/react-query'

export interface WatchEvent {
  type: 'ADDED' | 'MODIFIED' | 'DELETED' | 'ERROR'
  resource: string
  data: any
  timestamp: string
  resourceVersion?: string
  cluster?: string
}

export interface WatchConnection {
  type: 'connected' | 'heartbeat' | 'error'
  timestamp: string
  activeWatches?: number
}

export interface UseWatchOptions {
  enabled?: boolean
  onEvent?: (event: WatchEvent) => void
  onConnection?: (connection: WatchConnection) => void
  onError?: (error: Error) => void
  queryKey?: string[]
  cluster?: string // For filtering agents by cluster
}

let watchInstanceCounter = 0

export function useWatch(endpoint: string, options: UseWatchOptions = {}) {
  const {
    enabled = true,
    onEvent,
    onConnection,
    onError,
    queryKey,
    cluster
  } = options

  // Create unique instance ID for debugging
  const instanceId = useMemo(() => ++watchInstanceCounter, [])
  const [isConnected, setIsConnected] = useState(false)
  const [lastEvent, setLastEvent] = useState<WatchEvent | null>(null)
  const [connectionError, setConnectionError] = useState<string | null>(null)
  const [reconnectCount, setReconnectCount] = useState(0)
  
  const eventSourceRef = useRef<EventSource | null>(null)
  const queryClient = useQueryClient()
  const reconnectTimeoutRef = useRef<NodeJS.Timeout | null>(null)

  // Use refs for callbacks to avoid recreating connect() function
  const onEventRef = useRef(onEvent)
  const onConnectionRef = useRef(onConnection)
  const onErrorRef = useRef(onError)
  const queryKeyRef = useRef(queryKey)

  // Update refs when callbacks change
  useEffect(() => {
    onEventRef.current = onEvent
    onConnectionRef.current = onConnection
    onErrorRef.current = onError
    queryKeyRef.current = queryKey
  })

  // Build URL with query parameters
  const buildUrl = useCallback(() => {
    const url = new URL(endpoint, window.location.origin)
    if (cluster) {
      url.searchParams.set('cluster', cluster)
    }
    return url.toString()
  }, [endpoint, cluster])

  const cleanup = useCallback(() => {
    if (eventSourceRef.current) {
      eventSourceRef.current.close()
      eventSourceRef.current = null
    }
    if (reconnectTimeoutRef.current) {
      clearTimeout(reconnectTimeoutRef.current)
      reconnectTimeoutRef.current = null
    }
    setIsConnected(false)
  }, [])

  const connect = useCallback(() => {
    if (!enabled) return

    // Prevent duplicate connections
    if (eventSourceRef.current?.readyState === EventSource.OPEN) {
      console.log(`⚠️ [${instanceId}] EventSource already connected, skipping duplicate connection`)
      return
    }

    cleanup()

    const url = buildUrl()
    console.log(`🔗 [${instanceId}] Connecting to watch stream: ${url}`)

    try {
      const eventSource = new EventSource(url, {
        withCredentials: true
      })
      eventSourceRef.current = eventSource

      eventSource.addEventListener('connection', (event) => {
        const connectionData: WatchConnection = JSON.parse(event.data)
        console.log('📡 Watch connected:', connectionData)
        setIsConnected(true)
        setConnectionError(null)
        setReconnectCount(0)

        if (onConnectionRef.current) {
          onConnectionRef.current(connectionData)
        }
      })

      eventSource.addEventListener('resource-update', (event) => {
        try {
          const watchEvent: WatchEvent = JSON.parse(event.data)
          console.log(`📡 Resource update: ${watchEvent.type} ${watchEvent.resource} - ${watchEvent.data?.metadata?.name}`)
          
          setLastEvent(watchEvent)
          
          // Clear connection error only if we're connected (not during reconnection attempts)
          setIsConnected(true)
          setConnectionError(null)
          setReconnectCount(0)
          
          // Invalidate relevant React Query cache
          if (queryKeyRef.current) {
            console.log(`🔄 Invalidating queries with key:`, queryKeyRef.current)
            queryClient.invalidateQueries({
              queryKey: queryKeyRef.current,
              exact: false, // Match all queries starting with this prefix
              refetchType: 'active' // Refetch active queries immediately
            })
          }

          // Call custom event handler
          if (onEventRef.current) {
            onEventRef.current(watchEvent)
          }
        } catch (error) {
          console.error('Failed to parse watch event:', error)
        }
      })

      eventSource.addEventListener('heartbeat', (event) => {
        const heartbeatData = JSON.parse(event.data)
        
        // Clear connection error and confirm connected state since heartbeat indicates healthy connection
        setIsConnected(true)
        setConnectionError(null)
        setReconnectCount(0)

        if (onConnectionRef.current) {
          onConnectionRef.current(heartbeatData)
        }
      })

      eventSource.addEventListener('error', (event) => {
        try {
          const errorData = JSON.parse((event as any).data || '{}')
          console.error('Watch error event:', errorData)
          setConnectionError(errorData.message || 'Watch error')

          if (onErrorRef.current) {
            onErrorRef.current(new Error(errorData.message || 'Watch error'))
          }
        } catch (err) {
          // Error events may not have parseable data
          console.error('Watch error event (no data):', event)
          setConnectionError('Watch connection error')

          if (onErrorRef.current) {
            onErrorRef.current(new Error('Watch connection error'))
          }
        }
      })

      eventSource.onerror = (error) => {
        const readyState = eventSource.readyState;
        const readyStateNames = ['CONNECTING', 'OPEN', 'CLOSED'];
        console.error(`EventSource error (readyState: ${readyStateNames[readyState] || readyState}):`, {
          error,
          url: eventSource.url,
          timestamp: new Date().toISOString(),
          instanceId
        })
        setIsConnected(false)

        const errorMessage = 'Connection lost. Attempting to reconnect...'
        setConnectionError(errorMessage)

        if (onErrorRef.current) {
          onErrorRef.current(new Error(errorMessage))
        }

        // Attempt reconnect with exponential backoff, max 5 attempts
        if (reconnectCount < 5) {
          const delay = Math.min(5000 * Math.pow(1.5, reconnectCount), 60000) // Start at 5s, max 60 seconds
          console.log(`🔄 Reconnecting in ${delay}ms (attempt ${reconnectCount + 1})`)

          setReconnectCount(prev => prev + 1)
          reconnectTimeoutRef.current = setTimeout(() => {
            if (enabled) {
              connect()
            }
          }, delay)
        } else {
          console.error('❌ Max reconnection attempts reached, stopping reconnects')
          setConnectionError('Connection failed after multiple attempts. Refresh page to retry.')
        }
      }

      eventSource.onopen = () => {
        console.log('📡 Watch stream opened')
      }

    } catch (error) {
      console.error('Failed to create EventSource:', error)
      setConnectionError('Failed to establish connection')

      if (onErrorRef.current) {
        onErrorRef.current(error instanceof Error ? error : new Error('Connection failed'))
      }
    }
  }, [enabled, buildUrl, instanceId, cleanup]) // Stable dependencies only

  // Start/stop watching based on enabled flag
  useEffect(() => {
    if (enabled) {
      connect()
    } else {
      cleanup()
    }

    // Cleanup on unmount
    return cleanup
  }, [enabled, connect, cleanup]) // Added connect and cleanup back to match their definitions

  // Handle page visibility changes to reconnect when page becomes visible
  useEffect(() => {
    const handleVisibilityChange = () => {
      if (!document.hidden && enabled && !isConnected) {
        console.log('🔄 Page visible, reconnecting watch...')
        connect()
      }
    }

    document.addEventListener('visibilitychange', handleVisibilityChange)
    return () => document.removeEventListener('visibilitychange', handleVisibilityChange)
  }, [enabled, isConnected, connect])

  const reconnect = useCallback(() => {
    setReconnectCount(0)
    connect()
  }, [connect])

  return {
    isConnected,
    lastEvent,
    connectionError,
    reconnectCount,
    reconnect,
    cleanup
  }
}

// Resource-specific watch hooks

export function useWatchClusters(options: Omit<UseWatchOptions, 'queryKey'> = {}) {
  return useWatch('/api/watch/clusters', {
    ...options,
    queryKey: ['clusters']
  })
}

export function useWatchAgents(options: Omit<UseWatchOptions, 'queryKey'> & { clusterName?: string } = {}) {
  const { clusterName, ...watchOptions } = options
  return useWatch('/api/watch/agents', {
    ...watchOptions,
    queryKey: ['agents', clusterName || ''],
    cluster: clusterName || undefined
  })
}

export function useWatchModels(options: Omit<UseWatchOptions, 'queryKey'> = {}) {
  return useWatch('/api/watch/models', {
    ...options,
    queryKey: ['models']
  })
}

export function useWatchTools(options: Omit<UseWatchOptions, 'queryKey'> = {}) {
  return useWatch('/api/watch/tools', {
    ...options,
    queryKey: ['tools']
  })
}

export function useWatchPersonas(options: Omit<UseWatchOptions, 'queryKey'> = {}) {
  return useWatch('/api/watch/personas', {
    ...options,
    queryKey: ['personas']
  })
}

export function useWatchEvents(options: Omit<UseWatchOptions, 'queryKey'> & { 
  clusterName?: string
  resourceType?: string
  resourceName?: string
  namespace?: string
} = {}) {
  const { clusterName, resourceType, resourceName, namespace, ...watchOptions } = options
  
  // Build query parameters for the watch endpoint
  const buildEndpoint = () => {
    // Only build endpoint on client side to avoid SSR issues
    if (typeof window === 'undefined') {
      return '/api/watch/events'
    }
    
    const url = new URL('/api/watch/events', window.location.origin)
    if (clusterName) url.searchParams.set('cluster', clusterName)
    if (resourceType) url.searchParams.set('resourceType', resourceType)
    if (resourceName) url.searchParams.set('resourceName', resourceName)
    if (namespace) url.searchParams.set('namespace', namespace)
    return url.pathname + url.search
  }
  
  return useWatch(buildEndpoint(), {
    ...watchOptions,
    queryKey: ['events', clusterName ?? '', resourceType ?? '', resourceName ?? '', namespace ?? '']
  })
}