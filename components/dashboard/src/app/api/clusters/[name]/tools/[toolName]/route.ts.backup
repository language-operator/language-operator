import { NextRequest, NextResponse } from 'next/server'
import { getServerSession } from 'next-auth'
import { authOptions } from '@/lib/auth'
import { k8sClient } from '@/lib/k8s-client'
import { db } from '@/lib/db'
import { requirePermission } from '@/lib/permissions'

// GET /api/clusters/[name]/tools/[toolName] - Get a specific tool in a cluster
export async function GET(
  request: NextRequest,
  { params }: { params: Promise<{ name: string; toolName: string }> }
) {
  try {
    const { name: clusterName, toolName } = await params
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

    if (!clusterName || !toolName) {
      return NextResponse.json({ error: 'Cluster name and tool name are required' }, { status: 400 })
    }

    // Get the specific tool
    const tool = await k8sClient.getLanguageTool(organization.namespace, toolName)
    
    if (!tool) {
      return NextResponse.json({ error: 'Tool not found' }, { status: 404 })
    }
    
    return NextResponse.json({ 
      success: true,
      data: tool,
      cluster: clusterName
    })
    
  } catch (error) {
    console.error('Error fetching cluster tool:', error)
    return NextResponse.json(
      { 
        error: 'Failed to fetch tool',
        details: error instanceof Error ? error.message : 'Unknown error'
      },
      { status: 500 }
    )
  }
}

// DELETE /api/clusters/[name]/tools/[toolName] - Delete a specific tool in a cluster
export async function DELETE(
  request: NextRequest,
  { params }: { params: Promise<{ name: string; toolName: string }> }
) {
  try {
    const { name: clusterName, toolName } = await params
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

    if (!clusterName || !toolName) {
      return NextResponse.json({ error: 'Cluster name and tool name are required' }, { status: 400 })
    }

    // Check if tool exists
    const existingTool = await k8sClient.getLanguageTool(organization.namespace, toolName)
    if (!existingTool) {
      return NextResponse.json({ error: 'Tool not found' }, { status: 404 })
    }

    // Delete the tool
    await k8sClient.deleteLanguageTool(organization.namespace, toolName)

    // Log the deletion for audit trail
    console.log(`Tool deleted: ${toolName} in cluster ${clusterName} by ${session.user.email} in namespace ${organization.namespace}`)

    return NextResponse.json({ 
      success: true,
      cluster: clusterName
    })
    
  } catch (error) {
    console.error('Error deleting cluster tool:', error)
    return NextResponse.json(
      { 
        error: 'Failed to delete tool',
        details: error instanceof Error ? error.message : 'Unknown error'
      },
      { status: 500 }
    )
  }
}