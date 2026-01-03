import { NextRequest, NextResponse } from 'next/server'
import { getServerSession } from 'next-auth'
import { authOptions } from '@/lib/auth'
import { k8sClient } from '@/lib/k8s-client'
import { db } from '@/lib/db'
import { requirePermission } from '@/lib/permissions'
import { getUserOrganization } from '@/lib/organization-context'

interface RouteParams {
  params: Promise<{
    name: string
    toolName: string
  }>
}

export async function GET(request: NextRequest, { params }: RouteParams) {
  try {
    // Get user's selected organization
    const { user, organization, userRole } = await getUserOrganization(request)
    
    const hasPermission = await requirePermission(user.id, organization.id, 'view')
    if (!hasPermission) {
      return NextResponse.json({ error: 'Insufficient permissions' }, { status: 403 })
    }

    const { name: clusterName, toolName } = await params
    const searchParams = new URL(request.url).searchParams
    const podName = searchParams.get('podName')
    const containerName = searchParams.get('containerName')

    console.log(`Fetching logs for tool ${toolName} in cluster ${clusterName}, namespace ${organization.namespace}${podName ? `, pod ${podName}` : ''}${containerName ? `, container ${containerName}` : ''}`)

    // First, get the tool to understand its deployment mode
    const toolResource = await k8sClient.getLanguageTool(organization.namespace, toolName)
    
    let toolData: any
    if ((toolResource as any)?.body) {
      toolData = (toolResource as any).body
    } else if ((toolResource as any)?.data) {
      toolData = (toolResource as any).data
    } else {
      toolData = toolResource
    }

    if (!toolData) {
      return NextResponse.json({ error: 'Tool not found' }, { status: 404 })
    }

    const deploymentMode = toolData.spec?.deploymentMode || 'service'

    // Find the appropriate pods based on deployment mode
    let labelSelector = ''
    if (deploymentMode === 'sidecar') {
      // For sidecar tools, find agent pods that use this tool
      labelSelector = 'langop.io/kind=LanguageAgent'
    } else {
      // For service tools, find the tool's own deployment pods
      labelSelector = `app.kubernetes.io/name=${toolName}`
    }

    const pods = await k8sClient.listPods(organization.namespace, {
      labelSelector
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

    // For sidecar mode, filter to only pods that actually have this tool
    if (deploymentMode === 'sidecar') {
      podList = podList.filter(pod => {
        const containers = pod.spec?.containers || []
        const initContainers = pod.spec?.initContainers || []
        const allContainers = [...containers, ...initContainers]
        return allContainers.some((c: any) => c.name?.includes(toolName) || c.image?.includes(toolName))
      })
    }

    console.log(`Found ${podList.length} pods for ${deploymentMode} tool ${toolName}`)

    if (podList.length === 0) {
      return NextResponse.json({
        logs: deploymentMode === 'sidecar' 
          ? 'No agent pods found using this sidecar tool.'
          : 'No pods found for this service tool.',
        message: `Tool has no running pods in ${deploymentMode} mode`
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
          message: `Pod "${podName}" not found for tool ${toolName}`
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

    // Determine which container to get logs from
    let targetContainer = containerName
    if (!targetContainer) {
      // Auto-select the appropriate container
      const containers = pod.spec?.containers || []
      const initContainers = pod.spec?.initContainers || []
      
      
      if (deploymentMode === 'sidecar') {
        // For sidecar mode, prefer the tool container (check both regular and init containers)
        const allContainers = [...containers, ...initContainers]
        
        // Try different matching strategies for tool containers
        let toolContainer = allContainers.find((c: any) => 
          c.name?.includes(toolName) || c.image?.includes(toolName)
        )
        
        // If not found with exact tool name, try common sidecar container naming patterns
        if (!toolContainer) {
          toolContainer = allContainers.find((c: any) => 
            c.name?.includes(`tool-${toolName}`) || 
            c.name?.startsWith('tool-') ||
            c.name?.endsWith(`-${toolName}`)
          )
        }
        
        
        targetContainer = toolContainer?.name || containers[0]?.name
      } else {
        // For service mode, prefer the main container (usually the first one)
        targetContainer = containers[0]?.name
      }
    }

    console.log(`Getting logs from container: ${targetContainer}`)

    // Fetch logs from the pod/container
    const logOptions: any = {
      tailLines: 500, // Get last 500 lines
      timestamps: true
    }

    // Add container name if specified
    if (targetContainer) {
      logOptions.container = targetContainer
      
      // Check if this is an init container - if so, we need to handle it specially
      const isInitContainer = pod.spec?.initContainers?.some((ic: any) => ic.name === targetContainer)
      if (isInitContainer) {
        // For init containers, we need to include previous logs since they may have completed
        logOptions.previous = false  // Get current logs, not just previous run
        logOptions.sinceTime = undefined  // Remove any time restrictions for init containers
      }
    }

    const logs = await k8sClient.getPodLogs(organization.namespace, pod.metadata.name, logOptions)

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
      containerName: targetContainer,
      deploymentMode,
      message: 'Logs retrieved successfully'
    })

  } catch (error) {
    console.error('Error fetching tool logs:', error)
    
    return NextResponse.json(
      { 
        error: 'Failed to fetch tool logs',
        message: error instanceof Error ? error.message : 'Unknown error'
      },
      { status: 500 }
    )
  }
}