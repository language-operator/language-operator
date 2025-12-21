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

    console.log(`🔍 Starting cluster watch for organization ${organization.name}`)

    // Track watch state
    let watchCleanup: (() => void) | null = null
    let retryTimeout: NodeJS.Timeout | null = null
    let lastResourceVersion: string | undefined = undefined

    // Create safe SSE stream with automatic lifecycle management
    const { stream, sendEvent, isActive } = createSSEWatchStream(request, {
      heartbeatInterval: 30000,
      onCleanup: () => {
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

    const startWatch = async () => {
      // Don't retry if client has disconnected
      if (request.signal.aborted || !isActive()) {
        console.log('🛑 Client disconnected, not starting cluster watch')
        return
      }

      try {
        watchCleanup = await watchService.watchLanguageClusters(
          {
            namespace: organization.namespace,
            labelSelector: `langop.io/organization-id=${organization.id}`,
            resourceVersion: lastResourceVersion,
            // No timeout - let the watch run indefinitely
            // K8s will close it eventually, but we'll reconnect immediately
          },
          (event: WatchEvent) => {
            try {
              // Update last known resourceVersion for bookmarking
              if (event.resourceVersion) {
                lastResourceVersion = event.resourceVersion
              }

              const clientEvent = {
                type: event.type,
                resource: 'cluster',
                data: event.object,
                timestamp: new Date().toISOString(),
                resourceVersion: event.resourceVersion,
              }

              if (event.error) {
                clientEvent.data = { error: event.error }
              }

              console.log(`📡 Cluster watch event: ${event.type} - ${event.object?.metadata?.name || 'unknown'}`)
              sendEvent(clientEvent, 'resource-update')
            } catch (error) {
              console.error('⚠️  Error processing cluster watch event:', error)
              // Don't let event processing errors kill the watch stream
            }
          },
          (error: Error | null) => {
            console.error('Cluster watch error:', error)

            // Send error to client if stream is still active
            if (isActive()) {
              sendEvent({
                type: 'error',
                message: error?.message || 'Watch stream ended unexpectedly',
                timestamp: new Date().toISOString()
              }, 'error')
            }

            // Retry immediately if client is still connected
            // This prevents missing events during reconnection
            if (!request.signal.aborted && isActive()) {
              console.log('🔄 Reconnecting cluster watch immediately...')
              retryTimeout = setTimeout(startWatch, 100) // 100ms delay to prevent tight loop
            }
          }
        )
      } catch (error) {
        console.error('Failed to start cluster watch:', error)

        // Send error to client if stream is still active
        if (isActive()) {
          sendEvent({
            type: 'error',
            message: error instanceof Error ? error.message : 'Failed to start watch',
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
        'X-Accel-Buffering': 'no', // Disable nginx buffering
      }
    })

  } catch (error) {
    console.error('Unhandled error in cluster watch route:', error)
    return new Response(
      JSON.stringify({ error: 'Internal server error' }),
      { status: 500, headers: { 'Content-Type': 'application/json' } }
    )
  }
}