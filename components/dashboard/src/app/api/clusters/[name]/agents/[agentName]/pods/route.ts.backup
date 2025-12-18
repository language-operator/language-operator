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

    console.log(`Fetching pods for agent ${agentName} in cluster ${clusterName}, namespace ${organization.namespace}`)

    // Find all pods for this agent
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

    // Transform pods into a more user-friendly format
    const transformedPods = podList.map((pod) => {
      const status = pod.status?.phase || 'Unknown'
      const creationTimestamp = pod.metadata?.creationTimestamp
      const name = pod.metadata?.name || 'unknown'
      
      // Determine if this is a running pod
      const isRunning = status === 'Running'
      
      // Get container statuses for more detailed info
      const containerStatuses = pod.status?.containerStatuses || []
      const hasRunningContainers = containerStatuses.some((c: any) => c.state?.running)
      
      return {
        name,
        status,
        creationTimestamp,
        isRunning,
        hasRunningContainers,
        // Additional metadata that might be useful
        labels: pod.metadata?.labels || {},
        restartCount: containerStatuses.reduce((sum: number, c: any) => sum + (c.restartCount || 0), 0),
      }
    })

    // Sort pods by creation timestamp (newest first)
    const sortedPods = transformedPods.sort((a, b) => 
      new Date(b.creationTimestamp).getTime() - new Date(a.creationTimestamp).getTime()
    )

    // Determine the recommended pod (first running pod, or most recent if none running)
    const runningPods = sortedPods.filter(p => p.isRunning && p.hasRunningContainers)
    const recommendedPod = runningPods.length > 0 ? runningPods[0] : sortedPods[0]

    return NextResponse.json({
      data: sortedPods,
      recommendedPod: recommendedPod?.name || null,
      totalCount: sortedPods.length,
      runningCount: runningPods.length
    })

  } catch (error) {
    console.error('Error fetching agent pods:', error)
    
    return NextResponse.json(
      { 
        error: 'Failed to fetch agent pods',
        message: error instanceof Error ? error.message : 'Unknown error'
      },
      { status: 500 }
    )
  }
}