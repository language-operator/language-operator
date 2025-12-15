import { NextRequest } from 'next/server'
import { getServerSession } from 'next-auth'
import { authOptions } from '@/lib/auth'
import { db } from '@/lib/db'
import { requirePermission } from '@/lib/permissions'
import { watchService, WatchEvent } from '@/lib/watch-service'

export async function GET(request: NextRequest) {
  try {
    const session = await getServerSession(authOptions)
    if (!session?.user?.email) {
      return new Response('Unauthorized', { status: 401 })
    }

    const user = await db.user.findUnique({
      where: { email: session.user.email },
      include: { memberships: { include: { organization: true } } },
    })

    if (!user || user.memberships.length === 0) {
      return new Response('No organization found', { status: 404 })
    }

    const organization = user.memberships[0].organization
    
    const hasPermission = await requirePermission(user.id, organization.id, 'view')
    if (!hasPermission) {
      return new Response('Insufficient permissions', { status: 403 })
    }

    console.log(`🔍 Starting personas watch for organization ${organization.name}`)

    const stream = new ReadableStream({
      start(controller) {
        const encoder = new TextEncoder()
        let watchCleanup: (() => void) | null = null
        
        const sendEvent = (data: any, event?: string) => {
          const eventData = `${event ? `event: ${event}\n` : ''}data: ${JSON.stringify(data)}\n\n`
          controller.enqueue(encoder.encode(eventData))
        }

        sendEvent({ type: 'connected', timestamp: new Date().toISOString() }, 'connection')

        const startWatch = async () => {
          try {
            watchCleanup = await watchService.watchLanguagePersonas(
              {
                namespace: organization.namespace,
                labelSelector: `langop.io/organization-id=${organization.id}`,
                timeoutSeconds: 300,
              },
              (event: WatchEvent) => {
                const clientEvent = {
                  type: event.type,
                  resource: 'persona',
                  data: event.object,
                  timestamp: new Date().toISOString(),
                  resourceVersion: event.resourceVersion,
                }

                if (event.error) {
                  clientEvent.data = { error: event.error }
                }

                console.log(`📡 Persona watch event: ${event.type} - ${event.object?.metadata?.name || 'unknown'}`)
                sendEvent(clientEvent, 'resource-update')
              },
              (error: Error) => {
                console.error('Persona watch error:', error)
                sendEvent({
                  type: 'error',
                  message: error.message,
                  timestamp: new Date().toISOString()
                }, 'error')
                setTimeout(startWatch, 15000)
              }
            )
          } catch (error) {
            console.error('Failed to start persona watch:', error)
            sendEvent({
              type: 'error',
              message: error instanceof Error ? error.message : 'Failed to start watch',
              timestamp: new Date().toISOString()
            }, 'error')
          }
        }

        startWatch()

        request.signal.addEventListener('abort', () => {
          console.log('🛑 Persona watch client disconnected')
          if (watchCleanup) {
            watchCleanup()
          }
          controller.close()
        })

        const heartbeatInterval = setInterval(() => {
          try {
            sendEvent({
              type: 'heartbeat',
              timestamp: new Date().toISOString(),
              activeWatches: watchService.getActiveWatchCount()
            }, 'heartbeat')
          } catch (error) {
            clearInterval(heartbeatInterval)
            if (watchCleanup) {
              watchCleanup()
            }
          }
        }, 30000)

        const cleanup = () => {
          clearInterval(heartbeatInterval)
          if (watchCleanup) {
            watchCleanup()
          }
        }

        ;(controller as any).cleanup = cleanup
      },
      cancel() {
        console.log('🛑 Persona watch stream cancelled')
        if ((this as any).cleanup) {
          (this as any).cleanup()
        }
      }
    })

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
    console.error('Error setting up persona watch:', error)
    return new Response(
      `Error setting up persona watch: ${error instanceof Error ? error.message : 'Unknown error'}`,
      { status: 500 }
    )
  }
}