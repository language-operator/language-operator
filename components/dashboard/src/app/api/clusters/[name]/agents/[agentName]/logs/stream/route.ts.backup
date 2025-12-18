import { NextRequest } from 'next/server'
import { getServerSession } from 'next-auth'
import { authOptions } from '@/lib/auth'
import { k8sClient } from '@/lib/k8s-client'
import { db } from '@/lib/db'
import { requirePermission } from '@/lib/permissions'

interface RouteParams {
  params: Promise<{
    name: string
    agentName: string
  }>
}

export async function GET(request: NextRequest, { params }: RouteParams) {
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

    const { name: clusterName, agentName } = await params
    const url = new URL(request.url)
    const podName = url.searchParams.get('podName')

    console.log(`Starting log stream for agent ${agentName} in cluster ${clusterName}${podName ? `, pod ${podName}` : ''}`)

    // Find the pod for this agent
    const pods = await k8sClient.listPods(organization.namespace, {
      labelSelector: `app.kubernetes.io/name=${agentName}`
    })

    // Handle different response structures from k8s client
    let podList: any[] = []
    if ((pods as any)?.body?.items) {
      podList = (pods as any).body.items
    } else if ((pods as any)?.data?.items) {
      podList = (pods as any).data.items
    } else if (Array.isArray(pods)) {
      podList = pods
    } else if ((pods as any)?.items) {
      podList = (pods as any).items
    }

    if (podList.length === 0) {
      return new Response('No pods found for this agent', { status: 404 })
    }

    // Select the appropriate pod
    let pod
    if (podName) {
      // Find the specific pod requested
      pod = podList.find(p => p.metadata.name === podName)
      if (!pod) {
        return new Response(`Pod "${podName}" not found for agent ${agentName}`, { status: 404 })
      }
    } else {
      // Default behavior: get the most recent running pod, or most recent if none running
      const runningPods = podList.filter(p => p.status?.phase === 'Running')
      if (runningPods.length > 0) {
        pod = runningPods.sort((a, b) => 
          new Date(b.metadata.creationTimestamp).getTime() - 
          new Date(a.metadata.creationTimestamp).getTime()
        )[0]
      } else {
        pod = podList.sort((a, b) => 
          new Date(b.metadata.creationTimestamp).getTime() - 
          new Date(a.metadata.creationTimestamp).getTime()
        )[0]
      }
    }

    console.log(`Streaming logs from pod: ${pod.metadata.name}`)

    // Create a readable stream for Server-Sent Events
    const stream = new ReadableStream({
      start(controller) {
        // Send headers for SSE
        const encoder = new TextEncoder()
        
        // Start streaming logs
        const streamLogs = async () => {
          try {
            const logStream = await k8sClient.streamPodLogs(organization.namespace, pod.metadata.name, {
              follow: true,
              timestamps: true,
              tailLines: 10 // Start with last 10 lines
            })

            // Handle the response - kubernetes client returns a string
            if (typeof logStream === 'string') {
              const lines = logStream.split('\n').filter(line => line.trim())
              lines.forEach(line => {
                if (line.trim()) {
                  const sseData = `data: ${line}\n\n`
                  controller.enqueue(encoder.encode(sseData))
                }
              })
              controller.close()
            } else {
              // Fallback: poll for logs periodically
              const pollLogs = async () => {
                try {
                  const logs = await k8sClient.getPodLogs(organization.namespace, pod.metadata.name, {
                    tailLines: 1,
                    timestamps: true,
                    sinceSeconds: 1
                  })

                  let logContent = ''
                  if (typeof logs === 'string') {
                    logContent = logs
                  } else if ((logs as any)?.body) {
                    logContent = (logs as any).body
                  } else if ((logs as any)?.data) {
                    logContent = (logs as any).data
                  }

                  if (logContent && logContent.trim()) {
                    const lines = logContent.split('\\n').filter(line => line.trim())
                    lines.forEach(line => {
                      if (line.trim()) {
                        const sseData = `data: ${line}\\n\\n`
                        controller.enqueue(encoder.encode(sseData))
                      }
                    })
                  }
                } catch (error) {
                  console.error('Polling error:', error)
                }
              }

              // Poll every 2 seconds
              const interval = setInterval(pollLogs, 2000)
              
              // Cleanup on stream close
              const cleanup = () => {
                clearInterval(interval)
                controller.close()
              }

              // Handle client disconnect
              request.signal.addEventListener('abort', cleanup)
            }
          } catch (error) {
            console.error('Error starting log stream:', error)
            const sseError = `data: [ERROR] Failed to start log stream: ${error}\\n\\n`
            controller.enqueue(encoder.encode(sseError))
            controller.close()
          }
        }

        streamLogs()
      },
      cancel() {
        console.log('Stream cancelled')
      }
    })

    return new Response(stream, {
      headers: {
        'Content-Type': 'text/event-stream',
        'Cache-Control': 'no-cache',
        'Connection': 'keep-alive',
        'Access-Control-Allow-Origin': '*',
        'Access-Control-Allow-Headers': 'Cache-Control'
      }
    })

  } catch (error) {
    console.error('Error setting up log stream:', error)
    return new Response(
      `Error setting up log stream: ${error instanceof Error ? error.message : 'Unknown error'}`,
      { status: 500 }
    )
  }
}