import { NextRequest, NextResponse } from 'next/server'
import { getServerSession } from 'next-auth'
import { authOptions } from '@/lib/auth'
import { getKubernetesClient } from '@/lib/kubernetes'
import { db } from '@/lib/db'
import { requirePermission } from '@/lib/permissions'

// GET /api/agents/[name]/logs - Get pod logs for an agent
export async function GET(
  request: NextRequest,
  { params }: { params: Promise<{ name: string }> }
) {
  try {
    const { name } = await params
    const session = await getServerSession(authOptions)
    
    if (!session?.user?.email) {
      return NextResponse.json({ error: 'Unauthorized' }, { status: 401 })
    }

    // Get user's current organization
    const user = await db.user.findUnique({
      where: { email: session.user.email },
      include: { 
        memberships: { 
          include: { organization: true } 
        } 
      },
    })

    if (!user || user.memberships.length === 0) {
      return NextResponse.json({ error: 'No organization found' }, { status: 404 })
    }

    const organization = user.memberships[0].organization
    
    // Check permissions
    const hasPermission = await requirePermission(user.id, organization.id, 'view')
    if (!hasPermission) {
      return NextResponse.json({ error: 'Insufficient permissions' }, { status: 403 })
    }

    // Parse query parameters
    const url = new URL(request.url)
    const tailLines = parseInt(url.searchParams.get('tail') || '100')
    const containerName = url.searchParams.get('container') || undefined

    const k8sClient = getKubernetesClient()
    
    // First, get the agent to verify it exists
    const agent = await k8sClient.getLanguageAgent(organization.namespace, name)
    if (!agent) {
      return NextResponse.json({ error: 'Agent not found' }, { status: 404 })
    }

    // Find pods associated with this agent
    const pods = await k8sClient.listPodsForAgent(organization.namespace, name)
    
    if (!pods || pods.length === 0) {
      return NextResponse.json({
        success: true,
        data: {
          logs: 'No pods found for this agent',
          pods: [],
        }
      })
    }

    // Get logs from all pods (or just the first one for simplicity)
    const logsPromises = pods.slice(0, 3).map(async (pod) => {
      try {
        const logs = await k8sClient.getPodLogs(
          organization.namespace, 
          pod.metadata?.name || '', 
          containerName,
          tailLines
        )
        return {
          podName: pod.metadata?.name || 'unknown',
          containerName: containerName || 'default',
          logs: logs || 'No logs available',
          phase: pod.status?.phase || 'Unknown',
        }
      } catch (error) {
        return {
          podName: pod.metadata?.name || 'unknown',
          containerName: containerName || 'default',
          logs: `Error fetching logs: ${error}`,
          phase: pod.status?.phase || 'Unknown',
          error: true,
        }
      }
    })

    const podLogs = await Promise.all(logsPromises)

    return NextResponse.json({
      success: true,
      data: {
        agentName: name,
        namespace: organization.namespace,
        podLogs,
        totalPods: pods.length,
        timestamp: new Date().toISOString(),
      }
    })

  } catch (error) {
    console.error('Error fetching agent logs:', error)
    return NextResponse.json(
      { error: 'Failed to fetch agent logs' },
      { status: 500 }
    )
  }
}