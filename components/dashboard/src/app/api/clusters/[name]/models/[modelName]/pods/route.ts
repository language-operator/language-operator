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
    // Get user's selected organization
    const { user, organization } = await getUserOrganization(request)
    
    const hasPermission = await requirePermission(user.id, organization.id, 'view')
    if (!hasPermission) {
      return NextResponse.json({ error: 'Insufficient permissions' }, { status: 403 })
    }

    const { name: clusterName, modelName } = await params

    console.log(`Fetching pods for model ${modelName} in cluster ${clusterName}, namespace ${organization.namespace}`)

    // Find the pods for this model
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

    // Transform pods into the expected format
    const transformedPods = podList.map(pod => ({
      name: pod.metadata?.name || '',
      status: pod.status?.phase || 'Unknown',
      isRunning: pod.status?.phase === 'Running',
      creationTimestamp: pod.metadata?.creationTimestamp || '',
      availableContainers: pod.spec?.containers?.map(container => ({
        name: container.name,
        image: container.image || '',
        isToolContainer: false
      })) || []
    }))

    // Find the recommended pod (most recent running pod, or most recent if none running)
    let recommendedPod = null
    if (transformedPods.length > 0) {
      const runningPods = transformedPods.filter(p => p.isRunning)
      if (runningPods.length > 0) {
        recommendedPod = runningPods.sort((a, b) => {
          const aTime = a.creationTimestamp ? new Date(a.creationTimestamp).getTime() : 0
          const bTime = b.creationTimestamp ? new Date(b.creationTimestamp).getTime() : 0
          return bTime - aTime
        })[0].name
      } else {
        recommendedPod = transformedPods.sort((a, b) => {
          const aTime = a.creationTimestamp ? new Date(a.creationTimestamp).getTime() : 0
          const bTime = b.creationTimestamp ? new Date(b.creationTimestamp).getTime() : 0
          return bTime - aTime
        })[0].name
      }
    }

    return NextResponse.json({
      data: transformedPods,
      recommendedPod,
      recommendedContainer: null, // Models typically don't have multiple containers
      deploymentMode: 'service',
      podType: 'model'
    })

  } catch (error) {
    console.error('Error fetching model pods:', error)
    
    return NextResponse.json(
      { 
        error: 'Failed to fetch model pods',
        message: error instanceof Error ? error.message : 'Unknown error'
      },
      { status: 500 }
    )
  }
}