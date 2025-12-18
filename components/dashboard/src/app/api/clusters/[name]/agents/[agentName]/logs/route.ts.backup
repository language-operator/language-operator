import { NextRequest, NextResponse } from 'next/server'
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
      return NextResponse.json({ error: 'Unauthorized' }, { status: 401 })
    }

    const user = await db.user.findUnique({
      where: { email: session.user.email },
      include: { memberships: { include: { organization: true } } },
    })

    if (!user || user.memberships.length === 0) {
      return NextResponse.json({ error: 'No organization found' }, { status: 404 })
    }

    const organization = user.memberships[0].organization
    
    const hasPermission = await requirePermission(user.id, organization.id, 'view')
    if (!hasPermission) {
      return NextResponse.json({ error: 'Insufficient permissions' }, { status: 403 })
    }

    const { name: clusterName, agentName } = await params
    const searchParams = new URL(request.url).searchParams
    const podName = searchParams.get('podName')

    console.log(`Fetching logs for agent ${agentName} in cluster ${clusterName}, namespace ${organization.namespace}${podName ? `, pod ${podName}` : ''}`)

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

    console.log(`Found ${podList.length} pods for agent ${agentName}`)

    if (podList.length === 0) {
      return NextResponse.json({
        logs: 'No pods found for this agent.',
        message: 'Agent has no running pods'
      })
    }

    // Select the appropriate pod
    let pod
    if (podName) {
      // Find the specific pod requested
      pod = podList.find(p => p.metadata.name === podName)
      if (!pod) {
        return NextResponse.json({
          error: `Pod "${podName}" not found`,
          message: `Pod "${podName}" not found for agent ${agentName}`
        }, { status: 404 })
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

    console.log(`Getting logs from pod: ${pod.metadata.name}`)

    // Fetch logs from the pod
    const logs = await k8sClient.getPodLogs(organization.namespace, pod.metadata.name, {
      tailLines: 500, // Get last 500 lines
      timestamps: true
    })

    // Handle different response structures from k8s client
    let logContent = ''
    if (typeof logs === 'string') {
      logContent = logs
    } else if ((logs as any)?.body) {
      logContent = (logs as any).body
    } else if ((logs as any)?.data) {
      logContent = (logs as any).data
    }

    return NextResponse.json({
      logs: logContent || 'No logs available',
      podName: pod.metadata.name,
      message: 'Logs retrieved successfully'
    })

  } catch (error) {
    console.error('Error fetching agent logs:', error)
    
    return NextResponse.json(
      { 
        error: 'Failed to fetch agent logs',
        message: error instanceof Error ? error.message : 'Unknown error'
      },
      { status: 500 }
    )
  }
}