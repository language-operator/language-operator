import { NextRequest, NextResponse } from 'next/server'
import { getServerSession } from 'next-auth'
import { authOptions } from '@/lib/auth'
import { k8sClient } from '@/lib/k8s-client'
import { db } from '@/lib/db'
import { requirePermission } from '@/lib/permissions'
import { LanguageAgent } from '@/types/agent'

// GET /api/clusters/[name]/agents/[agentName] - Get specific agent details
export async function GET(
  request: NextRequest,
  { params }: { params: Promise<{ name: string; agentName: string }> }
) {
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
    if (!clusterName || !agentName) {
      return NextResponse.json({ error: 'Cluster name and agent name are required' }, { status: 400 })
    }

    // Fetch specific agent from organization namespace
    let agent: LanguageAgent | null = null
    
    try {
      const response = await k8sClient.getLanguageAgent(organization.namespace, agentName)
      
      // Handle different response structures from k8s client
      if ((response as any)?.body) {
        agent = (response as any).body
      } else if ((response as any)?.data) {
        agent = (response as any).data
      } else if (response) {
        agent = response as LanguageAgent
      }
    } catch (k8sError) {
      // If agent not found, return 404
      if (k8sError instanceof Error && k8sError.message.includes('404')) {
        return NextResponse.json({ 
          error: 'Agent not found',
          details: `Agent "${agentName}" not found in cluster "${clusterName}"` 
        }, { status: 404 })
      }
      
      console.error('Error fetching agent from Kubernetes:', k8sError)
      throw k8sError
    }

    if (!agent) {
      return NextResponse.json({ 
        error: 'Agent not found',
        details: `Agent "${agentName}" not found in cluster "${clusterName}"` 
      }, { status: 404 })
    }

    // Verify agent belongs to user's organization
    const agentOrgLabel = agent.metadata?.labels?.['langop.io/organization-id']
    if (agentOrgLabel && agentOrgLabel !== organization.id) {
      return NextResponse.json({ 
        error: 'Agent not found',
        details: `Agent "${agentName}" not found in cluster "${clusterName}"` 
      }, { status: 404 })
    }

    return NextResponse.json({
      success: true,
      data: agent,
      cluster: clusterName,
    })

  } catch (error) {
    console.error('Error fetching agent details:', error)
    return NextResponse.json({ 
      error: 'Failed to fetch agent details',
      details: error instanceof Error ? error.message : 'Unknown error'
    }, { status: 500 })
  }
}

// DELETE /api/clusters/[name]/agents/[agentName] - Delete specific agent
export async function DELETE(
  request: NextRequest,
  { params }: { params: Promise<{ name: string; agentName: string }> }
) {
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
    
    const hasPermission = await requirePermission(user.id, organization.id, 'delete')
    if (!hasPermission) {
      return NextResponse.json({ error: 'Insufficient permissions' }, { status: 403 })
    }

    const { name: clusterName, agentName } = await params
    if (!clusterName || !agentName) {
      return NextResponse.json({ error: 'Cluster name and agent name are required' }, { status: 400 })
    }

    // Delete agent from organization namespace
    try {
      const response = await k8sClient.deleteLanguageAgent(organization.namespace, agentName)
      
      console.log(`User ${user.email} deleted LanguageAgent ${agentName} from cluster ${clusterName} in organization ${organization.name}`)
      
      return NextResponse.json({
        success: true,
        message: `Agent "${agentName}" deleted successfully`,
        cluster: clusterName,
      })
    } catch (k8sError) {
      // If agent not found, return 404
      if (k8sError instanceof Error && k8sError.message.includes('404')) {
        return NextResponse.json({ 
          error: 'Agent not found',
          details: `Agent "${agentName}" not found in cluster "${clusterName}"` 
        }, { status: 404 })
      }
      
      console.error('Error deleting agent from Kubernetes:', k8sError)
      throw k8sError
    }

  } catch (error) {
    console.error('Error deleting agent:', error)
    return NextResponse.json({ 
      error: 'Failed to delete agent',
      details: error instanceof Error ? error.message : 'Unknown error'
    }, { status: 500 })
  }
}