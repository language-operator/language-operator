import { NextRequest } from 'next/server'
import { getServerSession } from 'next-auth'
import { authOptions } from '@/lib/auth'
import { db } from '@/lib/db'
import { requirePermission } from '@/lib/permissions'
import { getUserOrganization } from '@/lib/organization-context'
import { watchService, WatchEvent } from '@/lib/watch-service'
import { createSSEWatchStream } from '@/lib/sse-watch-helper'

export async function GET(request: NextRequest) {
  try {
    // Get user's selected organization
    const { user, organization, userRole } = await getUserOrganization(request)

    const hasPermission = await requirePermission(user.id, organization.id, 'view')
    if (!hasPermission) {
      return new Response('Insufficient permissions', { status: 403 })
    }

    // Get optional filters from query params
    const url = new URL(request.url)
    const clusterName = url.searchParams.get('cluster')
    const resourceType = url.searchParams.get('resourceType')
    const resourceName = url.searchParams.get('resourceName')

    console.log(`🔍 Starting events watch for organization ${organization.name}`, {
      cluster: clusterName,
      resourceType,
      resourceName
    })

    // Track watch state
    let watchCleanup: (() => void) | null = null
    let retryTimeout: NodeJS.Timeout | null = null
    let lastResourceVersion: string | undefined = undefined

    // Create safe SSE stream with automatic lifecycle management
    const { stream, sendEvent, isActive } = createSSEWatchStream(request, {
      heartbeatInterval: 30000,
      onCleanup: () => {
        // Cleanup watch and timers when client disconnects
        if (watchCleanup) {
          watchCleanup()
          watchCleanup = null
        }
        if (retryTimeout) {
          clearTimeout(retryTimeout)
          retryTimeout = null
        }
      }
    })

    // Send initial connection data
    sendEvent({
      cluster: clusterName,
      resourceType,
      resourceName,
    }, 'connection')

    const startWatch = async () => {
      // Don't retry if client has disconnected
      if (request.signal.aborted || !isActive()) {
        console.log('🛑 Client disconnected, not starting events watch')
        return
      }

      try {
        // Build field selector for filtering events
        const fieldSelectors: string[] = []

        // Filter by involved object if resource specified
        if (resourceName && resourceType) {
          const k8sKind = mapResourceTypeToK8sKind(resourceType)
          fieldSelectors.push(`involvedObject.kind=${k8sKind}`)
          fieldSelectors.push(`involvedObject.name=${resourceName}`)
        } else if (resourceType) {
          const k8sKind = mapResourceTypeToK8sKind(resourceType)
          fieldSelectors.push(`involvedObject.kind=${k8sKind}`)
        }

        const fieldSelector = fieldSelectors.length > 0 ? fieldSelectors.join(',') : undefined

        // Build label selector for organization filtering
        const labelSelector = `langop.io/organization-id=${organization.id}`

        watchCleanup = await watchService.watchEvents(
          {
            namespace: organization.namespace,
            labelSelector,
            fieldSelector,
            resourceVersion: lastResourceVersion,
            // No timeout - let the watch run indefinitely
            // K8s will close it eventually, but we will reconnect immediately
          },
          (event: WatchEvent) => {
            try {
              // Update last known resourceVersion for bookmarking
              if (event.resourceVersion) {
                lastResourceVersion = event.resourceVersion
              }

              // Filter and transform the event
              const k8sEvent = event.object
              if (!k8sEvent || !shouldIncludeEvent(k8sEvent, clusterName ?? undefined, resourceType ?? undefined, resourceName ?? undefined)) {
                return
              }

              const clientEvent: any = {
                type: mapK8sEventToClientEventType(k8sEvent),
                resource: 'event',
                data: transformEventData(k8sEvent, organization.namespace),
                timestamp: new Date().toISOString(),
                resourceVersion: event.resourceVersion,
                eventType: event.type, // ADDED, MODIFIED, DELETED
              }

              if (event.error) {
                clientEvent.data = { error: event.error }
              }

              console.log(`📡 Event watch: ${event.type} - ${k8sEvent?.involvedObject?.name || 'unknown'} (${k8sEvent?.reason})`)
              sendEvent(clientEvent, 'resource-update')
            } catch (error) {
              console.error('⚠️  Error processing event watch event:', error)
              // Don't let event processing errors kill the watch stream
            }
          },
          (error: Error | null) => {
            console.error('Events watch error:', error)

            // Send error to client if stream is still active
            if (isActive()) {
              sendEvent({
                type: 'error',
                message: error?.message || 'Watch stream ended unexpectedly',
                timestamp: new Date().toISOString()
              }, 'error')
            }

            // Retry only if client is still connected
            if (!request.signal.aborted && isActive()) {
              console.log('🔄 Reconnecting events watch immediately...')
              retryTimeout = setTimeout(startWatch, 100) // 100ms delay to prevent tight loop
            }
          }
        )
      } catch (error) {
        console.error('Failed to start events watch:', error)

        // Send error to client if stream is still active
        if (isActive()) {
          sendEvent({
            type: 'error',
            message: error instanceof Error ? error.message : 'Failed to start events watch',
            timestamp: new Date().toISOString()
          }, 'error')
        }
      }
    }

    startWatch()

    return new Response(stream, {
      headers: {
        'Content-Type': 'text/event-stream',
        'Cache-Control': 'no-cache, no-store, must-revalidate',
        'Connection': 'keep-alive',
        'Access-Control-Allow-Origin': '*',
        'Access-Control-Allow-Headers': 'Cache-Control',
        'Access-Control-Allow-Methods': 'GET',
        'X-Accel-Buffering': 'no',
      }
    })

  } catch (error) {
    console.error('Unhandled error in events watch route:', error)
    return new Response(
      JSON.stringify({ error: 'Internal server error' }),
      { status: 500, headers: { 'Content-Type': 'application/json' } }
    )
  }
}

// Helper function to map resource types to K8s kinds
function mapResourceTypeToK8sKind(resourceType: string): string {
  const mapping: Record<string, string> = {
    'agent': 'LanguageAgent',
    'model': 'LanguageModel',
    'tool': 'LanguageTool', 
    'persona': 'LanguagePersona',
    'cluster': 'LanguageCluster',
    'pod': 'Pod',
    'service': 'Service',
    'deployment': 'Deployment',
  }
  return mapping[resourceType.toLowerCase()] || resourceType
}

// Helper function to determine if event should be included
function shouldIncludeEvent(k8sEvent: any, clusterName?: string, resourceType?: string, resourceName?: string): boolean {
  const involvedObject = k8sEvent.involvedObject
  if (!involvedObject) return false

  // Always include Language Operator resources
  if (involvedObject.apiVersion === 'langop.io/v1alpha1') {
    const langOperatorTypes = ['LanguageAgent', 'LanguageModel', 'LanguageTool', 'LanguagePersona', 'LanguageCluster']
    if (!langOperatorTypes.includes(involvedObject.kind)) return false
  } else {
    // Include core K8s resources that are related to Language Operator
    const coreResourceTypes = ['Pod', 'Service', 'Deployment', 'ReplicaSet', 'Job', 'CronJob']
    if (!(involvedObject.apiVersion?.startsWith('v1') || involvedObject.apiVersion?.startsWith('apps/'))) {
      return false
    }
    if (!coreResourceTypes.includes(involvedObject.kind)) return false
    
    // Check if the resource has Language Operator labels
    const labels = k8sEvent.involvedObject?.labels || {}
    if (!(labels['langop.io/managed-by'] || labels['langop.io/organization-id'])) {
      return false
    }
  }

  // Apply cluster filtering if specified
  if (clusterName) {
    if (involvedObject.apiVersion === 'langop.io/v1alpha1') {
      const belongsToCluster = involvedObject.name === clusterName || 
                              k8sEvent.involvedObject?.labels?.['langop.io/cluster'] === clusterName
      if (!belongsToCluster) return false
    } else {
      const labels = k8sEvent.involvedObject?.labels || {}
      const belongsToCluster = labels['langop.io/cluster'] === clusterName ||
                              labels['langop.io/managed-by'] === clusterName
      if (!belongsToCluster) return false
    }
  }

  // Apply resource type filtering if specified
  if (resourceType) {
    const expectedKind = mapResourceTypeToK8sKind(resourceType)
    if (involvedObject.kind !== expectedKind) return false
  }

  // Apply resource name filtering if specified
  if (resourceName) {
    if (involvedObject.name !== resourceName) return false
  }

  return true
}

// Helper function to map K8s event type to client event type
function mapK8sEventToClientEventType(k8sEvent: any): string {
  const involvedObject = k8sEvent.involvedObject
  const mapping: Record<string, string> = {
    'LanguageAgent': 'agent',
    'LanguageModel': 'model',
    'LanguageTool': 'tool',
    'LanguagePersona': 'persona', 
    'LanguageCluster': 'cluster',
    'Pod': 'pod',
    'Service': 'service',
    'Deployment': 'deployment',
    'ReplicaSet': 'deployment',
    'Job': 'deployment',
    'CronJob': 'deployment',
  }
  return mapping[involvedObject.kind] || involvedObject.kind.toLowerCase()
}

// Helper function to transform K8s event data for client consumption
function transformEventData(k8sEvent: any, defaultNamespace: string) {
  const involvedObject = k8sEvent.involvedObject
  const timestamp = k8sEvent.lastTimestamp || k8sEvent.firstTimestamp || k8sEvent.metadata.creationTimestamp
  
  const resourceType = mapK8sEventToClientEventType(k8sEvent)
  const resourceName = involvedObject.name
  const action = getActionFromEvent(k8sEvent)
  const eventNamespace = involvedObject.namespace || defaultNamespace

  return {
    id: k8sEvent.metadata.uid || `${k8sEvent.metadata.name}-${timestamp}`,
    type: resourceType,
    action,
    resourceName,
    namespace: eventNamespace,
    message: formatEventMessage(resourceType, resourceName, action, k8sEvent),
    timestamp: new Date(timestamp).toISOString(),
    reason: k8sEvent.reason,
    source: k8sEvent.source?.component || 'kubernetes',
    eventType: k8sEvent.type || 'Normal',
    count: k8sEvent.count || 1,
    firstTimestamp: k8sEvent.firstTimestamp ? new Date(k8sEvent.firstTimestamp).toISOString() : undefined,
    lastTimestamp: k8sEvent.lastTimestamp ? new Date(k8sEvent.lastTimestamp).toISOString() : undefined,
    involvedObject: {
      kind: involvedObject.kind,
      name: involvedObject.name,
      namespace: involvedObject.namespace,
      uid: involvedObject.uid,
    }
  }
}

// Helper function to determine action from K8s event
function getActionFromEvent(event: any): string {
  const reason = event.reason?.toLowerCase() || ''
  const type = event.type?.toLowerCase() || ''
  
  // Map common K8s event reasons to user-friendly actions
  if (reason.includes('created') || reason.includes('successfulcreate')) return 'created'
  if (reason.includes('updated') || reason.includes('update')) return 'updated'
  if (reason.includes('scaled') || reason.includes('scaling')) return 'scaled'
  if (reason.includes('failed') || reason.includes('error') || type === 'warning') return 'failed'
  if (reason.includes('ready') || reason.includes('reconciled')) return 'ready'
  if (reason.includes('started')) return 'started'
  if (reason.includes('stopped') || reason.includes('killed')) return 'stopped'
  if (reason.includes('pulling')) return 'pulling'
  if (reason.includes('pulled')) return 'pulled'
  
  // Default mapping based on event type
  if (type === 'normal') return 'normal'
  if (type === 'warning') return 'warning'
  
  return reason || 'updated'
}

// Helper function to format event messages
function formatEventMessage(resourceType: string, resourceName: string, action: string, event: any): string {
  const capitalizedType = resourceType.charAt(0).toUpperCase() + resourceType.slice(1)
  
  // If the event has a specific message, use it for more context
  if (event.message && event.message.length < 100) {
    return event.message
  }
  
  // Otherwise, construct a user-friendly message
  switch (action) {
    case 'created':
      return `${capitalizedType} "${resourceName}" was created`
    case 'updated':
      return `${capitalizedType} "${resourceName}" was updated`
    case 'scaled':
      return `${capitalizedType} "${resourceName}" was scaled`
    case 'failed':
      return `${capitalizedType} "${resourceName}" failed`
    case 'ready':
      return `${capitalizedType} "${resourceName}" is ready`
    case 'started':
      return `${capitalizedType} "${resourceName}" started`
    case 'stopped':
      return `${capitalizedType} "${resourceName}" stopped`
    case 'warning':
      return `${capitalizedType} "${resourceName}" warning: ${event.reason}`
    default:
      return `${capitalizedType} "${resourceName}" ${action}`
  }
}