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

    // Get optional cluster filter from query params
    const url = new URL(request.url)
    const clusterName = url.searchParams.get('cluster')

    console.log(`🔍 Starting model watch for organization ${organization.name}${clusterName ? ` (cluster: ${clusterName})` : ''}`)

    // Track watch state
    let watchCleanup: (() => void) | null = null
    let retryTimeout: NodeJS.Timeout | null = null

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

    // Send initial connection data
    sendEvent({
      cluster: clusterName,
    }, 'connection')

    const startWatch = async () => {
      // Don't retry if client has disconnected
      if (request.signal.aborted || !isActive()) {
        console.log('🛑 Client disconnected, not starting model watch')
        return
      }

      try {
        // Build label selector
        let labelSelector = `langop.io/organization-id=${organization.id}`
        if (clusterName) {
          labelSelector += `,langop.io/cluster=${clusterName}`
        }

        watchCleanup = await watchService.watchLanguageModels(
          {
            namespace: organization.namespace,
            labelSelector,
            timeoutSeconds: 300,
          },
          (event: WatchEvent) => {
            const clientEvent = {
              type: event.type,
              resource: 'model',
              data: event.object,
              timestamp: new Date().toISOString(),
              resourceVersion: event.resourceVersion,
              cluster: event.object?.metadata?.labels?.['langop.io/cluster'],
            }

            if (event.error) {
              clientEvent.data = { error: event.error }
            }

            console.log(`📡 Model watch event: ${event.type} - ${event.object?.metadata?.name || 'unknown'}`)
            sendEvent(clientEvent, 'resource-update')
          },
          (error: Error | null) => {
            console.error('Model watch error:', error)

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
              retryTimeout = setTimeout(startWatch, 15000)
            }
          }
        )
      } catch (error) {
        console.error('Failed to start model watch:', error)

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
        'X-Accel-Buffering': 'no',
      }
    })

  } catch (error) {
    console.error('Unhandled error in model watch route:', error)
    return new Response(
      JSON.stringify({ error: 'Internal server error' }),
      { status: 500, headers: { 'Content-Type': 'application/json' } }
    )
  }
}