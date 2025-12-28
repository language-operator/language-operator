import { NextRequest, NextResponse } from 'next/server'
import { k8sClient } from '@/lib/k8s-client'
import { requirePermission } from '@/lib/permissions'
import { getUserOrganization } from '@/lib/organization-context'
import type { V1Pod } from '@kubernetes/client-node'

interface RouteParams {
  params: Promise<{
    name: string
    modelName: string
  }>
}

export async function GET(request: NextRequest, { params }: RouteParams) {
  try {
    // Get user's selected organization (replaces broken memberships[0] pattern)
    const { user, organization } = await getUserOrganization(request)
    
    const hasPermission = await requirePermission(user.id, organization.id, 'view')
    if (!hasPermission) {
      return NextResponse.json({ error: 'Insufficient permissions' }, { status: 403 })
    }

    const { name: clusterName, modelName } = await params
    const searchParams = new URL(request.url).searchParams
    const podName = searchParams.get('podName')

    console.log(`Fetching logs for model ${modelName} in cluster ${clusterName}, namespace ${organization.namespace}${podName ? `, pod ${podName}` : ''}`)

    // Find the pod for this model
    const pods = await k8sClient.listPods(organization.namespace, {
      labelSelector: `app.kubernetes.io/name=${modelName}`
    })

    // Handle different response structures from k8s client
    let podList: V1Pod[] = []
    if ((pods as any)?.body?.items) {
      podList = (pods as any).body.items
    } else if ((pods as any)?.data?.items) {
      podList = (pods as any).data.items
    } else if (Array.isArray(pods)) {
      podList = pods
    } else if ((pods as any)?.items) {
      podList = (pods as any).items
    }

    console.log(`Found ${podList.length} pods for model ${modelName}`)

    if (podList.length === 0) {
      return NextResponse.json({
        logs: 'No pods found for this model.',
        message: 'Model has no running pods'
      })
    }

    // Select the appropriate pod
    let pod: V1Pod
    if (podName) {
      // Find the specific pod requested
      const foundPod = podList.find(p => p.metadata?.name === podName)
      if (!foundPod) {
        return NextResponse.json({
          error: `Pod "${podName}" not found`,
          message: `Pod "${podName}" not found for model ${modelName}`
        }, { status: 404 })
      }
      pod = foundPod
    } else {
      // Default behavior: get the most recent running pod, or most recent if none running
      const runningPods = podList.filter(p => p.status?.phase === 'Running')
      if (runningPods.length > 0) {
        pod = runningPods.sort((a, b) => {
          const aTime = a.metadata?.creationTimestamp ? new Date(a.metadata.creationTimestamp).getTime() : 0
          const bTime = b.metadata?.creationTimestamp ? new Date(b.metadata.creationTimestamp).getTime() : 0
          return bTime - aTime
        })[0]
      } else {
        pod = podList.sort((a, b) => {
          const aTime = a.metadata?.creationTimestamp ? new Date(a.metadata.creationTimestamp).getTime() : 0
          const bTime = b.metadata?.creationTimestamp ? new Date(b.metadata.creationTimestamp).getTime() : 0
          return bTime - aTime
        })[0]
      }
    }

    console.log(`Getting logs from pod: ${pod.metadata?.name}`)

    // Fetch logs from the pod
    const logs = await k8sClient.getPodLogs(organization.namespace, pod.metadata?.name || '', {
      tailLines: 500, // Get last 500 lines
      timestamps: true
    })

    // Handle different response structures from k8s client
    let logContent = ''
    if (typeof logs === 'string') {
      logContent = logs
    } else if ((logs as { body?: string })?.body) {
      logContent = (logs as { body: string }).body
    } else if ((logs as { data?: string })?.data) {
      logContent = (logs as { data: string }).data
    }

    return NextResponse.json({
      logs: logContent || 'No logs available',
      podName: pod.metadata?.name || '',
      message: 'Logs retrieved successfully'
    })

  } catch (error) {
    console.error('Error fetching model logs:', error)
    
    return NextResponse.json(
      { 
        error: 'Failed to fetch model logs',
        message: error instanceof Error ? error.message : 'Unknown error'
      },
      { status: 500 }
    )
  }
}